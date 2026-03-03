# Objective: Document Management View

**Issue**: #59
**Phase**: Phase 3 — Web Client (v0.3.0)

## Scope

Build the primary web client view — the document management interface at `/app/`. Provides document upload (single + batch), browse with search/filter/pagination, classification triggers (single + bulk) with SSE-powered progress indicators, and navigation to the review view.

Establishes the service layer pattern (Signal.State + @lit/context) that subsequent views (Prompts, Review) will follow. Also enhances the core `stream()` API utility to support POST method and event type parsing required by the SSE classification endpoint.

## Sub-Issues

| # | Sub-Issue | Issue | Status | Depends On |
|---|-----------|-------|--------|------------|
| 1 | Document types, service, and SSE stream enhancement | #74 | Open | — |
| 2 | Document card and classify progress pure elements | #75 | Open | #74 |
| 3 | Document upload component with multi-file support | #76 | Open | #74 |
| 4 | Document grid, view integration, and bulk classify | #77 | Open | #74, #75, #76 |

## Dependency Graph

```
#74 (Types + Service + Stream)
      |           |
      v           v
#75 (Card +    #76 (Upload)
 Progress)        |
      |           |
      v           v
#77 (Grid + View Assembly + Bulk)
```

Sub-issues #75 and #76 can proceed in parallel after #74. Sub-issue #77 depends on all three.

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| stream() enhancement | Add `init?: RequestInit` param + event type parsing | SSE classify endpoint is POST (not GET) and sends `event: <type>\ndata: <json>\n\n` format. Current stream() only handles data lines with GET. |
| Service state management | Signal.State signals + @lit/context | Established pattern from web-development skill. SignalWatcher mixin drives reactive re-renders. |
| classifyingIds tracking | `Signal.State<Set<string>>` in DocumentService | Tracks which documents have active SSE classifications. Cards check membership to show/hide progress. |
| Component co-location | All components in `views/documents/` directory | Single domain, tightly coupled. Card, progress, upload, grid all serve the documents view. |
| Bulk classify | Client-orchestrated parallel SSE via Promise.allSettled | Same pattern as bulk upload — deterministic per-document behavior, per-document progress. |
| Upload metadata | Form inputs for external_id and external_platform | Required by POST /api/documents endpoint. Captured in the upload component. |
