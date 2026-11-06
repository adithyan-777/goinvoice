package addon

import (
	"fmt"

	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/tax"
)

func orgPartyRules() *rules.Set {
	return rules.For(new(org.Party),
		rules.Field("ext",
			rules.Assert("01", fmt.Sprintf("'%s' extension, when set, must be a valid code", ExtKeyRegime),
				tax.ExtensionHasValidCode(ExtKeyRegime),
			),
			rules.Assert("02", fmt.Sprintf("'%s' extension, when set, must be a valid code", ExtKeyFiscalIncentive),
				tax.ExtensionHasValidCode(ExtKeyFiscalIncentive),
			),
			rules.Assert("03", fmt.Sprintf("'%s' extension, when set, must be a valid code", ExtKeySpecialRegime),
				tax.ExtensionHasValidCode(ExtKeySpecialRegime),
			),
			rules.Assert("04", fmt.Sprintf("'%s' extension, when set, must be a valid code", ExtKeyStateRegInd),
				tax.ExtensionHasValidCode(ExtKeyStateRegInd),
			),
		),
	)
}
