# Issue #149 — Format-aware document processing (PDF + raw images)

## Context

Sub-issue of Objective #132 (Post-Deployment Quality of Life); the final task before Phase 5 closes and v0.5.0 tags.

Herald currently assumes every document is a PDF. The `init` node in `internal/workflow/init.go` opens the blob with `document.OpenPDF()` from `github.com/JaimeStill/document-context`, extracts pages via pdfcpu, and renders each page with ImageMagick through the same library. A downstream requirement has surfaced: Herald must natively accept raw image documents (PNG, JPEG, WEBP) — a single-"page" document whose bytes are already an image and whose security markings can be classified directly by the vision model.

Rather than ship a one-off `if contentType == "application/pdf"` branch through the workflow and web client, this task introduces a **document format abstraction** — a registry of per-format handlers on both backend and frontend — so adding a new format (DOCX, PPTX, TIFF, etc.) later becomes "drop in a handler" instead of "thread a new branch through init/enhance/upload/viewer."

The refactor also drops the `github.com/JaimeStill/document-context` dependency. Its role decomposes into two primitives already in the stack:
- `pdfcpu` for page counting — already a direct dep via `extractPDFPageCount` in `internal/documents/handler.go:191`
- ImageMagick (`magick`) for rendering — already required in `Dockerfile:17` (`apk add ... imagemagick ghostscript`)

Calling `magick` directly with its native PDF selector syntax (`source.pdf[N]`) gives the PDF handler everything it needs, so the abstraction gains a uniform rendering primitive shared by both handlers.

## Exploration findings

Herald's ingestion path is already structurally ready for multi-format documents:

- `internal/documents/document.go:20` — `ContentType` is a plain `string` column
- `internal/documents/document.go:22` — `PageCount *int` is nullable (already nil for non-PDFs)
- `internal/documents/repository.go:161` — `buildStorageKey` is format-agnostic
- `internal/workflow/classify.go` + `finalize.go` — both format-agnostic (consume `ClassificationPage` with an `ImagePath` field)
- `app/client/domains/documents/document.ts:16` — `content_type: string` is already on the model
- `app/client/domains/documents/service.ts:35` — upload form has no MIME restriction server-side

The real coupling lives in exactly four places, which is where the format abstraction slots in:
1. `internal/workflow/init.go:113-168` — hardcoded PDF render via `document-context`
2. `internal/workflow/enhance.go:85-191` — hardcoded PDF rerender via `document-context`
3. `app/client/ui/modules/document-upload.ts:47-48` and `:171` — hardcoded `.pdf` accept / `application/pdf` filter
4. `app/client/ui/elements/blob-viewer.ts:20` — iframe-only render

### Type movement — direct import, no aliases in `types.go`

The state types (`ClassificationPage`, `ClassificationState`, `EnhanceSettings`, `Confidence`) move to `internal/state/` and are referenced directly as `state.*` by every consumer. `internal/workflow/types.go` shrinks to just `WorkflowResult` (which genuinely is workflow-specific — it wraps `state.ClassificationState` with execution metadata like `CompletedAt`).

**Collateral updates:**

- Five workflow files currently import `github.com/tailored-agentic-units/orchestrate/state` as the bare `state` identifier (init, enhance, classify, finalize, workflow). Adding Herald's `internal/state` creates a package-name collision — per the import aliasing convention, the external tau package is aliased as `taustate` throughout. `state.State` / `state.NewFunctionNode` / `state.StateNode` / `state.StateGraph` → `taustate.*` in those five files.
- `prompts.go:20` — `ComposePrompt`'s `state *ClassificationState` parameter shadows the new package; rename to `cs` (matches the existing convention in `classify.go` / `finalize.go`).
- `internal/classifications/repository.go:316` — `[]workflow.ClassificationPage` → `[]state.ClassificationPage`.
- `tests/workflow/types_test.go` relocates to `tests/state/state_test.go` — the tests assert on `Enhance()`, `NeedsEnhance()`, `EnhancePages()`, and JSON round-trips, all of which are `state`-package concerns now.
- `tests/workflow/prompts_test.go` stays (it tests `ComposePrompt`), but its bare `workflow.ClassificationState` / `workflow.ClassificationPage` references become `state.*`.

### Ghostscript already present

`Dockerfile:17` — `RUN apk add --no-cache ca-certificates curl imagemagick ghostscript`. No Dockerfile change needed for the PDF render delegate; add an explicit note in the session summary.

