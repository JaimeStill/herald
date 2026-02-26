package workflow

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/google/uuid"

	"github.com/JaimeStill/document-context/pkg/config"
	"github.com/JaimeStill/document-context/pkg/document"
	"github.com/JaimeStill/document-context/pkg/image"

	"github.com/JaimeStill/go-agents-orchestration/pkg/state"

	"github.com/JaimeStill/herald/internal/documents"

	"golang.org/x/sync/errgroup"
)

const sourcePDF = "source.pdf"

// InitNode returns a state node that downloads a PDF from blob storage,
// renders all pages to PNG images concurrently via ImageMagick, and stores
// the initial ClassificationState in the workflow state bag.
func InitNode(rt *Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		documentID, tempDir, err := extractInitState(s)
		if err != nil {
			return s, fmt.Errorf("init: %w", err)
		}

		doc, err := downloadPDF(ctx, rt, documentID, tempDir)
		if err != nil {
			return s, fmt.Errorf("init: %w", err)
		}

		pages, err := renderPages(ctx, tempDir)
		if err != nil {
			return s, fmt.Errorf("init: %w", err)
		}

		rt.Logger.InfoContext(
			ctx, "init node complete",
			"document_id", documentID,
			"page_count", len(pages),
		)

		s = s.Set(KeyClassState, ClassificationState{Pages: pages})
		s = s.Set(KeyFilename, doc.Filename)
		s = s.Set(KeyPageCount, len(pages))

		return s, nil
	})
}

func downloadPDF(
	ctx context.Context,
	rt *Runtime,
	documentID uuid.UUID,
	tempDir string,
) (*documents.Document, error) {
	doc, err := rt.Documents.Find(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDocumentNotFound, err)
	}

	blob, err := rt.Storage.Download(ctx, doc.StorageKey)
	if err != nil {
		return nil, fmt.Errorf("%w: download blob: %w", ErrRenderFailed, err)
	}
	defer blob.Body.Close()

	pdfPath := filepath.Join(tempDir, sourcePDF)
	pdfFile, err := os.Create(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("%w: create temp pdf: %w", ErrRenderFailed, err)
	}

	if _, err := io.Copy(pdfFile, blob.Body); err != nil {
		pdfFile.Close()
		return nil, fmt.Errorf("%w: write temp pdf: %w", ErrRenderFailed, err)
	}
	pdfFile.Close()

	return doc, nil
}

func extractInitState(s state.State) (uuid.UUID, string, error) {
	docIDVal, ok := s.Get(KeyDocumentID)
	if !ok {
		return uuid.Nil, "", fmt.Errorf("%w: missing %s in state", ErrDocumentNotFound, KeyDocumentID)
	}

	documentID, ok := docIDVal.(uuid.UUID)
	if !ok {
		return uuid.Nil, "", fmt.Errorf("%w: %s is not uuid.UUID", ErrDocumentNotFound, KeyDocumentID)
	}

	tempDirVal, ok := s.Get(KeyTempDir)
	if !ok {
		return uuid.Nil, "", fmt.Errorf("%w: missing %s in state", ErrRenderFailed, KeyTempDir)
	}

	tempDir, ok := tempDirVal.(string)
	if !ok {
		return uuid.Nil, "", fmt.Errorf("%w: %s is not string", ErrRenderFailed, KeyTempDir)
	}

	return documentID, tempDir, nil
}

func renderPages(ctx context.Context, tempDir string) ([]ClassificationPage, error) {
	pdfPath := filepath.Join(tempDir, sourcePDF)
	pdfDoc, err := document.OpenPDF(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("%w: open pdf: %w", ErrRenderFailed, err)
	}
	defer pdfDoc.Close()

	renderer, err := image.NewImageMagickRenderer(config.DefaultImageConfig())
	if err != nil {
		return nil, fmt.Errorf("%w: create renderer: %w", ErrRenderFailed, err)
	}

	allPages, err := pdfDoc.ExtractAllPages()
	if err != nil {
		return nil, fmt.Errorf("%w: extract pages: %w", ErrRenderFailed, err)
	}

	pageCount := len(allPages)
	pages := make([]ClassificationPage, pageCount)

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(renderWorkerCount(pageCount))

	for i, page := range allPages {
		pageNum := i + 1
		imgPath := filepath.Join(tempDir, fmt.Sprintf("page-%d.png", pageNum))
		pages[i] = ClassificationPage{
			PageNumber: pageNum,
			ImagePath:  imgPath,
		}

		g.Go(func() error {
			if gctx.Err() != nil {
				return gctx.Err()
			}

			data, err := page.ToImage(renderer, nil)
			if err != nil {
				return fmt.Errorf("render page %d: %w", pageNum, err)
			}

			if err := os.WriteFile(imgPath, data, 0600); err != nil {
				return fmt.Errorf("write page %d image: %w", pageNum, err)
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRenderFailed, err)
	}

	return pages, nil
}

func renderWorkerCount(pageCount int) int {
	return max(min(runtime.NumCPU(), pageCount), 1)
}
