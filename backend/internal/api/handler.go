// Package api exposes the notes store over HTTP.
package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/RobertMicklePersonal/secure-sdlc-lab/backend/internal/notes"
)

// Input bounds. MaxBodyBytes leaves room for JSON escaping of a full-size body.
const (
	MaxTitleRunes = 200
	MaxBodyRunes  = 5000
	MaxBodyBytes  = 64 << 10
)

// idPattern matches the IDs the store generates; anything else is rejected
// before it reaches the store or the logs.
var idPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

// Config controls the HTTP handler.
type Config struct {
	// AllowedOrigins lists the exact origins allowed to call the API from a
	// browser. Empty means no cross-origin access.
	AllowedOrigins []string
	Logger         *slog.Logger
}

type server struct {
	store *notes.Store
	log   *slog.Logger
}

type noteInput struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// NewHandler returns the full HTTP handler, middleware included.
func NewHandler(store *notes.Store, cfg Config) http.Handler {
	log := cfg.Logger
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	s := &server{store: store, log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /api/notes", s.listNotes)
	mux.HandleFunc("POST /api/notes", s.createNote)
	mux.HandleFunc("GET /api/notes/{id}", s.getNote)
	mux.HandleFunc("PUT /api/notes/{id}", s.updateNote)
	mux.HandleFunc("DELETE /api/notes/{id}", s.deleteNote)

	var h http.Handler = mux
	h = cors(cfg.AllowedOrigins, h)
	h = securityHeaders(h)
	h = recoverPanics(log, h)
	h = requestLog(log, h)
	return h
}

func (s *server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) listNotes(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.List())
}

func (s *server) getNote(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	n, err := s.store.Get(id)
	if err != nil {
		s.storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (s *server) createNote(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeNote(w, r)
	if !ok {
		return
	}
	n, err := s.store.Create(in.Title, in.Body)
	if err != nil {
		s.storeError(w, err)
		return
	}
	w.Header().Set("Location", "/api/notes/"+n.ID)
	writeJSON(w, http.StatusCreated, n)
}

func (s *server) updateNote(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	in, ok := decodeNote(w, r)
	if !ok {
		return
	}
	n, err := s.store.Update(id, in.Title, in.Body)
	if err != nil {
		s.storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (s *server) deleteNote(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.store.Delete(id); err != nil {
		s.storeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) storeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, notes.ErrNotFound):
		writeError(w, http.StatusNotFound, "note not found")
	case errors.Is(err, notes.ErrFull):
		writeError(w, http.StatusInsufficientStorage, "note limit reached")
	default:
		// Log the detail server-side; never echo internals to the client.
		s.log.Error("store error", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func pathID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.PathValue("id")
	if !idPattern.MatchString(id) {
		writeError(w, http.StatusNotFound, "note not found")
		return "", false
	}
	return id, true
}

// decodeNote reads and validates a note from the request body. It rejects
// wrong content types, oversized bodies, unknown fields and trailing data.
func decodeNote(w http.ResponseWriter, r *http.Request) (noteInput, bool) {
	var in noteInput
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return in, false
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
		} else {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
		}
		return in, false
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "body must be a single JSON object")
		return in, false
	}

	in.Title = strings.TrimSpace(in.Title)
	if msg := validate(in); msg != "" {
		writeError(w, http.StatusUnprocessableEntity, msg)
		return in, false
	}
	return in, true
}

func validate(in noteInput) string {
	switch {
	case in.Title == "":
		return "title is required"
	case !utf8.ValidString(in.Title) || !utf8.ValidString(in.Body):
		return "text must be valid UTF-8"
	case utf8.RuneCountInString(in.Title) > MaxTitleRunes:
		return "title is too long"
	case utf8.RuneCountInString(in.Body) > MaxBodyRunes:
		return "body is too long"
	case hasControl(in.Title, false) || hasControl(in.Body, true):
		return "text contains control characters"
	}
	return ""
}

// hasControl reports whether s contains control characters. Newlines and tabs
// are allowed in multi-line fields.
func hasControl(s string, multiline bool) bool {
	for _, r := range s {
		if multiline && (r == '\n' || r == '\r' || r == '\t') {
			continue
		}
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
