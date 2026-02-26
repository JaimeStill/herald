package workflow_test

import (
	"encoding/json"
	"testing"

	"github.com/JaimeStill/herald/internal/workflow"
)

func intPtr(v int) *int { return &v }

func TestEnhance(t *testing.T) {
	tests := []struct {
		name string
		page workflow.ClassificationPage
		want bool
	}{
		{
			"nil enhancements",
			workflow.ClassificationPage{PageNumber: 1},
			false,
		},
		{
			"with enhancements",
			workflow.ClassificationPage{
				PageNumber:   1,
				Enhancements: &workflow.EnhanceSettings{Brightness: intPtr(130)},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.page.Enhance(); got != tt.want {
				t.Errorf("Enhance() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
				{PageNumber: 1},
				{PageNumber: 2},
			},
			false,
		},
		{
			"one page needs enhancement",
			[]workflow.ClassificationPage{
				{PageNumber: 1},
				{PageNumber: 2, Enhancements: &workflow.EnhanceSettings{Contrast: intPtr(30)}},
			},
			true,
		},
		{
			"all pages need enhancement",
			[]workflow.ClassificationPage{
				{PageNumber: 1, Enhancements: &workflow.EnhanceSettings{Brightness: intPtr(120)}},
				{PageNumber: 2, Enhancements: &workflow.EnhanceSettings{Saturation: intPtr(80)}},
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
				{PageNumber: 1},
				{PageNumber: 2},
			},
			nil,
		},
		{
			"mixed enhancement",
			[]workflow.ClassificationPage{
				{PageNumber: 1},
				{PageNumber: 2, Enhancements: &workflow.EnhanceSettings{Brightness: intPtr(130)}},
				{PageNumber: 3},
				{PageNumber: 4, Enhancements: &workflow.EnhanceSettings{Contrast: intPtr(20)}},
			},
			[]int{1, 3},
		},
		{
			"all pages need enhancement",
			[]workflow.ClassificationPage{
				{PageNumber: 1, Enhancements: &workflow.EnhanceSettings{Brightness: intPtr(120)}},
				{PageNumber: 2, Enhancements: &workflow.EnhanceSettings{Saturation: intPtr(80)}},
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
			},
			{
				PageNumber:    2,
				ImagePath:     "/tmp/doc/page-2.png",
				MarkingsFound: []string{"SECRET"},
				Rationale:     "banner partially obscured",
				Enhancements:  &workflow.EnhanceSettings{Brightness: intPtr(120), Contrast: intPtr(20)},
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
	if got.Pages[0].Enhancements != nil {
		t.Error("Pages[0].Enhancements should be nil")
	}
	if got.Pages[1].Enhancements == nil {
		t.Fatal("Pages[1].Enhancements should not be nil")
	}
	if *got.Pages[1].Enhancements.Brightness != 120 {
		t.Errorf("Pages[1].Enhancements.Brightness = %d, want 120", *got.Pages[1].Enhancements.Brightness)
	}
	if *got.Pages[1].Enhancements.Contrast != 20 {
		t.Errorf("Pages[1].Enhancements.Contrast = %d, want 20", *got.Pages[1].Enhancements.Contrast)
	}
	if got.Pages[1].Enhancements.Saturation != nil {
		t.Error("Pages[1].Enhancements.Saturation should be nil (omitted)")
	}
}

func TestEnhanceSettingsJSON(t *testing.T) {
	t.Run("null round-trips as nil", func(t *testing.T) {
		input := `{"page_number":1,"image_path":"/tmp/p.png","markings_found":null,"rationale":""}`
		var page workflow.ClassificationPage
		if err := json.Unmarshal([]byte(input), &page); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if page.Enhancements != nil {
			t.Error("expected nil Enhancements")
		}
	})

	t.Run("partial fields round-trip", func(t *testing.T) {
		settings := &workflow.EnhanceSettings{Brightness: intPtr(140)}
		data, err := json.Marshal(settings)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var got workflow.EnhanceSettings
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		if got.Brightness == nil || *got.Brightness != 140 {
			t.Errorf("Brightness = %v, want 140", got.Brightness)
		}
		if got.Contrast != nil {
			t.Errorf("Contrast = %v, want nil", got.Contrast)
		}
		if got.Saturation != nil {
			t.Errorf("Saturation = %v, want nil", got.Saturation)
		}
	})
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