## Design

### Package layout

```
internal/
├── state/                  # NEW — leaf package, shared by format + workflow
│   └── state.go            # ClassificationPage, ClassificationState, EnhanceSettings, Confidence, state keys
├── format/                 # NEW — handler registry, consumed by workflow + documents
│   ├── format.go           # Handler interface, Registry, SourceReader, ErrUnsupportedFormat
│   ├── pdf.go              # pdfHandler (pdfcpu page count + magick render)
│   ├── image.go            # imageHandler (PNG passthrough; JPEG/WEBP normalize via magick)
│   └── imagemagick.go      # Render helper: shared magick exec primitive
└── workflow/
    ├── init.go             # dispatcher — replaces renderPages / downloadPDF
    ├── enhance.go          # dispatcher — replaces rerender / enhancePages
    ├── classify.go         # unchanged
    ├── finalize.go         # unchanged
    ├── workflow.go         # unchanged
    ├── runtime.go          # Runtime gains Formats *format.Registry
    ├── types.go            # type aliases re-exporting state types
    ├── prompts.go          # unchanged
    ├── errors.go           # unchanged
    └── observer.go         # unchanged
```

### Dependency graph (no cycles)

```
internal/state  ←── internal/format  ←── internal/workflow  ←── internal/classifications
                          ↑
                    internal/documents
```

`state` has no Herald imports. `format` imports only `state`. `workflow` imports both. `documents` imports `format` directly for upload validation — no more routing through a workflow sub-package.

### Import aliasing on collision

`internal/workflow/enhance.go` currently imports `github.com/tailored-agentic-units/format` for `format.Image`. After this refactor it also imports Herald's new `internal/format`. Alias the **external** tau package (`tauformat "github.com/tailored-agentic-units/format"`) and let Herald's `format` keep its natural identifier.

### Backend — `internal/state/state.go`

Moves the types verbatim from `internal/workflow/types.go`. No behavioral change. New file owns:

- `KeyDocumentID`, `KeyTempDir`, `KeyFilename`, `KeyPageCount`, `KeyClassState`
- `Confidence` + `ConfidenceHigh|Medium|Low`
- `EnhanceSettings` — semantic intent only (brightness/contrast/saturation as optional integers); tool-specific argument formatting lives in the consumer (e.g. `internal/format/imagemagick.go`), not on this type.
- `ClassificationPage` + `Enhance()`
- `ClassificationState` + `NeedsEnhance()` + `EnhancePages()`

### Backend — `internal/format/format.go`

```go
package format

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

type Registry struct {
    handlers map[string]Handler
}

func NewRegistry(handlers ...Handler) *Registry
func (r *Registry) Lookup(contentType string) (Handler, error)            // returns ErrUnsupportedFormat
func (r *Registry) SupportedContentTypes() []string                       // derived from sorted map keys

var ErrUnsupportedFormat = errors.New("unsupported document format")
```

### Backend — `internal/format/imagemagick.go`

```go
// Render invokes `magick` with an optional density flag and optional enhancement filters.
// pdfHandler passes src="<tempDir>/source.pdf[N-1]" with density=true.
// imageHandler passes src="<tempDir>/page-1.png" (or source extension) with density=false.
// settings == nil applies no adjustments (initial render).
func Render(ctx context.Context, src, dst string, density bool, settings *state.EnhanceSettings) error
```

Arg order: `[-density 300] <src> [-brightness-contrast B,C] [-modulate 100,S,100] <dst>`. Uses `exec.CommandContext` so workflow cancellation propagates. Errors wrap stderr. The paired `-brightness-contrast` argument is assembled by an unexported helper `brightnessContrastArg(*state.EnhanceSettings) (string, bool)` in the same file — keeping the magick-specific formatting adjacent to `Render` while leaving `EnhanceSettings` itself tool-agnostic for future handlers.

### Backend — `internal/format/pdf.go`

- **ID**: `"pdf"`. **ContentTypes**: `[]string{"application/pdf"}`.
- **Extract**:
  1. Open `src` → copy to `<tempDir>/source.pdf`, close source.
  2. `pageCount, _ := api.PageCount(bytes.NewReader(...), nil)` — import `github.com/pdfcpu/pdfcpu/pkg/api` (already a direct dep).
  3. `errgroup` bounded by `core.WorkerCount(pageCount)` (shared helper in `pkg/core/workers.go` — see below). For each page `N`, invoke `Render(ctx, "<tempDir>/source.pdf["+strconv.Itoa(N-1)+"]", "<tempDir>/page-N.png", true, nil)`.
  4. Return `[]state.ClassificationPage{ {PageNumber: N, ImagePath: "<tempDir>/page-N.png"} }`.
