package metadata

import (
	"fmt"
	"strings"
)

func NewField(name string, ftype FieldType) *Field {
	return &Field{
		Name:    name,
		Type:    ftype,
		Loading: "eager",
	}
}

func (f *Field) SetLabel(s string) *Field {
	f.Label = s
	return f
}

func (f *Field) SetHelp(h string) *Field {
	f.Help = h
	return f
}

func (f *Field) SetRequired(b bool) *Field {
	f.Required = b
	return f
}

func (f *Field) SetReadOnly(b bool) *Field {
	f.ReadOnly = b
	return f
}

func (f *Field) SetDomain(d string) *Field {
	f.Domain = d
	return f
}

func (f *Field) SetSelection(opts []SelectionOption) *Field {
	f.Selection = opts
	return f
}

func (f *Field) SetRelation(r string) *Field {
	f.Relation = r
	return f
}

func (f *Field) SetReverse(r string) *Field {
	f.Reverse = r
	return f
}

func (f *Field) SetOrder(o string) *Field {
	f.Order = o
	return f
}

func (f *Field) SetLimit(l int) *Field {
	f.Limit = l
	return f
}

func (f *Field) SetDigits(d int) *Field {
	f.Digits = d
	return f
}

func (f *Field) SetLoading(l string) *Field {
	f.Loading = l
	return f
}

func (f *Field) SetOnChange(fields []string) *Field {
	f.OnChange = fields
	return f
}

func (f *Field) SetOnChangeWith(fields []string) *Field {
	f.OnChangeWith = fields
	return f
}

func (f *Field) SetAutocomplete(fields []string) *Field {
	f.Autocomplete = fields
	return f
}

func (f *Field) SetStates(s string) *Field {
	f.States = s
	return f
}

func (f *Field) SetContext(c string) *Field {
	f.Context = c
	return f
}

func (f *Field) SetSearchable(b bool) *Field {
	f.Searchable = b
	return f
}

func (f *Field) SetSortable(b bool) *Field {
	f.Sortable = b
	return f
}

func (f *Field) SetUnique(b bool) *Field {
	f.Unique = b
	return f
}

func (f *Field) SetIndex(b bool) *Field {
	f.Index = b
	return f
}

func (f *Field) ColumnName() string {
	return f.Name
}

func (f *Field) IsRelation() bool {
	switch f.Type {
	case FieldMany2One, FieldOne2One, FieldOne2Many, FieldMany2Many, FieldReference:
		return true
	}
	return false
}

func (f *Field) IsX2M() bool {
	switch f.Type {
	case FieldOne2Many, FieldMany2Many:
		return true
	}
	return false
}

func (f *Field) IsM2O() bool {
	return f.Type == FieldMany2One || f.Type == FieldOne2One
}

func (f *Field) SQLType() string {
	switch f.Type {
	case FieldChar, FieldSelection, FieldReference:
		if f.Limit > 0 {
			return fmt.Sprintf("varchar(%d)", f.Limit)
		}
		return "varchar"
	case FieldText:
		return "text"
	case FieldInteger:
		return "integer"
	case FieldFloat:
		return "double precision"
	case FieldNumeric:
		return "numeric"
	case FieldBoolean:
		return "boolean"
	case FieldDate:
		return "date"
	case FieldDateTime:
		return "timestamp with time zone"
	case FieldTime:
		return "time"
	case FieldTimeDelta:
		return "interval"
	case FieldBinary:
		return "bytea"
	case FieldDict:
		return "jsonb"
	case FieldMany2One, FieldOne2One:
		return "integer"
	default:
		return "text"
	}
}

func (f *Field) DefaultValue() interface{} {
	switch f.Type {
	case FieldBoolean:
		return false
	case FieldInteger:
		return 0
	case FieldFloat, FieldNumeric:
		return 0.0
	default:
		return nil
	}
}

func (f *Field) Validate(value interface{}) error {
	if f.Required && value == nil {
		return fmt.Errorf("field %s is required", f.Name)
	}
	if f.Domain != "" && value != nil {
		// Domain validation happens at ORM level
	}
	return nil
}

func (f *Field) Description() map[string]interface{} {
	desc := map[string]interface{}{
		"name":     f.Name,
		"type":     string(f.Type),
		"string":   f.Label,
		"help":     f.Help,
		"required": f.Required,
		"readonly": f.ReadOnly,
		"domain":   f.Domain,
		"loading":  f.Loading,
		"searchable": f.Searchable,
		"sortable": f.Sortable,
	}
	if f.Selection != nil {
		desc["selection"] = f.Selection
	}
	if f.Label != "" {
		desc["string"] = f.Label
	}
	if f.Relation != "" {
		desc["relation"] = f.Relation
	}
	if f.Reverse != "" {
		desc["reverse"] = f.Reverse
	}
	if f.Order != "" {
		desc["order"] = f.Order
	}
	if f.Limit > 0 {
		desc["limit"] = f.Limit
	}
	if f.Digits > 0 {
		desc["digits"] = f.Digits
	}
	if f.OnChange != nil {
		desc["on_change"] = f.OnChange
	}
	if f.OnChangeWith != nil {
		desc["on_change_with"] = f.OnChangeWith
	}
	if f.Autocomplete != nil {
		desc["autocomplete"] = f.Autocomplete
	}
	if f.States != "" {
		desc["states"] = f.States
	}
	if f.Context != "" {
		desc["context"] = f.Context
	}
	if f.Label != "" {
		desc["string"] = f.Label
	}
	if f.Unique {
		desc["unique"] = true
	}
	return desc
}

func (f *Field) Describe() string {
	var b strings.Builder
	b.WriteString(f.Name)
	b.WriteString(" ")
	b.WriteString(string(f.Type))
	if f.Required {
		b.WriteString(" required")
	}
	if f.ReadOnly {
		b.WriteString(" readonly")
	}
	if f.Unique {
		b.WriteString(" unique")
	}
	return b.String()
}
