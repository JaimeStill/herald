# 39 — Init Node: Concurrent Page Rendering with Temp Storage

## Problem Context

The init node is the first stage of the classification workflow. It downloads a PDF from blob storage, opens it via document-context, and renders all pages to image files in a temp directory using concurrent ImageMagick rendering. Downstream nodes (classify, enhance) read these image files just-in-time for LLM vision calls, keeping memory proportional to one page at a time.

## Architecture Approach

The init node follows the `state.NewFunctionNode` closure pattern from go-agents-orchestration. It receives `document_id` and `temp_dir` from the state bag (set by `Execute` in #41), performs all PDF-to-image work, and returns updated state with the initial `ClassificationState` and document metadata.

Page rendering is CPU-heavy (ImageMagick), so pages are rendered concurrently using `errgroup.Group` with `SetLimit`. Worker count is capped at `min(runtime.NumCPU(), pageCount)` — no point spawning more goroutines than pages or CPUs. Page extraction via `ExtractAllPages()` happens sequentially before the concurrent render phase.

The simplified type system from #38 means the init node creates `ClassificationPage` structs with only `PageNumber` and `ImagePath` populated — no separate `PageImage` type needed.

## Implementation

### Step 1: Add dependencies

```bash
go get github.com/JaimeStill/document-context@latest
go get github.com/JaimeStill/go-agents-orchestration@latest
```

`golang.org/x/sync` is already an indirect dependency. It will become direct when imported for `errgroup`.

Run `go mod tidy` after adding the imports in Step 3.

### Step 2: Add state key constants to `workflow/types.go`

Add these constants above the `Confidence` type:

```go
const (
	KeyDocumentID = "document_id"
	KeyTempDir    = "temp_dir"
	KeyFilename   = "filename"
	KeyPageCount  = "page_count"
	KeyClassState = "classification_state"
)
```

### Step 3: Create `workflow/init.go`

New file — complete implementation:

```go
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
	"github.com/JaimeStill/herald/internal/documents"

	"github.com/JaimeStill/go-agents-orchestration/pkg/state"

	"golang.org/x/sync/errgroup"
)

const sourcePDF = "source.pdf"

func renderWorkerCount(pageCount int) int {
	return max(min(runtime.NumCPU(), pageCount), 1)
}

// extractInitInputs reads the document ID and temp directory path from workflow state.
func extractInitInputs(s state.State) (uuid.UUID, string, error) {
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

// downloadPDF finds the document record, downloads its blob from storage,
// and writes it to {tempDir}/source.pdf. Returns the document for metadata access.
func downloadPDF(ctx context.Context, rt *Runtime, documentID uuid.UUID, tempDir string) (*documents.Document, error) {
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

// renderPages opens the PDF at {tempDir}/source.pdf, creates an ImageMagick renderer,
// extracts all pages, and renders them concurrently to PNG files in tempDir.
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

func initNode(rt *Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		documentID, tempDir, err := extractInitInputs(s)
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

		rt.Logger.InfoContext(ctx, "init node complete",
			"document_id", documentID,
			"page_count", len(pages),
		)

		s = s.Set(KeyClassState, ClassificationState{Pages: pages})
		s = s.Set(KeyFilename, doc.Filename)
		s = s.Set(KeyPageCount, len(pages))

		return s, nil
	})
}
```

### Step 4: Run `go mod tidy`

```bash
go mod tidy
```

This resolves the new direct dependencies and promotes `golang.org/x/sync` from indirect to direct.

## Validation Criteria

- [ ] `go mod tidy` produces no changes (after initial tidy)
- [ ] `go vet ./...` passes
- [ ] All existing tests pass: `go test ./tests/...`
- [ ] `initNode` extracts `document_id` and `temp_dir` from state
- [ ] PDF is downloaded from blob storage and written to `{tempDir}/source.pdf`
- [ ] All pages rendered concurrently with bounded workers (`min(NumCPU, pageCount)`)
- [ ] Each `ClassificationPage` has 1-indexed `PageNumber` and `ImagePath` at `{tempDir}/page-{N}.png`
- [ ] `ClassificationState` with pages stored in state under `KeyClassState`
- [ ] Document metadata (`filename`, `page_count`) stored in state
- [ ] Errors wrap `ErrDocumentNotFound` or `ErrRenderFailed` as appropriate
