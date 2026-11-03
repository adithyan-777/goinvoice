// Package invoicer wraps the GOBL pipeline: build an envelope from a bill.Invoice,
// calculate totals + taxes, validate against the regime, and render HTML.
package invoicer

import (
	"context"
	"fmt"
	"time"

	"github.com/invopop/ctxi18n/i18n"
	"github.com/invopop/gobl"
	goblhtml "github.com/invopop/gobl.html"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/org"
)

// Build runs the full GOBL pipeline on a client-supplied invoice:
// envelope → calculate → validate. The returned envelope contains the
// computed totals, tax breakdown, and digest — treat it as the source of truth.
func Build(inv *bill.Invoice) (*gobl.Envelope, error) {
	env := gobl.NewEnvelope()

	if err := env.Insert(inv); err != nil {
		return nil, fmt.Errorf("insert document: %w", err)
	}
	if err := env.Calculate(); err != nil {
		return nil, fmt.Errorf("calculate: %w", err)
	}
	if err := env.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}
	return env, nil
}

// ID returns the envelope's UUID, used as the storage key.
func ID(env *gobl.Envelope) string {
	if env == nil || env.Head == nil {
		return ""
	}
	return env.Head.UUID.String()
}

// Summary is a compact projection of a stored invoice for list endpoints.
type Summary struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Series    string `json:"series,omitempty"`
	Code      string `json:"code,omitempty"`
	Currency  string `json:"currency"`
	IssueDate string `json:"issue_date,omitempty"`
	Supplier  string `json:"supplier,omitempty"`
	Customer  string `json:"customer,omitempty"`
	Sum       string `json:"sum,omitempty"`     // lines total before discounts/taxes
	Total     string `json:"total,omitempty"`   // net total
	Tax       string `json:"tax,omitempty"`     // tax total
	Payable   string `json:"payable,omitempty"` // total including tax
	Due       string `json:"due,omitempty"`     // remaining to pay
	Digest    string `json:"digest,omitempty"`  // envelope digest (tamper-evidence)
}

// Summarize builds a Summary from a calculated envelope. Returns nil if the
// envelope does not contain a bill.Invoice.
func Summarize(env *gobl.Envelope) *Summary {
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok {
		return nil
	}

	s := &Summary{
		ID:       ID(env),
		Type:     inv.Type.String(),
		Series:   inv.Series.String(),
		Code:     inv.Code.String(),
		Currency: inv.Currency.String(),
	}
	if !inv.IssueDate.IsZero() {
		s.IssueDate = inv.IssueDate.String()
	}
	if inv.Supplier != nil {
		s.Supplier = inv.Supplier.Name
	}
	if inv.Customer != nil {
		s.Customer = inv.Customer.Name
	}
	if t := inv.Totals; t != nil {
		s.Sum = t.Sum.String()
		s.Total = t.Total.String()
		s.Tax = t.Tax.String()
		s.Payable = t.Payable.String()
		if t.Due != nil {
			s.Due = t.Due.String()
		}
	}
	if d, err := env.Digest(); err == nil && d != nil {
		s.Digest = d.Value
	}
	return s
}

// RenderOptions controls how the HTML document is generated.
type RenderOptions struct {
	// Locale for translations, e.g. "en", "es", "fr". Defaults to "en".
	Locale string
	// Watermark text, e.g. "DRAFT".
	Watermark string
	// Sandbox renders a simplified document without legal numbering.
	Sandbox bool
}

// Render converts a GOBL envelope into a standalone printable HTML document.
// Stylesheets are embedded so the output works with no external requests,
// which is what you want before converting to PDF.
func Render(ctx context.Context, env *gobl.Envelope, o RenderOptions) ([]byte, error) {
	opts := []goblhtml.Option{
		goblhtml.WithEmbeddedAssets(),
	}
	// Simple, human-friendly dates.
	opts = append(opts, goblhtml.WithCalFormatter("2006-01-02", "15:04", time.UTC))
	if o.Locale != "" {
		opts = append(opts, goblhtml.WithLocale(i18n.Code(o.Locale)))
	}
	if o.Watermark != "" {
		opts = append(opts, goblhtml.WithWatermark(o.Watermark))
	}
	if o.Sandbox {
		opts = append(opts, goblhtml.WithSandbox(true))
	}
	return goblhtml.Render(ctx, env, opts...)
}

// LogoFromEnvelope extracts the supplier's first logo from the envelope, if
// one exists, for passing to goblhtml.WithLogo when needed.
func LogoFromEnvelope(env *gobl.Envelope) *org.Image {
	inv, ok := env.Extract().(*bill.Invoice)
	if !ok || inv.Supplier == nil {
		return nil
	}
	if len(inv.Supplier.Logos) > 0 {
		return inv.Supplier.Logos[0]
	}
	return nil
}
