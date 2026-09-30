package sale

import (
	"github.com/ajepe/talon/internal/metadata"
)

func init() {
	metadata.RegisterModule("sale", &metadata.Module{
		Name:    "sale",
		Version: "0.1.0",
		Depends: []string{"ir", "res", "country", "party", "product", "account", "stock"},
	})

	metadata.RegisterModel("sale.sale", &metadata.Model{
		Name:  "sale.sale",
		Table: "sale_sale",
		Fields: map[string]*metadata.Field{
			"number":     {Name: "number", Type: metadata.FieldChar, Label: "Number"},
			"customer":   {Name: "customer", Type: metadata.FieldMany2One, Label: "Customer", Relation: "party.party", Required: true},
			"shipment_address": {Name: "shipment_address", Type: metadata.FieldMany2One, Label: "Shipment Address", Relation: "party.address"},
			"invoice_address":  {Name: "invoice_address", Type: metadata.FieldMany2One, Label: "Invoice Address", Relation: "party.address"},
			"date":       {Name: "date", Type: metadata.FieldDate, Label: "Date"},
			"state":      {Name: "state", Type: metadata.FieldSelection, Label: "State"},
			"lines":      {Name: "lines", Type: metadata.FieldOne2Many, Label: "Lines", Relation: "sale.line"},
			"amount":     {Name: "amount", Type: metadata.FieldNumeric, Label: "Amount"},
			"company":    {Name: "company", Type: metadata.FieldMany2One, Label: "Company", Relation: "company.company"},
		},
	})

	metadata.RegisterModel("sale.line", &metadata.Model{
		Name:  "sale.line",
		Table: "sale_line",
		Fields: map[string]*metadata.Field{
			"sale":       {Name: "sale", Type: metadata.FieldMany2One, Label: "Sale", Relation: "sale.sale", Required: true},
			"product":    {Name: "product", Type: metadata.FieldMany2One, Label: "Product", Relation: "product.product"},
			"quantity":   {Name: "quantity", Type: metadata.FieldNumeric, Label: "Quantity"},
			"unit_price": {Name: "unit_price", Type: metadata.FieldNumeric, Label: "Unit Price"},
			"amount":     {Name: "amount", Type: metadata.FieldNumeric, Label: "Amount"},
			"description": {Name: "description", Type: metadata.FieldChar, Label: "Description"},
		},
	})

	metadata.RegisterModel("sale.configuration", &metadata.Model{
		Name:  "sale.configuration",
		Table: "sale_configuration",
		Fields: map[string]*metadata.Field{
			"company": {Name: "company", Type: metadata.FieldMany2One, Label: "Company", Relation: "company.company"},
		},
	})

	metadata.RegisterModel("sale.configuration.sequence", &metadata.Model{
		Name:  "sale.configuration.sequence",
		Table: "sale_configuration_sequence",
		Fields: map[string]*metadata.Field{
			"configuration": {Name: "configuration", Type: metadata.FieldMany2One, Label: "Configuration", Relation: "sale.configuration"},
			"sale_sequence": {Name: "sale_sequence", Type: metadata.FieldMany2One, Label: "Sale Sequence", Relation: "ir.sequence"},
		},
	})
}
