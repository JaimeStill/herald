package workflow

import (
	"context"
	"fmt"

	"github.com/JaimeStill/go-agents/pkg/agent"

	"github.com/JaimeStill/go-agents-orchestration/pkg/state"

	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/pkg/formatting"
)

type finalizeResponse struct {
	Classification string     `json:"classification"`
	Confidence     Confidence `json:"confidence"`
	Rationale      string     `json:"rationale"`
}

// FinalizeNode returns a state node that synthesizes the document-level
// classification from all per-page findings. It performs a single Chat
// inference (not Vision — no images needed) that reviews all page data
// and produces the authoritative classification, confidence, and rationale.
func FinalizeNode(rt *Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		cs, err := extractClassState(s)
		if err != nil {
			return s, fmt.Errorf("finalize: %w", err)
		}

		if err := synthesize(ctx, rt, cs); err != nil {
			return s, fmt.Errorf("finalize: %w", err)
		}

		rt.Logger.InfoContext(
			ctx, "finalize node complete",
			"classification", cs.Classification,
			"confidence", cs.Confidence,
		)

		s = s.Set(KeyClassState, *cs)
		return s, nil
	})
}

func synthesize(ctx context.Context, rt *Runtime, cs *ClassificationState) error {
	a, err := agent.New(&rt.Agent)
	if err != nil {
		return fmt.Errorf("%w: create agent: %w", ErrFinalizeFailed, err)
	}

	prompt, err := ComposePrompt(ctx, rt.Prompts, prompts.StageFinalize, cs)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFinalizeFailed, err)
	}

	resp, err := a.Chat(ctx, prompt)
	if err != nil {
		return fmt.Errorf("%w: chat call: %w", ErrFinalizeFailed, err)
	}

	parsed, err := formatting.Parse[finalizeResponse](resp.Content())
	if err != nil {
		return fmt.Errorf("%w: parse response: %w", ErrFinalizeFailed, err)
	}

	cs.Classification = parsed.Classification
	cs.Confidence = parsed.Confidence
	cs.Rationale = parsed.Rationale

	return nil
}
