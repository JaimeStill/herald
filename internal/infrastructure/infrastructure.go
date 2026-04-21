// Package infrastructure provides core service initialization for application startup.
// It assembles common dependencies (logging, database, storage) that domain systems require.
package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/pkg/database"
	"github.com/JaimeStill/herald/pkg/lifecycle"
	"github.com/JaimeStill/herald/pkg/storage"
	"github.com/tailored-agentic-units/agent"
	"github.com/tailored-agentic-units/format"
	"github.com/tailored-agentic-units/provider"

	tauopenai "github.com/tailored-agentic-units/format/openai"
	tauconfig "github.com/tailored-agentic-units/protocol/config"
	tauazure "github.com/tailored-agentic-units/provider/azure"
	tauollama "github.com/tailored-agentic-units/provider/ollama"
)

var registerOnce sync.Once

// registerAgentBackends wires the tau provider and format factories into their
// global registries. Guarded by sync.Once so repeated Infrastructure.New calls
// (notably from tests) are safe and idempotent.
func registerAgentBackends() {
	registerOnce.Do(func() {
		tauazure.Register()
		tauollama.Register()
		tauopenai.Register()
	})
}

// Infrastructure holds the core systems required by all domain modules.
// It provides a single point of initialization for lifecycle coordination,
// logging, database access, file storage, and agent configuration.
type Infrastructure struct {
	Lifecycle  *lifecycle.Coordinator
	Logger     *slog.Logger
	Database   database.System
	Storage    storage.System
	Agent      tauconfig.AgentConfig
	Credential azcore.TokenCredential
	NewAgent   func(ctx context.Context) (agent.Agent, error)
}

// New creates an Infrastructure from the application configuration.
// It initializes all systems but does not start them; call Start separately.
// Agent configuration is validated by constructing a single agent via
// provider.Create + format.Create + agent.New, which exercises the full
// tau pipeline (factory lookup, option extraction, credential wiring) before
// any request is served.
func New(cfg *config.Config) (*Infrastructure, error) {
	registerAgentBackends()

	lc := lifecycle.New()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cred, err := cfg.Auth.TokenCredential()
	if err != nil {
		return nil, fmt.Errorf("credential init failed: %w", err)
	}

	db, store, err := initSystems(cfg, cred, logger)
	if err != nil {
		return nil, err
	}

	agentCfg := cfg.Agent
	newAgent := func(ctx context.Context) (agent.Agent, error) {
		p, perr := provider.Create(agentCfg.Provider)
		if perr != nil {
			return nil, fmt.Errorf("create provider: %w", perr)
		}
		f, ferr := format.Create(agentCfg.Format)
		if ferr != nil {
			return nil, fmt.Errorf("create format: %w", ferr)
		}
		return agent.New(&agentCfg, p, f), nil
	}

	if _, err := newAgent(context.Background()); err != nil {
		return nil, fmt.Errorf("agent validation failed: %w", err)
	}

	return &Infrastructure{
		Lifecycle:  lc,
		Logger:     logger,
		Database:   db,
		Storage:    store,
		Agent:      cfg.Agent,
		Credential: cred,
		NewAgent:   newAgent,
	}, nil
}

// Start registers all infrastructure systems with the lifecycle coordinator.
// Database and storage hooks are registered for startup and shutdown coordination.
func (i *Infrastructure) Start() error {
	if err := i.Database.Start(i.Lifecycle); err != nil {
		return fmt.Errorf("database start failed: %w", err)
	}
	if err := i.Storage.Start(i.Lifecycle); err != nil {
		return fmt.Errorf("storage start failed: %w", err)
	}
	return nil
}

func initSystems(
	cfg *config.Config,
	cred azcore.TokenCredential,
	logger *slog.Logger,
) (database.System, storage.System, error) {
	if cred != nil && cfg.Auth.ManagedIdentity {
		return initManagedSystems(cfg, cred, logger)
	}

	db, err := database.New(&cfg.Database, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("database init failed: %w", err)
	}

	store, err := storage.New(&cfg.Storage, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("storage init failed: %w", err)
	}

	return db, store, nil
}

func initManagedSystems(
	cfg *config.Config,
	cred azcore.TokenCredential,
	logger *slog.Logger,
) (database.System, storage.System, error) {
	db, err := database.NewWithCredential(&cfg.Database, cred, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("database credential init failed: %w", err)
	}

	store, err := storage.NewWithCredential(&cfg.Storage, cred, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("storage credential init failed: %w", err)
	}

	return db, store, nil
}
