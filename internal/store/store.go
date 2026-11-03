// Package store provides a tiny in-memory, thread-safe persistence layer for
// GOBL envelopes. Swap out with Postgres/sqlc later using the same interface.
package store

import (
	"sync"

	"github.com/invopop/gobl"
)

// Entry is a stored envelope plus its key.
type Entry struct {
	ID       string         `json:"id"`
	Envelope *gobl.Envelope `json:"envelope"`
}

// Store keeps envelopes in memory, preserving insertion order.
type Store struct {
	mu      sync.RWMutex
	entries map[string]Entry
	order   []string // newest last
}

// New returns an empty store.
func New() *Store {
	return &Store{entries: make(map[string]Entry)}
}

// Put stores an envelope under the given ID, replacing if it already exists.
func (s *Store) Put(id string, env *gobl.Envelope) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.entries[id]; !exists {
		s.order = append(s.order, id)
	}
	s.entries[id] = Entry{ID: id, Envelope: env}
}

// Get fetches an envelope by ID.
func (s *Store) Get(id string) (*gobl.Envelope, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[id]
	if !ok {
		return nil, false
	}
	return e.Envelope, true
}

// List returns all entries, newest first.
func (s *Store) List() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, 0, len(s.order))
	for i := len(s.order) - 1; i >= 0; i-- {
		out = append(out, s.entries[s.order[i]])
	}
	return out
}

// Delete removes an entry by ID, returning true if it existed.
func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.entries[id]; !ok {
		return false
	}
	delete(s.entries, id)
	for i, v := range s.order {
		if v == id {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	return true
}
