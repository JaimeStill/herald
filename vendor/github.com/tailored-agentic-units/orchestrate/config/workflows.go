package config

// ChainConfig defines configuration for sequential chain execution.
//
// This configuration follows the tau-core pattern: used only during initialization,
// then transformed into domain objects. The Observer field is a string to enable
// JSON configuration with observer resolution at runtime.
//
// Example JSON:
//
//	{
//	  "capture_intermediate_states": true,
//	  "observer": "slog"
//	}
//
// Example usage:
//
//	var cfg config.ChainConfig
//	json.Unmarshal(data, &cfg)
//	result, err := workflows.ProcessChain(ctx, cfg, items, initial, processor, progress)
type ChainConfig struct {
	// CaptureIntermediateStates determines whether to capture state after each step.
	// When true, ChainResult.Intermediate contains all intermediate states including initial.
	// When false, only final state is returned.
	CaptureIntermediateStates bool `json:"capture_intermediate_states"`

	// Observer specifies which observer implementation to use ("noop", "slog", etc.)
	Observer string `json:"observer"`
}

// DefaultChainConfig returns sensible defaults for chain execution.
//
// Uses "noop" observer for zero-overhead execution when observability not needed.
// Intermediate state capture is disabled by default to minimize memory usage.
func DefaultChainConfig() ChainConfig {
	return ChainConfig{
		CaptureIntermediateStates: false,
		Observer:                  "slog",
	}
}

func (c *ChainConfig) Merge(source *ChainConfig) {
	if c.CaptureIntermediateStates {
		c.CaptureIntermediateStates = source.CaptureIntermediateStates
	}

	if source.Observer != "" {
		c.Observer = source.Observer
	}
}

// ParallelConfig defines configuration for parallel execution pattern.
//
// This configuration controls worker pool sizing, error handling behavior, and
// observability for concurrent item processing. The configuration follows the
// tau-core pattern: used only during initialization, then transformed into
// domain objects.
//
// Worker Pool Sizing:
//   - MaxWorkers = 0: Auto-detect based on runtime.NumCPU() * 2, capped by WorkerCap
//   - MaxWorkers > 0: Use exact worker count, ignoring auto-detection
//   - WorkerCap: Maximum workers for auto-detection (prevents excessive goroutines)
//
// Error Handling:
//   - FailFast = true: Stop processing on first error, cancel all workers
//   - FailFast = false: Continue processing all items, collect all errors
//
// Example JSON:
//
//	{
//	  "max_workers": 4,
//	  "worker_cap": 16,
//	  "fail_fast": true,
//	  "observer": "slog"
//	}
//
// Example usage:
//
//	var cfg config.ParallelConfig
//	json.Unmarshal(data, &cfg)
//	result, err := workflows.ProcessParallel(ctx, cfg, items, processor, progress)
type ParallelConfig struct {
	// MaxWorkers specifies exact worker pool size (0 = auto-detect)
	MaxWorkers int `json:"max_workers"`

	// WorkerCap limits auto-detected workers (default: 16)
	WorkerCap int `json:"worker_cap"`

	// FailFastNil controls error handling behavior. Use FailFast() method to access.
	// When nil, defaults to true. Use pointer to distinguish unset from explicit false.
	FailFastNil *bool `json:"fail_fast"`

	// Observer specifies which observer implementation to use ("noop", "slog", etc.)
	Observer string `json:"observer"`
}

func (c *ParallelConfig) FailFast() bool {
	if c.FailFastNil == nil {
		return true
	}
	return *c.FailFastNil
}

// DefaultParallelConfig returns sensible defaults for parallel execution.
//
// Default configuration:
//   - MaxWorkers: 0 (auto-detect: min(NumCPU*2, WorkerCap, len(items)))
//   - WorkerCap: 16 (reasonable limit for I/O-bound work like agent API calls)
//   - FailFast: true (stop on first error for fast failure detection)
//   - Observer: "slog" (practical observability during development)
//
// The worker pool auto-detection balances concurrency with resource usage.
// For CPU-bound work, consider setting MaxWorkers to runtime.NumCPU().
// For I/O-bound work (agent API calls), the 2x multiplier provides good throughput.
func DefaultParallelConfig() ParallelConfig {
	failFast := true
	return ParallelConfig{
		MaxWorkers:  0,
		WorkerCap:   16,
		FailFastNil: &failFast,
		Observer:    "slog",
	}
}

func (c *ParallelConfig) Merge(source *ParallelConfig) {
	if source.MaxWorkers > 0 {
		c.MaxWorkers = source.MaxWorkers
	}

	if source.WorkerCap > 0 {
		c.WorkerCap = source.WorkerCap
	}

	if source.FailFastNil != nil {
		c.FailFastNil = source.FailFastNil
	}

	if source.Observer != "" {
		c.Observer = source.Observer
	}
}

type ConditionalConfig struct {
	Observer string `json:"observer"`
}

func DefaultConditionalConfig() ConditionalConfig {
	return ConditionalConfig{
		Observer: "slog",
	}
}

func (c *ConditionalConfig) Merge(source *ConditionalConfig) {
	if source.Observer != "" {
		c.Observer = source.Observer
	}
}
