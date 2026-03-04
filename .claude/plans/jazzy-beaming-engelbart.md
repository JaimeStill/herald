# Plan: #76 — Document Upload Component

## Context

Third sub-issue of Objective #59 (Document Management View). Creates `hd-document-upload`, a stateful component that coordinates multi-file PDF uploads with per-file progress tracking. Dependencies #74 (service) and #75 (elements) are both closed.

## Component Placement

**Location**: `app/client/components/documents/` (stateful component tier — calls `DocumentService.upload()`, manages local `@state()`)

## Design

### Upload Metadata

The Go endpoint requires `external_id` (int) and `external_platform` (string) per file. Each file in the queue gets its own editable `external_id` and `external_platform` inputs — maximum flexibility for mixed-source batches.

### File Queue & Status

```typescript
type UploadStatus = 'pending' | 'uploading' | 'success' | 'error';

interface UploadEntry {
  file: File;
  status: UploadStatus;
  externalId: number;
  platform: string;
  error?: string;
}
```

Tracked via `@state() private queue: UploadEntry[]`. Default `externalId` auto-increments from 1 for each added file. Default `platform` is empty string (required before upload).

### Upload Flow

1. User selects files via hidden `<input type="file" accept=".pdf" multiple>` triggered by styled button, **and/or** drag-and-drop zone
2. Files appear in a queue list with status badges (all `pending`)
3. User fills in per-file `external_platform` and `external_id` fields (defaults pre-populated), then clicks "Upload"
4. All files upload concurrently via `Promise.allSettled`, each calling `DocumentService.upload(entry.file, entry.externalId, entry.platform)`
5. Per-file status updates reactively as each promise settles
6. When all settled: dispatch `upload-complete` CustomEvent (bubbles, composed)

### Component API

- **Events out**: `upload-complete` — parent view refreshes document list
- **No @consume/@provide** — self-contained; just imports `DocumentService` directly
- **State**: `@state() queue`, `@state() uploading` (boolean to disable form during batch)

## Files

| File | Action |
|------|--------|
| `app/client/components/documents/document-upload.ts` | **New** — stateful upload component |
| `app/client/components/documents/document-upload.module.css` | **New** — component styles |
| `app/client/components/documents/index.ts` | **New** — barrel export |
| `app/client/components/index.ts` | **New** — barrel importing `./documents` |
| `app/client/app.ts` | **Edit** — add `import './components'` side-effect import |

## UI Layout

```
┌─────────────────────────────────────────┐
│  Drop Zone / Click to Select            │
│  ┌───────────────────────────────────┐  │
│  │  📄  Drag PDFs here or click to   │  │
│  │     browse                        │  │
│  └───────────────────────────────────┘  │
│                                         │
│  ┌─ File Queue ───────────────────────────────────────────┐ │
│  │ report.pdf    48.2 KB  ID:[1___] Platform:[____] pending │ │
│  │ analysis.pdf  102 KB   ID:[2___] Platform:[____] success │ │
│  │ summary.pdf   23.1 KB  ID:[3___] Platform:[____] error   │ │
│  └────────────────────────────────────────────────────────┘ │
│                                         │
│  [Clear]                     [Upload]   │
└─────────────────────────────────────────┘
```

## Styling Approach

- Design tokens from `tokens.css` (spacing, typography, colors, radii)
- Status badge pattern from `document-card.module.css` (`.badge.pending`, `.badge.review`, etc.)
- Drag-over visual feedback via host attribute reflection (`dragover` attr toggled in JS, CSS drives style)
- Dashed border drop zone, subtle background shift on hover/dragover
- Consistent `.btn` styling matching document-card buttons

## Validation

- `go vet ./...` passes (no Go changes, but verify)
- `bun run build` succeeds
- Component renders in browser at `/app/`
- File selection populates queue
- Drag-and-drop populates queue
- Upload dispatches to `/api/documents` per file
- Per-file status updates correctly
- `upload-complete` event fires when all settle
