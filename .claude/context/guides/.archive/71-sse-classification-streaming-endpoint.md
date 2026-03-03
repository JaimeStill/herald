# 71 - SSE Classification Streaming Endpoint

## Problem Context

The existing `POST /api/classifications/{documentId}` endpoint runs the classification workflow synchronously and returns a JSON response. The web client needs real-time progress updates during classification. Per the consolidation note in `_project/objective.md`, the endpoint is modified in-place to return an SSE event stream instead of JSON — same URL, same method, streaming response.

Depends on #70 (merged), which added `StreamingObserver`, `ExecutionEvent` types, and updated `workflow.Execute` to accept an `*StreamingObserver` parameter.

## Architecture Approach

The `Classify` method, handler, and route are modified in-place. The method signature changes to return a read-only event channel. The handler switches from JSON response to SSE streaming. The route stays `POST /{documentId}`.

Two error paths:
- **Pre-stream errors** (bad UUID, document not found): return JSON via `handlers.RespondError` before SSE headers are sent
- **Mid-stream errors** (workflow failures): arrive as `error` event type on the channel

The goroutine owns the workflow execution, DB persistence, and observer lifecycle. The handler only consumes events from the channel.

## Implementation

### Step 1: Update System interface

**`internal/classifications/system.go`** — Change `Classify` return type. Add the `workflow` import.

```go
import (
	"context"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/internal/workflow"
	"github.com/JaimeStill/herald/pkg/pagination"
)
```

Change the `Classify` signature from:

```go
Classify(ctx context.Context, documentID uuid.UUID) (*Classification, error)
```

To:

```go
Classify(ctx context.Context, documentID uuid.UUID) (<-chan workflow.ExecutionEvent, error)
```

### Step 2: Modify Classify in repository

**`internal/classifications/repository.go`** — Rewrite the `Classify` method body. The method signature, upsert SQL, and `collectMarkings` helper are reused.

Add a constant at the top of the file (after the import block):

```go
const streamBufferSize = 32
```

Replace the entire `Classify` method with:

