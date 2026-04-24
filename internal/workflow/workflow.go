package workflow

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/internal/state"

	tauconfig "github.com/tailored-agentic-units/orchestrate/config"
	taustate "github.com/tailored-agentic-units/orchestrate/state"
)

// Execute runs the classification workflow for a single document. It creates
// a temp directory for page images (cleaned up via defer), builds the state
// graph (init → classify → enhance? → finalize), executes it, and extracts
// the WorkflowResult from the final state.
func Execute(ctx context.Context, rt *Runtime, documentID uuid.UUID, observer *StreamingObserver) (*WorkflowResult, error) {
	tempDir, err := os.MkdirTemp("", "herald-classify-*")
	if err != nil {
		return nil, fmt.Errorf("create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	graph, err := buildGraph(rt, observer)
	if err != nil {
		return nil, fmt.Errorf("build graph: %w", err)
	}

	initialState := taustate.New(nil)
	initialState = initialState.Set(state.KeyDocumentID, documentID)
	initialState = initialState.Set(state.KeyTempDir, tempDir)

	finalState, err := graph.Execute(ctx, initialState)
	if err != nil {
		return nil, fmt.Errorf("execute graph: %w", err)
	}

	return extractResult(finalState)
}

func buildGraph(rt *Runtime, observer *StreamingObserver) (taustate.StateGraph, error) {
	cfg := tauconfig.DefaultGraphConfig("herald-classify")

	graph, err := taustate.NewGraphWithDeps(cfg, observer, nil)
	if err != nil {
		return nil, err
	}

	if err := graph.AddNode("init", InitNode(rt)); err != nil {
		return nil, err
	}

	if err := graph.AddNode("classify", ClassifyNode(rt)); err != nil {
		return nil, err
	}

	if err := graph.AddNode("enhance", EnhanceNode(rt)); err != nil {
		return nil, err
	}

	if err := graph.AddNode("finalize", FinalizeNode(rt)); err != nil {
		return nil, err
	}

	// init → classify (unconditional)
	if err := graph.AddEdge("init", "classify", nil); err != nil {
		return nil, err
	}

	// classify → enhance (when any page needs enhancement)
	if err := graph.AddEdge("classify", "enhance", needsEnhance); err != nil {
		return nil, err
	}

	// classify → finalize (when no enhancement needed)
	if err := graph.AddEdge("classify", "finalize", taustate.Not(needsEnhance)); err != nil {
		return nil, err
	}

	// enhance → finalize (unconditional)
	if err := graph.AddEdge("enhance", "finalize", nil); err != nil {
		return nil, err
	}

	if err := graph.SetEntryPoint("init"); err != nil {
		return nil, err
	}

	if err := graph.SetExitPoint("finalize"); err != nil {
		return nil, err
	}

	return graph, nil
}

func extractResult(s taustate.State) (*WorkflowResult, error) {
	val, ok := s.Get(state.KeyClassState)
	if !ok {
		return nil, fmt.Errorf("missing %s in final state", state.KeyClassState)
	}

	cs, ok := val.(state.ClassificationState)
	if !ok {
		return nil, fmt.Errorf("%s is not ClassificationState", state.KeyClassState)
	}

	docIDVal, ok := s.Get(state.KeyDocumentID)
	if !ok {
		return nil, fmt.Errorf("missing %s in final state", state.KeyDocumentID)
	}

	documentID, ok := docIDVal.(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("%s is not uuid.UUID", state.KeyDocumentID)
	}

	filenameVal, ok := s.Get(state.KeyFilename)
	if !ok {
		return nil, fmt.Errorf("missing %s in final state", state.KeyFilename)
	}

	filename, ok := filenameVal.(string)
	if !ok {
		return nil, fmt.Errorf("%s is not string", state.KeyFilename)
	}

	pageCountVal, ok := s.Get(state.KeyPageCount)
	if !ok {
		return nil, fmt.Errorf("missing %s in final state", state.KeyPageCount)
	}

	pageCount, ok := pageCountVal.(int)
	if !ok {
		return nil, fmt.Errorf("%s is not int", state.KeyPageCount)
	}

	return &WorkflowResult{
		DocumentID:  documentID,
		Filename:    filename,
		PageCount:   pageCount,
		State:       cs,
		CompletedAt: time.Now(),
	}, nil
}

func needsEnhance(s taustate.State) bool {
	val, ok := s.Get(state.KeyClassState)
	if !ok {
		return false
	}

	cs, ok := val.(state.ClassificationState)
	if !ok {
		return false
	}

	return cs.NeedsEnhance()
}
