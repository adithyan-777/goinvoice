package addon

import (
	"fmt"

	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/currency"
	"github.com/invopop/gobl/l10n"
	"github.com/invopop/gobl/num"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/regimes/br"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
	"github.com/invopop/gobl/tax"
)

// Validation patterns
const (
	seriesPattern = `^(?:0|[1-9]{1}[0-9]{0,2})$` // extracted from the NFe XSD to validate the series
)

var (
	// Supplier regimes that issue under Simples Nacional rules (CSOSN codes).
	regimeSimplesCodes = []cbc.Code{"1", "4"}
	// Supplier regimes that issue under normal ICMS rules (CST codes).
	regimeNormalCodes = []cbc.Code{"2", "3"}
	// Fuel monophase CSTs permitted for Simples Nacional issuers
	icmsFuelCSTCodes = []cbc.Code{"02", "15", "53", "61"}
)

// normalizeInvoice applies NF-e invoice-level defaults.
func normalizeInvoice(inv *bill.Invoice) {
	if inv == nil {
		return
	}

	// The model is set first so that the rest of the normalization can rely on it.
	normalizeInvoiceModel(inv)

	if inv.Supplier != nil {
		inv.Supplier.Ext = inv.Supplier.Ext.SetIfEmpty(ExtKeyRegime, "3") // Normal regime
	}
	if inv.Customer != nil {
		inv.Customer.Ext = inv.Customer.Ext.SetIfEmpty(ExtKeyStateRegInd, defaultStateRegInd(inv))
	}
}

// normalizeInvoiceModel sets the fiscal document model from the invoice tags: NFC-e (65)
// for simplified invoices, NF-e (55) otherwise.
//
// This would normally be defined as a tax scenario, but GOBL applies scenarios after
// normalization and other normalizers depend on the model. It is therefore implemented as
// a normalizer that runs before the rest of the invoice normalization.
func normalizeInvoiceModel(inv *bill.Invoice) {
	model := ModelNFe
	if inv.HasTags(tax.TagSimplified) {
		model = ModelNFCe
	}
	if inv.Tax == nil {
		inv.Tax = new(bill.Tax)
	}
	inv.Tax.Ext = inv.Tax.Ext.Set(ExtKeyModel, model)
}

// defaultStateRegInd determines the default state registration indicator of the invoice's
// customer.
func defaultStateRegInd(inv *bill.Invoice) cbc.Code {
	if invoiceIsNFe(inv) && partyHasStateReg(inv.Customer) {
		return StateRegIndTaxpayer
	}
	return StateRegIndNonTaxpayer
}

