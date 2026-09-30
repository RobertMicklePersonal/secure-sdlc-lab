package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RobertMicklePersonal/secure-sdlc-lab/backend/internal/notes"
)

const allowedOrigin = "http://localhost:5173"

func newTestHandler(capacity int) http.Handler {
	return NewHandler(notes.NewStore(capacity), Config{AllowedOrigins: []string{allowedOrigin}})
}

func do(t *testing.T, h http.Handler, method, path, body string, hdr map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(rec.Body).Decode(&v); err != nil {
		t.Fatalf("decode response: %v (body %q)", err, rec.Body.String())
	}
	return v
}

func TestHealthz(t *testing.T) {
	rec := do(t, newTestHandler(10), "GET", "/healthz", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestNotesLifecycle(t *testing.T) {
	h := newTestHandler(10)

	rec := do(t, h, "POST", "/api/notes", `{"title":"  hello ","body":"line1\nline2"}`, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body %s", rec.Code, rec.Body)
	}
	created := decode[notes.Note](t, rec)
	if created.Title != "hello" {
		t.Errorf("title not trimmed: %q", created.Title)
	}
	if loc := rec.Header().Get("Location"); loc != "/api/notes/"+created.ID {
		t.Errorf("Location = %q", loc)
	}

	rec = do(t, h, "GET", "/api/notes", "", nil)
	if list := decode[[]notes.Note](t, rec); len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("list = %+v", list)
	}

	rec = do(t, h, "PUT", "/api/notes/"+created.ID, `{"title":"updated","body":""}`, nil)
	if rec.Code != http.StatusOK || decode[notes.Note](t, rec).Title != "updated" {
		t.Fatalf("update status = %d", rec.Code)
	}

	rec = do(t, h, "GET", "/api/notes/"+created.ID, "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d", rec.Code)
	}

	rec = do(t, h, "DELETE", "/api/notes/"+created.ID, "", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d", rec.Code)
	}

	rec = do(t, h, "GET", "/api/notes/"+created.ID, "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get after delete status = %d", rec.Code)
	}

	rec = do(t, h, "GET", "/api/notes", "", nil)
	if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
		t.Errorf("empty list body = %q, want []", body)
	}
}

func TestCreateValidation(t *testing.T) {
	h := newTestHandler(10)
	tests := []struct {
		name   string
		body   string
		ctype  string
		status int
	}{
		{"wrong content type", `{"title":"a"}`, "text/plain", http.StatusUnsupportedMediaType},
		{"missing content type", `{"title":"a"}`, "-", http.StatusUnsupportedMediaType},
		{"malformed json", `{"title":`, "", http.StatusBadRequest},
		{"unknown field", `{"title":"a","admin":true}`, "", http.StatusBadRequest},
		{"trailing data", `{"title":"a"}{"title":"b"}`, "", http.StatusBadRequest},
		{"wrong type", `{"title":123}`, "", http.StatusBadRequest},
		{"empty title", `{"title":"   "}`, "", http.StatusUnprocessableEntity},
		{"title too long", `{"title":"` + strings.Repeat("a", MaxTitleRunes+1) + `"}`, "", http.StatusUnprocessableEntity},
		{"body too long", `{"title":"a","body":"` + strings.Repeat("b", MaxBodyRunes+1) + `"}`, "", http.StatusUnprocessableEntity},
		{"control char in title", `{"title":"a\u0000b"}`, "", http.StatusUnprocessableEntity},
		{"newline in title", `{"title":"a\nb"}`, "", http.StatusUnprocessableEntity},
		{"oversized request", `{"title":"a","body":"` + strings.Repeat("x", MaxBodyBytes) + `"}`, "", http.StatusRequestEntityTooLarge},
		{"json content type with charset", `{"title":"ok"}`, "application/json; charset=utf-8", http.StatusCreated},
		{"multibyte at limit", `{"title":"` + strings.Repeat("é", MaxTitleRunes) + `"}`, "", http.StatusCreated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hdr := map[string]string{}
			switch tt.ctype {
			case "":
			case "-":
				hdr["Content-Type"] = ""
			default:
				hdr["Content-Type"] = tt.ctype
			}
			rec := do(t, h, "POST", "/api/notes", tt.body, hdr)
			if rec.Code != tt.status {
				t.Errorf("status = %d, want %d (body %s)", rec.Code, tt.status, rec.Body)
			}
		})
	}
}

func TestInvalidID(t *testing.T) {
	h := newTestHandler(10)
	for _, id := range []string{"abc", "..%2F..%2Fetc%2Fpasswd", strings.Repeat("A", 32), strings.Repeat("0", 33)} {
		rec := do(t, h, "GET", "/api/notes/"+id, "", nil)
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %q: status = %d, want 404", id, rec.Code)
		}
	}
}

func TestCapacityLimit(t *testing.T) {
	h := newTestHandler(1)
	if rec := do(t, h, "POST", "/api/notes", `{"title":"a"}`, nil); rec.Code != http.StatusCreated {
		t.Fatalf("first create = %d", rec.Code)
	}
	if rec := do(t, h, "POST", "/api/notes", `{"title":"b"}`, nil); rec.Code != http.StatusInsufficientStorage {
		t.Fatalf("second create = %d, want 507", rec.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	rec := do(t, newTestHandler(10), "PATCH", "/api/notes", "", nil)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	rec := do(t, newTestHandler(10), "GET", "/api/notes", "", nil)
	want := map[string]string{
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
		"Content-Security-Policy":      "default-src 'none'; frame-ancestors 'none'",
		"Referrer-Policy":              "no-referrer",
		"Cache-Control":                "no-store",
		"Cross-Origin-Resource-Policy": "same-origin",
	}
	for k, v := range want {
		if got := rec.Header().Get(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("missing X-Request-ID")
	}
	// Headers must also be present on error responses.
	rec = do(t, newTestHandler(10), "GET", "/nope", "", nil)
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("security headers missing on 404")
	}
}

func TestCORS(t *testing.T) {
	h := newTestHandler(10)

	t.Run("allowed origin", func(t *testing.T) {
		rec := do(t, h, "GET", "/api/notes", "", map[string]string{"Origin": allowedOrigin})
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != allowedOrigin {
			t.Errorf("ACAO = %q", got)
		}
		if rec.Header().Get("Access-Control-Allow-Credentials") != "" {
			t.Error("credentials must not be allowed")
		}
	})

	t.Run("other origin not reflected", func(t *testing.T) {
		rec := do(t, h, "GET", "/api/notes", "", map[string]string{"Origin": "https://evil.example"})
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("ACAO = %q, want empty", got)
		}
	})

	t.Run("preflight allowed", func(t *testing.T) {
		rec := do(t, h, "OPTIONS", "/api/notes", "", map[string]string{
			"Origin": allowedOrigin, "Access-Control-Request-Method": "POST",
		})
		if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Errorf("status = %d, headers %v", rec.Code, rec.Header())
		}
	})

	t.Run("preflight denied", func(t *testing.T) {
		rec := do(t, h, "OPTIONS", "/api/notes", "", map[string]string{
			"Origin": "https://evil.example", "Access-Control-Request-Method": "DELETE",
		})
		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403", rec.Code)
		}
	})
}

func TestPanicRecovery(t *testing.T) {
	h := recoverPanics(slog.New(slog.DiscardHandler), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("secret detail")
	}))
	rec := do(t, h, "GET", "/", "", nil)
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "secret") {
		t.Errorf("status = %d, body %q", rec.Code, rec.Body)
	}
}
