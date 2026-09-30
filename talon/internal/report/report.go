package report

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"sync"

	"github.com/ajepe/talon/internal/rpc"
)

type Result struct {
	Extension   string
	Content     []byte
	DirectPrint bool
	Filename    string
}

type Report interface {
	Name() string
	Execute(ctx context.Context, ids []int, data map[string]interface{}, rpcContext map[string]interface{}) (*Result, error)
}

type Registry struct {
	mu      sync.RWMutex
	reports map[string]Report
}

func NewRegistry() *Registry {
	return &Registry{reports: map[string]Report{}}
}

func (r *Registry) Register(rep Report) error {
	if rep == nil || rep.Name() == "" {
		return fmt.Errorf("report requires a name")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.reports[rep.Name()]; exists {
		return fmt.Errorf("report %s already registered", rep.Name())
	}
	r.reports[rep.Name()] = rep
	return nil
}

func (r *Registry) Get(name string) (Report, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rep, ok := r.reports[name]
	return rep, ok
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.reports))
	for name := range r.reports {
		names = append(names, name)
	}
	return names
}

func (r *Registry) Execute(ctx context.Context, name string, ids []int, data map[string]interface{}, rpcContext map[string]interface{}) (*Result, error) {
	rep, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("report %s not found", name)
	}
	return rep.Execute(ctx, ids, data, rpcContext)
}

func (r *Registry) RegisterRPC(srv *rpc.Server, name string) {
	srv.RegisterReportMethods(name, func(ctx context.Context, params []interface{}) (interface{}, error) {
		ids := []int{}
		if len(params) > 0 {
			if raw, ok := params[0].([]interface{}); ok {
				for _, v := range raw {
					switch n := v.(type) {
					case float64:
						ids = append(ids, int(n))
					case int:
						ids = append(ids, n)
					}
				}
			}
		}
		data := map[string]interface{}{}
		if len(params) > 1 {
			if m, ok := params[1].(map[string]interface{}); ok {
				data = m
			}
		}
		rpcContext := map[string]interface{}{}
		if len(params) > 2 {
			if m, ok := params[2].(map[string]interface{}); ok {
				rpcContext = m
			}
		}
		result, err := r.Execute(ctx, name, ids, data, rpcContext)
		if err != nil {
			return nil, &rpc.UserError{Message: err.Error()}
		}
		return []interface{}{result.Extension, result.Content, result.DirectPrint, result.Filename}, nil
	})
}

type FuncReport struct {
	ReportName string
	Ext        string
	Print      bool
	File       string
	Render     func(ctx context.Context, ids []int, data map[string]interface{}, rpcContext map[string]interface{}) ([]byte, error)
}

func (f *FuncReport) Name() string { return f.ReportName }

func (f *FuncReport) Execute(ctx context.Context, ids []int, data map[string]interface{}, rpcContext map[string]interface{}) (*Result, error) {
	content, err := f.Render(ctx, ids, data, rpcContext)
	if err != nil {
		return nil, err
	}
	extension := f.Ext
	if extension == "" {
		extension = "txt"
	}
	return &Result{
		Extension:   extension,
		Content:     content,
		DirectPrint: f.Print,
		Filename:    f.File,
	}, nil
}

type CSVColumn struct {
	Header string
	Value  func(row map[string]interface{}) string
}

type CSVReport struct {
	ReportName string
	Columns    []CSVColumn
	Rows       func(ctx context.Context, ids []int, data map[string]interface{}, rpcContext map[string]interface{}) ([]map[string]interface{}, error)
	Filename   string
}

func (c *CSVReport) Name() string { return c.ReportName }

func (c *CSVReport) Execute(ctx context.Context, ids []int, data map[string]interface{}, rpcContext map[string]interface{}) (*Result, error) {
	rows, err := c.Rows(ctx, ids, data, rpcContext)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	header := make([]string, len(c.Columns))
	for i, column := range c.Columns {
		header[i] = column.Header
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}
	for _, row := range rows {
		record := make([]string, len(c.Columns))
		for i, column := range c.Columns {
			if column.Value != nil {
				record[i] = column.Value(row)
			}
		}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return &Result{
		Extension: "csv",
		Content:   buf.Bytes(),
		Filename:  c.Filename,
	}, nil
}
