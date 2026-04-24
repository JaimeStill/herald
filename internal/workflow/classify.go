package workflow

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/sync/errgroup"

	"github.com/tailored-agentic-units/format"
	"github.com/tailored-agentic-units/protocol"

	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/internal/state"
	"github.com/JaimeStill/herald/pkg/core"

	taustate "github.com/tailored-agentic-units/orchestrate/state"
)

type pageResponse struct {
	MarkingsFound []string               `json:"markings_found"`
	Rationale     string                 `json:"rationale"`
	Enhance       bool                   `json:"enhance"`
	Enhancements  *state.EnhanceSettings `json:"enhancements,omitempty"`
}

// ClassifyNode returns a state node that performs parallel page-by-page
// analysis using bounded errgroup concurrency. Each goroutine creates its
// own agent, encodes the page image to a data URI, and sends it to the
// vision model. Pages are classified independently (no accumulated context);
// document-level classification synthesis is deferred to the finalize node.
func ClassifyNode(rt *Runtime) taustate.StateNode {
	return taustate.NewFunctionNode(func(ctx context.Context, s taustate.State) (taustate.State, error) {
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

		s = s.Set(state.KeyClassState, *classState)
		return s, nil
	})
}

func extractClassState(s taustate.State) (*state.ClassificationState, error) {
	val, ok := s.Get(state.KeyClassState)
	if !ok {
		return nil, fmt.Errorf("%w: missing %s in state", ErrClassifyFailed, state.KeyClassState)
	}

	cs, ok := val.(state.ClassificationState)
	if !ok {
		return nil, fmt.Errorf("%w: %s is not ClassificationState", ErrClassifyFailed, state.KeyClassState)
	}

	return &cs, nil
}

func classifyPages(ctx context.Context, rt *Runtime, cs *state.ClassificationState) error {
	prompt, err := ComposePrompt(ctx, rt.Prompts, prompts.StageClassify, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrClassifyFailed, err)
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(core.WorkerCount(len(cs.Pages)))

	for i := range cs.Pages {
		g.Go(func() error {
			if gctx.Err() != nil {
				return gctx.Err()
			}

			a, err := rt.NewAgent(gctx)
			if err != nil {
				return fmt.Errorf("page %d: create agent: %w", i+1, err)
			}

			imgData, err := readPageImage(cs.Pages[i].ImagePath)
			if err != nil {
				return fmt.Errorf("page %d: %w", i+1, err)
			}

			resp, err := a.Vision(
				gctx,
				[]protocol.Message{protocol.UserMessage(prompt)},
				[]format.Image{{Data: imgData, Format: "png"}},
			)

			if err != nil {
				return fmt.Errorf("page %d: vision call: %w", i+1, err)
			}

			parsed, err := core.Parse[pageResponse](resp.Text())
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

func readPageImage(imagePath string) ([]byte, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}
	return data, nil
}

func applyPageResponse(page *state.ClassificationPage, resp pageResponse) {
	page.MarkingsFound = resp.MarkingsFound
	page.Rationale = resp.Rationale
	page.Enhancements = resp.Enhancements
}
