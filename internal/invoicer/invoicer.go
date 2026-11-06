// Package invoicer wraps the GOBL pipeline: build an envelope from any
// bill document (invoice, order, delivery, payment, status),
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

// BuildAny runs the full GOBL pipeline on any client-supplied bill document
// (*bill.Invoice, *bill.Order, *bill.Delivery, *bill.Payment, *bill.Status):
// envelope → calculate → validate. The returned envelope contains the
// computed totals, tax breakdown, and digest — treat it as the source of truth.
func BuildAny(doc any) (*gobl.Envelope, error) {
	env := gobl.NewEnvelope()

	if err := env.Insert(doc); err != nil {
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

// Build runs the full GOBL pipeline on a client-supplied invoice:
// envelope → calculate → validate. The returned envelope contains the
// computed totals, tax breakdown, and digest — treat it as the source of truth.
func Build(inv *bill.Invoice) (*gobl.Envelope, error) {
	return BuildAny(inv)
}

// BuildOrder runs the GOBL pipeline on a client-supplied order.
func BuildOrder(ord *bill.Order) (*gobl.Envelope, error) {
	return BuildAny(ord)
}

// BuildDelivery runs the GOBL pipeline on a client-supplied delivery.
func BuildDelivery(dlv *bill.Delivery) (*gobl.Envelope, error) {
	return BuildAny(dlv)
}

// BuildPayment runs the GOBL pipeline on a client-supplied payment.
func BuildPayment(pmt *bill.Payment) (*gobl.Envelope, error) {
	return BuildAny(pmt)
}

// BuildStatus runs the GOBL pipeline on a client-supplied status.
func BuildStatus(st *bill.Status) (*gobl.Envelope, error) {
	return BuildAny(st)
}

// ID returns the envelope's UUID, used as the storage key.
func ID(env *gobl.Envelope) string {
	if env == nil || env.Head == nil {
		return ""
	}
	return env.Head.UUID.String()
}

// Document kinds stored in the shared envelope store.
const (
	KindInvoice  = "invoice"
	KindOrder    = "order"
	KindDelivery = "delivery"
	KindPayment  = "payment"
	KindStatus   = "status"
)

// Summary is a compact projection of a stored document for list endpoints.
type Summary struct {
	ID        string `json:"id"`
	Document  string `json:"document"` // invoice|order|delivery|payment|status
	Type      string `json:"type"`
	Series    string `json:"series,omitempty"`
	Code      string `json:"code,omitempty"`
	Currency  string `json:"currency,omitempty"`
	IssueDate string `json:"issue_date,omitempty"`
	Supplier  string `json:"supplier,omitempty"`
	Customer  string `json:"customer,omitempty"`
	Sum       string `json:"sum,omitempty"`     // lines total before discounts/taxes
	Total     string `json:"total,omitempty"`   // net total (payment: total paid)
	Tax       string `json:"tax,omitempty"`     // tax total
	Payable   string `json:"payable,omitempty"` // total including tax
	Due       string `json:"due,omitempty"`     // remaining to pay
	Digest    string `json:"digest,omitempty"`  // envelope digest (tamper-evidence)
}

func fillDigest(env *gobl.Envelope, s *Summary) {
	if d, err := env.Digest(); err == nil && d != nil {
		s.Digest = d.Value
	}
}

func fillTotals(s *Summary, t *bill.Totals) {
	if t == nil {
		return
	}
	s.Sum = t.Sum.String()
	s.Total = t.Total.String()
	s.Tax = t.Tax.String()
	s.Payable = t.Payable.String()
	if t.Due != nil {
		s.Due = t.Due.String()
	}
}

// DocumentKind returns the bill document kind inside the envelope:
// invoice, order, delivery, payment, status, or "" when unknown.
func DocumentKind(env *gobl.Envelope) string {
	if env == nil {
		return ""
	}
	switch env.Extract().(type) {
	case *bill.Invoice:
		return KindInvoice
	case *bill.Order:
		return KindOrder
	case *bill.Delivery:
		return KindDelivery
	case *bill.Payment:
		return KindPayment
	case *bill.Status:
		return KindStatus
	default:
		return ""
	}
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
		Document: KindInvoice,
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
	fillTotals(s, inv.Totals)
	fillDigest(env, s)
	return s
}

// SummarizeOrder builds a Summary from an envelope holding a bill.Order.
func SummarizeOrder(env *gobl.Envelope) *Summary {
	ord, ok := env.Extract().(*bill.Order)
	if !ok {
		return nil
	}
	s := &Summary{
		ID:       ID(env),
		Document: KindOrder,
		Type:     ord.Type.String(),
		Series:   ord.Series.String(),
		Code:     ord.Code.String(),
		Currency: ord.Currency.String(),
	}
	if !ord.IssueDate.IsZero() {
		s.IssueDate = ord.IssueDate.String()
	}
	if ord.Supplier != nil {
		s.Supplier = ord.Supplier.Name
	}
	if ord.Customer != nil {
		s.Customer = ord.Customer.Name
	}
	fillTotals(s, ord.Totals)
	fillDigest(env, s)
	return s
}

// SummarizeDelivery builds a Summary from an envelope holding a bill.Delivery.
func SummarizeDelivery(env *gobl.Envelope) *Summary {
	dlv, ok := env.Extract().(*bill.Delivery)
	if !ok {
		return nil
	}
	s := &Summary{
		ID:       ID(env),
		Document: KindDelivery,
		Type:     dlv.Type.String(),
		Series:   dlv.Series.String(),
		Code:     dlv.Code.String(),
		Currency: dlv.Currency.String(),
	}
	if !dlv.IssueDate.IsZero() {
		s.IssueDate = dlv.IssueDate.String()
	}
	if dlv.Supplier != nil {
		s.Supplier = dlv.Supplier.Name
	}
	if dlv.Customer != nil {
		s.Customer = dlv.Customer.Name
	}
	fillTotals(s, dlv.Totals)
	fillDigest(env, s)
	return s
}

// SummarizePayment builds a Summary from an envelope holding a bill.Payment.
// Payments carry a single Total (no tax breakdown), mapped to Total/Payable.
func SummarizePayment(env *gobl.Envelope) *Summary {
	pmt, ok := env.Extract().(*bill.Payment)
	if !ok {
		return nil
	}
	s := &Summary{
		ID:       ID(env),
		Document: KindPayment,
		Type:     pmt.Type.String(),
		Series:   pmt.Series.String(),
		Code:     pmt.Code.String(),
		Currency: pmt.Currency.String(),
	}
	if !pmt.IssueDate.IsZero() {
		s.IssueDate = pmt.IssueDate.String()
	}
	if pmt.Supplier != nil {
		s.Supplier = pmt.Supplier.Name
	}
	if pmt.Customer != nil {
		s.Customer = pmt.Customer.Name
	}
	s.Total = pmt.Total.String()
	s.Payable = pmt.Total.String()
	fillDigest(env, s)
	return s
}

// SummarizeStatus builds a Summary from an envelope holding a bill.Status.
// Statuses carry no monetary totals.
func SummarizeStatus(env *gobl.Envelope) *Summary {
	st, ok := env.Extract().(*bill.Status)
	if !ok {
		return nil
	}
	s := &Summary{
		ID:       ID(env),
		Document: KindStatus,
		Type:     st.Type.String(),
		Series:   st.Series.String(),
		Code:     st.Code.String(),
	}
	if !st.IssueDate.IsZero() {
		s.IssueDate = st.IssueDate.String()
	}
	if st.Supplier != nil {
		s.Supplier = st.Supplier.Name
	}
	if st.Customer != nil {
		s.Customer = st.Customer.Name
	}
	fillDigest(env, s)
	return s
}

// SummarizeAny builds a Summary for any supported bill document.
// Returns nil for unknown document types.
func SummarizeAny(env *gobl.Envelope) *Summary {
	switch DocumentKind(env) {
	case KindInvoice:
		return Summarize(env)
	case KindOrder:
		return SummarizeOrder(env)
	case KindDelivery:
		return SummarizeDelivery(env)
	case KindPayment:
		return SummarizePayment(env)
	case KindStatus:
		return SummarizeStatus(env)
	default:
		return nil
	}
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
// Works for invoice, order, delivery, payment and status documents.
func LogoFromEnvelope(env *gobl.Envelope) *org.Image {
	var supplier *org.Party
	switch doc := env.Extract().(type) {
	case *bill.Invoice:
		if doc == nil {
			return nil
		}
		supplier = doc.Supplier
	case *bill.Order:
		if doc == nil {
			return nil
		}
		supplier = doc.Supplier
	case *bill.Delivery:
		if doc == nil {
			return nil
		}
		supplier = doc.Supplier
	case *bill.Payment:
		if doc == nil {
			return nil
		}
		supplier = doc.Supplier
	case *bill.Status:
		if doc == nil {
			return nil
		}
		supplier = doc.Supplier
	default:
		return nil
	}
	if supplier == nil {
		return nil
	}
	if len(supplier.Logos) > 0 {
		return supplier.Logos[0]
	}
	return nil
}
