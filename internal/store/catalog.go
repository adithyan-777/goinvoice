package store

import (
	"context"

	"github.com/invopop/gobl/org"

	"github.com/neo/goinvoice/internal/db/sqlc"
)

// Catalog abstracts the customers/suppliers/products master data.
// Invoices keep embedded snapshots; these tables exist for reuse,
// autofill and listing pages. Parties are stored relationally: scalars on
// customers/suppliers, repeated collections in party_* child tables.
type Catalog interface {
	ListCustomers(ctx context.Context) ([]sqlc.Customer, error)
	GetCustomer(ctx context.Context, id string) (sqlc.Customer, error)
	GetCustomerParty(ctx context.Context, id string) (*org.Party, error)
	SaveCustomer(ctx context.Context, id string, p *org.Party) error
	DeleteCustomer(ctx context.Context, id string) (int64, error)
	ListSuppliers(ctx context.Context) ([]sqlc.Supplier, error)
	GetSupplier(ctx context.Context, id string) (sqlc.Supplier, error)
	GetSupplierParty(ctx context.Context, id string) (*org.Party, error)
	SaveSupplier(ctx context.Context, id string, p *org.Party) error
	DeleteSupplier(ctx context.Context, id string) (int64, error)
	ListProducts(ctx context.Context) ([]sqlc.Product, error)
	GetProduct(ctx context.Context, id string) (sqlc.Product, error)
	GetProductItem(ctx context.Context, id string) (*org.Item, error)
	SaveProduct(ctx context.Context, id string, item *org.Item) error
	DeleteProduct(ctx context.Context, id string) (int64, error)
}
