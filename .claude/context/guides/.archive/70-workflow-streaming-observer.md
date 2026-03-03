# 70 — Workflow Streaming Observer and Event Types

## Problem Context

The classification workflow uses a hardcoded `"noop"` observer. To support real-time SSE progress streaming (issue #71), the workflow needs a channel-based observer that emits lean progress events as graph nodes start and complete. This establishes all workflow-level streaming infrastructure with no HTTP/SSE concerns.

## Architecture Approach

Direct adaptation of agent-lab's `StreamingObserver` pattern with Herald-specific simplifications:
- Only `node.start` and `node.complete` events pass through (edge transitions dropped as redundant)
- Event data is constructed fresh — orchestration framework's raw data never leaks to consumers
- `formatting.FromMap[T]` provides reusable generic decoding alongside existing `Parse[T]`
- Observer injection via `state.NewGraphWithDeps(cfg, observer, nil)` — nil defaults to `NoOpObserver{}`

## Implementation

### Step 1: `pkg/formatting/parse.go` (add to existing file)

Add `FromMap[T]` below the existing `Parse[T]` function:

```go
func FromMap[T any](data map[string]any) (T, error) {
	var result T
	b, err := json.Marshal(data)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(b, &result)
	return result, err
}
```

### Step 2: `internal/workflow/events.go` (new file)

```go
package workflow

import "time"

type ExecutionEventType string

const (
	NodeStart    ExecutionEventType = "node.start"
	NodeComplete ExecutionEventType = "node.complete"
	Complete     ExecutionEventType = "complete"
	Error        ExecutionEventType = "error"
)

type ExecutionEvent struct {
	Type      ExecutionEventType `json:"type"`
	Timestamp time.Time          `json:"timestamp"`
	Data      map[string]any     `json:"data"`
}

type NodeStartData struct {
	Node      string `json:"node"`
	Iteration int    `json:"iteration"`
}

type NodeCompleteData struct {
	Node         string `json:"node"`
	Iteration    int    `json:"iteration"`
	Error        bool   `json:"error,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}
```

### Step 3: `internal/workflow/observer.go` (new file)

```go
package workflow

import (
	"context"
	"sync"
	"time"

	"github.com/JaimeStill/go-agents-orchestration/pkg/observability"
	"github.com/JaimeStill/herald/pkg/formatting"
)

type StreamingObserver struct {
	events chan ExecutionEvent
	mu     sync.Mutex
	closed bool
}

func NewStreamingObserver(bufferSize int) *StreamingObserver {
	return &StreamingObserver{
		events: make(chan ExecutionEvent, bufferSize),
	}
}

func (o *StreamingObserver) Events() <-chan ExecutionEvent {
	return o.events
}

func (o *StreamingObserver) Close() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.closed {
		o.closed = true
		close(o.events)
	}
}

func (o *StreamingObserver) OnEvent(ctx context.Context, event observability.Event) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return
	}

	var execEvent *ExecutionEvent

	switch event.Type {
	case observability.EventNodeStart:
		execEvent = handleNodeStart(event)
	case observability.EventNodeComplete:
		execEvent = handleNodeComplete(event)
	}

	if execEvent != nil {
		select {
		case o.events <- *execEvent:
		default:
		}
	}
}

func (o *StreamingObserver) SendComplete(data map[string]any) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return
	}
	select {
	case o.events <- ExecutionEvent{
		Type:      Complete,
		Timestamp: time.Now(),
		Data:      data,
	}:
	default:
	}
}

func (o *StreamingObserver) SendError(err error, nodeName string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return
	}
	data := map[string]any{"message": err.Error()}
	if nodeName != "" {
		data["node"] = nodeName
	}
	select {
	case o.events <- ExecutionEvent{
		Type:      Error,
		Timestamp: time.Now(),
		Data:      data,
	}:
	default:
	}
}

func handleNodeStart(event observability.Event) *ExecutionEvent {
	data, err := formatting.FromMap[NodeStartData](event.Data)
	if err != nil {
		return nil
	}
	return &ExecutionEvent{
		Type:      NodeStart,
		Timestamp: event.Timestamp,
		Data: map[string]any{
			"node":      data.Node,
			"iteration": data.Iteration,
		},
	}
}

func handleNodeComplete(event observability.Event) *ExecutionEvent {
	data, err := formatting.FromMap[NodeCompleteData](event.Data)
	if err != nil {
		return nil
	}
	if data.Error {
		return &ExecutionEvent{
			Type:      Error,
			Timestamp: event.Timestamp,
			Data: map[string]any{
				"node":    data.Node,
				"message": data.ErrorMessage,
			},
		}
	}
	return &ExecutionEvent{
		Type:      NodeComplete,
		Timestamp: event.Timestamp,
		Data: map[string]any{
			"node":      data.Node,
			"iteration": data.Iteration,
		},
	}
}
```

### Step 4: `internal/workflow/workflow.go` (modify)

**`Execute` signature** — add `observer *StreamingObserver` parameter. Pass to `buildGraph`. On success, call `SendComplete` if observer is non-nil. On error, call `SendError` if observer is non-nil.

```go
func Execute(ctx context.Context, rt *Runtime, documentID uuid.UUID, observer *StreamingObserver) (*WorkflowResult, error) {
	tempDir, err := os.MkdirTemp("", "herald-classify-*")
	if err != nil {
		return nil, fmt.Errorf("create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	graph, err := buildGraph(rt, observer)
	if err != nil {
		return nil, fmt.Errorf("build graph: %w", err)
	}

	initialState := state.New(nil)
	initialState = initialState.Set(KeyDocumentID, documentID)
	initialState = initialState.Set(KeyTempDir, tempDir)

	finalState, err := graph.Execute(ctx, initialState)
	if err != nil {
		return nil, fmt.Errorf("execute graph: %w", err)
	}

	return extractResult(finalState)
}
```

**`buildGraph`** — accept observer, remove `cfg.Observer = "noop"`, switch to `state.NewGraphWithDeps`.

```go
func buildGraph(rt *Runtime, observer *StreamingObserver) (state.StateGraph, error) {
	cfg := gaoconfig.DefaultGraphConfig("herald-classify")

	graph, err := state.NewGraphWithDeps(cfg, observer, nil)
	if err != nil {
		return nil, err
	}

	// ... rest unchanged (AddNode, AddEdge, SetEntryPoint, SetExitPoint)
}
```

**Import change** — add `"github.com/JaimeStill/go-agents-orchestration/pkg/observability"` (needed by `state.NewGraphWithDeps` signature). Actually, check if `state.NewGraphWithDeps` accepts `observability.Observer` interface — if so, `*StreamingObserver` satisfies it via `OnEvent` method. The import is only needed if the compiler requires it for the interface type in the function signature.

### Step 5: `internal/classifications/repository.go` (modify)

Update the call site at line 115:

```go
result, err := workflow.Execute(ctx, r.rt, documentID, nil)
```

## Validation Criteria

- [ ] `go vet ./...` passes
- [ ] `go test ./tests/...` passes (existing tests unbroken)
- [ ] `go mod tidy` produces no changes
- [ ] `Execute(ctx, rt, docID, nil)` preserves existing noop behavior
- [ ] `NewStreamingObserver(32)` creates observer implementing `observability.Observer`
- [ ] `Close()` is safe to call multiple times
