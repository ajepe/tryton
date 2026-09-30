package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajepe/talon/internal/client"
	"github.com/ajepe/talon/internal/orm"
	"github.com/ajepe/talon/internal/rpc"
	"github.com/ajepe/talon/internal/testsupport"
)

func startServer(t *testing.T) *httptest.Server {
	t.Helper()
	db := testsupport.OpenDB(t)
	pool := orm.DefaultPool()
	pool.SetDB(db)

	srv := rpc.NewServer()
	modelServer := rpc.NewModelServer(pool)
	modelServer.RegisterAll(srv)

	return httptest.NewServer(http.HandlerFunc(srv.Handle))
}

func newClient(t *testing.T, server *httptest.Server) *client.Client {
	t.Helper()
	c := client.NewClient(server.URL)
	c.SetSession(&client.Session{Database: "test", Context: map[string]interface{}{}})
	return c
}

func TestModelCrudOverRPC(t *testing.T) {
	server := startServer(t)
	defer server.Close()
	c := newClient(t, server)
	ctx := context.Background()

	ids, err := c.Create(ctx, "party.party", []map[string]interface{}{
		{"name": "Acme Corp", "code": "ACME", "active": true},
	}, nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if len(ids) != 1 || ids[0] <= 0 {
		t.Fatalf("Create() ids = %v, want one positive id", ids)
	}
	id := ids[0]

	records, err := c.Read(ctx, "party.party", []int{id}, []string{"name", "code", "active"}, nil)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("Read() returned %d records, want 1", len(records))
	}
	if records[0]["name"] != "Acme Corp" {
		t.Errorf("name = %v, want Acme Corp", records[0]["name"])
	}

	if err := c.WriteOne(ctx, "party.party", id, map[string]interface{}{"name": "Acme Inc"}, nil); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	records, err = c.Read(ctx, "party.party", []int{id}, []string{"name"}, nil)
	if err != nil {
		t.Fatalf("Read() after write error = %v", err)
	}
	if records[0]["name"] != "Acme Inc" {
		t.Errorf("name = %v, want Acme Inc", records[0]["name"])
	}

	found, err := c.Search(ctx, "party.party", []interface{}{
		[]interface{}{"name", "like", "Acme"},
	}, 0, 10, "name ASC", nil)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(found) != 1 || found[0] != id {
		t.Errorf("Search() = %v, want [%d]", found, id)
	}

	if err := c.DeleteOne(ctx, "party.party", id, nil); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	records, err = c.Read(ctx, "party.party", []int{id}, []string{"name"}, nil)
	if err != nil {
		t.Fatalf("Read() after delete error = %v", err)
	}
	if len(records) != 0 {
		t.Errorf("Read() after delete returned %d records, want 0", len(records))
	}
}
