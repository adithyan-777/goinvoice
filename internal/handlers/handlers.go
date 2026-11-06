// Package handlers defines the Chi routes and their implementations.
// HTML pages are rendered with templ (see index.templ, run `templ generate`
// after editing). Invoice documents themselves are rendered by gobl.html.
//
//go:generate templ generate
package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

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
	r.Get("/customers", h.Customers)
	r.Post("/customers", h.CreateCustomer)
	r.Get("/customers/{id}/edit", h.EditCustomerForm)
	r.Post("/customers/{id}", h.UpdateCustomer)
	r.Post("/customers/{id}/delete", h.DeleteCustomerForm)
	r.Delete("/customers/{id}", h.DeleteCustomer)
	r.Get("/suppliers", h.Suppliers)
	r.Post("/suppliers", h.CreateSupplier)
	r.Get("/suppliers/{id}/edit", h.EditSupplierForm)
	r.Post("/suppliers/{id}", h.UpdateSupplier)
	r.Post("/suppliers/{id}/delete", h.DeleteSupplierForm)
	r.Delete("/suppliers/{id}", h.DeleteSupplier)
	r.Get("/products", h.Products)
	r.Post("/products", h.CreateProduct)
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
	r.Get("/partials/suppliers", h.SuppliersPartial)
	r.Get("/partials/products", h.ProductsPartial)
	r.Get("/partials/products/{id}/form", h.ProductModal)
	r.Get("/partials/customers/new", h.NewCustomerModal)
	r.Get("/partials/customers/{id}/form", h.CustomerModal)
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

// Suppliers handles GET /suppliers and renders the supplier catalog.
func (h *Handler) Suppliers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.catalog.ListSuppliers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := SuppliersPage(rows, r.URL.Query().Get("error")).Render(r.Context(), w); err != nil {
		log.Printf("render suppliers: %v", err)
	}
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

var errProductRequired = errors.New("ref, name and price are required")

// decodeProduct reads product fields from a JSON body or form values,
// trimming whitespace and defaulting empty currency to EUR.
func decodeProduct(r *http.Request) (sqlc.CreateProductParams, bool, error) {
	var p sqlc.CreateProductParams
	isJSON := strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")

	if isJSON {
		var body struct {
			Ref         string `json:"ref"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Price       string `json:"price"`
			Currency    string `json:"currency"`
			Unit        string `json:"unit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return p, true, err
		}
		p = sqlc.CreateProductParams{
			Ref:         strings.TrimSpace(body.Ref),
			Name:        strings.TrimSpace(body.Name),
			Description: nullIfEmpty(strings.TrimSpace(body.Description)),
			Price:       strings.TrimSpace(body.Price),
			Currency:    strings.TrimSpace(body.Currency),
			Unit:        nullIfEmpty(strings.TrimSpace(body.Unit)),
		}
	} else {
		if err := r.ParseForm(); err != nil {
			return p, false, err
		}
		p = sqlc.CreateProductParams{
			Ref:         strings.TrimSpace(r.FormValue("ref")),
			Name:        strings.TrimSpace(r.FormValue("name")),
			Description: nullIfEmpty(strings.TrimSpace(r.FormValue("description"))),
			Price:       strings.TrimSpace(r.FormValue("price")),
			Currency:    strings.TrimSpace(r.FormValue("currency")),
			Unit:        nullIfEmpty(strings.TrimSpace(r.FormValue("unit"))),
		}
	}

	if p.Ref == "" || p.Name == "" || p.Price == "" {
		return p, isJSON, errProductRequired
	}
	if p.Currency == "" {
		p.Currency = "EUR"
	}
	return p, isJSON, nil
}

// wantsFragment reports whether the request came from Datastar (SPA mode)
// and expects an HTML fragment instead of a redirect or full page.
func wantsFragment(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "text/event-stream")
}

// renderCustomersFragment writes the customers section for Datastar swaps.
func (h *Handler) renderCustomersFragment(w http.ResponseWriter, r *http.Request, errMsg string) {
	rows, err := h.catalog.ListCustomers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	respondFragment(w, r, "#main", CustomersSection(rows, errMsg))
}

// renderSuppliersFragment writes the suppliers section for Datastar swaps.
func (h *Handler) renderSuppliersFragment(w http.ResponseWriter, r *http.Request, errMsg string) {
	rows, err := h.catalog.ListSuppliers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	respondFragment(w, r, "#main", SuppliersSection(rows, errMsg))
}

