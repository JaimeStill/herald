package format_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/JaimeStill/herald/internal/format"
	"github.com/JaimeStill/herald/internal/state"
)

// pngSignature is the first 8 bytes of any valid PNG file — used to verify
// normalization output without pulling in an image-decoding dependency.
var pngSignature = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

func TestImageHandlerMetadata(t *testing.T) {
	h := format.NewImageHandler()
	if h.ID() != "image" {
		t.Errorf("ID = %q, want image", h.ID())
	}

	cts := h.ContentTypes()
	want := map[string]bool{"image/png": false, "image/jpeg": false, "image/webp": false}
	for _, ct := range cts {
		if _, ok := want[ct]; !ok {
			t.Errorf("unexpected content type: %s", ct)
			continue
		}
		want[ct] = true
	}
	for ct, seen := range want {
		if !seen {
			t.Errorf("missing content type: %s", ct)
		}
	}
}

func TestImageHandlerExtractPNGPassthrough(t *testing.T) {
	h := format.NewImageHandler()
	tempDir := t.TempDir()

	src := &fixtureSource{
		path:        fixturePath(t, "_project/marked-documents/images/marked-document.1.png"),
		contentType: "image/png",
	}

	pages, err := h.Extract(context.Background(), src, tempDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}

	if len(pages) != 1 {
		t.Fatalf("pages length = %d, want 1", len(pages))
	}
	if pages[0].PageNumber != 1 {
		t.Errorf("PageNumber = %d, want 1", pages[0].PageNumber)
	}

	want := filepath.Join(tempDir, "page-1.png")
	if pages[0].ImagePath != want {
		t.Errorf("ImagePath = %q, want %q", pages[0].ImagePath, want)
	}

	// PNG passthrough: output bytes should equal the source bytes exactly.
	origBytes, err := os.ReadFile(src.path)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	outBytes, err := os.ReadFile(pages[0].ImagePath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !bytes.Equal(origBytes, outBytes) {
		t.Error("PNG passthrough: output bytes differ from source (expected byte-identical copy)")
	}
}

func TestImageHandlerExtractJPEGNormalization(t *testing.T) {
	requireMagick(t)

	h := format.NewImageHandler()
	tempDir := t.TempDir()

	src := &fixtureSource{
		path:        fixturePath(t, "_project/marked-documents/images/marked-document.1.jpg"),
		contentType: "image/jpeg",
	}

	pages, err := h.Extract(context.Background(), src, tempDir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}

	if len(pages) != 1 {
		t.Fatalf("pages length = %d, want 1", len(pages))
	}

	outBytes, err := os.ReadFile(pages[0].ImagePath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !bytes.HasPrefix(outBytes, pngSignature) {
		t.Errorf("output is not a valid PNG (missing PNG signature)")
	}

	// Normalized output should be saved under the canonical page-1.png name,
	// regardless of the source extension.
	wantName := "page-1.png"
	if filepath.Base(pages[0].ImagePath) != wantName {
		t.Errorf("ImagePath basename = %q, want %q", filepath.Base(pages[0].ImagePath), wantName)
	}
}

func TestImageHandlerEnhance(t *testing.T) {
	requireMagick(t)

	h := format.NewImageHandler()
	tempDir := t.TempDir()

	// Enhance reads from <tempDir>/page-1.png, so seed it via Extract first.
	src := &fixtureSource{
		path:        fixturePath(t, "_project/marked-documents/images/marked-document.1.png"),
		contentType: "image/png",
	}
	pages, err := h.Extract(context.Background(), src, tempDir)
	if err != nil {
		t.Fatalf("Extract (setup) error: %v", err)
	}
	if len(pages) != 1 {
		t.Fatal("setup failed to produce page")
	}

	brightness := 15
	saturation := 120
	settings := &state.EnhanceSettings{Brightness: &brightness, Saturation: &saturation}

	enhancedPath, err := h.Enhance(context.Background(), tempDir, &pages[0], settings)
	if err != nil {
		t.Fatalf("Enhance error: %v", err)
	}

	wantName := "page-1-enhanced.png"
	if filepath.Base(enhancedPath) != wantName {
		t.Errorf("enhanced path basename = %q, want %q", filepath.Base(enhancedPath), wantName)
	}

	info, err := os.Stat(enhancedPath)
	if err != nil {
		t.Fatalf("stat enhanced: %v", err)
	}
	if info.Size() == 0 {
		t.Error("enhanced image is empty")
	}
}

func TestImageHandlerExtractUnsupportedContentType(t *testing.T) {
	h := format.NewImageHandler()
	tempDir := t.TempDir()

	src := &fixtureSource{
		path:        fixturePath(t, "_project/marked-documents/images/marked-document.1.png"),
		contentType: "image/bogus",
	}

	_, err := h.Extract(context.Background(), src, tempDir)
	if err == nil {
		t.Fatal("expected error for unsupported content type")
	}
}
