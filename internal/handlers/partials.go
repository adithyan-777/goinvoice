package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/neo/goinvoice/internal/invoicer"
	"github.com/neo/goinvoice/internal/store"
)

// This file implements Datastar partials: page fragments that replace the
// #main content area in place, so navigation and mutations feel like an SPA
// (no full-page reloads).

// InvoicesPartial handles GET /partials/invoices and renders the invoice
// list fragment for in-place swaps.
func (h *Handler) InvoicesPartial(w http.ResponseWriter, r *http.Request) {
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
	fragmentHeaders(w, "#main")
	if err := InvoicesSection(summaries, invoicer.SummarizeStats(summaries)).Render(r.Context(), w); err != nil {
		log.Printf("render invoices partial: %v", err)
	}
}

// CustomersPartial renders the customers list fragment.
func (h *Handler) CustomersPartial(w http.ResponseWriter, r *http.Request) {
	rows, err := h.catalog.ListCustomers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	fragmentHeaders(w, "#main")
	if err := CustomersSection(rows, "").Render(r.Context(), w); err != nil {
		log.Printf("render customers partial: %v", err)
	}
}

// SuppliersPartial renders the suppliers list fragment.
func (h *Handler) SuppliersPartial(w http.ResponseWriter, r *http.Request) {
	rows, err := h.catalog.ListSuppliers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	fragmentHeaders(w, "#main")
	if err := SuppliersSection(rows, "").Render(r.Context(), w); err != nil {
		log.Printf("render suppliers partial: %v", err)
	}
}

// ProductsPartial renders the products list fragment.
func (h *Handler) ProductsPartial(w http.ResponseWriter, r *http.Request) {
	rows, err := h.catalog.ListProducts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	fragmentHeaders(w, "#main")
	if err := ProductsSection(rows, "").Render(r.Context(), w); err != nil {
		log.Printf("render products partial: %v", err)
	}
}

// ProductModal returns the product edit form fragment for modal injection.
func (h *Handler) ProductModal(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := h.catalog.GetProduct(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, store.ErrProductNotFound)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	fragmentHeaders(w, "#modal-body")
	if err := ProductForm(p, "").Render(r.Context(), w); err != nil {
		log.Printf("render product modal: %v", err)
	}
}

// CustomerModal returns the customer edit form fragment for modal injection.
func (h *Handler) CustomerModal(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, err := h.catalog.GetCustomer(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrCustomerNotFound) {
			writeError(w, http.StatusNotFound, store.ErrCustomerNotFound)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	fragmentHeaders(w, "#modal-body")
	if err := CustomerForm(c, "").Render(r.Context(), w); err != nil {
		log.Printf("render customer modal: %v", err)
	}
}

// SupplierModal returns the supplier edit form fragment for modal injection.
func (h *Handler) SupplierModal(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sp, err := h.catalog.GetSupplier(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrSupplierNotFound) {
			writeError(w, http.StatusNotFound, store.ErrSupplierNotFound)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	fragmentHeaders(w, "#modal-body")
	if err := SupplierForm(sp, "").Render(r.Context(), w); err != nil {
		log.Printf("render supplier modal: %v", err)
	}
}
