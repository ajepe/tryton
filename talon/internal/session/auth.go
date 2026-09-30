package session

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	maxLoginAttempts = 5
	attemptWindow    = 5 * time.Minute
)

type LoginError struct {
	Message string
	Type    string
}

func (e *LoginError) Error() string { return e.Message }

func HashPassword(password string) (string, error) {
	return hashArgon2(password)
}

func CheckPassword(hash, password string) bool {
	if hash == "" {
		return false
	}
	switch {
	case strings.HasPrefix(hash, "$argon2"):
		return checkArgon2(hash, password)
	case strings.HasPrefix(hash, "$2"):
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
	default:
		return false
	}
}

type attemptTracker struct {
	mu       sync.Mutex
	failures map[string][]time.Time
}

func newAttemptTracker() *attemptTracker {
	return &attemptTracker{failures: map[string][]time.Time{}}
}

func (a *attemptTracker) tooMany(login string, now time.Time) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.recent(login, now)) >= maxLoginAttempts
}

func (a *attemptTracker) fail(login string, now time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.failures[login] = append(a.recent(login, now), now)
}

func (a *attemptTracker) reset(login string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.failures, login)
}

func (a *attemptTracker) recent(login string, now time.Time) []time.Time {
	entries := a.failures[login]
	recent := entries[:0]
	for _, entry := range entries {
		if now.Sub(entry) < attemptWindow {
			recent = append(recent, entry)
		}
	}
	a.failures[login] = recent
	return recent
}

func (m *Manager) SetDB(db *sql.DB) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.db = db
	if m.attempts == nil {
		m.attempts = newAttemptTracker()
	}
}

func (m *Manager) authenticate(ctx context.Context, login, password string) (int, error) {
	m.mu.RLock()
	db := m.db
	attempts := m.attempts
	m.mu.RUnlock()

	if db == nil {
		return 0, &LoginError{Message: "no database connection", Type: "server"}
	}
	if attempts == nil {
		attempts = newAttemptTracker()
	}
	if attempts.tooMany(login, time.Now()) {
		return 0, &LoginError{Message: "too many login attempts", Type: "rate_limit"}
	}

	var userID int
	var hash string
	var active sql.NullBool
	err := db.QueryRowContext(ctx,
		`SELECT id, COALESCE(password_hash, ''), active FROM res_user WHERE login = $1`,
		login).Scan(&userID, &hash, &active)
	if err != nil {
		// Older Tryton databases store the hash in the password column.
		err = db.QueryRowContext(ctx,
			`SELECT id, COALESCE(password, ''), active FROM res_user WHERE login = $1`,
			login).Scan(&userID, &hash, &active)
	}
	if err != nil {
		attempts.fail(login, time.Now())
		return 0, &LoginError{Message: "invalid credentials", Type: "password"}
	}
	if active.Valid && !active.Bool {
		attempts.fail(login, time.Now())
		return 0, &LoginError{Message: "user is deactivated", Type: "password"}
	}
	if !CheckPassword(hash, password) {
		attempts.fail(login, time.Now())
		return 0, &LoginError{Message: "invalid credentials", Type: "password"}
	}

	attempts.reset(login)
	return userID, nil
}

func (m *Manager) Login(ctx context.Context, database, login, password string) (*Session, error) {
	userID, err := m.authenticate(ctx, login, password)
	if err != nil {
		return nil, err
	}
	return m.Create(database, login, userID), nil
}

func (m *Manager) EnsureUser(ctx context.Context, login, password, name string) (int, error) {
	return m.CreateUser(ctx, login, password, name, true)
}

func (m *Manager) CreateUser(ctx context.Context, login, password, name string, administrator bool) (int, error) {
	m.mu.RLock()
	db := m.db
	m.mu.RUnlock()
	if db == nil {
		return 0, fmt.Errorf("no database connection")
	}

	var userID int
	err := db.QueryRowContext(ctx, `SELECT id FROM res_user WHERE login = $1`, login).Scan(&userID)
	if err == nil {
		hash, err := HashPassword(password)
		if err != nil {
			return 0, err
		}
		if _, err := db.ExecContext(ctx,
			`UPDATE res_user
			 SET password_hash = $1, password = NULL, administrator = $2, active = true
			 WHERE id = $3`,
			hash, administrator, userID); err != nil {
			if _, legacyErr := db.ExecContext(ctx,
				`UPDATE res_user SET password = $1, administrator = $2, active = true WHERE id = $3`,
				hash, administrator, userID); legacyErr != nil {
				return 0, err
			}
		}
		return userID, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	hash, err := HashPassword(password)
	if err != nil {
		return 0, err
	}
	if name == "" {
		name = login
	}
	err = db.QueryRowContext(ctx,
		`INSERT INTO res_user (login, name, password_hash, active, administrator, create_uid, create_date, write_uid, write_date)
		 VALUES ($1, $2, $3, true, $4, 1, now(), 1, now())
		 RETURNING id`,
		login, name, hash, administrator).Scan(&userID)
	if err != nil {
		err = db.QueryRowContext(ctx,
			`INSERT INTO res_user (login, name, password, active, administrator, create_uid, create_date, write_uid, write_date)
			 VALUES ($1, $2, $3, true, $4, 1, now(), 1, now())
			 RETURNING id`,
			login, name, hash, administrator).Scan(&userID)
		if err != nil {
			return 0, err
		}
	}
	return userID, nil
}
