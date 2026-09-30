package conformance

import (
	"context"
	"os"
	"testing"
)

func TestLiveDebug(t *testing.T) {
	if os.Getenv("TALON_LIVE") == "" {
		t.Skip("set TALON_LIVE=1 to run against live servers")
	}
	runner := NewRunner("http://127.0.0.1:8000", "http://127.0.0.1:8001", "tryton_conf")
	ctx := context.Background()
	if err := runner.Authenticate(ctx, "admin", "secret"); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	t.Logf("pythonAuth=%q", runner.pythonAuth)
	t.Logf("goAuth=%q", runner.goAuth)

	result, err := runner.callPython(ctx, "model.party.party.search_count",
		[]interface{}{[]interface{}{}, map[string]interface{}{}})
	t.Logf("python result=%v err=%v", result, err)

	result, err = runner.callGo(ctx, "model.party.party.search_count",
		[]interface{}{[]interface{}{}, map[string]interface{}{}})
	t.Logf("go result=%v err=%v", result, err)
}
