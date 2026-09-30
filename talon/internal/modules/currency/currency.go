package currency

import (
	"github.com/ajepe/talon/internal/metadata"
)

func init() {
	metadata.RegisterModule("currency", &metadata.Module{
		Name:    "currency",
		Version: "0.1.0",
		Depends: []string{"ir", "res"},
	})

	metadata.RegisterModel("currency.currency", &metadata.Model{
		Name:  "currency.currency",
		Table: "currency_currency",
		Fields: map[string]*metadata.Field{
			"name":         {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"symbol":       {Name: "symbol", Type: metadata.FieldChar, Label: "Symbol"},
			"code":         {Name: "code", Type: metadata.FieldChar, Label: "Code", Required: true},
			"numeric_code": {Name: "numeric_code", Type: metadata.FieldChar, Label: "Numeric Code"},
			"rounding":     {Name: "rounding", Type: metadata.FieldNumeric, Label: "Rounding factor", Required: true},
			"digits":       {Name: "digits", Type: metadata.FieldInteger, Label: "Digits", Required: true},
			"active":       {Name: "active", Type: metadata.FieldBoolean, Label: "Active"},
			"rates":        {Name: "rates", Type: metadata.FieldOne2Many, Label: "Rates", Relation: "currency.currency.rate"},
		},
	})

	metadata.RegisterModel("currency.currency.rate", &metadata.Model{
		Name:  "currency.currency.rate",
		Table: "currency_currency_rate",
		Fields: map[string]*metadata.Field{
			"date":     {Name: "date", Type: metadata.FieldDate, Label: "Date", Required: true},
			"rate":     {Name: "rate", Type: metadata.FieldNumeric, Label: "Rate", Required: true},
			"currency": {Name: "currency", Type: metadata.FieldMany2One, Label: "Currency", Relation: "currency.currency", Required: true},
		},
	})
}
