// Package handlers — bill document endpoints (order, delivery, payment,
// status). Mirrors the existing /invoices shape:
//
//	POST   /orders[?dry_run=true|html=true]      create
//	GET    /orders                               list summaries
//	GET    /orders/{id}                          full envelope
//	GET    /orders/{id}/html[?locale|watermark|sandbox]
//	DELETE /orders/{id}
//
// Same for /deliveries, /payments, /statuses. Amounts are JSON strings
// ("100.00"), percents need the "%" suffix — see README client rules.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/tax"

	"github.com/neo/goinvoice/internal/invoicer"
	"github.com/neo/goinvoice/internal/store"
)

// newDoc constructors per kind.
func newBillDoc(kind string) any {
	switch kind {
	case invoicer.KindOrder:
		return new(bill.Order)
	case invoicer.KindDelivery:
		return new(bill.Delivery)
	case invoicer.KindPayment:
		return new(bill.Payment)
	case invoicer.KindStatus:
		return new(bill.Status)
	default:
		return new(bill.Invoice)
	}
}

func buildBillDoc(kind string, doc any) (*gobl.Envelope, error) {
	// Delegates to the typed invoicer builders so Calculate+Validate run
	// the same pipeline as invoices.
	switch kind {
	case invoicer.KindOrder:
		return invoicer.BuildOrder(doc.(*bill.Order))
	case invoicer.KindDelivery:
		return invoicer.BuildDelivery(doc.(*bill.Delivery))
	case invoicer.KindPayment:
		return invoicer.BuildPayment(doc.(*bill.Payment))
	case invoicer.KindStatus:
		return invoicer.BuildStatus(doc.(*bill.Status))
	default:
		return invoicer.Build(doc.(*bill.Invoice))
	}
}

// decodeBillDoc reads a bill document from a JSON body or from structured
// UI form fields.
func (h *Handler) decodeBillDoc(r *http.Request, kind string) (any, bool, error) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		doc := newBillDoc(kind)
		if err := json.NewDecoder(r.Body).Decode(doc); err != nil {
			return nil, true, err
		}
		return doc, true, nil
	}
	doc, err := h.parseDocForm(r, kind)
	if err != nil {
		return nil, false, err
	}
	return doc, false, nil
}

// docFormFromRequest rebuilds form values from a submitted UI form (used to
// re-render the form with the user's input after a validation failure).
func docFormFromRequest(r *http.Request, kind string) DocForm {
	f := emptyDocForm(kind)
	get := func(k string) string { return strings.TrimSpace(r.FormValue(k)) }
	f.Type, f.Series, f.Code = get("type"), get("series"), get("code")
	if v := get("issue_date"); v != "" {
		f.IssueDate = v
	}
	if v := get("currency"); v != "" {
		f.Currency = v
	}
	f.SupID = get("sup_id")
	f.SupName, f.SupCountry, f.SupCode = get("sup_name"), get("sup_country"), get("sup_code")
	f.CusID = get("cus_id")
	f.CusName, f.CusCountry, f.CusCode = get("cus_name"), get("cus_country"), get("cus_code")
	f.MethodKey, f.MethodAmount = get("method_key"), get("method_amount")
	f.Lines = f.Lines[:0]
	switch kind {
	case invoicer.KindInvoice, invoicer.KindOrder, invoicer.KindDelivery:
		for _, i := range formLineIndexes(r.Form, "item", "product", "qty", "price", "rate") {
			sfx := "_" + strconv.Itoa(i)
			f.Lines = append(f.Lines, DocLineForm{
				ProductID: strings.TrimSpace(r.FormValue("product" + sfx)),
				Item:      strings.TrimSpace(r.FormValue("item" + sfx)),
				Qty:       nonEmpty(strings.TrimSpace(r.FormValue("qty"+sfx)), "1"),
				Price:     strings.TrimSpace(r.FormValue("price" + sfx)),
				Rate:      strings.TrimSpace(r.FormValue("rate" + sfx)),
			})
		}
	case invoicer.KindPayment:
		for _, i := range formLineIndexes(r.Form, "amount", "ref_series", "ref_code") {
			sfx := "_" + strconv.Itoa(i)
			f.Lines = append(f.Lines, DocLineForm{
				Qty:       "1",
				Amount:    strings.TrimSpace(r.FormValue("amount" + sfx)),
				RefSeries: strings.TrimSpace(r.FormValue("ref_series" + sfx)),
				RefCode:   strings.TrimSpace(r.FormValue("ref_code" + sfx)),
			})
		}
	case invoicer.KindStatus:
		for _, i := range formLineIndexes(r.Form, "linekey", "ref_series", "ref_code") {
			sfx := "_" + strconv.Itoa(i)
			f.Lines = append(f.Lines, DocLineForm{
				Qty:       "1",
				LineKey:   strings.TrimSpace(r.FormValue("linekey" + sfx)),
				RefSeries: strings.TrimSpace(r.FormValue("ref_series" + sfx)),
				RefCode:   strings.TrimSpace(r.FormValue("ref_code" + sfx)),
			})
		}
	}
	return f
}

// formLineIndexes collects the sorted numeric suffixes of submitted line
// fields (item_0, price_2, …) for the given prefixes. Rows added or removed
// client-side may leave gaps, so indexes are discovered, not assumed
// contiguous.
func formLineIndexes(form url.Values, prefixes ...string) []int {
	var out []int
	seen := map[int]bool{}
	for k := range form {
		for _, p := range prefixes {
			rest, ok := strings.CutPrefix(k, p+"_")
			if !ok {
				continue
			}
			n, err := strconv.Atoi(rest)
			if err != nil || n < 0 || n > 500 || seen[n] {
				continue
			}
			seen[n] = true
			out = append(out, n)
		}
	}
	sort.Ints(out)
	return out
}