- **Enhance**: `Render(ctx, "<tempDir>/source.pdf["+strconv.Itoa(page.PageNumber-1)+"]", "<tempDir>/page-N-enhanced.png", true, settings)` → return new path.

### Backend — `internal/format/image.go`

- **ID**: `"image"`. **ContentTypes**: `[]string{"image/png", "image/jpeg", "image/webp"}`.
- **Extract**:
  1. Open `src`. If `src.ContentType() == "image/png"`, copy bytes directly to `<tempDir>/page-1.png`.
  2. Else copy to `<tempDir>/source.<ext>` (derive ext from content type: `.jpg`, `.webp`) and `Render(ctx, "<tempDir>/source.<ext>", "<tempDir>/page-1.png", false, nil)`.
  3. Return single-element `[]state.ClassificationPage{ {PageNumber: 1, ImagePath: "<tempDir>/page-1.png"} }`.
- **Enhance**: `Render(ctx, "<tempDir>/page-1.png", "<tempDir>/page-1-enhanced.png", false, settings)` → return new path. (Re-applies from the already-normalized PNG, not the original; avoids re-fetching the blob.)

### Backend — `internal/workflow/init.go` (rewrite)

Becomes ~30 lines. State extraction stays. The node now:

1. Resolve `documentID` + `tempDir` from state.
2. `doc, err := rt.Documents.Find(ctx, documentID)`.
3. `handler, err := rt.Formats.Lookup(doc.ContentType)` — unsupported content types should never reach here thanks to upload validation, but return `fmt.Errorf("%w: %w", ErrRenderFailed, err)` as a defense-in-depth safeguard.
4. Build a `SourceReader` that wraps `rt.Storage.Download(ctx, doc.StorageKey)` via a small struct local to `init.go`. Its `Open` can be lazy (single-use), `ContentType` returns `doc.ContentType`, `Filename` returns `doc.Filename`.
5. `pages, err := handler.Extract(ctx, src, tempDir)`.
6. Set `KeyClassState`, `KeyFilename`, `KeyPageCount`. Done.

### Backend — `internal/workflow/enhance.go` (rewrite)

Simplifies dramatically. Current `enhancePages` opens the PDF inside the per-page goroutine. After the rewrite:

1. Extract `cs`, `tempDir`, `doc` (via `rt.Documents.Find`).
2. `handler, err := rt.Formats.Lookup(doc.ContentType)`.
3. Compose prompt once (same as today).
4. `errgroup` across `cs.EnhancePages()`. Each goroutine:
   - `a, _ := rt.NewAgent(gctx)`
   - `imgPath, err := handler.Enhance(gctx, tempDir, &cs.Pages[i], cs.Pages[i].Enhancements)`
   - `cs.Pages[i].ImagePath = imgPath`
   - Read bytes, call Vision, parse response (unchanged from here down), clear `Enhancements`.

`buildEnhanceConfig` and `rerender` are deleted. The `sourcePDF` constant in `init.go` moves into the PDF handler since nothing else references it.

### Backend — `internal/workflow/runtime.go`

Add one field:

```go
type Runtime struct {
    // existing fields...
    Formats   *format.Registry
}
```

### Backend — `internal/workflow/workflow.go`

`workerCount` is needed by three callers after this refactor (`internal/format/pdf.go`, `internal/workflow/classify.go`, `internal/workflow/enhance.go`). Rather than three copies, promote it to an exported `core.WorkerCount(n int) int` in `pkg/core/workers.go`. Delete the original from `workflow.go`.

### Backend — `internal/workflow/types.go`

Shrinks to just `WorkflowResult`:

```go
package workflow

import (
    "time"

    "github.com/google/uuid"

    "github.com/JaimeStill/herald/internal/state"
)

// WorkflowResult bundles the final classification output with the execution
// metadata that the workflow engine produced. The state-owned fields live on
// state.ClassificationState; this struct layers CompletedAt and surface metadata.
type WorkflowResult struct {
    DocumentID  uuid.UUID                  `json:"document_id"`
    Filename    string                     `json:"filename"`
    PageCount   int                        `json:"page_count"`
    State       state.ClassificationState  `json:"state"`
    CompletedAt time.Time                  `json:"completed_at"`
}
```

