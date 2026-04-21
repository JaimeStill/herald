package workflow_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/tailored-agentic-units/orchestrate/observability"
	"github.com/tailored-agentic-units/orchestrate/state"

	"github.com/JaimeStill/herald/internal/workflow"
)

var testLogger = slog.Default()

func TestNewStreamingObserver(t *testing.T) {
	obs := workflow.NewStreamingObserver(32, testLogger)
	if obs == nil {
		t.Fatal("NewStreamingObserver returned nil")
	}
	ch := obs.Events()
	if ch == nil {
		t.Fatal("Events() returned nil channel")
	}
}

func TestOnEventNodeStart(t *testing.T) {
	obs := workflow.NewStreamingObserver(8, testLogger)
	defer obs.Close()

	ts := time.Now()
	obs.OnEvent(context.Background(), observability.Event{
		Type:      state.EventNodeStart,
		Timestamp: ts,
		Source:    "graph",
		Data: map[string]any{
			"node":      "classify",
			"iteration": float64(1),
		},
	})

	select {
	case event := <-obs.Events():
		if event.Type != workflow.NodeStart {
			t.Errorf("Type = %q, want %q", event.Type, workflow.NodeStart)
		}
		if event.Timestamp != ts {
			t.Error("Timestamp not preserved")
		}
		if event.Data["node"] != "classify" {
			t.Errorf("Data[node] = %v, want classify", event.Data["node"])
		}
		if event.Data["iteration"] != 1 {
			t.Errorf("Data[iteration] = %v, want 1", event.Data["iteration"])
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestOnEventNodeComplete(t *testing.T) {
	obs := workflow.NewStreamingObserver(8, testLogger)
	defer obs.Close()

	ts := time.Now()
	obs.OnEvent(context.Background(), observability.Event{
		Type:      state.EventNodeComplete,
		Timestamp: ts,
		Source:    "graph",
		Data: map[string]any{
			"node":      "init",
			"iteration": float64(1),
		},
	})

	select {
	case event := <-obs.Events():
		if event.Type != workflow.NodeComplete {
			t.Errorf("Type = %q, want %q", event.Type, workflow.NodeComplete)
		}
		if event.Data["node"] != "init" {
			t.Errorf("Data[node] = %v, want init", event.Data["node"])
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestOnEventNodeCompleteWithError(t *testing.T) {
	obs := workflow.NewStreamingObserver(8, testLogger)
	defer obs.Close()

	obs.OnEvent(context.Background(), observability.Event{
		Type:      state.EventNodeComplete,
		Timestamp: time.Now(),
		Source:    "graph",
		Data: map[string]any{
			"node":          "classify",
			"iteration":     float64(1),
			"error":         true,
			"error_message": "vision call failed",
		},
	})

	select {
	case event := <-obs.Events():
		if event.Type != workflow.Error {
			t.Errorf("Type = %q, want %q", event.Type, workflow.Error)
		}
		if event.Data["node"] != "classify" {
			t.Errorf("Data[node] = %v, want classify", event.Data["node"])
		}
		if event.Data["message"] != "vision call failed" {
			t.Errorf("Data[message] = %v, want 'vision call failed'", event.Data["message"])
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestOnEventIgnoresUnhandledTypes(t *testing.T) {
	obs := workflow.NewStreamingObserver(8, testLogger)
	defer obs.Close()

	ignoredTypes := []observability.EventType{
		state.EventStateCreate,
		state.EventStateSet,
		state.EventGraphStart,
		state.EventGraphComplete,
		state.EventEdgeTransition,
		state.EventEdgeEvaluate,
	}

	for _, et := range ignoredTypes {
		obs.OnEvent(context.Background(), observability.Event{
			Type:      et,
			Timestamp: time.Now(),
			Source:    "graph",
			Data:      map[string]any{"node": "test"},
		})
	}

	select {
	case event := <-obs.Events():
		t.Errorf("expected no events, got %+v", event)
	default:
	}
}

func TestSendComplete(t *testing.T) {
	obs := workflow.NewStreamingObserver(8, testLogger)
	defer obs.Close()

	data := map[string]any{"classification": "SECRET"}
	obs.SendComplete(data)

	select {
	case event := <-obs.Events():
		if event.Type != workflow.Complete {
			t.Errorf("Type = %q, want %q", event.Type, workflow.Complete)
		}
		if event.Data["classification"] != "SECRET" {
			t.Errorf("Data[classification] = %v, want SECRET", event.Data["classification"])
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestSendError(t *testing.T) {
	t.Run("with node name", func(t *testing.T) {
		obs := workflow.NewStreamingObserver(8, testLogger)
		defer obs.Close()

		obs.SendError(fmt.Errorf("connection lost"), "classify")

		select {
		case event := <-obs.Events():
			if event.Type != workflow.Error {
				t.Errorf("Type = %q, want %q", event.Type, workflow.Error)
			}
			if event.Data["message"] != "connection lost" {
				t.Errorf("Data[message] = %v, want 'connection lost'", event.Data["message"])
			}
			if event.Data["node"] != "classify" {
				t.Errorf("Data[node] = %v, want classify", event.Data["node"])
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for event")
		}
	})

	t.Run("without node name", func(t *testing.T) {
		obs := workflow.NewStreamingObserver(8, testLogger)
		defer obs.Close()

		obs.SendError(fmt.Errorf("general failure"), "")

		select {
		case event := <-obs.Events():
			if event.Data["message"] != "general failure" {
				t.Errorf("Data[message] = %v, want 'general failure'", event.Data["message"])
			}
			if _, ok := event.Data["node"]; ok {
				t.Error("Data[node] should not be present when nodeName is empty")
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for event")
		}
	})
}

func TestCloseIdempotent(t *testing.T) {
	obs := workflow.NewStreamingObserver(8, testLogger)
	obs.Close()
	obs.Close() // second call should not panic
}

func TestCloseStopsEvents(t *testing.T) {
	obs := workflow.NewStreamingObserver(8, testLogger)
	obs.Close()

	obs.OnEvent(context.Background(), observability.Event{
		Type:      state.EventNodeStart,
		Timestamp: time.Now(),
		Source:    "graph",
		Data:      map[string]any{"node": "init", "iteration": float64(1)},
	})

	// channel is closed, range would exit immediately
	count := 0
	for range obs.Events() {
		count++
	}
	if count != 0 {
		t.Errorf("expected 0 events after close, got %d", count)
	}
}

func TestSendCompleteAfterClose(t *testing.T) {
	obs := workflow.NewStreamingObserver(8, testLogger)
	obs.Close()
	obs.SendComplete(map[string]any{"result": "test"}) // should not panic
}

func TestSendErrorAfterClose(t *testing.T) {
	obs := workflow.NewStreamingObserver(8, testLogger)
	obs.Close()
	obs.SendError(fmt.Errorf("test"), "node") // should not panic
}

func TestNonBlockingSendOnFullBuffer(t *testing.T) {
	obs := workflow.NewStreamingObserver(1, testLogger) // buffer of 1
	defer obs.Close()

	// fill the buffer
	obs.SendComplete(map[string]any{"first": true})

	// this should not block — event is silently dropped
	obs.SendComplete(map[string]any{"second": true})

	event := <-obs.Events()
	if event.Data["first"] != true {
		t.Error("expected first event in buffer")
	}
}
