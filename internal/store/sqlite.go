package store

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/org"

	"github.com/neo/goinvoice/internal/db/sqlc"
)

var _ Entries = (*SQLite)(nil)
var _ Catalog = (*SQLite)(nil)

// SQLite persists envelopes via sqlc, same semantics as Store.
// The full envelope is stored as JSON text (source of truth); catalog
// parties are normalized into customers/suppliers plus party_* child tables.
type SQLite struct {
	db *sql.DB
	q  *sqlc.Queries
}

// NewSQLite wires sqlc queries over an existing *sql.DB.
func NewSQLite(db *sql.DB) *SQLite {
	return &SQLite{db: db, q: sqlc.New(db)}
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

// SaveCustomer upserts a customer and its child collections.
func (s *SQLite) SaveCustomer(ctx context.Context, id string, p *org.Party) error {
	return s.SaveCustomerParty(ctx, id, p)
}

// DeleteCustomer removes a customer, cascading to its child rows.
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

// SaveSupplier upserts a supplier and its child collections.
func (s *SQLite) SaveSupplier(ctx context.Context, id string, p *org.Party) error {
	return s.SaveSupplierParty(ctx, id, p)
}

// DeleteSupplier removes a supplier, cascading to its child rows.
func (s *SQLite) DeleteSupplier(ctx context.Context, id string) (int64, error) {
	return s.q.DeleteSupplier(ctx, id)
}

// ListProducts returns all products ordered by ref.
func (s *SQLite) ListProducts(ctx context.Context) ([]sqlc.Product, error) {
	return s.q.ListProducts(ctx)
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

// DeleteProduct removes a product, cascading to its child rows.
func (s *SQLite) DeleteProduct(ctx context.Context, id string) (int64, error) {
	return s.q.DeleteProduct(ctx, id)
}
