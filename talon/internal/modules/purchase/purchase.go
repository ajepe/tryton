package purchase

import (
	"github.com/ajepe/talon/internal/metadata"
)

func init() {
	metadata.RegisterModule("purchase", &metadata.Module{
		Name:    "purchase",
		Version: "0.1.0",
		Depends: []string{"ir", "res", "country", "party", "product", "account", "stock"},
	})

	metadata.RegisterModel("purchase.purchase", &metadata.Model{
		Name:  "purchase.purchase",
		Table: "purchase_purchase",
		Fields: map[string]*metadata.Field{
			"number":     {Name: "number", Type: metadata.FieldChar, Label: "Number"},
			"supplier":   {Name: "supplier", Type: metadata.FieldMany2One, Label: "Supplier", Relation: "party.party", Required: true},
			"shipment_address": {Name: "shipment_address", Type: metadata.FieldMany2One, Label: "Shipment Address", Relation: "party.address"},
			"invoice_address":  {Name: "invoice_address", Type: metadata.FieldMany2One, Label: "Invoice Address", Relation: "party.address"},
			"date":       {Name: "date", Type: metadata.FieldDate, Label: "Date"},
			"state":      {Name: "state", Type: metadata.FieldSelection, Label: "State"},
			"lines":      {Name: "lines", Type: metadata.FieldOne2Many, Label: "Lines", Relation: "purchase.line"},
			"amount":     {Name: "amount", Type: metadata.FieldNumeric, Label: "Amount"},
			"company":    {Name: "company", Type: metadata.FieldMany2One, Label: "Company", Relation: "company.company"},
		},
	})

	metadata.RegisterModel("purchase.line", &metadata.Model{
		Name:  "purchase.line",
		Table: "purchase_line",
		Fields: map[string]*metadata.Field{
			"purchase":   {Name: "purchase", Type: metadata.FieldMany2One, Label: "Purchase", Relation: "purchase.purchase", Required: true},
			"product":    {Name: "product", Type: metadata.FieldMany2One, Label: "Product", Relation: "product.product"},
			"quantity":   {Name: "quantity", Type: metadata.FieldNumeric, Label: "Quantity"},
			"unit_price": {Name: "unit_price", Type: metadata.FieldNumeric, Label: "Unit Price"},
			"amount":     {Name: "amount", Type: metadata.FieldNumeric, Label: "Amount"},
			"description": {Name: "description", Type: metadata.FieldChar, Label: "Description"},
		},
	})

	metadata.RegisterModel("purchase.configuration", &metadata.Model{
		Name:  "purchase.configuration",
		Table: "purchase_configuration",
		Fields: map[string]*metadata.Field{
			"company": {Name: "company", Type: metadata.FieldMany2One, Label: "Company", Relation: "company.company"},
		},
	})

	metadata.RegisterModel("purchase.configuration.sequence", &metadata.Model{
		Name:  "purchase.configuration.sequence",
		Table: "purchase_configuration_sequence",
		Fields: map[string]*metadata.Field{
			"configuration": {Name: "configuration", Type: metadata.FieldMany2One, Label: "Configuration", Relation: "purchase.configuration"},
			"purchase_sequence": {Name: "purchase_sequence", Type: metadata.FieldMany2One, Label: "Purchase Sequence", Relation: "ir.sequence"},
		},
	})
}
