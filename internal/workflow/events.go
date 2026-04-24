package workflow

import "time"

// ExecutionEventType categorizes streaming events emitted during workflow execution.
type ExecutionEventType string

// Event type constants for SSE streaming. Only node-level progress events are
// emitted; orchestration internals (state operations, edge transitions) are filtered out.
const (
	NodeStart    ExecutionEventType = "node.start"
	NodeComplete ExecutionEventType = "node.complete"
	Complete     ExecutionEventType = "complete"
	Error        ExecutionEventType = "error"
)

// ExecutionEvent is a lean progress event emitted by StreamingObserver.
// Data contains only controlled fields — orchestration framework internals
// are never passed through.
type ExecutionEvent struct {
	Type      ExecutionEventType `json:"type"`
	Timestamp time.Time          `json:"timestamp"`
	Data      map[string]any     `json:"data"`
}

// NodeStartData is the typed representation of Data for NodeStart events.
// Consumers decode it via core.FromMap[NodeStartData].
type NodeStartData struct {
	Node      string `json:"node"`
	Iteration int    `json:"iteration"`
}

// NodeCompleteData is the typed representation of Data for NodeComplete events.
// When Error is true, the event is re-typed as Error with the ErrorMessage.
type NodeCompleteData struct {
	Node         string `json:"node"`
	Iteration    int    `json:"iteration"`
	Error        bool   `json:"error,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}
