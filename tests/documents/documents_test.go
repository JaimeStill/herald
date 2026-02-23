package documents_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/JaimeStill/herald/internal/documents"
	"github.com/JaimeStill/herald/pkg/query"
)

func ptr[T any](v T) *T { return &v }

func TestMapHTTPStatus(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		want   int
	}{
		{"not found", documents.ErrNotFound, http.StatusNotFound},
		{"duplicate", documents.ErrDuplicate, http.StatusConflict},
		{"file too large", documents.ErrFileTooLarge, http.StatusRequestEntityTooLarge},
		{"invalid file", documents.ErrInvalidFile, http.StatusBadRequest},
		{"unknown error", errors.New("something else"), http.StatusInternalServerError},
		{"wrapped not found", fmt.Errorf("find failed: %w", documents.ErrNotFound), http.StatusNotFound},
		{"wrapped duplicate", fmt.Errorf("insert failed: %w", documents.ErrDuplicate), http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := documents.MapHTTPStatus(tt.err)
			if got != tt.want {
				t.Errorf("MapHTTPStatus(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestFiltersFromQuery(t *testing.T) {
	t.Run("all params present", func(t *testing.T) {
		values := url.Values{
			"status":            {"pending"},
			"filename":          {"report"},
			"external_id":       {"42"},
			"external_platform": {"sharepoint"},
			"content_type":      {"application/pdf"},
			"storage_key":       {"documents/abc"},
		}

		f := documents.FiltersFromQuery(values)

		if f.Status == nil || *f.Status != "pending" {
			t.Errorf("Status = %v, want pending", f.Status)
		}
		if f.Filename == nil || *f.Filename != "report" {
			t.Errorf("Filename = %v, want report", f.Filename)
		}
		if f.ExternalID == nil || *f.ExternalID != 42 {
			t.Errorf("ExternalID = %v, want 42", f.ExternalID)
		}
		if f.ExternalPlatform == nil || *f.ExternalPlatform != "sharepoint" {
			t.Errorf("ExternalPlatform = %v, want sharepoint", f.ExternalPlatform)
		}
		if f.ContentType == nil || *f.ContentType != "application/pdf" {
			t.Errorf("ContentType = %v, want application/pdf", f.ContentType)
		}
		if f.StorageKey == nil || *f.StorageKey != "documents/abc" {
			t.Errorf("StorageKey = %v, want documents/abc", f.StorageKey)
		}
	})

	t.Run("empty params yield nil fields", func(t *testing.T) {
		f := documents.FiltersFromQuery(url.Values{})

		if f.Status != nil {
			t.Errorf("Status = %v, want nil", f.Status)
		}
		if f.Filename != nil {
			t.Errorf("Filename = %v, want nil", f.Filename)
		}
		if f.ExternalID != nil {
			t.Errorf("ExternalID = %v, want nil", f.ExternalID)
		}
		if f.ExternalPlatform != nil {
			t.Errorf("ExternalPlatform = %v, want nil", f.ExternalPlatform)
		}
		if f.ContentType != nil {
			t.Errorf("ContentType = %v, want nil", f.ContentType)
		}
		if f.StorageKey != nil {
			t.Errorf("StorageKey = %v, want nil", f.StorageKey)
		}
	})

	t.Run("invalid external_id ignored", func(t *testing.T) {
		values := url.Values{"external_id": {"not-a-number"}}
		f := documents.FiltersFromQuery(values)

		if f.ExternalID != nil {
			t.Errorf("ExternalID = %v, want nil for invalid input", f.ExternalID)
		}
	})

	t.Run("partial params", func(t *testing.T) {
		values := url.Values{
			"status":   {"review"},
			"filename": {"classified"},
		}

		f := documents.FiltersFromQuery(values)

		if f.Status == nil || *f.Status != "review" {
			t.Errorf("Status = %v, want review", f.Status)
		}
		if f.Filename == nil || *f.Filename != "classified" {
			t.Errorf("Filename = %v, want classified", f.Filename)
		}
		if f.ExternalPlatform != nil {
			t.Errorf("ExternalPlatform = %v, want nil", f.ExternalPlatform)
		}
	})
}

func TestFiltersApply(t *testing.T) {
	projection := query.
		NewProjectionMap("public", "documents", "d").
		Project("status", "Status").
		Project("filename", "Filename").
		Project("external_id", "ExternalID").
		Project("external_platform", "ExternalPlatform").
		Project("content_type", "ContentType").
		Project("storage_key", "StorageKey")

	t.Run("no filters produces no WHERE clause", func(t *testing.T) {
		b := query.NewBuilder(projection)
		f := documents.Filters{}
		f.Apply(b)
		sql, args := b.Build()

		wantSQL := "SELECT d.status, d.filename, d.external_id, d.external_platform, d.content_type, d.storage_key FROM public.documents d"
		if sql != wantSQL {
			t.Errorf("sql = %q, want %q", sql, wantSQL)
		}
		if len(args) != 0 {
			t.Errorf("args = %v, want empty", args)
		}
	})

	t.Run("status equals filter", func(t *testing.T) {
		b := query.NewBuilder(projection)
		f := documents.Filters{Status: ptr("pending")}
		f.Apply(b)
		_, args := b.Build()

		if len(args) != 1 {
			t.Fatalf("args length = %d, want 1", len(args))
		}
		if v, ok := args[0].(*string); !ok || *v != "pending" {
			t.Errorf("args[0] = %v, want *pending", args[0])
		}
	})

	t.Run("filename contains filter", func(t *testing.T) {
		b := query.NewBuilder(projection)
		f := documents.Filters{Filename: ptr("report")}
		f.Apply(b)
		_, args := b.Build()

		if len(args) != 1 || args[0] != "%report%" {
			t.Errorf("args = %v, want [%%report%%]", args)
		}
	})

	t.Run("storage key contains filter", func(t *testing.T) {
		b := query.NewBuilder(projection)
		f := documents.Filters{StorageKey: ptr("documents/abc")}
		f.Apply(b)
		_, args := b.Build()

		if len(args) != 1 || args[0] != "%documents/abc%" {
			t.Errorf("args = %v, want [%%documents/abc%%]", args)
		}
	})

	t.Run("multiple filters combine with AND", func(t *testing.T) {
		b := query.NewBuilder(projection)
		f := documents.Filters{
			Status:           ptr("pending"),
			Filename:         ptr("report"),
			ExternalPlatform: ptr("sharepoint"),
		}
		f.Apply(b)
		_, args := b.Build()

		if len(args) != 3 {
			t.Errorf("args length = %d, want 3", len(args))
		}
	})

	t.Run("external_id equals filter", func(t *testing.T) {
		b := query.NewBuilder(projection)
		f := documents.Filters{ExternalID: ptr(42)}
		f.Apply(b)
		_, args := b.Build()

		if len(args) != 1 {
			t.Fatalf("args length = %d, want 1", len(args))
		}
		if v, ok := args[0].(*int); !ok || *v != 42 {
			t.Errorf("args[0] = %v, want *42", args[0])
		}
	})

	t.Run("content_type equals filter", func(t *testing.T) {
		b := query.NewBuilder(projection)
		f := documents.Filters{ContentType: ptr("application/pdf")}
		f.Apply(b)
		_, args := b.Build()

		if len(args) != 1 {
			t.Fatalf("args length = %d, want 1", len(args))
		}
		if v, ok := args[0].(*string); !ok || *v != "application/pdf" {
			t.Errorf("args[0] = %v, want *application/pdf", args[0])
		}
	})
}
