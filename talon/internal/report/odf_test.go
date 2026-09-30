package report

import (
	"archive/zip"
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRenderFlatTemplate(t *testing.T) {
	template := []byte(`<office:document><text:p>${object.name}</text:p><text:p>${len(objects)}</text:p></office:document>`)
	context := map[string]interface{}{
		"object":  map[string]interface{}{"name": "Acme & Co"},
		"objects": []map[string]interface{}{{"name": "Acme & Co"}},
	}
	rendered, err := RenderTemplate(template, context)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}
	if !strings.Contains(string(rendered), "Acme &amp; Co") {
		t.Errorf("rendered = %q, want escaped name", rendered)
	}
	if !strings.Contains(string(rendered), ">1<") {
		t.Errorf("rendered = %q, want object count", rendered)
	}
}

func TestRenderZipTemplate(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	part, _ := writer.Create("content.xml")
	part.Write([]byte(`<office:document><text:p>${object.code}</text:p></office:document>`))
	part, _ = writer.Create("styles.xml")
	part.Write([]byte(`<styles/>`))
	writer.Close()

	rendered, err := RenderTemplate(buffer.Bytes(), map[string]interface{}{
		"object": map[string]interface{}{"code": "ACME"},
	})
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(rendered), int64(len(rendered)))
	if err != nil {
		t.Fatalf("zip.NewReader() error = %v", err)
	}
	foundContent := false
	foundStyles := false
	for _, file := range reader.File {
		content, _ := readZipFile(file)
		switch file.Name {
		case "content.xml":
			foundContent = true
			if !strings.Contains(string(content), "ACME") {
				t.Errorf("content.xml = %q, want ACME", content)
			}
		case "styles.xml":
			foundStyles = true
		}
	}
	if !foundContent || !foundStyles {
		t.Errorf("zip entries missing: content=%v styles=%v", foundContent, foundStyles)
	}
}

func TestRenderUnknownExpression(t *testing.T) {
	template := []byte(`<text>${object.missing}</text>`)
	_, err := RenderTemplate(template, map[string]interface{}{
		"object": map[string]interface{}{"name": "x"},
	})
	if err == nil {
		t.Error("RenderTemplate() expected error for unknown expression")
	}
}

type stubLoader struct {
	records map[int]map[string]interface{}
}

func (s *stubLoader) LoadRecord(ctx context.Context, model string, id int, fields []string) (map[string]interface{}, error) {
	return s.records[id], nil
}

func TestTemplateReportExecute(t *testing.T) {
	report := &TemplateReport{
		ReportName: "party.label",
		Model:      "party.party",
		Template:   []byte(`<text>${object.name} (${object.code})</text>`),
		Loader: &stubLoader{records: map[int]map[string]interface{}{
			1: {"name": "Acme", "code": "ACME"},
		}},
	}
	result, err := report.Execute(context.Background(), []int{1}, nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Extension != "txt" {
		t.Errorf("extension = %q, want txt", result.Extension)
	}
	if !strings.Contains(string(result.Content), "Acme (ACME)") {
		t.Errorf("content = %q", result.Content)
	}
	if result.Filename != "party.label" {
		t.Errorf("filename = %q", result.Filename)
	}
}

func TestFormatValue(t *testing.T) {
	if got := formatValue(nil); got != "" {
		t.Errorf("formatValue(nil) = %q", got)
	}
	if got := formatValue(true); got != "True" {
		t.Errorf("formatValue(true) = %q", got)
	}
	if got := formatValue(1.5); got != "1.5" {
		t.Errorf("formatValue(1.5) = %q", got)
	}
}
