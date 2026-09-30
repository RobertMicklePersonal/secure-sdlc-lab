// Package notes holds the note model and an in-memory, concurrency-safe store.
package notes

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sort"
	"sync"
	"time"
)

var (
	// ErrNotFound is returned when no note has the requested ID.
	ErrNotFound = errors.New("note not found")
	// ErrFull is returned when the store already holds its maximum number of notes.
	ErrFull = errors.New("note limit reached")
)

// Note is a single note.
type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Store is an in-memory note store. The capacity bound keeps a client from
// exhausting server memory by creating notes in a loop.
type Store struct {
	mu    sync.RWMutex
	notes map[string]Note
	max   int
	now   func() time.Time
}

// NewStore returns an empty store that holds at most max notes.
func NewStore(max int) *Store {
	return &Store{notes: make(map[string]Note), max: max, now: time.Now}
}

// List returns all notes, newest first.
func (s *Store) List() []Note {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Note, 0, len(s.notes))
	for _, n := range s.notes {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}

// Get returns the note with the given ID.
func (s *Store) Get(id string) (Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.notes[id]
	if !ok {
		return Note{}, ErrNotFound
	}
	return n, nil
}

// Create stores a new note and returns it with its generated ID.
func (s *Store) Create(title, body string) (Note, error) {
	id, err := newID()
	if err != nil {
		return Note{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.notes) >= s.max {
		return Note{}, ErrFull
	}
	now := s.now().UTC()
	n := Note{ID: id, Title: title, Body: body, CreatedAt: now, UpdatedAt: now}
	s.notes[id] = n
	return n, nil
}

// Update replaces the title and body of an existing note.
func (s *Store) Update(id, title, body string) (Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.notes[id]
	if !ok {
		return Note{}, ErrNotFound
	}
	n.Title, n.Body, n.UpdatedAt = title, body, s.now().UTC()
	s.notes[id] = n
	return n, nil
}

// Delete removes the note with the given ID.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.notes[id]; !ok {
		return ErrNotFound
	}
	delete(s.notes, id)
	return nil
}

// newID returns 128 random bits as hex, so IDs can't be guessed or enumerated.
func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
