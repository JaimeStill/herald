# 70 — Workflow Streaming Observer and Event Types

## Context

Part of Objective #58 (SSE Classification Streaming). The classification workflow currently uses a hardcoded `"noop"` observer. To support real-time progress streaming via SSE (issue #71), the workflow needs a channel-based observer that emits typed events as the graph executes. This issue establishes all workflow-level streaming infrastructure with no HTTP/SSE concerns.

## Files

| File | Action | Purpose |
|------|--------|---------|
| `pkg/formatting/map.go` | New | Generic `FromMap[T]` JSON round-trip decoder |
| `internal/workflow/events.go` | New | `ExecutionEvent`, `ExecutionEventType` constants, typed data structs |
| `internal/workflow/observer.go` | New | `StreamingObserver` — buffered channel, non-blocking sends, idempotent close |
| `internal/workflow/workflow.go` | Modify | `Execute` gains `observer` param; `buildGraph` switches to `NewGraphWithDeps` |
| `internal/classifications/repository.go` | Modify | Pass `nil` observer to preserve existing behavior |

## Implementation

### Step 1: `pkg/formatting/map.go`

Generic JSON round-trip decoder adapted from agent-lab's `pkg/decode.FromMap[T]`. Fits alongside existing `Parse[T]` in the formatting package.

```go
func FromMap[T any](data map[string]any) (T, error)
```

Marshal `data` to JSON bytes, unmarshal into `T`. Simple two-step round-trip.

### Step 2: `internal/workflow/events.go`

**Event type constants** (string type `ExecutionEventType`):
- `NodeStart` = `"node.start"`
- `NodeComplete` = `"node.complete"`
- `Complete` = `"complete"`
- `Error` = `"error"`

**Event struct** (`ExecutionEvent`):
- `Type ExecutionEventType`
- `Timestamp time.Time`
- `Data map[string]any`

**Typed data structs** for consumers to decode via `formatting.FromMap[T]`:
- `NodeStartData` — `Node string`, `Iteration int`
- `NodeCompleteData` — `Node string`, `Iteration int`, `Error bool`, `ErrorMessage string`

Edge transitions are omitted — they duplicate the node complete + node start pair for adjacent nodes.

### Step 3: `internal/workflow/observer.go`

`StreamingObserver` struct with:
- `events chan ExecutionEvent` (buffered)
- `mu sync.Mutex` + `closed bool` for idempotent close

**Constructor**: `NewStreamingObserver(bufferSize int) *StreamingObserver`

**Methods**:
- `OnEvent(ctx context.Context, event observability.Event)` — Implements `observability.Observer`. Acts as a **translation boundary**: receives verbose orchestration events but only emits lean progress events. Constructs new `Data map[string]any` with controlled fields — never passes through the framework's raw event data. Uses non-blocking `select` with `default` to prevent slow consumers from blocking graph execution.
- `Events() <-chan ExecutionEvent` — Read-only channel accessor.
- `Close()` — Mutex-protected idempotent channel close.
- `SendComplete(data map[string]any)` — Sends `Complete` event with result data.
- `SendError(err error, nodeName string)` — Sends `Error` event with error details.

**Event filtering in `OnEvent`** — only 2 of ~12 orchestration event types pass through, each translated to a lean Herald event with new data:
- `observability.EventNodeStart` → `NodeStart` with `NodeStartData{Node, Iteration}` (extracted from event, no state snapshots)
- `observability.EventNodeComplete` → `NodeComplete` with `NodeCompleteData{Node, Iteration, Error, ErrorMessage}` (no output snapshots)
- **All other event types silently dropped** — `state.*`, `graph.*`, `edge.*` events never reach SSE consumers

### Step 4: `internal/workflow/workflow.go`

**`Execute` signature change**:
```go
func Execute(ctx context.Context, rt *Runtime, documentID uuid.UUID, observer *StreamingObserver) (*WorkflowResult, error)
```

- Pass `observer` to `buildGraph`
- When observer is non-nil, call `observer.SendComplete` with final state data on success, or `observer.SendError` on failure

**`buildGraph` change**:
```go
func buildGraph(rt *Runtime, observer *StreamingObserver) (state.StateGraph, error)
```

- Remove `cfg.Observer = "noop"`
- Replace `state.NewGraph(cfg)` with `state.NewGraphWithDeps(cfg, observer, nil)`
- When `observer` is nil, `NewGraphWithDeps` defaults to `NoOpObserver{}` internally

### Step 5: `internal/classifications/repository.go`

Update call site at line 115:
```go
result, err := workflow.Execute(ctx, r.rt, documentID, nil)
```

Pass `nil` to preserve existing noop behavior. The streaming call path (issue #71) will construct and pass a `StreamingObserver`.

## Validation

- `go vet ./...` passes
- `go test ./tests/...` passes (existing tests unbroken)
- `go mod tidy` produces no changes
- `Execute(ctx, rt, docID, nil)` preserves existing noop behavior
- `NewStreamingObserver(32)` creates observer implementing `observability.Observer`
- `Close()` is safe to call multiple times
