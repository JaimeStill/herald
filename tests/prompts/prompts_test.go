package prompts_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/pkg/query"
)

func ptr[T any](v T) *T { return &v }

func TestMapHTTPStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"not found", prompts.ErrNotFound, http.StatusNotFound},
		{"duplicate", prompts.ErrDuplicate, http.StatusConflict},
		{"invalid stage", prompts.ErrInvalidStage, http.StatusBadRequest},
		{"unknown error", errors.New("something else"), http.StatusInternalServerError},
		{"wrapped not found", fmt.Errorf("find failed: %w", prompts.ErrNotFound), http.StatusNotFound},
		{"wrapped duplicate", fmt.Errorf("insert failed: %w", prompts.ErrDuplicate), http.StatusConflict},
		{"wrapped invalid stage", fmt.Errorf("decode failed: %w", prompts.ErrInvalidStage), http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prompts.MapHTTPStatus(tt.err)
			if got != tt.want {
				t.Errorf("MapHTTPStatus(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestStages(t *testing.T) {
	stages := prompts.Stages()

	if len(stages) != 2 {
		t.Fatalf("len(Stages()) = %d, want 2", len(stages))
	}

	want := []prompts.Stage{prompts.StageClassify, prompts.StageEnhance}
	for i, s := range stages {
		if s != want[i] {
			t.Errorf("Stages()[%d] = %q, want %q", i, s, want[i])
		}
	}
}

func TestStageUnmarshalJSON(t *testing.T) {
	t.Run("valid stages", func(t *testing.T) {
		tests := []struct {
			input string
			want  prompts.Stage
		}{
			{`"classify"`, prompts.StageClassify},
			{`"enhance"`, prompts.StageEnhance},
		}

		for _, tt := range tests {
			t.Run(string(tt.want), func(t *testing.T) {
				var s prompts.Stage
				if err := json.Unmarshal([]byte(tt.input), &s); err != nil {
					t.Fatalf("Unmarshal(%s) error: %v", tt.input, err)
				}
				if s != tt.want {
					t.Errorf("Unmarshal(%s) = %q, want %q", tt.input, s, tt.want)
				}
			})
		}
	})

	t.Run("init is invalid", func(t *testing.T) {
		var s prompts.Stage
		err := json.Unmarshal([]byte(`"init"`), &s)
		if !errors.Is(err, prompts.ErrInvalidStage) {
			t.Errorf("Unmarshal(init) error = %v, want ErrInvalidStage", err)
		}
	})

	t.Run("invalid stage returns error", func(t *testing.T) {
		var s prompts.Stage
		err := json.Unmarshal([]byte(`"banana"`), &s)
		if !errors.Is(err, prompts.ErrInvalidStage) {
			t.Errorf("Unmarshal(banana) error = %v, want ErrInvalidStage", err)
		}
	})

	t.Run("empty string returns error", func(t *testing.T) {
		var s prompts.Stage
		err := json.Unmarshal([]byte(`""`), &s)
		if !errors.Is(err, prompts.ErrInvalidStage) {
			t.Errorf("Unmarshal('') error = %v, want ErrInvalidStage", err)
		}
	})

	t.Run("non-string returns error", func(t *testing.T) {
		var s prompts.Stage
		err := json.Unmarshal([]byte(`42`), &s)
		if err == nil {
			t.Error("Unmarshal(42) should return error")
		}
	})

	t.Run("struct with stage field", func(t *testing.T) {
		type payload struct {
			Stage prompts.Stage `json:"stage"`
		}

		var p payload
		if err := json.Unmarshal([]byte(`{"stage":"classify"}`), &p); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if p.Stage != prompts.StageClassify {
			t.Errorf("Stage = %q, want classify", p.Stage)
		}
	})

	t.Run("struct with invalid stage field", func(t *testing.T) {
		type payload struct {
			Stage prompts.Stage `json:"stage"`
		}

		var p payload
		err := json.Unmarshal([]byte(`{"stage":"invalid"}`), &p)
		if !errors.Is(err, prompts.ErrInvalidStage) {
			t.Errorf("Unmarshal error = %v, want ErrInvalidStage", err)
		}
	})
}

func TestParseStage(t *testing.T) {
	t.Run("valid stages", func(t *testing.T) {
		tests := []struct {
			input string
			want  prompts.Stage
		}{
			{"classify", prompts.StageClassify},
			{"enhance", prompts.StageEnhance},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				got, err := prompts.ParseStage(tt.input)
				if err != nil {
					t.Fatalf("ParseStage(%q) error: %v", tt.input, err)
				}
				if got != tt.want {
					t.Errorf("ParseStage(%q) = %q, want %q", tt.input, got, tt.want)
				}
			})
		}
	})

	t.Run("init is invalid", func(t *testing.T) {
		_, err := prompts.ParseStage("init")
		if !errors.Is(err, prompts.ErrInvalidStage) {
			t.Errorf("ParseStage(init) error = %v, want ErrInvalidStage", err)
		}
	})

	t.Run("unknown stage returns error", func(t *testing.T) {
		_, err := prompts.ParseStage("banana")
		if !errors.Is(err, prompts.ErrInvalidStage) {
			t.Errorf("ParseStage(banana) error = %v, want ErrInvalidStage", err)
		}
	})

	t.Run("empty string returns error", func(t *testing.T) {
		_, err := prompts.ParseStage("")
		if !errors.Is(err, prompts.ErrInvalidStage) {
			t.Errorf("ParseStage('') error = %v, want ErrInvalidStage", err)
		}
	})
}

