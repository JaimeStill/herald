package workflow

import (
	"context"
	"fmt"
	"slices"

	"golang.org/x/sync/errgroup"

	"github.com/google/uuid"
	"github.com/tailored-agentic-units/protocol"

	"github.com/JaimeStill/herald/internal/format"
	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/internal/state"
	"github.com/JaimeStill/herald/pkg/core"

	tauformat "github.com/tailored-agentic-units/format"
	taustate "github.com/tailored-agentic-units/orchestrate/state"
)

type enhanceResponse struct {
	ResolvedMarkings     []string         `json:"resolved_markings"`
	UndeterminedMarkings []string         `json:"undetermined_markings"`
	Confidence           state.Confidence `json:"confidence"`
	Rationale            string           `json:"rationale"`
}

// EnhanceNode returns a state node that re-renders flagged pages with adjusted
// ImageMagick settings and reclassifies them via vision. For each page with
// non-nil Enhancements, it re-renders from the source PDF using the specified
// brightness/contrast/saturation adjustments, sends the enhanced image to the
// vision model, and updates the per-page findings. Clears Enhancements after
// processing so the page is no longer flagged.
func EnhanceNode(rt *Runtime) taustate.StateNode {
	return taustate.NewFunctionNode(func(
		ctx context.Context,
		s taustate.State,
	) (taustate.State, error) {
		cs, err := extractClassState(s)
		if err != nil {
			return s, fmt.Errorf("enhance: %w", err)
		}

		tempDir, err := extractTempDir(s)
		if err != nil {
			return s, fmt.Errorf("enhance: %w", err)
		}

		documentID, err := extractDocumentID(s)
		if err != nil {
			return s, fmt.Errorf("enhance: %w", err)
		}

		doc, err := rt.Documents.Find(ctx, documentID)
		if err != nil {
			return s, fmt.Errorf("enhance: %w: %w", ErrEnhanceFailed, err)
		}

		handler, err := rt.Formats.Lookup(doc.ContentType)
		if err != nil {
			return s, fmt.Errorf("enhance: %w: %w", ErrEnhanceFailed, err)
		}

		enhanced := cs.EnhancePages()

		if err := enhancePages(ctx, rt, handler, cs, tempDir); err != nil {
			return s, fmt.Errorf("enhance: %w", err)
		}

		rt.Logger.InfoContext(
			ctx, "enhance node complete",
			"pages_enhanced", len(enhanced),
		)

		s = s.Set(state.KeyClassState, *cs)
		return s, nil
	})
}

func enhancePages(
	ctx context.Context,
	rt *Runtime,
	handler format.Handler,
	cs *state.ClassificationState,
	tempDir string,
) error {
	enhanced := cs.EnhancePages()

	prompt, err := ComposePrompt(ctx, rt.Prompts, prompts.StageEnhance, cs)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrEnhanceFailed, err)
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(core.WorkerCount(len(enhanced)))

	for _, i := range enhanced {
		g.Go(func() error {
			if gctx.Err() != nil {
				return gctx.Err()
			}

			a, err := rt.NewAgent(gctx)
			if err != nil {
				return fmt.Errorf("page %d: create agent: %w", cs.Pages[i].PageNumber, err)
			}

			imgPath, err := handler.Enhance(
				gctx,
				tempDir,
				&cs.Pages[i],
				cs.Pages[i].Enhancements,
			)
			if err != nil {
				return fmt.Errorf("page %d: %w", cs.Pages[i].PageNumber, err)
			}
			cs.Pages[i].ImagePath = imgPath

			imgData, err := readPageImage(imgPath)
			if err != nil {
				return fmt.Errorf("page %d: %w", cs.Pages[i].PageNumber, err)
			}

			resp, err := a.Vision(
				gctx,
				[]protocol.Message{protocol.UserMessage(prompt)},
				[]tauformat.Image{{Data: imgData, Format: "png"}},
			)
			if err != nil {
				return fmt.Errorf("page %d: vision call: %w", cs.Pages[i].PageNumber, err)
			}

			parsed, err := core.Parse[enhanceResponse](resp.Text())
			if err != nil {
				return fmt.Errorf("page %d: parse response: %w", cs.Pages[i].PageNumber, err)
			}

			p := &cs.Pages[i]
			p.MarkingsFound = unionMarkings(p.MarkingsFound, parsed.ResolvedMarkings)
			p.UndeterminedMarkings = parsed.UndeterminedMarkings
			p.Confidence = parsed.Confidence
			p.Rationale = parsed.Rationale
			p.Enhancements = nil

			inputTokens, outputTokens := 0, 0
			if resp.Usage != nil {
				inputTokens, outputTokens = resp.Usage.InputTokens, resp.Usage.OutputTokens
			}

			rt.Logger.DebugContext(
				gctx, "enhance page complete",
				"page", p.PageNumber,
				"confidence", p.Confidence,
				"resolved_markings", parsed.ResolvedMarkings,
				"markings_found", p.MarkingsFound,
				"undetermined_markings", p.UndeterminedMarkings,
				"input_tokens", inputTokens,
				"output_tokens", outputTokens,
			)

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("%w: %w", ErrEnhanceFailed, err)
	}

	return nil
}

// unionMarkings combines a page's prior markings with the markings an
// enhancement pass newly resolved, preserving the prior-confident set verbatim
// and adding the recovered ones. The result is sorted and de-duplicated. This is
// the resolution-aware merge that replaced the previous wholesale overwrite: a
// banner the first pass read clearly survives even when the enhanced re-render —
// adjusted to surface a faint region — obscures it.
func unionMarkings(prior, resolved []string) []string {
	merged := make([]string, 0, len(prior)+len(resolved))
	merged = append(merged, prior...)
	merged = append(merged, resolved...)
	slices.Sort(merged)
	return slices.Compact(merged)
}

func extractDocumentID(s taustate.State) (uuid.UUID, error) {
	val, ok := s.Get(state.KeyDocumentID)
	if !ok {
		return uuid.Nil, fmt.Errorf("%w: missing %s in state", ErrEnhanceFailed, state.KeyDocumentID)
	}
	id, ok := val.(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("%w: %s is not uuid.UUID", ErrEnhanceFailed, state.KeyDocumentID)
	}
	return id, nil
}

func extractTempDir(s taustate.State) (string, error) {
	val, ok := s.Get(state.KeyTempDir)
	if !ok {
		return "", fmt.Errorf("%w: missing %s in state", ErrEnhanceFailed, state.KeyTempDir)
	}

	tempDir, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("%w: %s is not string", ErrEnhanceFailed, state.KeyTempDir)
	}

	return tempDir, nil
}
