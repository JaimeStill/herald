# 41 — Enhance Node, Finalize Node, Graph Assembly, and Execute Function

## Context

This is the final sub-issue of Objective #26 (Classification Workflow). The init node (#39) and classify node (#40) are complete. This issue adds the remaining pieces: a placeholder enhance node, a finalize node for document-level synthesis, the state graph wiring all nodes together, and the top-level `Execute` function.

## Files to Create

- `workflow/enhance.go` — placeholder enhance node
- `workflow/finalize.go` — document-level classification synthesis node
- `workflow/workflow.go` — graph assembly + `Execute` entry point

## Files to Modify

- `workflow/errors.go` — add `ErrFinalizeFailed` sentinel

## Implementation

### Step 1: Add `ErrFinalizeFailed` to `workflow/errors.go`

Add alongside existing sentinels:

```go
ErrFinalizeFailed = errors.New("finalize failed")
```

### Step 2: `workflow/enhance.go` — Placeholder Node

Pass-through node that returns state unchanged. Full signature so graph topology is complete.

```go
func EnhanceNode(rt *Runtime) state.StateNode {
    return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
        rt.Logger.InfoContext(ctx, "enhance node skipped (placeholder)")
        return s, nil
    })
}
```

Minimal — no state extraction needed since it's a no-op placeholder.

### Step 3: `workflow/finalize.go` — Synthesis Node

Performs a single `agent.Chat` inference (not Vision — no images) to produce document-level classification from all page findings.

**Response type:**

```go
type finalizeResponse struct {
    Classification string     `json:"classification"`
    Confidence     Confidence `json:"confidence"`
    Rationale      string     `json:"rationale"`
}
```

**Node flow:**
1. Extract `ClassificationState` from state bag (reuse `extractClassState` from classify.go)
2. Create agent via `agent.New(&rt.Agent)`
3. Compose prompt via `ComposePrompt(ctx, rt.Prompts, prompts.StageFinalize, classState)` — passes full state with all page data
4. Call `a.Chat(ctx, prompt)` — text-only, no images
5. Parse response via `formatting.Parse[finalizeResponse](resp.Content())`
6. Apply response fields to `ClassificationState` document-level fields
7. Write updated state back to bag

**Key:** Uses `prompts.StageFinalize` — the instructions and spec already exist in the prompts package.

### Step 4: `workflow/workflow.go` — Graph Assembly and Execute

**Custom predicate** for the conditional edge:

```go
func needsEnhance(s state.State) bool {
    val, ok := s.Get(KeyClassState)
    if !ok {
        return false
    }
    cs, ok := val.(ClassificationState)
    if !ok {
        return false
    }
    return cs.NeedsEnhance()
}
```

**Graph assembly** (`buildGraph`):
- Nodes: `init`, `classify`, `enhance`, `finalize`
- Edges:
  - `init → classify` (unconditional)
  - `classify → enhance` (predicate: `needsEnhance`)
  - `classify → finalize` (predicate: `!needsEnhance` via `state.Not`)
  - `enhance → finalize` (unconditional)
- Entry point: `init`
- Exit point: `finalize`
- Graph config: `config.DefaultGraphConfig("herald-classify")` with observer overridden to `"noop"` (Herald excludes observer infrastructure)

**`Execute` function:**

```go
func Execute(ctx context.Context, rt *Runtime, documentID uuid.UUID) (*WorkflowResult, error)
```

1. `os.MkdirTemp("", "herald-classify-*")` — create temp dir
2. `defer os.RemoveAll(tempDir)` — cleanup on all paths
3. Create initial state via `state.New(nil)`, set `KeyDocumentID` and `KeyTempDir`
4. `buildGraph(rt)` — assemble the graph
5. `graph.Execute(ctx, initialState)` — run it
6. Extract `WorkflowResult` from final state (`KeyClassState`, `KeyFilename`, `KeyPageCount`)
7. Return result with `time.Now()` as `CompletedAt`

## Key Patterns

- **Reuse `extractClassState`** from classify.go — finalize.go calls it the same way
- **Predicate wraps `NeedsEnhance()`** — custom `TransitionPredicate` extracts from state bag
- **`state.Not(needsEnhance)`** for the skip-enhance edge — uses built-in predicate combinator
- **`config.DefaultGraphConfig` with noop observer** — Herald explicitly excludes observer/checkpoint infrastructure
- **Node names as constants** not needed — only used in `buildGraph`, string literals are fine for a single-workflow project

## Validation Criteria

- [ ] `go vet ./...` passes
- [ ] `go build ./...` passes
- [ ] Enhance node returns state unchanged (placeholder)
- [ ] Finalize node uses `agent.Chat` (not Vision)
- [ ] Graph topology: init → classify → enhance? → finalize (single exit point)
- [ ] Execute creates temp directory and defers cleanup
- [ ] Execute creates initial state, runs graph, returns WorkflowResult
- [ ] Conditional edge uses `ClassificationState.NeedsEnhance()` via custom predicate
