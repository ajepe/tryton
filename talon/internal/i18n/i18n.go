package i18n

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Language struct {
	Code         string
	Name         string
	Direction    string
	DateFormat   string
	DecimalPoint string
	ThousandsSep string
}

func DefaultLanguage() *Language {
	return &Language{
		Code:         "en",
		Name:         "English",
		Direction:    "ltr",
		DateFormat:   "%m/%d/%Y",
		DecimalPoint: ".",
		ThousandsSep: ",",
	}
}

type Manager struct {
	db    *sql.DB
	mu    sync.RWMutex
	cache map[string]string
	langs map[string]*Language
}

func NewManager(db *sql.DB) *Manager {
	return &Manager{
		db:    db,
		cache: map[string]string{},
		langs: map[string]*Language{},
	}
}

func parentLanguage(code string) string {
	if idx := strings.IndexAny(code, "_@"); idx > 0 {
		return code[:idx]
	}
	return ""
}

func (m *Manager) Translate(ctx context.Context, name, lang, source string) string {
	if source == "" {
		return source
	}
	if lang == "" {
		return source
	}
	key := name + "\x00" + lang + "\x00" + source
	m.mu.RLock()
	if value, ok := m.cache[key]; ok {
		m.mu.RUnlock()
		return value
	}
	m.mu.RUnlock()

	value := ""
	for code := lang; code != ""; code = parentLanguage(code) {
		if m.db == nil {
			break
		}
		err := m.db.QueryRowContext(ctx,
			`SELECT value FROM ir_translation
			 WHERE name = $1 AND lang = $2 AND src = $3
			   AND COALESCE(fuzzy, false) = false
			   AND res_id IS NULL
			   AND value IS NOT NULL AND value <> ''
			 LIMIT 1`,
			name, code, source).Scan(&value)
		if err == nil && value != "" {
			break
		}
		value = ""
	}
	if value == "" {
		value = source
	}
	m.mu.Lock()
	m.cache[key] = value
	m.mu.Unlock()
	return value
}

func (m *Manager) Language(ctx context.Context, code string) (*Language, error) {
	if code == "" {
		return DefaultLanguage(), nil
	}
	m.mu.RLock()
	if lang, ok := m.langs[code]; ok {
		m.mu.RUnlock()
		return lang, nil
	}
	m.mu.RUnlock()

	if m.db == nil {
		return DefaultLanguage(), nil
	}

	lang := DefaultLanguage()
	var name, direction, dateFormat, decimalPoint, thousandsSep sql.NullString
	err := m.db.QueryRowContext(ctx,
		`SELECT code, name, direction, date_format, decimal_point, thousands_sep
		 FROM ir_lang WHERE code = $1`, code).Scan(
		&lang.Code, &name, &direction, &dateFormat, &decimalPoint, &thousandsSep)
	if err == sql.ErrNoRows {
		return lang, nil
	}
	if err != nil {
		return nil, err
	}
	if name.Valid {
		lang.Name = name.String
	}
	if direction.Valid {
		lang.Direction = direction.String
	}
	if dateFormat.Valid {
		lang.DateFormat = dateFormat.String
	}
	if decimalPoint.Valid {
		lang.DecimalPoint = decimalPoint.String
	}
	if thousandsSep.Valid {
		lang.ThousandsSep = thousandsSep.String
	}

	m.mu.Lock()
	m.langs[code] = lang
	m.mu.Unlock()
	return lang, nil
}

func (m *Manager) List(ctx context.Context) ([]*Language, error) {
	if m.db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	rows, err := m.db.QueryContext(ctx,
		`SELECT code, name, direction, date_format, decimal_point, thousands_sep
		 FROM ir_lang ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	languages := []*Language{}
	for rows.Next() {
		lang := &Language{}
		var name, direction, dateFormat, decimalPoint, thousandsSep sql.NullString
		if err := rows.Scan(&lang.Code, &name, &direction, &dateFormat, &decimalPoint, &thousandsSep); err != nil {
			return nil, err
		}
		if name.Valid {
			lang.Name = name.String
		}
		if direction.Valid {
			lang.Direction = direction.String
		}
		if dateFormat.Valid {
			lang.DateFormat = dateFormat.String
		}
		if decimalPoint.Valid {
			lang.DecimalPoint = decimalPoint.String
		}
		if thousandsSep.Valid {
			lang.ThousandsSep = thousandsSep.String
		}
		languages = append(languages, lang)
	}
	return languages, rows.Err()
}

func (m *Manager) Invalidate() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache = map[string]string{}
	m.langs = map[string]*Language{}
}

func FormatNumber(value float64, lang *Language, decimals int) string {
	if lang == nil {
		lang = DefaultLanguage()
	}
	decimalPoint := lang.DecimalPoint
	if decimalPoint == "" {
		decimalPoint = "."
	}
	thousandsSep := lang.ThousandsSep
	if thousandsSep == "" {
		thousandsSep = ","
	}

	negative := value < 0
	if negative {
		value = -value
	}
	formatted := strconv.FormatFloat(value, 'f', decimals, 64)
	parts := strings.SplitN(formatted, ".", 2)

	integer := parts[0]
	var grouped strings.Builder
	for i, digit := range integer {
		if i > 0 && (len(integer)-i)%3 == 0 {
			grouped.WriteString(thousandsSep)
		}
		grouped.WriteRune(digit)
	}

	result := grouped.String()
	if len(parts) == 2 {
		result += decimalPoint + parts[1]
	}
	if negative {
		result = "-" + result
	}
	return result
}

func FormatDate(value time.Time, lang *Language) string {
	if lang == nil {
		lang = DefaultLanguage()
	}
	return value.Format(StrftimeToGo(lang.DateFormat))
}

func StrftimeToGo(format string) string {
	if format == "" {
		return "2006-01-02"
	}
	var b strings.Builder
	for i := 0; i < len(format); i++ {
		if format[i] != '%' || i+1 >= len(format) {
			b.WriteByte(format[i])
			continue
		}
		i++
		switch format[i] {
		case 'Y':
			b.WriteString("2006")
		case 'y':
			b.WriteString("06")
		case 'm':
			b.WriteString("01")
		case 'd':
			b.WriteString("02")
		case 'e':
			b.WriteString("_2")
		case 'H':
			b.WriteString("15")
		case 'I':
			b.WriteString("03")
		case 'M':
			b.WriteString("04")
		case 'S':
			b.WriteString("05")
		case 'p':
			b.WriteString("PM")
		case 'B':
			b.WriteString("January")
		case 'b', 'h':
			b.WriteString("Jan")
		case 'A':
			b.WriteString("Monday")
		case 'a':
			b.WriteString("Mon")
		case 'j':
			b.WriteString("002")
		case 'Z':
			b.WriteString("MST")
		case 'z':
			b.WriteString("-0700")
		case '%':
			b.WriteByte('%')
		default:
			b.WriteByte('%')
			b.WriteByte(format[i])
		}
	}
	return b.String()
}

func LanguageFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if contextMap, ok := ctx.Value("context").(map[string]interface{}); ok {
		if language, ok := contextMap["language"].(string); ok {
			return language
		}
	}
	return ""
}
