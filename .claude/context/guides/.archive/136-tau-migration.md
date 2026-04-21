# 136 — Migrate to tau/agent and tau/orchestrate

## Problem Context

Herald currently depends on `github.com/JaimeStill/go-agents@v0.4.0` and `github.com/JaimeStill/go-agents-orchestration@v0.3.3`. Those libraries have been forked into the `tailored-agentic-units` GitHub org as independent modules. The migration replaces both dependencies with the new tau modules so the Phase 5 release can cut ties with the personal fork.

Exploration showed the fork is **not** a trivial import-path rename. Tau restructured the package layout, split one module into seven, introduced a required registration step for providers/formats, and changed the Vision/Chat method signatures to use typed messages and images. The steps below capture every code change required to complete the migration — preserving Herald's behavior end-to-end.

## Architecture Approach

Replace the module graph wholesale in a single branch. Keep the existing agent factory, per-request concurrency pattern, and observer translation semantics. Delegate base64 image encoding to tau's format package. Centralize provider/format registration in `infrastructure.New` behind a `sync.Once` so the production binary and all unit tests get consistent wiring without extra setup.

No functional changes: prompt wording, workflow topology, and the conditional `enhance` edge stay exactly as they are.

## Implementation

### Step 1 — Update module dependencies

Remove the JaimeStill modules and add the eight tau modules. Run from the repo root:

```bash
go get github.com/tailored-agentic-units/agent@v0.1.1
go get github.com/tailored-agentic-units/orchestrate@v0.1.0
go get github.com/tailored-agentic-units/protocol@v0.1.0
go get github.com/tailored-agentic-units/format@v0.1.0
go get github.com/tailored-agentic-units/format/openai@v0.1.0
go get github.com/tailored-agentic-units/provider@v0.1.0
go get github.com/tailored-agentic-units/provider/azure@v0.1.0
go get github.com/tailored-agentic-units/provider/ollama@v0.1.0
```

After the source changes below are in, run:

```bash
go mod tidy
```

to drop the JaimeStill entries and reconcile the indirect set. Do not add any `replace` directives. Verify `go.mod` has no `JaimeStill/go-agents` lines.

### Step 2 — `internal/config/config.go`

Swap the gaconfig import path:

```go
// before
gaconfig "github.com/JaimeStill/go-agents/pkg/config"

// after
gaconfig "github.com/tailored-agentic-units/protocol/config"
```

No other edits in this file — `gaconfig.AgentConfig` and its `Merge` method exist at the new path with compatible shape.

### Step 3 — `internal/config/agent.go`

Same import swap:

```go
gaconfig "github.com/tailored-agentic-units/protocol/config"
```

The rest of `FinalizeAgent`, `loadAgentDefaults`, `loadAgentEnv`, and `validateAgent` stays untouched — `gaconfig.DefaultAgentConfig()`, `gaconfig.ProviderConfig{}`, and `gaconfig.ModelConfig{}` all exist with matching signatures.

### Step 4 — `internal/infrastructure/infrastructure.go`

Three coordinated changes in this file.

**4a. Update imports** (replace the go-agents imports, add tau registration packages and `sync`):

```go
import (
    "context"
    "fmt"
    "log/slog"
    "os"
    "sync"

    "github.com/Azure/azure-sdk-for-go/sdk/azcore"

    "github.com/tailored-agentic-units/agent"
    "github.com/tailored-agentic-units/format"
    formatopenai "github.com/tailored-agentic-units/format/openai"
    "github.com/tailored-agentic-units/provider"
    provazure "github.com/tailored-agentic-units/provider/azure"
    provollama "github.com/tailored-agentic-units/provider/ollama"

    "github.com/JaimeStill/herald/internal/config"
    "github.com/JaimeStill/herald/pkg/database"
    "github.com/JaimeStill/herald/pkg/lifecycle"
    "github.com/JaimeStill/herald/pkg/storage"

    gaconfig "github.com/tailored-agentic-units/protocol/config"
)
```

**4b. Add a `sync.Once` registrar** at package scope (above the `Infrastructure` struct):

```go
var registerOnce sync.Once

func registerAgentBackends() {
    registerOnce.Do(func() {
        provazure.Register()
        provollama.Register()
        formatopenai.Register()
    })
}
```

