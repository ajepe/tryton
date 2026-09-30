package orm

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"sync"

	"github.com/ajepe/talon/internal/metadata"
)

type Pool struct {
	mu     sync.RWMutex
	db     *sql.DB
	models map[string]*metadata.Model
	groups map[string]*Group
	screens map[string]*Screen
}

var defaultPool *Pool
var poolOnce sync.Once

func DefaultPool() *Pool {
	poolOnce.Do(func() {
		defaultPool = &Pool{
			models: map[string]*metadata.Model{},
			groups: map[string]*Group{},
			screens: map[string]*Screen{},
		}
	})
	return defaultPool
}

func (p *Pool) SetDB(db *sql.DB) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.db = db
}

func (p *Pool) DB() *sql.DB {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.db
}

func (p *Pool) RegisterModel(name string, m *metadata.Model) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.models[name]; exists {
		return fmt.Errorf("model %s already registered", name)
	}
	p.models[name] = m
	return nil
}

func (p *Pool) GetModel(name string) (*metadata.Model, error) {
	if m, ok := metadata.GetModel(name); ok {
		return m, nil
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	m, ok := p.models[name]
	if !ok {
		return nil, fmt.Errorf("model %s not found", name)
	}
	return m, nil
}

func (p *Pool) GetGroup(name string) (*Group, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	g, ok := p.groups[name]
	if !ok {
		return nil, fmt.Errorf("group %s not found", name)
	}
	return g, nil
}

func (p *Pool) NewGroup(name string) (*Group, error) {
	m, err := p.GetModel(name)
	if err != nil {
		return nil, err
	}
	g := NewGroup(m)
	g.db = p.DB()
	p.mu.Lock()
	p.groups[name] = g
	p.mu.Unlock()
	return g, nil
}

func (p *Pool) GetScreen(name string) (*Screen, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	s, ok := p.screens[name]
	if !ok {
		return nil, fmt.Errorf("screen %s not found", name)
	}
	return s, nil
}

func (p *Pool) NewScreen(name string) (*Screen, error) {
	m, err := p.GetModel(name)
	if err != nil {
		return nil, err
	}
	s := NewScreen(m)
	s.db = p.DB()
	p.mu.Lock()
	p.screens[name] = s
	p.mu.Unlock()
	return s, nil
}

func (p *Pool) ListModels() []string {
	seen := map[string]bool{}
	names := make([]string, 0, len(p.models))
	for _, name := range metadata.ListModels() {
		seen[name] = true
		names = append(names, name)
	}
	p.mu.RLock()
	for name := range p.models {
		if !seen[name] {
			names = append(names, name)
		}
	}
	p.mu.RUnlock()
	sort.Strings(names)
	return names
}

func (p *Pool) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.models = map[string]*metadata.Model{}
	p.groups = map[string]*Group{}
	p.screens = map[string]*Screen{}
}

func (p *Pool) Init(ctx context.Context, update bool, lang string) error {
	return nil
}

func (p *Pool) Upgrade(ctx context.Context, modules []string) error {
	return nil
}

func (p *Pool) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	db := p.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	return db.QueryContext(ctx, query, args...)
}

func (p *Pool) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	db := p.DB()
	if db == nil {
		return nil
	}
	return db.QueryRowContext(ctx, query, args...)
}

func (p *Pool) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	db := p.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	return db.ExecContext(ctx, query, args...)
}

func (p *Pool) Begin(ctx context.Context) (*sql.Tx, error) {
	db := p.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	return db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
}
