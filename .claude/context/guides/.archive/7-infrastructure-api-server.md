# 7 - Infrastructure Assembly, API Module, and Server Entry Point

## Problem Context

Issues #4, #5, and #6 established Herald's foundational packages — lifecycle coordination, database toolkit, storage abstraction, middleware, module/routing, handlers, and configuration. These are all disconnected pieces. This issue wires them together into the Infrastructure assembly, API module shell, and server entry point so Herald becomes a running Go web service with health/readiness probes and graceful shutdown.

## Architecture Approach

All patterns adapted from agent-lab's Layered Composition Architecture:

- **Infrastructure** assembles shared subsystems (lifecycle, logger, database, storage) in a single struct. `New()` is cold start (creates subsystems, no connections). `Start()` is hot start (registers lifecycle hooks that establish connections).
- **API Module** follows three-step initialization: Runtime (infrastructure + API config) → Domain (empty placeholder) → Module (mux + middleware). Returns a `*module.Module` for the router.
- **Server** coordinates the full lifecycle: config load → infrastructure create → modules create → router build → HTTP server create (cold start) → infrastructure start → HTTP start → wait for readiness → block on signal → shutdown.

## Implementation

### Step 1: Add Pagination Config to APIConfig

**`internal/config/api.go`** — add pagination env var mapping, field, and wiring:

```go
// Add import
"github.com/JaimeStill/herald/pkg/pagination"

// Add env var mapping (alongside corsEnv and openAPIEnv)
var paginationEnv = &pagination.ConfigEnv{
	DefaultPageSize: "HERALD_PAGINATION_DEFAULT_PAGE_SIZE",
	MaxPageSize:     "HERALD_PAGINATION_MAX_PAGE_SIZE",
}

// Add Pagination field to APIConfig
type APIConfig struct {
	BasePath   string                `toml:"base_path"`
	CORS       middleware.CORSConfig `toml:"cors"`
	OpenAPI    openapi.Config        `toml:"openapi"`
	Pagination pagination.Config     `toml:"pagination"`
}
```

Update `Finalize` — add after the OpenAPI finalize call:

```go
if err := c.Pagination.Finalize(paginationEnv); err != nil {
	return fmt.Errorf("pagination: %w", err)
}
```

Update `Merge` — add at the end:

```go
c.Pagination.Merge(&overlay.Pagination)
```

No changes needed to `loadDefaults` or `loadEnv` — pagination defaults are handled by `pagination.Config.Finalize`.

**`config.toml`** — add at the end of the `[api]` section (before any non-api sections):

```toml
[api.pagination]
default_page_size = 20
max_page_size = 100
```

### Step 2: Infrastructure Assembly

**Delete** `internal/infrastructure/doc.go`

**Create `internal/infrastructure/infrastructure.go`:**

```go
package infrastructure

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/pkg/database"
	"github.com/JaimeStill/herald/pkg/lifecycle"
	"github.com/JaimeStill/herald/pkg/storage"
)

type Infrastructure struct {
	Lifecycle *lifecycle.Coordinator
	Logger    *slog.Logger
	Database  database.System
	Storage   storage.System
}

func New(cfg *config.Config) (*Infrastructure, error) {
	lc := lifecycle.New()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	db, err := database.New(&cfg.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("database init failed: %w", err)
	}

	store, err := storage.New(&cfg.Storage, logger)
	if err != nil {
		return nil, fmt.Errorf("storage init failed: %w", err)
	}

	return &Infrastructure{
		Lifecycle: lc,
		Logger:    logger,
		Database:  db,
		Storage:   store,
	}, nil
}

func (i *Infrastructure) Start() error {
	if err := i.Database.Start(i.Lifecycle); err != nil {
		return fmt.Errorf("database start failed: %w", err)
	}
	if err := i.Storage.Start(i.Lifecycle); err != nil {
		return fmt.Errorf("storage start failed: %w", err)
	}
	return nil
}
```

### Step 3: API Module

**Delete** `internal/api/doc.go`

**Create `internal/api/runtime.go`:**

```go
package api

import (
	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/internal/infrastructure"
	"github.com/JaimeStill/herald/pkg/pagination"
)

type Runtime struct {
	*infrastructure.Infrastructure
	Pagination pagination.Config
}

func NewRuntime(cfg *config.Config, infra *infrastructure.Infrastructure) *Runtime {
	return &Runtime{
		Infrastructure: &infrastructure.Infrastructure{
			Lifecycle: infra.Lifecycle,
			Logger:    infra.Logger.With("module", "api"),
			Database:  infra.Database,
			Storage:   infra.Storage,
		},
		Pagination: cfg.API.Pagination,
	}
}
```

**Create `internal/api/domain.go`:**

```go
package api

type Domain struct{}

func NewDomain(runtime *Runtime) *Domain {
	return &Domain{}
}
```

**Create `internal/api/routes.go`:**

```go
package api

import (
	"net/http"

	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/pkg/openapi"
)

func registerRoutes(
	mux *http.ServeMux,
	spec *openapi.Spec,
	domain *Domain,
	cfg *config.Config,
) {
	// Domain route groups registered here as systems are implemented.
}
```

**Create `internal/api/api.go`:**

