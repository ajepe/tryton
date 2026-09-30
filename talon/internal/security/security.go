package security

import (
	"context"
	"database/sql"
	"fmt"
)

type Permission struct {
	Read   bool
	Write  bool
	Create bool
	Delete bool
}

func (p Permission) Allows(mode string) bool {
	switch mode {
	case "read":
		return p.Read
	case "write":
		return p.Write
	case "create":
		return p.Create
	case "delete":
		return p.Delete
	default:
		return false
	}
}

type Manager struct {
	db *sql.DB
}

func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

func (m *Manager) IsAdmin(ctx context.Context, userID int) (bool, error) {
	if m.db == nil {
		return false, nil
	}
	var admin bool
	err := m.db.QueryRowContext(ctx,
		`SELECT COALESCE(administrator, false) FROM res_user WHERE id = $1`,
		userID).Scan(&admin)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return admin, nil
}

func (m *Manager) groups(ctx context.Context, userID int) ([]int, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT "group" FROM "res_user-res_group" WHERE "user" = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := []int{}
	for rows.Next() {
		var groupID int
		if err := rows.Scan(&groupID); err != nil {
			return nil, err
		}
		groups = append(groups, groupID)
	}
	return groups, rows.Err()
}

func (m *Manager) Permission(ctx context.Context, userID int, model string) (Permission, error) {
	if m.db == nil {
		return Permission{}, fmt.Errorf("no database connection")
	}
	groups, err := m.groups(ctx, userID)
	if err != nil {
		return Permission{}, err
	}

	var permission Permission
	err = m.db.QueryRowContext(ctx,
		`SELECT
			COALESCE(bool_or(a.perm_read), false),
			COALESCE(bool_or(a.perm_write), false),
			COALESCE(bool_or(a.perm_create), false),
			COALESCE(bool_or(a.perm_delete), false)
		 FROM ir_model_access a
		 JOIN ir_model m ON m.id = a.model
		 WHERE m.model = $1
		   AND (a."group" IS NULL OR a."group" = ANY($2))`,
		model, groups).Scan(
		&permission.Read, &permission.Write, &permission.Create, &permission.Delete)
	if err != nil {
		return Permission{}, err
	}
	return permission, nil
}

func (m *Manager) Check(ctx context.Context, userID int, model, mode string) error {
	admin, err := m.IsAdmin(ctx, userID)
	if err != nil {
		return err
	}
	if admin {
		return nil
	}
	permission, err := m.Permission(ctx, userID, model)
	if err != nil {
		return err
	}
	if !permission.Allows(mode) {
		return fmt.Errorf("access denied: %s on %s", mode, model)
	}
	return nil
}

func (m *Manager) FieldPermission(ctx context.Context, userID int, model, field string) (Permission, error) {
	if m.db == nil {
		return Permission{}, fmt.Errorf("no database connection")
	}
	groups, err := m.groups(ctx, userID)
	if err != nil {
		return Permission{}, err
	}

	var count int
	if err := m.db.QueryRowContext(ctx,
		`SELECT COUNT(*)
		 FROM ir_model_field_access a
		 JOIN ir_model_field f ON f.id = a.field
		 JOIN ir_model m ON m.id = f.model
		 WHERE m.model = $1 AND f.name = $2`,
		model, field).Scan(&count); err != nil {
		return Permission{}, err
	}
	if count == 0 {
		return Permission{Read: true, Write: true, Create: true, Delete: true}, nil
	}

	var permission Permission
	err = m.db.QueryRowContext(ctx,
		`SELECT
			COALESCE(bool_or(a.perm_read), false),
			COALESCE(bool_or(a.perm_write), false),
			COALESCE(bool_or(a.perm_create), false),
			COALESCE(bool_or(a.perm_delete), false)
		 FROM ir_model_field_access a
		 JOIN ir_model_field f ON f.id = a.field
		 JOIN ir_model m ON m.id = f.model
		 WHERE m.model = $1 AND f.name = $2
		   AND (a."group" IS NULL OR a."group" = ANY($3))`,
		model, field, groups).Scan(
		&permission.Read, &permission.Write, &permission.Create, &permission.Delete)
	if err != nil {
		return Permission{}, err
	}
	return permission, nil
}

func (m *Manager) CheckField(ctx context.Context, userID int, model, field, mode string) error {
	admin, err := m.IsAdmin(ctx, userID)
	if err != nil {
		return err
	}
	if admin {
		return nil
	}
	permission, err := m.FieldPermission(ctx, userID, model, field)
	if err != nil {
		return err
	}
	if !permission.Allows(mode) {
		return fmt.Errorf("field access denied: %s on %s.%s", mode, model, field)
	}
	return nil
}

func UserFromContext(ctx context.Context) int {
	if ctx == nil {
		return 0
	}
	if user, ok := ctx.Value("user").(int); ok {
		return user
	}
	return 0
}

func (m *Manager) ContextChecker() func(ctx context.Context, model, mode string) error {
	return func(ctx context.Context, model, mode string) error {
		userID := UserFromContext(ctx)
		if userID == 0 {
			return nil
		}
		return m.Check(ctx, userID, model, mode)
	}
}
