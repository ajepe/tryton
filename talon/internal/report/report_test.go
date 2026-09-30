package report

import (
	"context"
	"strings"
	"testing"

	"github.com/ajepe/talon/internal/rpc"
)

func TestRegistryRegisterAndGet(t *testing.T) {
	registry := NewRegistry()
	rep := &FuncReport{
		ReportName: "party.party.label",
		Ext:        "txt",
		Render: func(ctx context.Context, ids []int, data map[string]interface{}, rpcContext map[string]interface{}) ([]byte, error) {
			return []byte("labels"), nil
		},
	}
	if err := registry.Register(rep); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := registry.Register(rep); err == nil {
		t.Error("Register() expected error for duplicate")
	}
	if len(registry.Names()) != 1 {
		t.Errorf("Names() = %v, want 1 report", registry.Names())
	}

	result, err := registry.Execute(context.Background(), "party.party.label", []int{1}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if string(result.Content) != "labels" {
		t.Errorf("content = %q, want labels", result.Content)
	}
	if result.Extension != "txt" {
		t.Errorf("extension = %q, want txt", result.Extension)
	}
}

func TestRegistryExecuteUnknown(t *testing.T) {
	registry := NewRegistry()
	if _, err := registry.Execute(context.Background(), "missing", nil, nil, nil); err == nil {
		t.Error("Execute() expected error for unknown report")
	}
}

func TestCSVReport(t *testing.T) {
	rep := &CSVReport{
		ReportName: "party.party.export",
		Columns: []CSVColumn{
			{Header: "Name", Value: func(row map[string]interface{}) string {
				return row["name"].(string)
			}},
			{Header: "Code", Value: func(row map[string]interface{}) string {
				return row["code"].(string)
			}},
		},
		Rows: func(ctx context.Context, ids []int, data map[string]interface{}, rpcContext map[string]interface{}) ([]map[string]interface{}, error) {
			return []map[string]interface{}{
				{"name": "Acme", "code": "ACME"},
				{"name": "Globex", "code": "GLBX"},
			}, nil
		},
		Filename: "parties.csv",
	}

	result, err := rep.Execute(context.Background(), nil, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Extension != "csv" {
		t.Errorf("extension = %q, want csv", result.Extension)
	}
	lines := strings.Split(strings.TrimSpace(string(result.Content)), "\n")
	if len(lines) != 3 {
		t.Fatalf("csv has %d lines, want 3", len(lines))
	}
	if lines[0] != "Name,Code" {
		t.Errorf("header = %q, want Name,Code", lines[0])
	}
	if lines[1] != "Acme,ACME" {
		t.Errorf("row = %q, want Acme,ACME", lines[1])
	}
}

func TestRegistryRegisterRPC(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&FuncReport{
		ReportName: "party.party.label",
		Ext:        "txt",
		Render: func(ctx context.Context, ids []int, data map[string]interface{}, rpcContext map[string]interface{}) ([]byte, error) {
			return []byte("labels"), nil
		},
	})
	srv := rpc.NewServer()
	registry.RegisterRPC(srv, "party.party.label")

	result, err := srv.Call(context.Background(), "report.party.party.label.execute",
		[]interface{}{[]interface{}{1.0}, map[string]interface{}{}})
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	arr, ok := result.([]interface{})
	if !ok || len(arr) != 4 {
		t.Fatalf("result = %v, want 4-element array", result)
	}
	if arr[0] != "txt" {
		t.Errorf("extension = %v, want txt", arr[0])
	}
	if string(arr[1].([]byte)) != "labels" {
		t.Errorf("content = %q, want labels", arr[1])
	}
}

func TestFuncReportDefaults(t *testing.T) {
	rep := &FuncReport{
		ReportName: "test",
		Render: func(ctx context.Context, ids []int, data map[string]interface{}, rpcContext map[string]interface{}) ([]byte, error) {
			return nil, nil
		},
	}
	result, err := rep.Execute(context.Background(), nil, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Extension != "txt" {
		t.Errorf("default extension = %q, want txt", result.Extension)
	}
}
