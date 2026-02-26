package workflow_test

import (
	"encoding/json"
	"testing"

	"github.com/JaimeStill/herald/workflow"
)

func TestNeedsEnhance(t *testing.T) {
	tests := []struct {
		name  string
		pages []workflow.ClassificationPage
		want  bool
	}{
		{
			"no pages",
			nil,
			false,
		},
		{
			"no enhancement needed",
			[]workflow.ClassificationPage{
				{PageNumber: 1, Enhance: false},
				{PageNumber: 2, Enhance: false},
			},
			false,
		},
		{
			"one page needs enhancement",
			[]workflow.ClassificationPage{
				{PageNumber: 1, Enhance: false},
				{PageNumber: 2, Enhance: true},
			},
			true,
		},
		{
			"all pages need enhancement",
			[]workflow.ClassificationPage{
				{PageNumber: 1, Enhance: true},
				{PageNumber: 2, Enhance: true},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &workflow.ClassificationState{Pages: tt.pages}
			if got := state.NeedsEnhance(); got != tt.want {
				t.Errorf("NeedsEnhance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnhancePages(t *testing.T) {
	tests := []struct {
		name  string
		pages []workflow.ClassificationPage
		want  []int
	}{
		{
			"no pages",
			nil,
			nil,
		},
		{
			"no enhancement needed",
			[]workflow.ClassificationPage{
				{PageNumber: 1, Enhance: false},
				{PageNumber: 2, Enhance: false},
			},
			nil,
		},
		{
			"mixed enhancement",
			[]workflow.ClassificationPage{
				{PageNumber: 1, Enhance: false},
				{PageNumber: 2, Enhance: true},
				{PageNumber: 3, Enhance: false},
				{PageNumber: 4, Enhance: true},
			},
			[]int{1, 3},
		},
		{
			"all pages need enhancement",
			[]workflow.ClassificationPage{
				{PageNumber: 1, Enhance: true},
				{PageNumber: 2, Enhance: true},
			},
			[]int{0, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &workflow.ClassificationState{Pages: tt.pages}
			got := state.EnhancePages()

			if tt.want == nil {
				if got != nil {
					t.Errorf("EnhancePages() = %v, want nil", got)
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Fatalf("EnhancePages() length = %d, want %d", len(got), len(tt.want))
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("EnhancePages()[%d] = %d, want %d", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestClassificationStateJSON(t *testing.T) {
	state := workflow.ClassificationState{
		Classification: "SECRET",
		Confidence:     workflow.ConfidenceHigh,
		Rationale:      "clear markings on all pages",
		Pages: []workflow.ClassificationPage{
			{
				PageNumber:    1,
				ImagePath:     "/tmp/doc/page-1.png",
				MarkingsFound: []string{"SECRET", "NOFORN"},
				Rationale:     "banner and portion markings visible",
				Enhance:       false,
			},
			{
				PageNumber:    2,
				ImagePath:     "/tmp/doc/page-2.png",
				MarkingsFound: []string{"SECRET"},
				Rationale:     "banner partially obscured",
				Enhance:       true,
				Enhancements:  "brightness +20%",
			},
		},
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got workflow.ClassificationState
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.Classification != state.Classification {
		t.Errorf("Classification = %q, want %q", got.Classification, state.Classification)
	}
	if got.Confidence != state.Confidence {
		t.Errorf("Confidence = %q, want %q", got.Confidence, state.Confidence)
	}
	if got.Rationale != state.Rationale {
		t.Errorf("Rationale = %q, want %q", got.Rationale, state.Rationale)
	}
	if len(got.Pages) != 2 {
		t.Fatalf("Pages length = %d, want 2", len(got.Pages))
	}
	if got.Pages[1].Enhance != true {
		t.Errorf("Pages[1].Enhance = %v, want true", got.Pages[1].Enhance)
	}
	if got.Pages[1].Enhancements != "brightness +20%" {
		t.Errorf("Pages[1].Enhancements = %q, want %q", got.Pages[1].Enhancements, "brightness +20%")
	}
}

func TestConfidenceConstants(t *testing.T) {
	tests := []struct {
		name string
		c    workflow.Confidence
		want string
	}{
		{"high", workflow.ConfidenceHigh, "HIGH"},
		{"medium", workflow.ConfidenceMedium, "MEDIUM"},
		{"low", workflow.ConfidenceLow, "LOW"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.c) != tt.want {
				t.Errorf("Confidence = %q, want %q", tt.c, tt.want)
			}
		})
	}
}
