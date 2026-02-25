# Plan: Issue #37 — Prompts Domain Extensions

## Context

The classification workflow (#26) needs a single source of truth for all prompt content. Currently the prompts domain only handles CRUD for named prompt overrides stored in PostgreSQL. The workflow needs two additional capabilities: (1) resolve the effective instructions for a stage — the active DB override or a hardcoded default — and (2) retrieve the immutable specification (output JSON schema + behavioral constraints) for a stage. Both must never return empty. Exposing specs via API gives prompt authors read-only context when crafting instructions.

## Approach

Extend the existing prompts domain with three additions: a `defaults.go` file containing hardcoded content, two new methods on the `System` interface, and two new handler routes.

## Files to Modify

| File | Action |
|------|--------|
| `internal/prompts/instructions.go` | **Create** — per-stage instruction constants, `instructions` map, `DefaultInstructions()` accessor |
| `internal/prompts/specs.go` | **Create** — per-stage spec constants, `specs` map, `Specification()` accessor |
| `internal/prompts/stages.go` | **Edit** — add `ParseStage(s string) (Stage, error)` for path param validation |
| `internal/prompts/system.go` | **Edit** — add `Instructions` and `Spec` to the interface |
| `internal/prompts/repository.go` | **Edit** — implement both methods on `repo` |
| `internal/prompts/handler.go` | **Edit** — add handler methods and routes |
| `_project/api/prompts/README.md` | **Edit** — document new endpoints |
| `_project/api/prompts/prompts.http` | **Edit** — add curl examples |

## Implementation Steps

### Step 1: `instructions.go` — hardcoded default instructions

New file with one constant per stage and an accessor function:

```go
const (
    initInstructions     = `...`
    classifyInstructions = `...`
    enhanceInstructions  = `...`
)

var instructions = map[Stage]string{
    StageInit:     initInstructions,
    StageClassify: classifyInstructions,
    StageEnhance:  enhanceInstructions,
}

func DefaultInstructions(stage Stage) (string, error)  // validates stage, returns content
```

Returns `ErrInvalidStage` if the stage is not found. Content will be placeholder text that captures the intent of each stage — refined during workflow implementation (#38–41).

### Step 2: `specs.go` — hardcoded specifications

Same structure as instructions — one constant per stage and an accessor function:

```go
const (
    initSpec     = `...`
    classifySpec = `...`
    enhanceSpec  = `...`
)

var specs = map[Stage]string{
    StageInit:     initSpec,
    StageClassify: classifySpec,
    StageEnhance:  enhanceSpec,
}

func Specification(stage Stage) (string, error)  // validates stage, returns content
```

Returns `ErrInvalidStage` if the stage is not found. Specs define the expected JSON output structure and behavioral constraints the workflow parser depends on.

### Step 3: `stages.go` — add `ParseStage`

Add a standalone validation function for path parameter parsing:

```go
func ParseStage(s string) (Stage, error)
```

Returns `ErrInvalidStage` if the string doesn't match a known stage. Used by handler methods to validate `{stage}` path params.

### Step 4: `system.go` — extend interface

Add two methods:

```go
Instructions(ctx context.Context, stage Stage) (string, error)
Spec(ctx context.Context, stage Stage) (string, error)
```

### Step 5: `repository.go` — implement methods

**`Instructions`**: Query for the active prompt for the given stage (`WHERE stage = $1 AND active = true`). If found, return its `instructions` field. If no active override exists (`sql.ErrNoRows`), fall back to `DefaultInstructions(stage)`. Validates stage before querying.

**`Spec`**: Delegates directly to `Specification(stage)`. No DB interaction — specs are always hardcoded. Included on the System interface so callers don't need to know the sourcing distinction.

### Step 6: `handler.go` — add routes and handlers

Response type for both endpoints:

```go
type StageContent struct {
    Stage   Stage  `json:"stage"`
    Content string `json:"content"`
}
```

Two new handler methods:
- `Instructions(w, r)` — parses `{stage}` via `ParseStage`, calls `sys.Instructions`, responds with `StageContent`
- `Spec(w, r)` — parses `{stage}` via `ParseStage`, calls `sys.Spec`, responds with `StageContent`

Two new routes added to `Routes()`:
- `GET /{stage}/instructions`
- `GET /{stage}/spec`

### Step 7: API Cartographer — update docs

Add endpoint documentation for both new routes in `_project/api/prompts/README.md` and curl examples in `prompts.http`.

## Validation Criteria

- [ ] `Instructions(ctx, stage)` returns active DB override when one exists
- [ ] `Instructions(ctx, stage)` returns hardcoded default when no active override exists
- [ ] `Spec(ctx, stage)` returns hardcoded specification
- [ ] Both methods return `ErrInvalidStage` for unknown stages and never return empty
- [ ] `ParseStage` validates string input against known stages
- [ ] Handler routes return `StageContent` JSON with 200 OK
- [ ] Handler routes return 400 Bad Request for invalid stage
- [ ] `go vet ./...` passes
- [ ] All existing tests pass
- [ ] API docs updated with new endpoints
