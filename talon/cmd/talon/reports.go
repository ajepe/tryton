package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajepe/talon/internal/orm"
	"github.com/ajepe/talon/internal/report"
	"github.com/ajepe/talon/internal/rpc"
)

var templateExtensions = map[string]bool{
	".odt": true, ".ods": true, ".fodt": true, ".fods": true, ".txt": true,
}

type ormRecordLoader struct {
	pool *orm.Pool
}

func (l *ormRecordLoader) LoadRecord(ctx context.Context, model string, id int, fields []string) (map[string]interface{}, error) {
	group, err := l.pool.NewGroup(model)
	if err != nil {
		return nil, err
	}
	if err := group.Load([]int{id}); err != nil {
		return nil, err
	}
	records := group.Records()
	if len(records) == 0 {
		return nil, fmt.Errorf("record %d of %s not found", id, model)
	}
	record := records[0]
	if err := record.Load(fields); err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		fields = record.LoadedFields()
	}
	return record.ExportData(fields), nil
}

func loadReportTemplates(registry *report.Registry, srv *rpc.Server, db *sql.DB, pool *orm.Pool, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !templateExtensions[ext] {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return err
		}
		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		template := &report.TemplateReport{
			ReportName: name,
			Model:      reportModel(db, name),
			Template:   content,
			Extension:  strings.TrimPrefix(ext, "."),
			Loader:     &ormRecordLoader{pool: pool},
		}
		if err := registry.Register(template); err != nil {
			return err
		}
		registry.RegisterRPC(srv, name)
	}
	return nil
}

func reportModel(db *sql.DB, reportName string) string {
	var model sql.NullString
	err := db.QueryRow(
		`SELECT model FROM ir_action_report WHERE report_name = $1 LIMIT 1`,
		reportName).Scan(&model)
	if err != nil || !model.Valid {
		return ""
	}
	return model.String
}
