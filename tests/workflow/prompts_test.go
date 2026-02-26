package workflow_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/pkg/pagination"
	"github.com/JaimeStill/herald/workflow"
)

type mockPrompts struct {
	instructions map[prompts.Stage]string
	specs        map[prompts.Stage]string
}

func (m *mockPrompts) Handler() *prompts.Handler                          { return nil }
func (m *mockPrompts) List(context.Context, pagination.PageRequest, prompts.Filters) (*pagination.PageResult[prompts.Prompt], error) {
	return nil, nil
}
func (m *mockPrompts) Find(context.Context, uuid.UUID) (*prompts.Prompt, error) { return nil, nil }
func (m *mockPrompts) Create(context.Context, prompts.CreateCommand) (*prompts.Prompt, error) {
	return nil, nil
}
func (m *mockPrompts) Update(context.Context, uuid.UUID, prompts.UpdateCommand) (*prompts.Prompt, error) {
	return nil, nil
}
func (m *mockPrompts) Delete(context.Context, uuid.UUID) error                    { return nil }
func (m *mockPrompts) Activate(context.Context, uuid.UUID) (*prompts.Prompt, error)   { return nil, nil }
func (m *mockPrompts) Deactivate(context.Context, uuid.UUID) (*prompts.Prompt, error) { return nil, nil }

func (m *mockPrompts) Instructions(_ context.Context, stage prompts.Stage) (string, error) {
	text, ok := m.instructions[stage]
	if !ok {
		return "", prompts.ErrInvalidStage
	}
	return text, nil
}

func (m *mockPrompts) Spec(_ context.Context, stage prompts.Stage) (string, error) {
	text, ok := m.specs[stage]
	if !ok {
		return "", prompts.ErrInvalidStage
	}
	return text, nil
}

func newMockPrompts() *mockPrompts {
	return &mockPrompts{
		instructions: map[prompts.Stage]string{
			prompts.StageClassify: "classify instructions",
			prompts.StageEnhance:  "enhance instructions",
		},
		specs: map[prompts.Stage]string{
			prompts.StageClassify: "classify spec",
			prompts.StageEnhance:  "enhance spec",
		},
	}
}

func TestComposePrompt(t *testing.T) {
	ctx := context.Background()
	mock := newMockPrompts()

	t.Run("nil state produces instructions and spec", func(t *testing.T) {
		got, err := workflow.ComposePrompt(ctx, mock, prompts.StageClassify, nil)
		if err != nil {
			t.Fatalf("ComposePrompt error: %v", err)
		}

		if !strings.Contains(got, "classify instructions") {
			t.Error("missing instructions in prompt")
		}
		if !strings.Contains(got, "classify spec") {
			t.Error("missing spec in prompt")
		}
		if strings.Contains(got, "Current classification state") {
			t.Error("nil state should not include state section")
		}
	})

	t.Run("with state includes serialized state", func(t *testing.T) {
		state := &workflow.ClassificationState{
			Classification: "SECRET",
			Confidence:     workflow.ConfidenceHigh,
			Rationale:      "clear markings",
			Pages: []workflow.ClassificationPage{
				{
					PageNumber:    1,
					MarkingsFound: []string{"SECRET"},
					Rationale:     "banner visible",
				},
			},
		}

		got, err := workflow.ComposePrompt(ctx, mock, prompts.StageClassify, state)
		if err != nil {
			t.Fatalf("ComposePrompt error: %v", err)
		}

		if !strings.Contains(got, "classify instructions") {
			t.Error("missing instructions in prompt")
		}
		if !strings.Contains(got, "classify spec") {
			t.Error("missing spec in prompt")
		}
		if !strings.Contains(got, "Current classification state") {
			t.Error("missing state header in prompt")
		}
		if !strings.Contains(got, "SECRET") {
			t.Error("missing classification value in serialized state")
		}
	})

	t.Run("enhance stage uses enhance instructions and spec", func(t *testing.T) {
		got, err := workflow.ComposePrompt(ctx, mock, prompts.StageEnhance, nil)
		if err != nil {
			t.Fatalf("ComposePrompt error: %v", err)
		}

		if !strings.Contains(got, "enhance instructions") {
			t.Error("missing enhance instructions in prompt")
		}
		if !strings.Contains(got, "enhance spec") {
			t.Error("missing enhance spec in prompt")
		}
	})

	t.Run("invalid stage returns error", func(t *testing.T) {
		_, err := workflow.ComposePrompt(ctx, mock, "banana", nil)
		if err == nil {
			t.Error("expected error for invalid stage")
		}
	})

	t.Run("prompt structure is instructions then spec then state", func(t *testing.T) {
		state := &workflow.ClassificationState{
			Classification: "UNCLASSIFIED",
			Confidence:     workflow.ConfidenceLow,
			Rationale:      "no markings found",
		}

		got, err := workflow.ComposePrompt(ctx, mock, prompts.StageClassify, state)
		if err != nil {
			t.Fatalf("ComposePrompt error: %v", err)
		}

		instrIdx := strings.Index(got, "classify instructions")
		specIdx := strings.Index(got, "classify spec")
		stateIdx := strings.Index(got, "Current classification state")

		if instrIdx >= specIdx {
			t.Error("instructions should appear before spec")
		}
		if specIdx >= stateIdx {
			t.Error("spec should appear before state")
		}
	})
}
