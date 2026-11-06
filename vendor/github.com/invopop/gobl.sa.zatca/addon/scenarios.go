package zatca

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
)

// scenarios overrides the EN 16931 UNTDID document type for standard invoices:
// ZATCA requires "388" (tax invoice) where EN 16931 defaults to "380". Credit
// and debit notes already map to ZATCA-accepted codes (381 / 383) in EN 16931.
var scenarios = []*tax.ScenarioSet{
	{
		Schema: bill.ShortSchemaInvoice,
		List: []*tax.Scenario{
			{
				Types: []cbc.Key{
					bill.InvoiceTypeStandard,
				},
				Ext: tax.ExtensionsOf(cbc.CodeMap{
					untdid.ExtKeyDocumentType: "388",
				}),
			},
		},
	},
}