func nonEmpty(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// parseDocForm builds a bill document from structured UI form fields via a
// JSON round-trip, reusing the same parsing as the API.
func (h *Handler) parseDocForm(r *http.Request, kind string) (any, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	get := func(k string) string { return strings.TrimSpace(r.FormValue(k)) }
	m := map[string]any{}
	set := func(k, v string) {
		if v != "" {
			m[k] = v
		}
	}
	set("type", get("type"))
	set("series", get("series"))
	set("code", get("code"))
	set("issue_date", get("issue_date"))
	set("currency", get("currency"))

	// Parties picked from the searchable catalog dropdown are loaded whole
	// (all GOBL org.Party fields), not just name + tax identity.
	if p, ok := h.supplierParty(r.Context(), get("sup_id"), get("sup_name")); ok {
		if pm := partyMap(p); pm != nil {
			m["supplier"] = pm
		}
	} else if name := get("sup_name"); name != "" {
		sup := map[string]any{"name": name}
		if c, code := get("sup_country"), get("sup_code"); c != "" || code != "" {
			sup["tax_id"] = map[string]any{"country": c, "code": code}
		}
		m["supplier"] = sup
	}
	if p, ok := h.customerParty(r.Context(), get("cus_id"), get("cus_name")); ok {
		if pm := partyMap(p); pm != nil {
			m["customer"] = pm
		}
	} else if name := get("cus_name"); name != "" {
		cus := map[string]any{"name": name}
		if c, code := get("cus_country"), get("cus_code"); c != "" || code != "" {
			cus["tax_id"] = map[string]any{"country": c, "code": code}
		}
		m["customer"] = cus
	}

	switch kind {
	case invoicer.KindInvoice, invoicer.KindOrder, invoicer.KindDelivery:
		var lines []any
		for pos, i := range formLineIndexes(r.Form, "item", "product", "qty", "price", "rate") {
			p := func(n string) string { return get(n + "_" + strconv.Itoa(i)) }
			prodID := p("product")
			item, qty, price, rate := p("item"), p("qty"), p("price"), p("rate")
			// Line item picked from the searchable product picker: the unit
			// price and full item details default from the catalog but stay
			// editable.
			prod, hasProd := h.lookupProductByID(r.Context(), prodID)
			if hasProd {
				if item == "" {
					item = prod.Name
				}
				if price == "" {
					price = prod.Price
				}
			} else if item != "" && price == "" {
				if p, ok := h.lookupProduct(r.Context(), strings.TrimSpace(item)); ok {
					price = p.Price
				}
			}
			if item == "" && price == "" {
				continue
			}
			if qty == "" {
				qty = "1"
			}
			if item != "" && price == "" {
				return nil, errors.New("line " + strconv.Itoa(pos+1) + ": price is required")
			}
			itemMap := map[string]any{"name": item, "price": price}
			// Product picked from the catalog: embed the whole org.Item
			// (unit, description, ref, identities, attributes, images, …)
			// while keeping the user's edited name/price.
			if hasProd && prod.Item != nil {
				if im := itemMapOf(prod.Item); im != nil {
					if item != "" {
						im["name"] = item
					}
					if price != "" {
						im["price"] = price
					}
					itemMap = im
				}
			}
			line := map[string]any{
				"quantity": qty,
				"item":     itemMap,
			}
			if rate != "" {
				line["taxes"] = []any{map[string]any{"cat": "VAT", "rate": rate}}
			}
			lines = append(lines, line)
		}
		if lines != nil {
			m["lines"] = lines
		}
	case invoicer.KindPayment:
		var lines []any
		for i := 0; i < 3; i++ {
			p := func(n string) string { return get(n + "_" + string(rune('0'+i))) }
			amount := p("amount")
			if amount == "" {
				continue
			}
			line := map[string]any{"amount": amount}
			if rs, rc := p("ref_series"), p("ref_code"); rs != "" || rc != "" {
				line["document"] = map[string]any{"series": rs, "code": rc}
			}
			lines = append(lines, line)
		}
		if lines != nil {
			m["lines"] = lines
		}
		methodKey, methodAmount := get("method_key"), get("method_amount")
		if methodKey == "" && methodAmount == "" {
			break
		}
		if methodAmount == "" {
			return nil, errors.New("payment method amount is required")
		}
		m["methods"] = []any{map[string]any{"key": methodKey, "amount": methodAmount}}
	case invoicer.KindStatus:
		var lines []any
		for i := 0; i < 3; i++ {
			p := func(n string) string { return get(n + "_" + string(rune('0'+i))) }
			key := p("linekey")
			rs, rc := p("ref_series"), p("ref_code")
			if key == "" && rs == "" && rc == "" {
				continue
			}
			line := map[string]any{}
			if key != "" {
				line["key"] = key
			}
			if rs != "" || rc != "" {
				line["doc"] = map[string]any{"series": rs, "code": rc}
			}
			lines = append(lines, line)
		}
		if lines != nil {
			m["lines"] = lines
		}
	}

	raw, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	doc := newBillDoc(kind)
	if err := json.Unmarshal(raw, doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// createBillDoc handles POST /{kind}s from JSON API clients and the UI "New"
// form. JSON clients get 201/400/409-style JSON; UI posts re-render the
// section fragment (Datastar) or redirect (plain).
func (h *Handler) createBillDoc(kind string) http.HandlerFunc {
	route, title, _, _ := docMeta(kind)
	return func(w http.ResponseWriter, r *http.Request) {
		doc, isJSON, err := h.decodeBillDoc(r, kind)
		if err != nil {
			h.createDocError(w, r, kind, route, title, err)
			return
		}
		env, err := buildBillDoc(kind, doc)
		if err != nil {
			h.createDocError(w, r, kind, route, title, err)
			return
		}
		id := invoicer.ID(env)

		if r.URL.Query().Get("dry_run") == "true" {
			writeJSON(w, http.StatusOK, map[string]any{"id": id, "envelope": env})
			return
		}
		if err := h.entries.Put(r.Context(), id, env); err != nil {
			if isJSON {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			h.createDocError(w, r, kind, route, title, err)
			return
		}
		if !isJSON {
			if wantsFragment(r) {
				h.respondDocList(w, r, kind, true)
				return
			}
			http.Redirect(w, r, "/"+route, http.StatusSeeOther)
			return
		}
		if r.URL.Query().Get("html") == "true" {
			w.Header().Set("Location", "/"+route+"/"+id+"/html")
			writeJSON(w, http.StatusCreated, map[string]any{
				"id":       id,
				"summary":  invoicer.SummarizeAny(env),
				"html_url": "/" + route + "/" + id + "/html",
			})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"id": id, "envelope": env})
	}
}

// createDocError reports a create failure: JSON for API clients, the New
// form re-rendered with the user's input and the error for UI posts.
func (h *Handler) createDocError(w http.ResponseWriter, r *http.Request, kind, route, title string, err error) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		if isDocValidationError(err) {
			writeError(w, http.StatusUnprocessableEntity, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	form := docFormFromRequest(r, kind)
	form.Err = err.Error()
	form.Action = "/" + route
	form.Submit = "Create " + form.One
	h.attachCustomers(r.Context(), &form)
	h.attachSuppliers(r.Context(), &form)
	h.attachProducts(r.Context(), &form)
	if wantsFragment(r) {
		if r.FormValue("modal") == "1" {
			form.Modal = true
			respondFragment(w, r, "#modal-body", DocFormBody(form))
			return
		}
		respondFragment(w, r, "#main", NewDocSection(route, docOne(kind), title, form))
		return
	}
	http.Redirect(w, r, "/"+route+"/new?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
}

// isDocValidationError reports whether err came from the GOBL calculate or
// validate step rather than from malformed input.
func isDocValidationError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "validation:") || strings.Contains(msg, "calculate:") || strings.Contains(msg, "insert document:")
}

// listBillDocs handles GET /{kind}s, filtered to that document kind.
//
// Content negotiation: browsers (Accept: text/html) get the full HTML list
// page; API clients (curl, Accept: application/json or */*) get the JSON
// summary list. This shares one clean URL for page and API because Chi
// mounts Route("/orders") over an exact Get("/orders") — a standalone page
// route would be silently dropped.
func (h *Handler) listBillDocs(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, err := h.entries.List(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		out := make([]*invoicer.Summary, 0, len(raw))
		for _, e := range raw {
			s := invoicer.SummarizeAny(e.Envelope)
			if s == nil || s.Document != kind {
				continue
			}
			out = append(out, s)
		}
		if isBrowserPage(r) {
			route, title, subtitle, sample := docMeta(kind)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := DocumentsPage(route, title, docOne(kind), subtitle, sample, out, invoicer.SummarizeStats(out)).Render(r.Context(), w); err != nil {
				log.Printf("render %s: %v", kind, err)
			}
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": out})
	}
}

// isBrowserPage reports whether the request wants a full HTML page: the
// Accept header asks for text/html and it is not a Datastar fragment fetch
// (those carry Accept: text/event-stream and are served under /partials/*).
func isBrowserPage(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "text/html") && !wantsFragment(r)
}

// getBillDoc handles GET /{kind}s/{id}, 404 when the id holds another kind.
func (h *Handler) getBillDoc(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		if invoicer.DocumentKind(env) != kind {
			writeError(w, http.StatusNotFound, store.ErrNotFound)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "envelope": env})
	}
}

// htmlBillDoc handles GET /{kind}s/{id}/html.
func (h *Handler) htmlBillDoc(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		if invoicer.DocumentKind(env) != kind {
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
}

// deleteBillDoc handles DELETE /{kind}s/{id}, 404 when kind mismatches.
func (h *Handler) deleteBillDoc(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		if invoicer.DocumentKind(env) != kind {
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
}

// itoa renders a line index for form field names (item_0, amount_1, …).
func itoa(i int) string { return strconv.Itoa(i) }

// DocOption is one choice in a form <select>.
type DocOption struct {
	Value string
	Label string
}

// docTypeOptions returns the document types valid for a kind.
func docTypeOptions(kind string) []DocOption {
	switch kind {
	case invoicer.KindOrder:
		return []DocOption{{"purchase", "Purchase order"}, {"sale", "Sales order"}, {"quote", "Quote"}}
	case invoicer.KindDelivery:
		return []DocOption{{"advice", "Delivery advice"}, {"note", "Delivery note"}, {"waybill", "Waybill"}, {"receipt", "Delivery receipt"}, {"other", "Other"}}
	case invoicer.KindPayment:
		return []DocOption{{"receipt", "Payment receipt"}, {"request", "Payment request"}, {"advice", "Remittance advice"}}
	case invoicer.KindStatus:
		return []DocOption{{"response", "Response"}, {"update", "Update"}, {"system", "System"}}
	default: // invoice
		return []DocOption{
			{"standard", "Standard"},
			{"proforma", "Proforma"},
			{"corrective", "Corrective"},
			{"credit-note", "Credit note"},
			{"debit-note", "Debit note"},
			{"other", "Other"},
		}
	}
}

// vatRateOptions lists the VAT rates offered on item lines.
func vatRateOptions() []DocOption {
	return []DocOption{
		{"standard", "Standard"},
		{"reduced", "Reduced"},
		{"super-reduced", "Super-reduced"},
		{"exempt", "Exempt"},
	}
}

// paymentMethodOptions lists common payment means.
func paymentMethodOptions() []DocOption {
	return []DocOption{
		{"credit-transfer", "Credit transfer"},
		{"direct-debit", "Direct debit"},
		{"card", "Card"},
		{"cash", "Cash"},
		{"cheque", "Cheque"},
		{"online", "Online"},
		{"other", "Other"},
	}
}

// statusKeyOptions lists the statuses a status line can report.
func statusKeyOptions() []DocOption {
	return []DocOption{
		{"issued", "Issued"},
		{"acknowledged", "Acknowledged"},
		{"processing", "Processing"},
		{"querying", "Under query"},
		{"rejected", "Rejected"},
		{"accepted", "Accepted"},
		{"paid", "Paid"},
		{"error", "Error"},
		{"other", "Other"},
	}
}

// DocLineForm is one editable line row in the UI document form.
type DocLineForm struct {
	ProductID string // order/delivery/invoice catalog product, "" when free-typed
	Item      string // order/delivery item name
	Qty       string // order/delivery quantity
	Price     string // order/delivery unit price
	Rate      string // VAT rate: standard|reduced|super-reduced|exempt|""
	Amount    string // payment line/method amount
	LineKey   string // status line key
	RefSeries string // payment/status referenced document series
	RefCode   string // payment/status referenced document code
}

// DocForm carries the UI document form values: shared header/party fields
// plus a dynamic number of line rows. Payment methods use
// MethodKey/MethodAmount.
type DocForm struct {
	Kind, Route, One                    string
	Action, Submit, Err                 string
	Modal                               bool
	Type, Series, Code                  string
	IssueDate, Currency                 string
	SupID, SupName, SupCountry, SupCode string
	CusID, CusName, CusCountry, CusCode string
	MethodKey, MethodAmount             string
	Suppliers                           []DocSupplier
	Customers                           []DocCustomer
	Products                            []DocProduct
	Lines                               []DocLineForm
}

// DocCustomer is one searchable customer choice in the document form,
// sourced from the customers catalog so tax IDs fill in automatically.
type DocCustomer struct {
	ID      string
	Name    string
	Country string
	Code    string
}

// DocSupplier is one searchable supplier choice in the document form,
// sourced from the suppliers catalog so tax IDs fill in automatically.
type DocSupplier struct {
	ID      string
	Name    string
	Country string
	Code    string
}

// DocProduct is one searchable product choice for document line items,
// sourced from the products catalog so unit prices and item details fill in
// automatically.
type DocProduct struct {
	ID       string
	Ref      string
	Name     string
	Price    string
	Currency string
	Unit     string
	Item     *org.Item
}

// attachProducts loads the product catalog for the searchable line-item
// picker.
func (h *Handler) attachProducts(ctx context.Context, f *DocForm) {
	rows, err := h.catalog.ListProducts(ctx)
	if err != nil {
		log.Printf("form products: %v", err)
		return
	}
	for _, p := range rows {
		f.Products = append(f.Products, DocProduct{
			ID:       p.ID,
			Ref:      p.Ref,
			Name:     p.Name,
			Price:    p.Price,
			Currency: p.Currency,
			Unit:     nsStr(p.Unit),
		})
		for i := range f.Lines {
			if f.Lines[i].ProductID == "" && f.Lines[i].Item != "" &&
				(strings.EqualFold(strings.TrimSpace(p.Name), strings.TrimSpace(f.Lines[i].Item)) ||
					strings.EqualFold(strings.TrimSpace(p.Ref), strings.TrimSpace(f.Lines[i].Item))) {
				f.Lines[i].ProductID = p.ID
			}
		}
	}
}

// lookupProductByID finds a catalog product by primary key and loads its full
// org.Item representation.
func (h *Handler) lookupProductByID(ctx context.Context, id string) (DocProduct, bool) {
	if id == "" {
		return DocProduct{}, false
	}
	it, err := h.catalog.GetProductItem(ctx, id)
	if err != nil {
		return DocProduct{}, false
	}
	return DocProduct{
		ID:       id,
		Ref:      it.Ref.String(),
		Name:     it.Name,
		Price:    amountOrEmpty(it.Price),
		Currency: it.Currency.String(),
		Unit:     it.Unit.String(),
		Item:     it,
	}, true
}

// lookupProduct finds a catalog product by exact name or ref
// (case-insensitive).
func (h *Handler) lookupProduct(ctx context.Context, name string) (DocProduct, bool) {
	rows, err := h.catalog.ListProducts(ctx)
	if err != nil {
		return DocProduct{}, false
	}
	for _, p := range rows {
		if strings.EqualFold(strings.TrimSpace(p.Name), name) ||
			strings.EqualFold(strings.TrimSpace(p.Ref), name) {
			return h.lookupProductByID(ctx, p.ID)
		}
	}
	return DocProduct{}, false
}

// attachSuppliers loads the supplier catalog for the searchable picker.
func (h *Handler) attachSuppliers(ctx context.Context, f *DocForm) {
	rows, err := h.catalog.ListSuppliers(ctx)
	if err != nil {
		log.Printf("form suppliers: %v", err)
		return
	}
	for _, s := range rows {
		f.Suppliers = append(f.Suppliers, DocSupplier{
			ID:      s.ID,
			Name:    s.Name,
			Country: nsStr(s.TaxCountry),
			Code:    nsStr(s.TaxCode),
		})
		if f.SupID == "" && f.SupName != "" && strings.EqualFold(strings.TrimSpace(s.Name), strings.TrimSpace(f.SupName)) {
			f.SupID = s.ID
		}
	}
}

// lookupSupplierByID finds a catalog supplier by primary key.
func (h *Handler) lookupSupplierByID(ctx context.Context, id string) (DocSupplier, bool) {
	if id == "" {
		return DocSupplier{}, false
	}
	s, err := h.catalog.GetSupplier(ctx, id)
	if err != nil {
		return DocSupplier{}, false
	}
	return DocSupplier{ID: s.ID, Name: s.Name, Country: nsStr(s.TaxCountry), Code: nsStr(s.TaxCode)}, true
}

// lookupSupplier finds a catalog supplier by exact (case-insensitive) name.
func (h *Handler) lookupSupplier(ctx context.Context, name string) (DocSupplier, bool) {
	rows, err := h.catalog.ListSuppliers(ctx)
	if err != nil {
		return DocSupplier{}, false
	}
	for _, s := range rows {
		if strings.EqualFold(strings.TrimSpace(s.Name), name) {
			return DocSupplier{
				ID:      s.ID,
				Name:    s.Name,
				Country: nsStr(s.TaxCountry),
				Code:    nsStr(s.TaxCode),
			}, true
		}
	}
	return DocSupplier{}, false
}

// attachCustomers loads the customer catalog for the searchable picker.
func (h *Handler) attachCustomers(ctx context.Context, f *DocForm) {
	rows, err := h.catalog.ListCustomers(ctx)
	if err != nil {
		log.Printf("form customers: %v", err)
		return
	}
	for _, c := range rows {
		f.Customers = append(f.Customers, DocCustomer{
			ID:      c.ID,
			Name:    c.Name,
			Country: nsStr(c.TaxCountry),
			Code:    nsStr(c.TaxCode),
		})
		if f.CusID == "" && f.CusName != "" && strings.EqualFold(strings.TrimSpace(c.Name), strings.TrimSpace(f.CusName)) {
			f.CusID = c.ID
		}
	}
}

// lookupCustomerByID finds a catalog customer by primary key.
func (h *Handler) lookupCustomerByID(ctx context.Context, id string) (DocCustomer, bool) {
	if id == "" {
		return DocCustomer{}, false
	}
	c, err := h.catalog.GetCustomer(ctx, id)
	if err != nil {
		return DocCustomer{}, false
	}
	return DocCustomer{ID: c.ID, Name: c.Name, Country: nsStr(c.TaxCountry), Code: nsStr(c.TaxCode)}, true
}

// kindForRoute maps a plural route ("orders") to its bill document kind.
func kindForRoute(route string) string {
	for _, d := range docKinds {
		if d.route == route {
			return d.kind
		}
	}
	return invoicer.KindInvoice
}

// renderDocFormModal re-renders a New document form in the modal, optionally
// preselecting a customer or supplier (used after creating one from the
// form itself).
func (h *Handler) renderDocFormModal(w http.ResponseWriter, r *http.Request, kind, preselectCustomerID, preselectSupplierID string) {
	form := emptyDocForm(kind)
	form.Modal = true
	form.Action = "/" + form.Route
	form.Submit = "Create " + form.One
	h.attachCustomers(r.Context(), &form)
	h.attachSuppliers(r.Context(), &form)
	h.attachProducts(r.Context(), &form)
	if c, ok := h.lookupCustomerByID(r.Context(), preselectCustomerID); ok {
		form.CusID, form.CusName = c.ID, c.Name
	}
	if s, ok := h.lookupSupplierByID(r.Context(), preselectSupplierID); ok {
		form.SupID, form.SupName = s.ID, s.Name
	}
	sseBegin(w)
	sseSignals(w, `{"modalTitle": "New `+form.One+`"}`)
	sseElements(w, r, "#modal-body", "inner", DocFormBody(form))
	sseFlush(w)
}

// lookupCustomer finds a catalog customer by exact (case-insensitive) name.
func (h *Handler) lookupCustomer(ctx context.Context, name string) (DocCustomer, bool) {
	rows, err := h.catalog.ListCustomers(ctx)
	if err != nil {
		return DocCustomer{}, false
	}
	for _, c := range rows {
		if strings.EqualFold(strings.TrimSpace(c.Name), name) {
			return DocCustomer{
				ID:      c.ID,
				Name:    c.Name,
				Country: nsStr(c.TaxCountry),
				Code:    nsStr(c.TaxCode),
			}, true
		}
	}
	return DocCustomer{}, false
}

// supplierParty resolves a catalog supplier by ID (preferred) or exact name.
func (h *Handler) supplierParty(ctx context.Context, id, name string) (*org.Party, bool) {
	if id != "" {
		if p, err := h.catalog.GetSupplierParty(ctx, id); err == nil {
			return p, true
		}
	}
	if name != "" {
		if s, ok := h.lookupSupplier(ctx, name); ok {
			if p, err := h.catalog.GetSupplierParty(ctx, s.ID); err == nil {
				return p, true
			}
		}
	}
	return nil, false
}

// customerParty resolves a catalog customer by ID (preferred) or exact name.
func (h *Handler) customerParty(ctx context.Context, id, name string) (*org.Party, bool) {
	if id != "" {
		if p, err := h.catalog.GetCustomerParty(ctx, id); err == nil {
			return p, true
		}
	}
	if name != "" {
		if c, ok := h.lookupCustomer(ctx, name); ok {
			if p, err := h.catalog.GetCustomerParty(ctx, c.ID); err == nil {
				return p, true
			}
		}
	}
	return nil, false
}

// partyMap converts a party to the generic map used by parseDocForm before it
// round-trips into the typed bill document.
func partyMap(p *org.Party) map[string]any {
	return jsonMap(p)
}

// itemMapOf converts a catalog item to the generic line-item map.
func itemMapOf(it *org.Item) map[string]any {
	return jsonMap(it)
}

func jsonMap(v any) map[string]any {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

// emptyDocForm returns blank form values with per-kind defaults.
func emptyDocForm(kind string) DocForm {
	route, _, _, _ := docMeta(kind)
	defType := map[string]string{
		invoicer.KindInvoice:  "standard",
		invoicer.KindOrder:    "purchase",
		invoicer.KindDelivery: "note",
		invoicer.KindPayment:  "receipt",
		invoicer.KindStatus:   "response",
	}[kind]
	f := DocForm{
		Kind: kind, Route: route, One: docOne(kind),
		Type: defType, Currency: "EUR",
		IssueDate: time.Now().Format("2006-01-02"),
		MethodKey: "credit-transfer",
		Lines:     []DocLineForm{{Qty: "1", Rate: "standard"}},
	}
	return f
}

// ensureDocLine grows the form lines so index i is addressable.
func ensureDocLine(f *DocForm, i int) {
	for len(f.Lines) <= i {
		f.Lines = append(f.Lines, DocLineForm{Qty: "1", Rate: "standard"})
	}
}

// dateStr renders a GOBL date for <input type="date"> (YYYY-MM-DD).
func dateStr(d cal.Date) string {
	if d.IsZero() {
		return ""
	}
	return d.String()
}

// fillParties copies supplier/customer names and tax IDs into the form.
func fillParties(f *DocForm, supplier, customer *org.Party) {
	if supplier != nil {
		f.SupName = supplier.Name
		if supplier.TaxID != nil {
			f.SupCountry, f.SupCode = supplier.TaxID.Country.String(), supplier.TaxID.Code.String()
		}
	}
	if customer != nil {
		f.CusName = customer.Name
		if customer.TaxID != nil {
			f.CusCountry, f.CusCode = customer.TaxID.Country.String(), customer.TaxID.Code.String()
		}
	}
}

// lineTaxRate returns the first line-tax rate (e.g. "standard"), or "".
func lineTaxRate(taxes tax.Set) string {
	for _, t := range taxes {
		if t == nil {
			continue
		}
		if r := t.Rate.String(); r != "" {
			return r
		}
	}
	return ""
}

// docFormFromEnvelope prefills the form from a stored envelope (all lines;
// saving replaces the document lines with the submitted ones).
func docFormFromEnvelope(kind string, env *gobl.Envelope) DocForm {
	f := emptyDocForm(kind)
	switch doc := env.Extract().(type) {
	case *bill.Invoice:
		f.Type, f.Series, f.Code = doc.Type.String(), doc.Series.String(), doc.Code.String()
		f.IssueDate, f.Currency = dateStr(doc.IssueDate), doc.Currency.String()
		fillParties(&f, doc.Supplier, doc.Customer)
		for i, l := range doc.Lines {
			if l == nil {
				continue
			}
			if i >= 200 {
				break
			}
			ensureDocLine(&f, i)
			f.Lines[i].Qty = l.Quantity.String()
			if l.Item != nil {
				f.Lines[i].Item = l.Item.Name
				if l.Item.Price != nil {
					f.Lines[i].Price = l.Item.Price.String()
				}
			}
			f.Lines[i].Rate = lineTaxRate(l.Taxes)
		}
	case *bill.Order:
		f.Type, f.Series, f.Code = doc.Type.String(), doc.Series.String(), doc.Code.String()
		f.IssueDate, f.Currency = dateStr(doc.IssueDate), doc.Currency.String()
		fillParties(&f, doc.Supplier, doc.Customer)
		for i, l := range doc.Lines {
			if l == nil {
				continue
			}
			if i >= 200 {
				break
			}
			ensureDocLine(&f, i)
			f.Lines[i].Qty = l.Quantity.String()
			if l.Item != nil {
				f.Lines[i].Item = l.Item.Name
				if l.Item.Price != nil {
					f.Lines[i].Price = l.Item.Price.String()
				}
			}
			f.Lines[i].Rate = lineTaxRate(l.Taxes)
		}
	case *bill.Delivery:
		f.Type, f.Series, f.Code = doc.Type.String(), doc.Series.String(), doc.Code.String()
		f.IssueDate, f.Currency = dateStr(doc.IssueDate), doc.Currency.String()
		fillParties(&f, doc.Supplier, doc.Customer)
		for i, l := range doc.Lines {
			if l == nil {
				continue
			}
			if i >= 200 {
				break
			}
			ensureDocLine(&f, i)
			f.Lines[i].Qty = l.Quantity.String()
			if l.Item != nil {
				f.Lines[i].Item = l.Item.Name
				if l.Item.Price != nil {
					f.Lines[i].Price = l.Item.Price.String()
				}
			}
			f.Lines[i].Rate = lineTaxRate(l.Taxes)
		}
	case *bill.Payment:
		f.Type, f.Series, f.Code = doc.Type.String(), doc.Series.String(), doc.Code.String()
		f.IssueDate, f.Currency = dateStr(doc.IssueDate), doc.Currency.String()
		fillParties(&f, doc.Supplier, doc.Customer)
		for i, l := range doc.Lines {
			if l == nil {
				continue
			}
			if i >= 200 {
				break
			}
			ensureDocLine(&f, i)
			f.Lines[i].Amount = l.Amount.String()
			if l.Document != nil {
				f.Lines[i].RefSeries, f.Lines[i].RefCode = l.Document.Series.String(), l.Document.Code.String()
			}
		}
		if len(doc.Methods) > 0 && doc.Methods[0] != nil {
			f.MethodKey, f.MethodAmount = doc.Methods[0].Key.String(), doc.Methods[0].Amount.String()
		}
	case *bill.Status:
		f.Type, f.Code = doc.Type.String(), doc.Code.String()
		f.IssueDate = dateStr(doc.IssueDate)
		fillParties(&f, doc.Supplier, doc.Customer)
		for i, l := range doc.Lines {
			if l == nil {
				continue
			}
			if i >= 200 {
				break
			}
			ensureDocLine(&f, i)
			f.Lines[i].LineKey = l.Key.String()
			if l.Doc != nil {
				f.Lines[i].RefSeries, f.Lines[i].RefCode = l.Doc.Series.String(), l.Doc.Code.String()
			}
		}
	}
	return f
}

type DocOverview struct {
	Route string
	Title string
	Count int
	Icon  string
}

// docKinds lists every bill document kind with its UI metadata: singular
// GOBL kind, plural route, title, subtitle, sample file and sidebar icon.
var docKinds = []struct {
	kind     string
	route    string
	title    string
	one      string
	subtitle string
	sample   string
	icon     string
}{
	{invoicer.KindInvoice, "invoices", "Invoices", "Invoice", "Billing documents with calculated totals", "sample-invoice.json", "bx-receipt"},
	{invoicer.KindOrder, "orders", "Orders", "Order", "Purchase and sales orders, quotes", "sample-order.json", "bx-cart"},
	{invoicer.KindDelivery, "deliveries", "Deliveries", "Delivery", "Despatch advice, notes, waybills and receipts", "sample-delivery.json", "bx-truck"},
	{invoicer.KindPayment, "payments", "Payments", "Payment", "Payment requests, receipts and advice", "sample-payment.json", "bx-wallet"},
	{invoicer.KindStatus, "statuses", "Statuses", "Status", "Responses and updates on exchanged documents", "sample-status.json", "bx-pulse"},
}

// docMeta returns the UI metadata for a singular document kind.
func docMeta(kind string) (route, title, subtitle, sample string) {
	for _, d := range docKinds {
		if d.kind == kind {
			return d.route, d.title, d.subtitle, d.sample
		}
	}
	return kind + "s", kind, "", ""
}

// docOne returns the singular display title ("Order") for a document kind.
func docOne(kind string) string {
	for _, d := range docKinds {
		if d.kind == kind {
			return d.one
		}
	}
	return kind
}

// docOverviews counts stored envelopes per document kind for the dashboard.
func docOverviews(entries []store.Entry) []DocOverview {
	counts := make(map[string]int)
	for _, e := range entries {
		if k := invoicer.DocumentKind(e.Envelope); k != "" {
			counts[k]++
		}
	}
	out := make([]DocOverview, 0, len(docKinds))
	for _, d := range docKinds {
		out = append(out, DocOverview{
			Route: d.route,
			Title: d.title,
			Count: counts[d.kind],
			Icon:  d.icon,
		})
	}
	return out
}

// deleteBillDocForm handles POST /{route}/{id}/delete from the list pages
// (Datastar buttons and plain forms). Datastar posts refresh the section in
// place; plain posts follow PRG redirects.
func (h *Handler) deleteBillDocForm(kind, route, title string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		env, err := h.entries.Get(r.Context(), id)
		if err != nil {
			if wantsFragment(r) {
				h.respondDocList(w, r, kind, true)
				return
			}
			if errors.Is(err, store.ErrNotFound) {
				http.Redirect(w, r, "/"+route, http.StatusSeeOther)
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if invoicer.DocumentKind(env) != kind {
			if wantsFragment(r) {
				h.respondDocList(w, r, kind, true)
				return
			}
			http.Redirect(w, r, "/"+route, http.StatusSeeOther)
			return
		}
		if _, err := h.entries.Delete(r.Context(), id); err != nil {
			if wantsFragment(r) {
				h.respondDocList(w, r, kind, true)
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if wantsFragment(r) {
			h.respondDocList(w, r, kind, true)
			return
		}
		http.Redirect(w, r, "/"+route, http.StatusSeeOther)
	}
}

// CreateOrder handles POST /orders.
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	h.createBillDoc(invoicer.KindOrder)(w, r)
}

// ListOrders handles GET /orders.
func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	h.listBillDocs(invoicer.KindOrder)(w, r)
}

// GetOrder handles GET /orders/{id}.
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	h.getBillDoc(invoicer.KindOrder)(w, r)
}

// OrderHTML handles GET /orders/{id}/html.
func (h *Handler) OrderHTML(w http.ResponseWriter, r *http.Request) {
	h.htmlBillDoc(invoicer.KindOrder)(w, r)
}

// DeleteOrder handles DELETE /orders/{id}.
func (h *Handler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	h.deleteBillDoc(invoicer.KindOrder)(w, r)
}

// DeleteOrderForm handles POST /orders/{id}/delete from the list page.
func (h *Handler) DeleteOrderForm(w http.ResponseWriter, r *http.Request) {
	route, title, _, _ := docMeta(invoicer.KindOrder)
	h.deleteBillDocForm(invoicer.KindOrder, route, title)(w, r)
}

// CreateDelivery handles POST /deliveries.
func (h *Handler) CreateDelivery(w http.ResponseWriter, r *http.Request) {
	h.createBillDoc(invoicer.KindDelivery)(w, r)
}

// ListDeliveries handles GET /deliveries.
func (h *Handler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	h.listBillDocs(invoicer.KindDelivery)(w, r)
}

// GetDelivery handles GET /deliveries/{id}.
func (h *Handler) GetDelivery(w http.ResponseWriter, r *http.Request) {
	h.getBillDoc(invoicer.KindDelivery)(w, r)
}

// DeliveryHTML handles GET /deliveries/{id}/html.
func (h *Handler) DeliveryHTML(w http.ResponseWriter, r *http.Request) {
	h.htmlBillDoc(invoicer.KindDelivery)(w, r)
}

// DeleteDelivery handles DELETE /deliveries/{id}.
func (h *Handler) DeleteDelivery(w http.ResponseWriter, r *http.Request) {
	h.deleteBillDoc(invoicer.KindDelivery)(w, r)
}

// DeleteDeliveryForm handles POST /deliveries/{id}/delete from the list page.
func (h *Handler) DeleteDeliveryForm(w http.ResponseWriter, r *http.Request) {
	route, title, _, _ := docMeta(invoicer.KindDelivery)
	h.deleteBillDocForm(invoicer.KindDelivery, route, title)(w, r)
}

// CreatePayment handles POST /payments.
func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	h.createBillDoc(invoicer.KindPayment)(w, r)
}

// ListPayments handles GET /payments.
func (h *Handler) ListPayments(w http.ResponseWriter, r *http.Request) {
	h.listBillDocs(invoicer.KindPayment)(w, r)
}

// GetPayment handles GET /payments/{id}.
func (h *Handler) GetPayment(w http.ResponseWriter, r *http.Request) {
	h.getBillDoc(invoicer.KindPayment)(w, r)
}

// PaymentHTML handles GET /payments/{id}/html.
func (h *Handler) PaymentHTML(w http.ResponseWriter, r *http.Request) {
	h.htmlBillDoc(invoicer.KindPayment)(w, r)
}

// DeletePayment handles DELETE /payments/{id}.
func (h *Handler) DeletePayment(w http.ResponseWriter, r *http.Request) {
	h.deleteBillDoc(invoicer.KindPayment)(w, r)
}

// DeletePaymentForm handles POST /payments/{id}/delete from the list page.
func (h *Handler) DeletePaymentForm(w http.ResponseWriter, r *http.Request) {
	route, title, _, _ := docMeta(invoicer.KindPayment)
	h.deleteBillDocForm(invoicer.KindPayment, route, title)(w, r)
}

// CreateStatus handles POST /statuses.
func (h *Handler) CreateStatus(w http.ResponseWriter, r *http.Request) {
	h.createBillDoc(invoicer.KindStatus)(w, r)
}

// ListStatuses handles GET /statuses.
func (h *Handler) ListStatuses(w http.ResponseWriter, r *http.Request) {
	h.listBillDocs(invoicer.KindStatus)(w, r)
}

// GetStatus handles GET /statuses/{id}.
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	h.getBillDoc(invoicer.KindStatus)(w, r)
}

// StatusHTML handles GET /statuses/{id}/html.
func (h *Handler) StatusHTML(w http.ResponseWriter, r *http.Request) {
	h.htmlBillDoc(invoicer.KindStatus)(w, r)
}

// DeleteStatus handles DELETE /statuses/{id}.
func (h *Handler) DeleteStatus(w http.ResponseWriter, r *http.Request) {
	h.deleteBillDoc(invoicer.KindStatus)(w, r)
}

// DeleteStatusForm handles POST /statuses/{id}/delete from the list page.
func (h *Handler) DeleteStatusForm(w http.ResponseWriter, r *http.Request) {
	route, title, _, _ := docMeta(invoicer.KindStatus)
	h.deleteBillDocForm(invoicer.KindStatus, route, title)(w, r)
}

// updateBillDoc handles POST and PUT /{route}/{id}: it rebuilds the
// envelope from the new document (calculate + validate) while preserving
// the envelope UUID, so links and IDs stay stable. JSON clients get
// 200/400/404/422 JSON; UI posts re-render the section or redirect.
func (h *Handler) updateBillDoc(kind string) http.HandlerFunc {
	route, title, _, _ := docMeta(kind)
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		existing, err := h.entries.Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				if wantsFragment(r) {
					h.respondDocList(w, r, kind, true)
					return
				}
				writeError(w, http.StatusNotFound, err)
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if invoicer.DocumentKind(existing) != kind {
			if wantsFragment(r) {
				h.respondDocList(w, r, kind, true)
				return
			}
			writeError(w, http.StatusNotFound, store.ErrNotFound)
			return
		}
		doc, isJSON, err := h.decodeBillDoc(r, kind)
		if err != nil {
			h.updateDocError(w, r, kind, route, title, id, err)
			return
		}
		env, err := buildBillDoc(kind, doc)
		if err != nil {
			h.updateDocError(w, r, kind, route, title, id, err)
			return
		}
		env.Head.UUID = existing.Head.UUID
		if err := h.entries.Put(r.Context(), id, env); err != nil {
			if isJSON {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			h.updateDocError(w, r, kind, route, title, id, err)
			return
		}
		if isJSON {
			writeJSON(w, http.StatusOK, map[string]any{"id": id, "envelope": env})
			return
		}
		if wantsFragment(r) {
			h.respondDocList(w, r, kind, true)
			return
		}
		http.Redirect(w, r, "/"+route, http.StatusSeeOther)
	}
}

// updateDocError reports an update failure: JSON for API clients, the Edit
// form re-rendered with the user's input and the error for UI posts.
func (h *Handler) updateDocError(w http.ResponseWriter, r *http.Request, kind, route, title, id string, err error) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		if isDocValidationError(err) {
			writeError(w, http.StatusUnprocessableEntity, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	form := docFormFromRequest(r, kind)
	if form.Type == "" && form.Series == "" && form.Code == "" {
		if env, e := h.entries.Get(r.Context(), id); e == nil {
			form = docFormFromEnvelope(kind, env)
		}
	}
	form.Err = err.Error()
	form.Action = "/" + route + "/" + id
	form.Submit = "Save " + form.One
	h.attachCustomers(r.Context(), &form)
	h.attachSuppliers(r.Context(), &form)
	h.attachProducts(r.Context(), &form)
	if wantsFragment(r) {
		if r.FormValue("modal") == "1" {
			form.Modal = true
			respondFragment(w, r, "#modal-body", DocFormBody(form))
			return
		}
		respondFragment(w, r, "#main", EditDocSection(route, docOne(kind), title, id, form))
		return
	}
	http.Redirect(w, r, "/"+route+"/"+id+"/edit?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
}

// newDocPage renders the UI "New" form for a document kind, prefilled with
// defaults. It always returns HTML (also for API-style Accept).
func (h *Handler) newDocPage(kind string) http.HandlerFunc {
	route, title, _, _ := docMeta(kind)
	return func(w http.ResponseWriter, r *http.Request) {
		form := emptyDocForm(kind)
		form.Err = r.URL.Query().Get("error")
		form.Action = "/" + route
		form.Submit = "Create " + form.One
		h.attachCustomers(r.Context(), &form)
		h.attachSuppliers(r.Context(), &form)
		h.attachProducts(r.Context(), &form)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := NewDocPage(route, docOne(kind), title, form).Render(r.Context(), w); err != nil {
			log.Printf("render new %s: %v", kind, err)
		}
	}
}

// editDocForm renders the UI "Edit" form for one document, prefilled with
// its current JSON. 404 when the id is unknown or holds another kind.
func (h *Handler) editDocForm(kind string) http.HandlerFunc {
	route, title, _, _ := docMeta(kind)
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		env, err := h.entries.Get(r.Context(), id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				http.Redirect(w, r, "/"+route, http.StatusSeeOther)
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if invoicer.DocumentKind(env) != kind {
			http.Redirect(w, r, "/"+route, http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		form := docFormFromEnvelope(kind, env)
		form.Err = r.URL.Query().Get("error")
		form.Action = "/" + route + "/" + id
		form.Submit = "Save " + form.One
		h.attachCustomers(r.Context(), &form)
		h.attachSuppliers(r.Context(), &form)
		h.attachProducts(r.Context(), &form)
		if err := EditDocPage(route, docOne(kind), title, id, form).Render(r.Context(), w); err != nil {
			log.Printf("render edit %s: %v", kind, err)
		}
	}
}

// UpdateOrder handles POST/PUT /orders/{id}.
func (h *Handler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	h.updateBillDoc(invoicer.KindOrder)(w, r)
}

// NewOrderPage handles GET /orders/new.
func (h *Handler) NewOrderPage(w http.ResponseWriter, r *http.Request) {
	h.newDocPage(invoicer.KindOrder)(w, r)
}

// EditOrderForm handles GET /orders/{id}/edit.
func (h *Handler) EditOrderForm(w http.ResponseWriter, r *http.Request) {
	h.editDocForm(invoicer.KindOrder)(w, r)
}

// UpdateDelivery handles POST/PUT /deliveries/{id}.
func (h *Handler) UpdateDelivery(w http.ResponseWriter, r *http.Request) {
	h.updateBillDoc(invoicer.KindDelivery)(w, r)
}

// NewDeliveryPage handles GET /deliveries/new.
func (h *Handler) NewDeliveryPage(w http.ResponseWriter, r *http.Request) {
	h.newDocPage(invoicer.KindDelivery)(w, r)
}

// EditDeliveryForm handles GET /deliveries/{id}/edit.
func (h *Handler) EditDeliveryForm(w http.ResponseWriter, r *http.Request) {
	h.editDocForm(invoicer.KindDelivery)(w, r)
}

// UpdatePayment handles POST/PUT /payments/{id}.
func (h *Handler) UpdatePayment(w http.ResponseWriter, r *http.Request) {
	h.updateBillDoc(invoicer.KindPayment)(w, r)
}

// NewPaymentPage handles GET /payments/new.
func (h *Handler) NewPaymentPage(w http.ResponseWriter, r *http.Request) {
	h.newDocPage(invoicer.KindPayment)(w, r)
}

// EditPaymentForm handles GET /payments/{id}/edit.
func (h *Handler) EditPaymentForm(w http.ResponseWriter, r *http.Request) {
	h.editDocForm(invoicer.KindPayment)(w, r)
}

// UpdateStatus handles POST/PUT /statuses/{id}.
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	h.updateBillDoc(invoicer.KindStatus)(w, r)
}

// NewStatusPage handles GET /statuses/new.
func (h *Handler) NewStatusPage(w http.ResponseWriter, r *http.Request) {
	h.newDocPage(invoicer.KindStatus)(w, r)
}

// EditStatusForm handles GET /statuses/{id}/edit.
func (h *Handler) EditStatusForm(w http.ResponseWriter, r *http.Request) {
	h.editDocForm(invoicer.KindStatus)(w, r)
}

// docOverviewHref is the no-JS link for a dashboard card: the invoices list
// lives on the dashboard itself (route "/"), the rest have their own pages.
func docOverviewHref(route string) string {
	if route == "invoices" {
		return "/"
	}
	return "/" + route
}

// dsDocOverviewNav swaps the dashboard card target into #main in place.
func dsDocOverviewNav(route string) string {
	return "$nav = '" + route + "'; @get('/partials/" + route + "')"
}

// respondDocList sends the list section for a kind over SSE, optionally
// closing the modal (used after a successful create/update).
func (h *Handler) respondDocList(w http.ResponseWriter, r *http.Request, kind string, closeModal bool) {
	raw, err := h.entries.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	route, title, subtitle, sample := docMeta(kind)
	summaries := make([]*invoicer.Summary, 0, len(raw))
	for _, e := range raw {
		if s := invoicer.SummarizeAny(e.Envelope); s != nil && s.Document == kind {
			summaries = append(summaries, s)
		}
	}
	stats := invoicer.SummarizeStats(summaries)
	var section templ.Component
	if kind == invoicer.KindInvoice {
		section = InvoicesSection(summaries, stats, true)
	} else {
		section = DocumentsSection(route, title, docOne(kind), subtitle, sample, summaries, stats)
	}
	sseBegin(w)
	if closeModal {
		sseSignals(w, `{"modalOpen": false}`)
	}
	sseElements(w, r, "#main", "inner", section)
	sseFlush(w)
}

// documentsPartial renders the Datastar fragment for a document kind
// (sidebar navigation — also closes any open modal).
func (h *Handler) documentsPartial(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.respondDocList(w, r, kind, true)
	}
}

// OrdersPartial handles GET /partials/orders.
func (h *Handler) OrdersPartial(w http.ResponseWriter, r *http.Request) {
	h.documentsPartial(invoicer.KindOrder)(w, r)
}

// DeliveriesPartial handles GET /partials/deliveries.
func (h *Handler) DeliveriesPartial(w http.ResponseWriter, r *http.Request) {
	h.documentsPartial(invoicer.KindDelivery)(w, r)
}

// PaymentsPartial handles GET /partials/payments.
func (h *Handler) PaymentsPartial(w http.ResponseWriter, r *http.Request) {
	h.documentsPartial(invoicer.KindPayment)(w, r)
}

// StatusesPartial handles GET /partials/statuses.
func (h *Handler) StatusesPartial(w http.ResponseWriter, r *http.Request) {
	h.documentsPartial(invoicer.KindStatus)(w, r)
}

// newDocModal handles GET /partials/{route}/new: the New form fragment
// rendered into the modal body.
func (h *Handler) newDocModal(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		form := emptyDocForm(kind)
		form.Modal = true
		form.Action = "/" + form.Route
		form.Submit = "Create " + form.One
		h.attachCustomers(r.Context(), &form)
		h.attachSuppliers(r.Context(), &form)
		h.attachProducts(r.Context(), &form)
		respondFragment(w, r, "#modal-body", DocFormBody(form))
	}
}

// editDocModal handles GET /partials/{route}/{id}/edit: the Edit form
// fragment rendered into the modal body.
func (h *Handler) editDocModal(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		env, err := h.entries.Get(r.Context(), id)
		if err != nil || invoicer.DocumentKind(env) != kind {
			respondFragment(w, r, "#modal-body", DocFormBody(emptyDocForm(kind)))
			return
		}
		form := docFormFromEnvelope(kind, env)
		form.Modal = true
		form.Action = "/" + form.Route + "/" + id
		form.Submit = "Save " + form.One
		h.attachCustomers(r.Context(), &form)
		h.attachSuppliers(r.Context(), &form)
		h.attachProducts(r.Context(), &form)
		respondFragment(w, r, "#modal-body", DocFormBody(form))
	}
}

// UpdateInvoice handles POST/PUT /invoices/{id}.
func (h *Handler) UpdateInvoice(w http.ResponseWriter, r *http.Request) {
	h.updateBillDoc(invoicer.KindInvoice)(w, r)
}

// NewInvoicePage handles GET /invoices/new.
func (h *Handler) NewInvoicePage(w http.ResponseWriter, r *http.Request) {
	h.newDocPage(invoicer.KindInvoice)(w, r)
}

// EditInvoiceForm handles GET /invoices/{id}/edit.
func (h *Handler) EditInvoiceForm(w http.ResponseWriter, r *http.Request) {
	h.editDocForm(invoicer.KindInvoice)(w, r)
}

// DeleteInvoiceForm handles POST /invoices/{id}/delete from the list page.
func (h *Handler) DeleteInvoiceForm(w http.ResponseWriter, r *http.Request) {
	route, title, _, _ := docMeta(invoicer.KindInvoice)
	h.deleteBillDocForm(invoicer.KindInvoice, route, title)(w, r)
}

// NewInvoiceModal handles GET /partials/invoices/new.
func (h *Handler) NewInvoiceModal(w http.ResponseWriter, r *http.Request) {
	h.newDocModal(invoicer.KindInvoice)(w, r)
}

// EditInvoiceModal handles GET /partials/invoices/{id}/edit.
func (h *Handler) EditInvoiceModal(w http.ResponseWriter, r *http.Request) {
	h.editDocModal(invoicer.KindInvoice)(w, r)
}

// NewOrderModal handles GET /partials/orders/new.
func (h *Handler) NewOrderModal(w http.ResponseWriter, r *http.Request) {
	h.newDocModal(invoicer.KindOrder)(w, r)
}

// EditOrderModal handles GET /partials/orders/{id}/edit.
func (h *Handler) EditOrderModal(w http.ResponseWriter, r *http.Request) {
	h.editDocModal(invoicer.KindOrder)(w, r)
}

// NewDeliveryModal handles GET /partials/deliveries/new.
func (h *Handler) NewDeliveryModal(w http.ResponseWriter, r *http.Request) {
	h.newDocModal(invoicer.KindDelivery)(w, r)
}

// EditDeliveryModal handles GET /partials/deliveries/{id}/edit.
func (h *Handler) EditDeliveryModal(w http.ResponseWriter, r *http.Request) {
	h.editDocModal(invoicer.KindDelivery)(w, r)
}

// NewPaymentModal handles GET /partials/payments/new.
func (h *Handler) NewPaymentModal(w http.ResponseWriter, r *http.Request) {
	h.newDocModal(invoicer.KindPayment)(w, r)
}

// EditPaymentModal handles GET /partials/payments/{id}/edit.
func (h *Handler) EditPaymentModal(w http.ResponseWriter, r *http.Request) {
	h.editDocModal(invoicer.KindPayment)(w, r)
}

// NewStatusModal handles GET /partials/statuses/new.
func (h *Handler) NewStatusModal(w http.ResponseWriter, r *http.Request) {
	h.newDocModal(invoicer.KindStatus)(w, r)
}

// EditStatusModal handles GET /partials/statuses/{id}/edit.
func (h *Handler) EditStatusModal(w http.ResponseWriter, r *http.Request) {
	h.editDocModal(invoicer.KindStatus)(w, r)
}
