package report

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"time"
)

type RecordLoader interface {
	LoadRecord(ctx context.Context, model string, id int, fields []string) (map[string]interface{}, error)
}

type TemplateReport struct {
	ReportName string
	Model      string
	Template   []byte
	Extension  string
	Filename   string
	Loader     RecordLoader
}

func (t *TemplateReport) Name() string { return t.ReportName }

func (t *TemplateReport) Execute(ctx context.Context, ids []int, data map[string]interface{}, rpcContext map[string]interface{}) (*Result, error) {
	renderContext := map[string]interface{}{}
	for key, value := range data {
		renderContext[key] = value
	}
	for key, value := range rpcContext {
		renderContext[key] = value
	}

	objects := []map[string]interface{}{}
	for _, id := range ids {
		record := map[string]interface{}{"id": id}
		if t.Loader != nil {
			loaded, err := t.Loader.LoadRecord(ctx, t.Model, id, nil)
			if err != nil {
				return nil, err
			}
			for key, value := range loaded {
				record[key] = value
			}
		}
		objects = append(objects, record)
	}
	renderContext["objects"] = objects
	if len(objects) > 0 {
		renderContext["object"] = objects[0]
	}

	content, err := RenderTemplate(t.Template, renderContext)
	if err != nil {
		return nil, err
	}

	extension := t.Extension
	if extension == "" {
		extension = detectExtension(t.Template)
	}
	filename := t.Filename
	if filename == "" {
		filename = t.ReportName
	}
	return &Result{
		Extension: extension,
		Content:   content,
		Filename:  filename,
	}, nil
}

func detectExtension(template []byte) string {
	if len(template) >= 2 && template[0] == 'P' && template[1] == 'K' {
		return "odt"
	}
	if bytes.Contains(template[:minInt(len(template), 4096)], []byte("<office:document")) {
		return "fodt"
	}
	return "txt"
}

func RenderTemplate(template []byte, renderContext map[string]interface{}) ([]byte, error) {
	if len(template) >= 2 && template[0] == 'P' && template[1] == 'K' {
		return renderZip(template, renderContext)
	}
	return renderXML(template, renderContext)
}

func renderZip(template []byte, renderContext map[string]interface{}) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(template), int64(len(template)))
	if err != nil {
		return nil, err
	}

	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, file := range reader.File {
		content, err := readZipFile(file)
		if err != nil {
			return nil, err
		}
		if path.Base(file.Name) == "content.xml" {
			content, err = renderXML(content, renderContext)
			if err != nil {
				return nil, err
			}
		}
		header := &zip.FileHeader{Name: file.Name, Method: zip.Deflate}
		header.SetMode(file.Mode())
		part, err := writer.CreateHeader(header)
		if err != nil {
			return nil, err
		}
		if _, err := part.Write(content); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func readZipFile(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func renderXML(content []byte, renderContext map[string]interface{}) ([]byte, error) {
	source := string(content)
	var output strings.Builder
	for {
		start := strings.Index(source, "${")
		if start < 0 {
			output.WriteString(source)
			break
		}
		end := strings.Index(source[start:], "}")
		if end < 0 {
			output.WriteString(source)
			break
		}
		end += start
		output.WriteString(source[:start])
		value, err := evalExpression(source[start+2:end], renderContext)
		if err != nil {
			return nil, err
		}
		output.WriteString(escapeXML(value))
		source = source[end+1:]
	}
	return []byte(output.String()), nil
}

var xmlEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
	"'", "&apos;",
)

func escapeXML(value string) string {
	return xmlEscaper.Replace(value)
}

func evalExpression(expression string, renderContext map[string]interface{}) (string, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return "", nil
	}

	if strings.HasPrefix(expression, "len(") && strings.HasSuffix(expression, ")") {
		inner := strings.TrimSpace(expression[4 : len(expression)-1])
		value, ok := lookup(inner, renderContext)
		if !ok {
			return "0", nil
		}
		switch v := value.(type) {
		case []map[string]interface{}:
			return strconv.Itoa(len(v)), nil
		case []interface{}:
			return strconv.Itoa(len(v)), nil
		case string:
			return strconv.Itoa(len(v)), nil
		default:
			return "0", nil
		}
	}

	value, ok := lookup(expression, renderContext)
	if !ok {
		return "", fmt.Errorf("unknown expression %q", expression)
	}
	return formatValue(value), nil
}

func lookup(expression string, renderContext map[string]interface{}) (interface{}, bool) {
	current := interface{}(renderContext)
	for _, segment := range parsePath(expression) {
		switch node := current.(type) {
		case map[string]interface{}:
			current = node[segment]
		case []map[string]interface{}:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(node) {
				return nil, false
			}
			current = node[index]
		default:
			return nil, false
		}
	}
	return current, current != nil
}

func parsePath(expression string) []string {
	expression = strings.ReplaceAll(expression, "['", ".")
	expression = strings.ReplaceAll(expression, "']", "")
	expression = strings.ReplaceAll(expression, `["`, ".")
	expression = strings.ReplaceAll(expression, `"]`, "")
	parts := strings.Split(expression, ".")
	result := []string{}
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func formatValue(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case bool:
		if v {
			return "True"
		}
		return "False"
	case time.Time:
		return v.Format("2006-01-02")
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
