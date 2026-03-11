// Package infrastructure provides core service initialization for application startup.
// It assembles common dependencies (logging, database, storage) that domain systems require.
package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"

	"github.com/JaimeStill/go-agents/pkg/agent"
	"github.com/JaimeStill/herald/internal/config"
	"github.com/JaimeStill/herald/pkg/auth"
	"github.com/JaimeStill/herald/pkg/database"
	"github.com/JaimeStill/herald/pkg/lifecycle"
	"github.com/JaimeStill/herald/pkg/storage"

	gaconfig "github.com/JaimeStill/go-agents/pkg/config"
)

// Infrastructure holds the core systems required by all domain modules.
// It provides a single point of initialization for lifecycle coordination,
// logging, database access, file storage, and agent configuration.
type Infrastructure struct {
	Lifecycle  *lifecycle.Coordinator
	Logger     *slog.Logger
	Database   database.System
	Storage    storage.System
	Agent      gaconfig.AgentConfig
	Credential azcore.TokenCredential
	NewAgent   func(ctx context.Context) (agent.Agent, error)
}

// New creates an Infrastructure from the application configuration.
// It initializes all systems but does not start them; call Start separately.
// Agent configuration is validated by creating a test agent via agent.New,
// which verifies the full provider pipeline (registration, option extraction).
func New(cfg *config.Config) (*Infrastructure, error) {
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

	if _, err := agent.New(&cfg.Agent); err != nil {
		return nil, fmt.Errorf("agent validation failed: %w", err)
	}

	newAgent := newAgentFactory(cfg.Agent, cred, cfg.Auth.ManagedIdentity)

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

func newAgentFactory(
	agentCfg gaconfig.AgentConfig,
	cred azcore.TokenCredential,
	managedIdentity bool,
) func(ctx context.Context) (agent.Agent, error) {
	if cred != nil && managedIdentity {
		return func(ctx context.Context) (agent.Agent, error) {
			tok, err := cred.GetToken(ctx, policy.TokenRequestOptions{
				Scopes: []string{auth.AgentScope},
			})
			if err != nil {
				return nil, fmt.Errorf("acquire agent token: %w", err)
			}

			pc := agentCfg.Provider
			opts := maps.Clone(pc.Options)
			opts["token"] = tok.Token
			opts["auth_type"] = "bearer"
			pc.Options = opts

			cloned := agentCfg
			cloned.Provider = pc
			return agent.New(&cloned)
		}
	}

	return func(ctx context.Context) (agent.Agent, error) {
		return agent.New(&agentCfg)
	}
}