**4c. Rewrite `New`** to call the registrar and swap the one-shot `agent.New(&cfg.Agent)` validation for a three-step factory. The updated `New` body:

```go
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
```

Notes:
- `Infrastructure.Agent` and `Infrastructure.NewAgent` field declarations stay the same — they pick up the tau types automatically via the import swap.
- Keep `Start`, `initSystems`, and `initManagedSystems` as-is. They don't touch agents.

### Step 5 — `internal/workflow/runtime.go`

Import swap only:

```go
// before
"github.com/JaimeStill/go-agents/pkg/agent"

// after
"github.com/tailored-agentic-units/agent"
```

`Runtime.NewAgent func(ctx context.Context) (agent.Agent, error)` stays untouched.

### Step 6 — `internal/workflow/workflow.go`

Update the two orchestrate imports:

```go
gaoconfig "github.com/tailored-agentic-units/orchestrate/config"
"github.com/tailored-agentic-units/orchestrate/state"
```

No body changes — `gaoconfig.DefaultGraphConfig`, `state.New(nil)`, `state.NewGraphWithDeps`, `state.Not`, `state.NewFunctionNode`, and the `state.State` Get/Set methods all exist at the new paths with compatible signatures.

### Step 7 — `internal/workflow/init.go`

Orchestrate import swap only:

```go
// before
"github.com/JaimeStill/go-agents-orchestration/pkg/state"

// after
"github.com/tailored-agentic-units/orchestrate/state"
```

The `document-context` imports (`pkg/config`, `pkg/document`, `pkg/image`) stay — this is where PDF→PNG rendering lives, and `document-context` is unaffected by the tau migration.

### Step 8 — `internal/workflow/classify.go`

**7a. Update imports:**

```go
import (
    "context"
    "fmt"
    "os"

    "golang.org/x/sync/errgroup"

    "github.com/tailored-agentic-units/format"
    "github.com/tailored-agentic-units/orchestrate/state"
    "github.com/tailored-agentic-units/protocol"

    "github.com/JaimeStill/herald/internal/prompts"
    "github.com/JaimeStill/herald/pkg/formatting"
)
```

