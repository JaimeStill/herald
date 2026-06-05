package observability

import (
	"context"
	"log/slog"
)

// SlogObserver emits events to a slog.Logger. Event levels are mapped via
// SlogLevel, the event type becomes the log message, and Data keys are
// flattened as top-level slog attributes.
type SlogObserver struct {
	logger *slog.Logger
}

// NewSlogObserver creates a SlogObserver that emits to the given logger.
func NewSlogObserver(logger *slog.Logger) *SlogObserver {
	return &SlogObserver{logger: logger}
}

func (o *SlogObserver) OnEvent(ctx context.Context, event Event) {
	attrs := make([]slog.Attr, 0, len(event.Data)+1)
	attrs = append(attrs, slog.String("source", event.Source))
	for k, v := range event.Data {
		attrs = append(attrs, slog.Any(k, v))
	}

	o.logger.LogAttrs(ctx, event.Level.SlogLevel(), string(event.Type), attrs...)
}
