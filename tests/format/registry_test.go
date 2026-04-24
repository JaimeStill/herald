package format_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JaimeStill/herald/internal/format"
	"github.com/JaimeStill/herald/internal/state"
)

// stubHandler is a minimal Handler implementation for registry tests.
// Extract and Enhance are not exercised here — those are covered in
// pdf_test.go and image_test.go against real magick.
type stubHandler struct {
	id           string
	contentTypes []string
}

func (s *stubHandler) ID() string             { return s.id }
func (s *stubHandler) ContentTypes() []string { return s.contentTypes }
func (s *stubHandler) Extract(context.Context, format.SourceReader, string) ([]state.ClassificationPage, error) {
	return nil, nil
}
func (s *stubHandler) Enhance(context.Context, string, *state.ClassificationPage, *state.EnhanceSettings) (string, error) {
	return "", nil
}

func newStubRegistry() *format.Registry {
	return format.NewRegistry(
		&stubHandler{id: "pdf", contentTypes: []string{"application/pdf"}},
		&stubHandler{id: "image", contentTypes: []string{"image/png", "image/jpeg", "image/webp"}},
	)
}

func TestRegistryLookup(t *testing.T) {
	r := newStubRegistry()

	tests := []struct {
		name        string
		contentType string
		wantID      string
		wantErr     bool
	}{
		{"pdf hit", "application/pdf", "pdf", false},
		{"png hit", "image/png", "image", false},
		{"jpeg hit", "image/jpeg", "image", false},
		{"webp hit", "image/webp", "image", false},
		{"docx miss", "application/msword", "", true},
		{"empty string miss", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := r.Lookup(tt.contentType)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !errors.Is(err, format.ErrUnsupportedFormat) {
					t.Errorf("error = %v, want wraps ErrUnsupportedFormat", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if h.ID() != tt.wantID {
				t.Errorf("handler ID = %q, want %q", h.ID(), tt.wantID)
			}
		})
	}
}

func TestRegistrySupportedContentTypes(t *testing.T) {
	r := newStubRegistry()
	got := r.SupportedContentTypes()

	want := []string{
		"application/pdf",
		"image/jpeg",
		"image/png",
		"image/webp",
	}

	if len(got) != len(want) {
		t.Fatalf("length = %d, want %d (got %v)", len(got), len(want), got)
	}
	for i, ct := range want {
		if got[i] != ct {
			t.Errorf("[%d] = %q, want %q", i, got[i], ct)
		}
	}
}

func TestRegistryDeterministicOrder(t *testing.T) {
	// Multiple calls must return the same order — guards against accidentally
	// reintroducing map iteration or other nondeterministic sources.
	r := newStubRegistry()
	first := r.SupportedContentTypes()
	for range 10 {
		next := r.SupportedContentTypes()
		if len(next) != len(first) {
			t.Fatalf("length drift: %d vs %d", len(next), len(first))
		}
		for i := range first {
			if first[i] != next[i] {
				t.Errorf("order drift at [%d]: %q vs %q", i, first[i], next[i])
			}
		}
	}
}

func TestRegistryEmpty(t *testing.T) {
	r := format.NewRegistry()

	if _, err := r.Lookup("application/pdf"); !errors.Is(err, format.ErrUnsupportedFormat) {
		t.Errorf("empty registry Lookup error = %v, want ErrUnsupportedFormat", err)
	}

	got := r.SupportedContentTypes()
	if len(got) != 0 {
		t.Errorf("empty registry SupportedContentTypes = %v, want []", got)
	}
}

