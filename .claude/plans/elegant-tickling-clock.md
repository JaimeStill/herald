# 39 — Init Node: Concurrent Page Rendering with Temp Storage

## Context

The init node is the first stage of Herald's 3-node classification workflow (init → classify → enhance?). It bridges blob storage and the document-context library: downloading a PDF, opening it, rendering all pages to images concurrently via ImageMagick, and storing the results as files in a request-bound temp directory. Downstream nodes (classify, enhance) read these image files just-in-time for LLM vision calls.

Issue #38 (workflow foundation) established the type system with two core types — `ClassificationState` and `ClassificationPage` — eliminating the need for a separate `PageImage` type. The init node populates `ClassificationPage` with only `PageNumber` and `ImagePath`, leaving classification fields for the classify node.

## Implementation

### Step 1: Add state key constants to `workflow/types.go`

Define string constants for the keys used in the `state.State` data map. These are shared across all workflow nodes and the `Execute` function.

```go
const (
    KeyDocumentID = "document_id"
    KeyTempDir    = "temp_dir"
    KeyFilename   = "filename"
    KeyPageCount  = "page_count"
    KeyClassState = "classification_state"
)
```

**File:** `workflow/types.go`

### Step 2: Create `workflow/init.go`

New file containing the `initNode` function. Returns a `state.NewFunctionNode` closure that:

1. Extracts `document_id` (uuid.UUID) and `temp_dir` (string) from state
2. Looks up the document record via `runtime.Documents.Find(ctx, documentID)`
3. Downloads the PDF blob via `runtime.Storage.Download(ctx, doc.StorageKey)` to `{tempDir}/source.pdf`
4. Opens the PDF via `document.OpenPDF(pdfPath)`
5. Creates a renderer via `image.NewImageMagickRenderer(config.DefaultImageConfig())`
6. Extracts all pages via `doc.ExtractAllPages()`
7. Renders pages concurrently using `errgroup.Group` with `SetLimit(renderWorkerCount(pageCount))`
   - Worker count: `max(min(runtime.NumCPU(), pageCount), 1)` — adapted from agent-lab's pattern
   - Each goroutine: `page.ToImage(renderer, nil)` → `os.WriteFile(filepath.Join(tempDir, "page-{N}.png"), data, 0600)`
8. Builds `[]ClassificationPage` (only `PageNumber` + `ImagePath` populated)
9. Stores initial `ClassificationState` (with pages) in state via `KeyClassState`
10. Stores document metadata (`KeyFilename`, `KeyPageCount`) in state for `WorkflowResult` assembly

**Signature:** `func initNode(runtime *Runtime) state.StateNode` (unexported — only used by graph assembly in #41)

**Error wrapping:**
- Document lookup failure → `ErrDocumentNotFound`
- Blob download, PDF open, page extraction, rendering, file write failures → `ErrRenderFailed`

**Dependencies added to go.mod:**
- `github.com/JaimeStill/document-context`
- `github.com/JaimeStill/go-agents-orchestration`
- `golang.org/x/sync` (for `errgroup`) — already an indirect dep, becomes direct

**Key files referenced:**
- `workflow/runtime.go` — Runtime struct with Documents, Storage, Logger
- `workflow/types.go` — ClassificationState, ClassificationPage
- `workflow/errors.go` — ErrDocumentNotFound, ErrRenderFailed
- `internal/documents/system.go` — `Find(ctx, uuid.UUID) (*Document, error)`
- `pkg/storage/storage.go` — `Download(ctx, key) (*BlobResult, error)`, BlobResult.Body is `io.ReadCloser`
- `document-context/pkg/document/pdf.go` — `OpenPDF(path)`, `ExtractAllPages()`, `ToImage(renderer, nil)`
- `document-context/pkg/image/imagemagick.go` — `NewImageMagickRenderer(cfg)`
- `document-context/pkg/config/image.go` — `DefaultImageConfig()`
- `go-agents-orchestration/pkg/state/node.go` — `NewFunctionNode(fn)`
- `go-agents-orchestration/pkg/state/state.go` — `State.Get(key)`, `State.Set(key, value)`

## Validation Criteria

- [ ] `go vet ./...` passes
- [ ] `go mod tidy` produces no changes
- [ ] All existing tests pass: `go test ./tests/...`
- [ ] New tests for `initNode` cover: state key extraction, document lookup, page rendering output, error wrapping with sentinels
- [ ] Init node produces `ClassificationState` with correct number of `ClassificationPage` entries
- [ ] Each `ClassificationPage` has `PageNumber` (1-indexed) and `ImagePath` pointing to `{tempDir}/page-{N}.png`
- [ ] Concurrency is bounded via `errgroup.SetLimit`
- [ ] Errors wrap `ErrDocumentNotFound` or `ErrRenderFailed` as appropriate
