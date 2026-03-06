# 88 — PDF Viewer Element and Storage Inline Endpoint

## Context

Sub-issue 1 of Objective 5 (Document Review View, #61). The review view needs to display PDFs inline. This requires a backend endpoint that streams blobs with `Content-Disposition: inline` and a pure Lit element that renders the PDF in an iframe.

## Implementation

### Step 1: Add `view` handler to storage (`internal/api/storage.go`)

Copy the `download` method, change `attachment` to `inline` in the `Content-Disposition` header. Register route `GET /view/{key...}` in `routes()`.

**Route registration** (add between download and find):
```go
{Method: "GET", Pattern: "/view/{key...}", Handler: h.view},
```

**Handler** — identical to `download` except line 116 uses `inline`:
```go
func (h *storageHandler) view(w http.ResponseWriter, r *http.Request) {
    // same as download but Content-Disposition: inline
}
```

### Step 2: Create `hd-pdf-viewer` element

**`app/client/ui/elements/pdf-viewer.ts`** — pure element:
- `@property() storageKey: string` — blob storage key
- Renders `<iframe src="/api/storage/view/${this.storageKey}">`
- Full-height flex layout
- No services, no state

**`app/client/ui/elements/pdf-viewer.module.css`** — styles:
- `:host` — `display: flex; flex: 1; min-height: 0`
- `iframe` — `flex: 1; border` styling with design tokens

### Step 3: Update elements barrel

Add `PdfViewer` export to `app/client/ui/elements/index.ts`.

### Step 4: Compose `hd-pdf-viewer` into review view

Update `app/client/ui/views/review-view.ts` to load the document by ID and render `hd-pdf-viewer` with its `storageKey`. The current review view is a placeholder — replace the placeholder content with a layout that fetches the document via `DocumentService.find()` and passes `document.storage_key` to `hd-pdf-viewer`.

- Add `@state() document?: Document` — loaded in `willUpdate` when `documentId` changes
- Render `hd-pdf-viewer` with the document's `storage_key` when loaded
- Show loading/error states appropriately

Update `app/client/ui/views/review-view.module.css` to use a full-height flex layout instead of the centered placeholder styles.

### Step 5: Update API Cartographer docs

Add "View Blob" section to `_project/api/storage/README.md` and corresponding request to `_project/api/storage/storage.http`.

## Files

| File | Action |
|------|--------|
| `internal/api/storage.go` | Add `view` handler + route |
| `app/client/ui/elements/pdf-viewer.ts` | New |
| `app/client/ui/elements/pdf-viewer.module.css` | New |
| `app/client/ui/elements/index.ts` | Add export |
| `app/client/ui/views/review-view.ts` | Compose pdf-viewer |
| `app/client/ui/views/review-view.module.css` | Update layout |
| `_project/api/storage/README.md` | Add View Blob section |
| `_project/api/storage/storage.http` | Add View Blob request |

## Validation

- [ ] `go vet ./...` passes
- [ ] `bun run build` succeeds
- [ ] `GET /api/storage/view/{key}` streams PDF with `Content-Disposition: inline`
- [ ] `hd-pdf-viewer` renders PDF in iframe given a storage key
- [ ] Navigate to `/app/review/:documentId` — PDF renders in the left panel
- [ ] API Cartographer docs updated
