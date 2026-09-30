package bus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHubPublishAndPoll(t *testing.T) {
	hub := NewHub()
	hub.SetTimeout(50 * time.Millisecond)

	published := hub.Notify("user:1", map[string]interface{}{"title": "hello"})
	if published.ID == "" {
		t.Fatal("Publish() did not assign an ID")
	}

	messages, err := hub.Poll(context.Background(), []string{"user:1"}, "", time.Second)
	if err != nil {
		t.Fatalf("Poll() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("Poll() returned %d messages, want 1", len(messages))
	}
	if messages[0].Data["title"] != "hello" {
		t.Errorf("message title = %v, want hello", messages[0].Data["title"])
	}
	if messages[0].Channel != "user:1" {
		t.Errorf("message channel = %v, want user:1", messages[0].Channel)
	}
}

func TestHubPollFromLastMessage(t *testing.T) {
	hub := NewHub()
	first := hub.Publish("user:1", Message{Type: "message"})
	second := hub.Publish("user:1", Message{Type: "message"})

	messages, err := hub.Poll(context.Background(), []string{"user:1"}, first.ID, time.Second)
	if err != nil {
		t.Fatalf("Poll() error = %v", err)
	}
	if len(messages) != 1 || messages[0].ID != second.ID {
		t.Fatalf("Poll() = %v, want only second message", messages)
	}
}

func TestHubPollWaitsForMessage(t *testing.T) {
	hub := NewHub()
	go func() {
		time.Sleep(20 * time.Millisecond)
		hub.Publish("user:2", Message{Type: "notification"})
	}()

	messages, err := hub.Poll(context.Background(), []string{"user:2"}, "", time.Second)
	if err != nil {
		t.Fatalf("Poll() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("Poll() returned %d messages, want 1", len(messages))
	}
}

func TestHubPollTimeout(t *testing.T) {
	hub := NewHub()
	start := time.Now()
	messages, err := hub.Poll(context.Background(), []string{"user:3"}, "", 30*time.Millisecond)
	if err != nil {
		t.Fatalf("Poll() error = %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("Poll() returned %d messages, want 0", len(messages))
	}
	if elapsed := time.Since(start); elapsed < 25*time.Millisecond {
		t.Errorf("Poll() returned after %v, want >= 25ms", elapsed)
	}
}

func TestHubPollContextCancelled(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	_, err := hub.Poll(ctx, []string{"user:4"}, "", time.Second)
	if err == nil {
		t.Error("Poll() error = nil, want context cancellation")
	}
}

func TestHubHistoryBound(t *testing.T) {
	hub := NewHub()
	hub.historySize = 2
	for i := 0; i < 5; i++ {
		hub.Publish("user:5", Message{Type: "message"})
	}
	if len(hub.channels["user:5"]) != 2 {
		t.Errorf("history length = %d, want 2", len(hub.channels["user:5"]))
	}
}

func TestHubServeHTTP(t *testing.T) {
	hub := NewHub()
	hub.SetTimeout(50 * time.Millisecond)
	go func() {
		time.Sleep(20 * time.Millisecond)
		hub.Publish("user:6", Message{Type: "notification", Data: map[string]interface{}{"body": "hi"}})
	}()

	body := strings.NewReader(`{"channels": ["user:6"], "last_message": null}`)
	req := httptest.NewRequest(http.MethodPost, "/db/bus", body)
	rec := httptest.NewRecorder()
	hub.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var response []struct {
		Channel string  `json:"channel"`
		Message Message `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid response: %v", err)
	}
	if len(response) != 1 {
		t.Fatalf("response length = %d, want 1", len(response))
	}
	if response[0].Channel != "user:6" {
		t.Errorf("channel = %q, want user:6", response[0].Channel)
	}
}

func TestHubServeHTTPRejectsGet(t *testing.T) {
	hub := NewHub()
	req := httptest.NewRequest(http.MethodGet, "/db/bus", nil)
	rec := httptest.NewRecorder()
	hub.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestNewIDUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id := newID()
		if seen[id] {
			t.Fatalf("duplicate ID %q", id)
		}
		seen[id] = true
	}
}
