# 107 - Add agent token provider for AI Foundry bearer auth

## Problem Context

Herald's workflow nodes create agents via `agent.New(&rt.Agent)`, directly coupling the workflow package to `gaconfig.AgentConfig`. When managed identity is enabled (issue #108), agents need fresh Entra bearer tokens injected per-creation. This task replaces the direct config dependency with a `NewAgent` factory closure that encapsulates auth-mode-aware agent creation.

## Architecture Approach

`workflow.Runtime` receives a `NewAgent func(ctx context.Context) (agent.Agent, error)` factory closure built at the infrastructure layer during cold start. The factory resolves the auth strategy once — API key mode captures the config by value and delegates to `agent.New`, bearer mode (wired in #108) will acquire a cached token and inject it into a stack-local config copy. Workflow nodes call `rt.NewAgent(ctx)` without auth awareness.

`Model` and `Provider` string fields on `Runtime` store the config-derived names for classification DB records.

## Implementation

### Step 1: Update workflow.Runtime

**File:** `internal/workflow/runtime.go`

Replace the entire file contents with:

```go
package workflow

import (
	"context"
	"log/slog"

	"github.com/JaimeStill/go-agents/pkg/agent"

	"github.com/JaimeStill/herald/internal/documents"
	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/pkg/storage"
)

type Runtime struct {
	NewAgent func(ctx context.Context) (agent.Agent, error)
	Model    string
	Provider string
	Storage  storage.System
	Documents documents.System
	Prompts   prompts.System
	Logger    *slog.Logger
}
```

### Step 2: Update classify node

**File:** `internal/workflow/classify.go`

Replace the `agent` import with nothing (remove it), and update the agent creation call.

In the import block, remove:
```go
"github.com/JaimeStill/go-agents/pkg/agent"
```

In `classifyPages`, replace:
```go
a, err := agent.New(&rt.Agent)
if err != nil {
    return fmt.Errorf("page %d: create agent: %w", i+1, err)
}
```

With:
```go
a, err := rt.NewAgent(gctx)
if err != nil {
    return fmt.Errorf("page %d: create agent: %w", i+1, err)
}
```

### Step 3: Update enhance node

**File:** `internal/workflow/enhance.go`

In the import block, remove:
```go
"github.com/JaimeStill/go-agents/pkg/agent"
```

In `enhancePages`, replace:
```go
a, err := agent.New(&rt.Agent)
if err != nil {
    return fmt.Errorf("page %d: create agent: %w", cs.Pages[i].PageNumber, err)
}
```

With:
```go
a, err := rt.NewAgent(gctx)
if err != nil {
    return fmt.Errorf("page %d: create agent: %w", cs.Pages[i].PageNumber, err)
}
```

### Step 4: Update finalize node

**File:** `internal/workflow/finalize.go`

In the import block, remove:
```go
"github.com/JaimeStill/go-agents/pkg/agent"
```

In `synthesize`, replace:
```go
a, err := agent.New(&rt.Agent)
if err != nil {
    return fmt.Errorf("%w: create agent: %w", ErrFinalizeFailed, err)
}
```

With:
```go
a, err := rt.NewAgent(ctx)
if err != nil {
    return fmt.Errorf("%w: create agent: %w", ErrFinalizeFailed, err)
}
```

### Step 5: Update classification repository

**File:** `internal/classifications/repository.go`

Update the `New` function signature — replace the `agent gaconfig.AgentConfig` parameter with three parameters and update the `workflow.Runtime` construction:

Replace:
```go
func New(
	db *sql.DB,
	agent gaconfig.AgentConfig,
	logger *slog.Logger,
	pagination pagination.Config,
	storage storage.System,
	docs documents.System,
	prompts prompts.System,
) System {
	rt := &workflow.Runtime{
		Agent:     agent,
		Storage:   storage,
		Documents: docs,
		Prompts:   prompts,
		Logger:    logger.With("workflow", "classify"),
	}
```

With:
```go
func New(
	db *sql.DB,
	newAgent func(ctx context.Context) (agent.Agent, error),
	modelName string,
	providerName string,
	logger *slog.Logger,
	pagination pagination.Config,
	storage storage.System,
	docs documents.System,
	prompts prompts.System,
) System {
	rt := &workflow.Runtime{
		NewAgent:     newAgent,
		Model:    modelName,
		Provider: providerName,
		Storage:      storage,
		Documents:    docs,
		Prompts:      prompts,
		Logger:       logger.With("workflow", "classify"),
	}
```

Update the imports — replace:
```go
gaconfig "github.com/JaimeStill/go-agents/pkg/config"
```

With:
```go
"github.com/JaimeStill/go-agents/pkg/agent"
```

Update the upsert args in the `Classify` method. Replace:
```go
r.rt.Agent.Model.Name,
r.rt.Agent.Provider.Name,
```

With:
```go
r.rt.Model,
r.rt.Provider,
```

### Step 6: Add NewAgent factory to Infrastructure

**File:** `internal/infrastructure/infrastructure.go`

Add `NewAgent` field to the struct:

```go
type Infrastructure struct {
	Lifecycle  *lifecycle.Coordinator
	Logger     *slog.Logger
	Database   database.System
	Storage    storage.System
	Agent      gaconfig.AgentConfig
	Credential azcore.TokenCredential
	NewAgent   func(ctx context.Context) (agent.Agent, error)
}
```

Add `"context"` to the import block.

In the `New` function, after the agent validation block and before the return statement, build the factory:

```go
agentCfg := cfg.Agent
newAgent := func(ctx context.Context) (agent.Agent, error) {
	return agent.New(&agentCfg)
}
```

Add `NewAgent: newAgent,` to the returned `Infrastructure` struct literal.

### Step 7: Thread NewAgent through API runtime

**File:** `internal/api/runtime.go`

Add `NewAgent` to the inner `Infrastructure` struct in `NewRuntime`:

```go
return &Runtime{
    Infrastructure: &infrastructure.Infrastructure{
        Agent:      cfg.Agent,
        Credential: infra.Credential,
        Lifecycle:  infra.Lifecycle,
        Logger:     infra.Logger.With("module", "api"),
        Database:   infra.Database,
        Storage:    infra.Storage,
        NewAgent:   infra.NewAgent,
    },
    Pagination: cfg.API.Pagination,
}
```

### Step 8: Update domain wiring

**File:** `internal/api/domain.go`

Update the `classifications.New()` call to pass the factory and metadata strings instead of the raw config:

Replace:
```go
classificationsSystem := classifications.New(
    runtime.Database.Connection(),
    runtime.Agent,
    runtime.Logger,
    runtime.Pagination,
    runtime.Storage,
    docsSystem,
    promptsSystem,
)
```

With:
```go
classificationsSystem := classifications.New(
    runtime.Database.Connection(),
    runtime.NewAgent,
    runtime.Agent.Model.Name,
    runtime.Agent.Provider.Name,
    runtime.Logger,
    runtime.Pagination,
    runtime.Storage,
    docsSystem,
    promptsSystem,
)
```

## Validation Criteria

- [ ] `go vet ./...` passes
- [ ] `go build ./cmd/server/` succeeds
- [ ] No direct `agent.New` calls remain in `internal/workflow/`
- [ ] `workflow.Runtime` has no `gaconfig.AgentConfig` dependency
- [ ] `internal/api/domain.go` does not import `workflow`
