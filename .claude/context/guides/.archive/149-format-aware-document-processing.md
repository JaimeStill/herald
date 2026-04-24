# 149 - Format-aware document processing (PDF + raw images)

## Problem Context

Herald currently hardcodes PDF handling throughout the classification workflow and web client. Raw-image documents (PNG/JPEG/WEBP) need to flow through the same pipeline natively, and adding further formats later (DOCX, PPTX, TIFF) should not require re-threading branches through init/enhance/upload/viewer.

This task introduces a document-format registry on both backend and frontend, implements PDF and image as the first two registered formats, and drops the `github.com/JaimeStill/document-context` dependency in favor of direct `pdfcpu` + `magick` calls.

## Architecture Approach

- **`internal/state/`** (new, top-level) — shared types (`ClassificationPage`, `ClassificationState`, `EnhanceSettings`, `Confidence`, state keys) live in a leaf package so both `format` and `workflow` import them cycle-free.
- **`internal/format/`** (new, top-level) — `Handler` interface, `Registry`, `SourceReader`, PDF + image handlers, shared `magick` exec helper. Consumed directly by `internal/documents/` for upload validation and by `internal/workflow/` for dispatch.
- **`internal/workflow/types.go`** — shrinks to just `WorkflowResult` (which wraps `state.ClassificationState` with execution metadata). No re-export aliases; every consumer references `state.*` directly.
- **`init.go` / `enhance.go`** — become thin dispatchers: `rt.Formats.Lookup(doc.ContentType).Extract(...)` / `.Enhance(...)`.
- **Registry composition** — constructed in `internal/api/domain.go` (API-domain concern, co-located with `documents.System` / `classifications.System`), wired into `documents.New` for upload validation and into `workflow.Runtime.Formats` for dispatch.
- **Upload validation** — `internal/documents/handler.go` looks up the content type in the registry and returns HTTP 400 (`ErrUnsupportedContentType`) when the lookup misses.
- **Frontend registry** (`app/client/domains/formats/`) — drives the upload widget's `accept`, MIME filter, drop text, rejection toast, and the blob viewer's render choice (iframe vs img).

Dependency graph (no cycles):

```
internal/state  ←──  internal/format  ←──  internal/workflow  ←──  internal/classifications
                           ↑
                     internal/documents
```

**Import aliasing on collision.** `internal/workflow/enhance.go` imports both `github.com/tailored-agentic-units/format` (for `format.Image`) and Herald's new `internal/format` package. Alias the external tau package — `tauformat "github.com/tailored-agentic-units/format"` — so Herald's `format` keeps its natural identifier.

## Implementation

### Step 1: Extract shared types to `internal/state/`

Create `internal/state/state.go`:

```go
// Package state defines the shared types that Herald's format handlers and
// classification workflow both consume. It lives as a leaf package so higher
// layers (format, workflow) can import it without cycles.
package state

import "slices"

const (
	KeyDocumentID = "document_id"
	KeyTempDir    = "temp_dir"
	KeyFilename   = "filename"
	KeyPageCount  = "page_count"
	KeyClassState = "classification_state"
)

type Confidence string

const (
	ConfidenceHigh   Confidence = "HIGH"
	ConfidenceMedium Confidence = "MEDIUM"
	ConfidenceLow    Confidence = "LOW"
)

// EnhanceSettings captures the semantic intent of a page-level enhancement
// pass — what kind of visual adjustment to apply — without encoding how any
// particular rasterizer consumes them. Format handlers translate the fields
// into their own tool-specific arguments.
type EnhanceSettings struct {
	Brightness *int `json:"brightness,omitempty"`
	Contrast   *int `json:"contrast,omitempty"`
	Saturation *int `json:"saturation,omitempty"`
}

type ClassificationPage struct {
	PageNumber    int              `json:"page_number"`
	ImagePath     string           `json:"image_path"`
	MarkingsFound []string         `json:"markings_found"`
	Rationale     string           `json:"rationale"`
	Enhancements  *EnhanceSettings `json:"enhancements,omitempty"`
}

func (p *ClassificationPage) Enhance() bool {
	return p.Enhancements != nil
}

type ClassificationState struct {
	Classification string               `json:"classification"`
	Confidence     Confidence           `json:"confidence"`
	Rationale      string               `json:"rationale"`
	Pages          []ClassificationPage `json:"pages"`
}

func (s *ClassificationState) NeedsEnhance() bool {
	return slices.ContainsFunc(s.Pages, func(p ClassificationPage) bool {
		return p.Enhance()
	})
}

func (s *ClassificationState) EnhancePages() []int {
	var indices []int
	for i, p := range s.Pages {
		if p.Enhance() {
			indices = append(indices, i)
		}
	}
	return indices
}
```

### Step 2: Shrink `internal/workflow/types.go` to `WorkflowResult`

Replace the entire file contents:

