# Objective Planning: Document Review View (#61)

## Context

Objective #61 is the fifth and final objective in Phase 3 (Web Client). It builds the document review view — a side-by-side interface showing a PDF viewer alongside the classification record, with actions to validate or manually update. All backend APIs and frontend domain services already exist. The previous objective (#60 — Prompt Management View) is 100% complete (3/3 sub-issues closed) and ready for transition closeout.

## Transition Closeout (Objective #60)

1. Close objective #60 (all sub-issues complete)
2. Update `_project/phase.md` — mark Objective 4 status as Complete
3. Delete `_project/objective.md`

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| PDF display | New `GET /storage/view/{key...}` endpoint with `Content-Disposition: inline` | Separate route for different representation. No risk to existing download behavior. |
| No-classification state | Empty state with "Back to Documents" link | SSE streaming stays in document-grid only. Review view is for existing classifications. |
| Classification panel data | Module loads its own classification via `ClassificationService.findByDocument()` | Follows module pattern — modules own their async data. |
| Post-action refresh | Panel updates `@state()` from API response, dispatches event for view to re-fetch document | Both validate/update return the updated Classification. View needs fresh document for status change. |
| No re-classification | Not supported from review view | Keeps SSE orchestration in one place (document-grid). |

## Sub-Issues (3 tasks)

### Sub-Issue 1: PDF Viewer Element and Storage Inline Endpoint

**One backend change + one pure element.**

**Backend** (`internal/api/storage.go`):
- Add `view` handler mirroring `download` but with `Content-Disposition: inline`
- Add route: `{Method: "GET", Pattern: "/view/{key...}", Handler: h.view}`
- Update API Cartographer: `_project/api/storage/README.md` + `.http`

**Frontend**:
- `app/client/ui/elements/pdf-viewer.ts` + `pdf-viewer.module.css` — pure element
  - `@property() storageKey: string` — renders `<iframe src="/api/storage/view/${storageKey}">`
  - Full-height flex layout
- Update `app/client/ui/elements/index.ts` barrel

**Labels**: `web`, `pkg/storage`
**Depends on**: —

---

### Sub-Issue 2: Markings List Element and Classification Panel Module

**One pure element + one stateful module. Core review functionality.**

**Element — `hd-markings-list`** (`app/client/ui/elements/markings-list.ts` + CSS module):
- `@property({ type: Array }) markings: string[]` — renders styled badge tags
- Display-only, no events

**Module — `hd-classification-panel`** (`app/client/ui/modules/classification-panel.ts` + CSS module):
- Props: `@property() documentId`, `@property({ type: Object }) document`
- State: `@state() classification`, `@state() loading`, `@state() error`, `@state() mode: 'view' | 'validate' | 'update'`, `@state() submitting`
- Loads classification via `ClassificationService.findByDocument(documentId)`
- Display: classification + confidence badge, markings list, rationale, model/provider, timestamps, validated-by
- Validate action: name input + button → `ClassificationService.validate()`
- Update action: expand form (classification, rationale, name) → `ClassificationService.update()`
- Both dispatch custom events (`validate` / `update`) with updated classification
- Empty state when no classification exists

**Barrel updates**: `ui/elements/index.ts`, `ui/modules/index.ts`

**Labels**: `web`
**Depends on**: —

---

### Sub-Issue 3: Review View Composition

**Wire the skeleton `hd-review-view` into a full two-panel layout.**

**View — `hd-review-view`** (`app/client/ui/views/review-view.ts` + CSS module):
- `@property() documentId` (already exists from router)
- `@state() document` — loaded via `DocumentService.find(documentId)`
- `@state() loading`, `@state() error`
- Header: back button (`navigate('')`), document filename, status badge
- Two-panel flex layout: left (~60%) PDF viewer, right (~40%) classification panel
- Listens for `validate`/`update` events from panel → re-fetches document to update status

**Labels**: `web`
**Depends on**: Sub-Issue 1, Sub-Issue 2

---

## Dependency Graph

```
Sub-Issue 1 (PDF viewer + storage endpoint)
                                              \
                                               → Sub-Issue 3 (Review view composition)
                                              /
Sub-Issue 2 (Markings list + classification panel)
```

Sub-issues 1 and 2 are independent. Sub-issue 3 composes both.

## Verification

After all 3 sub-issues:
1. `go vet ./...` passes
2. `go test ./tests/...` passes
3. `bun run build` succeeds (from `app/`)
4. Navigate to a document with status `review` in the document grid → click "Review"
5. PDF renders inline on the left panel via iframe
6. Classification data displays on the right panel with markings, rationale, metadata
7. Validate action transitions document to `complete`
8. Update action with modified classification/rationale transitions document to `complete`
9. Back navigation returns to document grid
10. Unclassified document shows empty state with back link
