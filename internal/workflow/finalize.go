package workflow

import (
	"context"
	"fmt"

	"github.com/tailored-agentic-units/protocol"

	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/internal/state"
	"github.com/JaimeStill/herald/pkg/core"

	taustate "github.com/tailored-agentic-units/orchestrate/state"
)

type finalizeResponse struct {
	Classification string           `json:"classification"`
	Confidence     state.Confidence `json:"confidence"`
	Rationale      string           `json:"rationale"`
}

// FinalizeNode returns a state node that synthesizes the document-level
// classification from all per-page findings. It performs a single Chat
// inference (not Vision — no images needed) that reviews all page data
// and produces the authoritative classification, confidence, and rationale.
func FinalizeNode(rt *Runtime) taustate.StateNode {
	return taustate.NewFunctionNode(func(ctx context.Context, s taustate.State) (taustate.State, error) {
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

		s = s.Set(state.KeyClassState, *cs)
		return s, nil
	})
}

func synthesize(ctx context.Context, rt *Runtime, cs *state.ClassificationState) error {
	a, err := rt.NewAgent(ctx)
	if err != nil {
		return fmt.Errorf("%w: create agent: %w", ErrFinalizeFailed, err)
	}

	prompt, err := ComposePrompt(ctx, rt.Prompts, prompts.StageFinalize, cs)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFinalizeFailed, err)
	}

	resp, err := a.Chat(ctx, []protocol.Message{protocol.UserMessage(prompt)})
	if err != nil {
		return fmt.Errorf("%w: chat call: %w", ErrFinalizeFailed, err)
	}

	parsed, err := core.Parse[finalizeResponse](resp.Text())
	if err != nil {
		return fmt.Errorf("%w: parse response: %w", ErrFinalizeFailed, err)
	}

	cs.Classification = parsed.Classification
	cs.Confidence = parsed.Confidence
	cs.Rationale = parsed.Rationale

	return nil
}
