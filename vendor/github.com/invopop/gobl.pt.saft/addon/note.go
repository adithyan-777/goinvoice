package addon

import (
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/regimes/pt"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
	"github.com/invopop/gobl/tax"
)

func orgNoteRules() *rules.Set {
	return rules.For(new(org.Note),
		rules.When(is.Func("legal exemption note", noteIsLegalExemption),
			rules.Field("text",
				rules.Assert("01", "the length must be between 6 and 60", is.Length(6, 60)),
			),
		),
	)
}

// noteIsLegalExemption checks if the note is a legal exemption note.
func noteIsLegalExemption(val any) bool {
	note, ok := val.(*org.Note)
	return ok && note != nil && note.Key == org.NoteKeyLegal && note.Src == ExtKeyExemption
}

func taxNoteRules() *rules.Set {
	return rules.For(new(tax.Note),
		rules.When(is.Func("withholding note", noteIsWithholding),
			rules.Field("text",
				rules.Assert("02", "the length must be at most 60", is.Length(0, 60)),
			),
		),
	)
}

func noteIsWithholding(val any) bool {
	note, ok := val.(*tax.Note)
	return ok && note != nil && note.Category.In(
		pt.TaxCategoryIRS,
		pt.TaxCategoryIRC,
	)
}
