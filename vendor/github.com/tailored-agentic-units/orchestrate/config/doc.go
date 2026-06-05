// Package config provides configuration structures for orchestration components.
//
// This package defines configuration types for hub instances and other orchestration
// primitives, establishing sensible defaults while allowing customization for
// different deployment scenarios.
//
// # Hub Configuration
//
// HubConfig defines settings for hub instances:
//
//	cfg := config.HubConfig{
//	    Name:              "processing-hub",
//	    ChannelBufferSize: 100,
//	    DefaultTimeout:    30 * time.Second,
//	    Logger:            slog.New(slog.NewJSONHandler(os.Stdout, nil)),
//	}
//
//	hub := hub.New(ctx, cfg)
//
// # Default Configuration
//
// The package provides defaults for common scenarios:
//
//	cfg := config.DefaultHubConfig()
//	// Name: "default"
//	// ChannelBufferSize: 100
//	// DefaultTimeout: 30s
//	// Logger: slog.Default()
//
// # Configuration Fields
//
// Name: Identifies the hub instance for logging and metrics
//
// ChannelBufferSize: Controls message channel capacity, affecting:
//   - Message throughput under load
//   - Memory usage per participant
//   - Backpressure characteristics
//
// DefaultTimeout: Request-response timeout when not specified by context:
//   - Prevents indefinite blocking
//   - Can be overridden per-request via context.WithTimeout
//
// Logger: Structured logging for hub operations:
//   - Participant registration/unregistration
//   - Message routing
//   - Error conditions
//
// # Design Principles
//
// Following orchestrate design principles:
//
//   - Configuration only exists during initialization
//   - Does not persist into runtime components
//   - Validation happens at point of use (hub/messaging packages)
//   - No circular dependencies with domain packages
//
// # Configuration Merging
//
// All configuration types support a Merge pattern. This enables layered
// configuration where loaded configs merge over defaults:
//
//	cfg := config.DefaultGraphConfig("workflow")
//	var loaded config.GraphConfig
//	json.Unmarshal(data, &loaded)
//	cfg.Merge(&loaded)
//
// Merge semantics by field type:
//
//   - Strings: Merge if source is non-empty
//   - Integers: Merge if source is greater than zero
//   - Durations: Merge if source is greater than zero
//   - Pointers: Merge if source is non-nil
//   - Nested configs: Recursive merge
//
// # Boolean Fields with Non-False Defaults
//
// For boolean fields where the default is true (e.g., ParallelConfig.FailFast),
// a pointer type (*bool) is used with an accessor method to distinguish between:
//
//   - nil: Field not specified, accessor returns default value
//   - &false: Explicitly set to false, accessor returns false
//   - &true: Explicitly set to true, accessor returns true
//
// The convention is to name the field with a "Nil" suffix (e.g., FailFastNil)
// and provide an accessor method with the original name (e.g., FailFast()):
//
//	type ParallelConfig struct {
//	    FailFastNil *bool `json:"fail_fast"`
//	}
//
//	func (c *ParallelConfig) FailFast() bool {
//	    if c.FailFastNil == nil {
//	        return true  // default
//	    }
//	    return *c.FailFastNil
//	}
//
// This prevents unintended behavior when unmarshaling partial JSON configs,
// where unspecified boolean fields would otherwise unmarshal to false and
// incorrectly override a true default.
//
// Example:
//
//	// Config file omits fail_fast entirely
//	{"max_workers": 4}
//
//	// Without *bool: FailFast becomes false (zero value), overriding default
//	// With *bool: FailFastNil is nil, FailFast() returns true (default)
//
// For boolean fields with false defaults, plain bool is sufficient since
// the zero value matches the default and only explicit true values need merging.
package config
