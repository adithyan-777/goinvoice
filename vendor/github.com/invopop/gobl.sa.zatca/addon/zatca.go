// Package zatca provides extensions and validations for the Saudi Arabia
// ZATCA (Zakat, Tax and Customs Authority) e-invoicing requirements.
package zatca

import (
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/norm"
	"github.com/invopop/gobl/pkg/here"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
	"github.com/invopop/gobl/tax"
)

const (
	// Namespace is the rules namespace for the Saudi Arabia ZATCA addon.
	Namespace rules.Code = "SA-ZATCA"

	// Key identifies the Saudi Arabia ZATCA addon family.
	Key cbc.Key = "sa-zatca"

	// V1 is the first version of the Saudi Arabia ZATCA addon.
	V1 cbc.Key = Key + "-v1"

	// StampQR is the base64-encoded TLV QR code carried by every ZATCA invoice.
	StampQR cbc.Key = "zatca-qr"
)

func init() {
	tax.RegisterAddonDef(newV1Addon())
	rules.RegisterWithGuard(
		Key.String(),
		rules.GOBL.Add(Namespace),
		is.InContext(tax.AddonIn(V1)),
		billInvoiceRules(),
		billLineRules(),
		orgAddressRules(),
		taxComboRules(),
	)
	norm.RegisterWithGuard(
		is.InContext(tax.AddonIn(V1)),
		norm.For(normalizeInvoice),
	)
}

func newV1Addon() *tax.AddonDef {
	return &tax.AddonDef{
		Key: V1,
		Name: i18n.String{
			i18n.EN: "Saudi Arabia ZATCA",
			i18n.AR: "هيئة الزكاة والضريبة والجمارك",
		},
		Requires: []cbc.Key{
			en16931.V2017,
		},
		Description: i18n.String{
			i18n.EN: here.Doc(`
				Support for the Saudi Arabia ZATCA (Zakat, Tax and Customs Authority) e-invoicing
				requirements based on UBL 2.1 with EN 16931 as an intermediate layer and KSA-specific
				extensions (BR-KSA-* rules).

				ZATCA e-invoicing covers both standard tax invoices (B2B/B2G) sent for clearance
				and simplified tax invoices (B2C) sent for reporting through the FATOORA platform.

				This addon extends EN 16931 with Saudi-specific fields and validations including
				invoice type transactions, address requirements, and supply date handling.
			`),
		},
		Sources: []*cbc.Source{
			{
				Title: i18n.NewString("ZATCA E-Invoicing Developer Portal"),
				URL:   "https://zatca.gov.sa/en/E-Invoicing/SystemsDevelopers/Pages/E-Invoice-specifications.aspx",
			},
		},
		Extensions: extensions,
		Tags:       tags,
		Scenarios:  scenarios,
	}
}
