package addon

import (
	"fmt"

	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/regimes/br"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
	"github.com/invopop/gobl/tax"
)

// Situation codes that carry no tax value: the combo's percent must be zero.
var (
	icmsNoValueCSOSNCodes = []cbc.Code{"102", "103", "202", "203", "300", "400", "500"}
	icmsNoValueCSTCodes   = []cbc.Code{"40", "41", "50", "60"}
	pisCofinsNTCSTCodes   = []cbc.Code{"04", "05", "06", "07", "08", "09"}
)

// normalizeTaxCombo sets default situation codes on ICMS, PIS and COFINS combos.
func normalizeTaxCombo(tc *tax.Combo) {
	if tc == nil {
		return
	}
	switch tc.Category {
	case br.TaxCategoryICMS:
		if !tc.Ext.Has(ExtKeyICMSCSOSN) {
			tc.Ext = tc.Ext.SetIfEmpty(ExtKeyICMSCST, "00") // Taxed in full
		}
		tc.Ext = tc.Ext.SetIfEmpty(ExtKeyICMSOrigin, "0") // National
	case br.TaxCategoryPIS:
		tc.Ext = tc.Ext.SetIfEmpty(ExtKeyPISCST, "01") // Standard-rate taxable operation
	case br.TaxCategoryCOFINS:
		tc.Ext = tc.Ext.SetIfEmpty(ExtKeyCOFINSCST, "01") // Standard-rate taxable operation
	}
}

func taxComboRules() *rules.Set {
	return rules.For(new(tax.Combo),
		rules.When(
			is.Func("ICMS category", taxCategoryIs(br.TaxCategoryICMS)),
			rules.Field("ext",
				rules.Assert("01", fmt.Sprintf("ICMS tax combo requires '%s' or '%s' extension", ExtKeyICMSCST, ExtKeyICMSCSOSN),
					is.AnyOf(
						tax.ExtensionsRequire(ExtKeyICMSCST),
						tax.ExtensionsRequire(ExtKeyICMSCSOSN),
					),
				),
				rules.Assert("02", fmt.Sprintf("ICMS tax combo requires '%s' extension", ExtKeyICMSOrigin),
					tax.ExtensionsRequire(ExtKeyICMSOrigin),
				),
				rules.Assert("05", fmt.Sprintf("ICMS tax combo cannot include both '%s' and '%s' extensions", ExtKeyICMSCST, ExtKeyICMSCSOSN),
					tax.ExtensionsAllowOneOf(ExtKeyICMSCST, ExtKeyICMSCSOSN),
				),
				rules.Assert("10", fmt.Sprintf("'%s' extension, when set, must be a valid code", ExtKeyICMSCST),
					tax.ExtensionHasValidCode(ExtKeyICMSCST),
				),
				rules.Assert("11", fmt.Sprintf("'%s' extension, when set, must be a valid code", ExtKeyICMSCSOSN),
					tax.ExtensionHasValidCode(ExtKeyICMSCSOSN),
				),
				rules.Assert("12", fmt.Sprintf("'%s' extension, when set, must be a valid code", ExtKeyICMSOrigin),
					tax.ExtensionHasValidCode(ExtKeyICMSOrigin),
				),
			),
			rules.When(
				is.Func("no-value CSOSN code", comboExtCodeIn(ExtKeyICMSCSOSN, icmsNoValueCSOSNCodes...)),
				rules.Field("percent",
					rules.Assert("06", "ICMS percent must be zero for CSOSN codes 102, 103, 202, 203, 300, 400 and 500: they carry no ICMS value",
						num.Equals(num.PercentageZero),
					),
				),
			),
			rules.When(
				is.Func("no-value CST code", comboExtCodeIn(ExtKeyICMSCST, icmsNoValueCSTCodes...)),
				rules.Field("percent",
					rules.Assert("07", "ICMS percent must be zero for CST codes 40, 41, 50 and 60: they carry no ICMS value",
						num.Equals(num.PercentageZero),
					),
				),
			),
		),
		rules.When(
			is.Func("PIS category", taxCategoryIs(br.TaxCategoryPIS)),
			rules.Field("ext",
				rules.Assert("03", fmt.Sprintf("PIS tax combo requires '%s' extension", ExtKeyPISCST),
					tax.ExtensionsRequire(ExtKeyPISCST),
				),
				rules.Assert("13", fmt.Sprintf("'%s' extension, when set, must be a valid code", ExtKeyPISCST),
					tax.ExtensionHasValidCode(ExtKeyPISCST),
				),
			),
			rules.When(
				is.Func("non-taxed CST code", comboExtCodeIn(ExtKeyPISCST, pisCofinsNTCSTCodes...)),
				rules.Field("percent",
					rules.Assert("08", "PIS percent must be zero for CST codes 04 to 09: they carry no PIS value",
						num.Equals(num.PercentageZero),
					),
				),
			),
		),
		rules.When(
			is.Func("COFINS category", taxCategoryIs(br.TaxCategoryCOFINS)),
			rules.Field("ext",
				rules.Assert("04", fmt.Sprintf("COFINS tax combo requires '%s' extension", ExtKeyCOFINSCST),
					tax.ExtensionsRequire(ExtKeyCOFINSCST),
				),
				rules.Assert("14", fmt.Sprintf("'%s' extension, when set, must be a valid code", ExtKeyCOFINSCST),
					tax.ExtensionHasValidCode(ExtKeyCOFINSCST),
				),
			),
			rules.When(
				is.Func("non-taxed CST code", comboExtCodeIn(ExtKeyCOFINSCST, pisCofinsNTCSTCodes...)),
				rules.Field("percent",
					rules.Assert("09", "COFINS percent must be zero for CST codes 04 to 09: they carry no COFINS value",
						num.Equals(num.PercentageZero),
					),
				),
			),
		),
	)
}

// taxCategoryIs returns a tester that matches a tax combo of the given category.
func taxCategoryIs(cat cbc.Code) func(any) bool {
	return func(val any) bool {
		tc, ok := val.(*tax.Combo)
		return ok && tc != nil && tc.Category == cat
	}
}

// comboExtCodeIn returns a tester that matches a tax combo whose extension
// key holds one of the given codes.
func comboExtCodeIn(key cbc.Key, codes ...cbc.Code) func(any) bool {
	return func(val any) bool {
		tc, ok := val.(*tax.Combo)
		return ok && tc != nil && tc.Ext.Get(key).In(codes...)
	}
}
