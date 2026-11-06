package invoicer

import "math/big"

// InvoiceStats aggregates list-level totals for dashboard display.
// Amounts are summed exactly with big.Rat; GOBL-calculated per-invoice
// totals remain the legal source of truth.
type InvoiceStats struct {
	Count    int
	Currency string
	Payable  string
	Tax      string
}

// SummarizeStats folds a list of summaries into dashboard totals.
// Mixed currencies are summed numerically under the first currency seen;
// the demo dataset is single-currency.
func SummarizeStats(sums []*Summary) InvoiceStats {
	st := InvoiceStats{}
	pay, tax := new(big.Rat), new(big.Rat)
	for _, s := range sums {
		st.Count++
		if st.Currency == "" {
			st.Currency = s.Currency
		}
		if r, ok := new(big.Rat).SetString(s.Payable); ok {
			pay.Add(pay, r)
		}
		if r, ok := new(big.Rat).SetString(s.Tax); ok {
			tax.Add(tax, r)
		}
	}
	st.Payable = pay.FloatString(2)
	st.Tax = tax.FloatString(2)
	return st
}
