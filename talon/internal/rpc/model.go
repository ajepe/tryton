package rpc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ajepe/talon/internal/metadata"
	"github.com/ajepe/talon/internal/orm"
	"github.com/ajepe/talon/internal/pyson"
	"github.com/ajepe/talon/internal/xml"
)

type Translator interface {
	Translate(ctx context.Context, name, lang, source string) string
}

type ModuleHooks interface {
	OnChange(ctx context.Context, model string, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error)
	OnChangeWith(ctx context.Context, model string, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error)
	Validate(ctx context.Context, model string, group *orm.Group, record *orm.Record) error
	Preprocess(ctx context.Context, model string, group *orm.Group, values map[string]interface{}, mode string) (map[string]interface{}, error)
}

type ModelServer struct {
	pool        *orm.Pool
	checkAccess func(ctx context.Context, model, mode string) error
	checkField  func(ctx context.Context, model, field, mode string) error
	ruleDomain  func(ctx context.Context, model, mode string) ([]interface{}, error)
	translator  Translator
	hooks       ModuleHooks
}

func NewModelServer(pool *orm.Pool) *ModelServer {
	return &ModelServer{pool: pool}
}

func (s *ModelServer) SetTranslator(translator Translator) {
	s.translator = translator
}

func (s *ModelServer) SetModuleHooks(hooks ModuleHooks) {
	s.hooks = hooks
}

func (s *ModelServer) SetRuleDomainProvider(provider func(ctx context.Context, model, mode string) ([]interface{}, error)) {
	s.ruleDomain = provider
}

func (s *ModelServer) rules(ctx context.Context, model, mode string) ([]interface{}, error) {
	if s.ruleDomain == nil {
		return nil, nil
	}
	return s.ruleDomain(ctx, model, mode)
}

func (s *ModelServer) withRules(ruleDomain, domain []interface{}) []interface{} {
	if len(ruleDomain) == 0 {
		return domain
	}
	if len(domain) == 0 {
		return ruleDomain
	}
	combined := make([]interface{}, 0, len(domain)+1)
	combined = append(combined, ruleDomain)
	combined = append(combined, domain...)
	return combined
}

