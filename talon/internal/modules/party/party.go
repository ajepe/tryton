package party

import (
	"github.com/ajepe/talon/internal/metadata"
)

func init() {
	metadata.RegisterModule("party", &metadata.Module{
		Name:    "party",
		Version: "0.1.0",
		Depends: []string{"ir", "res", "country"},
	})

	metadata.RegisterModel("party.party", &metadata.Model{
		Name:  "party.party",
		Table: "party_party",
		Fields: map[string]*metadata.Field{
			"name":        {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"code":        {Name: "code", Type: metadata.FieldChar, Label: "Code"},
			"categories":  {Name: "categories", Type: metadata.FieldMany2Many, Label: "Categories", Relation: "party.category"},
			"addresses":   {Name: "addresses", Type: metadata.FieldOne2Many, Label: "Addresses", Relation: "party.address"},
			"contact_mechanisms": {Name: "contact_mechanisms", Type: metadata.FieldOne2Many, Label: "Contact Mechanisms", Relation: "party.contact_mechanism"},
			"identifiers": {Name: "identifiers", Type: metadata.FieldOne2Many, Label: "Identifiers", Relation: "party.identifier"},
			"active":      {Name: "active", Type: metadata.FieldBoolean, Label: "Active"},
			"lang":        {Name: "lang", Type: metadata.FieldMany2One, Label: "Language", Relation: "ir.lang"},
		},
	})

	metadata.RegisterModel("party.category", &metadata.Model{
		Name:  "party.category",
		Table: "party_category",
		Fields: map[string]*metadata.Field{
			"name":   {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"parent": {Name: "parent", Type: metadata.FieldMany2One, Label: "Parent", Relation: "party.category"},
			"parties": {Name: "parties", Type: metadata.FieldMany2Many, Label: "Parties", Relation: "party.party"},
		},
	})

	metadata.RegisterModel("party.address", &metadata.Model{
		Name:  "party.address",
		Table: "party_address",
		Fields: map[string]*metadata.Field{
			"party":        {Name: "party", Type: metadata.FieldMany2One, Label: "Party", Relation: "party.party", Required: true},
			"street":       {Name: "street", Type: metadata.FieldChar, Label: "Street"},
			"street_bis":   {Name: "street_bis", Type: metadata.FieldChar, Label: "Street (bis)"},
			"zip":          {Name: "zip", Type: metadata.FieldChar, Label: "Zip"},
			"city":         {Name: "city", Type: metadata.FieldChar, Label: "City"},
			"country":      {Name: "country", Type: metadata.FieldMany2One, Label: "Country", Relation: "country.country"},
			"subdivision":  {Name: "subdivision", Type: metadata.FieldMany2One, Label: "Subdivision", Relation: "country.subdivision"},
			"sequence":     {Name: "sequence", Type: metadata.FieldInteger, Label: "Sequence"},
		},
	})

	metadata.RegisterModel("party.contact_mechanism", &metadata.Model{
		Name:  "party.contact_mechanism",
		Table: "party_contact_mechanism",
		Fields: map[string]*metadata.Field{
			"party":    {Name: "party", Type: metadata.FieldMany2One, Label: "Party", Relation: "party.party", Required: true},
			"type":     {Name: "type", Type: metadata.FieldSelection, Label: "Type", Required: true},
			"value":    {Name: "value", Type: metadata.FieldChar, Label: "Value", Required: true},
			"sequence": {Name: "sequence", Type: metadata.FieldInteger, Label: "Sequence"},
		},
	})

	metadata.RegisterModel("party.identifier", &metadata.Model{
		Name:  "party.identifier",
		Table: "party_identifier",
		Fields: map[string]*metadata.Field{
			"party":   {Name: "party", Type: metadata.FieldMany2One, Label: "Party", Relation: "party.party", Required: true},
			"type":    {Name: "type", Type: metadata.FieldSelection, Label: "Type", Required: true},
			"code":    {Name: "code", Type: metadata.FieldChar, Label: "Code", Required: true},
			"active":  {Name: "active", Type: metadata.FieldBoolean, Label: "Active"},
		},
	})

	metadata.RegisterModel("party.party.lang", &metadata.Model{
		Name:  "party.party.lang",
		Table: "party_party_lang",
		Fields: map[string]*metadata.Field{
			"party": {Name: "party", Type: metadata.FieldMany2One, Label: "Party", Relation: "party.party", Required: true},
			"lang":  {Name: "lang", Type: metadata.FieldMany2One, Label: "Language", Relation: "ir.lang", Required: true},
		},
	})

	metadata.RegisterModel("party.replace.ask", &metadata.Model{
		Name:  "party.replace.ask",
		Table: "party_replace_ask",
		Fields: map[string]*metadata.Field{
			"source": {Name: "source", Type: metadata.FieldMany2One, Label: "Source", Relation: "party.party"},
			"target": {Name: "target", Type: metadata.FieldMany2One, Label: "Target", Relation: "party.party"},
		},
	})

	metadata.RegisterModel("party.erase.ask", &metadata.Model{
		Name:  "party.erase.ask",
		Table: "party_erase_ask",
		Fields: map[string]*metadata.Field{
			"party": {Name: "party", Type: metadata.FieldMany2One, Label: "Party", Relation: "party.party"},
		},
	})
}
