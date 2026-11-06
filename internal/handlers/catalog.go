package handlers

import "database/sql"

// nsStr renders a nullable column, empty when NULL.
func nsStr(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}