func (s *ModelServer) contextLanguage(ctx context.Context) string {
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

func (s *ModelServer) translate(ctx context.Context, name, source string) string {
	if s.translator == nil {
		return source
	}
	return s.translator.Translate(ctx, name, s.contextLanguage(ctx), source)
}

func (s *ModelServer) SetAccessChecker(check func(ctx context.Context, model, mode string) error) {
	s.checkAccess = check
}

func (s *ModelServer) SetFieldAccessChecker(check func(ctx context.Context, model, field, mode string) error) {
	s.checkField = check
}

func (s *ModelServer) authorize(ctx context.Context, model, mode string) error {
	if s.checkAccess == nil {
		return nil
	}
	if err := s.checkAccess(ctx, model, mode); err != nil {
		return &UserError{Message: err.Error()}
	}
	return nil
}

func (s *ModelServer) authorizeField(ctx context.Context, model, field, mode string) error {
	if s.checkField == nil {
		return nil
	}
	if err := s.checkField(ctx, model, field, mode); err != nil {
		return &UserError{Message: err.Error()}
	}
	return nil
}

func (s *ModelServer) authorizeValues(ctx context.Context, model string, values map[string]interface{}, mode string) error {
	for field := range values {
		if err := s.authorizeField(ctx, model, field, mode); err != nil {
			return err
		}
	}
	return nil
}

func (s *ModelServer) RegisterAll(srv *Server) {
	srv.Register("model.<model>.read", s.Read)
	srv.Register("model.<model>.write", s.Write)
	srv.Register("model.<model>.create", s.Create)
	srv.Register("model.<model>.delete", s.Delete)
	srv.Register("model.<model>.search", s.Search)
	srv.Register("model.<model>.search_read", s.SearchRead)
	srv.Register("model.<model>.search_count", s.SearchCount)
	srv.Register("model.<model>.fields_get", s.FieldsGet)
	srv.Register("model.<model>.fields_view_get", s.FieldsViewGet)
	srv.Register("model.<model>.view_toolbar_get", s.ViewToolbarGet)
	srv.Register("model.<model>.default_get", s.DefaultGet)
	srv.Register("model.<model>.on_change", s.OnChange)
	srv.Register("model.<model>.on_change_with", s.OnChangeWith)
	srv.Register("model.<model>.copy", s.Copy)
	srv.Register("model.<model>.pre_validate", s.PreValidate)
	srv.Register("model.<model>.export_data", s.ExportData)
	srv.Register("model.<model>.import_data", s.ImportData)
	srv.Register("model.<model>.resources", s.Resources)
	srv.Register("model.<model>.history_revisions", s.HistoryRevisions)
}

func (s *ModelServer) modelName(method string) string {
	parts := strings.SplitN(method, ".", 3)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func (s *ModelServer) getGroup(model string) (*orm.Group, error) {
	return s.pool.NewGroup(model)
}

func (s *ModelServer) Read(ctx context.Context, params []interface{}) (interface{}, error) {
	if len(params) < 2 {
		return nil, &UserError{Message: "read requires ids and fields"}
	}

	model := s.modelNameFromContext(ctx)
	if err := s.authorize(ctx, model, "read"); err != nil {
		return nil, err
	}
	ids, ok := params[0].([]interface{})
	if !ok {
		return nil, &UserError{Message: "invalid ids"}
	}

	fields, ok := params[1].([]interface{})
	if !ok {
		fields = []interface{}{}
	}

	group, err := s.getGroup(model)
	if err != nil {
		return nil, err
	}

	fieldNames := make([]string, len(fields))
	for i, f := range fields {
		fieldNames[i], _ = f.(string)
	}

	idInts := make([]int, len(ids))
	for i, id := range ids {
		switch v := id.(type) {
		case float64:
			idInts[i] = int(v)
		case int:
			idInts[i] = v
		}
	}

	if err := group.Load(idInts); err != nil {
		return nil, err
	}

	ruleDomain, err := s.rules(ctx, model, "read")
	if err != nil {
		return nil, &UserError{Message: err.Error()}
	}

	result := make([]interface{}, 0, len(group.Records()))
	for _, r := range group.Records() {
		if err := r.Load(fieldNames); err != nil {
			continue
		}
		if len(ruleDomain) > 0 && !pyson.MatchDomain(ruleDomain, r.GetOnChangeValue(), ruleContext(ctx)) {
			continue
		}
		result = append(result, r.ExportData(fieldNames))
	}

	return result, nil
}

func (s *ModelServer) Write(ctx context.Context, params []interface{}) (interface{}, error) {
	if len(params) < 2 {
		return nil, &UserError{Message: "write requires ids and values"}
	}

	model := s.modelNameFromContext(ctx)
	if err := s.authorize(ctx, model, "write"); err != nil {
		return nil, err
	}
	ids, ok := params[0].([]interface{})
	if !ok {
		return nil, &UserError{Message: "invalid ids"}
	}

	values, ok := params[1].(map[string]interface{})
	if !ok {
		return nil, &UserError{Message: "invalid values"}
	}
	if err := s.authorizeValues(ctx, model, values, "write"); err != nil {
		return nil, err
	}

	group, err := s.getGroup(model)
	if err != nil {
		return nil, err
	}

	idInts := make([]int, len(ids))
	for i, id := range ids {
		switch v := id.(type) {
		case float64:
			idInts[i] = int(v)
		case int:
			idInts[i] = v
		}
	}

	if err := group.Load(idInts); err != nil {
		return nil, err
	}

	ruleDomain, err := s.rules(ctx, model, "write")
	if err != nil {
		return nil, &UserError{Message: err.Error()}
	}
	if s.hooks != nil {
		processed, err := s.hooks.Preprocess(ctx, model, group, values, "write")
		if err != nil {
			return nil, &UserError{Message: err.Error()}
		}
		values = processed
	}

	for _, r := range group.Records() {
		if err := r.Load(nil); err != nil {
			return nil, err
		}
		r.ImportData(values)
		if len(ruleDomain) > 0 && !pyson.MatchDomain(ruleDomain, r.GetOnChangeValue(), ruleContext(ctx)) {
			return nil, &UserError{Message: "access denied"}
		}
		if s.hooks != nil {
			if err := s.hooks.Validate(ctx, model, group, r); err != nil {
				return nil, &UserError{Message: err.Error()}
			}
		}
		if err := r.Save(); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (s *ModelServer) Create(ctx context.Context, params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, &UserError{Message: "create requires values"}
	}

	model := s.modelNameFromContext(ctx)
	if err := s.authorize(ctx, model, "create"); err != nil {
		return nil, err
	}
	values, ok := params[0].([]interface{})
	if !ok {
		return nil, &UserError{Message: "invalid values"}
	}

	group, err := s.getGroup(model)
	if err != nil {
		return nil, err
	}

	ids := make([]int, 0, len(values))
	ruleDomain, err := s.rules(ctx, model, "create")
	if err != nil {
		return nil, &UserError{Message: err.Error()}
	}

	for _, v := range values {
		valMap, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		if s.hooks != nil {
			processed, err := s.hooks.Preprocess(ctx, model, group, valMap, "create")
			if err != nil {
				return nil, &UserError{Message: err.Error()}
			}
			valMap = processed
		}
		if len(ruleDomain) > 0 && !pyson.MatchDomain(ruleDomain, valMap, ruleContext(ctx)) {
			return nil, &UserError{Message: "access denied"}
		}
		r := group.New()
		r.ImportData(valMap)
		if s.hooks != nil {
			if err := s.hooks.Validate(ctx, model, group, r); err != nil {
				return nil, &UserError{Message: err.Error()}
			}
		}
		if err := r.Save(); err != nil {
			return nil, err
		}
		ids = append(ids, r.ID())
	}

	return ids, nil
}

func (s *ModelServer) Delete(ctx context.Context, params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, &UserError{Message: "delete requires ids"}
	}

	model := s.modelNameFromContext(ctx)
	if err := s.authorize(ctx, model, "delete"); err != nil {
		return nil, err
	}
	ids, ok := params[0].([]interface{})
	if !ok {
		return nil, &UserError{Message: "invalid ids"}
	}

	group, err := s.getGroup(model)
	if err != nil {
		return nil, err
	}

	idInts := make([]int, len(ids))
	for i, id := range ids {
		switch v := id.(type) {
		case float64:
			idInts[i] = int(v)
		case int:
			idInts[i] = v
		}
	}

	if err := group.Load(idInts); err != nil {
		return nil, err
	}

	ruleDomain, err := s.rules(ctx, model, "delete")
	if err != nil {
		return nil, &UserError{Message: err.Error()}
	}

	for _, r := range group.Records() {
		if err := r.Load(nil); err != nil {
			return nil, err
		}
		if len(ruleDomain) > 0 && !pyson.MatchDomain(ruleDomain, r.GetOnChangeValue(), ruleContext(ctx)) {
			return nil, &UserError{Message: "access denied"}
		}
		if err := r.Delete(); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (s *ModelServer) Search(ctx context.Context, params []interface{}) (interface{}, error) {
	model := s.modelNameFromContext(ctx)
	if err := s.authorize(ctx, model, "read"); err != nil {
		return nil, err
	}
	group, err := s.getGroup(model)
	if err != nil {
		return nil, err
	}

	domain := []interface{}{}
	if len(params) > 0 {
		if d, ok := params[0].([]interface{}); ok {
			domain = d
		}
	}

	limit := 1000
	if len(params) > 2 {
		if l, ok := params[2].(float64); ok {
			limit = int(l)
		}
	}

	offset := 0
	if len(params) > 1 {
		if o, ok := params[1].(float64); ok {
			offset = int(o)
		}
	}

	order := orderFromParams(params, 3)

	db := s.pool.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}

	meta := group.Model()
	table := quoteIdent(meta.TableName())
	query := fmt.Sprintf("SELECT id FROM %s", table)
	args := []interface{}{}

	ruleDomain, err := s.rules(ctx, model, "read")
	if err != nil {
		return nil, &UserError{Message: err.Error()}
	}
	domain = s.withRules(ruleDomain, domain)

	if len(domain) > 0 {
		where, whereArgs := s.domainToSQL(meta, domain, 1)
		if where != "" {
			query += " WHERE " + where
			args = append(args, whereArgs...)
		}
	}

	if clause := orderToSQL(meta, order); clause != "" {
		query += " ORDER BY " + clause
	}

	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []interface{}{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (s *ModelServer) SearchRead(ctx context.Context, params []interface{}) (interface{}, error) {
	model := s.modelNameFromContext(ctx)
	if err := s.authorize(ctx, model, "read"); err != nil {
		return nil, err
	}
	group, err := s.getGroup(model)
	if err != nil {
		return nil, err
	}

	domain := []interface{}{}
	if len(params) > 0 {
		if d, ok := params[0].([]interface{}); ok {
			domain = d
		}
	}

	offset := 0
	if len(params) > 1 {
		if o, ok := params[1].(float64); ok {
			offset = int(o)
		}
	}

	limit := 1000
	if len(params) > 2 {
		if l, ok := params[2].(float64); ok {
			limit = int(l)
		}
	}

	order := orderFromParams(params, 3)

	fields := []interface{}{}
	if len(params) > 4 {
		if f, ok := params[4].([]interface{}); ok {
			fields = f
		}
	}

	db := s.pool.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}

	meta := group.Model()
	table := quoteIdent(meta.TableName())
	requested := make([]string, 0, len(fields))
	for _, f := range fields {
		name, _ := f.(string)
		requested = append(requested, name)
	}

	if len(requested) == 0 {
		requested = meta.FieldNames()
	}

	fieldNames := make([]string, 0, len(requested))
	columns := make([]string, 0, len(requested))
	for _, name := range requested {
		field, ok := meta.Fields[name]
		if !ok {
			continue
		}
		if field.IsX2M() {
			continue
		}
		fieldNames = append(fieldNames, name)
		columns = append(columns, quoteIdent(field.ColumnName()))
	}

	query := fmt.Sprintf("SELECT id, %s FROM %s", strings.Join(columns, ", "), table)
	args := []interface{}{}

	ruleDomain, err := s.rules(ctx, model, "read")
	if err != nil {
		return nil, &UserError{Message: err.Error()}
	}
	domain = s.withRules(ruleDomain, domain)

	if len(domain) > 0 {
		where, whereArgs := s.domainToSQL(meta, domain, 1)
		if where != "" {
			query += " WHERE " + where
			args = append(args, whereArgs...)
		}
	}

	if clause := orderToSQL(meta, order); clause != "" {
		query += " ORDER BY " + clause
	}

	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []interface{}{}
	for rows.Next() {
		values := make([]interface{}, len(columns)+1)
		ptrs := make([]interface{}, len(columns)+1)
		for i := range values {
			ptrs[i] = &values[i]
		}

		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}

		record := map[string]interface{}{
			"id": normalizeValue(values[0]),
		}
		for i, name := range fieldNames {
			record[name] = normalizeValue(values[i+1])
		}
		result = append(result, record)
	}

	return result, nil
}

func (s *ModelServer) SearchCount(ctx context.Context, params []interface{}) (interface{}, error) {
	model := s.modelNameFromContext(ctx)
	if err := s.authorize(ctx, model, "read"); err != nil {
		return nil, err
	}
	group, err := s.getGroup(model)
	if err != nil {
		return nil, err
	}

	domain := []interface{}{}
	if len(params) > 0 {
		if d, ok := params[0].([]interface{}); ok {
			domain = d
		}
	}

	db := s.pool.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}

	meta := group.Model()
	table := quoteIdent(meta.TableName())
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	args := []interface{}{}

	ruleDomain, err := s.rules(ctx, model, "read")
	if err != nil {
		return nil, &UserError{Message: err.Error()}
	}
	domain = s.withRules(ruleDomain, domain)

	if len(domain) > 0 {
		where, whereArgs := s.domainToSQL(meta, domain, 1)
		if where != "" {
			query += " WHERE " + where
			args = append(args, whereArgs...)
		}
	}

	var count int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return nil, err
	}

	return count, nil
}

func (s *ModelServer) FieldsGet(ctx context.Context, params []interface{}) (interface{}, error) {
	model := s.modelNameFromContext(ctx)
	if err := s.authorize(ctx, model, "read"); err != nil {
		return nil, err
	}
	m, err := s.pool.GetModel(model)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{}
	for name, field := range m.Fields {
		description := field.Description()
		description["string"] = s.translate(ctx, model+","+name, field.Label)
		result[name] = description
	}

	return result, nil
}

func (s *ModelServer) FieldsViewGet(ctx context.Context, params []interface{}) (interface{}, error) {
	model := s.modelNameFromContext(ctx)
	if err := s.authorize(ctx, model, "read"); err != nil {
		return nil, err
	}
	m, err := s.pool.GetModel(model)
	if err != nil {
		return nil, err
	}

	viewID := 0
	if len(params) > 0 {
		viewID, _ = orm.AsInt(params[0])
	}
	viewType := "form"
	if len(params) > 1 {
		if value, ok := params[1].(string); ok && value != "" {
			viewType = value
		}
	}

	fields := map[string]interface{}{}
	for name, field := range m.Fields {
		description := field.Description()
		description["string"] = s.translate(ctx, model+","+name, field.Label)
		fields[name] = description
	}

	arch := s.viewArch(ctx, model, viewID, viewType)
	if arch == "" {
		arch = `<form><field name="id"/></form>`
	}

	return map[string]interface{}{
		"model":  model,
		"type":   viewType,
		"arch":   arch,
		"fields": fields,
	}, nil
}

type viewRow struct {
	id       int
	arch     string
	inherit  int
	priority int
}

func (s *ModelServer) viewArch(ctx context.Context, model string, viewID int, viewType string) string {
	db := s.pool.DB()
	if db == nil {
		return ""
	}
	rows, err := db.QueryContext(ctx,
		`SELECT id, COALESCE(arch, ''), COALESCE(inherit, 0), COALESCE(priority, 0)
		 FROM ir_ui_view WHERE model = $1 ORDER BY priority, id`, model)
	if err != nil {
		return ""
	}
	defer rows.Close()

	views := []viewRow{}
	byID := map[int]viewRow{}
	for rows.Next() {
		var row viewRow
		if err := rows.Scan(&row.id, &row.arch, &row.inherit, &row.priority); err != nil {
			return ""
		}
		views = append(views, row)
		byID[row.id] = row
	}

	if viewID == 0 {
		for _, row := range views {
			if row.inherit == 0 {
				viewID = row.id
				break
			}
		}
		if viewID == 0 {
			return ""
		}
	}

	root := viewID
	for {
		row, ok := byID[root]
		if !ok || row.inherit == 0 {
			break
		}
		root = row.inherit
	}

	inTree := func(id int) bool {
		seen := map[int]bool{}
		for id != 0 {
			if seen[id] {
				return false
			}
			seen[id] = true
			if id == root {
				return true
			}
			row, ok := byID[id]
			if !ok {
				return false
			}
			id = row.inherit
		}
		return false
	}

	selected := []xml.View{}
	for _, row := range views {
		if !inTree(row.id) {
			continue
		}
		selected = append(selected, xml.View{
			ID:       row.id,
			Arch:     row.arch,
			Inherit:  row.inherit,
			Priority: row.priority,
		})
	}
	if len(selected) == 0 {
		return ""
	}
	merged, err := xml.MergeViews(selected)
	if err != nil {
		return ""
	}
	return merged
}

func (s *ModelServer) ViewToolbarGet(ctx context.Context, params []interface{}) (interface{}, error) {
	return map[string]interface{}{
		"print":   []interface{}{},
		"action":  []interface{}{},
		"relate":  []interface{}{},
		"exports": []interface{}{},
		"emails":  []interface{}{},
	}, nil
}

func (s *ModelServer) DefaultGet(ctx context.Context, params []interface{}) (interface{}, error) {
	model := s.modelNameFromContext(ctx)
	m, err := s.pool.GetModel(model)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{}
	for name, field := range m.Fields {
		if field.Required {
			result[name] = field.DefaultValue()
		}
	}

	return result, nil
}

func (s *ModelServer) OnChange(ctx context.Context, params []interface{}) (interface{}, error) {
	if s.hooks == nil {
		return map[string]interface{}{}, nil
	}
	model := s.modelNameFromContext(ctx)
	group, err := s.getGroup(model)
	if err != nil {
		return nil, err
	}
	values := map[string]interface{}{}
	if len(params) > 0 {
		values, _ = params[0].(map[string]interface{})
	}
	return s.hooks.OnChange(ctx, model, group, values, fieldsFromParams(params, 1))
}

func (s *ModelServer) OnChangeWith(ctx context.Context, params []interface{}) (interface{}, error) {
	if s.hooks == nil {
		return map[string]interface{}{}, nil
	}
	model := s.modelNameFromContext(ctx)
	group, err := s.getGroup(model)
	if err != nil {
		return nil, err
	}
	values := map[string]interface{}{}
	if len(params) > 0 {
		values, _ = params[0].(map[string]interface{})
	}
	return s.hooks.OnChangeWith(ctx, model, group, values, fieldsFromParams(params, 1))
}

func fieldsFromParams(params []interface{}, index int) []string {
	if len(params) <= index {
		return nil
	}
	raw, ok := params[index].([]interface{})
	if !ok {
		return nil
	}
	fields := make([]string, 0, len(raw))
	for _, value := range raw {
		if name, ok := value.(string); ok {
			fields = append(fields, name)
		}
	}
	return fields
}

func (s *ModelServer) Copy(ctx context.Context, params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, &UserError{Message: "copy requires ids"}
	}

	model := s.modelNameFromContext(ctx)
	ids, ok := params[0].([]interface{})
	if !ok {
		return nil, &UserError{Message: "invalid ids"}
	}

	group, err := s.getGroup(model)
	if err != nil {
		return nil, err
	}

	idInts := make([]int, len(ids))
	for i, id := range ids {
		switch v := id.(type) {
		case float64:
			idInts[i] = int(v)
		case int:
			idInts[i] = v
		}
	}

	if err := group.Load(idInts); err != nil {
		return nil, err
	}

	newIDs := []interface{}{}
	for _, r := range group.Records() {
		duplicate, err := group.Copy(r)
		if err != nil {
			return nil, err
		}
		if err := duplicate.Save(); err != nil {
			return nil, err
		}
		newIDs = append(newIDs, duplicate.ID())
	}

	return newIDs, nil
}

