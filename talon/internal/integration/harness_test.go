package integration

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ajepe/talon/internal/i18n"
	"github.com/ajepe/talon/internal/modules/account"
	"github.com/ajepe/talon/internal/modules/company"
	"github.com/ajepe/talon/internal/modules/country"
	"github.com/ajepe/talon/internal/modules/currency"
	"github.com/ajepe/talon/internal/modules/ir"
	"github.com/ajepe/talon/internal/modules/party"
	"github.com/ajepe/talon/internal/modules/product"
	"github.com/ajepe/talon/internal/modules/purchase"
	"github.com/ajepe/talon/internal/modules/res"
	"github.com/ajepe/talon/internal/modules/sale"
	"github.com/ajepe/talon/internal/modules/stock"
	"github.com/ajepe/talon/internal/orm"
	"github.com/ajepe/talon/internal/rpc"
	"github.com/ajepe/talon/internal/security"
	"github.com/ajepe/talon/internal/session"
	"github.com/ajepe/talon/internal/testsupport"
)

type harness struct {
	db       *sql.DB
	server   *httptest.Server
	srv      *rpc.Server
	sessions *session.Manager
	security *security.Manager
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	return newHarnessWithDB(t, testsupport.OpenDB(t))
}

func newHarnessWithDB(t *testing.T, db *sql.DB) *harness {
	t.Helper()
	pool := orm.DefaultPool()
	pool.SetDB(db)

	sessions := session.NewManager(30 * time.Minute)
	sessions.SetDB(db)

	securityMgr := security.NewManager(db)

	srv := rpc.NewServer()
	srv.SetAuthenticator(session.RPCAuthenticator(sessions, "test"))
	session.RegisterRPC(srv, sessions, "test")
	currency.RegisterRPC(srv)
	modelServer := rpc.NewModelServer(pool)
	modelServer.SetTranslator(i18n.NewManager(db))
	modelServer.SetAccessChecker(securityMgr.ContextChecker())
	modelServer.SetRuleDomainProvider(securityMgr.ContextRuleDomain())
	modelServer.SetFieldAccessChecker(func(ctx context.Context, model, field, mode string) error {
		userID := security.UserFromContext(ctx)
		if userID == 0 {
			return nil
		}
		return securityMgr.CheckField(ctx, userID, model, field, mode)
	})
	modelServer.RegisterAll(srv)

	moduleRegistry := rpc.NewModuleRegistry()
	ir.RegisterHandlers(moduleRegistry)
	res.RegisterHandlers(moduleRegistry)
	country.RegisterHandlers(moduleRegistry)
	party.RegisterHandlers(moduleRegistry)
	product.RegisterHandlers(moduleRegistry)
	account.RegisterHandlers(moduleRegistry)
	currency.RegisterHandlers(moduleRegistry)
	company.RegisterHandlers(moduleRegistry)
	stock.RegisterHandlers(moduleRegistry)
	sale.RegisterHandlers(moduleRegistry)
	purchase.RegisterHandlers(moduleRegistry)
	moduleRegistry.RegisterAll(srv)
	modelServer.SetModuleHooks(moduleRegistry)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/rpc/"):
			srv.Handle(w, r)
		case strings.HasSuffix(path, "/session/login"):
			session.LoginHandler(sessions, "test")(w, r)
		case strings.HasSuffix(path, "/session/logout"):
			session.LogoutHandler(sessions, "test")(w, r)
		default:
			http.NotFound(w, r)
		}
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return &harness{db: db, server: server, srv: srv, sessions: sessions, security: securityMgr}
}

func authenticatedRPC(t *testing.T, h *harness, login, password, body string) string {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar}
	loginBody := fmt.Sprintf(
		`{"id": 0, "method": "common.db.login", "params": [%q, {"password": %q}, "en"]}`,
		login, password)
	resp, data := rpcPost(t, c, h.server.URL+"/test/session/login", loginBody)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", resp.StatusCode, data)
	}
	resp, data = rpcPost(t, c, h.server.URL+"/test/rpc/", body)
	if resp.StatusCode == http.StatusForbidden {
		t.Fatalf("unauthorized: %s", data)
	}
	return string(data)
}
