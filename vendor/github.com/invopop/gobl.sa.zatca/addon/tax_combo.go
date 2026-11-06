package zatca

import (
	"github.com/invopop/gobl/addons/eu/en16931"
	"github.com/invopop/gobl/catalogues/cef"
	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
	"github.com/invopop/gobl/tax"
)

func taxComboRules() *rules.Set {
	return rules.For(new(tax.Combo),

		// ZATCA requires that UNTDID tax category Z carries a VATEX exemption
		// code, which conflicts with EN16931 BR-S-10/BR-Z-10.
		rules.Ignore("GOBL-EU-EN16931-TAX-COMBO-07"),

		// Extensions
		rules.Field("ext",
			rules.Assert("01", "invoice line tax cef-vatex extension code must be present and valid for Z/E/O categories, and must not be set for Standard (BR-KSA-CL-04)",
				is.Func("valid SA VAT exemption code", taxComboHasValidVATEX),
			),
			rules.Assert("02", "invoice line tax untdid-tax-category extension code must contain one of the values (S, Z, E, O) (BR-KSA-18)",
				tax.ExtensionsHasCodes(untdid.ExtKeyTaxCategory,
					en16931.TaxCategoryStandard,
					en16931.TaxCategoryZero,
					en16931.TaxCategoryExempt,
					en16931.TaxCategoryOutsideScope,
				),
			),
		),

		// Category
		rules.Field("cat",
			rules.Assert("03", "tax schema id must be 'VAT' (BR-KSA-54)", is.In(tax.CategoryVAT)),
		),
	)
}

func taxComboHasValidVATEX(val any) bool {
	ext, ok := val.(*tax.Extensions)
	if !ok || ext == nil {
		return false
	}
	category := ext.Get(untdid.ExtKeyTaxCategory)
	vatex := ext.Get(cef.ExtKeyVATEX)

	switch category {
	case en16931.TaxCategoryStandard:
		return vatex == cbc.CodeEmpty
	case en16931.TaxCategoryExempt:
		// Exempt from VAT.
		return vatex.In(
			"VATEX-SA-29",
			"VATEX-SA-29-7",
			"VATEX-SA-30",
		)
	case en16931.TaxCategoryZero:
		// Zero-rated.
		return vatex.In(
			"VATEX-SA-32",
			"VATEX-SA-33",
			"VATEX-SA-34-1",
			"VATEX-SA-34-2",
			"VATEX-SA-34-3",
			"VATEX-SA-34-4",
			"VATEX-SA-34-5",
			"VATEX-SA-35",
			"VATEX-SA-36",
			"VATEX-SA-EDU",
			"VATEX-SA-HEA",
			"VATEX-SA-MLTRY",
		)
	case en16931.TaxCategoryOutsideScope:
		return vatex == "VATEX-SA-OOS"
	default:
		return true
	}
}