func (s *ModelServer) PreValidate(ctx context.Context, params []interface{}) (interface{}, error) {
	if s.hooks == nil {
		return nil, nil
	}
	model := s.modelNameFromContext(ctx)
	group, err := s.getGroup(model)
	if err != nil {
		return nil, err
	}
	values := map[string]interface{}{}
	if len(params) > 0 {
		values, _ = params[0].(map[string]interface{})
	}
	record := group.New()
	record.ImportData(values)
	if err := s.hooks.Validate(ctx, model, group, record); err != nil {
		return nil, &UserError{Message: err.Error()}
	}
	return nil, nil
}

func (s *ModelServer) ExportData(ctx context.Context, params []interface{}) (interface{}, error) {
	if len(params) < 2 {
		return nil, &UserError{Message: "export_data requires ids and fields"}
	}

	model := s.modelNameFromContext(ctx)
	ids, ok := params[0].([]interface{})
	if !ok {
		return nil, &UserError{Message: "invalid ids"}
	}

	fields, ok := params[1].([]interface{})
	if !ok {
		return nil, &UserError{Message: "invalid fields"}
	}

	group, err := s.getGroup(model)
	if err != nil {
		return nil, err
	}

	idInts := make([]int, len(ids))
	for i, id := range ids {
		switch v := id.(type) {
		case float64:
			idInts[i] = int(v)
		case int:
			idInts[i] = v
		}
	}

	if err := group.Load(idInts); err != nil {
		return nil, err
	}

	fieldNames := make([]string, len(fields))
	for i, f := range fields {
		fieldNames[i], _ = f.(string)
	}

	result := [][]interface{}{}
	for _, r := range group.Records() {
		if err := r.Load(fieldNames); err != nil {
			continue
		}
		row := make([]interface{}, len(fieldNames))
		for i, name := range fieldNames {
			v, _ := r.Get(name)
			row[i] = v
		}
		result = append(result, row)
	}

	return result, nil
}

