package zatca

import (
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/i18n"
	"github.com/invopop/gobl/pkg/here"
)

const (
	// ExtKeyInvoiceTypeTransactions identifies the ZATCA invoice transaction type (KSA-2).
	ExtKeyInvoiceType cbc.Key = "sa-zatca-invoice-type"
)

var extensions = []*cbc.Definition{
	{
		Key: ExtKeyInvoiceType,
		Name: i18n.String{
			i18n.EN: "ZATCA Invoice Transaction Type",
			i18n.AR: "نوع الفاتورة",
		},
		Desc: i18n.String{
			i18n.EN: here.Doc(`
				Code used to describe the ZATCA invoice transaction type and flags (KSA-2).
				The code is a 7-character string where positions 1-2 indicate the main type
				(01 = Standard Tax Invoice, 02 = Simplified Tax Invoice) and positions 3-7
				are binary flags for third-party, nominal, export, summary, and self-billed
				transactions respectively.
			`),
		},
		Pattern: invTypeCodePattern,
		Values: []*cbc.Definition{
			{
				Code: "0100000",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice",
					i18n.AR: "فاتورة ضريبية",
				},
			},
			{
				Code: "0100001",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Self-billed",
					i18n.AR: "فاتورة ضريبية — ذاتية الإصدار",
				},
			},
			{
				Code: "0100010",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Summary",
					i18n.AR: "فاتورة ضريبية — ملخص",
				},
			},
			{
				Code: "0100011",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Summary, Self-billed",
					i18n.AR: "فاتورة ضريبية — ملخص، ذاتية الإصدار",
				},
			},
			{
				Code: "0100100",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Export",
					i18n.AR: "فاتورة ضريبية — تصدير",
				},
			},
			{
				Code: "0100110",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Export, Summary",
					i18n.AR: "فاتورة ضريبية — تصدير، ملخص",
				},
			},
			{
				Code: "0101000",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Nominal",
					i18n.AR: "فاتورة ضريبية — اسمية",
				},
			},
			{
				Code: "0101001",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Nominal, Self-billed",
					i18n.AR: "فاتورة ضريبية — اسمية، ذاتية الإصدار",
				},
			},
			{
				Code: "0101010",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Nominal, Summary",
					i18n.AR: "فاتورة ضريبية — اسمية، ملخص",
				},
			},
			{
				Code: "0101011",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Nominal, Summary, Self-billed",
					i18n.AR: "فاتورة ضريبية — اسمية، ملخص، ذاتية الإصدار",
				},
			},
			{
				Code: "0101100",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Nominal, Export",
					i18n.AR: "فاتورة ضريبية — اسمية، تصدير",
				},
			},
			{
				Code: "0101110",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Nominal, Export, Summary",
					i18n.AR: "فاتورة ضريبية — اسمية، تصدير، ملخص",
				},
			},
			{
				Code: "0110000",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Third-party",
					i18n.AR: "فاتورة ضريبية — طرف ثالث",
				},
			},
			{
				Code: "0110001",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Third-party, Self-billed",
					i18n.AR: "فاتورة ضريبية — طرف ثالث، ذاتية الإصدار",
				},
			},
			{
				Code: "0110010",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Third-party, Summary",
					i18n.AR: "فاتورة ضريبية — طرف ثالث، ملخص",
				},
			},
			{
				Code: "0110011",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Third-party, Summary, Self-billed",
					i18n.AR: "فاتورة ضريبية — طرف ثالث، ملخص، ذاتية الإصدار",
				},
			},
			{
				Code: "0110100",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Third-party, Export",
					i18n.AR: "فاتورة ضريبية — طرف ثالث، تصدير",
				},
			},
			{
				Code: "0110110",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Third-party, Export, Summary",
					i18n.AR: "فاتورة ضريبية — طرف ثالث، تصدير، ملخص",
				},
			},
			{
				Code: "0111000",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Third-party, Nominal",
					i18n.AR: "فاتورة ضريبية — طرف ثالث، اسمية",
				},
			},
			{
				Code: "0111001",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Third-party, Nominal, Self-billed",
					i18n.AR: "فاتورة ضريبية — طرف ثالث، اسمية، ذاتية الإصدار",
				},
			},
			{
				Code: "0111010",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Third-party, Nominal, Summary",
					i18n.AR: "فاتورة ضريبية — طرف ثالث، اسمية، ملخص",
				},
			},
			{
				Code: "0111011",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Third-party, Nominal, Summary, Self-billed",
					i18n.AR: "فاتورة ضريبية — طرف ثالث، اسمية، ملخص، ذاتية الإصدار",
				},
			},
			{
				Code: "0111100",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Third-party, Nominal, Export",
					i18n.AR: "فاتورة ضريبية — طرف ثالث، اسمية، تصدير",
				},
			},
			{
				Code: "0111110",
				Name: i18n.String{
					i18n.EN: "Standard Tax Invoice — Third-party, Nominal, Export, Summary",
					i18n.AR: "فاتورة ضريبية — طرف ثالث، اسمية، تصدير، ملخص",
				},
			},
			{
				Code: "0200000",
				Name: i18n.String{
					i18n.EN: "Simplified Tax Invoice",
					i18n.AR: "فاتورة ضريبية مبسطة",
				},
			},
			{
				Code: "0200010",
				Name: i18n.String{
					i18n.EN: "Simplified Tax Invoice — Summary",
					i18n.AR: "فاتورة ضريبية مبسطة — ملخص",
				},
			},
			{
				Code: "0201000",
				Name: i18n.String{
					i18n.EN: "Simplified Tax Invoice — Nominal",
					i18n.AR: "فاتورة ضريبية مبسطة — اسمية",
				},
			},
			{
				Code: "0201010",
				Name: i18n.String{
					i18n.EN: "Simplified Tax Invoice — Nominal, Summary",
					i18n.AR: "فاتورة ضريبية مبسطة — اسمية، ملخص",
				},
			},
			{
				Code: "0210000",
				Name: i18n.String{
					i18n.EN: "Simplified Tax Invoice — Third-party",
					i18n.AR: "فاتورة ضريبية مبسطة — طرف ثالث",
				},
			},
			{
				Code: "0210010",
				Name: i18n.String{
					i18n.EN: "Simplified Tax Invoice — Third-party, Summary",
					i18n.AR: "فاتورة ضريبية مبسطة — طرف ثالث، ملخص",
				},
			},
			{
				Code: "0211000",
				Name: i18n.String{
					i18n.EN: "Simplified Tax Invoice — Third-party, Nominal",
					i18n.AR: "فاتورة ضريبية مبسطة — طرف ثالث، اسمية",
				},
			},
			{
				Code: "0211010",
				Name: i18n.String{
					i18n.EN: "Simplified Tax Invoice — Third-party, Nominal, Summary",
					i18n.AR: "فاتورة ضريبية مبسطة — طرف ثالث، اسمية، ملخص",
				},
			},
		},
	},
}
