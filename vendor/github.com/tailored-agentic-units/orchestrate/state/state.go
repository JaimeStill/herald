package state

import (
	"context"
	"maps"
	"time"

	"github.com/google/uuid"
	"github.com/tailored-agentic-units/orchestrate/observability"
)

// State represents immutable workflow state flowing through graph execution.
//
// State uses map[string]any for maximum flexibility, enabling dynamic workflows
// similar to LangGraph. All operations are immutable - modifications return new
// State instances with updated values.
//
// State separates regular data from secrets:
//   - Data: Persisted to checkpoints and included in observer snapshots
//   - Secrets: Never persisted or observed (e.g., authentication tokens)
//
// Observer integration is built-in, enabling production-grade observability
// without retrofit friction.
//
// Checkpoint metadata (runID, checkpointNode, timestamp) provides execution
// provenance for workflow persistence and recovery. This metadata flows through
// all State transformations maintaining execution identity.
type State struct {
	Data           map[string]any        `json:"data"`
	Secrets        map[string]any        `json:"-"`
	Observer       observability.Observer `json:"-"`
	RunID          string                `json:"run_id"`
	CheckpointNode string                `json:"checkpoint_node"`
	Timestamp      time.Time             `json:"timestamp"`
}

// New creates a new empty State with the given observer.
//
// If observer is nil, NoOpObserver is used automatically. This prevents nil
// pointer dereferences while enabling zero-overhead operation when observability
// is not needed.
//
// Example:
//
//	observer := observability.NoOpObserver{}
//	s := state.New(observer)
func New(observer observability.Observer) State {
	if observer == nil {
		observer = observability.NoOpObserver{}
	}

	s := State{
		Data:      make(map[string]any),
		Secrets:   make(map[string]any),
		Observer:  observer,
		RunID:     uuid.New().String(),
		Timestamp: time.Now(),
	}

	observer.OnEvent(context.Background(), observability.Event{
		Type:      EventStateCreate,
		Level:     observability.LevelVerbose,
		Timestamp: s.Timestamp,
		Source:    "state",
		Data:      map[string]any{},
	})

	return s
}

// Clone creates an independent copy of the State.
//
// The returned State has its own data map (shallow clone) but preserves the
// same observer reference. Modifications to the clone do not affect the original.
//
// Uses maps.Clone for efficient copying.
//
// Example:
//
//	original := state.New(observer).Set("key", "value")
//	cloned := original.Clone()
//	cloned = cloned.Set("key", "modified")
//	// original still has "value", cloned has "modified"
func (s State) Clone() State {
	newState := State{
		Data:           maps.Clone(s.Data),
		Secrets:        maps.Clone(s.Secrets),
		Observer:       s.Observer,
		RunID:          s.RunID,
		CheckpointNode: s.CheckpointNode,
		Timestamp:      s.Timestamp,
	}

	s.Observer.OnEvent(context.Background(), observability.Event{
		Type:      EventStateClone,
		Level:     observability.LevelVerbose,
		Timestamp: time.Now(),
		Source:    "state",
		Data:      map[string]any{"keys": len(newState.Data)},
	})

	return newState
}

// Get retrieves a value from the State by key.
//
// Returns the value and true if the key exists, nil and false otherwise.
// Callers should check the exists flag before using the value to avoid nil panics.
//
// Example:
//
//	value, exists := state.Get("user")
//	if !exists {
//	    log.Fatal("user not found in state")
//	}
//	user := value.(string)  // Type assertion required due to any type
func (s State) Get(key string) (any, bool) {
	val, exists := s.Data[key]
	return val, exists
}

// Set creates a new State with the key-value pair added or updated.
//
// The original State is not modified (immutability). The new State preserves
// all existing keys and adds/updates the specified key.
//
// Emits EventStateSet through the observer.
//
// Example:
//
//	s1 := state.New(observer)
//	s2 := s1.Set("user", "alice")
//	s3 := s2.Set("count", 42)
//	// s1 is empty, s2 has user, s3 has user+count
func (s State) Set(key string, value any) State {
	newState := s.Clone()
	newState.Data[key] = value

	s.Observer.OnEvent(context.Background(), observability.Event{
		Type:      EventStateSet,
		Level:     observability.LevelVerbose,
		Timestamp: time.Now(),
		Source:    "state",
		Data:      map[string]any{"key": key},
	})

	return newState
}

