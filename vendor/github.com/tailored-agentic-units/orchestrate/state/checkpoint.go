package state

import (
	"fmt"
	"sync"
)

// CheckpointStore provides persistence for workflow state during execution.
//
// Implementations save State snapshots identified by RunID, enabling workflow
// recovery after failures or interruptions. The interface supports multiple
// storage backends (memory, disk, database) through the registry pattern.
//
// Checkpoint lifecycle:
//  1. Graph execution saves State at configured intervals via Save
//  2. On successful completion, checkpoints are deleted (unless Preserve=true)
//  3. On failure, checkpoints remain available for Resume
//  4. Resume loads checkpoint and continues from next node
//
// Implementations must be thread-safe for concurrent graph executions.
type CheckpointStore interface {
	// Save persists State identified by its RunID.
	// Overwrites any existing checkpoint for the same RunID.
	Save(state State) error

	// Load retrieves State for the given RunID.
	// Returns error if checkpoint not found.
	Load(runID string) (State, error)

	// Delete removes the checkpoint for the given RunID.
	// No error if checkpoint doesn't exist.
	Delete(runID string) error

	// List returns all RunIDs with stored checkpoints.
	// Useful for monitoring and cleanup operations.
	List() ([]string, error)
}

// memoryCheckpointStore implements CheckpointStore with in-memory storage.
//
// Thread-safe implementation using sync.RWMutex. Checkpoints are lost when
// process terminates - suitable for development and testing but not production
// recovery scenarios.
type memoryCheckpointStore struct {
	states map[string]State
	mu     sync.RWMutex
}

// NewMemoryCheckpointStore creates a CheckpointStore with in-memory storage.
//
// The memory store is registered by default as "memory" and can be used
// via configuration:
//
//	cfg := config.DefaultGraphConfig("workflow")
//	cfg.Checkpoint.Store = "memory"
//	cfg.Checkpoint.Interval = 5
func NewMemoryCheckpointStore() CheckpointStore {
	return &memoryCheckpointStore{
		states: make(map[string]State),
	}
}

func (m *memoryCheckpointStore) Save(state State) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.states[state.RunID] = state
	return nil
}

func (m *memoryCheckpointStore) Load(runID string) (State, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.states[runID]
	if !exists {
		return State{}, fmt.Errorf("checkpoint not found: %s", runID)
	}
	return state, nil
}

func (m *memoryCheckpointStore) Delete(runID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.states, runID)
	return nil
}

func (m *memoryCheckpointStore) List() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.states))
	for id := range m.states {
		ids = append(ids, id)
	}
	return ids, nil
}

// checkpointStores is the global registry of named CheckpointStore implementations.
//
// The "memory" store is registered by default. Custom stores can be added via
// RegisterCheckpointStore before graph initialization.
var (
	checkpointStores = map[string]CheckpointStore{
		"memory": NewMemoryCheckpointStore(),
	}
	mutex sync.RWMutex
)

// GetCheckpointStore retrieves a CheckpointStore by name from the registry.
//
// Returns error if the requested store is not registered. Use
// RegisterCheckpointStore to add custom implementations.
//
// This function is called by NewGraph during initialization to resolve the
// store specified in CheckpointConfig.Store.
//
// Example:
//
//	store, err := state.GetCheckpointStore("memory")
//	if err != nil {
//	    log.Fatal(err)
//	}
func GetCheckpointStore(name string) (CheckpointStore, error) {
	mutex.RLock()
	defer mutex.RUnlock()

	store, exists := checkpointStores[name]
	if !exists {
		return nil, fmt.Errorf("unknown checkpoint store: %s", name)
	}
	return store, nil
}

// RegisterCheckpointStore adds a named CheckpointStore to the global registry.
//
// Call this function before creating graphs that use the custom store. The
// store name can then be referenced in CheckpointConfig.Store.
//
// Example:
//
//	diskStore := NewDiskCheckpointStore("/var/checkpoints")
//	state.RegisterCheckpointStore("disk", diskStore)
//
//	cfg := config.DefaultGraphConfig("workflow")
//	cfg.Checkpoint.Store = "disk"
//	cfg.Checkpoint.Interval = 10
func RegisterCheckpointStore(name string, store CheckpointStore) {
	mutex.Lock()
	defer mutex.Unlock()

	checkpointStores[name] = store
}
