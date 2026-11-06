// Package seed populates demo documents on first boot.
//
// Catalog rows (customers/suppliers/products) come from the goose migration
// db/migrations/00003_seed_catalog.sql. Documents can't be plain SQL — they
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

// orderTemplates are valid bill.Order documents (purchase/sale/quote).
var orderTemplates = []string{
	`{
	  "type": "purchase", "series": "PO", "code": "PO-2026-001",
	  "issue_date": "2026-11-03", "currency": "EUR",
	  "supplier": {
	    "name": "Acme Consultancy SL",
	    "tax_id": { "country": "ES", "code": "B98602642" },
	    "addresses": [{ "lines": ["Calle Mayor 1"], "locality": "Madrid", "post_code": "28001", "country": "ES" }]
	  },
	  "customer": {
	    "name": "Globex Ibérica SA",
	    "tax_id": { "country": "ES", "code": "A12345674" },
	    "addresses": [{ "lines": ["Avenida de la Innovación 42"], "locality": "Madrid", "post_code": "28020", "country": "ES" }]
	  },
	  "lines": [
	    {"quantity": "10", "item": { "name": "Backend consulting", "price": "100.00" }, "taxes": [{ "cat": "VAT", "rate": "standard" }]}
	  ]
	}`,
	`{
	  "type": "sale", "series": "SO", "code": "SO-2026-002",
	  "issue_date": "2026-10-20", "currency": "EUR",
	  "supplier": {
	    "name": "Acme Consultancy SL",
	    "tax_id": { "country": "ES", "code": "B98602642" },
	    "addresses": [{ "lines": ["Calle Mayor 1"], "locality": "Madrid", "post_code": "28001", "country": "ES" }]
	  },
	  "customer": {
	    "name": "Globex Ibérica SA",
	    "tax_id": { "country": "ES", "code": "A12345674" },
	    "addresses": [{ "lines": ["Avenida de la Innovación 42"], "locality": "Madrid", "post_code": "28020", "country": "ES" }]
	  },
	  "lines": [
	    {"quantity": "4", "item": { "name": "Invoice generator license", "price": "250.00" }, "taxes": [{ "cat": "VAT", "rate": "standard" }]},
	    {"quantity": "5", "item": { "name": "Support hour", "price": "80.00" }, "taxes": [{ "cat": "VAT", "rate": "standard" }]}
	  ]
	}`,
}

