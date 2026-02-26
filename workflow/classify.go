package workflow

import (
	"context"
	"fmt"
	"os"

	"github.com/JaimeStill/document-context/pkg/document"
	"github.com/JaimeStill/document-context/pkg/encoding"

	"github.com/JaimeStill/go-agents/pkg/agent"

	"github.com/JaimeStill/go-agents-orchestration/pkg/state"

	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/pkg/formatting"
)

type pageResponse struct {
	MarkingsFound []string `json:"markings_found"`
	Rationale     string   `json:"rationale"`
	Enhance       bool     `json:"enhance"`
	Enhancements  string   `json:"enhancements"`
}

// ClassifyNode returns a state node that performs sequential page-by-page
// analysis. Each page image is encoded to a data URI just-in-time and sent
// to the vision model with accumulated prior page findings as context. The
// node populates per-page ClassificationPage fields only; document-level
// classification synthesis is deferred to the finalize node.
func ClassifyNode(rt *Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		classState, err := extractClassState(s)
		if err != nil {
			return s, fmt.Errorf("classify: %w", err)
		}

		if err := classifyPages(ctx, rt, classState); err != nil {
			return s, fmt.Errorf("classify: %w", err)
		}

		rt.Logger.InfoContext(
			ctx, "classify node complete",
			"page_count", len(classState.Pages),
		)

		s = s.Set(KeyClassState, *classState)
		return s, nil
	})
}

func extractClassState(s state.State) (*ClassificationState, error) {
	val, ok := s.Get(KeyClassState)
	if !ok {
		return nil, fmt.Errorf("%w: missing %s in state", ErrClassifyFailed, KeyClassState)
	}

	cs, ok := val.(ClassificationState)
	if !ok {
		return nil, fmt.Errorf("%w: %s is not ClassificationState", ErrClassifyFailed, KeyClassState)
	}

	return &cs, nil
}

func classifyPages(ctx context.Context, rt *Runtime, cs *ClassificationState) error {
	a, err := agent.New(&rt.Agent)
	if err != nil {
		return fmt.Errorf("%w: create agent: %w", ErrClassifyFailed, err)
	}

	for i := range cs.Pages {
		if err := classifyPage(ctx, a, rt, cs, i); err != nil {
			return fmt.Errorf("%w: page %d: %w", ErrClassifyFailed, i+1, err)
		}

		rt.Logger.InfoContext(
			ctx, "page classified",
			"page", i+1,
			"total", len(cs.Pages),
			"markings", cs.Pages[i].MarkingsFound,
			"enhance", cs.Pages[i].Enhance,
		)
	}

	return nil
}

func classifyPage(ctx context.Context, a agent.Agent, rt *Runtime, cs *ClassificationState, pageIdx int) error {
	dataURI, err := encodePageImage(cs.Pages[pageIdx].ImagePath)
	if err != nil {
		return err
	}

	var promptState *ClassificationState
	if pageIdx > 0 {
		promptState = cs
	}

	prompt, err := ComposePrompt(ctx, rt.Prompts, prompts.StageClassify, promptState)
	if err != nil {
		return err
	}

	resp, err := a.Vision(ctx, prompt, []string{dataURI})
	if err != nil {
		return fmt.Errorf("vision call: %w", err)
	}

	parsed, err := formatting.Parse[pageResponse](resp.Content())
	if err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	applyPageResponse(&cs.Pages[pageIdx], parsed)
	return nil
}

func encodePageImage(imagePath string) (string, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("read image: %w", err)
	}

	dataURI, err := encoding.EncodeImageDataURI(data, document.PNG)
	if err != nil {
		return "", fmt.Errorf("encode image: %w", err)
	}

	return dataURI, nil
}

func applyPageResponse(page *ClassificationPage, resp pageResponse) {
	page.MarkingsFound = resp.MarkingsFound
	page.Rationale = resp.Rationale
	page.Enhance = resp.Enhance
	page.Enhancements = resp.Enhancements
}