func billInvoiceRules() *rules.Set {
	return rules.For(new(bill.Invoice),
		rules.Assert("34", "invoice currency must be BRL or provide exchange rate for conversion", currency.CanConvertTo(currency.BRL)),
		// Supplier validation
		rules.Field("supplier",
			rules.Field("tax_id",
				rules.Assert("01", "invoice supplier tax ID is required", is.Present),
				rules.Field("code",
					rules.Assert("02", "invoice supplier tax ID code is required", is.Present),
				),
			),
			rules.Field("addresses",
				rules.Each(
					rules.Assert("03", "invoice supplier address must not be empty", is.Present),
					rules.Field("street",
						rules.Assert("04", "invoice supplier address requires a street", is.Present),
					),
					rules.Field("num",
						rules.Assert("05", "invoice supplier address requires a number", is.Present),
					),
					rules.Field("locality",
						rules.Assert("06", "invoice supplier address requires a locality", is.Present),
					),
					rules.Field("state",
						rules.Assert("07", "invoice supplier address requires a state", is.Present),
					),
					rules.Field("code",
						rules.Assert("08", "invoice supplier address requires a postal code", is.Present),
					),
					rules.Field("country",
						rules.Assert("37", "invoice supplier address requires a country", is.Present),
					),
				),
			),
			rules.When(
				is.Func("has addresses", partyHasAddresses),
				rules.Field("ext",
					rules.Assert("09", fmt.Sprintf("invoice supplier requires '%s' extension when addresses are present", br.ExtKeyMunicipality),
						tax.ExtensionsRequire(br.ExtKeyMunicipality),
					),
				),
			),
			rules.Field("ext",
				rules.Assert("39", fmt.Sprintf("invoice supplier requires '%s' extension", ExtKeyRegime),
					tax.ExtensionsRequire(ExtKeyRegime),
				),
			),
			rules.Field("name",
				rules.Assert("10", "invoice supplier name is required", is.Present),
			),
			rules.Field("addresses",
				rules.Assert("11", "invoice supplier must have at least one address", is.Present),
			),
		),
		// Customer: required for NF-e model
		rules.When(
			is.Func("invoice is NFe", invoiceIsNFe),
			rules.Field("customer",
				rules.Assert("12", "invoice customer is required for NF-e invoices", is.Present),
				rules.Field("addresses",
					rules.Assert("13", "invoice customer must have at least one address for NF-e invoices", is.Present),
				),
			),
		),
		// Customer: general party validation when present
		rules.Field("customer",
			rules.When(
				is.Func("no tax ID", hasNoTaxID),
				rules.Assert("36", "invoice customer must have a tax ID or a foreign country identity",
					is.Func("has foreign country identity", hasForeignCountryIdentity),
				),
			),
			rules.Field("tax_id",
				rules.Field("code",
					rules.Assert("15", "invoice customer tax ID code is required", is.Present),
				),
			),
			rules.Field("addresses",
				rules.Each(
					rules.Assert("16", "invoice customer address must not be empty", is.Present),
					rules.Field("street",
						rules.Assert("17", "invoice customer address requires a street", is.Present),
					),
					rules.Field("num",
						rules.Assert("18", "invoice customer address requires a number", is.Present),
					),
					rules.Field("locality",
						rules.Assert("19", "invoice customer address requires a locality", is.Present),
					),
					rules.Field("code",
						rules.Assert("21", "invoice customer address requires a postal code", is.Present),
					),
					rules.Field("country",
						rules.Assert("38", "invoice customer address requires a country", is.Present),
					),
				),
			),
			rules.When(
				is.Func("is Brazilian", partyIsBrazilian),
				rules.Field("addresses",
					rules.Each(
						rules.Field("state",
							rules.Assert("20", "invoice customer address requires a state", is.Present),
						),
					),
				),
				rules.When(
					is.Func("has addresses", partyHasAddresses),
					rules.Field("ext",
						rules.Assert("22", fmt.Sprintf("invoice customer requires '%s' extension when addresses are present", br.ExtKeyMunicipality),
							tax.ExtensionsRequire(br.ExtKeyMunicipality),
						),
					),
				),
			),
		),
		// Customer: state registration indicator (MOC E16a)
		rules.Field("customer",
			rules.Field("ext",
				rules.Assert("50", fmt.Sprintf("invoice customer requires '%s' extension", ExtKeyStateRegInd),
					tax.ExtensionsRequire(ExtKeyStateRegInd),
				),
			),
			rules.When(
				is.Func("ICMS taxpayer", partyStateRegIndIs(StateRegIndTaxpayer)),
				rules.Assert("51", fmt.Sprintf("invoice customer requires '%s' identity when '%s' is '%s'", IdentityKeyStateReg, ExtKeyStateRegInd, StateRegIndTaxpayer),
					is.Func("has state registration", partyHasStateReg),
				),
			),
			rules.When(
				is.Func("ICMS taxpayer exempt", partyStateRegIndIs(StateRegIndExempt)),
				rules.Assert("52", fmt.Sprintf("invoice customer must not have '%s' identity when '%s' is '%s'", IdentityKeyStateReg, ExtKeyStateRegInd, StateRegIndExempt),
					is.Func("has no state registration", partyHasNoStateReg),
				),
			),
		),
		rules.When(
			is.Func("invoice is NFCe", invoiceIsNFCe),
			rules.Field("customer",
				rules.Field("ext",
					rules.Assert("53", fmt.Sprintf("NFC-e invoices require '%s' for '%s'", StateRegIndNonTaxpayer, ExtKeyStateRegInd),
						tax.ExtensionsHasCodes(ExtKeyStateRegInd, StateRegIndNonTaxpayer),
					),
				),
			),
		),
		// Series
		rules.Field("series",
			rules.Assert("23", "invoice series is required", is.Present),
			rules.Assert("24", "invoice series format is invalid; must be 0 or 1-999", is.Matches(seriesPattern)),
		),
		// Tax
		rules.Field("tax",
			rules.Assert("25", "invoice tax is required", is.Present),
			rules.Field("ext",
				rules.Assert("26", fmt.Sprintf("invoice tax requires '%s', '%s', '%s' and '%s' extensions", ExtKeyModel, ExtKeyPresence, ExtKeyPurpose, ExtKeyOperationType),
					tax.ExtensionsRequire(ExtKeyModel, ExtKeyPresence, ExtKeyPurpose, ExtKeyOperationType),
				),
				rules.Assert("44", fmt.Sprintf("'%s' extension, when set, must be a valid code", ExtKeyModel),
					tax.ExtensionHasValidCode(ExtKeyModel),
				),
				rules.Assert("45", fmt.Sprintf("'%s' extension, when set, must be a valid code", ExtKeyPresence),
					tax.ExtensionHasValidCode(ExtKeyPresence),
				),
				rules.Assert("46", fmt.Sprintf("'%s' extension, when set, must be a valid code", ExtKeyPurpose),
					tax.ExtensionHasValidCode(ExtKeyPurpose),
				),
				rules.Assert("47", fmt.Sprintf("'%s' extension, when set, must be a valid code", ExtKeyOperationType),
					tax.ExtensionHasValidCode(ExtKeyOperationType),
				),
				rules.When(
					is.Func("NFe model", modelIsNFe),
					rules.Assert("27", fmt.Sprintf("NF-e invoices do not support '%s' for '%s'", PresenceDelivery, ExtKeyPresence),
						tax.ExtensionsExcludeCodes(ExtKeyPresence, PresenceDelivery),
					),
				),
				rules.When(
					is.Func("NFCe model", modelIsNFCe),
					rules.Assert("28", fmt.Sprintf("NFC-e invoices require in-person or delivery for '%s'", ExtKeyPresence),
						tax.ExtensionsHasCodes(ExtKeyPresence, PresenceInPerson, PresenceDelivery),
					),
				),
				rules.When(
					is.Func("credit note purpose", purposeIsCreditNote),
					rules.Assert("40", fmt.Sprintf("credit note invoices require '%s' extension", ExtKeyCreditNoteType),
						tax.ExtensionsRequire(ExtKeyCreditNoteType),
					),
					rules.Assert("48", fmt.Sprintf("'%s' extension, when set, must be a valid code", ExtKeyCreditNoteType),
						tax.ExtensionHasValidCode(ExtKeyCreditNoteType),
					),
				),
				rules.When(
					is.Func("debit note purpose", purposeIsDebitNote),
					rules.Assert("41", fmt.Sprintf("debit note invoices require '%s' extension", ExtKeyDebitNoteType),
						tax.ExtensionsRequire(ExtKeyDebitNoteType),
					),
					rules.Assert("49", fmt.Sprintf("'%s' extension, when set, must be a valid code", ExtKeyDebitNoteType),
						tax.ExtensionHasValidCode(ExtKeyDebitNoteType),
					),
				),
			),
		),
		// Notes
		rules.Field("notes",
			rules.Each(
				rules.When(
					is.Func("reason note", isReasonNote),
					rules.Field("text",
						rules.Assert("29", "invoice reason note text must be between 1 and 60 characters", is.Length(1, 60)),
					),
				),
			),
		),
		rules.Assert("30", "invoice requires a note with key 'reason' to describe the nature of the operation (natOp)",
			is.Func("has reason note", invoiceHasReasonNote),
		),
		// Payment: required when unpaid
		rules.When(
			is.Func("invoice not paid", invoiceNotPaid),
			rules.Field("payment",
				rules.Assert("31", "invoice payment is required when invoice is unpaid", is.Present),
				rules.Field("instructions",
					rules.Assert("32", "invoice payment instructions are required when invoice is unpaid", is.Present),
				),
			),
		),
		// Totals
		rules.Field("totals",
			rules.Field("due",
				rules.Assert("33", "invoice due amount must not be negative",
					is.Func("zero or positive", amountZeroOrPositive),
				),
			),
		),
		// Lines
		rules.Field("lines",
			rules.Each(
				rules.Field("ext",
					rules.Assert("35", fmt.Sprintf("invoice lines require '%s' extension", ExtKeyCFOP),
						tax.ExtensionsRequire(ExtKeyCFOP),
					),
				),
			),
		),
		// Rules specific to suppliers under normal regime
		rules.When(
			is.Func("supplier under normal regime", supplierInNormalRegime),
			eachLineTax(
				rules.When(
					is.Func("ICMS category", taxCategoryIs(br.TaxCategoryICMS)),
					rules.Field("ext",
						rules.Assert("42", fmt.Sprintf("only Simples Nacional issuers can use '%s'; others must use '%s'", ExtKeyICMSCSOSN, ExtKeyICMSCST),
							tax.ExtensionsExclude(ExtKeyICMSCSOSN),
						),
					),
				),
			),
		),
		// Rules specific to suppliers under Simples Nacional
		rules.When(
			is.Func("supplier under Simples Nacional", supplierInSimplesNacional),
			eachLineTax(
				rules.When(
					is.Func("ICMS category", taxCategoryIs(br.TaxCategoryICMS)),
					rules.Field("ext",
						rules.Assert("43", fmt.Sprintf("Simples Nacional issuers (regime 1 or 4) must use '%s'; '%s' is only allowed for fuel codes 02, 15, 53 and 61", ExtKeyICMSCSOSN, ExtKeyICMSCST),
							is.AnyOf(
								tax.ExtensionsExclude(ExtKeyICMSCST),
								tax.ExtensionsHasCodes(ExtKeyICMSCST, icmsFuelCSTCodes...),
							),
						),
					),
				),
			),
		),
	)
}

