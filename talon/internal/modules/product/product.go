package product

import (
	"github.com/ajepe/talon/internal/metadata"
)

func init() {
	metadata.RegisterModule("product", &metadata.Module{
		Name:    "product",
		Version: "0.1.0",
		Depends: []string{"ir", "res", "country", "party"},
	})

	metadata.RegisterModel("product.template", &metadata.Model{
		Name:  "product.template",
		Table: "product_template",
		Fields: map[string]*metadata.Field{
			"name":           {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"code":           {Name: "code", Type: metadata.FieldChar, Label: "Code"},
			"type":           {Name: "type", Type: metadata.FieldSelection, Label: "Type"},
			"salable":        {Name: "salable", Type: metadata.FieldBoolean, Label: "Salable"},
			"sale_price":     {Name: "sale_price", Type: metadata.FieldNumeric, Label: "Sale Price"},
			"cost_price":     {Name: "cost_price", Type: metadata.FieldNumeric, Label: "Cost Price"},
			"uom":            {Name: "uom", Type: metadata.FieldMany2One, Label: "Unit of Measure", Relation: "product.uom"},
			"category":       {Name: "category", Type: metadata.FieldMany2One, Label: "Category", Relation: "product.category"},
			"products":       {Name: "products", Type: metadata.FieldOne2Many, Label: "Products", Relation: "product.product"},
			"active":         {Name: "active", Type: metadata.FieldBoolean, Label: "Active"},
		},
	})

	metadata.RegisterModel("product.product", &metadata.Model{
		Name:  "product.product",
		Table: "product_product",
		Fields: map[string]*metadata.Field{
			"template":       {Name: "template", Type: metadata.FieldMany2One, Label: "Template", Relation: "product.template", Required: true},
			"name":           {Name: "name", Type: metadata.FieldChar, Label: "Name"},
			"code":           {Name: "code", Type: metadata.FieldChar, Label: "Code"},
			"description":    {Name: "description", Type: metadata.FieldText, Label: "Description"},
			"active":         {Name: "active", Type: metadata.FieldBoolean, Label: "Active"},
		},
	})

	metadata.RegisterModel("product.category", &metadata.Model{
		Name:  "product.category",
		Table: "product_category",
		Fields: map[string]*metadata.Field{
			"name":       {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"parent":     {Name: "parent", Type: metadata.FieldMany2One, Label: "Parent", Relation: "product.category"},
			"templates":  {Name: "templates", Type: metadata.FieldOne2Many, Label: "Templates", Relation: "product.template"},
		},
	})

	metadata.RegisterModel("product.uom", &metadata.Model{
		Name:  "product.uom",
		Table: "product_uom",
		Fields: map[string]*metadata.Field{
			"name":       {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"symbol":     {Name: "symbol", Type: metadata.FieldChar, Label: "Symbol"},
			"category":   {Name: "category", Type: metadata.FieldMany2One, Label: "Category", Relation: "product.uom.category"},
			"rate":       {Name: "rate", Type: metadata.FieldNumeric, Label: "Rate"},
			"factor":     {Name: "factor", Type: metadata.FieldNumeric, Label: "Factor"},
		},
	})

	metadata.RegisterModel("product.uom.category", &metadata.Model{
		Name:  "product.uom.category",
		Table: "product_uom_category",
		Fields: map[string]*metadata.Field{
			"name":  {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"uoms":  {Name: "uoms", Type: metadata.FieldOne2Many, Label: "Units of Measure", Relation: "product.uom"},
		},
	})
}