func (s *ModelServer) ImportData(ctx context.Context, params []interface{}) (interface{}, error) {
	if len(params) < 2 {
		return nil, &UserError{Message: "import_data requires fields and data"}
	}

	model := s.modelNameFromContext(ctx)
	fields, ok := params[0].([]interface{})
	if !ok {
		return nil, &UserError{Message: "invalid fields"}
	}

	data, ok := params[1].([]interface{})
	if !ok {
		return nil, &UserError{Message: "invalid data"}
	}

	group, err := s.getGroup(model)
	if err != nil {
		return nil, err
	}

	fieldNames := make([]string, len(fields))
	for i, f := range fields {
		fieldNames[i], _ = f.(string)
	}

	for _, row := range data {
		rowArr, ok := row.([]interface{})
		if !ok {
			continue
		}
		r := group.New()
		values := map[string]interface{}{}
		for i, name := range fieldNames {
			if i < len(rowArr) {
				values[name] = rowArr[i]
			}
		}
		r.ImportData(values)
		if err := r.Save(); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (s *ModelServer) Resources(ctx context.Context, params []interface{}) (interface{}, error) {
	return map[string]interface{}{}, nil
}

func (s *ModelServer) HistoryRevisions(ctx context.Context, params []interface{}) (interface{}, error) {
	return []interface{}{}, nil
}

var domainOperators = map[string]string{
	"=": "=", "!=": "!=", "<>": "!=", "<": "<", "<=": "<=", ">": ">", ">=": ">=",
	"like": "LIKE", "not like": "NOT LIKE",
	"ilike": "ILIKE", "not ilike": "NOT ILIKE",
	"in": "IN", "not in": "NOT IN",
	"is": "IS", "is not": "IS NOT",
}

func (s *ModelServer) domainToSQL(model *metadata.Model, domain []interface{}, start int) (string, []interface{}) {
	parts := []string{}
	args := []interface{}{}
	for _, d := range domain {
		arr, ok := d.([]interface{})
		if !ok {
			continue
		}
		if len(arr) > 0 {
			if operator, ok := arr[0].(string); ok {
				switch strings.ToUpper(operator) {
				case "AND", "OR", "NOT":
					subParts := []string{}
					for _, sub := range arr[1:] {
						subDomain, ok := sub.([]interface{})
						if !ok {
							continue
						}
						where, whereArgs := s.domainToSQL(model, subDomain, start+len(args))
						args = append(args, whereArgs...)
						if where != "" {
							subParts = append(subParts, where)
						}
					}
					if len(subParts) == 0 {
						continue
					}
					joined := strings.Join(subParts, " "+strings.ToUpper(operator)+" ")
					if strings.EqualFold(operator, "NOT") {
						joined = "NOT (" + joined + ")"
					} else {
						joined = "(" + joined + ")"
					}
					parts = append(parts, joined)
					continue
				}
			}
		}
		if len(arr) < 3 {
			continue
		}
		fieldName, _ := arr[0].(string)
		if fieldName != "id" {
			if _, ok := model.Fields[fieldName]; !ok {
				continue
			}
		}
		op, _ := arr[1].(string)
		sqlOp, ok := domainOperators[strings.ToLower(op)]
		if !ok {
			continue
		}
		value := arr[2]

		if sqlOp == "IN" || sqlOp == "NOT IN" {
			values, ok := value.([]interface{})
			if !ok || len(values) == 0 {
				parts = append(parts, "FALSE")
				continue
			}
			placeholders := make([]string, len(values))
			for i, v := range values {
				placeholders[i] = fmt.Sprintf("$%d", start+len(args))
				args = append(args, v)
			}
			parts = append(parts, fmt.Sprintf("%s %s (%s)",
				quoteIdent(fieldName), sqlOp, strings.Join(placeholders, ", ")))
			continue
		}

		if sqlOp == "LIKE" || sqlOp == "NOT LIKE" || sqlOp == "ILIKE" || sqlOp == "NOT ILIKE" {
			pattern, _ := value.(string)
			if !strings.Contains(pattern, "%") {
				pattern = "%" + pattern + "%"
			}
			value = pattern
		}

		parts = append(parts, fmt.Sprintf("%s %s $%d", quoteIdent(fieldName), sqlOp, start+len(args)))
		args = append(args, value)
	}
	if len(parts) == 0 {
		return "", nil
	}
	return strings.Join(parts, " AND "), args
}

func orderFromParams(params []interface{}, index int) string {
	if len(params) <= index {
		return ""
	}
	switch value := params[index].(type) {
	case string:
		return value
	case []interface{}:
		parts := []string{}
		for _, item := range value {
			pair, ok := item.([]interface{})
			if !ok || len(pair) < 2 {
				continue
			}
			field, _ := pair[0].(string)
			direction, _ := pair[1].(string)
			if field == "" {
				continue
			}
			parts = append(parts, field+" "+strings.ToUpper(direction))
		}
		return strings.Join(parts, ", ")
	default:
		return ""
	}
}

func orderToSQL(model *metadata.Model, order string) string {
	if strings.TrimSpace(order) == "" {
		if model.Order != "" {
			return orderToSQL(model, model.Order)
		}
		return ""
	}
	clauses := []string{}
	for _, item := range strings.Split(order, ",") {
		parts := strings.Fields(strings.TrimSpace(item))
		if len(parts) == 0 {
			continue
		}
		fieldName := parts[0]
		if fieldName != "id" {
			if _, ok := model.Fields[fieldName]; !ok {
				continue
			}
		}
		direction := "ASC"
		if len(parts) > 1 {
			switch strings.ToUpper(parts[1]) {
			case "DESC":
				direction = "DESC"
			case "ASC":
				direction = "ASC"
			}
		}
		clauses = append(clauses, quoteIdent(fieldName)+" "+direction)
	}
	return strings.Join(clauses, ", ")
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func ruleContext(ctx context.Context) map[string]interface{} {
	result := map[string]interface{}{}
	if contextMap, ok := ctx.Value("context").(map[string]interface{}); ok {
		for key, value := range contextMap {
			result[key] = value
		}
	}
	if user, ok := ctx.Value("user").(int); ok {
		result["user"] = user
	}
	return result
}

func normalizeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case nil:
		return nil
	case int64:
		return int(v)
	case []byte:
		return string(v)
	case time.Time:
		return v
	default:
		return v
	}
}

func (s *ModelServer) modelNameFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if model, ok := ctx.Value("model").(string); ok && model != "" {
		return model
	}
	if method, ok := ctx.Value("method").(string); ok {
		return s.modelName(method)
	}
	return ""
}