All state-owned identifiers are referenced directly by callers as `state.ClassificationState`, `state.KeyClassState`, etc. No re-export aliases.

### Backend — upload validation

`internal/documents/handler.go` gains a single guard in `Upload` after `detectContentType`:

```go
if _, err := h.registry.Lookup(contentType); err != nil {
    supported := strings.Join(h.registry.SupportedContentTypes(), ", ")
    handlers.RespondError(w, h.logger, http.StatusBadRequest,
        fmt.Errorf("%w: %s (supported: %s)", ErrUnsupportedContentType, contentType, supported))
    return
}
```

Plumbing:

- Add new sentinel `ErrUnsupportedContentType = errors.New("unsupported content type")` to `internal/documents/errors.go`; map it to 400 in `MapHTTPStatus`.
- `Handler` struct gains `registry *format.Registry`; `NewHandler` accepts it as a new arg.
- `documents.New()` in `repository.go` accepts the registry and passes it to `NewHandler` via the `Handler()` method. This means `documents.System.Handler()` signature can stay `Handler(maxUploadSize int64) *Handler` — the registry is stored on the repo struct and injected at handler creation time.
- `tests/documents/handler_test.go` — `mockSystem.Handler` and `newTestHandler` both call `documents.NewHandler`. Update both to pass a registry built from the real format handlers (a tiny registry built in the test helper, since no mocking is needed).

### Backend — composition root

`internal/api/domain.go:18-23` — `documents.New(...)` gains a trailing `registry` arg.

`internal/api/domain.go:31-41` — `classifications.New(...)` already builds the workflow Runtime. Update the Runtime literal to set `Formats: registry`.

The registry is constructed once in `internal/api/domain.go` at the top of `NewDomain`:

```go
registry := format.NewRegistry(
    format.NewPDFHandler(),
    format.NewImageHandler(),
)
```

No change to `infrastructure.Infrastructure` — format handling is an API-domain concern, not a service-wide one, and matches where `documents.System` / `classifications.System` are already composed.

### Backend — `go.mod` / `go.sum`

Remove `github.com/JaimeStill/document-context v0.1.1` after code changes compile. Run `go mod tidy` to drop transitive deps that only `document-context` pulled in.

### Frontend — `app/client/domains/formats/`

New module directory:

```
app/client/domains/formats/
├── types.ts        # DocumentFormat interface
├── pdf.ts          # pdfFormat constant
├── image.ts        # imageFormat constant
├── registry.ts     # formats array + findFormat / isSupported / acceptAttribute / allSupportedContentTypes
└── index.ts        # re-exports
```

```ts
// types.ts
import type { TemplateResult } from "lit";

export interface DocumentFormat {
  id: string;                 // "pdf" | "image"
  displayName: string;        // "PDF" | "Image"
  contentTypes: string[];     // MIME types
  extensions: string[];       // file extensions including the dot (".pdf", ".png", ...)
  renderViewer: (src: string, title: string) => TemplateResult;
}
```

```ts
// registry.ts
import { pdfFormat } from "./pdf";
import { imageFormat } from "./image";

export const formats: DocumentFormat[] = [pdfFormat, imageFormat];

export function findFormat(contentType?: string): DocumentFormat | undefined { ... }
export function isSupported(contentType?: string): boolean { ... }
export function acceptAttribute(): string {              // e.g. ".pdf,.png,.jpg,.jpeg,.webp"
    return formats.flatMap(f => f.extensions).join(",");
}
export function allSupportedContentTypes(): string[] { ... }
export function dropZoneText(): string {                 // "Drag PDFs / Images here or click to browse"
    return `Drag ${formats.map(f => f.displayName + "s").join(" / ")} here or click to browse`;
}
```

Adding the module means also updating `app/tsconfig.json` path mappings if a `@domains/formats` alias is used (check against existing `@domains/*` pattern — Herald already uses this). Likely a one-line entry mirroring existing domain aliases.

### Frontend — `blob-viewer.ts`

```ts
@property() override title = "Blob viewer";
@property() src?: string;
@property({ attribute: "content-type" }) contentType?: string;

render() {
  if (!this.src) return nothing;
  const format = findFormat(this.contentType);
  if (format) return format.renderViewer(this.src, this.title);
  return html`<iframe src=${this.src} title=${this.title}></iframe>`;   // generic fallback
}
```

