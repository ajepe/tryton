package action

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Manager struct {
	db *sql.DB
}

func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

func (m *Manager) GetActionValue(ctx context.Context, actionID int, rpcContext map[string]interface{}) (map[string]interface{}, error) {
	if m.db == nil {
		return nil, fmt.Errorf("no database connection")
	}

	var actionType, name sql.NullString
	var usage sql.NullString
	err := m.db.QueryRowContext(ctx,
		`SELECT type, name, usage FROM ir_action WHERE id = $1`, actionID).Scan(&actionType, &name, &usage)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("action %d not found", actionID)
	}
	if err != nil {
		return nil, err
	}

	value := map[string]interface{}{
		"id":   actionID,
		"type": actionType.String,
		"name": name.String,
	}
	if usage.Valid {
		value["usage"] = usage.String
	}

	switch actionType.String {
	case "ir.action.act_window":
		if err := m.actWindowValue(ctx, actionID, value); err != nil {
			return nil, err
		}
	case "ir.action.wizard":
		if err := m.wizardValue(ctx, actionID, value); err != nil {
			return nil, err
		}
	case "ir.action.report":
		if err := m.reportValue(ctx, actionID, value); err != nil {
			return nil, err
		}
	case "ir.action.url":
		if err := m.urlValue(ctx, actionID, value); err != nil {
			return nil, err
		}
	}
	return value, nil
}

func (m *Manager) actWindowValue(ctx context.Context, actionID int, value map[string]interface{}) error {
	var resModel, viewMode, order sql.NullString
	var domain, rpcContext sql.NullString
	var limit sql.NullInt64
	err := m.db.QueryRowContext(ctx,
		`SELECT res_model, view_mode, domain, context, "limit", "order"
		 FROM ir_action_act_window WHERE action = $1`, actionID).Scan(
		&resModel, &viewMode, &domain, &rpcContext, &limit, &order)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if resModel.Valid {
		value["res_model"] = resModel.String
	}
	if viewMode.Valid {
		value["view_mode"] = viewMode.String
	}
	if domain.Valid && domain.String != "" {
		value["domain"] = domain.String
	}
	if rpcContext.Valid && rpcContext.String != "" {
		value["context"] = rpcContext.String
	}
	if limit.Valid {
		value["limit"] = int(limit.Int64)
	}
	if order.Valid && order.String != "" {
		value["order"] = order.String
	}

	if resModel.Valid && viewMode.Valid {
		views, err := m.views(ctx, resModel.String, viewMode.String)
		if err != nil {
			return err
		}
		value["views"] = views
	}
	return nil
}

func (m *Manager) views(ctx context.Context, model, viewMode string) ([][]interface{}, error) {
	views := [][]interface{}{}
	seen := map[string]bool{}
	for _, viewType := range strings.Split(viewMode, ",") {
		viewType = strings.TrimSpace(viewType)
		if viewType == "" || seen[viewType] {
			continue
		}
		seen[viewType] = true
		var id int
		err := m.db.QueryRowContext(ctx,
			`SELECT id FROM ir_ui_view
			 WHERE model = $1 AND type = $2
			 ORDER BY priority, id LIMIT 1`,
			model, viewType).Scan(&id)
		if err == sql.ErrNoRows {
			views = append(views, []interface{}{0, viewType})
			continue
		}
		if err != nil {
			return nil, err
		}
		views = append(views, []interface{}{id, viewType})
	}
	return views, nil
}

func (m *Manager) wizardValue(ctx context.Context, actionID int, value map[string]interface{}) error {
	var wizName sql.NullString
	var window sql.NullBool
	err := m.db.QueryRowContext(ctx,
		`SELECT wiz_name, window FROM ir_action_wizard WHERE action = $1`, actionID).Scan(&wizName, &window)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if wizName.Valid {
		value["wiz_name"] = wizName.String
	}
	if window.Valid {
		value["window"] = window.Bool
	}
	return nil
}

func (m *Manager) reportValue(ctx context.Context, actionID int, value map[string]interface{}) error {
	var reportName sql.NullString
	var directPrint sql.NullBool
	err := m.db.QueryRowContext(ctx,
		`SELECT report_name, direct_print FROM ir_action_report WHERE action = $1`, actionID).Scan(
		&reportName, &directPrint)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if reportName.Valid {
		value["report_name"] = reportName.String
	}
	if directPrint.Valid {
		value["direct_print"] = directPrint.Bool
	}
	return nil
}

func (m *Manager) urlValue(ctx context.Context, actionID int, value map[string]interface{}) error {
	var url sql.NullString
	err := m.db.QueryRowContext(ctx,
		`SELECT url FROM ir_action_url WHERE action = $1`, actionID).Scan(&url)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if url.Valid {
		value["url"] = url.String
	}
	return nil
}

func (m *Manager) GetKeyword(ctx context.Context, keyword, model string, recordID int) ([]map[string]interface{}, error) {
	if m.db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	rows, err := m.db.QueryContext(ctx,
		`SELECT action FROM ir_action_keyword
		 WHERE keyword = $1 AND model = $2`,
		keyword, fmt.Sprintf("%s,%d", model, recordID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	actions := []map[string]interface{}{}
	for rows.Next() {
		var actionID int
		if err := rows.Scan(&actionID); err != nil {
			return nil, err
		}
		value, err := m.GetActionValue(ctx, actionID, nil)
		if err != nil {
			return nil, err
		}
		actions = append(actions, value)
	}
	return actions, rows.Err()
}

func ParseReference(value string) (string, int, bool) {
	parts := strings.SplitN(value, ",", 2)
	if len(parts) != 2 {
		return "", 0, false
	}
	var id int
	if _, err := fmt.Sscanf(parts[1], "%d", &id); err != nil {
		return "", 0, false
	}
	return parts[0], id, true
}
