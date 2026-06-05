package state

import "github.com/tailored-agentic-units/orchestrate/observability"

const (
	// State operations
	EventStateCreate observability.EventType = "state.create"
	EventStateClone  observability.EventType = "state.clone"
	EventStateSet    observability.EventType = "state.set"
	EventStateMerge  observability.EventType = "state.merge"

	// Graph execution
	EventGraphStart     observability.EventType = "graph.start"
	EventGraphComplete  observability.EventType = "graph.complete"
	EventNodeStart      observability.EventType = "node.start"
	EventNodeComplete   observability.EventType = "node.complete"
	EventNodeState      observability.EventType = "node.state"
	EventEdgeEvaluate   observability.EventType = "edge.evaluate"
	EventEdgeTransition observability.EventType = "edge.transition"
	EventCycleDetected  observability.EventType = "cycle.detected"

	// Checkpointing
	EventCheckpointSave   observability.EventType = "checkpoint.save"
	EventCheckpointLoad   observability.EventType = "checkpoint.load"
	EventCheckpointResume observability.EventType = "checkpoint.resume"
)
