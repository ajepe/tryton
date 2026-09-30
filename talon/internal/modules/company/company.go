package company

import (
	"github.com/ajepe/talon/internal/metadata"
)

func init() {
	metadata.RegisterModule("company", &metadata.Module{
		Name:    "company",
		Version: "0.1.0",
		Depends: []string{"currency", "ir", "party", "res"},
	})

	metadata.RegisterModel("company.company", &metadata.Model{
		Name:  "company.company",
		Table: "company_company",
		Fields: map[string]*metadata.Field{
			"party":     {Name: "party", Type: metadata.FieldMany2One, Label: "Party", Relation: "party.party", Required: true},
			"header":    {Name: "header", Type: metadata.FieldText, Label: "Header"},
			"footer":    {Name: "footer", Type: metadata.FieldText, Label: "Footer"},
			"logo":      {Name: "logo", Type: metadata.FieldBinary, Label: "Logo"},
			"currency":  {Name: "currency", Type: metadata.FieldMany2One, Label: "Currency", Relation: "currency.currency", Required: true},
			"timezone":  {Name: "timezone", Type: metadata.FieldSelection, Label: "Timezone"},
			"employees": {Name: "employees", Type: metadata.FieldOne2Many, Label: "Employees", Relation: "company.employee"},
		},
	})

	metadata.RegisterModel("company.employee", &metadata.Model{
		Name:  "company.employee",
		Table: "company_employee",
		Fields: map[string]*metadata.Field{
			"party":        {Name: "party", Type: metadata.FieldMany2One, Label: "Party", Relation: "party.party", Required: true},
			"company":      {Name: "company", Type: metadata.FieldMany2One, Label: "Company", Relation: "company.company", Required: true},
			"start_date":   {Name: "start_date", Type: metadata.FieldDate, Label: "Start Date"},
			"end_date":     {Name: "end_date", Type: metadata.FieldDate, Label: "End Date"},
			"supervisor":   {Name: "supervisor", Type: metadata.FieldMany2One, Label: "Supervisor", Relation: "company.employee"},
			"subordinates": {Name: "subordinates", Type: metadata.FieldOne2Many, Label: "Subordinates", Relation: "company.employee"},
		},
	})
}
