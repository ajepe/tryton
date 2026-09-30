package orm

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ajepe/talon/internal/metadata"
)

type Record struct {
	mu              sync.RWMutex
	model           *metadata.Model
	id              int
	values          map[string]interface{}
	loaded          map[string]bool
	modified        map[string]bool
	timestamp       map[string]time.Time
	resources       map[string]interface{}
	buttonClicks    map[string]interface{}
	stateAttrs      map[string]interface{}
	exception       error
	destroyed       bool
	parent          *Record
	parentName      string
	childName       string
	group           *Group
	onWrite         []string
	negativeID      int
}

var negativeIDCounter int
var negativeIDMu sync.Mutex

func NewRecord(model *metadata.Model) *Record {
	negativeIDMu.Lock()
	negativeIDCounter--
	nid := negativeIDCounter
	negativeIDMu.Unlock()

	return &Record{
		model:      model,
		id:         nid,
		values:     map[string]interface{}{},
		loaded:     map[string]bool{},
		modified:   map[string]bool{},
		timestamp:  map[string]time.Time{},
		resources:  map[string]interface{}{},
		stateAttrs: map[string]interface{}{},
		negativeID: nid,
	}
}

func NewExistingRecord(model *metadata.Model, id int) *Record {
	return &Record{
		model:      model,
		id:         id,
		values:     map[string]interface{}{},
		loaded:     map[string]bool{},
		modified:   map[string]bool{},
		timestamp:  map[string]time.Time{},
		resources:  map[string]interface{}{},
		stateAttrs: map[string]interface{}{},
	}
}

func (r *Record) ID() int {
	return r.id
}

func (r *Record) Model() *metadata.Model {
	return r.model
}

func (r *Record) SetID(id int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.id = id
}

func (r *Record) IsNew() bool {
	return r.id < 0
}

func (r *Record) IsDestroyed() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.destroyed
}

func (r *Record) Destroy() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.destroyed = true
}

func (r *Record) Get(name string) (interface{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.loaded[name] {
		return nil, false
	}
	v, ok := r.values[name]
	return v, ok
}

func (r *Record) Set(name string, value interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[name] = value
	r.loaded[name] = true
	r.modified[name] = true
}

func (r *Record) GetClient(name string) interface{} {
	v, ok := r.Get(name)
	if !ok {
		return nil
	}
	return v
}

func (r *Record) SetClient(name string, value interface{}) {
	r.Set(name, value)
}

func (r *Record) IsModified(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.modified[name]
}

func (r *Record) ModifiedFields() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fields := make([]string, 0, len(r.modified))
	for name := range r.modified {
		fields = append(fields, name)
	}
	return fields
}

func (r *Record) IsLoaded(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.loaded[name]
}

func (r *Record) LoadedFields() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fields := make([]string, 0, len(r.loaded))
	for name := range r.loaded {
		fields = append(fields, name)
	}
	return fields
}

func (r *Record) GetTimestamp() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ts := map[string]interface{}{}
	for k, v := range r.timestamp {
		ts[k] = v.Format(time.RFC3339)
	}
	return ts
}

func (r *Record) SetTimestamp(field string, ts time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.timestamp[field] = ts
}

func (r *Record) GetContext() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ctx := map[string]interface{}{
		"_user": r.id,
	}
	for k, v := range r.values {
		if !r.modified[k] {
			ctx[k] = v
		}
	}
	return ctx
}

func (r *Record) SetContext(ctx map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, v := range ctx {
		if k == "_user" || k == "_timestamp" {
			continue
		}
		r.values[k] = v
		r.loaded[k] = true
	}
}

func (r *Record) Validate() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for name, field := range r.model.Fields {
		if !r.loaded[name] {
			if field.Required {
				return fmt.Errorf("field %s: required", name)
			}
			continue
		}
		v := r.values[name]
		if err := field.Validate(v); err != nil {
			return fmt.Errorf("field %s: %w", name, err)
		}
	}
	return nil
}

func (r *Record) PreValidate() error {
	return r.Validate()
}

func (r *Record) GetOnChangeValue() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := map[string]interface{}{}
	for name := range r.modified {
		values[name] = r.values[name]
	}
	for name := range r.loaded {
		if !r.modified[name] {
			values[name] = r.values[name]
		}
	}
	return values
}

func (r *Record) SetOnChange(values map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for name, v := range values {
		r.values[name] = v
		r.loaded[name] = true
		r.modified[name] = true
	}
}

func (r *Record) OnChange(fields []string) error {
	// Trigger on_change RPC for given fields
	return nil
}

func (r *Record) OnChangeWith(fields []string) error {
	// Trigger on_change_with RPC for given fields
	return nil
}

func (r *Record) Save() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.IsNew() {
		return r.create()
	}
	return r.write()
}

