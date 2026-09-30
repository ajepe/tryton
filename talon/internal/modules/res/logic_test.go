package res

import (
	"strings"
	"testing"

	"github.com/ajepe/talon/internal/metadata"
	"github.com/ajepe/talon/internal/orm"
)

func testRecord(values map[string]interface{}) *orm.Record {
	r := orm.NewRecord(&metadata.Model{Name: "test.model"})
	r.ImportData(values)
	return r
}

func TestValidateUser(t *testing.T) {
	if err := ValidateUser(nil, nil, testRecord(map[string]interface{}{
		"login": "  ",
	})); err == nil {
		t.Error("ValidateUser() expected error for empty login")
	}
	if err := ValidateUser(nil, nil, testRecord(map[string]interface{}{
		"login": "admin",
		"email": "not-an-email",
	})); err == nil {
		t.Error("ValidateUser() expected error for invalid email")
	}
	if err := ValidateUser(nil, nil, testRecord(map[string]interface{}{
		"login": "admin",
		"email": "admin@example.com",
	})); err != nil {
		t.Errorf("ValidateUser() error = %v", err)
	}
}

func TestWarningFormat(t *testing.T) {
	first := WarningFormat("party.party.delete", []string{"party.party,-1"})
	second := WarningFormat("party.party.delete", []string{"party.party,-1"})
	other := WarningFormat("party.party.delete", []string{"party.party,-2"})

	if first != second {
		t.Errorf("WarningFormat() not deterministic: %q != %q", first, second)
	}
	if first == other {
		t.Errorf("WarningFormat() should differ for different records: %q", first)
	}
	if !strings.HasSuffix(first, ".party.party.delete") {
		t.Errorf("WarningFormat() = %q, missing name suffix", first)
	}
	hash := strings.TrimSuffix(first, ".party.party.delete")
	if len(hash) != 32 {
		t.Errorf("WarningFormat() hash = %q, want 32 hex chars", hash)
	}
}

func TestNextApplicationState(t *testing.T) {
	tests := []struct {
		action  string
		current string
		want    string
		wantErr bool
	}{
		{"validate", "requested", "validated", false},
		{"cancel", "requested", "cancelled", false},
		{"cancel", "validated", "cancelled", false},
		{"validate", "validated", "", true},
		{"cancel", "cancelled", "", true},
	}
	for _, tt := range tests {
		got, err := NextApplicationState(tt.action, tt.current)
		if (err != nil) != tt.wantErr {
			t.Errorf("NextApplicationState(%q, %q) error = %v, wantErr %v", tt.action, tt.current, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("NextApplicationState(%q, %q) = %q, want %q", tt.action, tt.current, got, tt.want)
		}
	}
}

type fakeNotifier struct {
	channel string
	data    map[string]interface{}
}

func (f *fakeNotifier) Notify(channel string, data map[string]interface{}) {
	f.channel = channel
	f.data = data
}

func TestPublish(t *testing.T) {
	fake := &fakeNotifier{}
	SetNotifier(fake)
	defer SetNotifier(nil)

	publish("notification:1", map[string]interface{}{"id": 5})
	if fake.channel != "notification:1" {
		t.Errorf("channel = %q, want notification:1", fake.channel)
	}
	if fake.data["id"] != 5 {
		t.Errorf("data = %v, want id 5", fake.data)
	}
}

func TestOnChangeUser(t *testing.T) {
	got, err := OnChangeUser(nil, nil, map[string]interface{}{
		"login": "  admin ",
		"email": " a@b.c ",
	}, nil)
	if err != nil {
		t.Fatalf("OnChangeUser() error = %v", err)
	}
	if got["login"] != "admin" {
		t.Errorf("login = %v, want admin", got["login"])
	}
	if got["email"] != "a@b.c" {
		t.Errorf("email = %v, want a@b.c", got["email"])
	}
}
