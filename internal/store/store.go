// Package store provides a tiny in-memory, thread-safe persistence layer for
// GOBL envelopes. See entries.go (Entries interface) and postgres.go
// (Postgres/sqlc implementation) for the persistent backend.
package store

import (
	"context"
	"errors"
	"sync"

	"github.com/invopop/gobl"

	"github.com/neo/goinvoice/internal/db/sqlc"
)

// Store keeps envelopes in memory, preserving insertion order.
type Store struct {
	mu      sync.RWMutex
	entries map[string]Entry
	order   []string // newest last
}

var _ Entries = (*Store)(nil)
var _ Catalog = (*Store)(nil)

// New returns an empty store.
func New() *Store {
	return &Store{entries: make(map[string]Entry)}
}

// Put stores an envelope under the given ID, replacing if it already exists.
func (s *Store) Put(_ context.Context, id string, env *gobl.Envelope) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.entries[id]; !exists {
		s.order = append(s.order, id)
	}
	s.entries[id] = Entry{ID: id, Envelope: env}
	return nil
}

// Get fetches an envelope by ID.
func (s *Store) Get(_ context.Context, id string) (*gobl.Envelope, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[id]
	if !ok {
		return nil, ErrNotFound
	}
	return e.Envelope, nil
}

// List returns all entries, newest first.
func (s *Store) List(_ context.Context) ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, 0, len(s.order))
	for i := len(s.order) - 1; i >= 0; i-- {
		out = append(out, s.entries[s.order[i]])
	}
	return out, nil
}

// Delete removes an entry by ID, returning true if it existed.
func (s *Store) Delete(_ context.Context, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.entries[id]; !ok {
		return false, nil
	}
	delete(s.entries, id)
	for i, v := range s.order {
		if v == id {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	return true, nil
}

// ListCustomers returns nothing; the in-memory store holds no catalog.
func (s *Store) ListCustomers(_ context.Context) ([]sqlc.Customer, error) {
	return nil, nil
}

// GetCustomer always misses; the in-memory store holds no catalog.
func (s *Store) GetCustomer(_ context.Context, _ string) (sqlc.Customer, error) {
	return sqlc.Customer{}, ErrCustomerNotFound
}

// CreateCustomer is unsupported; the in-memory store holds no catalog.
func (s *Store) CreateCustomer(_ context.Context, _ sqlc.CreateCustomerParams) error {
	return errors.New("catalog is not supported by the in-memory store")
}

// UpdateCustomer is unsupported; the in-memory store holds no catalog.
func (s *Store) UpdateCustomer(_ context.Context, _ sqlc.UpdateCustomerParams) (int64, error) {
	return 0, errors.New("catalog is not supported by the in-memory store")
}

// DeleteCustomer is unsupported; the in-memory store holds no catalog.
func (s *Store) DeleteCustomer(_ context.Context, _ string) (int64, error) {
	return 0, errors.New("catalog is not supported by the in-memory store")
}

// ListSuppliers returns nothing; the in-memory store holds no catalog.
func (s *Store) ListSuppliers(_ context.Context) ([]sqlc.Supplier, error) {
	return nil, nil
}

// GetSupplier always misses; the in-memory store holds no catalog.
func (s *Store) GetSupplier(_ context.Context, _ string) (sqlc.Supplier, error) {
	return sqlc.Supplier{}, ErrSupplierNotFound
}

// CreateSupplier is unsupported; the in-memory store holds no catalog.
func (s *Store) CreateSupplier(_ context.Context, _ sqlc.CreateSupplierParams) error {
	return errors.New("catalog is not supported by the in-memory store")
}

// UpdateSupplier is unsupported; the in-memory store holds no catalog.
func (s *Store) UpdateSupplier(_ context.Context, _ sqlc.UpdateSupplierParams) (int64, error) {
	return 0, errors.New("catalog is not supported by the in-memory store")
}

// DeleteSupplier is unsupported; the in-memory store holds no catalog.
func (s *Store) DeleteSupplier(_ context.Context, _ string) (int64, error) {
	return 0, errors.New("catalog is not supported by the in-memory store")
}

// ListProducts returns nothing; the in-memory store holds no catalog.
func (s *Store) ListProducts(_ context.Context) ([]sqlc.Product, error) {
	return nil, nil
}

// CreateProduct is unsupported; the in-memory store holds no catalog.
func (s *Store) CreateProduct(_ context.Context, _ sqlc.CreateProductParams) error {
	return errors.New("catalog is not supported by the in-memory store")
}

// GetProduct always misses; the in-memory store holds no catalog.
func (s *Store) GetProduct(_ context.Context, _ string) (sqlc.Product, error) {
	return sqlc.Product{}, ErrProductNotFound
}

// UpdateProduct is unsupported; the in-memory store holds no catalog.
func (s *Store) UpdateProduct(_ context.Context, _ sqlc.UpdateProductParams) (int64, error) {
	return 0, errors.New("catalog is not supported by the in-memory store")
}

// DeleteProduct is unsupported; the in-memory store holds no catalog.
func (s *Store) DeleteProduct(_ context.Context, _ string) (int64, error) {
	return 0, errors.New("catalog is not supported by the in-memory store")
}
