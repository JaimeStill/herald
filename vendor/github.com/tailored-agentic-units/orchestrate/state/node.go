package state

import "context"

// StateNode represents a computation step in a state graph.
//
// Nodes receive state, perform computation or agent calls, and return updated state.
// The interface is minimal to support diverse implementations (agent calls, data
// transforms, pattern execution, etc.).
type StateNode interface {
	// Execute transforms state based on node logic.
	// Returns updated state or error. Context enables cancellation/timeouts.
	Execute(ctx context.Context, state State) (State, error)
}

// FunctionNode wraps a function as a StateNode.
//
// This is the most common StateNode implementation, enabling inline node
// definitions without creating custom types.
type FunctionNode struct {
	fn func(ctx context.Context, state State) (State, error)
}

// NewFunctionNode creates a StateNode from a function.
//
// Example:
//
//	node := state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
//	    response, err := agent.Chat(ctx, "process data")
//	    if err != nil {
//	        return s, err
//	    }
//	    return s.Set("result", response.Content()), nil
//	})
func NewFunctionNode(fn func(context.Context, State) (State, error)) StateNode {
	return &FunctionNode{fn: fn}
}

// Execute runs the wrapped function with the given state.
func (n *FunctionNode) Execute(ctx context.Context, state State) (State, error) {
	return n.fn(ctx, state)
}