`blob-viewer.module.css` gains `img { flex: 1; object-fit: contain; border: 1px solid var(--divider); border-radius: var(--radius-md); }` alongside the existing iframe rule.

### Frontend — `document-upload.ts`

- `renderDropZone` template: `accept=${acceptAttribute()}` and `<span class="drop-text">${dropZoneText()}</span>`.
- `addFiles`: partition `Array.from(files)` into `accepted = files.filter(f => isSupported(f.type))` and `rejected = files.filter(f => !isSupported(f.type))`. If `rejected.length > 0`, `Toast.warning('Skipped N unsupported file(s). Supported formats: PDF, Image (PNG/JPEG/WEBP).')` (or use the formats list to build the message).
- Rename local variable `pdfs` to `accepted`; drop the `.pdf` extension dependency.

### Frontend — `review-view.ts` + CSS

- Template: `<hd-blob-viewer .title=${this.document.filename} .src=${this.blobUrl} content-type=${this.document.content_type}></hd-blob-viewer>`.
- Rename `.pdf-panel` → `.document-panel` in `review-view.ts:94` and `review-view.module.css:15`.

## Testing

Unit / integration tests under `tests/`:

- `tests/format/registry_test.go` — NEW. Table-driven: Lookup hits, Lookup miss returns `ErrUnsupportedFormat`, `SupportedContentTypes` returns deterministic order.
- `tests/format/pdf_test.go` — NEW. Uses a fixture from `_project/marked-documents/` (e.g. `single-unclassified.pdf`). Extract produces N PNGs on disk. Enhance with non-nil settings produces a `page-N-enhanced.png`. Skip gracefully if `magick` is not on PATH (use `exec.LookPath`; `t.Skip` with explanation — Herald has no coverage target but failing in CI without magick would be churn).
- `tests/format/image_test.go` — NEW. Three subtests: PNG passthrough (byte-identical copy), JPEG→PNG normalization (output is a valid PNG), WEBP→PNG normalization. Enhance applies filters. Magick-skip guard as above.
- `tests/documents/handler_test.go` — UPDATE. Add a subtest that uploads a `.docx` (or any unknown MIME) and asserts HTTP 400 + the error body. Update `mockSystem.Handler` and `newTestHandler` to build a real 2-handler `format.Registry` (no mocking — the registry is a simple value type with no side effects).
- `tests/state/state_test.go` — RELOCATED from `tests/workflow/types_test.go`. The tests are verifying `state`-owned behavior (`Enhance()`, `NeedsEnhance()`, `EnhancePages()`, JSON round-trip, Confidence constants). File moves; bare `workflow.*` references become `state.*`.
- `tests/workflow/prompts_test.go` — stays (tests `ComposePrompt`). Sweep bare `workflow.ClassificationState` / `workflow.ClassificationPage` → `state.*`.

Integration tests for `init`/`enhance` dispatch are deferred — they would require mocking `magick` + storage + the full workflow graph, yielding low value per dollar of test complexity. The smoke-test checklist covers these dispatch paths end-to-end.

Smoke-test checklist (local stack via `docker compose up -d` + `mise run dev`):

1. Upload PDF regression: existing flow, page_count populated, classify succeeds.
2. Upload PNG of a marked document: `content_type = image/png`, `page_count = NULL`, workflow succeeds.
3. Upload JPEG: JPEG → PNG normalization path.
4. Upload WEBP: WEBP → PNG normalization path.
5. Reject unsupported upload (`.docx`): HTTP 400, body cites supported types.
6. Review UI for image doc: `<hd-blob-viewer>` renders `<img>`.
7. Review UI for PDF doc: still renders `<iframe>`.
8. Enhance path for image: force `Enhance: true` via a doctored fixture; verify `page-1-enhanced.png` written.
9. Drag a `.docx` onto the upload widget: toast surfaces, no queue entry.
10. `magick -list delegate` inside the container: confirm `ps:alpha => gs` present (Dockerfile already installs ghostscript — verification only).

Automated: `mise run test` passes. `mise run vet` passes. `go mod tidy` leaves the tree clean.

## Critical files

### Backend (new)

- `internal/state/state.go`
- `internal/format/format.go`
- `internal/format/pdf.go`
- `internal/format/image.go`
- `internal/format/imagemagick.go`

### Backend (modified)

