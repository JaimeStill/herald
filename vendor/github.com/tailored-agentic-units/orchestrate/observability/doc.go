// Package observability provides event-based observability for orchestrate
// subsystems. Level values align with OpenTelemetry SeverityNumbers for
// zero-translation compatibility with OTel collectors.
//
// # Observers
//
// The package provides several Observer implementations:
//
//   - NoOpObserver: Discards all events with zero overhead
//   - SlogObserver: Emits events to a slog.Logger
//   - MultiObserver: Fans out events to multiple observers
//
// # Registry
//
// Observers are registered by name in a global registry. Pre-registered
// observers include "noop" and "slog". Custom observers can be added via
// RegisterObserver.
package observability
