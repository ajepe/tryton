package account

import (
	"github.com/ajepe/talon/internal/metadata"
)

func init() {
	metadata.RegisterModule("account", &metadata.Module{
		Name:    "account",
		Version: "0.1.0",
		Depends: []string{"ir", "res", "country", "party", "product"},
	})

	metadata.RegisterModel("account.account", &metadata.Model{
		Name:  "account.account",
		Table: "account_account",
		Fields: map[string]*metadata.Field{
			"name":     {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"code":     {Name: "code", Type: metadata.FieldChar, Label: "Code"},
			"type":     {Name: "type", Type: metadata.FieldMany2One, Label: "Type", Relation: "account.account.type"},
			"parent":   {Name: "parent", Type: metadata.FieldMany2One, Label: "Parent", Relation: "account.account"},
			"currency": {Name: "currency", Type: metadata.FieldMany2One, Label: "Currency", Relation: "currency.currency"},
			"company":  {Name: "company", Type: metadata.FieldMany2One, Label: "Company", Relation: "company.company"},
			"active":   {Name: "active", Type: metadata.FieldBoolean, Label: "Active"},
		},
	})

	metadata.RegisterModel("account.account.type", &metadata.Model{
		Name:  "account.account.type",
		Table: "account_account_type",
		Fields: map[string]*metadata.Field{
			"name":   {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"parent": {Name: "parent", Type: metadata.FieldMany2One, Label: "Parent", Relation: "account.account.type"},
		},
	})

	metadata.RegisterModel("account.fiscalyear", &metadata.Model{
		Name:  "account.fiscalyear",
		Table: "account_fiscalyear",
		Fields: map[string]*metadata.Field{
			"name":       {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"start_date": {Name: "start_date", Type: metadata.FieldDate, Label: "Start Date", Required: true},
			"end_date":   {Name: "end_date", Type: metadata.FieldDate, Label: "End Date", Required: true},
			"company":    {Name: "company", Type: metadata.FieldMany2One, Label: "Company", Relation: "company.company"},
			"periods":    {Name: "periods", Type: metadata.FieldOne2Many, Label: "Periods", Relation: "account.period"},
			"state":      {Name: "state", Type: metadata.FieldSelection, Label: "State"},
		},
	})

	metadata.RegisterModel("account.period", &metadata.Model{
		Name:  "account.period",
		Table: "account_period",
		Fields: map[string]*metadata.Field{
			"name":       {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"fiscalyear": {Name: "fiscalyear", Type: metadata.FieldMany2One, Label: "Fiscal Year", Relation: "account.fiscalyear", Required: true},
			"start_date": {Name: "start_date", Type: metadata.FieldDate, Label: "Start Date"},
			"end_date":   {Name: "end_date", Type: metadata.FieldDate, Label: "End Date"},
			"state":      {Name: "state", Type: metadata.FieldSelection, Label: "State"},
		},
	})

	metadata.RegisterModel("account.move", &metadata.Model{
		Name:  "account.move",
		Table: "account_move",
		Fields: map[string]*metadata.Field{
			"number":  {Name: "number", Type: metadata.FieldChar, Label: "Number"},
			"date":    {Name: "date", Type: metadata.FieldDate, Label: "Date", Required: true},
			"period":  {Name: "period", Type: metadata.FieldMany2One, Label: "Period", Relation: "account.period"},
			"journal": {Name: "journal", Type: metadata.FieldMany2One, Label: "Journal", Relation: "account.journal"},
			"company": {Name: "company", Type: metadata.FieldMany2One, Label: "Company", Relation: "company.company"},
			"state":   {Name: "state", Type: metadata.FieldSelection, Label: "State"},
			"lines":   {Name: "lines", Type: metadata.FieldOne2Many, Label: "Lines", Relation: "account.move.line"},
		},
	})

	metadata.RegisterModel("account.move.line", &metadata.Model{
		Name:  "account.move.line",
		Table: "account_move_line",
		Fields: map[string]*metadata.Field{
			"move":        {Name: "move", Type: metadata.FieldMany2One, Label: "Move", Relation: "account.move", Required: true},
			"account":     {Name: "account", Type: metadata.FieldMany2One, Label: "Account", Relation: "account.account", Required: true},
			"debit":       {Name: "debit", Type: metadata.FieldNumeric, Label: "Debit"},
			"credit":      {Name: "credit", Type: metadata.FieldNumeric, Label: "Credit"},
			"amount":      {Name: "amount", Type: metadata.FieldNumeric, Label: "Amount"},
			"description": {Name: "description", Type: metadata.FieldChar, Label: "Description"},
			"party":       {Name: "party", Type: metadata.FieldMany2One, Label: "Party", Relation: "party.party"},
		},
	})

	metadata.RegisterModel("account.journal", &metadata.Model{
		Name:  "account.journal",
		Table: "account_journal",
		Fields: map[string]*metadata.Field{
			"name":    {Name: "name", Type: metadata.FieldChar, Label: "Name", Required: true},
			"type":    {Name: "type", Type: metadata.FieldSelection, Label: "Type"},
			"company": {Name: "company", Type: metadata.FieldMany2One, Label: "Company", Relation: "company.company"},
		},
	})

	metadata.RegisterModel("account.configuration", &metadata.Model{
		Name:  "account.configuration",
		Table: "account_configuration",
		Fields: map[string]*metadata.Field{
			"company": {Name: "company", Type: metadata.FieldMany2One, Label: "Company", Relation: "company.company"},
		},
	})

	metadata.RegisterModel("account.configuration.sequence", &metadata.Model{
		Name:  "account.configuration.sequence",
		Table: "account_configuration_sequence",
		Fields: map[string]*metadata.Field{
			"configuration": {Name: "configuration", Type: metadata.FieldMany2One, Label: "Configuration", Relation: "account.configuration"},
			"move_sequence": {Name: "move_sequence", Type: metadata.FieldMany2One, Label: "Move Sequence", Relation: "ir.sequence"},
		},
	})
}