```go
package workflow

import (
	"time"

	"github.com/google/uuid"

	"github.com/JaimeStill/herald/internal/state"
)

// WorkflowResult bundles the final classification output with the execution
// metadata that the workflow engine produced.
type WorkflowResult struct {
	DocumentID  uuid.UUID                 `json:"document_id"`
	Filename    string                    `json:"filename"`
	PageCount   int                       `json:"page_count"`
	State       state.ClassificationState `json:"state"`
	CompletedAt time.Time                 `json:"completed_at"`
}
```

No type aliases. Every consumer references `state.*` directly — this requires the collision-resolving alias sweep in Step 2b.

### Step 2b: Alias tau's `orchestrate/state` as `taustate` across the workflow package

Five workflow files currently import `github.com/tailored-agentic-units/orchestrate/state` as the bare `state` identifier. Herald's new `internal/state` package collides with that name. Per the import-aliasing convention (alias the external on collision), update each file's import block:

**Affected files:** `init.go`, `enhance.go`, `classify.go`, `finalize.go`, `workflow.go`, `observer.go`

> `observer.go` has no collision on its own (it only imports tau's state for `state.EventNodeStart` / `state.EventNodeComplete`), but aliasing it along with the others keeps package-internal references uniform — a reader scanning the workflow package shouldn't have to remember which file uses which `state.` identifier.

Change every occurrence of:

```go
"github.com/tailored-agentic-units/orchestrate/state"
```

to:

```go
taustate "github.com/tailored-agentic-units/orchestrate/state"
```

Then update references within each file:

- `state.State` → `taustate.State`
- `state.StateNode` → `taustate.StateNode`
- `state.StateGraph` → `taustate.StateGraph`
- `state.NewFunctionNode` → `taustate.NewFunctionNode`
- `state.NewGraphWithDeps` → `taustate.NewGraphWithDeps`
- `state.New(nil)` → `taustate.New(nil)`
- `state.Not(...)` → `taustate.Not(...)`
- `state.EventNodeStart` / `state.EventNodeComplete` (observer.go only) → `taustate.EventNodeStart` / `taustate.EventNodeComplete`

(Steps 8 and 9 already provide the fully rewritten `init.go` / `enhance.go` with `taustate` in place — apply this step to `classify.go`, `finalize.go`, `workflow.go`, `observer.go`.)

Add `"github.com/JaimeStill/herald/internal/state"` to the import block of every file that touches Herald state types — that's `classify.go`, `finalize.go`, `workflow.go`, `prompts.go`. (`observer.go` doesn't use Herald state types; just swap its tau alias.)

### Step 2c: Prefix state-owned identifiers across the workflow package

After 2b, each of the five files (plus `prompts.go`) will compile-error on every bare reference to a type or constant that has moved to `state`. Mechanically prefix them:

**`classify.go`** — `*EnhanceSettings` → `*state.EnhanceSettings`; `*ClassificationState` → `*state.ClassificationState`; `ClassificationState` (in type assertion) → `state.ClassificationState`; `KeyClassState` → `state.KeyClassState`; `*ClassificationPage` → `*state.ClassificationPage`.

