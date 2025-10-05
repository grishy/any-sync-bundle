package adminui

import (
	"context"
	"embed"
	"net/http"
	"time"

	"github.com/anyproto/any-sync-coordinator/accountlimit"
	"github.com/anyproto/any-sync-coordinator/acleventlog"
	"github.com/anyproto/any-sync-coordinator/db"
	"github.com/anyproto/any-sync-coordinator/spacestatus"
	"github.com/anyproto/any-sync-filenode/index"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/nodeconf"
	"go.uber.org/zap"
)

//go:embed static/*
var staticFS embed.FS

const CName = "bundle.adminui"

var log = logger.NewNamed(CName)

// AdminUI provides web-based administrative interface for any-sync-bundle.
type AdminUI struct {
	config   Config
	server   *http.Server
	service  *Service
	handlers *Handlers

	// Direct app references
	coordinator *app.App
	consensus   *app.App
	filenode    *app.App
	sync        *app.App
}

// New creates AdminUI with direct app references.
func New(cfg Config, coordinator, consensus, filenode, sync *app.App) app.Component {
	return &AdminUI{
		config:      cfg,
		coordinator: coordinator,
		consensus:   consensus,
		filenode:    filenode,
		sync:        sync,
	}
}

func (a *AdminUI) Init(_ *app.App) error {
	if a.config.ListenAddr == "" {
		log.Info("admin UI disabled (no listen address)")
		return nil
	}

	// Initialize service with components from apps
	a.service = newService(
		app.MustComponent[db.Database](a.coordinator),
		app.MustComponent[accountlimit.AccountLimit](a.coordinator),
		app.MustComponent[spacestatus.SpaceStatus](a.coordinator),
		app.MustComponent[index.Index](a.filenode),
		app.MustComponent[acleventlog.AclEventLog](a.coordinator),
		app.MustComponent[nodeconf.NodeConf](a.coordinator),
	)

	// Initialize handlers and server
	a.handlers = newHandlers(a.service)
	a.setupServer()

	log.Info("admin UI initialized", zap.String("addr", a.config.ListenAddr))
	return nil
}

func (a *AdminUI) setupServer() {
	mux := http.NewServeMux()

	// Static files (CSS, JS, etc.)
	staticHandler := http.FileServer(http.FS(staticFS))
	mux.Handle("/static/", staticHandler)

	// Admin routes
	mux.HandleFunc("/admin/", a.handlers.handleIndex)
	mux.HandleFunc("/admin/users", a.handlers.handleAllUsers)
	mux.HandleFunc("/admin/spaces/all", a.handlers.handleAllSpacesList)
	mux.HandleFunc("/admin/user/{identity}", a.handlers.handleUserDetail)
	mux.HandleFunc("/admin/user/{identity}/quota", a.handlers.handleQuotaEdit)
	mux.HandleFunc("/admin/spaces", a.handlers.handleSpacesList)
	mux.HandleFunc("/admin/deletions", a.handlers.handleDeletions)
	mux.HandleFunc("/admin/health", a.handlers.handleHealth)
	mux.HandleFunc("/admin/acl-events", a.handlers.handleACLEvents)
	mux.HandleFunc("/admin/network", a.handlers.handleNetworkTopology)
	mux.HandleFunc("/admin/space/toggle-shareability", a.handlers.handleToggleShareability)

	a.server = &http.Server{
		Addr:              a.config.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

func (a *AdminUI) Name() string {
	return CName
}

func (a *AdminUI) Run(_ context.Context) error {
	if a.config.ListenAddr == "" {
		return nil
	}

	go func() {
		log.Info("starting admin UI server", zap.String("addr", a.config.ListenAddr))
		if serverErr := a.server.ListenAndServe(); serverErr != nil && serverErr != http.ErrServerClosed {
			log.Error("admin UI server error", zap.Error(serverErr))
		}
	}()

	return nil
}

func (a *AdminUI) Close(ctx context.Context) error {
	if a.server != nil {
		log.Info("shutting down admin UI server")
		return a.server.Shutdown(ctx)
	}
	return nil
}