// renderProductsFragment writes the products section for Datastar swaps.
func (h *Handler) renderProductsFragment(w http.ResponseWriter, r *http.Request, errMsg string) {
	rows, err := h.catalog.ListProducts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	respondFragment(w, r, "#main", ProductsSection(rows, errMsg))
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
	p, isJSON, err := decodeProduct(r)
	if err != nil {
		if isJSON {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if wantsFragment(r) {
			h.renderProductsFragment(w, r, err.Error())
			return
		}
		http.Redirect(w, r, "/products?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	p.ID = uuid.NewString()

	if err := h.catalog.CreateProduct(r.Context(), p); err != nil {
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
			h.renderProductsFragment(w, r, msg)
			return
		}
		http.Redirect(w, r, "/products?error="+url.QueryEscape(msg), http.StatusSeeOther)
		return
	}

	if isJSON {
		writeJSON(w, http.StatusCreated, map[string]any{"id": p.ID, "ref": p.Ref})
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

	p, isJSON, err := decodeProduct(r)
	if err != nil {
		if isJSON {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if wantsFragment(r) {
			h.renderProductsFragment(w, r, err.Error())
			return
		}
		http.Redirect(w, r, editURL+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}

	n, err := h.catalog.UpdateProduct(r.Context(), sqlc.UpdateProductParams{
		Ref:         p.Ref,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Currency:    p.Currency,
		Unit:        p.Unit,
		ID:          id,
	})
	if err != nil {
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
			h.renderProductsFragment(w, r, msg)
			return
		}
		http.Redirect(w, r, editURL+"?error="+url.QueryEscape(msg), http.StatusSeeOther)
		return
	}
	if n == 0 {
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

	if isJSON {
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "ref": p.Ref})
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

// partyFields holds the shared customer/supplier form fields.
type partyFields struct {
	Name        string
	TaxCountry  sql.NullString
	TaxCode     sql.NullString
	Email       sql.NullString
	AddressLine sql.NullString
	Locality    sql.NullString
	PostCode    sql.NullString
	Country     sql.NullString
}

var errPartyRequired = errors.New("name is required")

// decodeParty reads customer/supplier fields from a JSON body or form values.
func decodeParty(r *http.Request) (partyFields, bool, error) {
	var raw struct {
		Name        string `json:"name"`
		TaxCountry  string `json:"tax_country"`
		TaxCode     string `json:"tax_code"`
		Email       string `json:"email"`
		AddressLine string `json:"address_line"`
		Locality    string `json:"locality"`
		PostCode    string `json:"post_code"`
		Country     string `json:"country"`
	}
	isJSON := strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")
	if isJSON {
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			return partyFields{}, true, err
		}
	} else {
		if err := r.ParseForm(); err != nil {
			return partyFields{}, false, err
		}
		raw.Name = r.FormValue("name")
		raw.TaxCountry = r.FormValue("tax_country")
		raw.TaxCode = r.FormValue("tax_code")
		raw.Email = r.FormValue("email")
		raw.AddressLine = r.FormValue("address_line")
		raw.Locality = r.FormValue("locality")
		raw.PostCode = r.FormValue("post_code")
		raw.Country = r.FormValue("country")
	}
	t := func(s string) sql.NullString { return nullIfEmpty(strings.TrimSpace(s)) }
	if strings.TrimSpace(raw.Name) == "" {
		return partyFields{}, isJSON, errPartyRequired
	}
	return partyFields{
		Name:        strings.TrimSpace(raw.Name),
		TaxCountry:  t(raw.TaxCountry),
		TaxCode:     t(raw.TaxCode),
		Email:       t(raw.Email),
		AddressLine: t(raw.AddressLine),
		Locality:    t(raw.Locality),
		PostCode:    t(raw.PostCode),
		Country:     t(raw.Country),
	}, isJSON, nil
}

// CreateCustomer handles POST /customers, from the HTML form and JSON API.
func (h *Handler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	// When invoked from a document form's "New customer" modal, the created
	// customer is fed straight back into the document form.
	forRoute := r.FormValue("for")
	renderCreateErr := func(msg string) {
		if forRoute != "" && wantsFragment(r) {
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
		h.renderCustomersFragment(w, r, msg)
	}

	f, isJSON, err := decodeParty(r)
	if err != nil {
		if isJSON {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if wantsFragment(r) {
			renderCreateErr(err.Error())
			return
		}
		http.Redirect(w, r, "/customers?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	c := sqlc.CreateCustomerParams{
		ID: uuid.NewString(), Name: f.Name,
		TaxCountry: f.TaxCountry, TaxCode: f.TaxCode, Email: f.Email,
		AddressLine: f.AddressLine, Locality: f.Locality,
		PostCode: f.PostCode, Country: f.Country,
	}
	if err := h.catalog.CreateCustomer(r.Context(), c); err != nil {
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
			renderCreateErr(msg)
			return
		}
		http.Redirect(w, r, "/customers?error="+url.QueryEscape(msg), http.StatusSeeOther)
		return
	}
	if isJSON {
		writeJSON(w, http.StatusCreated, map[string]any{"id": c.ID, "name": c.Name})
		return
	}
	if forRoute != "" && wantsFragment(r) {
		h.renderDocFormModal(w, r, kindForRoute(forRoute), c.ID)
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
	c, err := h.catalog.GetCustomer(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrCustomerNotFound) {
			http.Redirect(w, r, "/customers?error="+url.QueryEscape("customer not found"), http.StatusSeeOther)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := CustomerEditPage(c, r.URL.Query().Get("error")).Render(r.Context(), w); err != nil {
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
			h.renderCustomersFragment(w, r, err.Error())
			return
		}
		http.Redirect(w, r, editURL+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	n, err := h.catalog.UpdateCustomer(r.Context(), sqlc.UpdateCustomerParams{
		Name:       f.Name,
		TaxCountry: f.TaxCountry, TaxCode: f.TaxCode, Email: f.Email,
		AddressLine: f.AddressLine, Locality: f.Locality,
		PostCode: f.PostCode, Country: f.Country,
		ID: id,
	})
	if err != nil {
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
			h.renderCustomersFragment(w, r, msg)
			return
		}
		http.Redirect(w, r, editURL+"?error="+url.QueryEscape(msg), http.StatusSeeOther)
		return
	}
	if n == 0 {
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
	if isJSON {
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "name": f.Name})
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
	f, isJSON, err := decodeParty(r)
	if err != nil {
		if isJSON {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if wantsFragment(r) {
			h.renderSuppliersFragment(w, r, err.Error())
			return
		}
		http.Redirect(w, r, "/suppliers?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	sp := sqlc.CreateSupplierParams{
		ID: uuid.NewString(), Name: f.Name,
		TaxCountry: f.TaxCountry, TaxCode: f.TaxCode, Email: f.Email,
		AddressLine: f.AddressLine, Locality: f.Locality,
		PostCode: f.PostCode, Country: f.Country,
	}
	if err := h.catalog.CreateSupplier(r.Context(), sp); err != nil {
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
			h.renderSuppliersFragment(w, r, msg)
			return
		}
		http.Redirect(w, r, "/suppliers?error="+url.QueryEscape(msg), http.StatusSeeOther)
		return
	}
	if isJSON {
		writeJSON(w, http.StatusCreated, map[string]any{"id": sp.ID, "name": sp.Name})
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
	sp, err := h.catalog.GetSupplier(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrSupplierNotFound) {
			http.Redirect(w, r, "/suppliers?error="+url.QueryEscape("supplier not found"), http.StatusSeeOther)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := SupplierEditPage(sp, r.URL.Query().Get("error")).Render(r.Context(), w); err != nil {
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
			h.renderSuppliersFragment(w, r, err.Error())
			return
		}
		http.Redirect(w, r, editURL+"?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	n, err := h.catalog.UpdateSupplier(r.Context(), sqlc.UpdateSupplierParams{
		Name:       f.Name,
		TaxCountry: f.TaxCountry, TaxCode: f.TaxCode, Email: f.Email,
		AddressLine: f.AddressLine, Locality: f.Locality,
		PostCode: f.PostCode, Country: f.Country,
		ID: id,
	})
	if err != nil {
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
			h.renderSuppliersFragment(w, r, msg)
			return
		}
		http.Redirect(w, r, editURL+"?error="+url.QueryEscape(msg), http.StatusSeeOther)
		return
	}
	if n == 0 {
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
	if isJSON {
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "name": f.Name})
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