**`finalize.go`** — `Confidence Confidence` (struct field type) → `Confidence state.Confidence`; `*ClassificationState` → `*state.ClassificationState`; `KeyClassState` → `state.KeyClassState`. The `cs.Confidence` field access stays unqualified (it's a field, not a package reference).

**`workflow.go`** — every `KeyDocumentID`, `KeyTempDir`, `KeyFilename`, `KeyPageCount`, `KeyClassState` → `state.*`; `ClassificationState` (type assertion) → `state.ClassificationState`.

**`prompts.go`** — `ComposePrompt`'s `state *ClassificationState` parameter shadows the new `state` package. Rename the parameter to `cs` and update the two references inside the function body (`if state != nil` → `if cs != nil`; `json.MarshalIndent(state, ...)` → `json.MarshalIndent(cs, ...)`). Then `*ClassificationState` → `*state.ClassificationState`. Add `"github.com/JaimeStill/herald/internal/state"` to the import block.

**`init.go` / `enhance.go`** — already handled in Steps 8 and 9; the code blocks there use `state.*` / `taustate.*` as written.

### Step 2d: Update `classifications/repository.go` call site

One line at `internal/classifications/repository.go:316`:

```go
func collectMarkings(pages []state.ClassificationPage) []string {
```

Add `"github.com/JaimeStill/herald/internal/state"` to the file's import block.

### Step 3: Create the format package core — `internal/format/format.go`

```go
// Package format defines the document-format abstraction for Herald's
// classification workflow. Handlers know how to extract per-page PNGs from
// a format-specific source (PDF, raw image, etc.) and how to re-render a
// page with enhancement filters applied. The Registry maps MIME types to
// handlers and is composed explicitly at startup — no init-time registration.
package format

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/JaimeStill/herald/internal/state"
)

var ErrUnsupportedFormat = errors.New("unsupported document format")

type Handler interface {
	ID() string
	ContentTypes() []string
	Extract(ctx context.Context, src SourceReader, tempDir string) ([]state.ClassificationPage, error)
	Enhance(ctx context.Context, tempDir string, page *state.ClassificationPage, settings *state.EnhanceSettings) (string, error)
}

type SourceReader interface {
	Open(ctx context.Context) (io.ReadCloser, error)
	ContentType() string
	Filename() string
}

// Registry maps MIME content types to the Handler that services them. The
// handlers map is the single source of truth; SupportedContentTypes derives
// a deterministic list by sorting its keys so the content type is the only
// piece of state the registry tracks.
type Registry struct {
	handlers map[string]Handler
}

func NewRegistry(handlers ...Handler) *Registry {
	r := &Registry{handlers: make(map[string]Handler)}
	for _, h := range handlers {
		for _, ct := range h.ContentTypes() {
			r.handlers[ct] = h
		}
	}
	return r
}

func (r *Registry) Lookup(contentType string) (Handler, error) {
	h, ok := r.handlers[contentType]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedFormat, contentType)
	}
	return h, nil
}

func (r *Registry) SupportedContentTypes() []string {
	out := make([]string, 0, len(r.handlers))
	for ct := range r.handlers {
		out = append(out, ct)
	}
	slices.Sort(out)
	return out
}

```

(The bounded-concurrency helper lives in `pkg/core/workers.go` — see Step 10. Both `pdf.go` and the workflow nodes call `core.WorkerCount(n)`.)

### Step 4: Shared `magick` helper — `internal/format/imagemagick.go`

```go
package format

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/JaimeStill/herald/internal/state"
)

// Render invokes `magick` to convert src → dst. When density is true, passes
// `-density 300` (required for PDF rasterization). When settings is non-nil,
// applies brightness/contrast and/or saturation filters. Errors include the
// magick stderr for diagnostics.
func Render(ctx context.Context, src, dst string, density bool, settings *state.EnhanceSettings) error {
	args := make([]string, 0, 8)
	if density {
		args = append(args, "-density", "300")
	}
	args = append(args, src)

	if settings != nil {
		if bc, ok := brightnessContrastArg(settings); ok {
			args = append(args, "-brightness-contrast", bc)
		}
		if settings.Saturation != nil {
			args = append(args, "-modulate", fmt.Sprintf("100,%d,100", *settings.Saturation))
		}
	}

	args = append(args, dst)

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "magick", args...)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("magick %s: %w: %s", src, err, stderr.String())
	}
	return nil
}

// brightnessContrastArg translates EnhanceSettings into the paired
// `brightness,contrast` argument that ImageMagick's -brightness-contrast
// operator expects. The operator takes a single paired value, so either
// component being set is enough to emit the argument; the unset side
// defaults to 0 (no change). Returns ("", false) when neither is set.
// The paired form is kept (vs. two separate invocations) so the combined
// math runs in a single pixel pass.
func brightnessContrastArg(s *state.EnhanceSettings) (string, bool) {
	if s.Brightness == nil && s.Contrast == nil {
		return "", false
	}
	b, c := 0, 0
	if s.Brightness != nil {
		b = *s.Brightness
	}
	if s.Contrast != nil {
		c = *s.Contrast
	}
	return fmt.Sprintf("%d,%d", b, c), true
}

// pdfPageSelector formats a PDF page selector for magick (zero-indexed).
func pdfPageSelector(src string, page int) string {
	return src + "[" + strconv.Itoa(page-1) + "]"
}
```

### Step 5: PDF handler — `internal/format/pdf.go`

```go
package format

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"golang.org/x/sync/errgroup"

	"github.com/JaimeStill/herald/internal/state"
	"github.com/JaimeStill/herald/pkg/core"
)

const sourcePDF = "source.pdf"

type pdfHandler struct{}

// NewPDFHandler returns a Handler that accepts application/pdf, counts pages
// via pdfcpu, and renders each page to PNG via magick's native PDF selector
// syntax (source.pdf[N-1]).
func NewPDFHandler() Handler { return &pdfHandler{} }

func (h *pdfHandler) ID() string             { return "pdf" }
func (h *pdfHandler) ContentTypes() []string { return []string{"application/pdf"} }

func (h *pdfHandler) Extract(ctx context.Context, src SourceReader, tempDir string) ([]state.ClassificationPage, error) {
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

func (h *pdfHandler) Enhance(ctx context.Context, tempDir string, page *state.ClassificationPage, settings *state.EnhanceSettings) (string, error) {
	pdfPath := filepath.Join(tempDir, sourcePDF)
	imgPath := filepath.Join(tempDir, fmt.Sprintf("page-%d-enhanced.png", page.PageNumber))
	if err := Render(ctx, pdfPageSelector(pdfPath, page.PageNumber), imgPath, true, settings); err != nil {
		return "", fmt.Errorf("enhance page %d: %w", page.PageNumber, err)
	}
	return imgPath, nil
}

func readAll(ctx context.Context, src SourceReader) ([]byte, error) {
	rc, err := src.Open(ctx)
	if err != nil {
		return nil, fmt.Errorf("open source: %w", err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}
	return data, nil
}
```

### Step 6: Image handler — `internal/format/image.go`

```go
package format

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JaimeStill/herald/internal/state"
)

type imageHandler struct{}

// NewImageHandler returns a Handler that accepts raw image uploads (PNG/JPEG/WEBP).
// PNG inputs are copied verbatim as page-1.png; JPEG and WEBP inputs are
// normalized to PNG via magick so downstream vision calls see uniform bytes.
func NewImageHandler() Handler { return &imageHandler{} }

func (h *imageHandler) ID() string             { return "image" }
func (h *imageHandler) ContentTypes() []string { return []string{"image/png", "image/jpeg", "image/webp"} }

func (h *imageHandler) Extract(ctx context.Context, src SourceReader, tempDir string) ([]state.ClassificationPage, error) {
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

func (h *imageHandler) Enhance(ctx context.Context, tempDir string, page *state.ClassificationPage, settings *state.EnhanceSettings) (string, error) {
	srcPath := filepath.Join(tempDir, "page-1.png")
	outPath := filepath.Join(tempDir, "page-1-enhanced.png")
	if err := Render(ctx, srcPath, outPath, false, settings); err != nil {
		return "", fmt.Errorf("enhance image: %w", err)
	}
	return outPath, nil
}

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
```

### Step 7: Wire Registry into `internal/workflow/runtime.go`

Add one field and the import:

```go
package workflow

import (
	"context"
	"log/slog"

	"github.com/tailored-agentic-units/agent"

	"github.com/JaimeStill/herald/internal/documents"
	"github.com/JaimeStill/herald/internal/format"
	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/pkg/storage"
)

type Runtime struct {
	NewAgent  func(ctx context.Context) (agent.Agent, error)
	Model     string
	Provider  string
	Storage   storage.System
	Documents documents.System
	Prompts   prompts.System
	Formats   *format.Registry
	Logger    *slog.Logger
}
```

### Step 8: Rewrite `internal/workflow/init.go`

Replace the entire file:

```go
package workflow

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	taustate "github.com/tailored-agentic-units/orchestrate/state"

	"github.com/JaimeStill/herald/internal/state"
)

// InitNode downloads the document blob, dispatches to the registered format
// handler for its content type, and stores the initial ClassificationState.
func InitNode(rt *Runtime) taustate.StateNode {
	return taustate.NewFunctionNode(func(ctx context.Context, s taustate.State) (taustate.State, error) {
		documentID, tempDir, err := extractInitState(s)
		if err != nil {
			return s, fmt.Errorf("init: %w", err)
		}

		doc, err := rt.Documents.Find(ctx, documentID)
		if err != nil {
			return s, fmt.Errorf("init: %w: %w", ErrDocumentNotFound, err)
		}

		handler, err := rt.Formats.Lookup(doc.ContentType)
		if err != nil {
			return s, fmt.Errorf("init: %w: %w", ErrRenderFailed, err)
		}

		src := &blobSource{
			rt:          rt,
			storageKey:  doc.StorageKey,
			contentType: doc.ContentType,
			filename:    doc.Filename,
		}

		pages, err := handler.Extract(ctx, src, tempDir)
		if err != nil {
			return s, fmt.Errorf("init: %w: %w", ErrRenderFailed, err)
		}

		rt.Logger.InfoContext(
			ctx, "init node complete",
			"document_id", documentID,
			"format", handler.ID(),
			"page_count", len(pages),
		)

		s = s.Set(state.KeyClassState, state.ClassificationState{Pages: pages})
		s = s.Set(state.KeyFilename, doc.Filename)
		s = s.Set(state.KeyPageCount, len(pages))

		return s, nil
	})
}

func extractInitState(s taustate.State) (uuid.UUID, string, error) {
	docIDVal, ok := s.Get(state.KeyDocumentID)
	if !ok {
		return uuid.Nil, "", fmt.Errorf("%w: missing %s in state", ErrDocumentNotFound, state.KeyDocumentID)
	}

	documentID, ok := docIDVal.(uuid.UUID)
	if !ok {
		return uuid.Nil, "", fmt.Errorf("%w: %s is not uuid.UUID", ErrDocumentNotFound, state.KeyDocumentID)
	}

	tempDirVal, ok := s.Get(state.KeyTempDir)
	if !ok {
		return uuid.Nil, "", fmt.Errorf("%w: missing %s in state", ErrRenderFailed, state.KeyTempDir)
	}

	tempDir, ok := tempDirVal.(string)
	if !ok {
		return uuid.Nil, "", fmt.Errorf("%w: %s is not string", ErrRenderFailed, state.KeyTempDir)
	}

	return documentID, tempDir, nil
}

// blobSource adapts Herald's storage system to the format.SourceReader interface.
// Open downloads the blob once per call; handlers copy bytes eagerly so no pooling is needed.
type blobSource struct {
	rt          *Runtime
	storageKey  string
	contentType string
	filename    string
}

func (b *blobSource) Open(ctx context.Context) (io.ReadCloser, error) {
	blob, err := b.rt.Storage.Download(ctx, b.storageKey)
	if err != nil {
		return nil, err
	}
	return blob.Body, nil
}

func (b *blobSource) ContentType() string { return b.contentType }
func (b *blobSource) Filename() string    { return b.filename }
```

### Step 9: Rewrite `internal/workflow/enhance.go`

> **Import aliasing.** Two external collisions resolved here. Tau's `github.com/tailored-agentic-units/format` is aliased as `tauformat` so Herald's `internal/format` keeps the natural `format` identifier. Tau's `github.com/tailored-agentic-units/orchestrate/state` is aliased as `taustate` so Herald's `internal/state` keeps the natural `state` identifier.

Replace the entire file:

```go
package workflow

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	tauformat "github.com/tailored-agentic-units/format"
	taustate "github.com/tailored-agentic-units/orchestrate/state"
	"github.com/tailored-agentic-units/protocol"

	"github.com/JaimeStill/herald/internal/format"
	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/internal/state"
	"github.com/JaimeStill/herald/pkg/core"
)

type enhanceResponse struct {
	MarkingsFound []string `json:"markings_found"`
	Rationale     string   `json:"rationale"`
}

// EnhanceNode re-renders flagged pages using the registered format handler
// for the document's content type, reclassifies each via vision, and clears
// the Enhancements flag so pages are no longer re-enhanced.
func EnhanceNode(rt *Runtime) taustate.StateNode {
	return taustate.NewFunctionNode(func(ctx context.Context, s taustate.State) (taustate.State, error) {
		cs, err := extractClassState(s)
		if err != nil {
			return s, fmt.Errorf("enhance: %w", err)
		}

		tempDir, err := extractTempDir(s)
		if err != nil {
			return s, fmt.Errorf("enhance: %w", err)
		}

		documentID, err := extractDocumentID(s)
		if err != nil {
			return s, fmt.Errorf("enhance: %w", err)
		}

		doc, err := rt.Documents.Find(ctx, documentID)
		if err != nil {
			return s, fmt.Errorf("enhance: %w: %w", ErrEnhanceFailed, err)
		}

		handler, err := rt.Formats.Lookup(doc.ContentType)
		if err != nil {
			return s, fmt.Errorf("enhance: %w: %w", ErrEnhanceFailed, err)
		}

		enhanced := cs.EnhancePages()

		if err := enhancePages(ctx, rt, handler, cs, tempDir); err != nil {
			return s, fmt.Errorf("enhance: %w", err)
		}

		rt.Logger.InfoContext(
			ctx, "enhance node complete",
			"pages_enhanced", len(enhanced),
		)

		s = s.Set(state.KeyClassState, *cs)
		return s, nil
	})
}

func enhancePages(ctx context.Context, rt *Runtime, handler format.Handler, cs *state.ClassificationState, tempDir string) error {
	enhanced := cs.EnhancePages()

	prompt, err := ComposePrompt(ctx, rt.Prompts, prompts.StageEnhance, cs)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrEnhanceFailed, err)
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(core.WorkerCount(len(enhanced)))

	for _, i := range enhanced {
		g.Go(func() error {
			if gctx.Err() != nil {
				return gctx.Err()
			}

			a, err := rt.NewAgent(gctx)
			if err != nil {
				return fmt.Errorf("page %d: create agent: %w", cs.Pages[i].PageNumber, err)
			}

			imgPath, err := handler.Enhance(gctx, tempDir, &cs.Pages[i], cs.Pages[i].Enhancements)
			if err != nil {
				return fmt.Errorf("page %d: %w", cs.Pages[i].PageNumber, err)
			}
			cs.Pages[i].ImagePath = imgPath

			imgData, err := readPageImage(imgPath)
			if err != nil {
				return fmt.Errorf("page %d: %w", cs.Pages[i].PageNumber, err)
			}

			resp, err := a.Vision(
				gctx,
				[]protocol.Message{protocol.UserMessage(prompt)},
				[]tauformat.Image{{Data: imgData, Format: "png"}},
			)
			if err != nil {
				return fmt.Errorf("page %d: vision call: %w", cs.Pages[i].PageNumber, err)
			}

			parsed, err := core.Parse[enhanceResponse](resp.Text())
			if err != nil {
				return fmt.Errorf("page %d: parse response: %w", cs.Pages[i].PageNumber, err)
			}

			cs.Pages[i].MarkingsFound = parsed.MarkingsFound
			cs.Pages[i].Rationale = parsed.Rationale
			cs.Pages[i].Enhancements = nil

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("%w: %w", ErrEnhanceFailed, err)
	}

	return nil
}

func extractTempDir(s taustate.State) (string, error) {
	val, ok := s.Get(state.KeyTempDir)
	if !ok {
		return "", fmt.Errorf("%w: missing %s in state", ErrEnhanceFailed, state.KeyTempDir)
	}

	tempDir, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("%w: %s is not string", ErrEnhanceFailed, state.KeyTempDir)
	}

	return tempDir, nil
}

func extractDocumentID(s taustate.State) (uuid.UUID, error) {
	val, ok := s.Get(state.KeyDocumentID)
	if !ok {
		return uuid.Nil, fmt.Errorf("%w: missing %s in state", ErrEnhanceFailed, state.KeyDocumentID)
	}
	id, ok := val.(uuid.UUID)
	if !ok {
		return uuid.Nil, fmt.Errorf("%w: %s is not uuid.UUID", ErrEnhanceFailed, state.KeyDocumentID)
	}
	return id, nil
}
```

### Step 10: Consolidate the bounded-concurrency helper in `pkg/core/workers.go`

The `workerCount` function currently at the bottom of `internal/workflow/workflow.go` is needed by three callers after this refactor (`internal/format/pdf.go`, `internal/workflow/classify.go`, `internal/workflow/enhance.go`). Rather than duplicating a one-liner in each, promote it to an exported helper in the existing `pkg/core` package.

Create `pkg/core/workers.go`:

```go
package core

import "runtime"

// WorkerCount returns a bounded worker-pool size for a given workload: at
// most NumCPU, at most n, and at least 1. Callers typically pass it to
// errgroup.(*Group).SetLimit when parallelizing per-page work.
func WorkerCount(n int) int {
	return max(min(runtime.NumCPU(), n), 1)
}
```

Then delete the `workerCount` function from the bottom of `internal/workflow/workflow.go` (the existing lines 163-165).

Update the remaining workflow call site in `internal/workflow/classify.go` (line 72):

```go
g.SetLimit(core.WorkerCount(len(cs.Pages)))
```

And add the import to `classify.go`:

```go
"github.com/JaimeStill/herald/pkg/core"
```

`enhance.go`'s call site and import are already handled by the Step 9 rewrite, and `pdf.go` by Step 5. No `runtime` import is needed in any workflow file.


### Step 11: Add `ErrUnsupportedContentType` to `internal/documents/errors.go`

Insert before the closing `)` of the var block, and update `MapHTTPStatus`:

```go
var (
	ErrNotFound               = errors.New("document not found")
	ErrDuplicate              = errors.New("document already exists")
	ErrFileTooLarge           = errors.New("file exceeds maximum upload size")
	ErrInvalidFile            = errors.New("invalid file")
	ErrUnsupportedContentType = errors.New("unsupported content type")
)

func MapHTTPStatus(err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrDuplicate):
		return http.StatusConflict
	case errors.Is(err, ErrFileTooLarge):
		return http.StatusRequestEntityTooLarge
	case errors.Is(err, ErrInvalidFile), errors.Is(err, ErrUnsupportedContentType):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
```

### Step 12: Add registry to documents Handler / repository

`internal/documents/handler.go` — update struct + `NewHandler` + `Upload`:

```go
import (
	// existing imports...
	"github.com/JaimeStill/herald/internal/format"
)

type Handler struct {
	sys           System
	logger        *slog.Logger
	pagination    pagination.Config
	maxUploadSize int64
	formats       *format.Registry
}

func NewHandler(
	sys System,
	logger *slog.Logger,
	pagination pagination.Config,
	maxUploadSize int64,
	formats *format.Registry,
) *Handler {
	return &Handler{
		sys:           sys,
		logger:        logger.With("handler", "documents"),
		pagination:    pagination,
		maxUploadSize: maxUploadSize,
		formats:       formats,
	}
}
```

In `Upload`, right after `contentType := detectContentType(...)`:

```go
if _, err := h.formats.Lookup(contentType); err != nil {
	supported := strings.Join(h.formats.SupportedContentTypes(), ", ")
	handlers.RespondError(w, h.logger, http.StatusBadRequest,
		fmt.Errorf("%w: %s (supported: %s)", ErrUnsupportedContentType, contentType, supported))
	return
}
```

Add `"fmt"` to the import block if not already present (`"strings"` is already imported — `handler.go:11`).

`internal/documents/repository.go` — update `New` and `Handler()`:

```go
import (
	// existing imports...
	"github.com/JaimeStill/herald/internal/format"
)

type repo struct {
	db         *sql.DB
	storage    storage.System
	logger     *slog.Logger
	pagination pagination.Config
	formats    *format.Registry
}

func New(
	db *sql.DB,
	store storage.System,
	logger *slog.Logger,
	pagination pagination.Config,
	formats *format.Registry,
) System {
	return &repo{
		db:         db,
		storage:    store,
		logger:     logger.With("system", "documents"),
		pagination: pagination,
		formats:    formats,
	}
}

func (r *repo) Handler(maxUploadSize int64) *Handler {
	return NewHandler(r, r.logger, r.pagination, maxUploadSize, r.formats)
}
```

### Step 13: Compose registry in `internal/api/domain.go`

```go
package api

import (
	"github.com/JaimeStill/herald/internal/classifications"
	"github.com/JaimeStill/herald/internal/documents"
	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/internal/format"
)

type Domain struct {
	Classifications classifications.System
	Documents       documents.System
	Prompts         prompts.System
}

func NewDomain(runtime *Runtime) *Domain {
	formats := format.NewRegistry(
		format.NewPDFHandler(),
		format.NewImageHandler(),
	)

	docsSystem := documents.New(
		runtime.Database.Connection(),
		runtime.Storage,
		runtime.Logger,
		runtime.Pagination,
		formats,
	)

	promptsSystem := prompts.New(
		runtime.Database.Connection(),
		runtime.Logger,
		runtime.Pagination,
	)

	classificationsSystem := classifications.New(
		runtime.Database.Connection(),
		runtime.NewAgent,
		runtime.Agent.Model.Name,
		runtime.Agent.Provider.Name,
		runtime.Logger,
		runtime.Pagination,
		runtime.Storage,
		docsSystem,
		promptsSystem,
		formats,
	)

	return &Domain{
		Classifications: classificationsSystem,
		Documents:       docsSystem,
		Prompts:         promptsSystem,
	}
}
```

### Step 14: Thread registry through `internal/classifications/repository.go`

Update `New` signature and `Runtime` literal:

```go
import (
	// existing imports...
	"github.com/JaimeStill/herald/internal/format"
)

func New(
	db *sql.DB,
	newAgent func(ctx context.Context) (agent.Agent, error),
	modelName string,
	providerName string,
	logger *slog.Logger,
	pagination pagination.Config,
	storage storage.System,
	docs documents.System,
	prompts prompts.System,
	formats *format.Registry,
) System {
	rt := &workflow.Runtime{
		NewAgent:  newAgent,
		Model:     modelName,
		Provider:  providerName,
		Storage:   storage,
		Documents: docs,
		Prompts:   prompts,
		Formats:   formats,
		Logger:    logger.With("workflow", "classify"),
	}
	return &repo{
		db:         db,
		rt:         rt,
		logger:     logger.With("system", "classifications"),
		pagination: pagination,
	}
}
```

### Step 15: Drop `document-context` from `go.mod`

After all source changes compile, run:

```bash
go mod tidy
```

This removes `github.com/JaimeStill/document-context` and any transitive deps only it pulled in. Review the `go.mod` / `go.sum` diff to confirm no unrelated churn.

### Step 16: Frontend — create `app/client/domains/formats/`

`app/client/domains/formats/types.ts`:

```ts
import type { TemplateResult } from "lit";

/** One supported document format. */
export interface DocumentFormat {
  id: string;
  displayName: string;
  contentTypes: string[];
  extensions: string[];
  renderViewer: (src: string, title: string) => TemplateResult;
}
```

`app/client/domains/formats/pdf.ts`:

```ts
import { html } from "lit";

import type { DocumentFormat } from "./types";

export const pdfFormat: DocumentFormat = {
  id: "pdf",
  displayName: "PDF",
  contentTypes: ["application/pdf"],
  extensions: [".pdf"],
  renderViewer: (src, title) => html`<iframe src=${src} title=${title}></iframe>`,
};
```

`app/client/domains/formats/image.ts`:

```ts
import { html } from "lit";

import type { DocumentFormat } from "./types";

export const imageFormat: DocumentFormat = {
  id: "image",
  displayName: "Image",
  contentTypes: ["image/png", "image/jpeg", "image/webp"],
  extensions: [".png", ".jpg", ".jpeg", ".webp"],
  renderViewer: (src, title) => html`<img src=${src} alt=${title} />`,
};
```

`app/client/domains/formats/registry.ts`:

```ts
import { imageFormat } from "./image";
import { pdfFormat } from "./pdf";
import type { DocumentFormat } from "./types";

export const formats: readonly DocumentFormat[] = [pdfFormat, imageFormat];

export function findFormat(contentType?: string): DocumentFormat | undefined {
  if (!contentType) return undefined;
  return formats.find((f) => f.contentTypes.includes(contentType));
}

export function isSupported(contentType?: string): boolean {
  return findFormat(contentType) !== undefined;
}

export function acceptAttribute(): string {
  return formats.flatMap((f) => f.extensions).join(",");
}

export function allSupportedContentTypes(): string[] {
  return formats.flatMap((f) => f.contentTypes);
}

export function dropZoneText(): string {
  const label = formats.map((f) => `${f.displayName}s`).join(" / ");
  return `Drag ${label} here or click to browse`;
}
```

`app/client/domains/formats/index.ts`:

```ts
export type { DocumentFormat } from "./types";
export { pdfFormat } from "./pdf";
export { imageFormat } from "./image";
export {
  formats,
  findFormat,
  isSupported,
  acceptAttribute,
  allSupportedContentTypes,
  dropZoneText,
} from "./registry";
```

> **No `tsconfig.json` change needed** — `app/tsconfig.json` already has `"@domains/*": ["./client/domains/*"]`, so `@domains/formats` resolves automatically.

### Step 17: Update `app/client/ui/elements/blob-viewer.ts`

```ts
import { LitElement, html, nothing } from "lit";
import { customElement, property } from "lit/decorators.js";

import { findFormat } from "@domains/formats";

import styles from "./blob-viewer.module.css";

/**
 * Generic inline blob viewer. Selects a renderer from the format registry
 * based on the `content-type` attribute; falls back to an iframe when no
 * handler matches so callers with generic blobs still get a reasonable view.
 */
@customElement("hd-blob-viewer")
export class BlobViewer extends LitElement {
  static styles = styles;

  @property() override title = "Blob viewer";
  @property() src?: string;
  @property({ attribute: "content-type" }) contentType?: string;

  render() {
    if (!this.src) return nothing;

    const format = findFormat(this.contentType);
    if (format) return format.renderViewer(this.src, this.title);

    return html`<iframe src=${this.src} title=${this.title}></iframe>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-blob-viewer": BlobViewer;
  }
}
```

Update `app/client/ui/elements/blob-viewer.module.css` — add an `img` rule alongside the `iframe` rule:

```css
:host {
  display: flex;
  flex: 1;
  min-height: 0;
}

