package security

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/ajepe/talon/internal/pyson"
)

func (m *Manager) RecordDomain(ctx context.Context, userID int, model, mode string) ([]interface{}, error) {
	admin, err := m.IsAdmin(ctx, userID)
	if err != nil {
		return nil, err
	}
	if admin {
		return nil, nil
	}
	if m.db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	groups, err := m.groups(ctx, userID)
	if err != nil {
		return nil, err
	}

	permColumn := map[string]string{
		"read":   "perm_read",
		"write":  "perm_write",
		"create": "perm_create",
		"delete": "perm_delete",
	}[mode]
	if permColumn == "" {
		return nil, fmt.Errorf("invalid mode %s", mode)
	}

	rows, err := m.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT g.id, g.global_p
		 FROM ir_rule_group g
		 LEFT JOIN "ir_rule_group-res_group" rgg ON rgg.rule_group = g.id
		 WHERE g.model = $1 AND g.%s = true
		 GROUP BY g.id, g.global_p
		 HAVING bool_or(rgg."group" = ANY($2)) OR bool_or(g.default_p) OR bool_or(g.global_p)`,
			permColumn),
		model, groups)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type group struct {
		id     int
		global bool
	}
	selected := []group{}
	for rows.Next() {
		var g group
		var global sql.NullBool
		if err := rows.Scan(&g.id, &global); err != nil {
			return nil, err
		}
		g.global = global.Valid && global.Bool
		selected = append(selected, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return nil, nil
	}

	domains := map[int][]interface{}{}
	groupIDs := []int{}
	for _, g := range selected {
		groupIDs = append(groupIDs, g.id)
	}
	ruleRows, err := m.db.QueryContext(ctx,
		`SELECT rule_group, domain FROM ir_rule WHERE rule_group = ANY($1)`, groupIDs)
	if err != nil {
		return nil, err
	}
	defer ruleRows.Close()
	for ruleRows.Next() {
		var groupID int
		var domain string
		if err := ruleRows.Scan(&groupID, &domain); err != nil {
			return nil, err
		}
		parsed, err := parseDomain(domain)
		if err != nil {
			return nil, err
		}
		domains[groupID] = append(domains[groupID], parsed)
	}
	if err := ruleRows.Err(); err != nil {
		return nil, err
	}

	userClause := []interface{}{"OR"}
	globalClause := []interface{}{"AND"}
	userRules := false
	globalRules := false

	for _, g := range selected {
		groupDomains := domains[g.id]
		if len(groupDomains) == 0 {
			if !g.global {
				return nil, nil
			}
			continue
		}
		clause := []interface{}{"OR"}
		clause = append(clause, groupDomains...)
		if g.global {
			globalClause = append(globalClause, clause)
			globalRules = true
		} else {
			userClause = append(userClause, clause)
			userRules = true
		}
	}

	switch {
	case globalRules && userRules:
		return []interface{}{"AND", globalClause, userClause}, nil
	case globalRules:
		return globalClause, nil
	case userRules:
		return userClause, nil
	default:
		return nil, nil
	}
}

func parseDomain(source string) ([]interface{}, error) {
	value, err := pyson.Parse(strings.TrimSpace(source))
	if err != nil {
		return nil, fmt.Errorf("invalid rule domain %q: %w", source, err)
	}
	domain, ok := value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("rule domain %q is not a list", source)
	}
	return domain, nil
}

func (m *Manager) ContextRuleDomain() func(ctx context.Context, model, mode string) ([]interface{}, error) {
	return func(ctx context.Context, model, mode string) ([]interface{}, error) {
		userID := UserFromContext(ctx)
		if userID == 0 {
			return nil, nil
		}
		return m.RecordDomain(ctx, userID, model, mode)
	}
}
