package addon

import (
	"regexp"

	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
)

const portugalPostCodePattern = `^[0-9]{4}-[0-9]{3}$`

var isPortuguesePostCode = is.MatchesRegexp(regexp.MustCompile(portugalPostCodePattern))

func portuguesePostCodeRule(id rules.Code, desc string) rules.Def {
	return rules.Field("addresses",
		rules.Each(
			rules.When(is.Func("Portuguese address", addressIsPortuguese),
				rules.Field("code",
					rules.AssertIfPresent(id, desc, isPortuguesePostCode),
				),
			),
		),
	)
}

func addressIsPortuguese(val any) bool {
	a, ok := val.(*org.Address)
	if !ok || a == nil {
		return false
	}
	return a.Country == "" || a.Country.In("PT")
}
