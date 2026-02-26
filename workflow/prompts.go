package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JaimeStill/herald/internal/prompts"
)

// ComposePrompt builds a system prompt by combining tunable instructions,
// immutable specifications, and the running classification state for a given
// workflow stage. When state is nil (first page), the prompt contains only
// instructions and spec.
func ComposePrompt(
	ctx context.Context,
	ps prompts.System,
	stage prompts.Stage,
	state *ClassificationState,
) (string, error) {
	instructions, err := ps.Instructions(ctx, stage)
	if err != nil {
		return "", fmt.Errorf("load instructions for %s: %w", stage, err)
	}

	spec, err := ps.Spec(ctx, stage)
	if err != nil {
		return "", fmt.Errorf("load spec for %s: %w", stage, err)
	}

	var sb strings.Builder
	sb.WriteString(instructions)
	sb.WriteString("\n\n")
	sb.WriteString(spec)

	if state != nil {
		stateJSON, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			return "", fmt.Errorf("serialize classification state: %w", err)
		}

		sb.WriteString("\n\nCurrent classification state:\n\n")
		sb.WriteString(string(stateJSON))
	}

	return sb.String(), nil
}
