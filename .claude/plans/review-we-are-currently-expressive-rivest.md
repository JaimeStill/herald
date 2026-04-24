# Format-Aware Document Support (final task for Objective #132 → v0.5.0)

## Context

Herald currently assumes every document is a PDF. The classification workflow opens the blob with `document.OpenPDF()`, extracts pages via pdfcpu, and renders each page to PNG with ImageMagick. A downstream requirement has surfaced: Herald must also natively accept raw image documents (PNG, JPEG, WEBP) — a single-"page" document whose bytes are already an image and whose security markings can be classified directly by the vision model.

Rather than ship a one-off branch for images, this task introduces a **document format abstraction** — a registry of per-format handlers on both backend and frontend — so adding a new format (DOCX, PPTX, TIFF, etc.) later becomes "drop in a handler" instead of "thread a new branch through init/enhance/upload/viewer." PDF and raw images become the two initial implementations of the abstraction.

The refactor also drops the `github.com/JaimeStill/document-context` dependency. Its role (open PDF → list pages → render with ImageMagick) decomposes into two primitives we already have in the stack: `pdfcpu` for page counting (already a direct dep via `extractPDFPageCount`) and `magick` for rendering (already a container runtime requirement). Calling `magick` directly with its native PDF selector syntax (`source.pdf[N]`) gives the PDF handler everything it needs, so the abstraction gains a uniform rendering primitive shared by both handlers.

This is the final task under Objective #132 (Post-Deployment Quality of Life). Once it lands, Phase 5 closes and we tag `v0.5.0`.

**Good news from exploration:** Herald's ingestion path is structurally ready. Content type is already a generic string column, `page_count` is nullable, storage keys are fully dynamic, the classify and finalize nodes are format-agnostic, and the web client's service layer and `Document` type already carry `content_type`. The real coupling lives in four places: the workflow's init + enhance nodes, the upload UI's MIME filter, and the blob viewer's iframe-only render — exactly the places the format abstraction slots into.

## Scope

**In scope:**
- Introduce a backend `format.Handler` abstraction with PDF + image implementations, behind a registry
- Introduce a frontend `DocumentFormat` registry with PDF + image implementations driving upload accept, MIME filtering, and viewer rendering
- Drop the `github.com/JaimeStill/document-context` dependency in favor of direct `pdfcpu` + `magick` calls
- Add a content-type allowlist on upload (today any MIME passes through); reject unsupported with 400 using the registry's supported set
- Replace the hardcoded PDF logic in `init.go` / `enhance.go` with registry dispatch
- Replace the iframe-only blob viewer with registry-driven rendering
- Update upload UI to read its accept attribute, MIME filter, and rejected-file toast from the registry
- Tests for the new abstraction and both initial handlers

**Out of scope:**
- Multi-page image formats (TIFF) — can ship later as a registered handler
- DOCX/PPTX handlers — the abstraction accommodates them but they're Phase 6+ work
- Prompt rewrites — inference stages are format-agnostic; init delivers uniform PNGs, classify/finalize see the same input regardless of source format

## Decisions locked in during planning

- **Upload validation:** allowlist + reject. Handler returns 400 for content types the registry doesn't know.
- **Image normalization:** init converts JPEG/WEBP → PNG via `magick`. Cost analysis: a typical DoD scan (~1600×2000 px) converts in ~200–500 ms of vCPU, so the worst-case 750k uploads amounts to ~60–100 vCPU-hours (~$5–18 on Container Apps Consumption). The enhance path runs `magick` anyway; threading format through state to avoid one init-time convert adds meaningful engineering surface for negligible savings.
- **Prompts:** unchanged. Init is a non-inference node whose job is to deliver uniform PNG binaries; the inference nodes are unaffected by source format.
- **Extensibility shape:** format-handler registry over ad-hoc branching. Explicit extension point for future formats (DOCX, PPTX, TIFF).
- **Dependency reduction:** `document-context` removed. Its functionality (open PDF → enumerate pages → render) is replaced with `pdfcpu` (page count) + direct `magick` calls using native PDF selector syntax (`magick source.pdf[N] page-N.png`). Both tools are already in the stack; the wrapper layer was redundant. Fewer transitive dependencies, one rendering primitive shared by both handlers.

## Design

### Backend: `internal/workflow/format/` package

New subpackage alongside the existing workflow nodes. Structure:

