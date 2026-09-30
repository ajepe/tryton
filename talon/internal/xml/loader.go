package xml

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ajepe/talon/internal/metadata"
	"github.com/ajepe/talon/internal/orm"
)

type Document struct {
	XMLName xml.Name `xml:"tryton"`
	Data    []Record `xml:"data>record"`
}

type Record struct {
	Model    string  `xml:"model,attr"`
	ID       string  `xml:"id,attr"`
	Inherit  string  `xml:"inherit,attr"`
	NoUpdate bool    `xml:"noupdate,attr"`
	Fields   []Field `xml:"field"`
}

type Field struct {
	Name    string   `xml:"name,attr"`
	Eval    string   `xml:"eval,attr"`
	Ref     string   `xml:"ref,attr"`
	Type    string   `xml:"type,attr"`
	Value   string   `xml:",innerxml"`
	Records []Record `xml:"record"`
}

type Loader struct {
	db     *sql.DB
	pool   *orm.Pool
	module string
}

func NewLoader(db *sql.DB, pool *orm.Pool, module string) *Loader {
	return &Loader{db: db, pool: pool, module: module}
}

func (l *Loader) LoadFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return l.Load(file)
}

func (l *Loader) LoadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	names := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".xml") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		if err := l.LoadFile(filepath.Join(dir, name)); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func (l *Loader) Load(reader io.Reader) error {
	var document Document
	decoder := xml.NewDecoder(reader)
	if err := decoder.Decode(&document); err != nil {
		return err
	}
	ctx := context.Background()
	for _, record := range document.Data {
		if _, err := l.loadRecord(ctx, record, l.module); err != nil {
			return fmt.Errorf("record %s.%s: %w", record.Model, record.ID, err)
		}
	}
	return nil
}

func (l *Loader) loadRecord(ctx context.Context, record Record, module string) (int, error) {
	if record.Model == "" {
		return 0, fmt.Errorf("record without model")
	}
	model, ok := metadata.GetModel(record.Model)
	if !ok {
		return 0, fmt.Errorf("unknown model %s", record.Model)
	}
	if l.db == nil {
		return 0, fmt.Errorf("no database connection")
	}

	values := map[string]interface{}{}
	for _, field := range record.Fields {
		value, err := l.fieldValue(ctx, model, field, module)
		if err != nil {
			return 0, fmt.Errorf("field %s: %w", field.Name, err)
		}
		if value == nil {
			continue
		}
		values[field.Name] = value
	}

	existing, err := l.modelData(ctx, module, record.ID)
	if err != nil {
		return 0, err
	}

	group, err := l.pool.NewGroup(record.Model)
	if err != nil {
		return 0, err
	}

	if existing != 0 {
		if err := group.Load([]int{existing}); err != nil {
			return 0, err
		}
		records := group.Records()
		if len(records) == 0 {
			return 0, fmt.Errorf("record %d for %s.%s not found", existing, module, record.ID)
		}
		records[0].ImportData(values)
		if err := records[0].Save(); err != nil {
			return 0, err
		}
		return existing, nil
	}

	r := group.New()
	r.ImportData(values)
	if err := r.Save(); err != nil {
		return 0, err
	}
	if record.ID != "" {
		if err := l.createModelData(ctx, module, record.Model, record.ID, r.ID(), record.NoUpdate); err != nil {
			return 0, err
		}
	}
	return r.ID(), nil
}

func (l *Loader) fieldValue(ctx context.Context, model *metadata.Model, field Field, module string) (interface{}, error) {
	if len(field.Records) > 0 {
		ids := []interface{}{}
		for _, nested := range field.Records {
			nestedModule := module
			if nested.Model == "" {
				nested.Model = model.Fields[field.Name].Relation
			}
			id, err := l.loadRecord(ctx, nested, nestedModule)
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
		return ids, nil
	}

	if field.Ref != "" {
		refModule, refID := module, field.Ref
		if idx := strings.Index(field.Ref, "."); idx >= 0 {
			refModule, refID = field.Ref[:idx], field.Ref[idx+1:]
		}
		value, err := l.modelData(ctx, refModule, refID)
		if err != nil {
			return nil, err
		}
		if value == 0 {
			return nil, fmt.Errorf("reference %s not found", field.Ref)
		}
		return l.coerce(model, field.Name, value)
	}

	if field.Eval != "" {
		value, err := EvalLiteral(field.Eval)
		if err != nil {
			return nil, err
		}
		return l.coerce(model, field.Name, value)
	}

	text := strings.TrimSpace(field.Value)
	if text == "" {
		return nil, nil
	}
	if field.Type == "xml" || field.Type == "text" {
		return text, nil
	}
	f, ok := model.Fields[field.Name]
	if !ok {
		return nil, fmt.Errorf("unknown field %s", field.Name)
	}
	switch f.Type {
	case metadata.FieldInteger, metadata.FieldMany2One, metadata.FieldOne2One:
		value, err := EvalLiteral(text)
		if err != nil {
			return nil, err
		}
		return l.coerce(model, field.Name, value)
	case metadata.FieldFloat, metadata.FieldNumeric, metadata.FieldBoolean:
		value, err := EvalLiteral(text)
		if err != nil {
			return nil, err
		}
		return l.coerce(model, field.Name, value)
	default:
		return text, nil
	}
}

func (l *Loader) coerce(model *metadata.Model, name string, value interface{}) (interface{}, error) {
	f, ok := model.Fields[name]
	if !ok {
		return value, nil
	}
	switch f.Type {
	case metadata.FieldInteger, metadata.FieldMany2One, metadata.FieldOne2One:
		if n, ok := orm.AsInt(value); ok {
			return n, nil
		}
		return value, nil
	case metadata.FieldFloat, metadata.FieldNumeric:
		if n, ok := orm.AsFloat(value); ok {
			return n, nil
		}
		return value, nil
	case metadata.FieldBoolean:
		return orm.AsBool(value), nil
	default:
		return value, nil
	}
}

func (l *Loader) modelData(ctx context.Context, module, fsID string) (int, error) {
	if fsID == "" {
		return 0, nil
	}
	var dbID sql.NullInt64
	err := l.db.QueryRowContext(ctx,
		`SELECT db_id FROM ir_model_data WHERE module = $1 AND fs_id = $2`,
		module, fsID).Scan(&dbID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !dbID.Valid {
		return 0, nil
	}
	return int(dbID.Int64), nil
}

func (l *Loader) createModelData(ctx context.Context, module, model, fsID string, dbID int, noUpdate bool) error {
	_, err := l.db.ExecContext(ctx,
		`INSERT INTO ir_model_data (module, model, fs_id, db_id, noupdate)
		 VALUES ($1, $2, $3, $4, $5)`,
		module, model, fsID, dbID, noUpdate)
	return err
}
