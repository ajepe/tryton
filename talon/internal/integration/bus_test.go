package integration

import (
	"context"
	"testing"
	"time"

	"github.com/ajepe/talon/internal/bus"
	"github.com/ajepe/talon/internal/testsupport"
)

func TestPGNotifierReceivesNotifications(t *testing.T) {
	h := newHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hub := bus.NewHub()
	notifier, err := bus.ConnectPG(ctx, testsupport.DatabaseURI(t, h.db), hub)
	if err != nil {
		t.Skipf("pgx connection unavailable: %v", err)
	}
	defer notifier.Close(context.Background())

	channel := "talon_test_bus"
	if err := notifier.Listen(ctx, channel); err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	go func() {
		_ = notifier.Run(ctx)
	}()

	if _, err := h.db.ExecContext(ctx, "SELECT pg_notify($1, $2)", channel, `{"title":"hello"}`); err != nil {
		t.Fatalf("pg_notify error = %v", err)
	}

	messages, err := hub.Poll(ctx, []string{channel}, "", 5*time.Second)
	if err != nil {
		t.Fatalf("Poll() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(messages))
	}
	if messages[0].Data["title"] != "hello" {
		t.Errorf("data = %v, want title hello", messages[0].Data)
	}
}

func TestPGNotifierNotify(t *testing.T) {
	h := newHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hub := bus.NewHub()
	notifier, err := bus.ConnectPG(ctx, testsupport.DatabaseURI(t, h.db), hub)
	if err != nil {
		t.Skipf("pgx connection unavailable: %v", err)
	}
	defer notifier.Close(context.Background())

	if err := notifier.Notify(ctx, "talon_notify_test", map[string]interface{}{"id": 7}); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

	messages, err := hub.Poll(ctx, []string{"talon_notify_test"}, "", time.Second)
	if err != nil {
		t.Fatalf("Poll() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(messages))
	}
	_ = h
}
