package session

import (
	"context"
	"testing"
	"time"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "secret" {
		t.Error("HashPassword() returned the plain password")
	}
	if !CheckPassword(hash, "secret") {
		t.Error("CheckPassword() = false for valid password")
	}
	if CheckPassword(hash, "wrong") {
		t.Error("CheckPassword() = true for invalid password")
	}
	if CheckPassword("", "secret") {
		t.Error("CheckPassword() = true for empty hash")
	}
}

func TestAttemptTracker(t *testing.T) {
	tracker := newAttemptTracker()
	now := time.Now()
	for i := 0; i < maxLoginAttempts; i++ {
		if tracker.tooMany("admin", now) {
			t.Fatalf("tooMany() = true after %d attempts", i)
		}
		tracker.fail("admin", now)
	}
	if !tracker.tooMany("admin", now) {
		t.Error("tooMany() = false after max attempts")
	}
	tracker.reset("admin")
	if tracker.tooMany("admin", now) {
		t.Error("tooMany() = true after reset")
	}

	old := now.Add(-2 * attemptWindow)
	tracker.fail("other", old)
	if tracker.tooMany("other", now) {
		t.Error("tooMany() = true for expired attempts")
	}
}

func TestManagerCreateGetRenewDelete(t *testing.T) {
	mgr := NewManager(50 * time.Millisecond)
	sess := mgr.Create("db", "admin", 1)
	if sess.UserID() != 1 || sess.Login() != "admin" || sess.Database() != "db" {
		t.Errorf("session = %+v", sess)
	}
	if sess.IsExpired() {
		t.Error("session expired immediately")
	}

	found, ok := mgr.Get(sess.Token())
	if !ok || found != sess {
		t.Error("Get() did not return the session")
	}

	if !mgr.Renew(sess.Token()) {
		t.Error("Renew() = false")
	}

	mgr.Delete(sess.Token())
	if _, ok := mgr.Get(sess.Token()); ok {
		t.Error("session still present after Delete()")
	}
}

func TestManagerExpiry(t *testing.T) {
	mgr := NewManager(time.Millisecond)
	sess := mgr.Create("db", "admin", 1)
	time.Sleep(5 * time.Millisecond)
	if _, ok := mgr.Get(sess.Token()); ok {
		t.Error("expired session returned by Get()")
	}
	mgr.Cleanup()
	if len(mgr.sessions) != 0 {
		t.Error("Cleanup() did not remove expired session")
	}
}

func TestSessionContextAndCookie(t *testing.T) {
	mgr := NewManager(time.Minute)
	sess := mgr.Create("db", "admin", 7)
	sess.SetContext(map[string]interface{}{"language": "en"})
	if sess.Context()["language"] != "en" {
		t.Error("SetContext() not applied")
	}
	cookie := sess.CookieValue()
	if cookie != "admin:7:"+sess.Token() {
		t.Errorf("CookieValue() = %q", cookie)
	}
	if sess.AuthorizationHeader() == "" {
		t.Error("AuthorizationHeader() is empty")
	}
}

func TestRequiresSession(t *testing.T) {
	tests := map[string]bool{
		"common.db.login":             false,
		"common.server.version":       false,
		"model.party.party.read":      true,
		"module.party.party.button":   true,
		"report.sale.execute":         true,
		"wizard.party.replace.create": true,
	}
	for method, want := range tests {
		if got := RequiresSession(method); got != want {
			t.Errorf("RequiresSession(%q) = %v, want %v", method, got, want)
		}
	}
}

func TestLoginWithoutDB(t *testing.T) {
	mgr := NewManager(time.Minute)
	if _, err := mgr.Login(context.Background(), "db", "admin", "secret"); err == nil {
		t.Error("Login() expected error without database")
	}
}
