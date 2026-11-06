package zatca

import (
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/catalogues/cef"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
	"github.com/invopop/gobl/tax"
)

var (
	// ZATCA party identification scheme codes valid for a customer.
	customerValidIdentities = []cbc.Code{"TIN", "CRN", "MOM", "MLS", "700", "SAG", "NAT", "GCC", "IQA", "PAS", "OTH"}

	// SupplierValidIdentities holds the scheme codes valid for a supplier.
	SupplierValidIdentities = []cbc.Code{"CRN", "MOM", "MLS", "700", "SAG", "OTH"}
)

func billInvoiceRules() *rules.Set {
	return rules.For(new(bill.Invoice),

		rules.Field("issue_time",
			rules.Assert("01", "invoice issue time must be present (BR-KSA-70)", is.Present),
		),

		// Tax
		rules.Field("tax",
			rules.Assert("02", "invoice tax must be present", is.Present),
			rules.Field("ext",
				rules.Assert("03", "invoice tax untdid-document-type extension is required",
					tax.ExtensionsRequire(untdid.ExtKeyDocumentType),
				),
				rules.Assert("04", "invoice tax sa-zatca-invoice-type extension is required",
					tax.ExtensionsRequire(ExtKeyInvoiceType),
				),
				rules.Assert("05", "invoice tax untdid-document-type extension must be a valid ZATCA type (388, 386, 383, 381) (BR-KSA-05)",
					tax.ExtensionsHasCodes(untdid.ExtKeyDocumentType, "388", "386", "383", "381"),
				),
				rules.Assert("06", "invoice tax sa-zatca-invoice-type extension must be valid (BR-KSA-06)",
					tax.ExtensionsHasCodes(ExtKeyInvoiceType, validTransactionTypes...),
				),
			),
		),

		// Credit or debit note
		rules.When(
			bill.InvoiceTypeIn(bill.InvoiceTypeCreditNote, bill.InvoiceTypeDebitNote),
			rules.Field("preceding",
				rules.Assert("07", "credit and debit notes must have a billing reference", is.Present),
				rules.Each(
					rules.Field("code",
						rules.Assert("08", "credit or debit note billing reference must have an identifier (BR-KSA-56)", is.Present),
					),
					rules.Field("reason",
						rules.Assert("09", "credit and debit notes must contain the reason for issuance (BR-KSA-17)",
							is.Present,
						),
					),
				),
			),
			rules.Field("payment",
				rules.Assert("10", "credit or debit notes must have payment details", is.Present),
				rules.Field("instructions",
					rules.Assert("11", "credit or debit notes must have payment instructions", is.Present),
					rules.Field("key",
						rules.Assert("12", "credit or debit note payment instructions must have a key (BR-KSA-17)", is.Present),
					),
				),
			),
		),

		// Standard
		rules.When(
			is.Func("standard tax invoice", invoiceIsStandard),
			rules.Field("customer",
				rules.Field("addresses",
					rules.Each(
						rules.Field("street",
							rules.Assert("13", "invoice customer address must have a street name for standard tax invoices (BR-KSA-10)", is.Present),
						),
						rules.Field("locality",
							rules.Assert("14", "invoice customer address must have a city name for standard tax invoices (BR-KSA-10)", is.Present),
						),
						rules.Field("country",
							rules.Assert("15", "invoice customer address must have a country code for standard tax invoices (BR-KSA-10)", is.Present),
						),
					),
				),
				rules.Assert("16", "invoice customer must have a valid identification scheme for standard invoices",
					is.Func("invoice customer must have a tax id code or a valid identification (TIN/CRN/MOM/MLS/700/SAG/NAT/GCC/IQA/PAS/OTH) for standard invoices (BR-KSA-14), (BR-KSA-81)",
						customerValidIdentity),
				),
			),
			rules.Field("lines",
				rules.Each(
					rules.Field("taxes",
						rules.Assert("17", "invoice line taxes are required for standard tax invoices (BR-KSA-52)", is.Present),
					),
				),
			),
			rules.Field("delivery",
				rules.Assert("18", "invoice delivery must be present for standard tax invoices", is.Present),
				rules.Field("date",
					rules.Assert("19", "invoice delivery must have a supply date for standard tax invoices (BR-KSA-15)", is.Present),
				),
			),
		),

		// Export invoice
		rules.When(
			is.Func("export invoice", invoiceIsExport),
			rules.Field("customer",
				rules.Field("tax_id",
					rules.Assert("20", "invoice customer must not have a tax id for export invoices (BR-KSA-46)",
						is.Empty,
					),
				),
			),
		),

		// Simplified and summary
		rules.When(
			is.Or(
				is.Func("invoice is simplified and summary", invoiceIsSimplifiedAndSummary),
			),
			rules.Field("delivery",
				rules.Assert("21", "invoice delivery must be present for simplified and summary invoices", is.Present),
				rules.Field("period",
					rules.Assert("22", "invoice supply must have a delivery period for simplified and summary invoices", is.Present),
					rules.Field("start",
						rules.Assert("23", "invoice delivery start date must be present for simplified and summary invoices (BR-KSA-72)", is.Present),
					),
					rules.Field("end",
						rules.Assert("24", "invoice delivery end date must be present for simplified and summary invoices (BR-KSA-72)", is.Present),
					),
				),
			),
		),

		// EDU or HEA exemptions
		rules.When(
			is.Func("has EDU or HEA tax exemption", invoiceHasEDUOrHEAExemption),
			rules.Field("customer",
				rules.Field("identities",
					rules.Assert("25", "invoice customer must have a national ID (NAT) when tax exemption is VATEX-SA-EDU or VATEX-SA-HEA (BR-KSA-49)",
						org.IdentitiesTypeIn("NAT"),
					),
				),
			),
		),

		// Customer name
		rules.When(
			is.Or(
				is.Func("simplified and (EDU or HEA exemptions)", invoiceIsSimplifiedAndEDUOrHEAExemption),
				is.Func("invoice is simplified and summary", invoiceIsSimplifiedAndSummary),
				is.Func("standard tax invoice", invoiceIsStandard),
			),
			rules.Field("customer",
				rules.Assert("26",
					"invoice customer must be present for standard tax invoices, simplified summary invoices, and simplified invoices with EDU or HEA exemptions",
					is.Present),
				rules.Field("name",
					rules.Assert("27",
						"invoice customer must be present for standard tax invoices, simplified summary invoices, and simplified invoices with EDU or HEA exemptions (BR-KSA-71), (BR-KSA-25), (BR-KSA-42)",
						is.Present),
				),
			),
		),

		// Supplier
		rules.Field("supplier",
			rules.Field("tax_id",
				rules.Assert("28", "invoice supplier must have a tax id (BR-KSA-39)", is.Present),
				rules.Field("code",
					rules.Assert("29", "invoice supplier must have a tax id code (BR-KSA-39)", is.Present),
				),
			),
			rules.Field("identities",
				rules.Assert("30", "invoice supplier must have a valid identity (CRN/MOM/MLS/700/SAG/OTH) (BR-KSA-08)",
					is.Func("identity must be one of: CRN/MOM/MLS/700/SAG/OTH", hasOneSupplierIdentity),
				),
			),
		),

		// Self-billed
		rules.When(
			is.Func("invoice is self-billed", invoiceIsSelfBilled),
			rules.Field("customer",
				rules.Field("tax_id",
					rules.Assert("31", "invoice customer must have a tax id when self-billed (BR-KSA-39)", is.Present),
					rules.Field("code",
						rules.Assert("32", "invoice customer must have a tax id code when self-billed (BR-KSA-39)", is.Present),
					),
				),
				rules.Field("identities",
					rules.Assert("33", "invoice customer must have a valid identity for self-billed invoices (CRN/MOM/MLS/700/SAG/OTH) (BR-KSA-08)",
						is.Func("identity must be one of: CRN/MOM/MLS/700/SAG/OTH", hasOneSupplierIdentity),
					),
				),
			),
		),

		// BT-111
		rules.Assert("34", "invoice currency must be SAR or include an exchange rate to convert to SAR",
			currency.CanConvertTo(currency.SAR),
		),
	)
}

