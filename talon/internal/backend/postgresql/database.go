package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const DriverName = "pgx"

type Database struct {
	db       *sql.DB
	database string
}

func NewDatabase(database, uri string) (*Database, error) {
	db, err := sql.Open(DriverName, uri)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &Database{db: db, database: database}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) DB() *sql.DB {
	return d.db
}

func (d *Database) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return d.db.QueryContext(ctx, query, args...)
}

func (d *Database) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return d.db.QueryRowContext(ctx, query, args...)
}

func (d *Database) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return d.db.ExecContext(ctx, query, args...)
}

func (d *Database) Begin(ctx context.Context) (*sql.Tx, error) {
	return d.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
}

func (d *Database) CreateTable(name string, columns []string) error {
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", name, joinColumns(columns))
	_, err := d.Exec(context.Background(), query)
	return err
}

func (d *Database) DropTable(name string) error {
	query := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", name)
	_, err := d.Exec(context.Background(), query)
	return err
}

func (d *Database) AddColumn(table, column, colType string) error {
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s %s", table, column, colType)
	_, err := d.Exec(context.Background(), query)
	return err
}

func (d *Database) DropColumn(table, column string) error {
	query := fmt.Sprintf("ALTER TABLE %s DROP COLUMN IF EXISTS %s", table, column)
	_, err := d.Exec(context.Background(), query)
	return err
}

func (d *Database) CreateIndex(name, table, column string) error {
	query := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)", name, table, column)
	_, err := d.Exec(context.Background(), query)
	return err
}

func (d *Database) DropIndex(name string) error {
	query := fmt.Sprintf("DROP INDEX IF EXISTS %s", name)
	_, err := d.Exec(context.Background(), query)
	return err
}

func (d *Database) CreateSequence(name string) error {
	query := fmt.Sprintf("CREATE SEQUENCE IF NOT EXISTS %s", name)
	_, err := d.Exec(context.Background(), query)
	return err
}

func (d *Database) DropSequence(name string) error {
	query := fmt.Sprintf("DROP SEQUENCE IF EXISTS %s", name)
	_, err := d.Exec(context.Background(), query)
	return err
}

func (d *Database) CurrentSequenceValue(name string) (int, error) {
	var val int
	query := fmt.Sprintf("SELECT last_value FROM %s", name)
	err := d.QueryRow(context.Background(), query).Scan(&val)
	return val, err
}

func (d *Database) SetSequenceValue(name string, val int) error {
	query := fmt.Sprintf("SELECT setval(%s, %d)", name, val)
	_, err := d.Exec(context.Background(), query)
	return err
}

func (d *Database) Insert(table string, columns []string, values []interface{}) (int, error) {
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id",
		table, joinColumns(columns), joinPlaceholders(placeholders))
	var id int
	err := d.QueryRow(context.Background(), query, values...).Scan(&id)
	return id, err
}

func (d *Database) Update(table string, columns []string, values []interface{}, where string, whereArgs ...interface{}) error {
	setParts := make([]string, len(columns))
	for i, col := range columns {
		setParts[i] = fmt.Sprintf("%s = $%d", col, i+1)
	}
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", table, joinParts(setParts), where)
	args := append(values, whereArgs...)
	_, err := d.Exec(context.Background(), query, args...)
	return err
}

func (d *Database) Delete(table, where string, args ...interface{}) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s", table, where)
	_, err := d.Exec(context.Background(), query, args...)
	return err
}

func (d *Database) Select(table, where string, args ...interface{}) (*sql.Rows, error) {
	query := fmt.Sprintf("SELECT * FROM %s", table)
	if where != "" {
		query += " WHERE " + where
	}
	return d.Query(context.Background(), query, args...)
}

func (d *Database) SelectCount(table, where string, args ...interface{}) (int, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	if where != "" {
		query += " WHERE " + where
	}
	var count int
	err := d.QueryRow(context.Background(), query, args...).Scan(&count)
	return count, err
}

func (d *Database) Search(table, where string, args ...interface{}) (*sql.Rows, error) {
	return d.Select(table, where, args...)
}

func (d *Database) IsolationLevel() string {
	return "REPEATABLE READ"
}

func (d *Database) SetIsolationLevel(level string) error {
	return nil
}

func (d *Database) Now() time.Time {
	return time.Now().UTC()
}

func (d *Database) Database() string {
	return d.database
}

func joinColumns(cols []string) string {
	return joinParts(cols)
}

func joinParts(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ", "
		}
		result += p
	}
	return result
}

func joinPlaceholders(parts []string) string {
	return joinParts(parts)
}