func TestInstructions(t *testing.T) {
	t.Run("returns content for valid stages", func(t *testing.T) {
		for _, stage := range prompts.Stages() {
			t.Run(string(stage), func(t *testing.T) {
				text, err := prompts.Instructions(stage)
				if err != nil {
					t.Fatalf("Instructions(%q) error: %v", stage, err)
				}
				if text == "" {
					t.Errorf("Instructions(%q) returned empty string", stage)
				}
			})
		}
	})

	t.Run("invalid stage returns error", func(t *testing.T) {
		_, err := prompts.Instructions("banana")
		if !errors.Is(err, prompts.ErrInvalidStage) {
			t.Errorf("Instructions(banana) error = %v, want ErrInvalidStage", err)
		}
	})
}

func TestSpec(t *testing.T) {
	t.Run("returns content for valid stages", func(t *testing.T) {
		for _, stage := range prompts.Stages() {
			t.Run(string(stage), func(t *testing.T) {
				text, err := prompts.Spec(stage)
				if err != nil {
					t.Fatalf("Spec(%q) error: %v", stage, err)
				}
				if text == "" {
					t.Errorf("Spec(%q) returned empty string", stage)
				}
			})
		}
	})

	t.Run("invalid stage returns error", func(t *testing.T) {
		_, err := prompts.Spec("banana")
		if !errors.Is(err, prompts.ErrInvalidStage) {
			t.Errorf("Spec(banana) error = %v, want ErrInvalidStage", err)
		}
	})
}

func TestFiltersFromQuery(t *testing.T) {
	t.Run("all params present", func(t *testing.T) {
		values := url.Values{
			"stage":  {"classify"},
			"name":   {"detailed"},
			"active": {"true"},
		}

		f := prompts.FiltersFromQuery(values)

		if f.Stage == nil || *f.Stage != prompts.StageClassify {
			t.Errorf("Stage = %v, want classify", f.Stage)
		}
		if f.Name == nil || *f.Name != "detailed" {
			t.Errorf("Name = %v, want detailed", f.Name)
		}
		if f.Active == nil || *f.Active != true {
			t.Errorf("Active = %v, want true", f.Active)
		}
	})

	t.Run("empty params yield nil fields", func(t *testing.T) {
		f := prompts.FiltersFromQuery(url.Values{})

		if f.Stage != nil {
			t.Errorf("Stage = %v, want nil", f.Stage)
		}
		if f.Name != nil {
			t.Errorf("Name = %v, want nil", f.Name)
		}
		if f.Active != nil {
			t.Errorf("Active = %v, want nil", f.Active)
		}
	})

	t.Run("invalid active ignored", func(t *testing.T) {
		values := url.Values{"active": {"not-a-bool"}}
		f := prompts.FiltersFromQuery(values)

		if f.Active != nil {
			t.Errorf("Active = %v, want nil for invalid input", f.Active)
		}
	})

	t.Run("active false", func(t *testing.T) {
		values := url.Values{"active": {"false"}}
		f := prompts.FiltersFromQuery(values)

		if f.Active == nil || *f.Active != false {
			t.Errorf("Active = %v, want false", f.Active)
		}
	})

	t.Run("partial params", func(t *testing.T) {
		values := url.Values{
			"stage": {"enhance"},
			"name":  {"verbose"},
		}

		f := prompts.FiltersFromQuery(values)

		if f.Stage == nil || *f.Stage != prompts.StageEnhance {
			t.Errorf("Stage = %v, want enhance", f.Stage)
		}
		if f.Name == nil || *f.Name != "verbose" {
			t.Errorf("Name = %v, want verbose", f.Name)
		}
		if f.Active != nil {
			t.Errorf("Active = %v, want nil", f.Active)
		}
	})
}

func TestFiltersApply(t *testing.T) {
	projection := query.
		NewProjectionMap("public", "prompts", "p").
		Project("stage", "Stage").
		Project("name", "Name").
		Project("active", "Active")

	t.Run("no filters produces no WHERE clause", func(t *testing.T) {
		b := query.NewBuilder(projection)
		f := prompts.Filters{}
		f.Apply(b)
		sql, args := b.Build()

		wantSQL := "SELECT p.stage, p.name, p.active FROM public.prompts p"
		if sql != wantSQL {
			t.Errorf("sql = %q, want %q", sql, wantSQL)
		}
		if len(args) != 0 {
			t.Errorf("args = %v, want empty", args)
		}
	})

	t.Run("stage equals filter", func(t *testing.T) {
		b := query.NewBuilder(projection)
		stage := prompts.StageClassify
		f := prompts.Filters{Stage: &stage}
		f.Apply(b)
		_, args := b.Build()

		if len(args) != 1 {
			t.Fatalf("args length = %d, want 1", len(args))
		}
	})

	t.Run("name contains filter", func(t *testing.T) {
		b := query.NewBuilder(projection)
		f := prompts.Filters{Name: ptr("detailed")}
		f.Apply(b)
		_, args := b.Build()

		if len(args) != 1 || args[0] != "%detailed%" {
			t.Errorf("args = %v, want [%%detailed%%]", args)
		}
	})

	t.Run("active equals filter", func(t *testing.T) {
		b := query.NewBuilder(projection)
		f := prompts.Filters{Active: ptr(true)}
		f.Apply(b)
		_, args := b.Build()

		if len(args) != 1 {
			t.Fatalf("args length = %d, want 1", len(args))
		}
		if v, ok := args[0].(*bool); !ok || *v != true {
			t.Errorf("args[0] = %v, want *true", args[0])
		}
	})

	t.Run("multiple filters combine with AND", func(t *testing.T) {
		b := query.NewBuilder(projection)
		stage := prompts.StageEnhance
		f := prompts.Filters{
			Stage:  &stage,
			Name:   ptr("verbose"),
			Active: ptr(false),
		}
		f.Apply(b)
		_, args := b.Build()

		if len(args) != 3 {
			t.Errorf("args length = %d, want 3", len(args))
		}
	})
}