// invoiceIsNFe checks if the invoice's tax model is NF-e.
func invoiceIsNFe(val any) bool {
	inv, ok := val.(*bill.Invoice)
	return ok && inv != nil && inv.Tax != nil && modelIsNFe(inv.Tax.Ext)
}

// invoiceIsNFCe checks if the invoice's tax model is NFC-e.
func invoiceIsNFCe(val any) bool {
	inv, ok := val.(*bill.Invoice)
	return ok && inv != nil && inv.Tax != nil && modelIsNFCe(inv.Tax.Ext)
}

// partyStateRegIndIs returns a tester that matches a party whose state registration
// indicator extension has the given code.
func partyStateRegIndIs(code cbc.Code) func(any) bool {
	return func(val any) bool {
		p, ok := val.(*org.Party)
		return ok && p != nil && p.Ext.Get(ExtKeyStateRegInd) == code
	}
}

// partyHasStateReg checks if the party has a state registration (IE) identity.
func partyHasStateReg(val any) bool {
	p, ok := val.(*org.Party)
	return ok && p != nil && org.IdentityForKey(p.Identities, IdentityKeyStateReg) != nil
}

// partyHasNoStateReg checks if the party has no state registration (IE) identity.
func partyHasNoStateReg(val any) bool {
	return !partyHasStateReg(val)
}

