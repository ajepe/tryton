package session

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

type Session struct {
	mu         sync.RWMutex
	database   string
	login      string
	userID     int
	token      string
	context    map[string]interface{}
	expiresAt  time.Time
	busURLHost string
}

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	maxAge   time.Duration
	db       *sql.DB
	attempts *attemptTracker
}

func NewManager(maxAge time.Duration) *Manager {
	return &Manager{
		sessions: map[string]*Session{},
		maxAge:   maxAge,
		attempts: newAttemptTracker(),
	}
}

func (m *Manager) Create(database, login string, userID int) *Session {
	token := generateToken()
	s := &Session{
		database:  database,
		login:     login,
		userID:    userID,
		token:     token,
		context:   map[string]interface{}{},
		expiresAt: time.Now().Add(m.maxAge),
	}
	m.mu.Lock()
	m.sessions[token] = s
	m.mu.Unlock()
	return s
}

func (m *Manager) Get(token string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[token]
	if !ok {
		return nil, false
	}
	if time.Now().After(s.expiresAt) {
		return nil, false
	}
	return s, true
}

func (m *Manager) Delete(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, token)
}

func (m *Manager) Renew(token string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[token]
	if !ok {
		return false
	}
	s.expiresAt = time.Now().Add(m.maxAge)
	return true
}

func (m *Manager) UpdateContext(token string, ctx map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[token]; ok {
		s.context = ctx
	}
}

func (m *Manager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for token, s := range m.sessions {
		if now.After(s.expiresAt) {
			delete(m.sessions, token)
		}
	}
}

func (s *Session) Context() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.context
}

func (s *Session) SetContext(ctx map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.context = ctx
}

func (s *Session) UserID() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.userID
}

func (s *Session) Login() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.login
}

func (s *Session) Database() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.database
}

func (s *Session) Token() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.token
}

func (s *Session) IsExpired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Now().After(s.expiresAt)
}

func (s *Session) CookieValue() string {
	return fmt.Sprintf("%s:%d:%s", s.login, s.userID, s.token)
}

func (s *Session) AuthorizationHeader() string {
	return "session " + base64.StdEncoding.EncodeToString([]byte(s.CookieValue()))
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func (m *Manager) Logout(token string) {
	m.Delete(token)
}
