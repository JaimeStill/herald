# Objective: Document Review View

**Issue**: #61
**Phase**: Phase 3 — Web Client (v0.3.0)

## Scope

Build the document review view — a side-by-side interface showing a PDF viewer alongside the classification record, with actions to validate (agree with AI) or manually update the classification. Accessible at `/app/review/:documentId`. All backend APIs and frontend domain services already exist. This objective is purely UI work plus one small storage endpoint addition.

## Sub-Issues

| # | Sub-Issue | Issue | Status | Depends On |
|---|-----------|-------|--------|------------|
| 1 | PDF viewer element and storage inline endpoint | #88 | Open | — |
| 2 | Markings list element and classification panel module | #89 | Open | — |
| 3 | Review view composition | #90 | Open | #88, #89 |

## Dependency Graph

```
#88 (PDF Viewer + Storage Endpoint)
                                     \
                                      → #90 (Review View Composition)
                                     /
#89 (Markings List + Classification Panel)
```

Sub-issues 1 and 2 are independent. Sub-issue 3 composes both.

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| PDF display | New `GET /storage/view/{key...}` with `Content-Disposition: inline` | Separate route for different representation. No risk to existing download behavior. |
| No-classification state | Empty state with "Back to Documents" link | SSE streaming stays in document-grid only. Review view is for existing classifications. |
| Classification panel data | Module loads own classification via `ClassificationService.findByDocument()` | Follows module pattern — modules own their async data. |
| Post-action refresh | Panel updates `@state()` from API response, dispatches event for view to re-fetch document | Both validate/update return updated Classification. View needs fresh document for status change. |
| No re-classification from review | Not supported | Keeps SSE orchestration in one place (document-grid module). |
| Layout | Two-panel flex: left ~60% (PDF), right ~40% (classification) | Side-by-side comparison is the primary use case for review. |