- `internal/workflow/init.go` — rewrite as registry dispatcher
- `internal/workflow/enhance.go` — rewrite as registry dispatcher
- `internal/workflow/types.go` — shrink to type aliases + `WorkflowResult`
- `internal/workflow/runtime.go` — `Runtime.Formats *format.Registry`
- `internal/workflow/workflow.go` — drop `workerCount` (promoted to `pkg/core/workers.go`)
- `pkg/core/workers.go` (new) — exported `WorkerCount(n int) int`
- `internal/documents/handler.go` — content-type allowlist, registry on Handler
- `internal/documents/repository.go` — `New()` accepts registry, forwards to `NewHandler`
- `internal/documents/errors.go` — add `ErrUnsupportedContentType`
- `internal/api/domain.go` — construct registry, pass into `documents.New` + workflow Runtime
- `go.mod` / `go.sum` — remove `document-context`

### Frontend (new)

- `app/client/domains/formats/types.ts`
- `app/client/domains/formats/pdf.ts`
- `app/client/domains/formats/image.ts`
- `app/client/domains/formats/registry.ts`
- `app/client/domains/formats/index.ts`

### Frontend (modified)

- `app/client/ui/elements/blob-viewer.ts` + `.module.css`
- `app/client/ui/modules/document-upload.ts`
- `app/client/ui/views/review-view.ts` + `.module.css`
- `app/tsconfig.json` — add `@domains/formats` alias if the codebase uses explicit per-domain aliases

### Tests

- `tests/format/registry_test.go` (new)
- `tests/format/pdf_test.go` (new)
- `tests/format/image_test.go` (new)
- `tests/documents/handler_test.go` (update — unsupported content type case + real registry in helpers)

## Existing utilities to reuse

- `github.com/pdfcpu/pdfcpu/pkg/api.PageCount` — already imported in `internal/documents/handler.go:14` for upload-time page count; reuse in `format/pdf.go`.
- `exec.CommandContext` — no helper needed; direct stdlib call in `format/imagemagick.go`.
- `runtime.NumCPU` + `min`/`max` — the existing `workerCount` function in `workflow.go`. Promote to `pkg/core/workers.go` as exported `WorkerCount`.
- `pkg/storage.System.Download` — existing blob download used by `init.go:68`.
- `app/client/ui/elements/toast.ts` — `Toast.warning(...)` for rejected-file feedback.
- `app/client/core/core.ts` — already imported by `document-upload.ts` for `formatBytes`.

## Open decisions (none blocking)

- **`workerCount` location**: promoted to `pkg/core/workers.go` as exported `WorkerCount`.
- **SourceReader lifetime**: single-use (close-on-Open-completion) is fine — the handler copies bytes eagerly. No pooling needed.

## Verification

Run in sequence:

1. `go mod tidy && go build ./...` — confirms import tree is clean after `document-context` removal.
2. `mise run vet` — static checks.
3. `mise run test` — all existing tests plus new format/registry/handler tests pass.
4. `docker compose up -d && mise run dev` — start local stack.
5. Execute the 10-item smoke-test checklist above.
6. Rebuild web client (`cd app && bun run build`) if frontend changes don't hot-reload in Air.

## Acceptance criteria (mirrors issue #149)

- [ ] Backend `internal/format/` package with Handler, Registry, PDF handler, image handler, shared magick helper
- [ ] Backend `internal/state/` owns format-shared types
- [ ] `init.go` / `enhance.go` dispatch via registry; no `document-context` imports in the tree
- [ ] `github.com/JaimeStill/document-context` removed from `go.mod` / `go.sum`
- [ ] `internal/documents/handler.go` rejects unsupported content types with HTTP 400
- [ ] Frontend `app/client/domains/formats/` module with DocumentFormat + PDF/image + registry
- [ ] `blob-viewer.ts` renders via the frontend registry (img for images, iframe for PDFs)
- [ ] `document-upload.ts` reads accept, MIME filter, drop text from the registry; rejections toast
- [ ] Uploads of PDF, PNG, JPEG, WEBP each succeed end-to-end
- [ ] Unsupported upload (`.docx`) returns 400 and is rejected by the UI
- [ ] Review UI for image renders `<img>`; PDF review unchanged
- [ ] Tests cover registry, both handlers, upload rejection
- [ ] Ghostscript present in Dockerfile (already confirmed — no change)
- [ ] `mise run test` and `mise run vet` pass
