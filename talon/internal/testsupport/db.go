package testsupport

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/ajepe/talon/internal/backend/postgresql"
	"github.com/ajepe/talon/internal/metadata"
	"github.com/ajepe/talon/internal/orm"
	"github.com/ajepe/talon/internal/schema"

	_ "github.com/ajepe/talon/internal/modules/account"
	_ "github.com/ajepe/talon/internal/modules/company"
	_ "github.com/ajepe/talon/internal/modules/country"
	_ "github.com/ajepe/talon/internal/modules/currency"
	_ "github.com/ajepe/talon/internal/modules/ir"
	_ "github.com/ajepe/talon/internal/modules/party"
	_ "github.com/ajepe/talon/internal/modules/product"
	_ "github.com/ajepe/talon/internal/modules/purchase"
	_ "github.com/ajepe/talon/internal/modules/res"
	_ "github.com/ajepe/talon/internal/modules/sale"
	_ "github.com/ajepe/talon/internal/modules/stock"
)

const defaultAdminURI = "postgres:///postgres?host=/var/run/postgresql"

func AdminURI() string {
	if uri := os.Getenv("TALON_TEST_URI"); uri != "" {
		return uri
	}
	return defaultAdminURI
}

func skipWithoutPostgres(t *testing.T, err error) {
	t.Helper()
	if os.Getenv("TALON_TEST_REQUIRED") != "" {
		t.Fatalf("PostgreSQL unavailable: %v", err)
	}
	t.Skipf("PostgreSQL unavailable, skipping: %v", err)
}

func openAdmin(t *testing.T) *sql.DB {
	t.Helper()
	admin, err := sql.Open("pgx", AdminURI())
	if err != nil {
		skipWithoutPostgres(t, err)
	}
	if err := admin.Ping(); err != nil {
		admin.Close()
		skipWithoutPostgres(t, err)
	}
	return admin
}

func databaseURI(base, name string) string {
	parsed, err := url.Parse(base)
	if err != nil {
		return base
	}
	parsed.Path = "/" + name
	return parsed.String()
}

func quoted(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func OpenDB(t *testing.T) *sql.DB {
	t.Helper()
	admin := openAdmin(t)

	name := fmt.Sprintf("talon_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE " + quoted(name)); err != nil {
		admin.Close()
		t.Fatalf("create database: %v", err)
	}

	db, err := sql.Open("pgx", databaseURI(AdminURI(), name))
	if err != nil {
		admin.Close()
		t.Fatalf("open database: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		admin.Close()
		t.Fatalf("ping database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
		admin.Exec("DROP DATABASE IF EXISTS " + quoted(name) + " WITH (FORCE)")
		admin.Close()
	})

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func Migrate(db *sql.DB) error {
	models := make([]*metadata.Model, 0)
	for _, name := range metadata.ListModels() {
		if m, ok := metadata.GetModel(name); ok {
			models = append(models, m)
		}
	}
	sync := schema.NewSync(db)
	if err := sync.Sync(models); err != nil {
		return err
	}
	orm.DefaultPool().SetDB(db)
	return nil
}

func DatabaseURI(t *testing.T, db *sql.DB) string {
	t.Helper()
	var name string
	if err := db.QueryRow("SELECT current_database()").Scan(&name); err != nil {
		t.Fatalf("current_database() error = %v", err)
	}
	return databaseURI(AdminURI(), name)
}

func TableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRow(
		`SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)`, table).Scan(&exists)
	if err != nil {
		t.Fatalf("table existence check: %v", err)
	}
	return exists
}

func ColumnExists(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRow(
		`SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2
		)`, table, column).Scan(&exists)
	if err != nil {
		t.Fatalf("column existence check: %v", err)
	}
	return exists
}
