package state

import "fmt"

// ExecutionError captures rich context when graph execution fails.
//
// This error type provides complete execution state for debugging:
//   - NodeName: Which node failed
//   - State: State snapshot at failure
//   - Path: Full execution path leading to failure
//   - Err: Underlying error from node or graph execution
type ExecutionError struct {
	NodeName string
	State    State
	Path     []string
	Err      error
}

// Error implements the error interface.
func (e *ExecutionError) Error() string {
	return fmt.Sprintf("execution failed at node %s: %v", e.NodeName, e.Err)
}

// Unwrap enables error unwrapping for errors.Is and errors.As.
func (e *ExecutionError) Unwrap() error {
	return e.Err
}
