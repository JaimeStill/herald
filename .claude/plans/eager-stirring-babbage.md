# Objective Planning — #59 Document Management View

## Context

Objective #59 is the third objective in Phase 3 (Web Client, v0.3.0). It builds the primary document management interface — the root view at `/app/` providing upload, browse, classify, search/filter, and SSE-powered classification progress.

**Dependencies complete:**
- #57 (Foundation): Router, design system, API utilities, build system, Go integration
- #58 (SSE): Streaming observer + SSE endpoint (2/2 sub-issues closed, needs formal close)

**Current state:** View scaffolds exist, router is configured, design tokens and API utilities (`request()`, `stream()`, `PageResult<T>`) are in place. No services, types, or components beyond empty view shells.

## Transition Closeout (Objective #58)

- Close #58 with completion comment
- Update `_project/phase.md`: mark Objective 2 status as `Complete`
- Delete `_project/objective.md` (replaces with #59 content in Step 5)

## Critical Finding: `stream()` API Gap

The existing `stream()` function in `core/api.ts` has two mismatches with the SSE classification endpoint:

1. **No POST support** — `stream()` only passes `{ signal }` to `fetch()`, but the classify endpoint is `POST /api/classifications/{documentId}`
2. **No event type parsing** — the SSE handler sends `event: <type>\ndata: <json>\n\n`, but `stream()` only reads `data:` lines, ignoring `event:` lines
3. **No `[DONE]` sentinel** — the handler closes the channel on completion (no `[DONE]`); the stream ends when the reader reports `done`

These gaps must be fixed in the first sub-issue as part of the service layer foundation.

## Sub-Issue Decomposition

### Sub-Issue 1: Document types, service layer, and stream() enhancement

**Title:** Document types, service, and SSE stream enhancement
**Labels:** `feature`, `task`
**Depends on:** —

**Scope:**
- Enhance `stream()` in `core/api.ts`: accept optional `RequestInit` for POST method; parse `event:` lines alongside `data:` lines; callback signature changes to `onEvent(type: string, data: string)` (or add typed SSE parsing)
- Create `app/client/views/documents/types.ts` — `Document`, `Classification`, `ExecutionEvent`, `DocumentStatus` TypeScript interfaces matching Go API models
- Create `app/client/views/documents/service.ts` — `DocumentService` interface with `Signal.State` signals (`documents`, `loading`, `error`, `classifyingIds`), context via `@lit/context`, factory function, CRUD methods (`list`, `upload`, `remove`), classify with SSE consumption
- Update barrel export in `app/client/views/documents/index.ts`

**Approach:**
- `stream()` enhancement: add optional `init?: RequestInit` parameter, track current `event:` line to pair with subsequent `data:` line, change `onMessage` to `onEvent(type: string, data: string)` for typed event dispatch. Keep backward compatibility or update the single existing reference.
- Types mirror Go structs: `Document` has all fields from the list endpoint (includes joined classification/confidence/classified_at). `Classification` for the full classification object. `ExecutionEvent` for SSE events.
- Service signals: `documents: Signal.State<Document[]>`, `loading: Signal.State<boolean>`, `error: Signal.State<string | null>`, `classifyingIds: Signal.State<Set<string>>` (tracks which documents have active SSE classifications)
- Service methods call `request()` and `stream()` from `@app/core`, update signals on response

**Acceptance criteria:**
- [ ] `stream()` supports POST method and event type parsing
- [ ] TypeScript types match Go API response shapes
- [ ] DocumentService compiles with all CRUD + classify methods
- [ ] Service factory creates instance with initialized signals
- [ ] Context + provider pattern ready for view consumption

---

### Sub-Issue 2: Document card and classify progress components

**Title:** Document card and classify progress pure elements
**Labels:** `feature`, `task`
**Depends on:** #1 (types)

**Scope:**
- `hd-document-card` — Pure element in `app/client/views/documents/document-card.ts`. Displays: filename, page count, status badge (pending/review/complete with color coding), classification + confidence (if exists), markings found, upload date. Action buttons: classify (dispatches `classify` CustomEvent), review (navigates to `/app/review/{id}`). Shows `hd-classify-progress` inline when classifying.
- `hd-classify-progress` — Pure element in `app/client/views/documents/classify-progress.ts`. Receives current stage name and list of completed stages via properties. Renders a stage pipeline indicator showing init → classify → enhance → finalize with completion state per stage.
- CSS module files for both components

**Approach:**
- Card uses `@property()` for Document data, boolean `classifying` flag, optional `progress` object
- Status badge: CSS classes driven by status value (pending=yellow, review=blue, complete=green)
- Classify button disabled when status !== 'pending' or already classifying
- Review button navigates via `navigate()` from `@app/router`
- Progress element is a horizontal pipeline with 4 stage indicators, current stage highlighted

**Acceptance criteria:**
- [ ] Card renders all document fields with proper formatting
- [ ] Status badge visually differentiates pending/review/complete
- [ ] Classify and review action buttons dispatch correct events
- [ ] Progress element shows stage pipeline with current/completed states
- [ ] Both components use `*.module.css` with design tokens

---

### Sub-Issue 3: Document upload component

**Title:** Document upload component with multi-file support
**Labels:** `feature`, `task`
**Depends on:** #1 (service)

**Scope:**
- `hd-document-upload` — Stateful component in `app/client/views/documents/document-upload.ts`. `<input type="file" accept=".pdf" multiple>` with styled trigger button/drop zone. Coordinates batch upload via `Promise.allSettled` — each file uploads independently via `DocumentService.upload()`. Per-file progress indicator showing filename + status (pending/uploading/success/error). Dispatches `upload-complete` event when all files finish.
- Requires `external_id` and `external_platform` fields (required by the upload endpoint) — either form inputs or sensible defaults

**Approach:**
- File selection via hidden `<input>` triggered by styled button, or drag-and-drop zone
- On file selection: display file list with per-file status badges
- Upload all files concurrently via `Promise.allSettled`, each calling `DocumentService.upload(file, externalId, externalPlatform)`
- Update per-file status as each resolves/rejects
- On all settled: dispatch `upload-complete` event, service refreshes document list
- Consumes DocumentService via `@consume`

**Acceptance criteria:**
- [ ] Multi-file PDF selection (input or drag-drop)
- [ ] Per-file upload status display (pending/uploading/success/error)
- [ ] `Promise.allSettled` coordination — one failure doesn't block others
- [ ] External ID and platform captured per upload
- [ ] Upload-complete event triggers grid refresh

---

### Sub-Issue 4: Document grid, view assembly, and bulk operations

**Title:** Document grid, view integration, and bulk classify
**Labels:** `feature`, `task`
**Depends on:** #1, #2, #3

**Scope:**
- `hd-document-grid` — Stateful component in `app/client/views/documents/document-grid.ts`. Consumes DocumentService. Renders `hd-document-card` elements in a responsive CSS grid. Toolbar with: debounced search input, status filter dropdown (all/pending/review/complete), sort control. Pagination controls (prev/next, page indicator). Handles `classify` events from cards to trigger single-document classification. Handles bulk select + bulk classify.
- Wire up `hd-documents-view` — Provide DocumentService via `@provide`. Compose `hd-document-upload` and `hd-document-grid`. Call `service.list()` on `connectedCallback`. Handle `upload-complete` event to refresh grid.
- Bulk classify: checkbox selection on cards, bulk classify button triggers parallel SSE connections via `Promise.allSettled`, each doc shows inline progress via `hd-classify-progress`

**Approach:**
- Grid uses `SignalWatcher` to reactively render when `service.documents` signal changes
- Search: debounced input (300ms) updates `PageRequest.search` and calls `service.list()`
- Status filter: dropdown updates filter param and calls `service.list()`
- Pagination: prev/next buttons update page param and call `service.list()`
- Single classify: card emits `classify` event → grid calls `service.classify(documentId)` → service adds to `classifyingIds` set → card shows progress → on complete, removes from set and refreshes list
- Bulk classify: selected document IDs → `Promise.allSettled` mapping each to `service.classify(id)`
- View is the composition root: creates service, provides it, renders upload + grid

**Acceptance criteria:**
- [ ] Responsive card grid layout
- [ ] Debounced search with real-time filtering
- [ ] Status filter dropdown
- [ ] Pagination controls (prev/next/page indicator)
- [ ] Single-document classify with inline SSE progress
- [ ] Bulk select and bulk classify
- [ ] View provides service, calls list on connect, handles upload-complete
- [ ] Full end-to-end flow: upload → browse → classify → see progress → see results

## Dependency Graph

```
#1 (Types + Service + Stream)
      |           |
      v           v
#2 (Card +    #3 (Upload)
 Progress)        |
      |           |
      v           v
#4 (Grid + View Assembly + Bulk)
```

Sub-issues #2 and #3 can proceed in parallel after #1. Sub-issue #4 depends on all three.

## Session Actions

After plan approval, execute these transition closeout and planning actions:

1. Close objective #58 with completion comment
2. Update `_project/phase.md` — mark #58 as Complete
3. Create 4 sub-issues on JaimeStill/herald with labels `feature` + `task`, milestone `v0.3.0 - Web Client`
4. Link sub-issues to objective #59 via GraphQL `addSubIssue`
5. Create `_project/objective.md` for #59
6. Add sub-issues to project board, assign to Phase 3
