package addon

import (
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
)

// Party identity keys
const (
	IdentityKeyStateReg = "br-nfe-state-reg"
)

// Item identity keys
const (
	IdentityKeyTaxable = "taxable"
	IdentityKeyCEST    = "br-nfe-cest"
)

// Identity patterns
const (
	cestPattern = `^\d{7}$`
	ncmPattern  = `^(\d{2}|\d{8})$`
)

var identities = []*cbc.Definition{
	// Party identities
	{
		Key: IdentityKeyStateReg,
		Name: i18n.String{
			i18n.EN: "Company State Registration",
			i18n.PT: "Inscrição Estadual da Empresa",
		},
	},

	// Item identities
	{
		Key: org.IdentityKeyGTIN,
		Name: i18n.String{
			i18n.EN: "Product's GTIN (Global Trade Item Number)",
			i18n.PT: "GTIN (Global Trade Item Number) do produto",
		},
	},
	{
		Key: org.IdentityKeyGTIN.With(IdentityKeyTaxable),
		Name: i18n.String{
			i18n.EN: "Taxable item's GTIN (Global Trade Item Number)",
			i18n.PT: "GTIN (Global Trade Item Number) da unidade tributável",
		},
	},
	{
		Key: org.IdentityKeyNCM,
		Name: i18n.String{
			i18n.EN: "Product's NCM (Mercosur Common Nomenclature) code",
			i18n.PT: "Código NCM (Nomenclatura Comum do Mercosul) do produto",
		},
		Pattern: ncmPattern,
	},
	{
		Key: IdentityKeyCEST,
		Name: i18n.String{
			i18n.EN: "Product's CEST (Tax Substitution Specifier Code)",
			i18n.PT: "CEST (Código Especificador da Substituição Tributária) do produto",
		},
		Pattern: cestPattern,
	},
}

func identityRules() *rules.Set {
	return rules.For(new(org.Identity),
		rules.When(
			is.Func("CEST identity", identityKeyIs(IdentityKeyCEST)),
			rules.Field("code",
				rules.Assert("01", "CEST identity code must be a 7-digit number", is.Matches(cestPattern)),
			),
		),
		rules.When(
			is.Func("NCM identity", identityKeyIs(org.IdentityKeyNCM)),
			rules.Field("code",
				rules.Assert("02", "NCM identity code must be an 8-digit number (or 2 digits in exceptional cases), without separators", is.Matches(ncmPattern)),
			),
		),
	)
}

// identityKeyIs returns a tester that matches an identity with the given key.
func identityKeyIs(key cbc.Key) func(any) bool {
	return func(val any) bool {
		id, ok := val.(*org.Identity)
		return ok && id != nil && id.Key == key
	}
}