// supplierInSimplesNacional checks if the invoice's supplier is under the Simples
// Nacional (1) or MEI (4) tax regimes.
func supplierInSimplesNacional(val any) bool {
	inv, ok := val.(*bill.Invoice)
	return ok && inv != nil && inv.Supplier != nil &&
		inv.Supplier.Ext.Get(ExtKeyRegime).In(regimeSimplesCodes...)
}

// supplierInNormalRegime checks if the invoice's supplier is under a regime following the
// normal ICMS rules (2 or 3).
func supplierInNormalRegime(val any) bool {
	inv, ok := val.(*bill.Invoice)
	return ok && inv != nil && inv.Supplier != nil &&
		inv.Supplier.Ext.Get(ExtKeyRegime).In(regimeNormalCodes...)
}

// modelIsNFe checks if the tax extensions indicate NF-e model.
func modelIsNFe(val any) bool {
	ext, ok := tax.ExtensionsFromValue(val)
	return ok && ext.Get(ExtKeyModel) == ModelNFe
}

// modelIsNFCe checks if the tax extensions indicate NFC-e model.
func modelIsNFCe(val any) bool {
	ext, ok := tax.ExtensionsFromValue(val)
	return ok && ext.Get(ExtKeyModel) == ModelNFCe
}

// purposeIsCreditNote checks if the tax extensions indicate a credit note purpose.
func purposeIsCreditNote(val any) bool {
	ext, ok := tax.ExtensionsFromValue(val)
	return ok && ext.Get(ExtKeyPurpose) == PurposeCreditNote
}

