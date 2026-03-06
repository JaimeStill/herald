# 88 — PDF Viewer Element and Storage Inline Endpoint

## Summary

Added a `GET /api/storage/view/{key...}` endpoint that streams blobs with `Content-Disposition: inline` for native browser rendering. Created `hd-blob-viewer`, a generic pure element that renders any URL in an iframe. Added `StorageService.view()` to encapsulate the view URL construction. Composed everything into the review view with document loading, error handling, and a two-panel layout.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Generic element naming | `hd-blob-viewer` with `src` prop | Not coupled to PDFs or storage routes — reusable for any inline content |
| URL construction in service | `StorageService.view(key)` | Services own the API boundary; callers don't assemble API paths |
| `title` property | `override title` on BlobViewer | Semantic pass-through to iframe `title` attribute for accessibility |
| Document loading | `willUpdate` with `changed.has("documentId")` | Fires on initial navigation and re-navigation; `@state` changes trigger re-render |

## Files Modified

- `internal/api/storage.go` — added `view` handler + route
- `app/client/ui/elements/blob-viewer.ts` — new generic blob viewer element
- `app/client/ui/elements/blob-viewer.module.css` — new blob viewer styles
- `app/client/ui/elements/index.ts` — added barrel export
- `app/client/ui/views/review-view.ts` — composed blob viewer with document loading
- `app/client/ui/views/review-view.module.css` — two-panel flex layout
- `app/client/domains/storage/service.ts` — added `view()` URL builder
- `_project/api/storage/README.md` — added View Blob endpoint docs
- `_project/api/storage/storage.http` — added View Blob request

## Patterns Established

- **Service URL builders**: Services can expose synchronous URL builder methods alongside async `Result<T>` methods. The API boundary is owned by services regardless of return type.
- **Generic blob viewer**: `hd-blob-viewer` accepts `src` and `title` — caller constructs the URL. Reusable for any inline-renderable content.
- **View data loading**: `willUpdate` + `changed.has()` for route-param-driven data fetching in view components.

## Validation Results

- `go vet ./...` — pass
- `bun run build` — pass
- `GET /api/storage/view/{key}` — streams PDF with `Content-Disposition: inline`
- Review view at `/app/review/:documentId` — PDF renders in left panel with filename and status in right panel
