# 149 - Format-aware document processing (PDF + raw images)

## Summary

Introduced a document-format abstraction so Herald natively accepts both PDFs and raw image uploads (PNG, JPEG, WEBP). Two new top-level packages (`internal/state/` for shared classification types, `internal/format/` for the handler registry + PDF/image implementations) replace the single-format PDF-only assumption baked into the workflow. Upload validation now rejects unsupported content types at the HTTP boundary with a 400 citing the supported set; the web client's blob viewer picks iframe or img based on a mirror registry under `app/client/domains/formats/`. The refactor also drops the `github.com/JaimeStill/document-context` dependency — its two jobs (page counting, ImageMagick rendering) are replaced by direct calls to `pdfcpu` and magick's native PDF page-selector syntax (`source.pdf[N]`). One mechanical rename promoted `pkg/formatting` → `pkg/core` since the package had accumulated helpers outside its original scope.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Package layout for shared types | `internal/state/` and `internal/format/` as top-level siblings of `internal/workflow/` | State and format aren't workflow-specific — `internal/documents/handler.go` needs `format.Registry` for upload validation, so nesting format inside workflow forced a sibling to import a sub-package. Flat layout matches Herald's existing convention (`internal/documents/`, `internal/prompts/`, `internal/classifications/`). |
| Re-export strategy in `internal/workflow/types.go` | Direct `state.*` references throughout, no type aliases | Aliases mask where the types actually live. Direct imports force a one-time `taustate`-alias sweep across the 6 workflow files (tau's `orchestrate/state` collides with Herald's `internal/state`) but leave the final code unambiguous. |
| Import aliasing on collision | Alias external packages, keep local identifiers natural | `tauformat "github.com/tailored-agentic-units/format"` and `taustate "github.com/tailored-agentic-units/orchestrate/state"` — Herald's `format` and `state` stay unaliased. Saved as a durable feedback memory. |
| Content-type validation | Allowlist at the HTTP handler with HTTP 400 citing supported types | Failing fast at ingress is cheaper than failing deep inside the classification workflow. The error body enumerates the supported set (derived from `Registry.SupportedContentTypes()`) so clients can self-correct. |
| Image normalization path | JPEG / WEBP normalized to PNG at init time; PNG passthrough | One downstream format in the pipeline. The enhance path re-applies filters from the normalized PNG, not the original source, so no second blob fetch. |
| `brightnessContrastArg` lives in `internal/format/imagemagick.go` | Not on `state.EnhanceSettings` | `EnhanceSettings` captures semantic intent (brightness/contrast/saturation adjustments); tool-specific argument formatting belongs with the consumer. Future non-magick handlers translate the same struct their own way. |
| Registry internal state | Single `handlers map[string]Handler`; `SupportedContentTypes` sorts keys on demand | Eliminated a redundant `order []Handler` slice — one source of truth, deterministic output via `slices.Sort`. |
| `WorkerCount` placement | `pkg/core/workers.go` (exported), not per-file unexported helpers | Three callers needed it after the refactor (`internal/format/pdf.go`, `internal/workflow/classify.go`, `internal/workflow/enhance.go`). One exported helper beats three duplicated one-liners. Formatting → core rename followed naturally. |
| Image panel CSS sizing | `min-width: 0` / `min-height: 0` at every flex ancestor between `<img>` and the viewport-sized container | Discovered during Phase 5 smoke testing — R1 remediation. `<iframe>` has a fixed default intrinsic size, `<img>` does not; flex items default to `min-width: auto`, so an image's natural dimensions propagate upward and burst the panel unless the cascade is explicitly broken at each level. |
| Compile-time interface assertions | Skipped per user preference | `var _ format.SourceReader = (*blobSource)(nil)` style assertions add noise without value at Herald's scale. Saved as a durable feedback memory. |

## Files Modified

### Backend (new packages)

- `internal/state/state.go` — ClassificationPage, ClassificationState, EnhanceSettings, Confidence, state keys
- `internal/format/format.go` — Handler interface, Registry, SourceReader, shared `readAll`
- `internal/format/errors.go` — ErrUnsupportedFormat
- `internal/format/imagemagick.go` — `Render` (shared magick exec) + `brightnessContrastArg`
- `internal/format/pdf.go` — pdfHandler (pdfcpu page count, errgroup-bounded rendering)
- `internal/format/image.go` — imageHandler (PNG passthrough, JPEG/WEBP → PNG normalization)
- `pkg/core/workers.go` — exported `WorkerCount`

### Backend (modified)

- `internal/workflow/{init,enhance,classify,finalize,workflow,prompts,observer,runtime,types}.go` — tau `orchestrate/state` aliased as `taustate`, Herald state imported, workflow/types.go reduced to `WorkflowResult`
- `internal/documents/handler.go` — `formats *format.Registry` field, content-type guard, error body enumerates supported types
- `internal/documents/repository.go` — `New` accepts registry and forwards to `Handler()`
- `internal/documents/errors.go` — `ErrUnsupportedContentType`, `MapHTTPStatus` returns 400
- `internal/api/domain.go` — composes `format.Registry` once, threads through documents + classifications
- `internal/classifications/repository.go` — `New` accepts registry; workflow `Runtime.Formats` set; `collectMarkings` takes `state.ClassificationPage`
- `go.mod` / `go.sum` — `github.com/JaimeStill/document-context` removed, `go mod tidy` clean

