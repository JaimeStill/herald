# 108 - Add AuthConfig managed_identity flag and infrastructure wiring

## Problem Context

Herald's three Azure service clients (Storage, Database, AI Foundry) each have credential-based constructors (#105, #106, #107), but nothing in the config or infrastructure layer decides when to use them. This issue adds the `managed_identity` boolean to `AuthConfig` and branches `infrastructure.New()` accordingly.

## Architecture Approach

Single decision point in `infrastructure.New()`. When `cred != nil && cfg.Auth.ManagedIdentity`, use credential-based constructors for all three services. No changes to downstream code — workflow, classifications, API module all receive the same interfaces regardless of auth path.

The agent factory closure pattern from #107 makes this clean: infrastructure builds the appropriate closure at cold start, and workflow nodes call `rt.NewAgent(ctx)` without knowing which auth path is active.

## Implementation

### Step 1: Add `ManagedIdentity` field and `AgentScope` constant to AuthConfig

**File:** `internal/config/auth.go`

Add to the env var const block:

```go
EnvAuthManagedIdentity = "HERALD_AUTH_MANAGED_IDENTITY"
```

Add a new const block for OAuth scopes:

```go
const AgentScope = "https://cognitiveservices.azure.com/.default"
```

Add field to `AuthConfig`:

```go
type AuthConfig struct {
	Mode            AuthMode `json:"auth_mode"`
	TenantID        string   `json:"tenant_id"`
	ClientID        string   `json:"client_id"`
	ClientSecret    string   `json:"client_secret"`
	ManagedIdentity bool     `json:"managed_identity"`
}
```

Add to `Merge()` (after the `ClientSecret` block):

```go
if overlay.ManagedIdentity {
	c.ManagedIdentity = true
}
```

Add to `loadEnv()` (after the `ClientSecret` block):

```go
if v := os.Getenv(EnvAuthManagedIdentity); v == "true" || v == "1" {
	c.ManagedIdentity = true
}
```

### Step 2: Extract private helpers in infrastructure

**File:** `internal/infrastructure/infrastructure.go`

Add import:

```go
"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
```

Add three private functions that encapsulate each concern:

**`initSystems`** — branches database and storage constructor selection:

```go
func initSystems(
	cfg *config.Config,
	cred azcore.TokenCredential,
	logger *slog.Logger,
) (database.System, storage.System, error) {
	if cred != nil && cfg.Auth.ManagedIdentity {
		return initCredentialSystems(cfg, cred, logger)
	}
	return initConnectionStringSystems(cfg, logger)
}

func initCredentialSystems(
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

func initConnectionStringSystems(
	cfg *config.Config,
	logger *slog.Logger,
) (database.System, storage.System, error) {
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
```

**`newAgentFactory`** — builds the auth-mode-aware agent factory closure:

```go
func newAgentFactory(
	agentCfg gaconfig.AgentConfig,
	cred azcore.TokenCredential,
	managedIdentity bool,
) func(ctx context.Context) (agent.Agent, error) {
	if cred != nil && managedIdentity {
		return func(ctx context.Context) (agent.Agent, error) {
			tok, err := cred.GetToken(ctx, policy.TokenRequestOptions{
				Scopes: []string{config.AgentScope},
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
```

### Step 3: Simplify `New()` to use the helpers

**File:** `internal/infrastructure/infrastructure.go`

Full `New()` function using the extracted helpers:

```go
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
```

## Validation Criteria

- [ ] `go vet ./...` passes
- [ ] `go build ./cmd/server/` passes
- [ ] `go test ./tests/...` passes
- [ ] `AuthConfig` has `ManagedIdentity` field with env var override and Merge support
- [ ] `infrastructure.New()` branches constructor selection based on `ManagedIdentity`
- [ ] Token provider closure built and stored when managed identity is active
- [ ] `auth_mode: "none"` behavior completely unchanged
- [ ] `auth_mode: "azure"` + `managed_identity: false` uses connection strings
