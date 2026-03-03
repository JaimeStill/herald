# 70 — Workflow Streaming Observer and Event Types

## Summary

Added streaming observer infrastructure to the classification workflow. A `StreamingObserver` translates verbose go-agents-orchestration events into lean progress events on a buffered channel, filtering ~12 orchestration event types down to 2 (node.start, node.complete) plus explicit complete/error events. `Execute` now accepts an optional observer parameter, and graph construction uses `NewGraphWithDeps` for direct observer injection.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Event scope | node.start + node.complete only | Edge transitions duplicate the node complete/start pair for adjacent nodes |
| `FromMap[T]` location | `pkg/formatting/parse.go` | Same family as `Parse[T]` — generic JSON decoders with different input types |
| Dead code cleanup | Removed `cfg.Observer = "noop"` | `NewGraphWithDeps` takes observer directly, ignoring config's Observer string field |
| Non-streaming path | Noted for removal in #71 | SSE streaming subsumes the non-streaming `POST` classify endpoint |

## Files Modified

- `pkg/formatting/parse.go` — added `FromMap[T]` generic map-to-struct decoder
- `internal/workflow/events.go` — new: `ExecutionEventType`, `ExecutionEvent`, `NodeStartData`, `NodeCompleteData`
- `internal/workflow/observer.go` — new: `StreamingObserver` with channel-based events, `observability.Observer` implementation
- `internal/workflow/workflow.go` — `Execute` and `buildGraph` accept observer; switched to `NewGraphWithDeps`
- `internal/classifications/repository.go` — pass `nil` observer at existing call site
- `_project/objective.md` — updated scope to note non-streaming path removal in #71, corrected SSE event types
- `tests/formatting/parse_test.go` — added `TestFromMap` (4 cases)
- `tests/workflow/observer_test.go` — new: 12 test functions covering event translation, filtering, close safety, non-blocking sends

## Patterns Established

- **Translation boundary pattern**: `StreamingObserver.OnEvent` receives orchestration framework events but constructs new `Data map[string]any` with only controlled fields — framework internals never leak to SSE consumers
- **Non-blocking channel send**: `select { case ch <- event: default: }` prevents slow consumers from blocking graph execution

## Validation Results

- `go vet ./...` — clean
- `go test ./tests/...` — 20 packages pass
- `go mod tidy` — no changes
