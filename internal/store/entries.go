// Package store provides persistence for GOBL envelopes.
//
// Store is the original in-memory implementation. Postgres is the
// Postgres/sqlc implementation with identical semantics: Put upserts
// by ID, List returns newest first. Both satisfy Entries so handlers
// stay backend-agnostic.
package store

import (
	"context"

	"github.com/invopop/gobl"
)

// Entry is a stored envelope plus its key.
type Entry struct {
	ID       string         `json:"id"`
	Envelope *gobl.Envelope `json:"envelope"`
}

// Entries abstracts the envelope persistence backend.
type Entries interface {
	Put(ctx context.Context, id string, env *gobl.Envelope) error
	Get(ctx context.Context, id string) (*gobl.Envelope, error)
	List(ctx context.Context) ([]Entry, error)
	Delete(ctx context.Context, id string) (bool, error)
}
