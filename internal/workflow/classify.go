package workflow

import (
	"context"
	"fmt"
	"os"

	"github.com/JaimeStill/document-context/pkg/document"
	"github.com/JaimeStill/document-context/pkg/encoding"
	"golang.org/x/sync/errgroup"

	"github.com/JaimeStill/go-agents/pkg/agent"

	"github.com/JaimeStill/go-agents-orchestration/pkg/state"

	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/pkg/formatting"
)

type pageResponse struct {
	MarkingsFound []string         `json:"markings_found"`
	Rationale     string           `json:"rationale"`
	Enhance       bool             `json:"enhance"`
	Enhancements  *EnhanceSettings `json:"enhancements,omitempty"`
}

// ClassifyNode returns a state node that performs parallel page-by-page
// analysis using bounded errgroup concurrency. Each goroutine creates its
// own agent, encodes the page image to a data URI, and sends it to the
// vision model. Pages are classified independently (no accumulated context);
// document-level classification synthesis is deferred to the finalize node.
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
	prompt, err := ComposePrompt(ctx, rt.Prompts, prompts.StageClassify, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrClassifyFailed, err)
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(workerCount(len(cs.Pages)))

	for i := range cs.Pages {
		g.Go(func() error {
			if gctx.Err() != nil {
				return gctx.Err()
			}

			a, err := agent.New(&rt.Agent)
			if err != nil {
				return fmt.Errorf("page %d: create agent: %w", i+1, err)
			}

			dataURI, err := encodePageImage(cs.Pages[i].ImagePath)
			if err != nil {
				return fmt.Errorf("page %d: %w", i+1, err)
			}

			resp, err := a.Vision(gctx, prompt, []string{dataURI})
			if err != nil {
				return fmt.Errorf("page %d: vision call: %w", i+1, err)
			}

			parsed, err := formatting.Parse[pageResponse](resp.Content())
			if err != nil {
				return fmt.Errorf("page %d: parse response: %w", i+1, err)
			}

			applyPageResponse(&cs.Pages[i], parsed)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("%w: %w", ErrClassifyFailed, err)
	}

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
	page.Enhancements = resp.Enhancements
}
