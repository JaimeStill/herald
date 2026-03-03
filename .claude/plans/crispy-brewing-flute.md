# 71 - SSE Classification Streaming Endpoint

## Context

Issue #71, part of Objective #58 (SSE Classification Streaming). Depends on #70 (merged in PR #72), which added `StreamingObserver`, `ExecutionEvent` types, and updated `workflow.Execute` to accept an `*StreamingObserver` parameter.

Per the consolidation note in `_project/objective.md`, the existing `POST /api/classifications/{documentId}` endpoint is modified in-place to return an SSE event stream instead of a JSON response. Same URL, same method — the behavior changes from synchronous JSON to streaming SSE.

## Architecture Approach

The SSE handler follows agent-lab's proven pattern (`internal/workflows/handler.go:71-116`): set SSE headers, flush, range over event channel, format as SSE, flush per event.

**Two error paths:**
- **Pre-stream errors** (document not found, parse failures): return JSON via `handlers.RespondError` — SSE headers haven't been sent yet
- **Mid-stream errors** (workflow failures): arrive as `error` event type on the channel

**Goroutine lifecycle:** `Classify` does pre-flight validation synchronously, then launches a goroutine that runs the workflow, persists the result (same upsert + status update logic), sends `complete` event with the classification data, and defers `observer.Close()`. The handler ranges over the channel and formats events as SSE.

## Implementation

### Step 1: Update System interface

**`internal/classifications/system.go`** — Change `Classify` return type:

```go
Classify(ctx context.Context, documentID uuid.UUID) (<-chan workflow.ExecutionEvent, error)
```

### Step 2: Modify Classify in repository

**`internal/classifications/repository.go`** — Modify `Classify` in-place:

1. Change return type to `(<-chan workflow.ExecutionEvent, error)`
2. Pre-flight: validate document exists via `r.rt.Documents.Find(ctx, documentID)` — returns error synchronously (before SSE headers)
3. Create `workflow.NewStreamingObserver(32)`
4. Launch goroutine with `defer observer.Close()`:
   - Call `workflow.Execute(ctx, r.rt, documentID, observer)`
   - On error: `observer.SendError(err, "")`, return
   - On success: collect markings, marshal, upsert classification (same SQL), update document status, then `observer.SendComplete(classMap)` with serialized `Classification`
   - DB errors after workflow also go to `observer.SendError`
5. Return `observer.Events(), nil`

The `collectMarkings` helper and upsert SQL are reused — the DB persistence logic moves into the goroutine.

### Step 3: Modify Classify handler

**`internal/classifications/handler.go`** — Modify `Classify` handler in-place:

1. Parse `documentId` path param — error returns JSON (pre-stream)
2. Call `sys.Classify(r.Context(), documentID)` — error returns JSON (pre-stream)
3. Set SSE headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`
4. `w.WriteHeader(http.StatusOK)` + flush
5. Range over events channel:
   - Check `r.Context().Done()` for client disconnect
   - Marshal event data to JSON
   - Write `event: <type>\ndata: <json>\n\n`
   - Flush after each event
6. `Routes()` unchanged — same `{Method: "POST", Pattern: "/{documentId}", Handler: h.Classify}`

### Step 4: Update API documentation

**`_project/api/classifications/README.md`** — Update "Classify Document" section:
- Same method/URL: `POST /api/classifications/{documentId}`
- Response type changes from JSON to SSE event stream
- Document event types: `node.start`, `node.complete`, `complete`, `error`
- curl example with `--no-buffer` for streaming

**`_project/api/classifications/classifications.http`** — Update classify request comment to note SSE response.

## Key Files

| File | Action |
|------|--------|
| `internal/classifications/system.go` | Modify (change `Classify` signature) |
| `internal/classifications/repository.go` | Modify (rewrite `Classify` to SSE-based) |
| `internal/classifications/handler.go` | Modify (rewrite handler, route unchanged) |
| `_project/api/classifications/README.md` | Modify (replace classify section) |
| `_project/api/classifications/classifications.http` | Modify (replace classify request) |

## Reference Patterns

- SSE handler: `~/code/agent-lab/internal/workflows/handler.go:71-116`
- Async execution with observer: `~/code/agent-lab/internal/workflows/executor.go:173-228`
- StreamingObserver API: `internal/workflow/observer.go` (NewStreamingObserver, Events, Close, SendComplete, SendError)
- ExecutionEvent types: `internal/workflow/events.go` (NodeStart, NodeComplete, Complete, Error)

## Validation Criteria

- [ ] `Classify` method on System interface returns `(<-chan workflow.ExecutionEvent, error)`
- [ ] `POST /api/classifications/{documentId}` returns SSE event stream
- [ ] SSE headers set correctly (`text/event-stream`, `no-cache`, `keep-alive`)
- [ ] Events formatted as `event: <type>\ndata: <json>\n\n`
- [ ] Client disconnect (context cancellation) stops the workflow
- [ ] Pre-stream errors return JSON error response (not SSE)
- [ ] Same `POST /{documentId}` endpoint, now returns SSE
- [ ] `_project/api/classifications/README.md` updated with SSE endpoint docs
- [ ] `go vet ./...` passes
