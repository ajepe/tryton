package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/ajepe/talon/internal/orm"
	"github.com/ajepe/talon/internal/report"
)

type integrationLoader struct {
	pool *orm.Pool
}

func (l integrationLoader) LoadRecord(ctx context.Context, model string, id int, fields []string) (map[string]interface{}, error) {
	group, err := l.pool.NewGroup(model)
	if err != nil {
		return nil, err
	}
	if err := group.Load([]int{id}); err != nil {
		return nil, err
	}
	record := group.Records()[0]
	if err := record.Load(fields); err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		fields = record.LoadedFields()
	}
	return record.ExportData(fields), nil
}

func TestReportExecuteOverRPC(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	if _, err := h.sessions.CreateUser(ctx, "reportadmin", "pass", "Report Admin", true); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	var partyID int
	if err := h.db.QueryRowContext(ctx,
		`INSERT INTO party_party (name, code) VALUES ('Report Acme', 'RPT') RETURNING id`,
	).Scan(&partyID); err != nil {
		t.Fatalf("insert party error = %v", err)
	}

	registry := report.NewRegistry()
	template := &report.TemplateReport{
		ReportName: "party.party.label",
		Model:      "party.party",
		Template:   []byte(`<text>${object.name} [${object.code}]</text>`),
		Loader:     integrationLoader{pool: orm.DefaultPool()},
	}
	if err := registry.Register(template); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	registry.RegisterRPC(h.srv, "party.party.label")

	body := `{"id": 1, "method": "report.party.party.label.execute", "params": [[` +
		itoaInt(partyID) + `], {}, {}]}`
	response := authenticatedRPC(t, h, "reportadmin", "pass", body)
	if !strings.Contains(response, `"result":["txt"`) {
		t.Fatalf("response = %s, want txt report result", response)
	}
	if !strings.Contains(response, `"__class__":"bytes"`) {
		t.Errorf("response = %s, want tagged bytes content", response)
	}
	if strings.Contains(response, "error") {
		t.Errorf("response = %s, want success", response)
	}
}
