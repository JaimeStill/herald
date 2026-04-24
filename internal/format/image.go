package format

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JaimeStill/herald/internal/state"
)

type imageHandler struct{}

// NewImageHandler returns a Handler that accepts raw image uploads
// (PNG, JPEG, WEBP). PNG inputs are copied verbatim as page-1.png;
// JPEG and WEBP inputs are normalized to PNG via magick so downstream
// vision calls see uniform bytes regardless of source encoding.
func NewImageHandler() Handler { return &imageHandler{} }

func (h *imageHandler) ID() string { return "image" }
func (h *imageHandler) ContentTypes() []string {
	return []string{
		"image/png",
		"image/jpeg",
		"image/webp",
	}
}

// Extract produces exactly one page for any supported image type. PNG
// sources are copied byte-for-byte to <tempDir>/page-1.png; JPEG and
// WEBP sources are staged as <tempDir>/source.<ext> and normalized to
// PNG via magick. The returned slice always has len == 1.
func (h *imageHandler) Extract(
	ctx context.Context,
	src SourceReader,
	tempDir string,
) ([]state.ClassificationPage, error) {
	outPath := filepath.Join(tempDir, "page-1.png")
	data, err := readAll(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("read image source: %w", err)
	}

	ct := src.ContentType()
	if ct == "image/png" {
		if err := os.WriteFile(outPath, data, 0600); err != nil {
			return nil, fmt.Errorf("write png: %w", err)
		}
	} else {
		ext := h.extension(ct)
		if ext == "" {
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, ct)
		}
		srcPath := filepath.Join(tempDir, "source"+ext)
		if err := os.WriteFile(srcPath, data, 0600); err != nil {
			return nil, fmt.Errorf("write image source: %w", err)
		}
		if err := Render(ctx, srcPath, outPath, false, nil); err != nil {
			return nil, fmt.Errorf("normalize %s: %w", ct, err)
		}
	}

	return []state.ClassificationPage{{PageNumber: 1, ImagePath: outPath}}, nil
}

// Enhance re-applies filter settings to the normalized PNG produced by
// Extract and writes the result to <tempDir>/page-1-enhanced.png. Unlike
// the PDF handler, Enhance works from the already-normalized intermediate
// rather than the original source, which avoids a second blob fetch.
func (h *imageHandler) Enhance(
	ctx context.Context,
	tempDir string,
	page *state.ClassificationPage,
	settings *state.EnhanceSettings,
) (string, error) {
	srcPath := filepath.Join(tempDir, "page-1.png")
	outPath := filepath.Join(tempDir, "page-1-enhanced.png")
	if err := Render(ctx, srcPath, outPath, false, settings); err != nil {
		return "", fmt.Errorf("enhance image: %w", err)
	}
	return outPath, nil
}

// extension maps a supported image content type to its file extension
// (including the leading dot). Returns "" for unrecognized types; callers
// should treat an empty return as an unsupported-format signal.
func (h *imageHandler) extension(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}
