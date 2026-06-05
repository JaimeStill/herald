package config

// CheckpointConfig controls workflow state persistence during graph execution.
//
// Configuration fields:
//   - Store: Name of CheckpointStore implementation to use (resolved via registry)
//   - Interval: Save checkpoint every N node executions (0 = disabled)
//   - Preserve: Keep checkpoints after successful completion (false = auto-cleanup)
//
// Example enabling checkpointing:
//
//	cfg := config.DefaultGraphConfig("workflow")
//	cfg.Checkpoint.Store = "memory"
//	cfg.Checkpoint.Interval = 5
//	cfg.Checkpoint.Preserve = true
type CheckpointConfig struct {
	// Store identifies which CheckpointStore to use (resolved via registry)
	Store string `json:"store"`

	// Interval controls checkpoint frequency (0 = disabled, N = every N nodes)
	Interval int `json:"interval"`

	// Preserve keeps checkpoints after successful execution (false = auto-cleanup)
	Preserve bool `json:"preserve"`
}

// DefaultCheckpointConfig returns checkpoint configuration with checkpointing disabled.
//
// Default values:
//   - Store: "memory" (though unused when Interval=0)
//   - Interval: 0 (checkpointing disabled)
//   - Preserve: false (auto-cleanup)
func DefaultCheckpointConfig() CheckpointConfig {
	return CheckpointConfig{
		Store:    "memory",
		Interval: 0,
		Preserve: false,
	}
}

func (c *CheckpointConfig) Merge(source *CheckpointConfig) {
	if source.Store != "" {
		c.Store = source.Store
	}

	if source.Interval > 0 {
		c.Interval = source.Interval
	}

	if source.Preserve {
		c.Preserve = source.Preserve
	}
}

// GraphConfig defines configuration for state graph execution.
//
// This configuration follows the tau-core pattern: used only during initialization,
// then transformed into domain objects. The Observer and Checkpoint.Store fields
// are strings to enable JSON configuration with runtime resolution via registries.
//
// Example JSON:
//
//	{
//	  "name": "document-workflow",
//	  "observer": "slog",
//	  "max_iterations": 500,
//	  "checkpoint": {
//	    "store": "memory",
//	    "interval": 10,
//	    "preserve": false
//	  }
//	}
//
// Example resolution:
//
//	var cfg config.GraphConfig
//	json.Unmarshal(data, &cfg)
//	graph, err := state.NewGraph(cfg)
type GraphConfig struct {
	// Name identifies the graph for observability
	Name string `json:"name"`

	// Observer specifies which observer implementation to use ("noop", "slog", etc.)
	Observer string `json:"observer"`

	// MaxIterations limits graph execution to prevent infinite loops
	MaxIterations int `json:"max_iterations"`

	// Checkpoint configures workflow state persistence and recovery
	Checkpoint CheckpointConfig `json:"checkpoint"`
}

// DefaultGraphConfig returns sensible defaults for graph execution.
//
// Default values:
//   - Observer: "slog" for structured logging
//   - MaxIterations: 1000 to protect against infinite loops
//   - Checkpoint: Disabled (Interval=0) for zero-overhead execution
func DefaultGraphConfig(name string) GraphConfig {
	return GraphConfig{
		Name:          name,
		Observer:      "slog",
		MaxIterations: 1000,
		Checkpoint:    DefaultCheckpointConfig(),
	}
}

func (c *GraphConfig) Merge(source *GraphConfig) {
	if source.Name != "" {
		c.Name = source.Name
	}

	if source.Observer != "" {
		c.Observer = source.Observer
	}

	if source.MaxIterations > 0 {
		c.MaxIterations = source.MaxIterations
	}

	c.Checkpoint.Merge(&source.Checkpoint)
}