// purposeIsDebitNote checks if the tax extensions indicate a debit note purpose.
func purposeIsDebitNote(val any) bool {
	ext, ok := tax.ExtensionsFromValue(val)
	return ok && ext.Get(ExtKeyPurpose) == PurposeDebitNote
}

// isReasonNote checks if a note has the reason key.
func isReasonNote(val any) bool {
	note, ok := val.(*org.Note)
	return ok && note != nil && note.Key == org.NoteKeyReason
}

// invoiceHasReasonNote checks if the invoice has at least one note with the reason key.
func invoiceHasReasonNote(val any) bool {
	inv, ok := val.(*bill.Invoice)
	if !ok || inv == nil {
		return true
	}
	for _, n := range inv.Notes {
		if n != nil && n.Key == org.NoteKeyReason {
			return true
		}
	}
	return false
}

// invoiceNotPaid checks if the invoice totals indicate the invoice is not paid.
func invoiceNotPaid(val any) bool {
	inv, ok := val.(*bill.Invoice)
	if !ok || inv == nil {
		return false
	}
	return !inv.Totals.Paid()
}

// amountZeroOrPositive checks that a num.Amount pointer is nil, zero, or positive.
func amountZeroOrPositive(val any) bool {
	amt, ok := val.(*num.Amount)
	if !ok || amt == nil {
		return true
	}
	return !amt.IsNegative()
}

func partyHasAddresses(val any) bool {
	p, ok := val.(*org.Party)
	return ok && p != nil && len(p.Addresses) > 0
}

func partyIsBrazilian(val any) bool {
	p, ok := val.(*org.Party)
	return ok && p != nil && p.TaxID != nil && p.TaxID.Country == l10n.BR.Tax()
}

func hasNoTaxID(val any) bool {
	p, ok := val.(*org.Party)
	return ok && p != nil && p.TaxID == nil
}

func hasForeignCountryIdentity(val any) bool {
	p, ok := val.(*org.Party)
	if !ok || p == nil {
		return false
	}
	for _, id := range p.Identities {
		if id != nil && !id.Country.Empty() && id.Country != l10n.BR.ISO() {
			return true
		}
	}
	return false
}

func eachLineTax(defs ...rules.Def) rules.Def {
	return rules.Field("lines",
		rules.Each(
			rules.Field("taxes",
				rules.Each(defs...),
			),
		),
	)
}
