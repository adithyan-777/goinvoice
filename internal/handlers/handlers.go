// Package handlers defines the Chi routes and their implementations.
// HTML pages are rendered with templ (see index.templ, run `templ generate`
// after editing). Invoice documents themselves are rendered by gobl.html.
//
//go:generate templ generate
package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"

	"github.com/neo/goinvoice/internal/db/sqlc"
	"github.com/neo/goinvoice/internal/invoicer"
	"github.com/neo/goinvoice/internal/store"
)

// Handler bundles dependencies used by the route implementations.
type Handler struct {
	entries store.Entries
	catalog store.Catalog
}

// New wires a handler with the given stores (memory or SQLite).
func New(entries store.Entries, catalog store.Catalog) *Handler {
	return &Handler{entries: entries, catalog: catalog}
}

// Register mounts all invoice routes on the router.
func (h *Handler) Register(r chi.Router) {
	r.Get("/", h.Index)
	r.Get("/company", h.Company)
	r.Post("/company", h.SaveCompany)
	r.Get("/customers", h.Customers)
	r.Post("/customers", h.CreateCustomer)
	r.Get("/customers/new", h.NewCustomerPage)
	r.Get("/customers/{id}/edit", h.EditCustomerForm)
	r.Post("/customers/{id}", h.UpdateCustomer)
	r.Post("/customers/{id}/delete", h.DeleteCustomerForm)
	r.Delete("/customers/{id}", h.DeleteCustomer)
	r.Get("/suppliers", h.Suppliers)
	r.Post("/suppliers", h.CreateSupplier)
	r.Get("/suppliers/new", h.NewSupplierPage)
	r.Get("/suppliers/{id}/edit", h.EditSupplierForm)
	r.Post("/suppliers/{id}", h.UpdateSupplier)
	r.Post("/suppliers/{id}/delete", h.DeleteSupplierForm)
	r.Delete("/suppliers/{id}", h.DeleteSupplier)
	r.Get("/products", h.Products)
	r.Post("/products", h.CreateProduct)
	r.Get("/products/new", h.NewProductPage)
	r.Get("/products/{id}/edit", h.EditProductForm)
	r.Post("/products/{id}", h.UpdateProduct)
	r.Post("/products/{id}/delete", h.DeleteProductForm)
	r.Delete("/products/{id}", h.DeleteProduct)
	r.Get("/partials/invoices", h.InvoicesPartial)
	r.Get("/partials/orders", h.OrdersPartial)
	r.Get("/partials/deliveries", h.DeliveriesPartial)
	r.Get("/partials/payments", h.PaymentsPartial)
	r.Get("/partials/statuses", h.StatusesPartial)
	r.Get("/partials/invoices/new", h.NewInvoiceModal)
	r.Get("/partials/invoices/{id}/edit", h.EditInvoiceModal)
	r.Get("/partials/orders/new", h.NewOrderModal)
	r.Get("/partials/orders/{id}/edit", h.EditOrderModal)
	r.Get("/partials/deliveries/new", h.NewDeliveryModal)
	r.Get("/partials/deliveries/{id}/edit", h.EditDeliveryModal)
	r.Get("/partials/payments/new", h.NewPaymentModal)
	r.Get("/partials/payments/{id}/edit", h.EditPaymentModal)
	r.Get("/partials/statuses/new", h.NewStatusModal)
	r.Get("/partials/statuses/{id}/edit", h.EditStatusModal)
	r.Get("/partials/customers", h.CustomersPartial)
	r.Get("/partials/company", h.CompanyPartial)
	r.Get("/partials/suppliers", h.SuppliersPartial)
	r.Get("/partials/products", h.ProductsPartial)
	r.Get("/partials/products/new", h.NewProductModal)
	r.Get("/partials/products/{id}/form", h.ProductModal)
	r.Get("/partials/customers/new", h.NewCustomerModal)
	r.Get("/partials/customers/{id}/form", h.CustomerModal)
	r.Get("/partials/suppliers/new", h.NewSupplierModal)
	r.Get("/partials/suppliers/{id}/form", h.SupplierModal)
	r.Route("/invoices", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/new", h.NewInvoicePage)
		r.Get("/{id}", h.Get)
		r.Get("/{id}/html", h.HTML)
		r.Get("/{id}/edit", h.EditInvoiceForm)
		r.Post("/{id}", h.UpdateInvoice)
		r.Put("/{id}", h.UpdateInvoice)
		r.Post("/{id}/delete", h.DeleteInvoiceForm)
		r.Delete("/{id}", h.Delete)
	})
	r.Route("/orders", func(r chi.Router) {
		r.Post("/", h.CreateOrder)
		r.Get("/", h.ListOrders)
		r.Get("/new", h.NewOrderPage)
		r.Get("/{id}", h.GetOrder)
		r.Get("/{id}/html", h.OrderHTML)
		r.Get("/{id}/edit", h.EditOrderForm)
		r.Post("/{id}", h.UpdateOrder)
		r.Put("/{id}", h.UpdateOrder)
		r.Post("/{id}/delete", h.DeleteOrderForm)
		r.Delete("/{id}", h.DeleteOrder)
	})
	r.Route("/deliveries", func(r chi.Router) {
		r.Post("/", h.CreateDelivery)
		r.Get("/", h.ListDeliveries)
		r.Get("/new", h.NewDeliveryPage)
		r.Get("/{id}", h.GetDelivery)
		r.Get("/{id}/html", h.DeliveryHTML)
		r.Get("/{id}/edit", h.EditDeliveryForm)
		r.Post("/{id}", h.UpdateDelivery)
		r.Put("/{id}", h.UpdateDelivery)
		r.Post("/{id}/delete", h.DeleteDeliveryForm)
		r.Delete("/{id}", h.DeleteDelivery)
	})
	r.Route("/payments", func(r chi.Router) {
		r.Post("/", h.CreatePayment)
		r.Get("/", h.ListPayments)
		r.Get("/new", h.NewPaymentPage)
		r.Get("/{id}", h.GetPayment)
		r.Get("/{id}/html", h.PaymentHTML)
		r.Get("/{id}/edit", h.EditPaymentForm)
		r.Post("/{id}", h.UpdatePayment)
		r.Put("/{id}", h.UpdatePayment)
		r.Post("/{id}/delete", h.DeletePaymentForm)
		r.Delete("/{id}", h.DeletePayment)
	})
	r.Route("/statuses", func(r chi.Router) {
		r.Post("/", h.CreateStatus)
		r.Get("/", h.ListStatuses)
		r.Get("/new", h.NewStatusPage)
		r.Get("/{id}", h.GetStatus)
		r.Get("/{id}/html", h.StatusHTML)
		r.Get("/{id}/edit", h.EditStatusForm)
		r.Post("/{id}", h.UpdateStatus)
		r.Put("/{id}", h.UpdateStatus)
		r.Post("/{id}/delete", h.DeleteStatusForm)
		r.Delete("/{id}", h.DeleteStatus)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// Create handles POST /invoices from JSON API clients and the UI form.
//
// Accepts a bill.Invoice-shaped JSON body (or structured form fields from
// the UI). GOBL fills in totals and taxes; validation errors are reported
// with 422 so clients can distinguish them from malformed JSON (400).
//
// Query params:
//   - dry_run=true  → calculate + validate, but don't store. Returns the envelope.
//   - html=true     → also store AND return the rendered HTML document.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	h.createBillDoc(invoicer.KindInvoice)(w, r)
}

// List handles GET /invoices: browsers get the invoices page, API clients
// get compact summaries as JSON.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	raw, err := h.entries.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]*invoicer.Summary, 0, len(raw))
	for _, e := range raw {
		if s := invoicer.Summarize(e.Envelope); s != nil {
			out = append(out, s)
		}
	}
	if isBrowserPage(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := InvoicesPage(out, invoicer.SummarizeStats(out)).Render(r.Context(), w); err != nil {
			log.Printf("render invoices page: %v", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// Get handles GET /invoices/{id} and returns the full envelope as JSON.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	env, err := h.entries.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if invoicer.DocumentKind(env) != invoicer.KindInvoice {
		writeError(w, http.StatusNotFound, store.ErrNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "envelope": env})
}

// HTML handles GET /invoices/{id}/html?locale=es&watermark=DRAFT&sandbox=true
// and returns a printable HTML document with a client-side "Download PDF"
// button (html2pdf.js via CDN, no server-side converter needed).
func (h *Handler) HTML(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	env, err := h.entries.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if invoicer.DocumentKind(env) != invoicer.KindInvoice {
		writeError(w, http.StatusNotFound, store.ErrNotFound)
		return
	}

	opts := invoicer.RenderOptions{
		Locale:    r.URL.Query().Get("locale"),
		Watermark: r.URL.Query().Get("watermark"),
		Sandbox:   r.URL.Query().Get("sandbox") == "true",
	}

	html, err := invoicer.Render(r.Context(), env, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	html = stripGoblBranding(html)
	html = injectPDFToolbar(html, id)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(html)
}

// goblBranding lists the gobl.html footer branding removed from invoice
// pages: the logo div (linking to gobl.org) and the screen footer that
// contains nothing else. The print footer's page numbers are kept.
var (
	goblLogoRe         = regexp.MustCompile(`(?s)<div class="gobl-logo">.*?</div>`)
	goblScreenFooterRe = regexp.MustCompile(`(?s)<footer class="screen">\s*</footer>`)
	goblLogoCSSRe      = regexp.MustCompile(`(?s)footer[^{]*\.gobl-logo[^{]*\{[^}]*\}`)
)

// stripGoblBranding removes the GOBL logo and empty screen footer from a
// rendered invoice document.
func stripGoblBranding(html []byte) []byte {
	html = goblLogoRe.ReplaceAll(html, nil)
	html = goblScreenFooterRe.ReplaceAll(html, nil)
	return goblLogoCSSRe.ReplaceAll(html, nil)
}

// pdfToolbarCDN is the client-side lib used for in-browser PDF export:
// html2pdf.js bundle (html2canvas + jsPDF) converts the invoice to PDF.
const html2pdfCDN = "https://cdnjs.cloudflare.com/ajax/libs/html2pdf.js/0.10.1/html2pdf.bundle.min.js"

// injectPDFToolbar appends a floating "Download PDF" button plus the CDN
// scripts before </body>. The button converts the gobl.html
// `article.envelope` element to PDF entirely in the browser.
func injectPDFToolbar(html []byte, id string) []byte {
	toolbar := `<div class="pdf-toolbar" style="position:fixed;top:1rem;right:1rem;z-index:9999;">` +
		`<button id="download-pdf" type="button" style="padding:.5rem 1rem;font-size:.9rem;cursor:pointer;">Download PDF</button></div>` +
		`<style>@media print{.pdf-toolbar{display:none!important}}</style>` +
		`<script src="` + html2pdfCDN + `"></script>` +
		`<script>(function(){` +
		`var btn=document.getElementById("download-pdf");` +
		`if(!btn)return;` +
		`btn.addEventListener("click",function(){` +
		`var el=document.querySelector("article.envelope")||document.body;` +
		`btn.disabled=true;btn.textContent="Generating…";` +
		`var opt={margin:10,filename:"invoice-` + id + `.pdf",` +
		`image:{type:"jpeg",quality:0.95},html2canvas:{scale:2},jsPDF:{unit:"mm",format:"a4",orientation:"portrait"}};` +
		`html2pdf().set(opt).from(el).save().then(function(){btn.disabled=false;btn.textContent="Download PDF";})` +
		`.catch(function(){btn.disabled=false;btn.textContent="Download PDF";});` +
		`});})();</script>`
	if i := bytes.LastIndex(html, []byte("</body>")); i >= 0 {
		out := make([]byte, 0, len(html)+len(toolbar))
		out = append(out, html[:i]...)
		out = append(out, toolbar...)
		out = append(out, html[i:]...)
		return out
	}
	return append(html, toolbar...)
}

// Delete handles DELETE /invoices/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	env, err := h.entries.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if invoicer.DocumentKind(env) != invoicer.KindInvoice {
		writeError(w, http.StatusNotFound, store.ErrNotFound)
		return
	}
	ok, err := h.entries.Delete(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, store.ErrNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Customers handles GET /customers and renders the customer catalog.
func (h *Handler) Customers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.catalog.ListCustomers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := CustomersPage(rows, r.URL.Query().Get("error")).Render(r.Context(), w); err != nil {
		log.Printf("render customers: %v", err)
	}
}

// company returns the single issuing company (the first supplier row) and its
// party, or a nil party when none exists yet.
func (h *Handler) company(ctx context.Context) (string, *org.Party, error) {
	rows, err := h.catalog.ListSuppliers(ctx)
	if err != nil {
		return "", nil, err
	}
	if len(rows) == 0 {
		return "", nil, nil
	}
	id := rows[0].ID
	p, err := h.catalog.GetSupplierParty(ctx, id)
	if err != nil {
		return "", nil, err
	}
	return id, p, nil
}

// Suppliers now redirects to the single-company details page; the supplier
// API endpoints remain available below.
func (h *Handler) Suppliers(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/company", http.StatusSeeOther)
}

// Company handles GET /company and renders the single issuing company.
func (h *Handler) Company(w http.ResponseWriter, r *http.Request) {
	_, p, err := h.company(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := CompanyPage(partyFormData(p, "", r.URL.Query().Get("error"))).Render(r.Context(), w); err != nil {
		log.Printf("render company: %v", err)
	}
}

// CompanyPartial handles GET /partials/company for in-place navigation.
func (h *Handler) CompanyPartial(w http.ResponseWriter, r *http.Request) {
	_, p, err := h.company(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	respondFragment(w, r, "#main", CompanySection(partyFormData(p, "", "")))
}

// SaveCompany handles POST /company. It updates the single company, creating
// it on first save. The UI never creates a second supplier.
func (h *Handler) SaveCompany(w http.ResponseWriter, r *http.Request) {
	f, isJSON, err := decodeParty(r)
	if err != nil {
		if isJSON {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if wantsFragment(r) {
			respondFragment(w, r, "#main", CompanySection(partyFormData(f, "", err.Error())))
			return
		}
		http.Redirect(w, r, "/company?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	id, existing, err := h.company(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	next := f
	if existing != nil {
		applyPartyForm(existing, f)
		next = existing
	} else {
		id = uuid.NewString()
	}
	if err := h.catalog.SaveSupplier(r.Context(), id, next); err != nil {
		if isJSON {
			if productConflict(err) {
				writeError(w, http.StatusConflict, errors.New("company tax code already exists"))
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		msg := conflictMessage(err, "company tax code already exists")
		if wantsFragment(r) {
			respondFragment(w, r, "#main", CompanySection(partyFormData(next, "", msg)))
			return
		}
		http.Redirect(w, r, "/company?error="+url.QueryEscape(msg), http.StatusSeeOther)
		return
	}
	if isJSON {
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "name": next.Name})
		return
	}
	if wantsFragment(r) {
		respondFragment(w, r, "#main", CompanySection(partyFormData(next, "", "")))
		return
	}
	http.Redirect(w, r, "/company", http.StatusSeeOther)
}

// Products handles GET /products and renders the product catalog.
func (h *Handler) Products(w http.ResponseWriter, r *http.Request) {
	rows, err := h.catalog.ListProducts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ProductsPage(rows, r.URL.Query().Get("error")).Render(r.Context(), w); err != nil {
		log.Printf("render products: %v", err)
	}
}

// NewCustomerPage handles GET /customers/new (no-JS fallback for the modal).
func (h *Handler) NewCustomerPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := CustomerNewPage(partyFormData(nil, "", r.URL.Query().Get("error"))).Render(r.Context(), w); err != nil {
		log.Printf("render new customer: %v", err)
	}
}

// NewSupplierPage handles GET /suppliers/new (no-JS fallback for the modal).
func (h *Handler) NewSupplierPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := SupplierNewPage(partyFormData(nil, "", r.URL.Query().Get("error"))).Render(r.Context(), w); err != nil {
		log.Printf("render new supplier: %v", err)
	}
}

// NewProductPage handles GET /products/new (no-JS fallback for the modal).
func (h *Handler) NewProductPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	it := &org.Item{Currency: "EUR"}
	if err := ProductNewPage(productRow(it, "")).Render(r.Context(), w); err != nil {
		log.Printf("render new product: %v", err)
	}
}

// nullIfEmpty maps an optional form value to a nullable column.
func nullIfEmpty(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// productConflict reports whether err is a UNIQUE violation (duplicate ref).
func productConflict(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint")
}

var errProductRequired = errors.New("ref and name are required")

// uneceUnitPattern matches legacy UN/ECE unit codes, which GOBL accepts and
// normalizes into the `untdid-unit` extension.
var uneceUnitPattern = regexp.MustCompile(`^[A-Z0-9]{2,3}$`)

// validItemUnit reports whether u is a GOBL unit key or a legacy UN/ECE code.
func validItemUnit(u cbc.Key) bool {
	if u == "" {
		return true
	}
	return org.HasValidUnitKey.Check(u) || uneceUnitPattern.MatchString(u.String())
}

// unitOptions lists the GOBL units for the product form dropdown.
func unitOptions() []DocOption {
	opts := []DocOption{{Value: "", Label: "None"}}
	for _, d := range org.UnitDefinitions {
		opts = append(opts, DocOption{
			Value: d.Key.String(),
			Label: d.Name.String() + " (" + d.Key.String() + ")",
		})
	}
	return opts
}

// unitLabel returns the display label for a unit key.
func unitLabel(key string) string {
	if key == "" {
		return ""
	}
	for _, o := range unitOptions() {
		if o.Value == key {
			return o.Label
		}
	}
	return key
}

// decodeProduct reads a GOBL org.Item from a JSON body (full shape) or from
// the flat product form (ref, name, description, pricing, unit, origin).
func decodeProduct(r *http.Request) (*org.Item, bool, error) {
	isJSON := strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")
	if isJSON {
		it := new(org.Item)
		if err := json.NewDecoder(r.Body).Decode(it); err != nil {
			return nil, true, err
		}
		if strings.TrimSpace(it.Ref.String()) == "" || strings.TrimSpace(it.Name) == "" {
			return it, true, errProductRequired
		}
		return it, true, nil
	}
	if err := r.ParseForm(); err != nil {
		return nil, false, err
	}
	it, err := productFromForm(r)
	if err != nil {
		return it, false, err
	}
	if strings.TrimSpace(it.Ref.String()) == "" || strings.TrimSpace(it.Name) == "" {
		return it, false, errProductRequired
	}
	return it, false, nil
}

func formAmount(v string) (*num.Amount, error) {
	if v == "" {
		return nil, nil
	}
	a, err := num.AmountFromString(v)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// productFromForm maps the flat product form onto GOBL org.Item fields.
func productFromForm(r *http.Request) (*org.Item, error) {
	get := func(k string) string { return strings.TrimSpace(r.FormValue(k)) }
	it := &org.Item{
		Ref:         cbc.Code(get("ref")),
		Key:         cbc.Key(get("key")),
		Name:        get("name"),
		Description: get("description"),
		Currency:    currency.Code(get("currency")),
		Unit:        cbc.Key(get("unit")),
		Origin:      l10n.ISOCountryCode(get("origin")),
	}
	var err error
	if it.Price, err = formAmount(get("price")); err != nil {
		return it, fmt.Errorf("price: %w", err)
	}
	if it.List, err = formAmount(get("list")); err != nil {
		return it, fmt.Errorf("list: %w", err)
	}
	if it.Discount, err = formAmount(get("discount")); err != nil {
		return it, fmt.Errorf("discount: %w", err)
	}
	if it.Per, err = formAmount(get("per")); err != nil {
		return it, fmt.Errorf("per: %w", err)
	}
	if it.Currency == "" {
		it.Currency = "EUR"
	}
	if it.Unit != "" && !validItemUnit(it.Unit) {
		return it, fmt.Errorf("unit %q is not a valid GOBL unit key (try unit, item, piece, h, day, kg, m…)", it.Unit)
	}
	if it.Key != "" && !cbc.KeyValidationRegexp.MatchString(it.Key.String()) {
		return it, fmt.Errorf("key %q must be lowercase letters, digits or dashes", it.Key)
	}
	return it, nil
}

// applyProductForm merges the flat form scalars into an existing item while
// preserving collections created through the JSON API.
func applyProductForm(dst, src *org.Item) {
	dst.Ref, dst.Key, dst.Name, dst.Description = src.Ref, src.Key, src.Name, src.Description
	dst.Currency, dst.Price = src.Currency, src.Price
	dst.List, dst.Discount, dst.Per = src.List, src.Discount, src.Per
	dst.Unit, dst.Origin = src.Unit, src.Origin
}

// productRow projects an item into the sqlc.Product shape used to re-render
// the form after a validation error.
func productRow(it *org.Item, id string) sqlc.Product {
	if it == nil {
		it = &org.Item{}
	}
	amt := func(a *num.Amount) sql.NullString {
		if a == nil {
			return sql.NullString{}
		}
		return nullIfEmpty(a.String())
	}
	return sqlc.Product{
		ID:          id,
		Ref:         it.Ref.String(),
		Name:        it.Name,
		Description: nullIfEmpty(it.Description),
		Price:       amountOrEmpty(it.Price),
		Currency:    it.Currency.String(),
		Unit:        nullIfEmpty(it.Unit.String()),
		Key:         nullIfEmpty(it.Key.String()),
		List:        amt(it.List),
		Discount:    amt(it.Discount),
		Per:         amt(it.Per),
		Origin:      nullIfEmpty(it.Origin.String()),
	}
}

func amountOrEmpty(a *num.Amount) string {
	if a == nil {
		return ""
	}
	return a.String()
}

// wantsFragment reports whether the request came from Datastar (SPA mode)
// and expects an HTML fragment instead of a redirect or full page.
func wantsFragment(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "text/event-stream")
}

// renderCustomersFragment swaps the customers section in place and closes any
// open modal (used by the modal create/edit forms).
func (h *Handler) renderCustomersFragment(w http.ResponseWriter, r *http.Request, errMsg string) {
	rows, err := h.catalog.ListCustomers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	respondSection(w, r, CustomersSection(rows, errMsg))
}

// renderSuppliersFragment swaps the suppliers section in place and closes any
// open modal (used by the modal create/edit forms).
func (h *Handler) renderSuppliersFragment(w http.ResponseWriter, r *http.Request, errMsg string) {
	rows, err := h.catalog.ListSuppliers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	respondSection(w, r, SuppliersSection(rows, errMsg))
}

// renderProductsFragment swaps the products section in place and closes any
// open modal (used by the modal create/edit forms).
func (h *Handler) renderProductsFragment(w http.ResponseWriter, r *http.Request, errMsg string) {
	rows, err := h.catalog.ListProducts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	respondSection(w, r, ProductsSection(rows, errMsg))
}

// conflictMessage maps UNIQUE violations to a friendly duplicate message.
func conflictMessage(err error, dup string) string {
	if productConflict(err) {
		return dup
	}
	return err.Error()
}

// CreateProduct handles POST /products, from both the HTML form and JSON
// API clients (Content-Type: application/json).
//
// Datastar form posts (SPA mode) swap the section fragment in place;
// plain form posts follow PRG with redirects. JSON clients get
// 201/400/409 JSON.
func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	it, isJSON, err := decodeProduct(r)
	if err != nil {
		if isJSON {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if wantsFragment(r) {
			respondFragment(w, r, "#modal-body", ProductForm(productRow(it, ""), err.Error()))
			return
		}
		http.Redirect(w, r, "/products?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	id := uuid.NewString()

	if err := h.catalog.SaveProduct(r.Context(), id, it); err != nil {
		if isJSON {
			if productConflict(err) {
				writeError(w, http.StatusConflict, errors.New("product ref already exists"))
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		msg := conflictMessage(err, "product ref already exists")
		if wantsFragment(r) {
			respondFragment(w, r, "#modal-body", ProductForm(productRow(it, ""), msg))
			return
		}
		http.Redirect(w, r, "/products?error="+url.QueryEscape(msg), http.StatusSeeOther)
		return
	}

	if isJSON {
		writeJSON(w, http.StatusCreated, map[string]any{"id": id, "ref": it.Ref.String()})
		return
	}
	if wantsFragment(r) {
		h.renderProductsFragment(w, r, "")
		return
	}
	http.Redirect(w, r, "/products", http.StatusSeeOther)
}

// EditProductForm handles GET /products/{id}/edit and renders the edit form.
func (h *Handler) EditProductForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := h.catalog.GetProduct(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrProductNotFound) {
			http.Redirect(w, r, "/products?error="+url.QueryEscape("product not found"), http.StatusSeeOther)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ProductEditPage(p, r.URL.Query().Get("error")).Render(r.Context(), w); err != nil {
		log.Printf("render product edit: %v", err)
	}
}

// UpdateProduct handles POST /products/{id}, from both the HTML edit form
// and JSON API clients. Datastar forms swap the section fragment in place;
// plain forms redirect (PRG). JSON gets 200/400/404/409.
func (h *Handler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	editURL := "/products/" + id + "/edit"

	it, isJSON, err := decodeProduct(r)
	if err != nil {
		if isJSON {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if wantsFragment(r) {
			respondFragment(w, r, "#modal-body", ProductForm(productRow(it, id), err.Error()))
			return
		}
		http.Redirect(w, r, editURL+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}

	existing, err := h.catalog.GetProductItem(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrProductNotFound) {
			if isJSON {
				writeError(w, http.StatusNotFound, store.ErrProductNotFound)
				return
			}
			if wantsFragment(r) {
				h.renderProductsFragment(w, r, "product not found")
				return
			}
			http.Redirect(w, r, "/products?error="+url.QueryEscape("product not found"), http.StatusSeeOther)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	next := it
	if !isJSON {
		applyProductForm(existing, it)
		next = existing
	}
	if err := h.catalog.SaveProduct(r.Context(), id, next); err != nil {
		if isJSON {
			if productConflict(err) {
				writeError(w, http.StatusConflict, errors.New("product ref already exists"))
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		msg := conflictMessage(err, "product ref already exists")
		if wantsFragment(r) {
			respondFragment(w, r, "#modal-body", ProductForm(productRow(it, id), msg))
			return
		}
		http.Redirect(w, r, editURL+"?error="+url.QueryEscape(msg), http.StatusSeeOther)
		return
	}

	if isJSON {
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "ref": next.Ref.String()})
		return
	}
	if wantsFragment(r) {
		h.renderProductsFragment(w, r, "")
		return
	}
	http.Redirect(w, r, "/products", http.StatusSeeOther)
}

// DeleteProduct handles DELETE /products/{id} for API clients.
func (h *Handler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	n, err := h.catalog.DeleteProduct(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, store.ErrProductNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteProductForm handles POST /products/{id}/delete from the list page.
func (h *Handler) DeleteProductForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	n, err := h.catalog.DeleteProduct(r.Context(), id)
	if err != nil {
		if wantsFragment(r) {
			h.renderProductsFragment(w, r, err.Error())
			return
		}
		http.Redirect(w, r, "/products?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	if n == 0 {
		if wantsFragment(r) {
			h.renderProductsFragment(w, r, "product not found")
			return
		}
		http.Redirect(w, r, "/products?error="+url.QueryEscape("product not found"), http.StatusSeeOther)
		return
	}
	if wantsFragment(r) {
		h.renderProductsFragment(w, r, "")
		return
	}
	http.Redirect(w, r, "/products", http.StatusSeeOther)
}

var errPartyRequired = errors.New("name is required")

// decodeParty reads a GOBL org.Party from a JSON body (full shape, including
// nested collections) or from the flat catalog form (name, tax id, primary
// email/address, phone, website).
func decodeParty(r *http.Request) (*org.Party, bool, error) {
	isJSON := strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")
	if isJSON {
		p := new(org.Party)
		if err := json.NewDecoder(r.Body).Decode(p); err != nil {
			return nil, true, err
		}
		if strings.TrimSpace(p.Name) == "" {
			return nil, true, errPartyRequired
		}
		return p, true, nil
	}
	if err := r.ParseForm(); err != nil {
		return nil, false, err
	}
	p := partyFromForm(r)
	if strings.TrimSpace(p.Name) == "" {
		return nil, false, errPartyRequired
	}
	return p, false, nil
}

// partyFromForm maps the flat catalog form onto the corresponding GOBL party
// fields. Anything the form does not expose is left empty on create and
// preserved on update by applyPartyForm.
func partyFromForm(r *http.Request) *org.Party {
	get := func(k string) string { return strings.TrimSpace(r.FormValue(k)) }
	p := &org.Party{
		Label: get("label"),
		Name:  get("name"),
		Alias: get("alias"),
	}
	if c, code, scheme, typ := get("tax_country"), get("tax_code"), get("tax_scheme"), get("tax_type"); c != "" || code != "" || scheme != "" || typ != "" {
		p.TaxID = &tax.Identity{
			Country: l10n.TaxCountryCode(c),
			Code:    cbc.Code(code),
			Scheme:  cbc.Code(scheme),
			Type:    cbc.Key(typ),
		}
	}
	if v := get("email"); v != "" {
		p.Emails = []*org.Email{{Address: v}}
	}
	if v := get("phone"); v != "" {
		p.Telephones = []*org.Telephone{{Number: v}}
	}
	if v := get("website"); v != "" {
		p.Websites = []*org.Website{{URL: v}}
	}
	addr := &org.Address{
		Street:   get("address_line"),
		Locality: get("locality"),
		Code:     cbc.Code(get("post_code")),
		Country:  l10n.ISOCountryCode(get("country")),
	}
	if addr.Street != "" || addr.Locality != "" || addr.Code != "" || addr.Country != "" {
		p.Addresses = []*org.Address{addr}
	}
	for _, i := range formLineIndexes(r.Form, "identity_type", "identity_code", "identity_desc") {
		sfx := "_" + strconv.Itoa(i)
		typ := strings.TrimSpace(r.FormValue("identity_type" + sfx))
		code := strings.TrimSpace(r.FormValue("identity_code" + sfx))
		desc := strings.TrimSpace(r.FormValue("identity_desc" + sfx))
		if typ == "" && code == "" && desc == "" {
			continue
		}
		if code == "" {
			continue
		}
		p.Identities = append(p.Identities, &org.Identity{
			Key:         cbc.Key(typ),
			Code:        cbc.Code(code),
			Description: desc,
		})
	}
	for _, i := range formLineIndexes(r.Form, "logo_url", "logo_label", "logo_alt") {
		sfx := "_" + strconv.Itoa(i)
		logoURL := strings.TrimSpace(r.FormValue("logo_url" + sfx))
		if logoURL == "" {
			continue
		}
		p.Logos = append(p.Logos, &org.Image{
			URL:   logoURL,
			Label: strings.TrimSpace(r.FormValue("logo_label"+sfx)),
			Alt:   strings.TrimSpace(r.FormValue("logo_alt"+sfx)),
		})
	}
	return p
}

// applyPartyForm merges the flat form values into an existing party. The form
// only manages the scalar fields plus the first email/address/telephone/
// website, so entries created through the JSON API are preserved.
func applyPartyForm(dst, src *org.Party) {
	dst.Label, dst.Name, dst.Alias, dst.TaxID = src.Label, src.Name, src.Alias, src.TaxID
	// Extra codes and logos are fully managed by the form, so replace them
	// (including clearing when the user removes every row).
	dst.Identities, dst.Logos = src.Identities, src.Logos
	if len(src.Emails) > 0 {
		if len(dst.Emails) > 0 {
			dst.Emails[0] = src.Emails[0]
		} else {
			dst.Emails = src.Emails
		}
	}
	if len(src.Addresses) > 0 {
		if len(dst.Addresses) > 0 {
			dst.Addresses[0] = src.Addresses[0]
		} else {
			dst.Addresses = src.Addresses
		}
	}
	if len(src.Telephones) > 0 {
		if len(dst.Telephones) > 0 {
			dst.Telephones[0] = src.Telephones[0]
		} else {
			dst.Telephones = src.Telephones
		}
	}
	if len(src.Websites) > 0 {
		if len(dst.Websites) > 0 {
			dst.Websites[0] = src.Websites[0]
		} else {
			dst.Websites = src.Websites
		}
	}
}

// CreateCustomer handles POST /customers, from the HTML form and JSON API.
func (h *Handler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	// When invoked from a document form's "New customer" modal, the created
	// customer is fed straight back into the document form.
	forRoute := r.FormValue("for")
	renderCreateErr := func(p *org.Party, msg string) {
		if !wantsFragment(r) {
			return
		}
		if forRoute != "" {
			respondFragment(w, r, "#modal-body", CustomerCreateForm(CustomerFormData{
				For:        forRoute,
				Name:       strings.TrimSpace(r.FormValue("name")),
				TaxCountry: strings.TrimSpace(r.FormValue("tax_country")),
				TaxCode:    strings.TrimSpace(r.FormValue("tax_code")),
				Email:      strings.TrimSpace(r.FormValue("email")),
				Err:        msg,
			}))
			return
		}
		if p == nil {
			p = partyFromForm(r)
		}
		respondFragment(w, r, "#modal-body", CustomerForm(partyFormData(p, "", msg)))
	}

	f, isJSON, err := decodeParty(r)
	if err != nil {
		if isJSON {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if wantsFragment(r) {
			renderCreateErr(nil, err.Error())
			return
		}
		http.Redirect(w, r, "/customers?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	id := uuid.NewString()
	if err := h.catalog.SaveCustomer(r.Context(), id, f); err != nil {
		if isJSON {
			if productConflict(err) {
				writeError(w, http.StatusConflict, errors.New("customer tax code already exists"))
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		msg := conflictMessage(err, "customer tax code already exists")
		if wantsFragment(r) {
			renderCreateErr(f, msg)
			return
		}
		http.Redirect(w, r, "/customers?error="+url.QueryEscape(msg), http.StatusSeeOther)
		return
	}
	if isJSON {
		writeJSON(w, http.StatusCreated, map[string]any{"id": id, "name": f.Name})
		return
	}
	if forRoute != "" && wantsFragment(r) {
		h.renderDocFormModal(w, r, kindForRoute(forRoute), id, "")
		return
	}
	if wantsFragment(r) {
		h.renderCustomersFragment(w, r, "")
		return
	}
	http.Redirect(w, r, "/customers", http.StatusSeeOther)
}

// EditCustomerForm handles GET /customers/{id}/edit and renders the edit form.
func (h *Handler) EditCustomerForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, err := h.catalog.GetCustomerParty(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrCustomerNotFound) {
			http.Redirect(w, r, "/customers?error="+url.QueryEscape("customer not found"), http.StatusSeeOther)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := CustomerEditPage(partyFormData(c, id, r.URL.Query().Get("error"))).Render(r.Context(), w); err != nil {
		log.Printf("render customer edit: %v", err)
	}
}

// UpdateCustomer handles POST /customers/{id}, from the edit form and JSON API.
func (h *Handler) UpdateCustomer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	editURL := "/customers/" + id + "/edit"

	f, isJSON, err := decodeParty(r)
	if err != nil {
		if isJSON {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if wantsFragment(r) {
			respondFragment(w, r, "#modal-body", CustomerForm(partyFormData(f, id, err.Error())))
			return
		}
		http.Redirect(w, r, editURL+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	existing, err := h.catalog.GetCustomerParty(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrCustomerNotFound) {
			if isJSON {
				writeError(w, http.StatusNotFound, store.ErrCustomerNotFound)
				return
			}
			if wantsFragment(r) {
				h.renderCustomersFragment(w, r, "customer not found")
				return
			}
			http.Redirect(w, r, "/customers?error="+url.QueryEscape("customer not found"), http.StatusSeeOther)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	next := f
	if !isJSON {
		applyPartyForm(existing, f)
		next = existing
	}
	if err := h.catalog.SaveCustomer(r.Context(), id, next); err != nil {
		if isJSON {
			if productConflict(err) {
				writeError(w, http.StatusConflict, errors.New("customer tax code already exists"))
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		msg := conflictMessage(err, "customer tax code already exists")
		if wantsFragment(r) {
			respondFragment(w, r, "#modal-body", CustomerForm(partyFormData(next, id, msg)))
			return
		}
		http.Redirect(w, r, editURL+"?error="+url.QueryEscape(msg), http.StatusSeeOther)
		return
	}
	if isJSON {
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "name": next.Name})
		return
	}
	if wantsFragment(r) {
		h.renderCustomersFragment(w, r, "")
		return
	}
	http.Redirect(w, r, "/customers", http.StatusSeeOther)
}

// DeleteCustomer handles DELETE /customers/{id} for API clients.
func (h *Handler) DeleteCustomer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	n, err := h.catalog.DeleteCustomer(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, store.ErrCustomerNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteCustomerForm handles POST /customers/{id}/delete from the list page.
func (h *Handler) DeleteCustomerForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	n, err := h.catalog.DeleteCustomer(r.Context(), id)
	if err != nil {
		if wantsFragment(r) {
			h.renderCustomersFragment(w, r, err.Error())
			return
		}
		http.Redirect(w, r, "/customers?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	if n == 0 {
		if wantsFragment(r) {
			h.renderCustomersFragment(w, r, "customer not found")
			return
		}
		http.Redirect(w, r, "/customers?error="+url.QueryEscape("customer not found"), http.StatusSeeOther)
		return
	}
	if wantsFragment(r) {
		h.renderCustomersFragment(w, r, "")
		return
	}
	http.Redirect(w, r, "/customers", http.StatusSeeOther)
}

// CreateSupplier handles POST /suppliers, from the HTML form and JSON API.
func (h *Handler) CreateSupplier(w http.ResponseWriter, r *http.Request) {
	// When invoked from a document form's "New supplier" modal, the created
	// supplier is fed straight back into the document form.
	forRoute := r.FormValue("for")
	renderCreateErr := func(p *org.Party, msg string) {
		if !wantsFragment(r) {
			return
		}
		if forRoute != "" {
			respondFragment(w, r, "#modal-body", SupplierCreateForm(CustomerFormData{
				For:        forRoute,
				Name:       strings.TrimSpace(r.FormValue("name")),
				TaxCountry: strings.TrimSpace(r.FormValue("tax_country")),
				TaxCode:    strings.TrimSpace(r.FormValue("tax_code")),
				Email:      strings.TrimSpace(r.FormValue("email")),
				Err:        msg,
			}))
			return
		}
		if p == nil {
			p = partyFromForm(r)
		}
		respondFragment(w, r, "#modal-body", SupplierForm(partyFormData(p, "", msg)))
	}

	f, isJSON, err := decodeParty(r)
	if err != nil {
		if isJSON {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if wantsFragment(r) {
			renderCreateErr(nil, err.Error())
			return
		}
		http.Redirect(w, r, "/suppliers?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	id := uuid.NewString()
	if err := h.catalog.SaveSupplier(r.Context(), id, f); err != nil {
		if isJSON {
			if productConflict(err) {
				writeError(w, http.StatusConflict, errors.New("supplier tax code already exists"))
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		msg := conflictMessage(err, "supplier tax code already exists")
		if wantsFragment(r) {
			renderCreateErr(f, msg)
			return
		}
		http.Redirect(w, r, "/suppliers?error="+url.QueryEscape(msg), http.StatusSeeOther)
		return
	}
	if isJSON {
		writeJSON(w, http.StatusCreated, map[string]any{"id": id, "name": f.Name})
		return
	}
	if forRoute != "" && wantsFragment(r) {
		h.renderDocFormModal(w, r, kindForRoute(forRoute), "", id)
		return
	}
	if wantsFragment(r) {
		h.renderSuppliersFragment(w, r, "")
		return
	}
	http.Redirect(w, r, "/suppliers", http.StatusSeeOther)
}

// EditSupplierForm handles GET /suppliers/{id}/edit and renders the edit form.
func (h *Handler) EditSupplierForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sp, err := h.catalog.GetSupplierParty(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrSupplierNotFound) {
			http.Redirect(w, r, "/suppliers?error="+url.QueryEscape("supplier not found"), http.StatusSeeOther)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := SupplierEditPage(partyFormData(sp, id, r.URL.Query().Get("error"))).Render(r.Context(), w); err != nil {
		log.Printf("render supplier edit: %v", err)
	}
}

// UpdateSupplier handles POST /suppliers/{id}, from the edit form and JSON API.
func (h *Handler) UpdateSupplier(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	editURL := "/suppliers/" + id + "/edit"

	f, isJSON, err := decodeParty(r)
	if err != nil {
		if isJSON {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if wantsFragment(r) {
			respondFragment(w, r, "#modal-body", SupplierForm(partyFormData(f, id, err.Error())))
			return
		}
		http.Redirect(w, r, editURL+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	existing, err := h.catalog.GetSupplierParty(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrSupplierNotFound) {
			if isJSON {
				writeError(w, http.StatusNotFound, store.ErrSupplierNotFound)
				return
			}
			if wantsFragment(r) {
				h.renderSuppliersFragment(w, r, "supplier not found")
				return
			}
			http.Redirect(w, r, "/suppliers?error="+url.QueryEscape("supplier not found"), http.StatusSeeOther)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	next := f
	if !isJSON {
		applyPartyForm(existing, f)
		next = existing
	}
	if err := h.catalog.SaveSupplier(r.Context(), id, next); err != nil {
		if isJSON {
			if productConflict(err) {
				writeError(w, http.StatusConflict, errors.New("supplier tax code already exists"))
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		msg := conflictMessage(err, "supplier tax code already exists")
		if wantsFragment(r) {
			respondFragment(w, r, "#modal-body", SupplierForm(partyFormData(next, id, msg)))
			return
		}
		http.Redirect(w, r, editURL+"?error="+url.QueryEscape(msg), http.StatusSeeOther)
		return
	}
	if isJSON {
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "name": next.Name})
		return
	}
	if wantsFragment(r) {
		h.renderSuppliersFragment(w, r, "")
		return
	}
	http.Redirect(w, r, "/suppliers", http.StatusSeeOther)
}

// DeleteSupplier handles DELETE /suppliers/{id} for API clients.
func (h *Handler) DeleteSupplier(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	n, err := h.catalog.DeleteSupplier(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if n == 0 {
		writeError(w, http.StatusNotFound, store.ErrSupplierNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteSupplierForm handles POST /suppliers/{id}/delete from the list page.
func (h *Handler) DeleteSupplierForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	n, err := h.catalog.DeleteSupplier(r.Context(), id)
	if err != nil {
		if wantsFragment(r) {
			h.renderSuppliersFragment(w, r, err.Error())
			return
		}
		http.Redirect(w, r, "/suppliers?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	if n == 0 {
		if wantsFragment(r) {
			h.renderSuppliersFragment(w, r, "supplier not found")
			return
		}
		http.Redirect(w, r, "/suppliers?error="+url.QueryEscape("supplier not found"), http.StatusSeeOther)
		return
	}
	if wantsFragment(r) {
		h.renderSuppliersFragment(w, r, "")
		return
	}
	http.Redirect(w, r, "/suppliers", http.StatusSeeOther)
}

// Index handles GET / and renders the dashboard: document-kind overview
// cards plus the browsable invoice list with links to the printable HTML
// view and the raw envelope JSON.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	raw, err := h.entries.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	summaries := make([]*invoicer.Summary, 0, len(raw))
	for _, e := range raw {
		if s := invoicer.Summarize(e.Envelope); s != nil {
			summaries = append(summaries, s)
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := IndexPage(summaries, invoicer.SummarizeStats(summaries), docOverviews(raw)).Render(r.Context(), w); err != nil {
		// Render streams directly; on failure output may be partial.
		// Just log — the status line is already committed.
		log.Printf("render index: %v", err)
	}
}