```
internal/workflow/
├── state/                  # NEW: format-shared types (broken out to avoid cycles)
│   └── state.go            # ClassificationPage, ClassificationState, EnhanceSettings, state keys
├── format/                 # NEW: format-handler registry
│   ├── format.go           # Handler interface, Registry, ErrUnsupportedFormat
│   ├── pdf.go              # pdfHandler (moved from init.go + enhance.go PDF logic)
│   ├── image.go            # imageHandler (new — PNG/JPEG/WEBP)
│   └── imagemagick.go      # magick exec helpers (convert-to-PNG, apply-filters)
├── init.go                 # thin dispatcher: registry.Lookup(doc.ContentType).Extract(...)
├── enhance.go              # thin dispatcher: registry.Lookup(...).Enhance(...)
├── classify.go             # unchanged (already format-agnostic)
├── finalize.go             # unchanged (already format-agnostic)
├── workflow.go
├── types.go                # re-exports state types if needed; owns Runtime + state keys
└── prompts.go
```

#### The `Handler` interface

```go
// internal/workflow/format/format.go
package format

type Handler interface {
    // ID returns a short identifier for the format ("pdf", "image").
    ID() string

    // ContentTypes returns the MIME types this handler accepts.
    ContentTypes() []string

    // Extract downloads the document blob to tempDir, renders it to one or
    // more PNG images on disk, and returns the per-page ClassificationPage
    // entries. For single-image formats, len() == 1.
    Extract(ctx context.Context, src SourceReader, tempDir string) ([]state.ClassificationPage, error)

    // Enhance re-renders a specific page with the given enhance settings.
    // For PDFs, this re-extracts from the source; for images, this re-applies
    // filters to the source image. Returns the new image path.
    Enhance(ctx context.Context, tempDir string, page *state.ClassificationPage, settings *state.EnhanceSettings) (string, error)
}

// SourceReader abstracts the "download this document" step so handlers don't
// import the storage or documents packages directly.
type SourceReader interface {
    Open(ctx context.Context) (io.ReadCloser, error)
    ContentType() string
    Filename() string
}
```

#### The registry

```go
type Registry struct {
    handlers map[string]Handler // content-type -> handler
    order    []Handler          // deterministic iteration
}

func NewRegistry(handlers ...Handler) *Registry { /* ... */ }
func (r *Registry) Lookup(contentType string) (Handler, error)
func (r *Registry) SupportedContentTypes() []string
```

Registry is constructed once in `internal/infrastructure/` alongside other subsystems and passed into the workflow Runtime. No package-level singleton / init() registration — explicit composition matches Herald's LCA pattern.

#### PDF handler (`format/pdf.go`)

- `Extract`: download blob → write `source.pdf` → `pdfcpu.PageCount` to determine page count → for each page, `magick -density 300 source.pdf[N-1] page-N.png` (concurrent, bounded via errgroup — existing `workerCount` helper) → return `[]ClassificationPage{PageNumber, ImagePath}`.
- `Enhance`: `magick -density 300 source.pdf[N-1] [-brightness-contrast B,C] [-modulate 100,S,100] page-N-enhanced.png`.

#### Image handler (`format/image.go`)

- `Extract`: download blob → if content type is `image/png`, write directly to `page-1.png`; else `magick <src> page-1.png` (via `imagemagick.go`). Return a single `ClassificationPage{PageNumber: 1, ImagePath: page-1.png}`.
- `Enhance`: `magick <source-image> -modulate 100,S,100 -brightness-contrast B,C <page-1-enhanced.png>` (via `imagemagick.go`).

#### `imagemagick.go` helper

Single primitive shared by both handlers:

```go
// Render runs: magick [-density 300] <src> [-brightness-contrast B,C] [-modulate 100,S,100] <dst>
// PDF handlers pass src="source.pdf[N]"; image handlers pass src="page-1.png".
// settings == nil applies no adjustments (init-time render).
func Render(ctx context.Context, src, dst string, settings *state.EnhanceSettings) error
```

