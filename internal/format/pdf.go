package format

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/JaimeStill/herald/internal/state"
	"github.com/JaimeStill/herald/pkg/core"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"golang.org/x/sync/errgroup"
)

const sourcePDF = "source.pdf"

type pdfHandler struct{}

// NewPDFHandler returns a Handler that accepts application/pdf. It uses
// pdfcpu to count pages and ImageMagick (via Render) to rasterize each
// page to PNG at 300 DPI, parallelized with bounded concurrency sized by
// core.WorkerCount.
func NewPDFHandler() Handler { return &pdfHandler{} }

func (h *pdfHandler) ID() string             { return "pdf" }
func (h *pdfHandler) ContentTypes() []string { return []string{"application/pdf"} }

// Extract writes the source PDF to <tempDir>/source.pdf, counts pages with
// pdfcpu, and renders each page to <tempDir>/page-N.png in parallel. The
// per-page rendering uses magick's native PDF page-selector syntax
// (source.pdf[N-1]) so we avoid pulling apart the document ourselves.
func (h *pdfHandler) Extract(
	ctx context.Context,
	src SourceReader,
	tempDir string,
) ([]state.ClassificationPage, error) {
	pdfPath := filepath.Join(tempDir, sourcePDF)

	data, err := readAll(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("read pdf source: %w", err)
	}

	if err := os.WriteFile(pdfPath, data, 0600); err != nil {
		return nil, fmt.Errorf("write pdf source: %w", err)
	}

	pageCount, err := api.PageCount(bytes.NewReader(data), nil)
	if err != nil {
		return nil, fmt.Errorf("count pages: %w", err)
	}

	pages := make([]state.ClassificationPage, pageCount)

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(core.WorkerCount(pageCount))

	for i := range pageCount {
		pageNum := i + 1
		imgPath := filepath.Join(tempDir, fmt.Sprintf("page-%d.png", pageNum))
		pages[i] = state.ClassificationPage{
			PageNumber: pageNum,
			ImagePath:  imgPath,
		}

		g.Go(func() error {
			if gctx.Err() != nil {
				return gctx.Err()
			}
			return Render(gctx, pdfPageSelector(pdfPath, pageNum), imgPath, true, nil)
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("render pdf pages: %w", err)
	}

	return pages, nil
}

// Enhance re-renders the given page from <tempDir>/source.pdf with the
// supplied filter settings applied. Extract must have run previously to
// seed the source PDF; the result is written to <tempDir>/page-N-enhanced.png
// and the path is returned.
func (h *pdfHandler) Enhance(
	ctx context.Context,
	tempDir string,
	page *state.ClassificationPage,
	settings *state.EnhanceSettings,
) (string, error) {
	pdfPath := filepath.Join(tempDir, sourcePDF)
	imgPath := filepath.Join(tempDir, fmt.Sprintf("page-%d-enhanced.png", page.PageNumber))

	if err := Render(
		ctx,
		pdfPageSelector(pdfPath, page.PageNumber),
		imgPath,
		true,
		settings,
	); err != nil {
		return "", fmt.Errorf("enhance page %d: %w", page.PageNumber, err)
	}

	return imgPath, nil
}

// pdfPageSelector formats a PDF page selector for magick (zero-indexed).
// ImageMagick interprets `file.pdf[N]` as "the N-th page of file.pdf",
// so we translate Herald's 1-indexed PageNumber by subtracting 1.
func pdfPageSelector(src string, page int) string {
	return src + "[" + strconv.Itoa(page-1) + "]"
}
