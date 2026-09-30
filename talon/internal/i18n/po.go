package i18n

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type POEntry struct {
	Msgctxt string
	Msgid   string
	Msgstr  string
	Fuzzy   bool
}

func ParsePO(reader io.Reader) ([]POEntry, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	entries := []POEntry{}
	current := POEntry{}
	field := ""
	fuzzy := false
	started := false

	flush := func() {
		if started {
			current.Fuzzy = fuzzy
			entries = append(entries, current)
			fuzzy = false
		}
		current = POEntry{}
		started = false
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "":
			flush()
			field = ""
		case strings.HasPrefix(line, "#"):
			if strings.HasPrefix(line, "#,") && strings.Contains(line, "fuzzy") {
				fuzzy = true
			}
		case strings.HasPrefix(line, "msgctxt "):
			flush()
			field = "msgctxt"
			started = true
			current.Msgctxt = decodePOString(strings.TrimSpace(line[len("msgctxt "):]))
		case strings.HasPrefix(line, "msgid "):
			if field == "msgstr" {
				flush()
				field = ""
			}
			field = "msgid"
			started = true
			current.Msgid = decodePOString(strings.TrimSpace(line[len("msgid "):]))
		case strings.HasPrefix(line, "msgstr "):
			field = "msgstr"
			current.Msgstr = decodePOString(strings.TrimSpace(line[len("msgstr "):]))
		case strings.HasPrefix(line, "msgid_plural "):
			field = "msgid_plural"
		case strings.HasPrefix(line, "msgstr["):
			field = "msgstr"
		case strings.HasPrefix(line, `"`):
			text := decodePOString(line)
			switch field {
			case "msgctxt":
				current.Msgctxt += text
			case "msgid":
				current.Msgid += text
			case "msgstr":
				current.Msgstr += text
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	flush()
	return entries, nil
}

func decodePOString(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
	}
	decoded, err := strconv.Unquote(`"` + value + `"`)
	if err != nil {
		return value
	}
	return decoded
}

type POLoader struct {
	db     *sql.DB
	module string
}

func NewPOLoader(db *sql.DB, module string) *POLoader {
	return &POLoader{db: db, module: module}
}

type poTranslation struct {
	Name     string
	Type     string
	Src      string
	Value    string
	Resource string
	Fuzzy    bool
}

func parsePOContext(entry POEntry) (*poTranslation, error) {
	parts := strings.SplitN(entry.Msgctxt, ":", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid msgctxt %q", entry.Msgctxt)
	}
	translation := &poTranslation{
		Type:  parts[0],
		Name:  parts[1],
		Src:   entry.Msgid,
		Value: entry.Msgstr,
		Fuzzy: entry.Fuzzy,
	}
	if len(parts) == 3 {
		translation.Resource = parts[2]
	}
	return translation, nil
}

func (l *POLoader) ImportPO(ctx context.Context, lang string, entries []POEntry) (int, error) {
	imported := 0
	for _, entry := range entries {
		if entry.Msgid == "" {
			continue
		}
		translation, err := parsePOContext(entry)
		if err != nil {
			return imported, err
		}
		resID, err := l.resolveResID(ctx, translation.Resource)
		if err != nil {
			return imported, err
		}
		if entry.Msgstr == "" {
			if err := l.delete(ctx, lang, translation, resID); err != nil {
				return imported, err
			}
			continue
		}
		if err := l.upsert(ctx, lang, translation, resID); err != nil {
			return imported, err
		}
		imported++
	}
	return imported, nil
}

func (l *POLoader) resolveResID(ctx context.Context, resource string) (*int, error) {
	if resource == "" {
		return nil, nil
	}
	parts := strings.SplitN(resource, ".", 2)
	if len(parts) != 2 || parts[1] == "" {
		return nil, nil
	}
	var dbID sql.NullInt64
	err := l.db.QueryRowContext(ctx,
		`SELECT db_id FROM ir_model_data WHERE module = $1 AND fs_id = $2`,
		parts[0], parts[1]).Scan(&dbID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !dbID.Valid {
		return nil, nil
	}
	id := int(dbID.Int64)
	return &id, nil
}

func (l *POLoader) delete(ctx context.Context, lang string, translation *poTranslation, resID *int) error {
	_, err := l.db.ExecContext(ctx,
		`DELETE FROM ir_translation
		 WHERE name = $1 AND lang = $2 AND type = $3 AND module = $4
		   AND COALESCE(res_id, 0) = COALESCE($5, 0)`,
		translation.Name, lang, translation.Type, l.module, resID)
	return err
}

func (l *POLoader) upsert(ctx context.Context, lang string, translation *poTranslation, resID *int) error {
	var existingID int
	err := l.db.QueryRowContext(ctx,
		`SELECT id FROM ir_translation
		 WHERE name = $1 AND lang = $2 AND type = $3 AND module = $4
		   AND COALESCE(res_id, 0) = COALESCE($5, 0)
		 LIMIT 1`,
		translation.Name, lang, translation.Type, l.module, resID).Scan(&existingID)
	switch {
	case err == sql.ErrNoRows:
		_, err = l.db.ExecContext(ctx,
			`INSERT INTO ir_translation
			 (name, lang, type, src, value, module, res_id, fuzzy)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			translation.Name, lang, translation.Type, translation.Src,
			translation.Value, l.module, resID, translation.Fuzzy)
		return err
	case err != nil:
		return err
	default:
		_, err = l.db.ExecContext(ctx,
			`UPDATE ir_translation
			 SET src = $1, value = $2, fuzzy = $3
			 WHERE id = $4`,
			translation.Src, translation.Value, translation.Fuzzy, existingID)
		return err
	}
}
