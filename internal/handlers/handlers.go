// Package handlers defines the Gin routes and their implementations.
package handlers

import (
	"net/http"

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
