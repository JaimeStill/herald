# 77 - Document Grid, View Integration, and Bulk Classify

## Context

Final sub-issue of Objective #59 (Document Management View). Assembles the full document management interface by composing all prior sub-issues: #74 (service layer), #75 (card + progress elements), #76 (upload component). This is the first time we see the documents view wired together, so we're incorporating frontend design quality improvements alongside the new code. The testing phase will focus on verifying functionality and locking in the application's look and feel.

## Architecture Decisions

### Data Fetching: `POST /api/documents/search`
The grid uses `DocumentService.search()` which POSTs a JSON body combining `PageRequest` + `Filters`. Go's `SortFields.UnmarshalJSON` accepts comma-separated strings, so `"sort": "-UploadedAt"` works in JSON bodies. This is cleaner than building complex query strings.

### No External Signals Needed
The grid manages its own state (`@state()`) for documents, pagination, filters, classifying progress, and selection. No signals/context infrastructure needed — only the grid consumes the documents list. The view refreshes the grid via `querySelector` + method call after upload completes. This follows the "no state orchestration layer" decision from #74.

### Upload Toggle
Upload section hidden behind a toggle button in the view header. Grid is the primary focus. Upload section collapses after successful upload.

### Selection State Ownership
Grid owns `selectedIds: Set<string>` as `@state()`. Card receives `selected: boolean` as `@property()` and dispatches `select` intent event. Card stays pure.

---

## Implementation Steps

### Step 1: Types and Service Updates