// SetCheckpointNode creates a new State with updated checkpoint metadata.
//
// This method updates the checkpointNode field and refreshes the timestamp
// to mark when the checkpoint was taken. The original State is not modified
// (immutability preserved).
//
// Called by the graph execution engine after successful node execution to
// track execution progress for workflow persistence and recovery.
//
// Example:
//
//	s1 := state.New(observer).Set("data", "value")
//	s2 := s1.SetCheckpointNode("process")
//	// s2 has checkpoint metadata, s1 is unchanged
func (s State) SetCheckpointNode(node string) State {
	newState := s.Clone()
	newState.CheckpointNode = node
	newState.Timestamp = time.Now()
	return newState
}

// Merge creates a new State combining this State with another State.
//
// Keys from the other State are copied into the new State, overwriting any
// existing keys with the same name. The original States are not modified.
//
// Uses maps.Copy for efficient merging.
//
// Emits EventStateMerge through the observer.
//
// Example:
//
//	s1 := state.New(observer).Set("user", "alice").Set("role", "admin")
//	s2 := state.New(observer).Set("count", 42).Set("role", "user")
//	merged := s1.Merge(s2)
//	// merged has: user=alice, role=user (overwritten), count=42
func (s State) Merge(other State) State {
	newState := s.Clone()
	maps.Copy(newState.Data, other.Data)

	s.Observer.OnEvent(context.Background(), observability.Event{
		Type:      EventStateMerge,
		Level:     observability.LevelVerbose,
		Timestamp: time.Now(),
		Source:    "state",
		Data:      map[string]any{"keys": len(other.Data)},
	})

	return newState
}

// Checkpoint saves this State to the given CheckpointStore.
//
// This is a convenience method that delegates to store.Save(s). It enables
// State to be self-checkpointing without directly depending on storage
// implementation details.
//
// Returns error if the checkpoint save fails. The graph execution engine
// treats checkpoint save errors as fatal when checkpointing is enabled.
//
// Example:
//
//	store := state.NewMemoryCheckpointStore()
//	s := state.New(observer).Set("progress", "50%")
//	if err := s.Checkpoint(store); err != nil {
//	    log.Fatal(err)
//	}
func (s State) Checkpoint(store CheckpointStore) error {
	return store.Save(s)
}

// GetSecret retrieves a secret value from the State by key.
//
// Returns the value and true if the key exists, nil and false otherwise.
// Secrets are never persisted to checkpoints or included in observer snapshots,
// making them suitable for sensitive data like authentication tokens.
//
// Example:
//
//	token, exists := state.GetSecret("token")
//	if !exists {
//	    log.Fatal("token not found in state")
//	}
//	authToken := token.(string)  // Type assertion required due to any type
func (s State) GetSecret(key string) (any, bool) {
	val, exists := s.Secrets[key]
	return val, exists
}

// SetSecret creates a new State with the secret key-value pair added or updated.
//
// The original State is not modified (immutability). The new State preserves
// all existing secrets and adds/updates the specified key.
//
// Unlike Set, SetSecret does not emit observer events, ensuring secrets remain
// invisible to observability systems.
//
// Example:
//
//	s1 := state.New(observer)
//	s2 := s1.SetSecret("token", "bearer-xyz")
//	// s1 has no secrets, s2 has token
func (s State) SetSecret(key string, value any) State {
	state := s.Clone()
	state.Secrets[key] = value
	return state
}

// DeleteSecret creates a new State with the specified secret removed.
//
// The original State is not modified (immutability). If the key does not exist,
// the returned State is effectively a clone of the original.
//
// Example:
//
//	s1 := state.New(observer).SetSecret("token", "bearer-xyz")
//	s2 := s1.DeleteSecret("token")
//	// s1 still has token, s2 has no secrets
func (s State) DeleteSecret(key string) State {
	state := s.Clone()
	delete(state.Secrets, key)
	return state
}
