// Package observability provides event-based observability for kernel and
// orchestrate subsystems. Level values align with OpenTelemetry SeverityNumbers
// for zero-translation compatibility with OTel collectors.
package observability

import (
	"context"
	"log/slog"
	"time"
)

// Level represents event severity aligned with OTel SeverityNumber ranges.
type Level int

const (
	LevelVerbose Level = 5  // OTel DEBUG (5-8), maps to slog.LevelDebug
	LevelInfo    Level = 9  // OTel INFO (9-12), maps to slog.LevelInfo
	LevelWarning Level = 13 // OTel WARN (13-16), maps to slog.LevelWarn
	LevelError   Level = 17 // OTel ERROR (17-20), maps to slog.LevelError
)

// String returns the OTel severity text for the level.
func (l Level) String() string {
	switch {
	case l <= 4:
		return "TRACE"
	case l <= 8:
		return "DEBUG"
	case l <= 12:
		return "INFO"
	case l <= 16:
		return "WARN"
	case l <= 20:
		return "ERROR"
	default:
		return "FATAL"
	}
}

// SlogLevel maps this level to the corresponding slog.Level for log emission.
func (l Level) SlogLevel() slog.Level {
	switch {
	case l <= 8:
		return slog.LevelDebug
	case l <= 12:
		return slog.LevelInfo
	case l <= 16:
		return slog.LevelWarn
	default:
		return slog.LevelError
	}
}

// EventType identifies the kind of event. Each subsystem defines its own
// constants using this type (e.g., "kernel.run.start", "graph.complete").
type EventType string

// Event is an observability event emitted by subsystems. Fields map to
// OTel LogRecord fields: Type→EventName, Level→SeverityNumber,
// Timestamp→Timestamp, Source→InstrumentationScope, Data→Attributes.
type Event struct {
	Type      EventType
	Level     Level
	Timestamp time.Time
	Source    string
	Data      map[string]any
}

// Observer receives events from subsystems for logging, tracing, or metrics.
type Observer interface {
	OnEvent(ctx context.Context, event Event)
}
