package workflow

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/google/uuid"

	gaoconfig "github.com/JaimeStill/go-agents-orchestration/pkg/config"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
)

// Execute runs the classification workflow for a single document. It creates
// a temp directory for page images (cleaned up via defer), builds the state
// graph (init → classify → enhance? → finalize), executes it, and extracts
// the WorkflowResult from the final state.
func Execute(ctx context.Context, rt *Runtime, documentID uuid.UUID) (*WorkflowResult, error) {
	tempDir, err := os.MkdirTemp("", "herald-classify-*")
	if err != nil {
		return nil, fmt.Errorf("create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	graph, err := buildGraph(rt)
	if err != nil {
		return nil, fmt.Errorf("build graph: %w", err)
	}

	initialState := state.New(nil)
	initialState = initialState.Set(KeyDocumentID, documentID)
	initialState = initialState.Set(KeyTempDir, tempDir)

	finalState, err := graph.Execute(ctx, initialState)
	if err != nil {
		return nil, fmt.Errorf("execute graph: %w", err)
	}

	return extractResult(finalState)
}

func buildGraph(rt *Runtime) (state.StateGraph, error) {
	cfg := gaoconfig.DefaultGraphConfig("herald-classify")
	cfg.Observer = "noop"

	graph, err := state.NewGraph(cfg)
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
	if err := graph.AddEdge("classify", "finalize", state.Not(needsEnhance)); err != nil {
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

func extractResult(s state.State) (*WorkflowResult, error) {
	val, ok := s.Get(KeyClassState)
	if !ok {
		return nil, fmt.Errorf("missing %s in final state", KeyClassState)
	}

	cs, ok := val.(ClassificationState)
	if !ok {
		return nil, fmt.Errorf("%s is not ClassificationState", KeyClassState)
	}

	docIDVal, ok := s.Get(KeyDocumentID)
	if !ok {
		return nil, fmt.Errorf("missing %s in final state", KeyDocumentID)
	}

	documentID, ok := docIDVal.(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("%s is not uuid.UUID", KeyDocumentID)
	}

	filenameVal, ok := s.Get(KeyFilename)
	if !ok {
		return nil, fmt.Errorf("missing %s in final state", KeyFilename)
	}

	filename, ok := filenameVal.(string)
	if !ok {
		return nil, fmt.Errorf("%s is not string", KeyFilename)
	}

	pageCountVal, ok := s.Get(KeyPageCount)
	if !ok {
		return nil, fmt.Errorf("missing %s in final state", KeyPageCount)
	}

	pageCount, ok := pageCountVal.(int)
	if !ok {
		return nil, fmt.Errorf("%s is not int", KeyPageCount)
	}

	return &WorkflowResult{
		DocumentID:  documentID,
		Filename:    filename,
		PageCount:   pageCount,
		State:       cs,
		CompletedAt: time.Now(),
	}, nil
}

func needsEnhance(s state.State) bool {
	val, ok := s.Get(KeyClassState)
	if !ok {
		return false
	}

	cs, ok := val.(ClassificationState)
	if !ok {
		return false
	}

	return cs.NeedsEnhance()
}

func workerCount(pageCount int) int {
	return max(min(runtime.NumCPU(), pageCount), 1)
}
