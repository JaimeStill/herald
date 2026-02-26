package classifications_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/internal/classifications"
	"github.com/JaimeStill/herald/pkg/query"
)

func ptr[T any](v T) *T { return &v }

func TestMapHTTPStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"not found", classifications.ErrNotFound, http.StatusNotFound},
		{"duplicate", classifications.ErrDuplicate, http.StatusConflict},
		{"invalid status", classifications.ErrInvalidStatus, http.StatusConflict},
		{"unknown error", errors.New("something else"), http.StatusInternalServerError},
		{"wrapped not found", fmt.Errorf("find failed: %w", classifications.ErrNotFound), http.StatusNotFound},
		{"wrapped duplicate", fmt.Errorf("insert failed: %w", classifications.ErrDuplicate), http.StatusConflict},
		{"wrapped invalid status", fmt.Errorf("validate failed: %w", classifications.ErrInvalidStatus), http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifications.MapHTTPStatus(tt.err)
			if got != tt.want {
				t.Errorf("MapHTTPStatus(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestFiltersFromQuery(t *testing.T) {
	t.Run("all params present", func(t *testing.T) {
		id := uuid.New()
		values := url.Values{
			"classification": {"SECRET"},
			"confidence":     {"HIGH"},
			"document_id":    {id.String()},
			"validated_by":   {"admin"},
		}

		f := classifications.FiltersFromQuery(values)

		if f.Classification == nil || *f.Classification != "SECRET" {
			t.Errorf("Classification = %v, want SECRET", f.Classification)
		}
		if f.Confidence == nil || *f.Confidence != "HIGH" {
			t.Errorf("Confidence = %v, want HIGH", f.Confidence)
		}
		if f.DocumentID == nil || *f.DocumentID != id {
			t.Errorf("DocumentID = %v, want %s", f.DocumentID, id)
		}
		if f.ValidatedBy == nil || *f.ValidatedBy != "admin" {
			t.Errorf("ValidatedBy = %v, want admin", f.ValidatedBy)
		}
	})

	t.Run("empty params yield nil fields", func(t *testing.T) {
		f := classifications.FiltersFromQuery(url.Values{})

		if f.Classification != nil {
			t.Errorf("Classification = %v, want nil", f.Classification)
		}
		if f.Confidence != nil {
			t.Errorf("Confidence = %v, want nil", f.Confidence)
		}
		if f.DocumentID != nil {
			t.Errorf("DocumentID = %v, want nil", f.DocumentID)
		}
		if f.ValidatedBy != nil {
			t.Errorf("ValidatedBy = %v, want nil", f.ValidatedBy)
		}
	})

	t.Run("invalid document_id ignored", func(t *testing.T) {
		values := url.Values{"document_id": {"not-a-uuid"}}
		f := classifications.FiltersFromQuery(values)

		if f.DocumentID != nil {
			t.Errorf("DocumentID = %v, want nil for invalid UUID", f.DocumentID)
		}
	})

	t.Run("partial params", func(t *testing.T) {
		values := url.Values{
			"classification": {"UNCLASSIFIED"},
			"validated_by":   {"reviewer"},
		}

		f := classifications.FiltersFromQuery(values)

		if f.Classification == nil || *f.Classification != "UNCLASSIFIED" {
			t.Errorf("Classification = %v, want UNCLASSIFIED", f.Classification)
		}
		if f.Confidence != nil {
			t.Errorf("Confidence = %v, want nil", f.Confidence)
		}
		if f.DocumentID != nil {
			t.Errorf("DocumentID = %v, want nil", f.DocumentID)
		}
		if f.ValidatedBy == nil || *f.ValidatedBy != "reviewer" {
			t.Errorf("ValidatedBy = %v, want reviewer", f.ValidatedBy)
		}
	})
}

func TestFiltersApply(t *testing.T) {
	proj := query.
		NewProjectionMap("public", "classifications", "c").
		Project("classification", "Classification").
		Project("confidence", "Confidence").
		Project("document_id", "DocumentID").
		Project("validated_by", "ValidatedBy")

	t.Run("no filters produces no WHERE clause", func(t *testing.T) {
		b := query.NewBuilder(proj)
		f := classifications.Filters{}
		f.Apply(b)
		sql, args := b.Build()

		wantSQL := "SELECT c.classification, c.confidence, c.document_id, c.validated_by FROM public.classifications c"
		if sql != wantSQL {
			t.Errorf("sql = %q, want %q", sql, wantSQL)
		}
		if len(args) != 0 {
			t.Errorf("args = %v, want empty", args)
		}
	})

	t.Run("classification equals filter", func(t *testing.T) {
		b := query.NewBuilder(proj)
		f := classifications.Filters{Classification: ptr("SECRET")}
		f.Apply(b)
		_, args := b.Build()

		if len(args) != 1 {
			t.Fatalf("args length = %d, want 1", len(args))
		}
	})

	t.Run("confidence equals filter", func(t *testing.T) {
		b := query.NewBuilder(proj)
		f := classifications.Filters{Confidence: ptr("HIGH")}
		f.Apply(b)
		_, args := b.Build()

		if len(args) != 1 {
			t.Fatalf("args length = %d, want 1", len(args))
		}
	})

	t.Run("document_id equals filter", func(t *testing.T) {
		id := uuid.New()
		b := query.NewBuilder(proj)
		f := classifications.Filters{DocumentID: &id}
		f.Apply(b)
		_, args := b.Build()

		if len(args) != 1 {
			t.Fatalf("args length = %d, want 1", len(args))
		}
	})

	t.Run("validated_by equals filter", func(t *testing.T) {
		b := query.NewBuilder(proj)
		f := classifications.Filters{ValidatedBy: ptr("admin")}
		f.Apply(b)
		_, args := b.Build()

		if len(args) != 1 {
			t.Fatalf("args length = %d, want 1", len(args))
		}
	})

	t.Run("multiple filters combine with AND", func(t *testing.T) {
		b := query.NewBuilder(proj)
		f := classifications.Filters{
			Classification: ptr("SECRET"),
			Confidence:     ptr("HIGH"),
			ValidatedBy:    ptr("admin"),
		}
		f.Apply(b)
		_, args := b.Build()

		if len(args) != 3 {
			t.Errorf("args length = %d, want 3", len(args))
		}
	})
}

