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
// forms expose, plus repeatable identity and logo rows.
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
	Identities []PartyIdentityForm
	Logos      []PartyLogoForm
	Err        string
}

// PartyIdentityForm is one extra code (GST, registration number, DUNS, …).
type PartyIdentityForm struct {
	Type        string
	Code        string
	Description string
}

// PartyLogoForm is one logo image row.
type PartyLogoForm struct {
	Label string
	URL   string
	Alt   string
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
	for _, idn := range p.Identities {
		if idn == nil {
			continue
		}
		typ := idn.Key.String()
		if typ == "" {
			typ = idn.Type.String()
		}
		f.Identities = append(f.Identities, PartyIdentityForm{
			Type:        typ,
			Code:        idn.Code.String(),
			Description: idn.Description,
		})
	}
	if len(f.Identities) == 0 {
		f.Identities = []PartyIdentityForm{{}}
	}
	for _, img := range p.Logos {
		if img == nil {
			continue
		}
		f.Logos = append(f.Logos, PartyLogoForm{Label: img.Label, URL: img.URL, Alt: img.Alt})
	}
	if len(f.Logos) == 0 {
		f.Logos = []PartyLogoForm{{}}
	}
	return f
}
