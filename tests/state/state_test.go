package state_test

import (
	"encoding/json"
	"testing"

	"github.com/JaimeStill/herald/internal/state"
)

func intPtr(v int) *int { return &v }

func TestEnhance(t *testing.T) {
	tests := []struct {
		name string
		page state.ClassificationPage
		want bool
	}{
		{
			"nil enhancements",
			state.ClassificationPage{PageNumber: 1},
			false,
		},
		{
			"with enhancements",
			state.ClassificationPage{
				PageNumber:   1,
				Enhancements: &state.EnhanceSettings{Brightness: intPtr(130)},
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
		pages []state.ClassificationPage
		want  bool
	}{
		{
			"no pages",
			nil,
			false,
		},
		{
			"no enhancement needed",
			[]state.ClassificationPage{
				{PageNumber: 1},
				{PageNumber: 2},
			},
			false,
		},
		{
			"one page needs enhancement",
			[]state.ClassificationPage{
				{PageNumber: 1},
				{PageNumber: 2, Enhancements: &state.EnhanceSettings{Contrast: intPtr(30)}},
			},
			true,
		},
		{
			"all pages need enhancement",
			[]state.ClassificationPage{
				{PageNumber: 1, Enhancements: &state.EnhanceSettings{Brightness: intPtr(120)}},
				{PageNumber: 2, Enhancements: &state.EnhanceSettings{Saturation: intPtr(80)}},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &state.ClassificationState{Pages: tt.pages}
			if got := cs.NeedsEnhance(); got != tt.want {
				t.Errorf("NeedsEnhance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnhancePages(t *testing.T) {
	tests := []struct {
		name  string
		pages []state.ClassificationPage
		want  []int
	}{
		{
			"no pages",
			nil,
			nil,
		},
		{
			"no enhancement needed",
			[]state.ClassificationPage{
				{PageNumber: 1},
				{PageNumber: 2},
			},
			nil,
		},
		{
			"mixed enhancement",
			[]state.ClassificationPage{
				{PageNumber: 1},
				{PageNumber: 2, Enhancements: &state.EnhanceSettings{Brightness: intPtr(130)}},
				{PageNumber: 3},
				{PageNumber: 4, Enhancements: &state.EnhanceSettings{Contrast: intPtr(20)}},
			},
			[]int{1, 3},
		},
		{
			"all pages need enhancement",
			[]state.ClassificationPage{
				{PageNumber: 1, Enhancements: &state.EnhanceSettings{Brightness: intPtr(120)}},
				{PageNumber: 2, Enhancements: &state.EnhanceSettings{Saturation: intPtr(80)}},
			},
			[]int{0, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &state.ClassificationState{Pages: tt.pages}
			got := cs.EnhancePages()

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

func TestAggregateConfidence(t *testing.T) {
	tests := []struct {
		name  string
		pages []state.ClassificationPage
		want  state.Confidence
	}{
		{
			"nil pages",
			nil,
			state.ConfidenceHigh,
		},
		{
			"all blank pages ignored",
			[]state.ClassificationPage{
				{PageNumber: 1},
				{PageNumber: 2},
			},
			state.ConfidenceHigh,
		},
		{
			"all content pages high",
			[]state.ClassificationPage{
				{PageNumber: 1, MarkingsFound: []string{"SECRET"}, Confidence: state.ConfidenceHigh},
				{PageNumber: 2, MarkingsFound: []string{"SECRET", "NOFORN"}, Confidence: state.ConfidenceHigh},
			},
			state.ConfidenceHigh,
		},
		{
			"one medium lowers to medium",
			[]state.ClassificationPage{
				{PageNumber: 1, MarkingsFound: []string{"SECRET"}, Confidence: state.ConfidenceHigh},
				{PageNumber: 2, MarkingsFound: []string{"CONFIDENTIAL"}, Confidence: state.ConfidenceMedium},
			},
			state.ConfidenceMedium,
		},
		{
			"undetermined markings floor to low despite high confidence",
			[]state.ClassificationPage{
				{PageNumber: 1, MarkingsFound: []string{"SECRET"}, Confidence: state.ConfidenceHigh},
				{
					PageNumber:           2,
					MarkingsFound:        []string{"SECRET"},
					UndeterminedMarkings: []string{"faded bottom stamp"},
					Confidence:           state.ConfidenceHigh,
				},
			},
			state.ConfidenceLow,
		},
		{
			"undetermined-only page is content-bearing and low",
			[]state.ClassificationPage{
				{PageNumber: 1, UndeterminedMarkings: []string{"illegible banner"}},
			},
			state.ConfidenceLow,
		},
		{
			"content page missing confidence defaults to low",
			[]state.ClassificationPage{
				{PageNumber: 1, MarkingsFound: []string{"SECRET"}, Confidence: state.ConfidenceHigh},
				{PageNumber: 2, MarkingsFound: []string{"SECRET"}},
			},
			state.ConfidenceLow,
		},
		{
			"blank pages do not lower a high content page",
			[]state.ClassificationPage{
				{PageNumber: 1},
				{PageNumber: 2, MarkingsFound: []string{"UNCLASSIFIED"}, Confidence: state.ConfidenceHigh},
				{PageNumber: 3},
			},
			state.ConfidenceHigh,
		},
		{
			"lowest of mixed levels wins",
			[]state.ClassificationPage{
				{PageNumber: 1, MarkingsFound: []string{"SECRET"}, Confidence: state.ConfidenceHigh},
				{PageNumber: 2, MarkingsFound: []string{"SECRET"}, Confidence: state.ConfidenceMedium},
				{PageNumber: 3, MarkingsFound: []string{"SECRET"}, Confidence: state.ConfidenceLow},
			},
			state.ConfidenceLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &state.ClassificationState{Pages: tt.pages}
			got := cs.AggregateConfidence()
			if got != tt.want {
				t.Errorf("AggregateConfidence() = %q, want %q", got, tt.want)
			}
			if got != state.ConfidenceHigh && got != state.ConfidenceMedium && got != state.ConfidenceLow {
				t.Errorf("AggregateConfidence() = %q, must be HIGH, MEDIUM, or LOW (never empty)", got)
			}
		})
	}
}

func TestClassificationStateJSON(t *testing.T) {
	cs := state.ClassificationState{
		Classification: "SECRET",
		Confidence:     state.ConfidenceHigh,
		Rationale:      "clear markings on all pages",
		Pages: []state.ClassificationPage{
			{
				PageNumber:    1,
				ImagePath:     "/tmp/doc/page-1.png",
				MarkingsFound: []string{"SECRET", "NOFORN"},
				Confidence:    state.ConfidenceHigh,
				Rationale:     "banner and portion markings visible",
			},
			{
				PageNumber:           2,
				ImagePath:            "/tmp/doc/page-2.png",
				MarkingsFound:        []string{"SECRET"},
				UndeterminedMarkings: []string{"faded bottom stamp"},
				Confidence:           state.ConfidenceLow,
				Rationale:            "banner partially obscured",
				Enhancements:         &state.EnhanceSettings{Brightness: intPtr(120), Contrast: intPtr(20)},
			},
		},
	}

	data, err := json.Marshal(cs)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got state.ClassificationState
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.Classification != cs.Classification {
		t.Errorf("Classification = %q, want %q", got.Classification, cs.Classification)
	}
	if got.Confidence != cs.Confidence {
		t.Errorf("Confidence = %q, want %q", got.Confidence, cs.Confidence)
	}
	if got.Rationale != cs.Rationale {
		t.Errorf("Rationale = %q, want %q", got.Rationale, cs.Rationale)
	}
	if len(got.Pages) != 2 {
		t.Fatalf("Pages length = %d, want 2", len(got.Pages))
	}
	if got.Pages[0].Confidence != state.ConfidenceHigh {
		t.Errorf("Pages[0].Confidence = %q, want HIGH", got.Pages[0].Confidence)
	}
	if len(got.Pages[0].UndeterminedMarkings) != 0 {
		t.Errorf("Pages[0].UndeterminedMarkings = %v, want empty (omitted)", got.Pages[0].UndeterminedMarkings)
	}
	if got.Pages[1].Confidence != state.ConfidenceLow {
		t.Errorf("Pages[1].Confidence = %q, want LOW", got.Pages[1].Confidence)
	}
	if len(got.Pages[1].UndeterminedMarkings) != 1 || got.Pages[1].UndeterminedMarkings[0] != "faded bottom stamp" {
		t.Errorf("Pages[1].UndeterminedMarkings = %v, want [faded bottom stamp]", got.Pages[1].UndeterminedMarkings)
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
		var page state.ClassificationPage
		if err := json.Unmarshal([]byte(input), &page); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if page.Enhancements != nil {
			t.Error("expected nil Enhancements")
		}
	})

	t.Run("partial fields round-trip", func(t *testing.T) {
		settings := &state.EnhanceSettings{Brightness: intPtr(140)}
		data, err := json.Marshal(settings)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		var got state.EnhanceSettings
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
		c    state.Confidence
		want string
	}{
		{"high", state.ConfidenceHigh, "HIGH"},
		{"medium", state.ConfidenceMedium, "MEDIUM"},
		{"low", state.ConfidenceLow, "LOW"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.c) != tt.want {
				t.Errorf("Confidence = %q, want %q", tt.c, tt.want)
			}
		})
	}
}