iframe,
img {
  flex: 1;
  border: 1px solid var(--divider);
  border-radius: var(--radius-md);
}

img {
  object-fit: contain;
  background: var(--surface-2, transparent);
}
```

### Step 18: Update `app/client/ui/modules/document-upload.ts`

Replace the file's imports and the three touched methods:

```ts
import { acceptAttribute, dropZoneText, isSupported } from "@domains/formats";
// ...existing imports including Toast
```

`addFiles`:

```ts
private addFiles(files: FileList) {
  const all = Array.from(files);
  const accepted = all.filter((f) => isSupported(f.type));
  const rejected = all.length - accepted.length;

  if (rejected > 0) {
    Toast.warning(
      `Skipped ${rejected} unsupported file${rejected === 1 ? "" : "s"}`,
    );
  }

  if (accepted.length < 1) return;

  const entries: UploadEntry[] = accepted.map((file) => ({
    file,
    status: "pending" as const,
    externalId: 0,
    platform: "",
  }));

  this.queue = [...this.queue, ...entries];
}
```

`renderDropZone` template — replace the `accept=".pdf"` and `<span class="drop-text">...</span>` lines:

```ts
<input
  id="file-input"
  type="file"
  accept=${acceptAttribute()}
  multiple
  hidden
  @change=${this.handleFileInput}
