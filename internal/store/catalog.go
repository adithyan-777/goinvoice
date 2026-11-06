package store

import (
	"context"

	"github.com/neo/goinvoice/internal/db/sqlc"
)

// Catalog abstracts the customers/suppliers/products master data.
// Invoices keep embedded snapshots; these tables exist for reuse,
// autofill and listing pages.
type Catalog interface {
	ListCustomers(ctx context.Context) ([]sqlc.Customer, error)
	GetCustomer(ctx context.Context, id string) (sqlc.Customer, error)
	CreateCustomer(ctx context.Context, c sqlc.CreateCustomerParams) error
	UpdateCustomer(ctx context.Context, c sqlc.UpdateCustomerParams) (int64, error)
	DeleteCustomer(ctx context.Context, id string) (int64, error)
	ListSuppliers(ctx context.Context) ([]sqlc.Supplier, error)
	GetSupplier(ctx context.Context, id string) (sqlc.Supplier, error)
	CreateSupplier(ctx context.Context, s sqlc.CreateSupplierParams) error
	UpdateSupplier(ctx context.Context, s sqlc.UpdateSupplierParams) (int64, error)
	DeleteSupplier(ctx context.Context, id string) (int64, error)
	ListProducts(ctx context.Context) ([]sqlc.Product, error)
	CreateProduct(ctx context.Context, p sqlc.CreateProductParams) error
	GetProduct(ctx context.Context, id string) (sqlc.Product, error)
	UpdateProduct(ctx context.Context, p sqlc.UpdateProductParams) (int64, error)
	DeleteProduct(ctx context.Context, id string) (int64, error)
}
