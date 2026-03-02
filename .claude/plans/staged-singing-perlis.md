# Objective Planning: #58 — SSE Classification Streaming

## Context

Objective #58 adds server-side SSE streaming for classification workflow progress. This is pure Go backend work — no UI consumption yet. The existing `POST /api/classifications/{documentId}` endpoint remains unchanged for non-streaming callers. A new `GET /api/classifications/{documentId}/stream` endpoint initiates classification and streams real-time progress events via SSE.

**Transition**: Objective #57 (Web Client Foundation) is 100% complete (4/4 sub-issues closed). Close #57, update `_project/phase.md`, replace `_project/objective.md`.

## Sub-Issues

### Sub-Issue 1: Workflow streaming observer and event types

**Scope**: Pure `internal/workflow/` package work — event types, channel-based observer, configurable observer on `Execute`.

**Files**:
- `internal/workflow/events.go` (new) — `ExecutionEvent` struct, `ExecutionEventType` constants (`NodeStart`, `NodeComplete`, `EdgeTransition`, `Complete`, `Error`), typed data structs (`NodeStartData`, `NodeCompleteData`, `EdgeTransitionData`)
- `internal/workflow/observer.go` (new) — `StreamingObserver` with buffered channel, mutex, closed flag. `OnEvent` converts orchestration events to `ExecutionEvent` via non-blocking send. `Events()`, `Close()`, `SendComplete()`, `SendError()`. Uses `formatting.FromMap[T]` for event data decoding
- `pkg/formatting/map.go` (new) — `FromMap[T any](data map[string]any) (T, error)` — generic JSON round-trip decoder for `map[string]any` → typed struct. Adapted from agent-lab's `pkg/decode.FromMap`. Fits alongside existing `Parse[T]` as a data conversion utility
- `internal/workflow/workflow.go` (modified) — `Execute` gains `observer *StreamingObserver` parameter. `buildGraph` switches from `state.NewGraph(cfg)` to `state.NewGraphWithDeps(cfg, observer, nil)` — directly injecting the observer instance, no registry. When observer is nil, `NewGraphWithDeps` defaults to `NoOpObserver{}`. Existing call site in classifications repo passes `nil`

**Key pattern** (from agent-lab `executor.go`): Agent-lab uses `state.NewGraphWithDeps(cfg, multiObs, checkpointStore)` to bypass the observer registry entirely. Herald follows the same approach — direct injection, no `RegisterObserver`/cleanup needed. Herald passes `nil` for checkpoint store (not used).

**Reference files**:
- `~/code/agent-lab/internal/workflows/streaming.go` — StreamingObserver adaptation source
- `~/code/agent-lab/internal/workflows/run.go` — ExecutionEvent type definitions
- `~/code/go-agents-orchestration/pkg/observability/observer.go` — Observer interface contract
- `~/code/go-agents-orchestration/pkg/state/graph.go` — `NewGraphWithDeps` API

**Acceptance Criteria**:
- [ ] `ExecutionEvent` and event type constants defined
- [ ] `StreamingObserver` implements `observability.Observer`
- [ ] Non-blocking sends (buffer size 32)
- [ ] `Close()` is idempotent
- [ ] `Execute(ctx, rt, docID, nil)` preserves noop behavior
- [ ] `Execute(ctx, rt, docID, observer)` injects observer via `NewGraphWithDeps`
- [ ] `formatting.FromMap[T]` added to `pkg/formatting/map.go`
- [ ] `go vet ./...` passes

---

### Sub-Issue 2: SSE classification streaming endpoint

**Scope**: Classifications system gains `ClassifyStream`, handler gains SSE endpoint, route registered.

**Files**:
- `internal/classifications/system.go` (modified) — Add `ClassifyStream(ctx, documentID) (<-chan workflow.ExecutionEvent, error)` to System interface
- `internal/classifications/repository.go` (modified) — Implement `ClassifyStream`: create observer, launch goroutine calling `workflow.Execute(ctx, rt, docID, observer)`, on success perform same upsert + status update as `Classify` then `observer.SendComplete(data)`, on error `observer.SendError(err)`, defer `observer.Close()`. Return `observer.Events()`. Update existing `Classify` to pass `nil` observer
- `internal/classifications/handler.go` (modified) — Add `ClassifyStream` handler: parse documentId, call `sys.ClassifyStream`, set SSE headers (`text/event-stream`, `no-cache`, `keep-alive`), flush, range over events writing `event: <type>\ndata: <json>\n\n`, flush per event, respect `r.Context().Done()`. Add route `GET /{documentId}/stream`
- `_project/api/classifications/README.md` (modified) — API Cartographer: SSE endpoint docs + curl example
- `_project/api/classifications/classifications.http` (modified) — SSE request example

**SSE handler pattern** (from agent-lab `handler.go`):
```
Set headers → WriteHeader(200) → Flush → range events { check ctx.Done; marshal+write; flush }
```

**Error handling**: Pre-stream errors (document not found, etc.) return JSON before SSE headers are set. Mid-stream errors arrive as `error` event type through the channel.

**Reference files**:
- `~/code/agent-lab/internal/workflows/handler.go` — SSE handler pattern (lines 71-116)
- `~/code/agent-lab/internal/workflows/executor.go` — Async execution with observer goroutine pattern

**Acceptance Criteria**:
- [ ] `ClassifyStream` on System interface and repo
- [ ] `GET /api/classifications/{documentId}/stream` returns SSE stream
- [ ] SSE headers correct
- [ ] Events formatted as `event: <type>\ndata: <json>\n\n`
- [ ] Client disconnect cancels workflow via context
- [ ] Pre-stream errors return JSON
- [ ] Existing `POST /{documentId}` unchanged
- [ ] API Cartographer updated
- [ ] `go vet ./...` passes

## Dependency Graph

```
Sub-Issue 1 (Workflow Observer)
        |
        v
Sub-Issue 2 (SSE Endpoint)
```

Sequential — sub-issue 2 imports types and calls the modified `Execute` from sub-issue 1.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Observer injection | `state.NewGraphWithDeps()` — direct instance injection | Agent-lab pattern. No registry registration/cleanup. Cleaner than string-based lookup |
| `Execute` parameter | `observer *StreamingObserver` (nil = noop) | Explicit, idiomatic Go. `NewGraphWithDeps` defaults nil to `NoOpObserver{}` |
| Event data decoding | `formatting.FromMap[T]` in `pkg/formatting/map.go` | Reusable JSON round-trip decoder alongside existing `Parse[T]`. Adapted from agent-lab's `pkg/decode.FromMap` |
| Buffer size | 32 | 4 nodes produce ~10 events. 3x headroom |
| Checkpoint store | `nil` | Herald doesn't use checkpointing |

## Transition Closeout

1. Close issue #57
2. Update `_project/phase.md` — mark Objective 1 status as `Complete`
3. Replace `_project/objective.md` with #58 content

## Verification

- `go vet ./...` passes after each sub-issue
- `go build ./cmd/server/` succeeds
- Manual test: `curl -N http://localhost:3000/api/classifications/{id}/stream` shows SSE events during classification
- Existing `POST` endpoint still works: `curl -X POST http://localhost:3000/api/classifications/{id}`
