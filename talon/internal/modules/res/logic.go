package res

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ajepe/talon/internal/orm"
)

var applicationTransitions = map[string]map[string]string{
	"validate": {
		"requested": "validated",
	},
	"cancel": {
		"requested": "cancelled",
		"validated": "cancelled",
	},
}

func ValidateUser(ctx context.Context, group *orm.Group, record *orm.Record) error {
	login := strings.TrimSpace(orm.AsString(valueAt(record, "login")))
	if login == "" {
		return fmt.Errorf("login is required")
	}
	if email := strings.TrimSpace(orm.AsString(valueAt(record, "email"))); email != "" {
		if !strings.Contains(email, "@") {
			return fmt.Errorf("invalid email address")
		}
	}
	if v, ok := record.Get("preferences"); ok && v != nil {
		if _, ok := v.(map[string]interface{}); !ok {
			return fmt.Errorf("preferences must be a dictionary")
		}
	}
	return nil
}

func OnChangeUser(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	if _, ok := values["login"]; ok {
		result["login"] = strings.TrimSpace(orm.AsString(values["login"]))
	}
	if _, ok := values["email"]; ok {
		result["email"] = strings.TrimSpace(orm.AsString(values["email"]))
	}
	return result, nil
}

func SearchUser(ctx context.Context, group *orm.Group, text string) ([]int, error) {
	db := group.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	pattern := "%" + text + "%"
	rows, err := db.QueryContext(ctx,
		`SELECT id FROM res_user
		 WHERE login ILIKE $1 OR name ILIKE $1
		 ORDER BY id`, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func ValidateGroup(ctx context.Context, group *orm.Group, record *orm.Record) error {
	if strings.TrimSpace(orm.AsString(valueAt(record, "name"))) == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func WarningFormat(name string, records []string) string {
	key := strings.Join(records, "|")
	sum := md5.Sum([]byte(key))
	return hex.EncodeToString(sum[:]) + "." + name
}

func OnChangeNotification(ctx context.Context, group *orm.Group, values map[string]interface{}, fields []string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	if _, ok := values["title"]; ok {
		result["title"] = strings.TrimSpace(orm.AsString(values["title"]))
	}
	return result, nil
}

func ValidateNotification(ctx context.Context, group *orm.Group, record *orm.Record) error {
	if strings.TrimSpace(orm.AsString(valueAt(record, "title"))) == "" {
		return fmt.Errorf("title is required")
	}
	return nil
}

func ButtonMarkRead(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error) {
	ids := []interface{}{}
	for _, r := range records {
		r.Set("read", true)
		if err := r.Save(); err != nil {
			return nil, err
		}
		if user, ok := orm.AsInt(valueAt(r, "user")); ok && user != 0 {
			publish(fmt.Sprintf("notification:%d", user), map[string]interface{}{
				"name":   "res.notification",
				"action": "read",
				"id":     r.ID(),
			})
		}
		ids = append(ids, r.ID())
	}
	return ids, nil
}

func NextApplicationState(action, current string) (string, error) {
	if current == "" {
		current = "requested"
	}
	transitions, ok := applicationTransitions[action]
	if !ok {
		return "", fmt.Errorf("unknown action %s", action)
	}
	next, ok := transitions[current]
	if !ok {
		return "", fmt.Errorf("cannot %s application in state %q", action, current)
	}
	return next, nil
}

func ButtonApplicationTransition(action string) func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error) {
	return func(ctx context.Context, group *orm.Group, records []*orm.Record) (interface{}, error) {
		ids := []interface{}{}
		for _, r := range records {
			next, err := NextApplicationState(action, orm.AsString(valueAt(r, "state")))
			if err != nil {
				return nil, err
			}
			r.Set("state", next)
			if err := r.Save(); err != nil {
				return nil, err
			}
			ids = append(ids, r.ID())
		}
		return ids, nil
	}
}

func ValidateApplication(ctx context.Context, group *orm.Group, record *orm.Record) error {
	switch orm.AsString(valueAt(record, "state")) {
	case "", "requested", "validated", "cancelled":
	default:
		return fmt.Errorf("invalid state")
	}
	switch orm.AsString(valueAt(record, "application")) {
	case "", "rest":
	default:
		return fmt.Errorf("invalid application")
	}
	return nil
}

func valueAt(r *orm.Record, name string) interface{} {
	v, _ := r.Get(name)
	return v
}
