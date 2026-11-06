package addon

import (
	"slices"

	"github.com/invopop/gobl/catalogues/untdid"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/org"
	"github.com/invopop/gobl/rules"
	"github.com/invopop/gobl/rules/is"
	"github.com/invopop/gobl/tax"
)

// List of units typically used for services. Used to infer a default product type.
var serviceUnits = []cbc.Key{
	cbc.KeyEmpty,
	org.UnitKilometre,
	org.UnitWatt,
	org.UnitKilowatt,
	org.UnitKilowattHour,
	org.UnitRate,
	org.UnitMonth,
	org.UnitDay,
	org.UnitSecond,
	org.UnitHour,
	org.UnitMinute,
	org.UnitService,
	org.UnitJob,
	org.UnitActivity,
	org.UnitTrip,

	// Ambivalent units but we default to service
	org.UnitItem,
	org.UnitOne,
}

func orgItemRules() *rules.Set {
	return rules.For(new(org.Item),
		rules.Field("unit",
			rules.Assert("01", "cannot be blank", is.Present),
		),
		rules.Field("ext",
			rules.Assert("02", "product type is required",
				tax.ExtensionsRequire(ExtKeyProductType),
			),
		),
	)
}

func normalizeItem(item *org.Item) {
	setDefaultUnit(item)
	setDefaultProductType(item)
}

func setDefaultProductType(item *org.Item) {
	if item == nil {
		return
	}

	if !item.Ext.Has(ExtKeyProductType) {
		if slices.Contains(serviceUnits, item.Unit) {
			item.Ext = item.Ext.Set(ExtKeyProductType, ProductTypeService)
		} else {
			item.Ext = item.Ext.Set(ExtKeyProductType, ProductTypeGoods)
		}
	}
}

// setDefaultUnit ensures every item carries a unit, which the AT requires as
// the line's `UnitOfMeasure`. Core normalization moves raw UN/ECE codes out of
// the unit field and into the `untdid-unit` extension, so recover the GOBL key
// from there before falling back to the generic unit. Without this a document
// using, say, `KGM` would silently end up as `one`, and be classified as a
// service instead of goods.
func setDefaultUnit(item *org.Item) {
	if item == nil {
		return
	}

	if item.Unit != cbc.KeyEmpty {
		return
	}

	if unit := untdid.UnitKey(item.Ext.Get(untdid.ExtKeyUnit)); unit != cbc.KeyEmpty {
		item.Unit = unit
		return
	}

	item.Unit = org.UnitOne
}
