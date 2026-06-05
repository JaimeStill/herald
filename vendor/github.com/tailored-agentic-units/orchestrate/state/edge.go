package state

// Edge represents a transition between nodes in a state graph.
//
// Edges define valid paths through the graph. The optional Predicate determines
// whether the transition should occur based on current state.
type Edge struct {
	// From is the source node name
	From string

	// To is the destination node name
	To string

	// Name is an optional identifier for the edge, typically used to describe
	// the predicate function being evaluated (e.g., "isApproved", "hasError")
	Name string

	// Predicate determines if this edge can be traversed (nil = always transition)
	Predicate TransitionPredicate
}

// TransitionPredicate evaluates state to determine if an edge can be traversed.
//
// Returns true if the transition should occur, false otherwise.
// Predicates enable conditional routing in state graphs.
type TransitionPredicate func(state State) bool

// AlwaysTransition returns a predicate that always evaluates to true.
//
// Use for unconditional transitions between nodes.
func AlwaysTransition() TransitionPredicate {
	return func(state State) bool { return true }
}

// KeyExists returns a predicate that checks if a key exists in state.
//
// Example:
//
//	edge := state.Edge{
//	    From: "process",
//	    To: "review",
//	    Predicate: state.KeyExists("result"),
//	}
func KeyExists(key string) TransitionPredicate {
	return func(state State) bool {
		_, exists := state.Get(key)
		return exists
	}
}

// KeyEquals returns a predicate that checks if a key has a specific value.
//
// Example:
//
//	edge := state.Edge{
//	    From: "review",
//	    To: "approve",
//	    Predicate: state.KeyEquals("status", "approved"),
//	}
func KeyEquals(key string, value any) TransitionPredicate {
	return func(state State) bool {
		val, exists := state.Get(key)
		return exists && val == value
	}
}

// Not inverts a predicate.
//
// Example:
//
//	notApproved := state.Not(state.KeyEquals("status", "approved"))
func Not(predicate TransitionPredicate) TransitionPredicate {
	return func(state State) bool {
		return !predicate(state)
	}
}

// And combines predicates with logical AND (all must be true).
//
// Example:
//
//	complex := state.And(
//	    state.KeyExists("user"),
//	    state.KeyEquals("role", "admin"),
//	)
func And(predicates ...TransitionPredicate) TransitionPredicate {
	return func(state State) bool {
		for _, p := range predicates {
			if !p(state) {
				return false
			}
		}
		return true
	}
}

// Or combines predicates with logical OR (at least one must be true).
//
// Example:
//
//	anyStatus := state.Or(
//	    state.KeyEquals("status", "approved"),
//	    state.KeyEquals("status", "pending"),
//	)
func Or(predicates ...TransitionPredicate) TransitionPredicate {
	return func(state State) bool {
		for _, p := range predicates {
			if p(state) {
				return true
			}
		}
		return false
	}
}