/>
<span class="drop-icon">📄</span>
<span class="drop-text">${dropZoneText()}</span>
```

### Step 19: Update `app/client/ui/views/review-view.ts` + CSS

In `review-view.ts`, pass `content-type` to `<hd-blob-viewer>` and rename the panel class:

```ts
<div class="panel document-panel">
  <hd-blob-viewer
    .title=${this.document.filename}
    .src=${this.blobUrl}
    content-type=${this.document.content_type}
  ></hd-blob-viewer>
</div>
```

In `review-view.module.css`, rename `.pdf-panel` → `.document-panel`:

```css
.document-panel {
  flex: 3;
}
```

## Remediation

### R1: Images overflow the document panel and hide the classification panel

**Blocker discovered during Phase 5 smoke test.** Uploading an image document and opening its review view rendered the image at its intrinsic pixel dimensions (~2550×3300 for a 300 DPI scan), which pushed `.document-panel` past the viewport and stole all available width from `.classification-panel`. PDFs didn't exhibit this because `<iframe>` has a fixed default intrinsic size (300×150) that flex easily grows — `<img>` elements, by contrast, inherit their intrinsic size from the image file, and flex items default to `min-width: auto` (= don't shrink below intrinsic content size). The reset's `img { max-width: 100%; }` rule doesn't help because in a flex context `max-width` is applied *after* the flex algorithm has already sized the container based on the item's `min-width: auto`.

**Fix.** Add `min-width: 0` at every flex ancestor between the image and the viewport-sized container, breaking the intrinsic-size propagation chain.

`app/client/ui/elements/blob-viewer.module.css` — three additions:

```css
:host {
  display: flex;
  flex: 1;
  min-width: 0;
  min-height: 0;
}

