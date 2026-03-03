# Objective: SSE Classification Streaming

**Issue**: #58
**Phase**: Phase 3 — Web Client (v0.3.0)

## Scope

Implement server-side SSE (Server-Sent Events) streaming for classification workflow progress. Adds a streaming observer to the workflow engine and an SSE endpoint that emits real-time progress events during document classification. Pure Go backend work — no UI consumption yet.

The SSE streaming endpoint replaces the existing `POST /api/classifications/{documentId}` endpoint as the sole classification path. A `GET /api/classifications/{documentId}/stream` endpoint initiates classification and streams progress via SSE, compatible with the browser EventSource API. The non-streaming `POST` classify endpoint and `classifications.System.Classify` method should be removed in #71 — maintaining two classify paths is unnecessary when SSE subsumes the non-streaming case.

## Sub-Issues

| # | Sub-Issue | Issue | Status | Depends On |
|---|-----------|-------|--------|------------|
| 1 | Workflow streaming observer and event types | #70 | Open | — |
| 2 | SSE classification streaming endpoint | #71 | Open | #70 |

## Dependency Graph

```
#70 (Workflow Observer)
        |
        v
#71 (SSE Endpoint)
```

Sequential — #71 imports types and calls the modified `Execute` from #70.

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Observer injection | `state.NewGraphWithDeps()` — direct instance injection | Agent-lab pattern. Bypasses string-based observer registry. No registration/cleanup needed. |
| `Execute` parameter | `observer *StreamingObserver` (nil = noop) | Explicit, idiomatic Go. `NewGraphWithDeps` defaults nil observer to `NoOpObserver{}`. |
| Event data decoding | `formatting.FromMap[T]` in `pkg/formatting/map.go` | Reusable JSON round-trip decoder alongside existing `Parse[T]`. Adapted from agent-lab's `pkg/decode.FromMap`. |
| SSE event format | Standard `event: <type>\ndata: <json>\n\n` | EventSource-compatible. Event types: `node.start`, `node.complete`, `complete`, `error`. Edge transitions omitted — redundant with node complete/start pairs. |
| Error handling | Pre-stream errors → JSON; mid-stream errors → SSE `error` event | Before SSE headers are sent, handler can return normal error responses. After headers, errors flow through the event channel. |
| Checkpoint store | `nil` passed to `NewGraphWithDeps` | Herald doesn't use checkpointing. |
