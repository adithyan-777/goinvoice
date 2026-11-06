package zatca

import (
	"github.com/invopop/gobl.sa.zatca/catalogues/vatexsa"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cal"
	"github.com/invopop/gobl/catalogues/cef"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/tax"
)

func normalizeInvoice(inv *bill.Invoice) {
	// Ensure Tax object exists
	if inv.Tax == nil {
		inv.Tax = &bill.Tax{}
	}

	// Always set rounding to currency for SA ZATCA
	inv.Tax.Rounding = tax.RoundingRuleCurrency

	if inv.IssueTime == nil {
		// Ensure issue time exists. Calculate will set it to current time.
		inv.IssueTime = &cal.Time{}
	}

	normalizeInvoiceType(inv)
	normalizeTaxNotes(inv)
}

// normalizeInvoiceType derives the ZATCA invoice transaction type from the invoice tags.
func normalizeInvoiceType(inv *bill.Invoice) {
	if inv.Tax.GetExt(ExtKeyInvoiceType) != cbc.CodeEmpty {
		return
	}
	it := InvoiceType{
		Simplified: inv.HasTags(tax.TagSimplified),
		ThirdParty: inv.HasTags(TagThirdParty),
		Nominal:    inv.HasTags(TagNominal),
		Export:     inv.HasTags(tax.TagExport),
		Summary:    inv.HasTags(TagSummary),
		SelfBilled: inv.HasTags(tax.TagSelfBilled),
	}
	inv.Tax.Ext = inv.Tax.Ext.Set(ExtKeyInvoiceType, it.Code())
}

// BR-KSA-83
// VAT categories E,Z,O must have an associated tax note. This
// validation adds them if not previously provided by the user
func normalizeTaxNotes(inv *bill.Invoice) {
	vatex := tax.ExtensionForKey(vatexsa.ExtKeyVATEX)
	if vatex == nil {
		return
	}
	for _, line := range inv.Lines {
		vat := line.Taxes.Get(tax.CategoryVAT)
		if vat == nil {
			continue
		}
		ec := vat.Ext.Get(cef.ExtKeyVATEX)
		if ec == cbc.CodeEmpty {
			continue
		}
		def := vatex.CodeDef(ec)
		if def == nil {
			continue
		}
		untdidCat := vat.Ext.Get(untdid.ExtKeyTaxCategory)
		if untdidCat == cbc.CodeEmpty || hasTaxNoteForCategory(inv.Tax, untdidCat) {
			continue
		}
		inv.Tax = inv.Tax.MergeNotes(&tax.Note{
			Category: tax.CategoryVAT,
			Key:      vat.Key,
			Text:     def.Name.In(i18n.EN),
			Ext:      tax.ExtensionsOf(cbc.CodeMap{untdid.ExtKeyTaxCategory: untdidCat}),
		})
	}
}

func hasTaxNoteForCategory(bt *bill.Tax, untdidCat cbc.Code) bool {
	for _, n := range bt.Notes {
		if n == nil {
			continue
		}
		if n.Category == tax.CategoryVAT && n.Ext.Get(untdid.ExtKeyTaxCategory) == untdidCat {
			return true
		}
	}
	return false
}
