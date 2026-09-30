package wizard

import (
	"context"
	"testing"

	"github.com/ajepe/talon/internal/rpc"
)

func testDefinition() *Definition {
	def := NewDefinition("party.replace", "ask")
	def.On("ask", "replace", func(ctx context.Context, w *Instance, values map[string]interface{}) (string, error) {
		w.Values["replaced"] = true
		return EndState, nil
	})
	def.On("ask", "cancel", func(ctx context.Context, w *Instance, values map[string]interface{}) (string, error) {
		return EndState, nil
	})
	return def
}

func TestEngineRegisterAndStart(t *testing.T) {
	engine := NewEngine()
	if err := engine.Register(testDefinition()); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := engine.Register(testDefinition()); err == nil {
		t.Error("Register() expected error for duplicate")
	}

	instance, err := engine.Start("party.replace", "sess-1", map[string]interface{}{"source": 1})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if instance.State != "ask" {
		t.Errorf("state = %q, want ask", instance.State)
	}
	if instance.ID == "" {
		t.Error("instance ID is empty")
	}
	if instance.Values["source"] != 1 {
		t.Errorf("values[source] = %v, want 1", instance.Values["source"])
	}
}

func TestEngineStartUnknown(t *testing.T) {
	engine := NewEngine()
	if _, err := engine.Start("unknown", "", nil); err == nil {
		t.Error("Start() expected error for unknown wizard")
	}
}

func TestEngineExecuteTransition(t *testing.T) {
	engine := NewEngine()
	engine.Register(testDefinition())

	instance, _ := engine.Start("party.replace", "sess-1", map[string]interface{}{})
	result, err := engine.Execute(context.Background(), instance.ID, "replace", map[string]interface{}{"destination": 2})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.State != EndState {
		t.Errorf("state = %q, want %s", result.State, EndState)
	}
	if result.Values["destination"] != 2 {
		t.Errorf("values[destination] = %v, want 2", result.Values["destination"])
	}
	if _, ok := engine.Instance(instance.ID); ok {
		t.Error("instance still present after end state")
	}
}

func TestEngineExecuteUnknownAction(t *testing.T) {
	engine := NewEngine()
	engine.Register(testDefinition())
	instance, _ := engine.Start("party.replace", "", nil)

	if _, err := engine.Execute(context.Background(), instance.ID, "bogus", nil); err == nil {
		t.Error("Execute() expected error for unknown action")
	}
	if _, err := engine.Execute(context.Background(), "missing", "replace", nil); err == nil {
		t.Error("Execute() expected error for unknown instance")
	}
}

func TestEngineDelete(t *testing.T) {
	engine := NewEngine()
	engine.Register(testDefinition())
	instance, _ := engine.Start("party.replace", "", nil)

	if err := engine.Delete(instance.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if err := engine.Delete(instance.ID); err == nil {
		t.Error("Delete() expected error for missing instance")
	}
}

func TestDefinitionActions(t *testing.T) {
	def := testDefinition()
	actions := def.Actions("ask")
	if len(actions) != 2 {
		t.Errorf("Actions(ask) = %v, want 2 actions", actions)
	}
}

func TestEngineRegisterRPC(t *testing.T) {
	engine := NewEngine()
	engine.Register(testDefinition())
	srv := rpc.NewServer()
	engine.RegisterRPC(srv, "party.replace")

	created, err := srv.Call(context.Background(), "wizard.party.replace.create",
		[]interface{}{map[string]interface{}{"source": 1}})
	if err != nil {
		t.Fatalf("create error = %v", err)
	}
	instance, ok := created.(*Instance)
	if !ok {
		t.Fatalf("create returned %T, want *Instance", created)
	}

	executed, err := srv.Call(context.Background(), "wizard.party.replace.execute",
		[]interface{}{instance.ID, "replace", map[string]interface{}{}})
	if err != nil {
		t.Fatalf("execute error = %v", err)
	}
	result := executed.(*Instance)
	if result.State != EndState {
		t.Errorf("state = %q, want %s", result.State, EndState)
	}
}
