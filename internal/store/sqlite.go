package store

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/invopop/gobl"

	"github.com/neo/goinvoice/internal/db/sqlc"
)

var _ Entries = (*SQLite)(nil)
var _ Catalog = (*SQLite)(nil)

// SQLite persists envelopes via sqlc, same semantics as Store.
// The full envelope is stored as JSON text (source of truth); no
// party/product normalization — supplier/customer/item stay embedded.
type SQLite struct {
	q *sqlc.Queries
}

// NewSQLite wires sqlc queries over an existing *sql.DB.
func NewSQLite(db *sql.DB) *SQLite {
	return &SQLite{q: sqlc.New(db)}
}

// Put marshals the envelope and upserts by ID.
func (s *SQLite) Put(ctx context.Context, id string, env *gobl.Envelope) error {
	raw, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return s.q.UpsertInvoice(ctx, sqlc.UpsertInvoiceParams{ID: id, Envelope: string(raw)})
}

// Get fetches one envelope, returning ErrNotFound when missing.
func (s *SQLite) Get(ctx context.Context, id string) (*gobl.Envelope, error) {
	row, err := s.q.GetInvoice(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	env := new(gobl.Envelope)
	if err := json.Unmarshal([]byte(row.Envelope), env); err != nil {
		return nil, err
	}
	return env, nil
}

// List returns all envelopes, newest first.
func (s *SQLite) List(ctx context.Context) ([]Entry, error) {
	rows, err := s.q.ListInvoices(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(rows))
	for _, r := range rows {
		env := new(gobl.Envelope)
		if err := json.Unmarshal([]byte(r.Envelope), env); err != nil {
			return nil, err
		}
		out = append(out, Entry{ID: r.ID, Envelope: env})
	}
	return out, nil
}

// Delete removes an envelope, returning false when the ID did not exist.
func (s *SQLite) Delete(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteInvoice(ctx, id)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// ListCustomers returns all customers ordered by name.
func (s *SQLite) ListCustomers(ctx context.Context) ([]sqlc.Customer, error) {
	return s.q.ListCustomers(ctx)
}

// GetCustomer fetches one customer, returning ErrCustomerNotFound when missing.
func (s *SQLite) GetCustomer(ctx context.Context, id string) (sqlc.Customer, error) {
	c, err := s.q.GetCustomer(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return sqlc.Customer{}, ErrCustomerNotFound
		}
		return sqlc.Customer{}, err
	}
	return c, nil
}

// CreateCustomer inserts a new customer; a duplicate tax code violates the
// UNIQUE constraint and returns an error.
func (s *SQLite) CreateCustomer(ctx context.Context, c sqlc.CreateCustomerParams) error {
	return s.q.CreateCustomer(ctx, c)
}

// UpdateCustomer updates a customer, returning rows affected.
func (s *SQLite) UpdateCustomer(ctx context.Context, c sqlc.UpdateCustomerParams) (int64, error) {
	return s.q.UpdateCustomer(ctx, c)
}

// DeleteCustomer removes a customer, returning rows affected.
func (s *SQLite) DeleteCustomer(ctx context.Context, id string) (int64, error) {
	return s.q.DeleteCustomer(ctx, id)
}

// ListSuppliers returns all suppliers ordered by name.
func (s *SQLite) ListSuppliers(ctx context.Context) ([]sqlc.Supplier, error) {
	return s.q.ListSuppliers(ctx)
}

// GetSupplier fetches one supplier, returning ErrSupplierNotFound when missing.
func (s *SQLite) GetSupplier(ctx context.Context, id string) (sqlc.Supplier, error) {
	sp, err := s.q.GetSupplier(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return sqlc.Supplier{}, ErrSupplierNotFound
		}
		return sqlc.Supplier{}, err
	}
	return sp, nil
}

// CreateSupplier inserts a new supplier; a duplicate tax code violates the
// UNIQUE constraint and returns an error.
func (s *SQLite) CreateSupplier(ctx context.Context, sp sqlc.CreateSupplierParams) error {
	return s.q.CreateSupplier(ctx, sp)
}

// UpdateSupplier updates a supplier, returning rows affected.
func (s *SQLite) UpdateSupplier(ctx context.Context, sp sqlc.UpdateSupplierParams) (int64, error) {
	return s.q.UpdateSupplier(ctx, sp)
}

// DeleteSupplier removes a supplier, returning rows affected.
func (s *SQLite) DeleteSupplier(ctx context.Context, id string) (int64, error) {
	return s.q.DeleteSupplier(ctx, id)
}

// ListProducts returns all products ordered by ref.
func (s *SQLite) ListProducts(ctx context.Context) ([]sqlc.Product, error) {
	return s.q.ListProducts(ctx)
}

// CreateProduct inserts a new product; duplicate ref/code violates the
// UNIQUE constraint and returns an error.
func (s *SQLite) CreateProduct(ctx context.Context, p sqlc.CreateProductParams) error {
	return s.q.CreateProduct(ctx, p)
}

// GetProduct fetches one product, returning ErrProductNotFound when missing.
func (s *SQLite) GetProduct(ctx context.Context, id string) (sqlc.Product, error) {
	p, err := s.q.GetProduct(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return sqlc.Product{}, ErrProductNotFound
		}
		return sqlc.Product{}, err
	}
	return p, nil
}

// UpdateProduct updates a product, returning rows affected.
func (s *SQLite) UpdateProduct(ctx context.Context, p sqlc.UpdateProductParams) (int64, error) {
	return s.q.UpdateProduct(ctx, p)
}

// DeleteProduct removes a product, returning rows affected.
func (s *SQLite) DeleteProduct(ctx context.Context, id string) (int64, error) {
	return s.q.DeleteProduct(ctx, id)
}
