package orm

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/ajepe/talon/internal/metadata"
)

type View struct {
	ID       int
	Type     string
	Arch     string
	Fields   map[string]*metadata.Field
	FieldChilds string
	Model    *metadata.Model
}

type Screen struct {
	mu           sync.RWMutex
	db           *sql.DB
	model        *metadata.Model
	group        *Group
	views        []*View
	currentView  int
	domain       string
	limit        int
	offset       int
	order        string
	searchText   string
	searchDomain string
	context      map[string]interface{}
	activeID     int
	activeIDs    []int
	selectedIDs []int
	readonly     bool
	viewType     string
	viewIDs      []int
	fieldChilds  string
	records      []*Record
}

func NewScreen(model *metadata.Model) *Screen {
	return &Screen{
		model:   model,
		group:   NewGroup(model),
		views:   []*View{},
		limit:   1000,
		context: map[string]interface{}{},
	}
}

func (s *Screen) Model() *metadata.Model {
	return s.model
}

func (s *Screen) Group() *Group {
	return s.group
}

func (s *Screen) SetGroup(g *Group) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.group = g
}

func (s *Screen) AddView(v *View) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.views = append(s.views, v)
}

func (s *Screen) Views() []*View {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.views
}

func (s *Screen) CurrentView() *View {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.currentView < 0 || s.currentView >= len(s.views) {
		return nil
	}
	return s.views[s.currentView]
}

func (s *Screen) SwitchView(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.views) {
		return fmt.Errorf("invalid view index %d", index)
	}
	s.currentView = index
	return nil
}

func (s *Screen) SetDomain(domain string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.domain = domain
}

func (s *Screen) Domain() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.domain
}

func (s *Screen) SetLimit(limit int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.limit = limit
}

func (s *Screen) Limit() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.limit
}

func (s *Screen) SetOffset(offset int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.offset = offset
}

func (s *Screen) Offset() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.offset
}

func (s *Screen) SetOrder(order string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.order = order
}

func (s *Screen) Order() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.order
}

func (s *Screen) SetSearchText(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.searchText = text
}

func (s *Screen) SearchText() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.searchText
}

func (s *Screen) SetContext(ctx map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.context = ctx
}

func (s *Screen) Context() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.context
}

func (s *Screen) SetActiveID(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeID = id
}

func (s *Screen) ActiveID() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeID
}

func (s *Screen) SetActiveIDs(ids []int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeIDs = ids
}

func (s *Screen) ActiveIDs() []int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeIDs
}

func (s *Screen) SetSelectedIDs(ids []int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.selectedIDs = ids
}

func (s *Screen) SelectedIDs() []int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.selectedIDs
}

func (s *Screen) SetReadonly(b bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readonly = b
}

func (s *Screen) Readonly() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.readonly
}

func (s *Screen) SetViewType(vt string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.viewType = vt
}

func (s *Screen) ViewType() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.viewType
}

func (s *Screen) SetViewIDs(ids []int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.viewIDs = ids
}

func (s *Screen) ViewIDs() []int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.viewIDs
}

func (s *Screen) SetFieldChilds(fc string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fieldChilds = fc
}

func (s *Screen) FieldChilds() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fieldChilds
}

func (s *Screen) Search() error {
	return nil
}

func (s *Screen) SearchFilter() error {
	return nil
}

func (s *Screen) SearchCount() (int, error) {
	return 0, nil
}

func (s *Screen) New() *Record {
	return s.group.New()
}

func (s *Screen) SaveCurrent() error {
	return s.group.Save()
}

func (s *Screen) Remove(r *Record) error {
	s.group.Remove(r)
	return nil
}

func (s *Screen) Copy(r *Record) *Record {
	return r.Copy(nil)
}

func (s *Screen) CancelCurrent() {
	s.group.Clear()
}

func (s *Screen) Reload() error {
	return nil
}

func (s *Screen) Display() error {
	return nil
}

func (s *Screen) DisplayNext() error {
	return nil
}

func (s *Screen) DisplayPrevious() error {
	return nil
}

func (s *Screen) Button(name string, records []*Record) error {
	return nil
}

func (s *Screen) ClientAction(action string) error {
	return nil
}

func (s *Screen) GetURL() string {
	return ""
}

func (s *Screen) Modified() bool {
	return s.group.IsModified()
}

func (s *Screen) SetRecords(records []*Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = records
}

func (s *Screen) Records() []*Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.records
}
