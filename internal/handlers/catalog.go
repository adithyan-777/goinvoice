package handlers

import (
	"database/sql"

	"github.com/invopop/gobl/org"
)

// nsStr renders a nullable column, empty when NULL.
func nsStr(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}

// CustomerFormData carries the compact "new customer" modal form values.
type CustomerFormData struct {
	For        string // document route to return to ("orders", "invoices", …)
	Name       string
	TaxCountry string
	TaxCode    string
	Email      string
	Err        string
}

// PartyFormData is the flat view model shared by the customer and supplier
// create/edit modals. It maps 1:1 to the scalar org.Party fields the HTML
// forms expose.
type PartyFormData struct {
	ID         string
	Label      string
	Name       string
	Alias      string
	TaxCountry string
	TaxCode    string
	TaxScheme  string
	TaxType    string
	Email      string
	Phone      string
	Website    string
	Address    string
	Locality   string
	PostCode   string
	Country    string
	Err        string
}

// partyFormData projects a GOBL party onto the flat form view model.
func partyFormData(p *org.Party, id, errMsg string) PartyFormData {
	if p == nil {
		p = &org.Party{}
	}
	f := PartyFormData{ID: id, Label: p.Label, Name: p.Name, Alias: p.Alias, Err: errMsg}
	if p.TaxID != nil {
		f.TaxCountry = p.TaxID.Country.String()
		f.TaxCode = p.TaxID.Code.String()
		f.TaxScheme = p.TaxID.Scheme.String()
		f.TaxType = p.TaxID.Type.String()
	}
	if len(p.Emails) > 0 && p.Emails[0] != nil {
		f.Email = p.Emails[0].Address
	}
	if len(p.Telephones) > 0 && p.Telephones[0] != nil {
		f.Phone = p.Telephones[0].Number
	}
	if len(p.Websites) > 0 && p.Websites[0] != nil {
		f.Website = p.Websites[0].URL
	}
	if len(p.Addresses) > 0 && p.Addresses[0] != nil {
		a := p.Addresses[0]
		f.Address = a.LineOne()
		f.Locality = a.Locality
		f.PostCode = a.Code.String()
		f.Country = a.Country.String()
	}
	return f
}
