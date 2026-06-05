package observability

import "context"

// NoOpObserver discards all events with zero overhead.
type NoOpObserver struct{}

func (NoOpObserver) OnEvent(ctx context.Context, event Event) {}