The `document-context/pkg/document` and `document-context/pkg/encoding` imports go away (tau's format package handles base64 encoding).

**7b. Replace `encodePageImage` with a simpler reader.** Delete the old function and add:

```go
func readPageImage(imagePath string) ([]byte, error) {
    data, err := os.ReadFile(imagePath)
    if err != nil {
        return nil, fmt.Errorf("read image: %w", err)
    }
    return data, nil
}
```

**7c. Rewrite the goroutine body inside `classifyPages`.** Replace the section from `a, err := rt.NewAgent(gctx)` through `applyPageResponse(&cs.Pages[i], parsed)`:

```go
a, err := rt.NewAgent(gctx)
if err != nil {
    return fmt.Errorf("page %d: create agent: %w", i+1, err)
}

imgData, err := readPageImage(cs.Pages[i].ImagePath)
if err != nil {
    return fmt.Errorf("page %d: %w", i+1, err)
}

resp, err := a.Vision(gctx,
    []protocol.Message{protocol.UserMessage(prompt)},
    []format.Image{{Data: imgData, Format: "png"}},
)
if err != nil {
    return fmt.Errorf("page %d: vision call: %w", i+1, err)
}

parsed, err := formatting.Parse[pageResponse](resp.Text())
if err != nil {
    return fmt.Errorf("page %d: parse response: %w", i+1, err)
}

applyPageResponse(&cs.Pages[i], parsed)
return nil
```

No other function in this file changes.

### Step 9 — `internal/workflow/enhance.go`

**8a. Update imports:**

```go
import (
    "context"
    "fmt"
    "os"
    "path/filepath"

    "github.com/JaimeStill/document-context/pkg/config"
    "github.com/JaimeStill/document-context/pkg/document"
    "github.com/JaimeStill/document-context/pkg/image"
    "golang.org/x/sync/errgroup"

    "github.com/tailored-agentic-units/format"
    "github.com/tailored-agentic-units/orchestrate/state"
    "github.com/tailored-agentic-units/protocol"

    "github.com/JaimeStill/herald/internal/prompts"
    "github.com/JaimeStill/herald/pkg/formatting"
)
```

`document-context/pkg/document` and `pkg/image` stay — `rerender` still needs them for ImageMagick. Only the encoding import goes away.

**8b. Rewrite the goroutine body inside `enhancePages`.** Replace the section from `dataURI, err := encodePageImage(imgPath)` through `cs.Pages[i].Enhancements = nil`:

```go
imgData, err := readPageImage(imgPath)
if err != nil {
    return fmt.Errorf("page %d: %w", cs.Pages[i].PageNumber, err)
}

resp, err := a.Vision(gctx,
    []protocol.Message{protocol.UserMessage(prompt)},
    []format.Image{{Data: imgData, Format: "png"}},
)
if err != nil {
    return fmt.Errorf("page %d: vision call: %w", cs.Pages[i].PageNumber, err)
}

parsed, err := formatting.Parse[enhanceResponse](resp.Text())
if err != nil {
    return fmt.Errorf("page %d: parse response: %w", cs.Pages[i].PageNumber, err)
}

cs.Pages[i].MarkingsFound = parsed.MarkingsFound
cs.Pages[i].Rationale = parsed.Rationale
cs.Pages[i].Enhancements = nil

return nil
```

The surrounding errgroup, pdfDoc handling, agent creation, and `rerender` call stay as-is.

### Step 10 — `internal/workflow/finalize.go`

**9a. Update imports:**

```go
import (
    "context"
    "fmt"

    "github.com/tailored-agentic-units/orchestrate/state"
    "github.com/tailored-agentic-units/protocol"

    "github.com/JaimeStill/herald/internal/prompts"
    "github.com/JaimeStill/herald/pkg/formatting"
)
```

**9b. Rewrite the Chat call inside `synthesize`.** Replace the section from `resp, err := a.Chat(ctx, prompt)` through the `parsed` assignment:

```go
resp, err := a.Chat(ctx, []protocol.Message{protocol.UserMessage(prompt)})
if err != nil {
    return fmt.Errorf("%w: chat call: %w", ErrFinalizeFailed, err)
}

parsed, err := formatting.Parse[finalizeResponse](resp.Text())
if err != nil {
    return fmt.Errorf("%w: parse response: %w", ErrFinalizeFailed, err)
}
```

The rest of `synthesize` (setting `cs.Classification`, `cs.Confidence`, `cs.Rationale`) is unchanged.

### Step 11 — `internal/workflow/observer.go`

**11a. Update imports** — split the old observability import into two:

```go
import (
    "context"
    "log/slog"
    "sync"
    "time"

    "github.com/tailored-agentic-units/orchestrate/observability"
    "github.com/tailored-agentic-units/orchestrate/state"

    "github.com/JaimeStill/herald/pkg/formatting"
)
```

**11b. Update the event-type switch inside `OnEvent`.** Change the two case constants (only):

```go
switch event.Type {
case state.EventNodeStart:
    execEvent = handleNodeStart(event)
case state.EventNodeComplete:
    execEvent = o.handleNodeComplete(event)
}
```

The `observability.Event` parameter type and the rest of the file (close semantics, send helpers, handler bodies) stay identical.

### Step 12 — `internal/classifications/repository.go`

Import swap only:

```go
// before
"github.com/JaimeStill/go-agents/pkg/agent"

// after
"github.com/tailored-agentic-units/agent"
```

The `newAgent func(ctx context.Context) (agent.Agent, error)` parameter on `New` and its wiring into `workflow.Runtime` stays unchanged.

## Validation Criteria

- [ ] `grep -r "JaimeStill/go-agents" internal tests cmd pkg` returns no matches
- [ ] `go.mod` contains no `replace` directives and no `JaimeStill/go-agents*` entries
- [ ] `go mod tidy` runs clean (no errors, no unused requires)
- [ ] `mise run vet` passes
- [ ] `mise run test` passes for `tests/workflow/observer_test.go`, `tests/api/api_test.go`, `tests/infrastructure/infrastructure_test.go` (other suites unaffected by this migration)
- [ ] `mise run dev` with Azure credentials classifies one test PDF end-to-end through the web client and produces the expected classification/confidence/rationale