`magick` is already required in deployment containers per `_project/README.md`. Ghostscript (required for ImageMagick's PDF delegate) is already present — confirm against the Dockerfile during implementation.

#### Content-type validation

`internal/documents/handler.go` gets a single new guard after `detectContentType`:

```go
if _, err := registry.Lookup(contentType); err != nil {
    handlers.RespondError(w, http.StatusBadRequest, fmt.Sprintf("unsupported content type: %s", contentType))
    return
}
```

The documents handler receives the registry via its `System` constructor (no global state).

### Frontend: `app/client/domains/formats/` module

```
app/client/domains/formats/
├── types.ts                # DocumentFormat interface
├── pdf.ts                  # pdfFormat
├── image.ts                # imageFormat
├── registry.ts             # findFormat, acceptAttribute, isSupported, allSupportedContentTypes, allExtensions
└── index.ts                # re-exports
```

```ts
// types.ts
export interface DocumentFormat {
  id: string;                                 // "pdf" | "image"
  displayName: string;                        // "PDF" | "Image"
  contentTypes: string[];                     // MIME types
  extensions: string[];                       // file extensions including the dot
  renderViewer: (src: string, title: string) => TemplateResult;
}

// pdf.ts
export const pdfFormat: DocumentFormat = {
  id: "pdf",
  displayName: "PDF",
  contentTypes: ["application/pdf"],
  extensions: [".pdf"],
  renderViewer: (src, title) => html`<iframe src=${src} title=${title}></iframe>`,
};

// image.ts
export const imageFormat: DocumentFormat = {
  id: "image",
  displayName: "Image",
  contentTypes: ["image/png", "image/jpeg", "image/webp"],
  extensions: [".png", ".jpg", ".jpeg", ".webp"],
  renderViewer: (src, title) => html`<img src=${src} alt=${title} />`,
};

// registry.ts
const formats: DocumentFormat[] = [pdfFormat, imageFormat];
export function findFormat(contentType: string): DocumentFormat | undefined
export function isSupported(contentType: string): boolean
export function acceptAttribute(): string                   // ".pdf,.png,.jpg,.jpeg,.webp"
export function allSupportedContentTypes(): string[]
```

#### Consumers

- `blob-viewer.ts`: accepts a `content-type` property; renders via `findFormat(this.contentType)?.renderViewer(this.src, this.title)` with a generic iframe fallback.
- `document-upload.ts`: `accept=${acceptAttribute()}`, `addFiles` filter uses `isSupported(f.type)`, drop text pulls from `formats.map(f => f.displayName).join(" / ")` ("Drag PDFs / Images here or click to browse"), rejected files surface a toast via the existing toast element.
- `review-view.ts`: passes `.content-type=${this.document?.content_type}` to `<hd-blob-viewer>`. Rename the CSS class `.pdf-panel` → `.document-panel` (cosmetic, touched anyway).

### Type / state plumbing

- `ContentType` is already persisted on the document row and exposed on the model. No schema change.
- `page_count` stays nullable; `imageHandler.Extract` doesn't populate it (handler has no opinion — the documents handler already returns `nil` for non-PDFs via the existing `extractPDFPageCount`).
- The workflow state bag already carries `KeyDocumentID`, `KeyTempDir`, `KeyClassState`, etc. No new state keys needed — the registry lookup happens at dispatch time from `doc.ContentType`.

## Critical files

### Backend
- `internal/workflow/state/state.go` (new) — move `ClassificationPage`, `ClassificationState`, `EnhanceSettings`, state key constants
- `internal/workflow/format/format.go` (new) — `Handler` interface, `Registry`, `ErrUnsupportedFormat`
- `internal/workflow/format/pdf.go` (new) — PDF handler using pdfcpu + magick directly
- `internal/workflow/format/image.go` (new) — image handler
- `internal/workflow/format/imagemagick.go` (new) — shared `magick` exec helper
- `internal/workflow/init.go` — dispatcher (drops PDF specifics, no more `document-context` imports)
- `internal/workflow/enhance.go` — dispatcher (drops PDF specifics, no more `document-context` imports)
- `internal/workflow/types.go` — re-exports from `state/`; `Runtime` gains `Formats *format.Registry`
- `internal/workflow/workflow.go` — workflow assembly receives the registry
- `internal/infrastructure/` — construct the registry during cold start, pass to workflow + documents systems
- `internal/documents/handler.go` — content-type allowlist check via registry lookup
- `internal/documents/system.go` / constructor — accept registry
- `go.mod` / `go.sum` — remove `github.com/JaimeStill/document-context`

### Web client
- `app/client/domains/formats/types.ts` (new)
- `app/client/domains/formats/pdf.ts` (new)
- `app/client/domains/formats/image.ts` (new)
- `app/client/domains/formats/registry.ts` (new)
- `app/client/domains/formats/index.ts` (new)
- `app/client/ui/elements/blob-viewer.ts` + `.module.css` — registry-driven render with iframe fallback
- `app/client/ui/modules/document-upload.ts` — registry-driven accept/filter/text + rejection toast
- `app/client/ui/views/review-view.ts` + `.module.css` — pass `content-type`; rename panel class

### Tests (`tests/`)
- `tests/workflow/format/pdf_test.go` — Extract produces N pages from a fixture PDF; Enhance applies adjustments
- `tests/workflow/format/image_test.go` — Extract produces 1 page for PNG/JPEG/WEBP; JPEG/WEBP normalize to PNG; Enhance applies filters
- `tests/workflow/format/registry_test.go` — Lookup, SupportedContentTypes, ErrUnsupportedFormat
- `tests/workflow/init_test.go` — integration: dispatches PDF and image via registry
- `tests/workflow/enhance_test.go` — integration: dispatches PDF and image via registry
- `tests/documents/handler_test.go` — rejects unsupported content type with 400 using registry

## Existing utilities to reuse

- `pkg/storage` download/view already type-agnostic (`storage.go:29`)
- `documents.CreateCommand` + repository already accept nullable `page_count` (`document.go:22`, migration `000001`)
- `workerCount(n)` in `workflow.go:163-165` handles single-page inputs correctly
- Existing `pageResponse` / `enhanceResponse` types in classify/enhance — format-agnostic, no changes
- `toast` element (`app/client/ui/elements/toast.ts`) for upload rejection feedback
- `formatting.Parse[T]` in vision response parsing — unchanged

## Verification

End-to-end smoke tests with the local stack (`docker compose up -d` for Postgres + Azurite, `mise run dev`):

1. **Upload PDF** (regression): existing flow works unchanged, `page_count` populated, classification succeeds.
2. **Upload PNG** of a marked document: content-type detected as `image/png`, `page_count = NULL`, workflow succeeds.
3. **Upload JPEG**: verifies the JPEG → PNG normalization path in `imageHandler.Extract`.
4. **Upload WEBP**: verifies WEBP → PNG normalization.
5. **Reject unsupported upload**: `curl` a `.docx`; expect 400 with clear message citing supported types.
6. **Review UI for image doc**: `<hd-blob-viewer>` renders `<img>`, not iframe.
7. **Review UI for PDF doc**: still renders `<iframe>` (registry dispatch regression).
8. **Enhance on image**: force `Enhance: true` on a local fixture; confirm `magick` invoked with adjusted parameters and `page-1-enhanced.png` is written.
9. **Rejected-file toast**: drag a `.docx` onto the upload widget; confirm the toast surfaces and no entry is added to the queue.
10. **Dockerfile / Ghostscript check**: confirm `magick -list delegate` inside the deployment container shows `ps:alpha => gs` or equivalent PDF delegate. If missing (unlikely — document-context relied on it), add `ghostscript` to the Dockerfile's apt install.

Automated: `mise run test` covers handler + format + workflow unit tests. `mise run vet` passes. `go mod tidy` after removing the document-context import.

## Post-execution (AI-owned)

- Update `_project/api/documents/README.md` + `.http` file to document the new allowlist and image-upload examples (API Cartographer).
- Update `.claude/skills/web-development/SKILL.md` with the format-registry pattern (new convention worth codifying) and its relationship to `blob-viewer` and `document-upload`.
- Update `_project/README.md` if the workflow topology section needs to mention the format dispatch layer (likely yes).
- Session summary at `.claude/context/sessions/<issue>-format-aware-documents.md` at closeout.
- CHANGELOG entry for the dev release tag (Phase 8c of task execution).

## Phase 5 closeout note

Once this task merges and the dev tag is cut, Phase 5 is complete. Release session: `/dev-workflow release v0.5.0` to consolidate dev-tag CHANGELOG sections and tag the phase release.

## Future extension sketch (not in this task)

To add a DOCX handler later:

1. `internal/workflow/format/docx.go` — `Extract` execs `soffice --headless --convert-to pdf <src>`, then delegates to `pdfHandler.Extract` on the converted PDF. `Enhance` delegates to `pdfHandler.Enhance`.
2. Register with `format.NewRegistry(pdf, image, docx)` in infrastructure.
3. `app/client/domains/formats/docx.ts` — `renderViewer` could use iframe to a server-side PDF-converted preview, or a third-party viewer.
4. Register in the frontend registry.

No changes to `init.go`, `enhance.go`, `blob-viewer.ts`, or `document-upload.ts`. That's the payoff.
