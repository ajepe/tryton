package rpc

import (
	"context"
	"testing"
)

func TestServer_lookupExact(t *testing.T) {
	s := NewServer()
	called := false
	s.Register("common.server.version", func(ctx context.Context, params []interface{}) (interface{}, error) {
		called = true
		return "1.0", nil
	})

	handler, model, ok := s.lookup("common.server.version")
	if !ok {
		t.Fatal("lookup(common.server.version) not found")
	}
	if model != "" {
		t.Errorf("lookup() model = %q, want empty", model)
	}
	if _, err := handler(context.Background(), nil); err != nil || !called {
		t.Errorf("handler error = %v, called = %v", err, called)
	}
}

func TestServer_lookupModelPlaceholder(t *testing.T) {
	s := NewServer()
	s.Register("model.<model>.read", func(ctx context.Context, params []interface{}) (interface{}, error) {
		return nil, nil
	})

	handler, model, ok := s.lookup("model.party.party.read")
	if !ok {
		t.Fatal("lookup(model.party.party.read) not found")
	}
	if handler == nil {
		t.Fatal("lookup() handler = nil")
	}
	if model != "party.party" {
		t.Errorf("lookup() model = %q, want party.party", model)
	}
}

func TestServer_lookupModulePlaceholder(t *testing.T) {
	s := NewServer()
	s.Register("module.<model>.button", func(ctx context.Context, params []interface{}) (interface{}, error) {
		return nil, nil
	})

	_, model, ok := s.lookup("module.stock.shipment.out.button")
	if !ok {
		t.Fatal("lookup(module.stock.shipment.out.button) not found")
	}
	if model != "stock.shipment.out" {
		t.Errorf("lookup() model = %q, want stock.shipment.out", model)
	}
}

func TestServer_lookupNotFound(t *testing.T) {
	s := NewServer()
	if _, _, ok := s.lookup("model.party.party.read"); ok {
		t.Error("lookup() found handler for empty server")
	}
}

func TestMatchPlaceholder(t *testing.T) {
	tests := []struct {
		pattern string
		method  string
		want    string
		ok      bool
	}{
		{"model.<model>.read", "model.party.party.read", "party.party", true},
		{"module.<model>.on_change", "module.account.move.on_change", "account.move", true},
		{"report.<report>.execute", "report.invoice.execute", "invoice", true},
		{"model.<model>.read", "common.server.version", "", false},
		{"model.<model>.read", "model.read", "", false},
		{"common.server.version", "common.server.version", "", false},
	}
	for _, tt := range tests {
		got, ok := matchPlaceholder(tt.pattern, tt.method)
		if ok != tt.ok || got != tt.want {
			t.Errorf("matchPlaceholder(%q, %q) = (%q, %v), want (%q, %v)",
				tt.pattern, tt.method, got, ok, tt.want, tt.ok)
		}
	}
}
