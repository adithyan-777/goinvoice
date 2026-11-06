package addon

import (
	"fmt"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
	"github.com/invopop/gobl/tax"
)

var paymentTags = &tax.TagSet{
	Schema: bill.ShortSchemaPayment,
	List:   []*cbc.Definition{cashVATTag},
}

func billPaymentRules() *rules.Set {
	return rules.For(new(bill.Payment),
		rules.Assert("01", "series format must be valid",
			is.FuncError("series format", paymentSeriesFormatValid),
		),
		rules.Assert("02", "code format must be valid",
			is.FuncError("code format", paymentCodeFormatValid),
		),
		rules.Field("ext",
			rules.Assert("03",
				fmt.Sprintf("'%s' extension is required", ExtKeyPaymentType),
				tax.ExtensionsRequire(ExtKeyPaymentType),
			),
			rules.Assert("04",
				fmt.Sprintf("'%s' extension is required", ExtKeySource),
				tax.ExtensionsRequire(ExtKeySource),
			),
		),
		rules.When(is.Func("source not produced", paymentSourceNotProduced),
			rules.Field("ext",
				rules.Assert("05",
					fmt.Sprintf("'%s' extension is required when source is not produced", ExtKeySourceRef),
					tax.ExtensionsRequire(ExtKeySourceRef),
				),
			),
		),
		rules.Assert("06", "source ref format is invalid",
			is.FuncError("source ref format", paymentSourceRefValid),
		),
		rules.Field("supplier",
			rules.Field("tax_id",
				rules.Assert("07", "supplier tax ID is required", is.Present),
				rules.Field("code",
					rules.Assert("08", "supplier tax ID code is required", is.Present),
				),
			),
		),
		rules.Assert("09", "customer name is required when customer has tax ID code",
			is.Func("customer name present", paymentCustomerNamePresent),
		),
		rules.Field("total",
			rules.Assert("10", "must be no less than 0", num.ZeroOrPositive),
		),
		rules.Assert("11", "payment method dates must match the issue date",
			is.FuncError("method dates", paymentMethodDatesMatchIssueDate),
		),
	)
}

func paymentSeriesFormatValid(val any) error {
	pmt, ok := val.(*bill.Payment)
	if !ok || pmt == nil {
		return nil
	}
	return validateSeriesFormat(paymentDocType(pmt)).Validate(pmt.Series)
}

func paymentCodeFormatValid(val any) error {
	pmt, ok := val.(*bill.Payment)
	if !ok || pmt == nil {
		return nil
	}
	dt := paymentDocType(pmt)
	return validateCodeFormat(pmt.Series, dt).Validate(pmt.Code)
}

func paymentSourceNotProduced(val any) bool {
	pmt, ok := val.(*bill.Payment)
	if !ok || pmt == nil {
		return false
	}
	return !pmt.Ext.IsZero() && pmt.Ext.Get(ExtKeySource) != "" && pmt.Ext.Get(ExtKeySource) != SourceBillingProduced
}

func paymentSourceRefValid(val any) error {
	pmt, ok := val.(*bill.Payment)
	if !ok || pmt == nil {
		return nil
	}
	return validateSourceRef(paymentDocType(pmt), pmt.Ext)
}

func paymentCustomerNamePresent(val any) bool {
	pmt, ok := val.(*bill.Payment)
	if !ok || pmt == nil || pmt.Customer == nil {
		return true
	}
	if pmt.Customer.TaxID == nil || pmt.Customer.TaxID.Code == cbc.CodeEmpty {
		return true
	}
	return pmt.Customer.Name != ""
}

// paymentMethodDatesMatchIssueDate asserts that any payment method with an
// explicit date matches the payment's issue date, as required by DL 71/2013
// Art. 6(3). Methods without a date are skipped.
func paymentMethodDatesMatchIssueDate(val any) error {
	pmt, ok := val.(*bill.Payment)
	if !ok || pmt == nil {
		return nil
	}
	for _, m := range pmt.Methods {
		if m == nil || m.Date == nil {
			continue
		}
		if *m.Date != pmt.IssueDate {
			return fmt.Errorf("method date '%s' must match the issue date '%s'", m.Date.String(), pmt.IssueDate.String())
		}
	}
	return nil
}

func paymentDocType(pmt *bill.Payment) cbc.Code {
	if pmt.Ext.IsZero() {
		return cbc.CodeEmpty
	}
	return pmt.Ext.Get(ExtKeyPaymentType)
}

func normalizePayment(pmt *bill.Payment) {
	if pmt.Ext.IsZero() {
		pmt.Ext = tax.ExtensionsOf(cbc.CodeMap{})
	}

	// TODO: This could be done with scenarios when supported
	if pmt.HasTags(TagCashVAT) {
		pmt.Ext = pmt.Ext.Set(ExtKeyPaymentType, PaymentTypeCash)
	} else {
		pmt.Ext = pmt.Ext.Set(ExtKeyPaymentType, PaymentTypeOther)
	}

	if !pmt.Ext.Has(ExtKeySource) {
		pmt.Ext = pmt.Ext.Set(ExtKeySource, SourceBillingProduced)
	}
}
