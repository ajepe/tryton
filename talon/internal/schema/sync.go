package schema

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/ajepe/talon/internal/metadata"
)

type Sync struct {
	db *sql.DB
}

func NewSync(db *sql.DB) *Sync {
	return &Sync{db: db}
}

func Quote(name string) string {
	if strings.HasPrefix(name, `"`) && strings.HasSuffix(name, `"`) {
		return name
	}
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func (s *Sync) CreateTable(model *metadata.Model) error {
	table := Quote(model.TableName())
	columns := []string{
		"id serial PRIMARY KEY",
		"create_uid integer REFERENCES res_user(id)",
		"write_uid integer REFERENCES res_user(id)",
		"create_date timestamp with time zone",
		"write_date timestamp with time zone",
		"_timestamp jsonb",
	}
	for _, name := range model.FieldNames() {
		field := model.Fields[name]
		if field.IsX2M() || isStandardColumn(field.ColumnName()) {
			continue
		}
		col := fmt.Sprintf("%s %s", Quote(field.ColumnName()), field.SQLType())
		if field.Required {
			col += " NOT NULL"
		}
		if field.Unique {
			col += " UNIQUE"
		}
		columns = append(columns, col)
	}
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", table, strings.Join(columns, ", "))
	_, err := s.db.ExecContext(context.Background(), query)
	return err
}

func (s *Sync) AlterTable(model *metadata.Model) error {
	for _, name := range model.FieldNames() {
		field := model.Fields[name]
		if field.IsX2M() || isStandardColumn(field.ColumnName()) {
			continue
		}
		query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s %s",
			Quote(model.TableName()), Quote(field.ColumnName()), field.SQLType())
		if _, err := s.db.ExecContext(context.Background(), query); err != nil {
			return err
		}
	}
	return nil
}

func isStandardColumn(name string) bool {
	switch name {
	case "create_uid", "create_date", "write_uid", "write_date", "_timestamp":
		return true
	}
	return false
}

func (s *Sync) DropTable(model *metadata.Model) error {
	query := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", Quote(model.TableName()))
	_, err := s.db.ExecContext(context.Background(), query)
	return err
}

func (s *Sync) CreateIndex(model *metadata.Model, field string) error {
	f, ok := model.Fields[field]
	if !ok {
		return fmt.Errorf("field %s not found on %s", field, model.Name)
	}
	name := fmt.Sprintf("%s_%s_idx", model.TableName(), field)
	if len(name) > 63 {
		name = name[:63]
	}
	query := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)",
		Quote(name), Quote(model.TableName()), Quote(f.ColumnName()))
	_, err := s.db.ExecContext(context.Background(), query)
	return err
}

func (s *Sync) DropIndex(model *metadata.Model, field string) error {
	name := fmt.Sprintf("%s_%s_idx", model.TableName(), field)
	query := fmt.Sprintf("DROP INDEX IF EXISTS %s", Quote(name))
	_, err := s.db.ExecContext(context.Background(), query)
	return err
}

func (s *Sync) CreateSequence(name string) error {
	query := fmt.Sprintf("CREATE SEQUENCE IF NOT EXISTS %s", Quote(name))
	_, err := s.db.ExecContext(context.Background(), query)
	return err
}

func (s *Sync) DropSequence(name string) error {
	query := fmt.Sprintf("DROP SEQUENCE IF EXISTS %s", Quote(name))
	_, err := s.db.ExecContext(context.Background(), query)
	return err
}

func (s *Sync) CreateRelationTable(model *metadata.Model, field *metadata.Field) error {
	relation, ok := metadata.GetModel(field.Relation)
	if !ok {
		return fmt.Errorf("relation model %s not found", field.Relation)
	}
	origin := relationColumn(model.Name)
	target := relationColumn(field.Relation)
	table := Quote(model.TableName() + "-" + relation.TableName())
	query := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s (
			%s integer NOT NULL REFERENCES %s(id) ON DELETE CASCADE,
			%s integer NOT NULL REFERENCES %s(id) ON DELETE CASCADE,
			UNIQUE (%s, %s)
		)`,
		table,
		Quote(origin), Quote(model.TableName()),
		Quote(target), Quote(relation.TableName()),
		Quote(origin), Quote(target),
	)
	_, err := s.db.ExecContext(context.Background(), query)
	return err
}

func relationColumn(name string) string {
	if idx := strings.Index(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	return strings.ReplaceAll(name, ".", "_")
}

func sortModels(models []*metadata.Model) []*metadata.Model {
	sorted := append([]*metadata.Model(nil), models...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Name == "res.user" {
			return true
		}
		if sorted[j].Name == "res.user" {
			return false
		}
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}

func (s *Sync) Sync(models []*metadata.Model) error {
	for _, m := range sortModels(models) {
		if err := s.CreateTable(m); err != nil {
			return fmt.Errorf("create table %s: %w", m.Name, err)
		}
		if err := s.AlterTable(m); err != nil {
			return fmt.Errorf("alter table %s: %w", m.Name, err)
		}
	}
	for _, m := range models {
		for _, name := range m.FieldNames() {
			field := m.Fields[name]
			if field.Type == metadata.FieldMany2Many {
				if err := s.CreateRelationTable(m, field); err != nil {
					return fmt.Errorf("relation table %s.%s: %w", m.Name, name, err)
				}
			}
			if field.IsM2O() || field.Index {
				if err := s.CreateIndex(m, name); err != nil {
					return fmt.Errorf("index %s.%s: %w", m.Name, name, err)
				}
			}
		}
	}
	return nil
}

func (s *Sync) Migrate(module string, version string) error {
	return nil
}

func (s *Sync) Init(ctx context.Context) error {
	return nil
}

func (s *Sync) Upgrade(ctx context.Context, modules []string) error {
	return nil
}
