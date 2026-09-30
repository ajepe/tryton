package stock

import (
	"github.com/ajepe/talon/internal/metadata"
)

func init() {
	metadata.RegisterModule("stock", &metadata.Module{
		Name:    "stock",
		Version: "0.1.0",
		Depends: []string{"ir", "res", "country", "party", "product", "account"},
	})

	metadata.RegisterModel("stock.location", &metadata.Model{
		Name:  "stock.location",
		Table: "stock_location",
		Fields: map[string]*metadata.Field{
			"name":       {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"type":       {Name: "type", Type: metadata.FieldSelection, Label: "Type"},
			"parent":     {Name: "parent", Type: metadata.FieldMany2One, Label: "Parent", Relation: "stock.location"},
			"company":    {Name: "company", Type: metadata.FieldMany2One, Label: "Company", Relation: "company.company"},
			"active":     {Name: "active", Type: metadata.FieldBoolean, Label: "Active"},
		},
	})

	metadata.RegisterModel("stock.warehouse", &metadata.Model{
		Name:  "stock.warehouse",
		Table: "stock_warehouse",
		Fields: map[string]*metadata.Field{
			"name":       {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"company":    {Name: "company", Type: metadata.FieldMany2One, Label: "Company", Relation: "company.company"},
			"input_location":  {Name: "input_location", Type: metadata.FieldMany2One, Label: "Input Location", Relation: "stock.location"},
			"output_location": {Name: "output_location", Type: metadata.FieldMany2One, Label: "Output Location", Relation: "stock.location"},
			"storage_location": {Name: "storage_location", Type: metadata.FieldMany2One, Label: "Storage Location", Relation: "stock.location"},
			"pick_type":  {Name: "pick_type", Type: metadata.FieldMany2One, Label: "Pick Type", Relation: "stock.shipment.out.pick"},
		},
	})

	metadata.RegisterModel("stock.shipment.out", &metadata.Model{
		Name:  "stock.shipment.out",
		Table: "stock_shipment_out",
		Fields: map[string]*metadata.Field{
			"number":     {Name: "number", Type: metadata.FieldChar, Label: "Number"},
			"customer":   {Name: "customer", Type: metadata.FieldMany2One, Label: "Customer", Relation: "party.party"},
			"warehouse":  {Name: "warehouse", Type: metadata.FieldMany2One, Label: "Warehouse", Relation: "stock.warehouse"},
			"state":      {Name: "state", Type: metadata.FieldSelection, Label: "State"},
			"moves":      {Name: "moves", Type: metadata.FieldOne2Many, Label: "Moves", Relation: "stock.move"},
		},
	})

	metadata.RegisterModel("stock.shipment.in", &metadata.Model{
		Name:  "stock.shipment.in",
		Table: "stock_shipment_in",
		Fields: map[string]*metadata.Field{
			"number":     {Name: "number", Type: metadata.FieldChar, Label: "Number"},
			"supplier":   {Name: "supplier", Type: metadata.FieldMany2One, Label: "Supplier", Relation: "party.party"},
			"warehouse":  {Name: "warehouse", Type: metadata.FieldMany2One, Label: "Warehouse", Relation: "stock.warehouse"},
			"state":      {Name: "state", Type: metadata.FieldSelection, Label: "State"},
			"moves":      {Name: "moves", Type: metadata.FieldOne2Many, Label: "Moves", Relation: "stock.move"},
		},
	})

	metadata.RegisterModel("stock.move", &metadata.Model{
		Name:  "stock.move",
		Table: "stock_move",
		Fields: map[string]*metadata.Field{
			"product":    {Name: "product", Type: metadata.FieldMany2One, Label: "Product", Relation: "product.product", Required: true},
			"quantity":   {Name: "quantity", Type: metadata.FieldNumeric, Label: "Quantity", Required: true},
			"uom":        {Name: "uom", Type: metadata.FieldMany2One, Label: "Unit of Measure", Relation: "product.uom"},
			"from_location": {Name: "from_location", Type: metadata.FieldMany2One, Label: "From Location", Relation: "stock.location"},
			"to_location":   {Name: "to_location", Type: metadata.FieldMany2One, Label: "To Location", Relation: "stock.location"},
			"state":      {Name: "state", Type: metadata.FieldSelection, Label: "State"},
			"shipment":   {Name: "shipment", Type: metadata.FieldReference, Label: "Shipment"},
		},
	})

	metadata.RegisterModel("stock.inventory", &metadata.Model{
		Name:  "stock.inventory",
		Table: "stock_inventory",
		Fields: map[string]*metadata.Field{
			"location":   {Name: "location", Type: metadata.FieldMany2One, Label: "Location", Relation: "stock.location", Required: true},
			"date":       {Name: "date", Type: metadata.FieldDate, Label: "Date"},
			"state":      {Name: "state", Type: metadata.FieldSelection, Label: "State"},
			"lines":      {Name: "lines", Type: metadata.FieldOne2Many, Label: "Lines", Relation: "stock.inventory.line"},
		},
	})

	metadata.RegisterModel("stock.inventory.line", &metadata.Model{
		Name:  "stock.inventory.line",
		Table: "stock_inventory_line",
		Fields: map[string]*metadata.Field{
			"inventory":  {Name: "inventory", Type: metadata.FieldMany2One, Label: "Inventory", Relation: "stock.inventory", Required: true},
			"product":    {Name: "product", Type: metadata.FieldMany2One, Label: "Product", Relation: "product.product", Required: true},
			"quantity":   {Name: "quantity", Type: metadata.FieldNumeric, Label: "Quantity"},
			"uom":        {Name: "uom", Type: metadata.FieldMany2One, Label: "Unit of Measure", Relation: "product.uom"},
		},
	})
}