### Backend (renamed)

- `pkg/formatting/` → `pkg/core/` (4 files, 12 Go call-site updates); package godoc rewritten to describe "stateless, cross-cutting primitives" scope
- `tests/formatting/` → `tests/core/`

### Frontend (new)

- `app/client/domains/formats/format.ts` — DocumentFormat interface
- `app/client/domains/formats/pdf.ts` — pdfFormat
- `app/client/domains/formats/image.ts` — imageFormat
- `app/client/domains/formats/registry.ts` — formats list + findFormat / isSupported / acceptAttribute / allSupportedContentTypes / dropZoneText
- `app/client/domains/formats/index.ts`

### Frontend (modified)

- `app/client/ui/elements/blob-viewer.ts` — `content-type` property; `findFormat()` dispatch with iframe fallback
- `app/client/ui/elements/blob-viewer.module.css` — `img` styling with `object-fit: contain` + `min-width: 0` / `min-height: 0` cascade (R1)
- `app/client/ui/modules/document-upload.ts` — registry-driven `accept`, filter, drop-zone text, rejection toast
- `app/client/ui/views/review-view.ts` — passes `content-type` to blob viewer; `.pdf-panel` renamed to `.document-panel`
- `app/client/ui/views/review-view.module.css` — class rename + `min-width: 0` on `.panel` (R1)
- `app/client/core/formatting/bytes.ts` — godoc reference updated `pkg/formatting.FormatBytes` → `pkg/core.FormatBytes`

### Tests

- `tests/state/state_test.go` (new; relocated from `tests/workflow/types_test.go`)
- `tests/workflow/prompts_test.go` — `workflow.*` state types → `state.*`
- `tests/format/registry_test.go` (new) — Lookup hits/misses, deterministic `SupportedContentTypes` ordering
- `tests/format/pdf_test.go` (new) — Extract page count + per-page PNG, Enhance applies filter; `requireMagick` skip guard
- `tests/format/image_test.go` (new) — PNG passthrough (byte-identical), JPEG/WEBP normalization (PNG signature check), Enhance, unsupported-type error path
- `tests/documents/handler_test.go` — real `format.Registry` wired into `mockSystem.Handler` and `newTestHandler`, existing PDF fixtures prefixed with `%PDF-1.4` so `http.DetectContentType` recognizes them, new "unsupported content type returns 400" subtest

### Docs / infrastructure

- `_project/README.md` — Phase 5 status "In Progress" → "Complete" with format-aware description; package tree updated (new `internal/state/`, `internal/format/`, renamed `pkg/core/`); Classification Workflow description replaces "open PDF via document-context" with format-handler dispatch; dependency list drops `document-context`; new Key Decisions entry for the format-handler registry; vision paragraph references "tau agentic-units ecosystem" rather than retired `go-agents`
- `_project/api/documents/README.md` — Upload section documents the supported content-type set and the new 400 response with example body; added PNG and "rejected upload" curl examples
- `_project/api/documents/documents.http` — split Upload request into PDF / PNG / JPEG variants
- `_project/marked-documents/images/` (new) — rasterized page PNGs (1-27) from `marked-documents.pdf` plus one JPEG for upload fixtures
- `.claude/context/guides/.archive/149-format-aware-document-processing.md` — implementation guide archived post-execution

## Patterns Established

- **External-package aliasing on collision.** When a file imports two packages with the same base name (e.g., tau's `format` and Herald's `internal/format`), alias the *external* one. Keeps Herald identifiers natural; external aliases communicate provenance. Captured as `feedback_import_aliasing.md`.
- **Skip compile-time interface assertions by default.** `var _ Iface = (*T)(nil)` is call-site-checkable at Herald's scale. Captured as `feedback_no_interface_assertions.md`.
- **Flex + intrinsic-sized replaced elements require a `min-width: 0` cascade.** Every flex ancestor between an `<img>` / `<video>` / `<canvas>` and the viewport-sized container must set it — otherwise the element's intrinsic pixel dimensions propagate upward through `min-width: auto` defaults and burst the container. Reset-layer `img { max-width: 100% }` doesn't help because flex clamping happens after the min-width computation.
- **Format-handler registry over branching.** New document formats (DOCX, PPTX, TIFF) land as `format.Handler` implementations registered in `internal/api/domain.go`; workflow nodes and documents handler remain closed to per-format changes. Mirror convention in `app/client/domains/formats/`.

## Validation Results

- `go mod tidy` clean; `document-context` fully removed from `go.mod` / `go.sum`
- `go build ./...` passes
- `go vet ./...` passes
- `mise run test` passes (all packages, including new `tests/format` exercising real magick against `_project/marked-documents/` fixtures)
- Local smoke tests (via `docker compose up -d` + `mise run dev`) — upload regressions verified for PDF, PNG of a marked document, JPEG normalization; review UI renders img for images and iframe for PDFs; unsupported upload (plain text) returns 400 with the supported-types message.
- Image review render broken initially during Phase 5 smoke testing (intrinsic pixel dimensions overflowing the document panel); fixed with R1 `min-width: 0` cascade in blob-viewer and review-view CSS and re-verified end-to-end.
