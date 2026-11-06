// Package vatexsa provides the Saudi Arabia VAT exemption reason codes
// (VATEX-SA) defined by ZATCA as the KSA extension of the EU VATEX code list.
package vatexsa

import (
	"encoding/json"
	"path"

	"github.com/invopop/gobl.sa.zatca/data"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/tax"
)

func init() {
	registerCatalogueDef("vatex_sa.json")
}

const (
	// ExtKeyVATEX is used for the ZATCA VATEX-SA exemption codes.
	ExtKeyVATEX cbc.Key = "vatex-sa"
)

// registerCatalogueDef mirrors gobl's tax.RegisterCatalogueDef for catalogues embedded
// in this module, which gobl cannot load from its own data filesystem.
func registerCatalogueDef(filename string) {
	catalogue := &tax.CatalogueDef{}
	out, err := data.Content.ReadFile(path.Join("catalogues", filename))
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(out, catalogue); err != nil {
		panic(err)
	}
	for _, ext := range catalogue.Extensions {
		tax.RegisterExtension(ext)
	}
}