func invoiceIsStandard(val any) bool {
	return !invoiceTypeOf(val).Simplified
}

func invoiceIsExport(val any) bool {
	return invoiceTypeOf(val).Export
}

func invoiceIsSelfBilled(val any) bool {
	return invoiceTypeOf(val).SelfBilled
}

func invoiceIsSimplifiedAndSummary(val any) bool {
	t := invoiceTypeOf(val)
	return t.Simplified && t.Summary
}

func invoiceIsSimplifiedAndEDUOrHEAExemption(val any) bool {
	return invoiceTypeOf(val).Simplified && invoiceHasEDUOrHEAExemption(val)
}

func invoiceHasEDUOrHEAExemption(val any) bool {
	// VATEX-SA-EDU (private education), VATEX-SA-HEA (private healthcare).
	return invoiceHasExemption(val, []cbc.Code{"VATEX-SA-EDU", "VATEX-SA-HEA"})
}

func hasOneSupplierIdentity(value any) bool {
	identities, _ := value.([]*org.Identity)
	return len(identities) == 1 && org.IdentitiesTypeIn(SupplierValidIdentities...).Check(identities)
}

func invoiceHasExemption(val any, exemptions []cbc.Code) bool {
	inv, ok := val.(*bill.Invoice)
	if !ok || inv == nil {
		return false
	}
	for _, line := range inv.Lines {
		vat := line.GetTaxes().Get(tax.CategoryVAT)
		if vat == nil {
			continue
		}
		code := vat.Ext.Get(cef.ExtKeyVATEX)
		if code.In(exemptions...) {
			return true
		}
	}
	return false
}

func customerValidIdentity(value any) bool {
	party, _ := value.(*org.Party)
	if party == nil {
		return false
	}
	if party.TaxID != nil && !party.TaxID.Code.IsEmpty() {
		return true
	}
	return len(party.Identities) == 1 && org.IdentitiesTypeIn(customerValidIdentities...).Check(party.Identities)
}
