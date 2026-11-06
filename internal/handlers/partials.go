package handlers

import (
	"errors"
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
	sseBegin(w)
	sseSignals(w, `{"modalOpen": false}`)
	sseElements(w, r, "#main", "inner", InvoicesSection(summaries, invoicer.SummarizeStats(summaries), true))
	sseFlush(w)
}

// CustomersPartial renders the customers list fragment.
func (h *Handler) CustomersPartial(w http.ResponseWriter, r *http.Request) {
	rows, err := h.catalog.ListCustomers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	respondFragment(w, r, "#main", CustomersSection(rows, ""))
}

// SuppliersPartial renders the suppliers list fragment.
func (h *Handler) SuppliersPartial(w http.ResponseWriter, r *http.Request) {
	rows, err := h.catalog.ListSuppliers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	respondFragment(w, r, "#main", SuppliersSection(rows, ""))
}

// ProductsPartial renders the products list fragment.
func (h *Handler) ProductsPartial(w http.ResponseWriter, r *http.Request) {
	rows, err := h.catalog.ListProducts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	respondFragment(w, r, "#main", ProductsSection(rows, ""))
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
	respondFragment(w, r, "#modal-body", ProductForm(p, ""))
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
	respondFragment(w, r, "#modal-body", CustomerForm(c, ""))
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
	respondFragment(w, r, "#modal-body", SupplierForm(sp, ""))
}

// NewCustomerModal handles GET /partials/customers/new: the compact create
// form rendered into the modal from a document form.
func (h *Handler) NewCustomerModal(w http.ResponseWriter, r *http.Request) {
	respondFragment(w, r, "#modal-body", CustomerCreateForm(CustomerFormData{
		For: r.URL.Query().Get("for"),
	}))
}
