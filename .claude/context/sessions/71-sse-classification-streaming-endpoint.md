# 71 - SSE Classification Streaming Endpoint

## Summary

Modified the existing `POST /api/classifications/{documentId}` endpoint to return an SSE event stream instead of a JSON response. The `Classify` method, handler, and route were updated in-place — same URL, same method, streaming behavior. The StreamingObserver and ExecutionEvent types from #70 drive the event channel.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| In-place replacement | Modify `Classify` signature and handler rather than creating separate `ClassifyStream` | Single classify path — SSE subsumes the non-streaming case. Avoids maintaining two endpoints. |
| Same URL and method | `POST /{documentId}` unchanged | Semantically correct (POST initiates a side effect). Client uses `fetch` + `ReadableStream`, not `EventSource` (which requires GET). |
| Pre-flight validation | `r.rt.Documents.Find()` before goroutine launch | Synchronous document-not-found errors return JSON before SSE headers are sent. |
| Stream buffer constant | `streamBufferSize = 32` package-level constant | Static value, same pattern as agent-lab's `defaultStreamBufferSize`. |
| Complete event data | Serialized `Classification` struct as `map[string]any` | Client receives the persisted result in the completion event without a separate fetch. |

## Files Modified

- `internal/classifications/system.go` — `Classify` return type changed to `(<-chan workflow.ExecutionEvent, error)`
- `internal/classifications/repository.go` — `Classify` rewritten: pre-flight validation, goroutine with observer, workflow execution, DB persistence, complete/error events
- `internal/classifications/handler.go` — `Classify` handler rewritten: SSE headers, event loop with flush, context cancellation
- `tests/classifications/handler_test.go` — Mock and `TestHandlerClassify` updated for SSE streaming (verifies headers, event parsing, pre-stream errors)
- `_project/api/classifications/README.md` — Classify section updated with SSE event types and streaming curl example
- `_project/api/classifications/classifications.http` — Classify request annotated as SSE response

## Patterns Established

- SSE streaming from a domain handler: set headers, flush, range over event channel, format `event: <type>\ndata: <json>\n\n`, flush per event, respect `r.Context().Done()`
- Two-phase error handling: pre-stream errors return JSON, mid-stream errors arrive as SSE `error` events
- Domain-level SSE: System method returns `<-chan EventType` — handler only consumes, goroutine owns lifecycle

## Validation Results

- `go vet ./...` passes
- `go test ./tests/...` — all 20 packages pass
- Manual curl test: full SSE stream with node.start, node.complete, and complete events confirmed against live Azure endpoint
