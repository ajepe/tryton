package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"

	"github.com/ajepe/talon/internal/client"
)

func ensureAdmin(t *testing.T, h *harness) {
	t.Helper()
	if _, err := h.sessions.EnsureUser(context.Background(), "admin", "secret", "Administrator"); err != nil {
		t.Fatalf("EnsureUser() error = %v", err)
	}
}

func TestLoginRPCSuccessAndFailure(t *testing.T) {
	h := newHarness(t)
	ensureAdmin(t, h)

	c := client.NewClient(h.server.URL)
	c.SetSession(&client.Session{Database: "test", Context: map[string]interface{}{}})

	if err := c.Login(context.Background(), "test", "admin", "secret"); err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if c.Session().UserID <= 0 {
		t.Errorf("user id = %d, want positive", c.Session().UserID)
	}

	bad := client.NewClient(h.server.URL)
	bad.SetSession(&client.Session{Database: "test", Context: map[string]interface{}{}})
	if err := bad.Login(context.Background(), "test", "admin", "wrong"); err == nil {
		t.Error("Login() with wrong password expected error")
	}
	if err := bad.Login(context.Background(), "test", "missing", "secret"); err == nil {
		t.Error("Login() with unknown user expected error")
	}
}

func rpcPost(t *testing.T, c *http.Client, url, body string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, data
}

func TestSessionHTTPFlow(t *testing.T) {
	h := newHarness(t)
	ensureAdmin(t, h)

	jar, _ := cookiejar.New(nil)
	httpClient := &http.Client{Jar: jar}

	loginBody := `{"id": 0, "method": "common.db.login", "params": ["admin", {"password": "secret"}, "en"]}`
	resp, data := rpcPost(t, httpClient, h.server.URL+"/test/session/login", loginBody)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", resp.StatusCode, data)
	}
	if len(jar.Cookies(resp.Request.URL)) == 0 {
		t.Fatal("login did not set a session cookie")
	}

	var loginResult struct {
		Result []interface{} `json:"result"`
	}
	if err := json.Unmarshal(data, &loginResult); err != nil {
		t.Fatalf("login response = %s: %v", data, err)
	}
	if len(loginResult.Result) < 2 {
		t.Fatalf("login result = %v", loginResult.Result)
	}

	createBody := `{"id": 1, "method": "model.party.party.create", "params": [[{"name": "Authenticated"}], {}]}`
	resp, data = rpcPost(t, httpClient, h.server.URL+"/test/rpc/", createBody)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("authorized create status = %d, body = %s", resp.StatusCode, data)
	}
	var createResult struct {
		Result []int `json:"result"`
	}
	if err := json.Unmarshal(data, &createResult); err != nil {
		t.Fatalf("create response = %s: %v", data, err)
	}
	if len(createResult.Result) != 1 {
		t.Fatalf("create result = %v", createResult.Result)
	}

	anonymous := &http.Client{}
	resp, _ = rpcPost(t, anonymous, h.server.URL+"/test/rpc/", createBody)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("anonymous create status = %d, want 403", resp.StatusCode)
	}

	resp, data = rpcPost(t, httpClient, h.server.URL+"/test/session/logout", `{}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("logout status = %d, body = %s", resp.StatusCode, data)
	}
	resp, _ = rpcPost(t, httpClient, h.server.URL+"/test/rpc/", createBody)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("post-logout create status = %d, want 403", resp.StatusCode)
	}
}

func TestRateLimit(t *testing.T) {
	h := newHarness(t)
	ensureAdmin(t, h)

	c := client.NewClient(h.server.URL)
	c.SetSession(&client.Session{Database: "test", Context: map[string]interface{}{}})
	for i := 0; i < 6; i++ {
		_ = c.Login(context.Background(), "test", "admin", "wrong")
	}
	err := c.Login(context.Background(), "test", "admin", "secret")
	if err == nil {
		t.Fatal("Login() expected rate limit error")
	}
	if !strings.Contains(err.Error(), "too many") {
		t.Errorf("error = %v, want rate limit", err)
	}
}

func TestEnsureUserResetsPassword(t *testing.T) {
	h := newHarness(t)
	ensureAdmin(t, h)

	userID, err := h.sessions.EnsureUser(context.Background(), "admin", "newpass", "Administrator")
	if err != nil {
		t.Fatalf("EnsureUser() error = %v", err)
	}
	sess, err := h.sessions.Login(context.Background(), "test", "admin", "newpass")
	if err != nil {
		t.Fatalf("Login() with new password error = %v", err)
	}
	if sess.UserID() != userID {
		t.Errorf("user id = %d, want %d", sess.UserID(), userID)
	}
}
