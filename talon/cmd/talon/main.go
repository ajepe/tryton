package main

import (
	"context"

	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ajepe/talon/internal/action"
	"github.com/ajepe/talon/internal/backend/postgresql"
	"github.com/ajepe/talon/internal/bus"
	"github.com/ajepe/talon/internal/cron"
	"github.com/ajepe/talon/internal/filestore"
	"github.com/ajepe/talon/internal/i18n"
	"github.com/ajepe/talon/internal/metadata"
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
	"github.com/ajepe/talon/internal/report"
	"github.com/ajepe/talon/internal/rpc"
	"github.com/ajepe/talon/internal/schema"
	"github.com/ajepe/talon/internal/security"
	"github.com/ajepe/talon/internal/session"
	"github.com/ajepe/talon/internal/web"
	"github.com/ajepe/talon/internal/wizard"
)

var (
	version   = "0.1.0"
	buildTime = "unknown"
)

func main() {
	var (
		host          = flag.String("host", "localhost", "listen host")
		port          = flag.Int("port", 8000, "listen port")
		database      = flag.String("database", "", "database name")
		uri           = flag.String("uri", "", "PostgreSQL URI")
		dataDir       = flag.String("data-dir", "", "data directory for file storage")
		webDir        = flag.String("web-dir", "", "directory serving the web client build")
		reportDir     = flag.String("report-dir", "", "directory containing report templates")
		adminLogin    = flag.String("admin-login", "admin", "administrator login")
		adminPassword = flag.String("admin-password", "", "set the administrator password on startup")
		showVer       = flag.Bool("version", false, "show version")
	)
	flag.Parse()

	if *showVer {
		fmt.Printf("talon %s (built %s)\n", version, buildTime)
		return
	}

	if *database == "" || *uri == "" {
		fmt.Fprintf(os.Stderr, "Usage: talon -database <name> -uri <postgresql-uri>\n")
		os.Exit(1)
	}

	db, err := postgresql.NewDatabase(*database, *uri)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	pool := orm.DefaultPool()
	pool.SetDB(db.DB())

	sync := schema.NewSync(db.DB())
	models := make([]*metadata.Model, 0)
	for _, name := range pool.ListModels() {
		if m, err := pool.GetModel(name); err == nil {
			models = append(models, m)
		}
	}
	if err := sync.Sync(models); err != nil {
		log.Fatalf("schema sync: %v", err)
	}

	sessMgr := session.NewManager(30 * time.Minute)
	sessMgr.SetDB(db.DB())
	if *adminPassword != "" {
		if _, err := sessMgr.EnsureUser(context.Background(), *adminLogin, *adminPassword, "Administrator"); err != nil {
			log.Fatalf("admin user: %v", err)
		}
	}

	rpcServer := rpc.NewServer()
	rpcServer.SetAuthenticator(session.RPCAuthenticator(sessMgr, *database))

	registerCommonMethods(rpcServer, sessMgr, db)

	securityMgr := security.NewManager(db.DB())
	i18nMgr := i18n.NewManager(db.DB())
	actionMgr := action.NewManager(db.DB())
	action.RegisterRPC(rpcServer, actionMgr)

	modelServer := rpc.NewModelServer(pool)
	modelServer.SetTranslator(i18nMgr)
	modelServer.SetAccessChecker(securityMgr.ContextChecker())
	modelServer.SetRuleDomainProvider(securityMgr.ContextRuleDomain())
	modelServer.SetFieldAccessChecker(func(ctx context.Context, model, field, mode string) error {
		userID := security.UserFromContext(ctx)
		if userID == 0 {
			return nil
		}
		return securityMgr.CheckField(ctx, userID, model, field, mode)
	})
	modelServer.RegisterAll(rpcServer)

	currency.RegisterRPC(rpcServer)

	moduleRegistry := rpc.NewModuleRegistry()
	registerModuleHandlers(moduleRegistry)
	moduleRegistry.RegisterAll(rpcServer)
	modelServer.SetModuleHooks(moduleRegistry)

	platform, err := newPlatformServices(db, *database, *dataDir)
	if err != nil {
		log.Fatalf("platform services: %v", err)
	}
	res.SetNotifier(busNotifier{hub: platform.Bus})
	if *reportDir != "" {
		if err := loadReportTemplates(platform.Reports, rpcServer, db.DB(), pool, *reportDir); err != nil {
			log.Fatalf("reports: %v", err)
		}
	}
	if attachments := filestore.NewAttachmentService(platform.Files, db.DB()); attachments != nil {
		attachments.RegisterRPC(rpcServer)
	}

	busCtx, busCancel := context.WithCancel(context.Background())
	defer busCancel()
	if *uri != "" {
		if pgNotifier, err := bus.ConnectPG(busCtx, *uri, platform.Bus); err == nil {
			if err := pgNotifier.Listen(busCtx, "tryton_bus"); err == nil {
				go pgNotifier.Run(busCtx)
			}
			defer pgNotifier.Close(context.Background())
		}
	}

	var webHandler *web.Handler
	if *webDir != "" {
		webHandler, err = web.NewHandler(*webDir)
		if err != nil {
			log.Fatalf("web client: %v", err)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case path == "/":
			if webHandler != nil {
				webHandler.ServeHTTP(w, r)
				return
			}
			w.Write([]byte("talon " + version))
		case strings.HasSuffix(path, "/rpc/"):
			rpcServer.Handle(w, r)
		case strings.HasSuffix(path, "/bus"):
			platform.Bus.ServeHTTP(w, r)
		case strings.HasSuffix(path, "/session/login"):
			session.LoginHandler(sessMgr, *database)(w, r)
		case strings.HasSuffix(path, "/session/logout"):
			session.LogoutHandler(sessMgr, *database)(w, r)
		default:
			if webHandler != nil {
				webHandler.ServeHTTP(w, r)
				return
			}
			http.NotFound(w, r)
		}
	})

	addr := fmt.Sprintf("%s:%d", *host, *port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 35 * time.Second,
	}

	go func() {
		log.Printf("talon %s listening on %s", version, addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

type platformServices struct {
	Bus     *bus.Hub
	Wizards *wizard.Engine
	Reports *report.Registry
	Cron    *cron.Scheduler
	Files   *filestore.Store
}

type busNotifier struct {
	hub *bus.Hub
}

func (n busNotifier) Notify(channel string, data map[string]interface{}) {
	n.hub.Notify(channel, data)
}

func newPlatformServices(db *postgresql.Database, database, dataDir string) (*platformServices, error) {
	services := &platformServices{
		Bus:     bus.NewHub(),
		Wizards: wizard.NewEngine(),
		Reports: report.NewRegistry(),
		Cron:    cron.NewScheduler(&cron.SQLStore{DB: db.DB()}),
	}
	if dataDir != "" {
		store, err := filestore.New(filepath.Join(dataDir, database))
		if err != nil {
			return nil, err
		}
		services.Files = store
	}
	return services, nil
}

func registerModuleHandlers(registry *rpc.ModuleRegistry) {
	ir.RegisterHandlers(registry)
	res.RegisterHandlers(registry)
	country.RegisterHandlers(registry)
	party.RegisterHandlers(registry)
	product.RegisterHandlers(registry)
	account.RegisterHandlers(registry)
	currency.RegisterHandlers(registry)
	company.RegisterHandlers(registry)
	stock.RegisterHandlers(registry)
	sale.RegisterHandlers(registry)
	purchase.RegisterHandlers(registry)
}

func registerCommonMethods(s *rpc.Server, sessMgr *session.Manager, db *postgresql.Database) {
	s.Register("common.server.version", func(ctx context.Context, params []interface{}) (interface{}, error) {
		return version, nil
	})

	s.Register("common.db.list", func(ctx context.Context, params []interface{}) (interface{}, error) {
		return []string{db.Database()}, nil
	})

	session.RegisterRPC(s, sessMgr, db.Database())

	s.Register("common.db.reset_password", func(ctx context.Context, params []interface{}) (interface{}, error) {
		return nil, &rpc.UserError{Message: "reset_password not implemented"}
	})

	s.Register("common.authentication.services", func(ctx context.Context, params []interface{}) (interface{}, error) {
		return []interface{}{}, nil
	})
}