```go
func (r *repo) Classify(ctx context.Context, documentID uuid.UUID) (<-chan workflow.ExecutionEvent, error) {
	if _, err := r.rt.Documents.Find(ctx, documentID); err != nil {
		return nil, fmt.Errorf("document %s: %w", documentID, err)
	}

	observer := workflow.NewStreamingObserver(streamBufferSize)

	go func() {
		defer observer.Close()

		result, err := workflow.Execute(ctx, r.rt, documentID, observer)
		if err != nil {
			observer.SendError(fmt.Errorf("classify document %s: %w", documentID, err), "")
			return
		}

		markings := collectMarkings(result.State.Pages)
		markingsJSON, err := json.Marshal(markings)
		if err != nil {
			observer.SendError(fmt.Errorf("marshal markings: %w", err), "")
			return
		}

		upsertQ := `
			INSERT INTO classifications(
				document_id, classification, confidence, markings_found,
				rationale, model_name, provider_name
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (document_id) DO UPDATE SET
				classification = EXCLUDED.classification,
				confidence = EXCLUDED.confidence,
				markings_found = EXCLUDED.markings_found,
				rationale = EXCLUDED.rationale,
				classified_at = NOW(),
				model_name = EXCLUDED.model_name,
				provider_name = EXCLUDED.provider_name,
				validated_by = NULL,
				validated_at = NULL
			RETURNING id, document_id, classification, confidence, markings_found,
					  rationale, classified_at, model_name, provider_name,
					  validated_by, validated_at`

		upsertArgs := []any{
			documentID,
			result.State.Classification,
			string(result.State.Confidence),
			markingsJSON,
			result.State.Rationale,
			r.rt.Agent.Model.Name,
			r.rt.Agent.Provider.Name,
		}

		c, err := repository.WithTx(ctx, r.db, func(tx *sql.Tx) (Classification, error) {
			cl, err := repository.QueryOne(ctx, tx, upsertQ, upsertArgs, scanClassification)
			if err != nil {
				return Classification{}, fmt.Errorf("upsert classification: %w", err)
			}

			if err := repository.ExecExpectOne(
				ctx, tx,
				"UPDATE documents SET status = 'review', updated_at = NOW() WHERE id = $1",
				documentID,
			); err != nil {
				return Classification{}, fmt.Errorf("update document status: %w", err)
			}

			return cl, nil
		})

		if err != nil {
			observer.SendError(err, "")
			return
		}

		r.logger.Info("document classified",
			"id", c.ID,
			"document_id", documentID,
			"classification", c.Classification,
			"confidence", c.Confidence,
		)

		classBytes, err := json.Marshal(c)
		if err != nil {
			observer.SendError(fmt.Errorf("marshal classification: %w", err), "")
			return
		}

		var classMap map[string]any
		if err := json.Unmarshal(classBytes, &classMap); err != nil {
			observer.SendError(fmt.Errorf("unmarshal classification map: %w", err), "")
			return
		}

		observer.SendComplete(classMap)
	}()

	return observer.Events(), nil
}
```

### Step 3: Modify Classify handler

**`internal/classifications/handler.go`** — Rewrite the `Classify` handler. Add `"fmt"` to imports. The `Routes()` method is unchanged.

Replace the `Classify` method with:

```go
func (h *Handler) Classify(w http.ResponseWriter, r *http.Request) {
	documentID, err := uuid.Parse(r.PathValue("documentId"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, ErrNotFound)
		return
	}

	events, err := h.sys.Classify(r.Context(), documentID)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	for event := range events {
		select {
		case <-r.Context().Done():
			return
		default:
		}

		data, err := json.Marshal(event)
		if err != nil {
			h.logger.Error("failed to marshal event", "error", err)
			continue
		}

		fmt.Fprintf(w, "event: %s\n", event.Type)
		fmt.Fprintf(w, "data: %s\n\n", data)

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}
```

Add `"fmt"` to the import block (alongside existing `"encoding/json"`).

### Step 4: Update API documentation

**`_project/api/classifications/README.md`** — Replace the "Classify Document" section (lines 156-179) with:

````markdown
## Classify Document (SSE)

`POST /api/classifications/{documentId}`

Initiates the classification workflow for a document and streams progress events via Server-Sent Events. Runs the full workflow graph (init, classify, enhance?, finalize), then persists the classification result and transitions the document status to `review`. Re-classification overwrites any existing result and resets validation fields.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| documentId | uuid | Document UUID to classify |

### SSE Event Types

| Event | Description | Data Fields |
|-------|-------------|-------------|
| `node.start` | Workflow node begins execution | `node`, `iteration` |
| `node.complete` | Workflow node finished successfully | `node`, `iteration` |
| `error` | Error during workflow or persistence | `message`, optionally `node` |
| `complete` | Classification finished and persisted | Full classification object |

### Responses

| Status | Description |
|--------|-------------|
| 200 | SSE event stream (Content-Type: text/event-stream) |
| 404 | Document not found (JSON error, before stream starts) |

### Example

```bash
curl -s -N -X POST "$HERALD_API_BASE/api/classifications/660e8400-e29b-41d4-a716-446655440000"
````

**`_project/api/classifications/classifications.http`** — Update the classify request section. Replace:

```
### Classify Document

# Replace with a valid document ID

@documentId = 660e8400-e29b-41d4-a716-446655440000

POST {{HOST}}/api/classifications/{{documentId}} HTTP/1.1
```

With:

```
### Classify Document (SSE)

# Replace with a valid document ID
# Response is an SSE event stream (text/event-stream)

@documentId = 660e8400-e29b-41d4-a716-446655440000

POST {{HOST}}/api/classifications/{{documentId}} HTTP/1.1
```

## Validation Criteria

- [ ] `Classify` method on System interface returns `(<-chan workflow.ExecutionEvent, error)`
- [ ] `POST /api/classifications/{documentId}` returns SSE event stream
- [ ] SSE headers set correctly (`text/event-stream`, `no-cache`, `keep-alive`)
- [ ] Events formatted as `event: <type>\ndata: <json>\n\n`
- [ ] Client disconnect (context cancellation) stops the workflow
- [ ] Pre-stream errors return JSON error response (not SSE)
- [ ] `_project/api/classifications/README.md` updated with SSE endpoint docs
- [ ] `go vet ./...` passes
