// Package seed populates demo invoices on first boot.
//
// Catalog rows (customers/suppliers/products) come from the goose migration
// db/migrations/00003_seed_catalog.sql. Invoices can't be plain SQL — they
// must pass the GOBL calculate + validate pipeline — so they are built here
// from JSON templates and stored only when the invoices table is empty.
package seed

import (
	"context"
	"encoding/json"
	"log"

	"github.com/invopop/gobl/bill"

	"github.com/neo/goinvoice/internal/invoicer"
	"github.com/neo/goinvoice/internal/store"
)

// templates are valid bill.Invoice documents using the seeded catalog parties.
// Tax IDs reuse checksum-validated values (see README client rules); the
// Initech invoice intentionally omits the customer tax ID.
var templates = []string{
	`{
	  "type": "standard", "series": "FA", "code": "R-2026-011",
	  "issue_date": "2026-11-03", "currency": "EUR",
	  "supplier": {
	    "name": "Acme Consultancy SL",
	    "tax_id": { "country": "ES", "code": "B98602642" },
	    "emails": [{ "label": "Billing", "addr": "billing@acme.example" }],
	    "addresses": [{ "lines": ["Calle Mayor 1"], "locality": "Madrid", "post_code": "28001", "country": "ES" }]
	  },
	  "customer": {
	    "name": "Globex Ibérica SA",
	    "tax_id": { "country": "ES", "code": "A12345674" },
	    "addresses": [{ "lines": ["Avenida de la Innovación 42"], "locality": "Madrid", "post_code": "28020", "country": "ES" }]
	  },
	  "lines": [
	    {"quantity": "10", "item": { "name": "Backend consulting", "price": "100.00" }, "taxes": [{ "cat": "VAT", "rate": "standard" }]},
	    {"quantity": "2", "item": { "name": "Invoice generator license", "price": "250.00" }, "discounts": [{ "percent": "10.0%" }], "taxes": [{ "cat": "VAT", "rate": "standard" }]}
	  ]
	}`,
	`{
	  "type": "standard", "series": "FA", "code": "R-2026-012",
	  "issue_date": "2026-10-14", "currency": "EUR",
	  "supplier": {
	    "name": "Acme Consultancy SL",
	    "tax_id": { "country": "ES", "code": "B98602642" },
	    "emails": [{ "label": "Billing", "addr": "billing@acme.example" }],
	    "addresses": [{ "lines": ["Calle Mayor 1"], "locality": "Madrid", "post_code": "28001", "country": "ES" }]
	  },
	  "customer": {
	    "name": "Initech SL",
	    "emails": [{ "label": "Accounts", "addr": "accounts@initech.example" }],
	    "addresses": [{ "lines": ["Calle de Alcalá 100"], "locality": "Madrid", "post_code": "28009", "country": "ES" }]
	  },
	  "lines": [
	    {"quantity": "5", "item": { "name": "Support hour", "price": "80.00" }, "taxes": [{ "cat": "VAT", "rate": "standard" }]},
	    {"quantity": "1", "item": { "name": "Hosting", "price": "49.00" }, "taxes": [{ "cat": "VAT", "rate": "standard" }]}
	  ]
	}`,
	`{
	  "type": "standard", "series": "FA", "code": "R-2026-013",
	  "issue_date": "2026-09-02", "currency": "EUR",
	  "supplier": {
	    "name": "Acme Consultancy SL",
	    "tax_id": { "country": "ES", "code": "B98602642" },
	    "emails": [{ "label": "Billing", "addr": "billing@acme.example" }],
	    "addresses": [{ "lines": ["Calle Mayor 1"], "locality": "Madrid", "post_code": "28001", "country": "ES" }]
	  },
	  "customer": {
	    "name": "Globex Ibérica SA",
	    "tax_id": { "country": "ES", "code": "A12345674" },
	    "addresses": [{ "lines": ["Avenida de la Innovación 42"], "locality": "Madrid", "post_code": "28020", "country": "ES" }]
	  },
	  "lines": [
	    {"quantity": "4", "item": { "name": "Invoice generator license", "price": "250.00" }, "taxes": [{ "cat": "VAT", "rate": "standard" }]}
	  ]
	}`,
}

// Seed builds and stores the demo invoices when none exist yet.
// Failures are logged, never fatal — the server must boot regardless.
func Seed(ctx context.Context, entries store.Entries) {
	existing, err := entries.List(ctx)
	if err != nil {
		log.Printf("seed: list invoices: %v", err)
		return
	}
	if len(existing) > 0 {
		return
	}
	for i, tpl := range templates {
		inv := new(bill.Invoice)
		if err := json.Unmarshal([]byte(tpl), inv); err != nil {
			log.Printf("seed: template %d decode: %v", i, err)
			continue
		}
		env, err := invoicer.Build(inv)
		if err != nil {
			log.Printf("seed: template %d build: %v", i, err)
			continue
		}
		if err := entries.Put(ctx, invoicer.ID(env), env); err != nil {
			log.Printf("seed: template %d store: %v", i, err)
			continue
		}
	}
	log.Printf("seed: stored %d demo invoices", len(templates))
}