func (r *Record) create() error {
	db := r.db()
	if db == nil {
		return fmt.Errorf("no database connection")
	}

	table := quoteIdent(r.model.TableName())
	columns := []string{}
	values := []interface{}{}

	for _, name := range r.model.FieldNames() {
		field := r.model.Fields[name]
		if field.IsX2M() {
			continue
		}
		if !r.loaded[name] {
			continue
		}
		columns = append(columns, quoteIdent(field.ColumnName()))
		values = append(values, r.values[name])
	}

	if len(columns) == 0 {
		return fmt.Errorf("no fields to insert")
	}

	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING id",
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	var id int
	err := db.QueryRowContext(context.Background(), query, values...).Scan(&id)
	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}

	r.id = id
	r.negativeID = 0
	return nil
}

func (r *Record) write() error {
	db := r.db()
	if db == nil {
		return fmt.Errorf("no database connection")
	}

	table := quoteIdent(r.model.TableName())
	setParts := []string{}
	values := []interface{}{}

	for _, name := range r.model.FieldNames() {
		field := r.model.Fields[name]
		if field.IsX2M() {
			continue
		}
		if !r.modified[name] {
			continue
		}
		setParts = append(setParts, fmt.Sprintf("%s = $%d", quoteIdent(field.ColumnName()), len(values)+1))
		values = append(values, r.values[name])
	}

	if len(setParts) == 0 {
		return nil
	}

	values = append(values, r.id)
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = $%d",
		table,
		strings.Join(setParts, ", "),
		len(values),
	)

	_, err := db.ExecContext(context.Background(), query, values...)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	return nil
}

func (r *Record) Delete() error {
	db := r.db()
	if db == nil {
		return fmt.Errorf("no database connection")
	}

	if r.IsNew() {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.destroyed = true
		return nil
	}

	table := quoteIdent(r.model.TableName())
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", table)

	_, err := db.ExecContext(context.Background(), query, r.id)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.destroyed = true
	return nil
}

func (r *Record) db() *sql.DB {
	if r.group != nil {
		return r.group.DB()
	}
	return nil
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func (r *Record) Load(fields []string) error {
	db := r.db()
	if db == nil {
		return fmt.Errorf("no database connection")
	}

	table := quoteIdent(r.model.TableName())
	if len(fields) == 0 {
		fields = r.model.FieldNames()
	}

	columns := make([]string, 0, len(fields))
	loaded := make([]string, 0, len(fields))
	for _, name := range fields {
		field, ok := r.model.Fields[name]
		if !ok {
			continue
		}
		if field.IsX2M() {
			continue
		}
		columns = append(columns, quoteIdent(field.ColumnName()))
		loaded = append(loaded, name)
	}

	if len(columns) == 0 {
		return nil
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE id = $1",
		strings.Join(columns, ", "),
		table,
	)

	row := db.QueryRowContext(context.Background(), query, r.id)
	values := make([]interface{}, len(columns))
	ptrs := make([]interface{}, len(columns))
	for i := range values {
		ptrs[i] = &values[i]
	}

	if err := row.Scan(ptrs...); err != nil {
		return fmt.Errorf("load: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for i, name := range loaded {
		r.values[name] = values[i]
		r.loaded[name] = true
	}

	return nil
}

func (r *Record) Copy(defaults map[string]interface{}) *Record {
	r.mu.RLock()
	defer r.mu.RUnlock()
	newRecord := NewRecord(r.model)
	for name, v := range r.values {
		newRecord.values[name] = v
		newRecord.loaded[name] = true
	}
	for name, v := range defaults {
		newRecord.values[name] = v
		newRecord.loaded[name] = true
		newRecord.modified[name] = true
	}
	return newRecord
}

func (r *Record) ExportData(fields []string) map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	data := map[string]interface{}{}
	for _, name := range fields {
		if r.loaded[name] {
			data[name] = r.values[name]
		}
	}
	return data
}

func (r *Record) ImportData(data map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for name, v := range data {
		r.values[name] = v
		r.loaded[name] = true
		r.modified[name] = true
	}
}

func (r *Record) MarshalJSON() ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return json.Marshal(r.values)
}

func (r *Record) UnmarshalJSON(data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return json.Unmarshal(data, &r.values)
}

func (r *Record) String() string {
	return fmt.Sprintf("%s(%d)", r.model.Name, r.id)
}

func (r *Record) RecName() string {
	if name, ok := r.Get("name"); ok {
		if s, ok := name.(string); ok {
			return s
		}
	}
	if code, ok := r.Get("code"); ok {
		if s, ok := code.(string); ok {
			return s
		}
	}
	return fmt.Sprintf("%s #%d", r.model.Name, r.id)
}
