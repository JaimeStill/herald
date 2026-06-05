// Package state provides LangGraph-inspired state management for Go-native orchestration workflows.
//
// This package implements state graph execution primitives adapted from LangGraph but designed
// for Go's type system and concurrency patterns. State graphs enable workflow orchestration
// through nodes (computation steps), edges (transitions), and predicates (conditional routing).
//
// # Core Components
//
// State - Immutable key-value state with observer integration
//
// StateNode - Interface for computation steps that transform state
//
// Edge - Graph transitions with optional predicates
//
// StateGraph - Workflow definition and execution interface
//
// # State Type
//
// State uses map[string]any for maximum flexibility, similar to LangGraph's dictionary-based
// approach. All operations are immutable - modifications return new State instances.
//
//	observer := observability.NoOpObserver{}
//	s := state.New(observer)
//	s = s.Set("user", "alice")
//	s = s.Set("count", 42)
//
//	value, exists := s.Get("user")  // "alice", true
//
// # Immutability
//
// State operations never modify the original state. This enables:
//   - Safe concurrent access across goroutines
//   - Predictable workflow execution
//   - Easy debugging (state snapshots)
//   - Rollback capability through checkpointing
//
// # Observer Integration
//
// All state operations emit events through the observer interface, enabling
// production-grade observability without retrofit friction:
//
//	observer := &MyObserver{}
//	s := state.New(observer)
//	s = s.Set("key", "value")  // Emits EventStateSet
//
// When observability is not needed, use NoOpObserver for zero overhead.
//
// # Usage with Patterns
//
// State is designed to work as the TContext type for workflow patterns:
//
//	// Sequential chain using State
//	processor := func(ctx context.Context, item string, current state.State) (state.State, error) {
//	    return current.Set("result", item), nil
//	}
//	result, err := workflows.ProcessChain(ctx, cfg, items, initialState, processor, nil)
package state