iframe,
img {
  flex: 1;
  min-width: 0;
  min-height: 0;
  border: 1px solid var(--divider);
  border-radius: var(--radius-md);
}

img {
  object-fit: contain;
  background: var(--surface-2, transparent);
}
```

`app/client/ui/views/review-view.module.css` — one addition to `.panel`:

```css
.panel {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
}
```

**Rule of thumb.** When a flex container holds replaced elements with intrinsic content size (`<img>`, `<video>`, `<canvas>`), every flex ancestor between that element and the outer viewport-sized container must set `min-width: 0` (and `min-height: 0` for vertical axes). Missing any one link in the chain re-introduces the overflow.

## Validation Criteria

- [ ] `go build ./...` compiles without errors
- [ ] `go mod tidy` leaves the tree clean; `document-context` no longer in `go.mod` / `go.sum`
- [ ] `grep -rn "document-context" --include="*.go"` returns no hits
- [ ] `mise run vet` passes
- [ ] `mise run test` passes (existing tests continue to compile; test updates for the new surfaces are handled post-execution)
- [ ] `cd app && bun run build` succeeds
- [ ] Upload a PDF through the UI → classification completes, `page_count` populated
- [ ] Upload a PNG of a marked document → classification completes, `page_count` NULL, review UI renders `<img>`
- [ ] Upload a JPEG → classification completes (JPEG→PNG normalization path)
- [ ] Upload a WEBP → classification completes (WEBP→PNG normalization path)
- [ ] `curl -F "file=@x.docx" -F external_id=1 -F external_platform=HQ http://localhost:8080/api/documents` returns HTTP 400
- [ ] Dragging a `.docx` into the upload widget surfaces a warning toast and does not add it to the queue
- [ ] PDF review UI still renders inside an `<iframe>`
