package orm

import (
	"database/sql"
	"fmt"
	"sort"
	"sync"

	"github.com/ajepe/talon/internal/metadata"
)

type Group struct {
	mu           sync.RWMutex
	db           *sql.DB
	model        *metadata.Model
	records      []*Record
	context      map[string]interface{}
	parent       *Record
	parentName   string
	childName    string
	removed      []*Record
	deleted      []*Record
	onWrite      []string
	readonly     bool
	screens      map[string]*Screen
	sequence     string
	sequenceField string
}

func NewGroup(model *metadata.Model) *Group {
	return &Group{
		model:   model,
		records: []*Record{},
		context: map[string]interface{}{},
		screens: map[string]*Screen{},
	}
}

func (g *Group) SetDB(db *sql.DB) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.db = db
}

func (g *Group) DB() *sql.DB {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.db
}

func (g *Group) Model() *metadata.Model {
	return g.model
}

func (g *Group) Records() []*Record {
	g.mu.RLock()
	defer g.mu.RUnlock()
	result := make([]*Record, len(g.records))
	copy(result, g.records)
	return result
}

func (g *Group) Len() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.records)
}

func (g *Group) Add(r *Record) {
	g.mu.Lock()
	defer g.mu.Unlock()
	r.group = g
	g.records = append(g.records, r)
}

func (g *Group) New() *Record {
	r := NewRecord(g.model)
	g.Add(r)
	return r
}

func (g *Group) Copy(r *Record) (*Record, error) {
	if err := r.Load(nil); err != nil {
		return nil, err
	}
	newRecord := g.New()
	for _, name := range r.LoadedFields() {
		field, ok := r.model.Fields[name]
		if !ok || field.IsX2M() {
			continue
		}
		value, _ := r.Get(name)
		newRecord.Set(name, value)
	}
	return newRecord, nil
}

func (g *Group) Remove(r *Record) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, rec := range g.records {
		if rec == r {
			g.records = append(g.records[:i], g.records[i+1:]...)
			g.removed = append(g.removed, r)
			return
		}
	}
}

func (g *Group) Unremove(r *Record) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, rec := range g.removed {
		if rec == r {
			g.removed = append(g.removed[:i], g.removed[i+1:]...)
			g.records = append(g.records, r)
			return
		}
	}
}

func (g *Group) Delete(r *Record) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, rec := range g.records {
		if rec == r {
			g.records = append(g.records[:i], g.records[i+1:]...)
			g.deleted = append(g.deleted, r)
			return
		}
	}
}

func (g *Group) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.records = []*Record{}
	g.removed = []*Record{}
	g.deleted = []*Record{}
}

func (g *Group) Load(ids []int) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.records = []*Record{}
	for _, id := range ids {
		r := NewExistingRecord(g.model, id)
		r.group = g
		g.records = append(g.records, r)
	}
	return nil
}

func (g *Group) GetContext() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()
	ctx := map[string]interface{}{}
	for k, v := range g.context {
		ctx[k] = v
	}
	if g.parent != nil {
		ctx["_parent_" + g.parentName] = g.parent
	}
	return ctx
}

func (g *Group) SetContext(ctx map[string]interface{}) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.context = ctx
}

func (g *Group) IsModified() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, r := range g.records {
		if len(r.ModifiedFields()) > 0 || r.IsNew() {
			return true
		}
	}
	return false
}

func (g *Group) Save() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, r := range g.records {
		if err := r.Save(); err != nil {
			return err
		}
	}
	for _, r := range g.deleted {
		if err := r.Delete(); err != nil {
			return err
		}
	}
	return nil
}

func (g *Group) Written(ids []int) error {
	return g.Load(ids)
}

func (g *Group) OnWrite(ids []int) error {
	return nil
}

func (g *Group) OnWriteIDs() []int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	ids := []int{}
	for _, r := range g.records {
		if !r.IsNew() {
			ids = append(ids, r.ID())
		}
	}
	return ids
}

func (g *Group) GetByPath(path string) *Record {
	g.mu.RLock()
	defer g.mu.RUnlock()
	parts := splitPath(path)
	if len(parts) == 0 {
		return nil
	}
	idx := parts[0]
	for i, r := range g.records {
		if fmt.Sprintf("%d", i) == idx {
			if len(parts) == 1 {
				return r
			}
			return r
		}
	}
	return nil
}

func (g *Group) SetSequence(field string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.sequence = field
}

func (g *Group) Domain4Inversion() string {
	return ""
}

func (g *Group) SelectedRecords() []*Record {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Records()
}

func (g *Group) ListedRecords() []*Record {
	return g.Records()
}

func (g *Group) SortBy(field string, desc bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	sort.Slice(g.records, func(i, j int) bool {
		vi, _ := g.records[i].Get(field)
		vj, _ := g.records[j].Get(field)
		if vi == nil || vj == nil {
			return false
		}
		switch a := vi.(type) {
		case string:
			b, _ := vj.(string)
			if desc {
				return a > b
			}
			return a < b
		case int:
			b, _ := vj.(int)
			if desc {
				return a > b
			}
			return a < b
		case float64:
			b, _ := vj.(float64)
			if desc {
				return a > b
			}
			return a < b
		}
		return false
	})
}

func (g *Group) Parent() *Record {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.parent
}

func (g *Group) SetParent(r *Record, name string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.parent = r
	g.parentName = name
}

func (g *Group) ChildName() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.childName
}

func (g *Group) SetChildName(name string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.childName = name
}

func (g *Group) Readonly() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.readonly
}

func (g *Group) SetReadonly(b bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.readonly = b
}

func (g *Group) Screens() map[string]*Screen {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.screens
}

func (g *Group) AddScreen(name string, s *Screen) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.screens[name] = s
}

func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, c := range path {
		if c == '.' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
