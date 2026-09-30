package country

import (
	"github.com/ajepe/talon/internal/metadata"
)

func init() {
	metadata.RegisterModule("country", &metadata.Module{
		Name:    "country",
		Version: "0.1.0",
		Depends: []string{"ir", "res"},
	})

	metadata.RegisterModel("country.country", &metadata.Model{
		Name:  "country.country",
		Table: "country_country",
		Fields: map[string]*metadata.Field{
			"name":         {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"code":         {Name: "code", Type: metadata.FieldChar, Label: "Code", Required: true},
			"iso_code":     {Name: "iso_code", Type: metadata.FieldChar, Label: "ISO Code"},
			"currency":     {Name: "currency", Type: metadata.FieldMany2One, Label: "Currency", Relation: "currency.currency"},
			"subdivisions": {Name: "subdivisions", Type: metadata.FieldOne2Many, Label: "Subdivisions", Relation: "country.subdivision"},
		},
	})

	metadata.RegisterModel("country.subdivision", &metadata.Model{
		Name:  "country.subdivision",
		Table: "country_subdivision",
		Fields: map[string]*metadata.Field{
			"name":    {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"code":    {Name: "code", Type: metadata.FieldChar, Label: "Code"},
			"country": {Name: "country", Type: metadata.FieldMany2One, Label: "Country", Relation: "country.country", Required: true},
			"parent":  {Name: "parent", Type: metadata.FieldMany2One, Label: "Parent", Relation: "country.subdivision"},
			"type":    {Name: "type", Type: metadata.FieldSelection, Label: "Type"},
		},
	})

	metadata.RegisterModel("country.postal_code", &metadata.Model{
		Name:  "country.postal_code",
		Table: "country_postal_code",
		Fields: map[string]*metadata.Field{
			"country":     {Name: "country", Type: metadata.FieldMany2One, Label: "Country", Relation: "country.country", Required: true},
			"subdivision": {Name: "subdivision", Type: metadata.FieldMany2One, Label: "Subdivision", Relation: "country.subdivision"},
			"postal_code": {Name: "postal_code", Type: metadata.FieldChar, Label: "Postal Code", Required: true},
			"city":        {Name: "city", Type: metadata.FieldChar, Label: "City"},
		},
	})
}
