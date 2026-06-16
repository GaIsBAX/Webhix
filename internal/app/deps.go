package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/GaIsBAX/Webhix/internal/config"
	"github.com/GaIsBAX/Webhix/internal/core"
	"github.com/GaIsBAX/Webhix/internal/hub"
	"github.com/GaIsBAX/Webhix/internal/repos"
	"github.com/GaIsBAX/Webhix/internal/server"
	"github.com/GaIsBAX/Webhix/internal/store"
	"github.com/GaIsBAX/Webhix/internal/web"
	"github.com/GaIsBAX/Webhix/pkg"
)

type dependencies struct {
	mux *http.ServeMux
	cfg *config.Config

	infra    *infrastructure
	repos    *repositories
	services *services
	handlers *handlers
}

func newDependencies(ctx context.Context, cfg *config.Config) (*dependencies, error) {
	var deps dependencies

	mux := http.NewServeMux()

	infra, err := newInfrastructure(ctx, cfg)
	if err != nil {
		return nil, err
	}

	repos := newRepositories(infra.db)
	services := newServices(repos)

	deps.mux = mux
	deps.cfg = cfg

	deps.infra = infra
	deps.repos = repos
	deps.services = services
	deps.handlers = newHandlers(&deps)
	deps.handlers.registerRoutes()

	staticFS, err := fs.Sub(web.Static, "static")
	if err != nil {
		return nil, err
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	return &deps, nil
}

type services struct {
	hook  *core.Hook
	serve *core.Serve
}

func newServices(repos *repositories) *services {
	hook := core.NewHook(repos.hook, func() string {
		return pkg.GeneratePrefixedString("ho")
	})
	serve := core.NewServe(repos.serve)

	return &services{
		hook:  hook,
		serve: serve,
	}
}

type repositories struct {
	hook  *repos.Hook
	serve *repos.Serve
}

func newRepositories(db *store.Database) *repositories {
	return &repositories{
		hook:  repos.NewHook(db.DB),
		serve: repos.NewServe(db.DB),
	}
}

func (d *dependencies) close() error {
	if d.infra.db != nil {
		return d.infra.db.Close()
	}

	return nil
}

type infrastructure struct {
	db  *store.Database
	hub *hub.Hub
}

func newInfrastructure(ctx context.Context, cfg *config.Config) (*infrastructure, error) {
	db, err := store.New(ctx, cfg.DBPath)
	if err != nil {
		return nil, err
	}

	if err := db.Migrate(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, errors.Join(
				fmt.Errorf("%w: %w", ErrMigrateDatabase, err),
				fmt.Errorf("%w after migration failure: %w", ErrCloseDatabase, closeErr),
			)
		}

		return nil, fmt.Errorf("%w: %w", ErrMigrateDatabase, err)
	}

	hub := hub.New()

	return &infrastructure{
		db:  db,
		hub: hub,
	}, nil
}

type handlers struct {
	hook *server.Hook
}

func newHandlers(deps *dependencies) *handlers {
	return &handlers{
		hook: server.NewHook(&server.HookDeps{
			Mux:           deps.mux,
			Service:       deps.services.hook,
			Notifications: deps.services.hook,
			Hub:           deps.infra.hub,
			Opts: server.HookOptions{
				BaseURL:     deps.cfg.BaseURL,
				MaxBodySize: deps.cfg.MaxBodySize,
				ReadOnly:    deps.cfg.ReadOnly,
			},
		}),
	}
}

func (h *handlers) registerRoutes() {
	h.hook.RegisterRoutes()
}