```go
package api

import (
	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/internal/infrastructure"
	"github.com/JaimeStill/herald/pkg/middleware"
	"github.com/JaimeStill/herald/pkg/module"
	"github.com/JaimeStill/herald/pkg/openapi"
	"net/http"
)

func NewModule(cfg *config.Config, infra *infrastructure.Infrastructure) (*module.Module, error) {
	runtime := NewRuntime(cfg, infra)
	domain := NewDomain(runtime)

	spec := openapi.NewSpec(cfg.API.OpenAPI.Title, cfg.Version)
	spec.SetDescription(cfg.API.OpenAPI.Description)

	mux := http.NewServeMux()
	registerRoutes(mux, spec, domain, cfg)

	specBytes, err := openapi.MarshalJSON(spec)
	if err != nil {
		return nil, err
	}
	mux.HandleFunc("GET /openapi.json", openapi.ServeSpec(specBytes))

	m := module.New(cfg.API.BasePath, mux)
	m.Use(middleware.CORS(&cfg.API.CORS))
	m.Use(middleware.Logger(runtime.Infrastructure.Logger))

	return m, nil
}
```

### Step 4: Server Entry Point

**Create `cmd/server/http.go`:**

```go
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/pkg/lifecycle"
)

type httpServer struct {
	http            *http.Server
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

func newHTTPServer(cfg *config.ServerConfig, handler http.Handler, logger *slog.Logger) *httpServer {
	return &httpServer{
		http: &http.Server{
			Addr:         cfg.Addr(),
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeoutDuration(),
			WriteTimeout: cfg.WriteTimeoutDuration(),
		},
		logger:          logger.With("system", "http"),
		shutdownTimeout: cfg.ShutdownTimeoutDuration(),
	}
}

func (s *httpServer) Start(lc *lifecycle.Coordinator) error {
	go func() {
		s.logger.Info("server listening", "addr", s.http.Addr)
		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("server error", "error", err)
		}
	}()

	lc.OnShutdown(func() {
		<-lc.Context().Done()
		s.logger.Info("shutting down server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		if err := s.http.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("server shutdown error", "error", err)
		} else {
			s.logger.Info("server shutdown complete")
		}
	})

	return nil
}
```

**Create `cmd/server/modules.go`:**

```go
package main

import (
	"encoding/json"
	"net/http"

	"github.com/JaimeStill/herald/internal/api"
	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/internal/infrastructure"
	"github.com/JaimeStill/herald/pkg/middleware"
	"github.com/JaimeStill/herald/pkg/module"
	"github.com/JaimeStill/herald/web/scalar"
)

type Modules struct {
	API    *module.Module
	Scalar *module.Module
}

func NewModules(infra *infrastructure.Infrastructure, cfg *config.Config) (*Modules, error) {
	apiModule, err := api.NewModule(cfg, infra)
	if err != nil {
		return nil, err
	}

	scalarModule := scalar.NewModule("/scalar")
	scalarModule.Use(middleware.Logger(infra.Logger))

	return &Modules{
		API:    apiModule,
		Scalar: scalarModule,
	}, nil
}

func (m *Modules) Mount(router *module.Router) {
	router.Mount(m.API)
	router.Mount(m.Scalar)
}

func buildRouter(infra *infrastructure.Infrastructure) *module.Router {
	router := module.NewRouter()

	router.HandleNative("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	router.HandleNative("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !infra.Lifecycle.Ready() {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})

	return router
}
```

**Create `cmd/server/server.go`:**

```go
package main

import (
	"time"

	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/internal/infrastructure"
)

type Server struct {
	infra   *infrastructure.Infrastructure
	modules *Modules
	http    *httpServer
}

func NewServer(cfg *config.Config) (*Server, error) {
	infra, err := infrastructure.New(cfg)
	if err != nil {
		return nil, err
	}

	modules, err := NewModules(infra, cfg)
	if err != nil {
		return nil, err
	}

	router := buildRouter(infra)
	modules.Mount(router)

	infra.Logger.Info(
		"server initialized",
		"addr", cfg.Server.Addr(),
		"version", cfg.Version,
	)

	return &Server{
		infra:   infra,
		modules: modules,
		http:    newHTTPServer(&cfg.Server, router, infra.Logger),
	}, nil
}

func (s *Server) Start() error {
	s.infra.Logger.Info("starting service")

	if err := s.infra.Start(); err != nil {
		return err
	}

	if err := s.http.Start(s.infra.Lifecycle); err != nil {
		return err
	}

	go func() {
		s.infra.Lifecycle.WaitForStartup()
		s.infra.Logger.Info("all subsystems ready")
	}()

	return nil
}

func (s *Server) Shutdown(timeout time.Duration) error {
	s.infra.Logger.Info("initiating shutdown")
	return s.infra.Lifecycle.Shutdown(timeout)
}
```

**Replace `cmd/server/main.go`:**

```go
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/JaimeStill/herald/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config load failed:", err)
	}

	srv, err := NewServer(cfg)
	if err != nil {
		log.Fatal("service init failed:", err)
	}

	if err := srv.Start(); err != nil {
		log.Fatal("service start failed:", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	if err := srv.Shutdown(cfg.ShutdownTimeoutDuration()); err != nil {
		log.Fatal("shutdown failed:", err)
	}

	log.Println("service stopped gracefully")
}
```

## Validation Criteria

- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] `go mod tidy` produces no changes
- [ ] `docker compose up -d` starts PostgreSQL and Azurite
- [ ] `mise run dev` starts the server successfully
- [ ] Server connects to PostgreSQL on startup (log: "database connection established")
- [ ] Server initializes storage container on startup (log: "storage container ready")
- [ ] `GET /healthz` returns 200 with `{"status":"ok"}`
- [ ] `GET /readyz` returns 200 with `{"status":"ready"}` after startup
- [ ] `GET /api/openapi.json` returns the OpenAPI spec
- [ ] `GET /scalar` serves the Scalar API reference UI
- [ ] SIGINT triggers graceful shutdown with ordered teardown logs