// deliveryTemplates are valid bill.Delivery documents (advice/note/waybill/receipt).
var deliveryTemplates = []string{
	`{
	  "type": "note", "series": "DN", "code": "DN-2026-001",
	  "issue_date": "2026-11-04", "currency": "EUR",
	  "supplier": {
	    "name": "Acme Consultancy SL",
	    "tax_id": { "country": "ES", "code": "B98602642" },
	    "addresses": [{ "lines": ["Calle Mayor 1"], "locality": "Madrid", "post_code": "28001", "country": "ES" }]
	  },
	  "customer": {
	    "name": "Globex Ibérica SA",
	    "tax_id": { "country": "ES", "code": "A12345674" },
	    "addresses": [{ "lines": ["Avenida de la Innovación 42"], "locality": "Madrid", "post_code": "28020", "country": "ES" }]
	  },
	  "lines": [
	    {"quantity": "10", "item": { "name": "Backend consulting", "price": "100.00" }, "taxes": [{ "cat": "VAT", "rate": "standard" }]}
	  ]
	}`,
	`{
	  "type": "advice", "series": "DA", "code": "DA-2026-002",
	  "issue_date": "2026-10-21", "currency": "EUR",
	  "supplier": {
	    "name": "Acme Consultancy SL",
	    "tax_id": { "country": "ES", "code": "B98602642" },
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

// paymentTemplates are valid bill.Payment documents (request/receipt/advice).
var paymentTemplates = []string{
	`{
	  "type": "receipt", "series": "RC", "code": "RC-2026-001",
	  "issue_date": "2026-11-05", "currency": "EUR",
	  "supplier": {
	    "name": "Acme Consultancy SL",
	    "tax_id": { "country": "ES", "code": "B98602642" }
	  },
	  "customer": {
	    "name": "Globex Ibérica SA",
	    "tax_id": { "country": "ES", "code": "A12345674" }
	  },
	  "lines": [
	    {"amount": "1210.00", "document": {"series": "FA", "code": "R-2026-011"}}
	  ],
	  "methods": [
	    {"key": "credit-transfer", "amount": "1210.00"}
	  ]
	}`,
	`{
	  "type": "request", "series": "RQ", "code": "RQ-2026-002",
	  "issue_date": "2026-10-15", "currency": "EUR",
	  "supplier": {
	    "name": "Acme Consultancy SL",
	    "tax_id": { "country": "ES", "code": "B98602642" }
	  },
	  "customer": {
	    "name": "Initech SL"
	  },
	  "lines": [
	    {"amount": "543.29", "document": {"series": "FA", "code": "R-2026-012"}}
	  ],
	  "methods": [
	    {"key": "credit-transfer", "amount": "543.29"}
	  ]
	}`,
}

// statusTemplates are valid bill.Status documents (response/update/system).
var statusTemplates = []string{
	`{
	  "type": "response", "code": "ST-2026-001",
	  "issue_date": "2026-11-05",
	  "supplier": {
	    "name": "Acme Consultancy SL",
	    "tax_id": { "country": "ES", "code": "B98602642" }
	  },
	  "customer": {
	    "name": "Globex Ibérica SA",
	    "tax_id": { "country": "ES", "code": "A12345674" }
	  },
	  "lines": [
	    {"key": "accepted", "doc": {"series": "FA", "code": "R-2026-011"}}
	  ]
	}`,
	`{
	  "type": "update", "code": "ST-2026-002",
	  "issue_date": "2026-10-16",
	  "supplier": {
	    "name": "Acme Consultancy SL",
	    "tax_id": { "country": "ES", "code": "B98602642" }
	  },
	  "customer": {
	    "name": "Initech SL"
	  },
	  "lines": [
	    {"key": "processing", "doc": {"series": "FA", "code": "R-2026-012"}}
	  ]
	}`,
}

// Seed builds and stores demo documents for every bill kind that has none
// yet. It is additive: existing user data is never touched, so restarting
// the server backfills kinds added after the database was created.
// Failures are logged, never fatal — the server must boot regardless.
func Seed(ctx context.Context, entries store.Entries) {
	existing, err := entries.List(ctx)
	if err != nil {
		log.Printf("seed: list documents: %v", err)
		return
	}
	present := make(map[string]bool)
	for _, e := range existing {
		if k := invoicer.DocumentKind(e.Envelope); k != "" {
			present[k] = true
		}
	}
	stored := 0
	if !present[invoicer.KindInvoice] {
		for i, tpl := range templates {
		inv := new(bill.Invoice)
		if err := json.Unmarshal([]byte(tpl), inv); err != nil {
			log.Printf("seed: invoice template %d decode: %v", i, err)
			continue
		}
		env, err := invoicer.Build(inv)
		if err != nil {
			log.Printf("seed: invoice template %d build: %v", i, err)
			continue
		}
		if err := entries.Put(ctx, invoicer.ID(env), env); err != nil {
			log.Printf("seed: invoice template %d store: %v", i, err)
			continue
		}
		stored++
		}
	}
	if !present[invoicer.KindOrder] {
		for i, tpl := range orderTemplates {
		ord := new(bill.Order)
		if err := json.Unmarshal([]byte(tpl), ord); err != nil {
			log.Printf("seed: order template %d decode: %v", i, err)
			continue
		}
		env, err := invoicer.BuildOrder(ord)
		if err != nil {
			log.Printf("seed: order template %d build: %v", i, err)
			continue
		}
		if err := entries.Put(ctx, invoicer.ID(env), env); err != nil {
			log.Printf("seed: order template %d store: %v", i, err)
			continue
		}
		stored++
		}
	}
	if !present[invoicer.KindDelivery] {
		for i, tpl := range deliveryTemplates {
		dlv := new(bill.Delivery)
		if err := json.Unmarshal([]byte(tpl), dlv); err != nil {
			log.Printf("seed: delivery template %d decode: %v", i, err)
			continue
		}
		env, err := invoicer.BuildDelivery(dlv)
		if err != nil {
			log.Printf("seed: delivery template %d build: %v", i, err)
			continue
		}
		if err := entries.Put(ctx, invoicer.ID(env), env); err != nil {
			log.Printf("seed: delivery template %d store: %v", i, err)
			continue
		}
		stored++
		}
	}
	if !present[invoicer.KindPayment] {
		for i, tpl := range paymentTemplates {
		pmt := new(bill.Payment)
		if err := json.Unmarshal([]byte(tpl), pmt); err != nil {
			log.Printf("seed: payment template %d decode: %v", i, err)
			continue
		}
		env, err := invoicer.BuildPayment(pmt)
		if err != nil {
			log.Printf("seed: payment template %d build: %v", i, err)
			continue
		}
		if err := entries.Put(ctx, invoicer.ID(env), env); err != nil {
			log.Printf("seed: payment template %d store: %v", i, err)
			continue
		}
		stored++
		}
	}
	if !present[invoicer.KindStatus] {
		for i, tpl := range statusTemplates {
		st := new(bill.Status)
		if err := json.Unmarshal([]byte(tpl), st); err != nil {
			log.Printf("seed: status template %d decode: %v", i, err)
			continue
		}
		env, err := invoicer.BuildStatus(st)
		if err != nil {
			log.Printf("seed: status template %d build: %v", i, err)
			continue
		}
		if err := entries.Put(ctx, invoicer.ID(env), env); err != nil {
			log.Printf("seed: status template %d store: %v", i, err)
			continue
		}
		stored++
		}
	}
	if stored > 0 {
		log.Printf("seed: stored %d demo documents", stored)
	}
}
