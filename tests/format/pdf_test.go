package format_test

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/JaimeStill/herald/internal/format"
	"github.com/JaimeStill/herald/internal/state"
)

// fixtureSource wraps a file on disk as a format.SourceReader so the handler
// exercises its real code path end-to-end without needing blob storage.
type fixtureSource struct {
	path        string
	contentType string
}

func (f *fixtureSource) Open(context.Context) (io.ReadCloser, error) {
	return os.Open(f.path)
}
func (f *fixtureSource) ContentType() string { return f.contentType }
func (f *fixtureSource) Filename() string    { return filepath.Base(f.path) }

// requireMagick skips the test when the ImageMagick CLI is not on PATH.
// PDF / image handler tests shell out to `magick`; skipping keeps CI
// environments without the binary from reporting spurious failures.
func requireMagick(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("magick"); err != nil {
		t.Skip("skipping: magick not on PATH")
	}
}

// fixturePath resolves a path under _project/ relative to the repository root.
// Tests run from the package directory, so we walk up to find the project root.
func fixturePath(t *testing.T, relative string) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	for {
		candidate := filepath.Join(dir, relative)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not resolve fixture %s", relative)
		}
		dir = parent
	}
}

func TestPDFHandlerExtract(t *testing.T) {
	requireMagick(t)

	h := format.NewPDFHandler()
	if h.ID() != "pdf" {
		t.Errorf("ID = %q, want pdf", h.ID())
	}
	if cts := h.ContentTypes(); len(cts) != 1 || cts[0] != "application/pdf" {
		t.Errorf("ContentTypes = %v, want [application/pdf]", cts)
	}

	tempDir := t.TempDir()
	src := &fixtureSource{
		path:        fixturePath(t, "_project/marked-documents/single-unclassified.pdf"),
		contentType: "application/pdf",
	}

	pages, err := h.Extract(context.Background(), src, tempDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}

	if len(pages) < 1 {
		t.Fatalf("expected at least 1 page, got %d", len(pages))
	}

	// Validate the page records and that each image file exists on disk.
	for i, p := range pages {
		wantPageNumber := i + 1
		if p.PageNumber != wantPageNumber {
			t.Errorf("page %d PageNumber = %d, want %d", i, p.PageNumber, wantPageNumber)
		}
		if p.ImagePath == "" {
			t.Errorf("page %d ImagePath empty", i)
			continue
		}
		info, err := os.Stat(p.ImagePath)
		if err != nil {
			t.Errorf("page %d stat %s: %v", i, p.ImagePath, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("page %d image is empty", i)
		}
	}

	// source.pdf copy should be in the temp dir (used by Enhance later).
	if _, err := os.Stat(filepath.Join(tempDir, "source.pdf")); err != nil {
		t.Errorf("source.pdf not written: %v", err)
	}
}

func TestPDFHandlerEnhance(t *testing.T) {
	requireMagick(t)

	h := format.NewPDFHandler()
	tempDir := t.TempDir()

	// Enhance reads from <tempDir>/source.pdf, so seed it first via Extract.
	src := &fixtureSource{
		path:        fixturePath(t, "_project/marked-documents/single-unclassified.pdf"),
		contentType: "application/pdf",
	}
	pages, err := h.Extract(context.Background(), src, tempDir)
	if err != nil {
		t.Fatalf("Extract (setup) error: %v", err)
	}
	if len(pages) == 0 {
		t.Fatal("no pages produced from fixture")
	}

	brightness := 20
	contrast := 10
	settings := &state.EnhanceSettings{Brightness: &brightness, Contrast: &contrast}

	enhancedPath, err := h.Enhance(context.Background(), tempDir, &pages[0], settings)
	if err != nil {
		t.Fatalf("Enhance error: %v", err)
	}

	wantName := "page-1-enhanced.png"
	if filepath.Base(enhancedPath) != wantName {
		t.Errorf("enhanced path = %q, want basename %q", enhancedPath, wantName)
	}

	info, err := os.Stat(enhancedPath)
	if err != nil {
		t.Fatalf("stat enhanced: %v", err)
	}
	if info.Size() == 0 {
		t.Error("enhanced page is empty")
	}
}
