package handlers

import "database/sql"

// nsStr renders a nullable column, empty when NULL.
func nsStr(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}

// CustomerFormData carries the "new customer" modal form values.
type CustomerFormData struct {
	For        string // document route to return to ("orders", "invoices", …)
	Name       string
	TaxCountry string
	TaxCode    string
	Email      string
	Err        string
}
