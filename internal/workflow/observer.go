package workflow

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/tailored-agentic-units/orchestrate/observability"
	"github.com/tailored-agentic-units/orchestrate/state"

	"github.com/JaimeStill/herald/pkg/formatting"
)

// StreamingObserver translates verbose tau/orchestrate events into lean
// ExecutionEvents on a buffered channel. It implements observability.Observer and
// acts as a filtering boundary: only node.start and node.complete events pass
// through with controlled data fields. Non-blocking sends prevent slow consumers
// from stalling graph execution.
type StreamingObserver struct {
	events chan ExecutionEvent
	logger *slog.Logger
	mu     sync.Mutex
	closed bool
}

// NewStreamingObserver creates a StreamingObserver with the given channel buffer size.
func NewStreamingObserver(bufferSize int, logger *slog.Logger) *StreamingObserver {
	return &StreamingObserver{
		events: make(chan ExecutionEvent, bufferSize),
		logger: logger,
	}
}

// Events returns a read-only channel for consuming execution events.
func (o *StreamingObserver) Events() <-chan ExecutionEvent {
	return o.events
}

// Close closes the event channel. Safe to call multiple times.
func (o *StreamingObserver) Close() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.closed {
		o.closed = true
		close(o.events)
	}
}

// OnEvent implements observability.Observer. It translates EventNodeStart and
// EventNodeComplete into lean Herald events; all other event types are dropped.
func (o *StreamingObserver) OnEvent(ctx context.Context, event observability.Event) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return
	}

	var execEvent *ExecutionEvent

	switch event.Type {
	case state.EventNodeStart:
		execEvent = handleNodeStart(event)
	case state.EventNodeComplete:
		execEvent = o.handleNodeComplete(event)
	}

	if execEvent != nil {
		select {
		case o.events <- *execEvent:
		default:
		}
	}
}

// SendComplete sends a Complete event with the provided result data.
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

// SendError sends an Error event with the error message and optional node context.
func (o *StreamingObserver) SendError(err error, nodeName string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return
	}
	o.logger.Error("workflow error", "node", nodeName, "error", err)
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

func (o *StreamingObserver) handleNodeComplete(event observability.Event) *ExecutionEvent {
	data, err := formatting.FromMap[NodeCompleteData](event.Data)
	if err != nil {
		return nil
	}
	if data.Error {
		o.logger.Error("node failed", "node", data.Node, "error", data.ErrorMessage)
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
