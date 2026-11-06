// Command server runs the Ittisal Invoice HTTP server built with Chi.
//
// Endpoints (each of invoices, orders, deliveries, payments, statuses):
//
//	POST   /invoices[?dry_run=true|html=true]   create (calculate + validate + store)
//	GET    /invoices                            list summaries
//	GET    /invoices/{id}                       full envelope as JSON (source of truth)
//	GET    /invoices/{id}/html[?locale|watermark|sandbox]  printable HTML + client-side PDF button
//	GET    /invoices/{id}/envelope.json         download envelope
//	DELETE /invoices/{id}                       delete (no legal cancellation semantics)
//	GET    /healthz                             liveness probe
//
// Replace /invoices with /orders, /deliveries, /payments or /statuses for the
// other bill documents. HTML pages live at /orders, /deliveries, /payments,
// /statuses with Datastar partials under /partials/*.
//
// Persistence: SQLite by default (SQLITE_PATH, default goinvoice.db).
// Set SQLITE_PATH=:memory: for tests. Goose migrations in db/migrations
// run automatically on boot.
//
// Config comes from the environment, with optional .env file support
// (see .env.example): ADDR, SQLITE_PATH.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"

	"github.com/neo/goinvoice/internal/handlers"
	"github.com/neo/goinvoice/internal/seed"
	"github.com/neo/goinvoice/internal/store"
)

func main() {
	// Load .env if present (real env vars win). Missing file is fine.
	_ = godotenv.Load()

	ctx := context.Background()
	path := os.Getenv("SQLITE_PATH")
	if path == "" {
		path = "goinvoice.db"
	}
	// :memory: gives each pooled connection its own DB; use shared cache instead.
	if path == ":memory:" {
		path = "file::memory:?cache=shared"
	}
	// Enable FK enforcement on every pooled connection so party child rows
	// cascade with their customer/supplier.
	if strings.Contains(path, "?") {
		path += "&_pragma=foreign_keys(1)"
	} else {
		path += "?_pragma=foreign_keys(1)"
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if err := goose.SetDialect("sqlite"); err != nil {
		log.Fatalf("goose dialect: %v", err)
	}
	if err := goose.Up(db, "db/migrations"); err != nil {
		log.Fatalf("goose up: %v", err)
	}

	sqliteStore := store.NewSQLite(db)
	var entries store.Entries = sqliteStore
	log.Printf("ittisal-invoice using sqlite (%s)", path)

	// Demo data on first boot; never fatal.
	seed.Seed(ctx, entries)

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Logger, middleware.Recoverer)

	handlers.New(entries, sqliteStore).Register(r)
	handlers.Static(r)

	r.Get("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Dump a full envelope to disk for inspection.
	envelopeDownload := func(kind string) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			id := chi.URLParam(req, "id")
			env, err := entries.Get(req.Context(), id)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": kind + " not found"})
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Disposition", "attachment; filename="+id+".json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(env)
		}
	}
	r.Get("/invoices/{id}/envelope.json", envelopeDownload("invoice"))
	r.Get("/orders/{id}/envelope.json", envelopeDownload("order"))
	r.Get("/deliveries/{id}/envelope.json", envelopeDownload("delivery"))
	r.Get("/payments/{id}/envelope.json", envelopeDownload("payment"))
	r.Get("/statuses/{id}/envelope.json", envelopeDownload("status"))

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("ittisal-invoice listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
