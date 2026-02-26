# Issue #38 — Workflow Foundation: Types, Runtime, Errors, and Parsing

## Context

Herald's `workflow/` package currently contains only `doc.go`. All subsequent workflow sub-issues (#39 init node, #40 classify node, #41 enhance/graph assembly) depend on the foundational types, runtime struct, sentinel errors, JSON parsing, and prompt composition defined here. This issue establishes the shared building blocks before any node implementation begins.

## Dependencies to Add

```bash
go get github.com/JaimeStill/go-agents-orchestration
go get github.com/JaimeStill/document-context
```

Neither is currently in `go.mod`. Both are needed by later workflow sub-issues but adding them now ensures `go mod tidy` resolves cleanly with the new imports.

## Implementation Steps

### Step 1: Add dependencies and remove placeholder

- `go get` both libraries
- Delete `workflow/doc.go` (replaced by the new files)

### Step 2: `workflow/errors.go` — Sentinel errors

Four workflow-scoped errors following Herald's `var ErrX = errors.New(...)` pattern:

- `ErrDocumentNotFound`
- `ErrRenderFailed`
- `ErrClassifyFailed`
- `ErrEnhanceFailed`

No `MapHTTPStatus` — workflow is not an HTTP domain. Parse error lives in `pkg/formatting/` (see Step 5).

**Pattern reference:** `internal/prompts/errors.go`, `internal/documents/errors.go`

### Step 3: `workflow/types.go` — All shared types

**`Confidence`** — String type with constants `ConfidenceHigh`, `ConfidenceMedium`, `ConfidenceLow` (mirrors `prompts.Stage` pattern from `internal/prompts/stages.go`).

**`PageImage`** — `PageNumber int`, `ImagePath string`. File path in temp directory, not data URI.

**`PageClassification`** — Per-page LLM response matching the spec in `internal/prompts/specs.go`:
- `PageNumber`, `Classification`, `Confidence`, `MarkingsFound`, `Rationale`, `ImageQualityLimiting`

**`ClassificationState`** — Accumulated across pages:
- `Classification`, `Confidence`, `MarkingsFound` (all pages), `Rationale`, `QualityFactor` (any page quality-limited), `Pages []PageClassification`
- `Enhanced`, `PriorConfidence` (set by enhance stage)

**`QualityAssessment`** — `QualityLimiting bool`, `AffectedPages []int`. Derived from ClassificationState for enhance decision.

**`WorkflowResult`** — Wraps ClassificationState with document metadata: `DocumentID`, `Filename`, `PageCount`, `State`, `CompletedAt`.

### Step 4: `workflow/runtime.go` — Dependency bundle

Simple struct with exported fields (matches `internal/infrastructure/infrastructure.go` pattern):

```
Runtime {
    Agent     gaconfig.AgentConfig
    Storage   storage.System
    Documents documents.System
    Prompts   prompts.System
    Logger    *slog.Logger
}
```

Uses `gaconfig` import alias per existing convention.

### Step 5: `pkg/formatting/parse.go` — Generic JSON parsing

Adds `Parse[T any](content string) (T, error)` to the existing `pkg/formatting/` package (alongside `bytes.go`):
1. Try direct `json.Unmarshal`
2. Fall back to markdown code fence extraction via regex
3. Return `ErrParseFailed` (new sentinel in `pkg/formatting/`) wrapped with the raw content

Also adds `ErrParseFailed` — either inline in `parse.go` or in a new `errors.go` in the package (single error, inline is fine).

**Pattern reference:** `agent-lab/workflows/classify/parse.go` (generic with regex), `go-agents/tools/classify-docs/pkg/classify/parser.go` (simpler variant)

### Step 6: `workflow/prompts.go` — Prompt composition

`ComposePrompt(ctx, ps prompts.System, stage prompts.Stage, state *ClassificationState) (string, error)`:
1. Call `ps.Instructions(ctx, stage)` — tunable layer (DB override or hardcoded default)
2. Call `ps.Spec(ctx, stage)` — immutable output format
3. Combine: instructions + spec + serialized ClassificationState (when non-nil)

State is nil on first page (prompt = instructions + spec only). On subsequent pages, the accumulated state is JSON-serialized as context so the LLM knows what has been found.

## Files Summary

| File | Contents |
|------|----------|
| `workflow/errors.go` | 4 sentinel errors |
| `workflow/types.go` | `Confidence`, `PageImage`, `PageClassification`, `ClassificationState`, `QualityAssessment`, `WorkflowResult` |
| `workflow/runtime.go` | `Runtime` struct (AgentConfig, Storage, Documents, Prompts, Logger) |
| `pkg/formatting/parse.go` | `Parse[T any]` with markdown code fence fallback + `ErrParseFailed` |
| `workflow/prompts.go` | `ComposePrompt` composing instructions + spec + state |

## Validation

- `go vet ./...` passes
- `mise run build` succeeds
- `go mod tidy` produces no changes
- JSON tags on types match the LLM response spec in `internal/prompts/specs.go`