**`app/client/documents/document.ts`** — Add `SearchRequest` type (flat interface extending `PageRequest` with filter fields matching Go's embedded struct JSON serialization):
```typescript
export interface SearchRequest {
  page?: number;
  page_size?: number;
  search?: string;
  sort?: string;
  status?: string;
  classification?: string;
  confidence?: string;
}
```

**`app/client/documents/service.ts`** — Update `search()` to accept `SearchRequest`.

**`app/client/documents/index.ts`** — Re-export `SearchRequest`.

### Step 2: Design System Updates

**`app/client/design/core/tokens.css`** — Replace duplicated `@media (prefers-color-scheme)` blocks with `light-dark()`. The `color-scheme: dark light` declaration already enables it. Cuts color definitions in half. Supported: Chrome 123+, Firefox 120+, Safari 17.5+.

**`app/client/design/index.css`** — Update layer declaration to `@layer tokens, reset, theme, app;`.

**`app/client/design/app/app.css`** — Wrap contents in `@layer app { }`.

**Delete `app/client/design/app/elements.css`** — Empty file, only contains a comment.

**New: `app/client/design/shared/buttons.module.css`** — Extract common `.btn` styles from card and upload CSS. Add `:focus-visible` rule. Use CSS nesting.

**New: `app/client/design/shared/badges.module.css`** — Extract common `.badge` styles. Include all status variants (pending, review, complete, uploading, success, error). Use CSS nesting.

### Step 3: Update Existing Components for Shared Styles

**`app/client/elements/documents/document-card.ts`** — Import shared styles: `static styles = [sharedButtons, sharedBadges, styles]`.

**`app/client/elements/documents/document-card.module.css`** — Remove duplicated `.btn` and `.badge` blocks. Keep only component-specific variants (`.classify-btn`, `.review-btn`). Apply CSS nesting.

**`app/client/elements/documents/classify-progress.module.css`** — Wrap pulse animation in `@media (prefers-reduced-motion: no-preference)`.

**`app/client/components/documents/document-upload.ts`** — Import shared styles. Add `min="1"` to external ID input.

**`app/client/components/documents/document-upload.module.css`** — Remove duplicated `.btn` and `.badge` blocks. Keep component-specific variants. Fix focus style: replace `outline: none` with `:focus-visible` outline. Apply CSS nesting.

### Step 4: Pagination Pure Element

**New: `app/client/elements/pagination/pagination-controls.ts`**
**New: `app/client/elements/pagination/pagination-controls.module.css`**
**New: `app/client/elements/pagination/index.ts`**

Reusable pure element for page navigation. Will be reused by Prompts view and Review view.

```typescript
@customElement('hd-pagination')
export class PaginationControls extends LitElement {
  @property({ type: Number }) page = 1;
  @property({ type: Number, attribute: 'total-pages' }) totalPages = 1;

  // Dispatches 'page-change' with { page: number } detail
}
```

Template: Prev button, "Page N of M" indicator, Next button. Renders `nothing` when `totalPages <= 1`.

CSS: Centered flex row, border-top separator, shared button styles.

**Update `app/client/elements/index.ts`** — Import `./pagination`.

### Step 5: Document Card Selection

**`app/client/elements/documents/document-card.ts`** — Add `@property({ type: Boolean }) selected = false`. Add checkbox to card header. Dispatch `select` CustomEvent with document ID.

**`app/client/elements/documents/document-card.module.css`** — Add `.select-control` styles. Use native checkbox with `accent-color: var(--blue)`.

### Step 6: Router and Navigation Fixes

**`app/client/router/router.ts`**:
- **SPA link interception**: Add delegated click handler on `document` in `start()`. Intercepts same-origin `<a>` clicks (skips `_blank`, `download`, absolute URLs). Calls `navigate()` instead of allowing full page reload. This fixes the header nav links in `app.html`.
- **View Transitions API**: Progressive enhancement in `mount()`. Wrap DOM update in `document.startViewTransition()` when available. Add ambient type declaration for `startViewTransition`.

### Step 7: Document Grid Component (Core)

**New: `app/client/components/documents/document-grid.ts`**

Stateful component that is the operational heart of the documents view:

- **State**: `@state()` for page, search, status filter, sort, documents (PageResult), classifying (Map of progress), selectedIds (Set), abortControllers (Map)
- **Toolbar**: Search `<input type="search">` with 300ms debounce timer, status `<select>` (All/Pending/Review/Complete), sort `<select>` (Newest/Oldest/Name A-Z/Name Z-A), conditional bulk classify button
- **Grid**: Responsive CSS grid of `hd-document-card` elements, passing `document`, `selected`, `classifying`, `currentNode`, `completedNodes` props
- **Pagination**: Renders `<hd-pagination>` element, handles `page-change` event
- **Empty state**: Message when no documents match
- **Classify handler**: On `classify` event from card, calls `ClassificationService.classify()` SSE, tracks progress in Map, updates Map on `node.start`/`node.complete` events, removes from Map + re-fetches on `complete`/`error`
- **Bulk classify**: Iterates `selectedIds`, calls classify for each via `Promise.allSettled`, clears selection
- **Review handler**: On `review` event from card, calls `navigate('/review/${id}')`
- **Cleanup**: `disconnectedCallback` aborts all SSE controllers and clears debounce timer
- **Public `refresh()` method**: Called by view after upload completes

**New: `app/client/components/documents/document-grid.module.css`**

- Flex column layout for toolbar → grid → pagination
- `.toolbar`: flex row with gap, wrapping on narrow screens
- `.search-input`, `.filter-select`, `.sort-select`: consistent sizing with `:focus-visible`
- `.grid`: `display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr))`
- `.pagination`: centered flex with border-top separator
- `.empty-state`: centered muted text
- `.bulk-classify-btn`: blue accent variant
- CSS nesting throughout

### Step 8: Documents View Integration

**`app/client/views/documents/documents-view.ts`** — Full rewrite as composition root:
- Imports `nothing` from lit for conditional rendering
- `@state() showUpload = false` for toggling upload section
- Header with "Documents" title + "Upload" toggle button
- Conditional `<hd-document-upload>` with `@upload-complete` handler
- `<hd-document-grid>` always rendered
- Upload complete handler: hides upload, calls `grid.refresh()` via querySelector
- Import shared button styles for the upload toggle button

**`app/client/views/documents/documents-view.module.css`** — Layout styles: padded flex column, header row with justify-content: space-between, flex-growing grid.

### Step 9: Update Barrels

**`app/client/components/documents/index.ts`** — Add `DocumentGrid` export.

---

## File Summary

### New Files (7)
- `app/client/design/shared/buttons.module.css`
- `app/client/design/shared/badges.module.css`
- `app/client/elements/pagination/pagination-controls.ts`
- `app/client/elements/pagination/pagination-controls.module.css`
- `app/client/elements/pagination/index.ts`
- `app/client/components/documents/document-grid.ts`
- `app/client/components/documents/document-grid.module.css`

### Modified Files (15)
- `app/client/documents/document.ts`
- `app/client/documents/service.ts`
- `app/client/documents/index.ts`
- `app/client/design/core/tokens.css`
- `app/client/design/index.css`
- `app/client/design/app/app.css`
- `app/client/elements/documents/document-card.ts`
- `app/client/elements/documents/document-card.module.css`
- `app/client/elements/documents/classify-progress.module.css`
- `app/client/components/documents/document-upload.ts`
- `app/client/components/documents/document-upload.module.css`
- `app/client/components/documents/index.ts`
- `app/client/elements/index.ts`
- `app/client/views/documents/documents-view.ts`
- `app/client/views/documents/documents-view.module.css`
- `app/client/router/router.ts`

### Deleted Files (1)
- `app/client/design/app/elements.css`

---

## Validation Criteria

- [ ] `bun run build` passes
- [ ] `go vet ./...` passes
- [ ] Documents view loads at `/app/`
- [ ] Document list renders in responsive grid
- [ ] Search input filters documents with debounce
- [ ] Status filter dropdown works
- [ ] Sort control changes ordering
- [ ] Pagination controls navigate pages
- [ ] Single classify triggers SSE progress on card
- [ ] Bulk select + classify works with parallel SSE
- [ ] Upload completes and grid refreshes
- [ ] Header nav links work without full page reload
- [ ] Focus-visible outlines appear on keyboard navigation
- [ ] light-dark() theme switching works
- [ ] View transitions animate route changes (Chrome/Safari)
- [ ] Pulse animation respects prefers-reduced-motion
