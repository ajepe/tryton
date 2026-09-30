package metadata

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type FieldType string

const (
	FieldChar          FieldType = "char"
	FieldText          FieldType = "text"
	FieldInteger       FieldType = "integer"
	FieldFloat         FieldType = "float"
	FieldNumeric       FieldType = "numeric"
	FieldBoolean       FieldType = "boolean"
	FieldDate          FieldType = "date"
	FieldDateTime      FieldType = "datetime"
	FieldTime          FieldType = "time"
	FieldTimeDelta     FieldType = "timedelta"
	FieldSelection     FieldType = "selection"
	FieldMultiSelection FieldType = "multiselection"
	FieldBinary        FieldType = "binary"
	FieldDict          FieldType = "dict"
	FieldMany2One      FieldType = "many2one"
	FieldOne2One       FieldType = "one2one"
	FieldOne2Many      FieldType = "one2many"
	FieldMany2Many     FieldType = "many2many"
	FieldReference     FieldType = "reference"
)

type Field struct {
	Name        string
	Type        FieldType
	Label       string
	Help        string
	Required    bool
	ReadOnly    bool
	Domain      string
	Selection   []SelectionOption
	Relation    string
	Reverse     string
	Order       string
	Limit       int
	Digits      int
	Loading     string
	OnChange    []string
	OnChangeWith []string
	Autocomplete []string
	States      string
	Context     string
	Searchable  bool
	Sortable    bool
	Unique      bool
	Index       bool
}

type SelectionOption struct {
	Value string
	Label string
}

type Model struct {
	Name       string
	Module     string
	Table      string
	Fields     map[string]*Field
	Inherit    []string
	Order      string
	Limit      int
	History    bool
	Audit      bool
	RecordName string
	PrimaryKey string
}

var (
	registryMu sync.RWMutex
	models     = map[string]*Model{}
	modules    = map[string]*Module{}
)

type Module struct {
	Name      string
	Version   string
	Depends   []string
	Models    []string
	XML       []string
	Upgrades  map[string]func() error
}

func RegisterModel(name string, m *Model) error {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := models[name]; exists {
		return fmt.Errorf("model %s already registered", name)
	}
	m.Name = name
	if m.Fields == nil {
		m.Fields = map[string]*Field{}
	}
	if m.Table == "" {
		m.Table = modelToTable(name)
	}
	models[name] = m
	return nil
}

func RegisterModule(name string, m *Module) error {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := modules[name]; exists {
		return fmt.Errorf("module %s already registered", name)
	}
	modules[name] = m
	return nil
}

func GetModel(name string) (*Model, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	m, ok := models[name]
	return m, ok
}

func GetModule(name string) (*Module, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	m, ok := modules[name]
	return m, ok
}

func ListModels() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(models))
	for name := range models {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func ListModules() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(modules))
	for name := range modules {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func modelToTable(name string) string {
	parts := strings.SplitN(name, ".", 2)
	if len(parts) == 2 {
		return parts[0] + "_" + strings.ReplaceAll(parts[1], ".", "_")
	}
	return strings.ReplaceAll(name, ".", "_")
}

func (m *Model) AddField(f *Field) {
	if m.Fields == nil {
		m.Fields = map[string]*Field{}
	}
	m.Fields[f.Name] = f
}

func (m *Model) FieldNames() []string {
	names := make([]string, 0, len(m.Fields))
	for name := range m.Fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (m *Model) TableName() string {
	if m.Table != "" {
		return m.Table
	}
	return modelToTable(m.Name)
}
