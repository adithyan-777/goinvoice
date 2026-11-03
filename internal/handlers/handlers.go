// Package handlers defines the Gin routes and their implementations.
package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/invopop/gobl/bill"

	"github.com/neo/goinvoice/internal/invoicer"
	"github.com/neo/goinvoice/internal/store"
)

// Handler bundles dependencies used by the route implementations.
type Handler struct {
	entries *store.Store
}

// New wires a handler with the given store.
func New(entries *store.Store) *Handler {
	return &Handler{entries: entries}
}

// Register mounts all invoice routes on the engine.
func (h *Handler) Register(r gin.IRouter) {
	r.GET("/", h.Index)
	inv := r.Group("/invoices")
	{
		inv.POST("", h.Create)
		inv.GET("", h.List)
		inv.GET("/:id", h.Get)
		inv.GET("/:id/html", h.HTML)
		inv.DELETE("/:id", h.Delete)
	}
}

// Create handles POST /invoices.
//
// Accepts a bill.Invoice-shaped JSON body. GOBL fills in totals, taxes and
// validation errors are reported with 422 so clients can distinguish them
// from malformed JSON (400).
//
// Query params:
//   - dry_run=true  → calculate + validate, but don't store. Returns the envelope.
//   - html=true     → also store AND return the rendered HTML document.
func (h *Handler) Create(c *gin.Context) {
	inv := new(bill.Invoice)
	// GOBL's num.Amount expects strings ("100.00") to avoid float precision
	// issues — clients must send amounts as JSON strings, not numbers.
	if err := c.ShouldBindJSON(inv); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	env, err := invoicer.Build(inv)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	id := invoicer.ID(env)

	if c.Query("dry_run") == "true" {
		c.JSON(http.StatusOK, gin.H{"id": id, "envelope": env})
		return
	}

	h.entries.Put(id, env)

	if c.Query("html") == "true" {
		c.Header("Location", "/invoices/"+id+"/html")
		c.JSON(http.StatusCreated, gin.H{
			"id":       id,
			"summary":  invoicer.Summarize(env),
			"html_url": "/invoices/" + id + "/html",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "envelope": env})
}

// List handles GET /invoices and returns compact summaries.
func (h *Handler) List(c *gin.Context) {
	raw := h.entries.List()
	out := make([]*invoicer.Summary, 0, len(raw))
	for _, e := range raw {
		if s := invoicer.Summarize(e.Envelope); s != nil {
			out = append(out, s)
		}
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

// Get handles GET /invoices/:id and returns the full envelope as JSON.
func (h *Handler) Get(c *gin.Context) {
	env, ok := h.entries.Get(c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": c.Param("id"), "envelope": env})
}

// HTML handles GET /invoices/:id/html?locale=es&watermark=DRAFT&sandbox=true
// and returns a printable HTML document ready for PDF conversion.
func (h *Handler) HTML(c *gin.Context) {
	env, ok := h.entries.Get(c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}

	opts := invoicer.RenderOptions{
		Locale:    c.Query("locale"),
		Watermark: c.Query("watermark"),
		Sandbox:   c.Query("sandbox") == "true",
	}

	html, err := invoicer.Render(c.Request.Context(), env, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", html)
}

// Delete handles DELETE /invoices/:id.
func (h *Handler) Delete(c *gin.Context) {
	if !h.entries.Delete(c.Param("id")) {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

// Index handles GET / and renders a simple browsable list of stored invoices
// with links to the printable HTML view and the raw envelope JSON.
func (h *Handler) Index(c *gin.Context) {
	raw := h.entries.List()

	var b strings.Builder
	b.WriteString(`<!doctype html><meta charset="utf-8">` +
		`<title>goinvoice</title>` +
		`<style>
			body{font-family:system-ui,sans-serif;max-width:60rem;margin:2rem auto;padding:0 1rem;color:#222}
			table{border-collapse:collapse;width:100%}
			th,td{padding:.4rem .6rem;border-bottom:1px solid #ddd;text-align:left;font-size:.9rem}
			th{border-bottom:2px solid #999}
			th.num,td.num{text-align:right}
			a{color:#0645ad;text-decoration:none}a:hover{text-decoration:underline}
			.muted{color:#666;font-size:.85rem}
		</style>` +
		`<h1>goinvoice</h1>`)

	if len(raw) == 0 {
		b.WriteString(`<p>No invoices yet. Create one with:</p><pre style="background:#f5f5f5;padding:1rem">` +
			`curl -s -X POST http://localhost:8080/invoices \\\n` +
			`  -H 'Content-Type: application/json' \\\n` +
			`  -d @examples/sample-invoice.json &gt; /dev/null</pre>`)
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(b.String()))
		return
	}

	b.WriteString(`<table><tr>` +
		`<th>Code</th><th>Date</th><th>Customer</th>` +
		`<th class="num">Tax</th><th class="num">Payable</th><th></th></tr>`)
	for _, e := range raw {
		s := invoicer.Summarize(e.Envelope)
		if s == nil {
			continue
		}
		b.WriteString(`<tr>` +
			`<td>` + s.Code + `</td>` +
			`<td>` + s.IssueDate + `</td>` +
			`<td>` + s.Customer + `</td>` +
			`<td class="num">` + s.Tax + `</td>` +
			`<td class="num">` + s.Payable + `</td>` +
			`<td><a href="/invoices/` + s.ID + `/html">view</a> · ` +
			`<a href="/invoices/` + s.ID + `/envelope.json">json</a></td>` +
			`</tr>`)
	}
	b.WriteString(`</table><p class="muted">` +
		`Amounts as stored in the envelope · rendered by gobl.html</p>`)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(b.String()))
}