func (s *ModelServer) RegisterModelMethods(srv *Server, model string, methods map[string]Handler) {
	for name, h := range methods {
		srv.Register(fmt.Sprintf("model.%s.%s", model, name), h)
	}
}

func (s *ModelServer) RegisterCommonMethods(srv *Server, methods map[string]Handler) {
	for name, h := range methods {
		srv.Register(fmt.Sprintf("common.%s", name), h)
	}
}

func (s *ModelServer) RegisterWizardMethods(srv *Server, wizard string, methods map[string]Handler) {
	for name, h := range methods {
		srv.Register(fmt.Sprintf("wizard.%s.%s", wizard, name), h)
	}
}

func (s *ModelServer) RegisterReportMethods(srv *Server, report string, h Handler) {
	srv.Register(fmt.Sprintf("report.%s.execute", report), h)
}

func (s *ModelServer) MethodPrefix(method string) string {
	parts := strings.SplitN(method, ".", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func (s *ModelServer) MethodName(method string) string {
	parts := strings.SplitN(method, ".", 3)
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

func (s *ModelServer) ModelName(method string) string {
	parts := strings.SplitN(method, ".", 3)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func (s *ModelServer) GetModel(name string) (*metadata.Model, error) {
	return s.pool.GetModel(name)
}

func (s *ModelServer) ListModels() []string {
	return s.pool.ListModels()
}

func (s *ModelServer) GetGroup(name string) (*orm.Group, error) {
	return s.pool.NewGroup(name)
}

func (s *ModelServer) NewGroup(name string) (*orm.Group, error) {
	return s.pool.NewGroup(name)
}

func (s *ModelServer) GetScreen(name string) (*orm.Screen, error) {
	return s.pool.NewScreen(name)
}

func (s *ModelServer) NewScreen(name string) (*orm.Screen, error) {
	return s.pool.NewScreen(name)
}

func (s *ModelServer) Clear() {
	s.pool.Clear()
}

func (s *ModelServer) Init(update bool, lang string) error {
	return s.pool.Init(context.Background(), update, lang)
}

func (s *ModelServer) Upgrade(modules []string) error {
	return s.pool.Upgrade(context.Background(), modules)
}
