# 76 - Document Upload Component

## Summary

Created `hd-document-upload`, a stateful component that coordinates multi-file PDF uploads with per-file metadata inputs and status tracking. This is the third sub-issue of Objective #59 (Document Management View) and establishes the `components/` directory tier.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Component placement | `components/documents/` | Follows three-tier hierarchy — component calls services and manages local state, distinct from views (route-level) and elements (pure) |
| Upload metadata | Per-file inputs | Each file gets its own `external_id` and `external_platform` fields for maximum flexibility |
| State mutations | Direct mutation + `requestUpdate()` | For single-entry field changes, avoids unnecessary array mapping; array replacement used for batch status transitions |
| Domain types | `documents/upload.ts` | `UploadStatus` and `UploadEntry` are domain types, not component-local |

## Files Modified

- `app/client/documents/upload.ts` — new: `UploadStatus` type and `UploadEntry` interface
- `app/client/documents/index.ts` — updated: re-exports upload types
- `app/client/components/documents/document-upload.ts` — new: `hd-document-upload` component
- `app/client/components/documents/document-upload.module.css` — new: component styles
- `app/client/components/documents/index.ts` — new: barrel export
- `app/client/components/index.ts` — new: barrel export
- `app/client/app.ts` — updated: imports `./components`

## Patterns Established

- **`components/` directory**: First stateful component created, establishing the directory structure parallel to `views/` and `elements/`
- **Direct mutation for field updates**: `this.queue[index].field = value; this.requestUpdate()` preferred over mapping entire arrays for single-entry changes
- **Upload queue pattern**: `UploadEntry[]` with per-file status tracking and `Promise.allSettled` coordination

## Validation Results

- `bun run build` passes
- `go vet ./...` passes
- Interactive validation deferred to #77 (view integration)
