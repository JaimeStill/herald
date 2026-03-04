# 76 - Document Upload Component

## Problem Context

The document management view (#59) needs an upload component that coordinates multi-file PDF uploads. Users must be able to select files (click or drag-and-drop), assign per-file metadata (`external_id` and `external_platform`), and upload all files concurrently with per-file status feedback.

## Architecture Approach

`hd-document-upload` is a **stateful component** in `components/documents/` — it calls `DocumentService.upload()` directly and manages local `@state()` for the file queue and upload lifecycle. It dispatches an `upload-complete` event when all uploads settle so the parent view can refresh its document list.

No `@consume`/`@provide` — this component is self-contained. The three-tier hierarchy is: view (route-level, #77) → component (this, service-calling) → elements (pure, already done in #75).

## Implementation

### Step 1: Create `app/client/documents/upload.ts`

New file — domain types for the upload lifecycle:

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

Then update `app/client/documents/index.ts` to re-export them. Add after the existing exports:

```typescript
export type { UploadStatus, UploadEntry } from './upload';
```

### Step 2: Create `app/client/components/documents/document-upload.ts`

New file — the complete component:

```typescript
import { LitElement, html, nothing } from 'lit';
import { customElement, state } from 'lit/decorators.js';
import { DocumentService } from '@app/documents';
import type { UploadEntry } from '@app/documents';
import { formatBytes } from '@app/formatting';
import styles from './document-upload.module.css';

@customElement('hd-document-upload')
export class DocumentUpload extends LitElement {
  static styles = styles;

  @state() private queue: UploadEntry[] = [];
  @state() private uploading = false;
  @state() private dragover = false;

  private get canUpload(): boolean {
    return this.queue.length > 0
      && !this.uploading
      && this.queue.every(e =>
        e.status === 'pending'
        && e.externalId > 0
        && e.platform.trim() !== ''
      );
  }

  updated(changed: Map<string, unknown>) {
    if (changed.has('dragover')) {
      this.toggleAttribute('dragover', this.dragover);
    }
  }

  private addFiles(files: FileList) {
    const pdfs = Array.from(files).filter(f => f.type === 'application/pdf');
    if (pdfs.length === 0) return;

    const entries: UploadEntry[] = pdfs.map(file => ({
      file,
      status: 'pending' as const,
      externalId: 0,
      platform: '',
    }));

    this.queue = [...this.queue, ...entries];
  }

  private handleFileInput(e: Event) {
    const input = e.target as HTMLInputElement;
    if (input.files) this.addFiles(input.files);
    input.value = '';
  }

  private handleClick() {
    this.renderRoot.querySelector<HTMLInputElement>('#file-input')?.click();
  }

  private handleDragOver(e: DragEvent) {
    e.preventDefault();
    this.dragover = true;
  }

  private handleDragLeave() {
    this.dragover = false;
  }

  private handleDrop(e: DragEvent) {
    e.preventDefault();
    this.dragover = false;
    if (e.dataTransfer?.files) this.addFiles(e.dataTransfer.files);
  }

  private handleIdChange(index: number, e: Event) {
    const input = e.target as HTMLInputElement;
    const value = parseInt(input.value, 10);
    if (Number.isNaN(value)) return;

    this.queue[index].externalId = value;
    this.requestUpdate();
  }

  private handlePlatformChange(index: number, e: Event) {
    const input = e.target as HTMLInputElement;

    this.queue[index].platform = input.value;
    this.requestUpdate();
  }

  private handleRemove(index: number) {
    this.queue = this.queue.filter((_, i) => i !== index);
  }

  private handleClear() {
    this.queue = [];
  }

  private async handleUpload() {
    this.uploading = true;

    this.queue = this.queue.map(entry => ({
      ...entry,
      status: 'uploading' as const,
    }));

    const results = await Promise.allSettled(
      this.queue.map(async (entry, index) => {
        const result = await DocumentService.upload(
          entry.file,
          entry.externalId,
          entry.platform,
        );

        this.queue[index].status = result.ok ? 'success' as const : 'error' as const;
        this.queue[index].error = result.ok ? undefined : result.error;
        this.requestUpdate();

        if (!result.ok)
          throw new Error(result.error);

        return result.data;
      }),
    );

    this.uploading = false;

    const hasSuccess = results.some(r => r.status === 'fulfilled');
    if (hasSuccess) {
      this.dispatchEvent(new CustomEvent('upload-complete', {
        bubbles: true,
        composed: true,
      }));
    }
  }

  private renderDropZone() {
    return html`
      <div
        class="drop-zone"
        @click=${this.handleClick}
        @dragover=${this.handleDragOver}
        @dragleave=${this.handleDragLeave}
        @drop=${this.handleDrop}
      >
        <input
          id="file-input"
          type="file"
          accept=".pdf"
          multiple
          hidden
          @change=${this.handleFileInput}
        />
        <span class="drop-icon">📄</span>
        <span class="drop-text">Drag PDFs here or click to browse</span>
      </div>
    `;
  }

  private renderQueueEntry(entry: UploadEntry, index: number) {
    const settled = entry.status === 'success' || entry.status === 'error';
    const editable = entry.status === 'pending';

    return html`
      <div class="queue-entry">
        <div class="entry-info">
          <span class="entry-filename">${entry.file.name}</span>
          <span class="entry-size">${formatBytes(entry.file.size)}</span>
          <span class="badge ${entry.status}">${entry.status}</span>
        </div>

        <div class="entry-fields">
          <label class="field">
            <span class="field-label">ID</span>
            <input
              type="number"
              class="field-input id-input"
              .value=${String(entry.externalId)}
              ?disabled=${!editable}
              @change=${(e: Event) => this.handleIdChange(index, e)}
            />
          </label>

          <label class="field">
            <span class="field-label">Platform</span>
            <input
              type="text"
              class="field-input"
              .value=${entry.platform}
              placeholder="e.g. HQ"
              ?disabled=${!editable}
              @input=${(e: Event) => this.handlePlatformChange(index, e)}
            />
          </label>

          ${editable ? html`
            <button
              class="btn remove-btn"
              @click=${() => this.handleRemove(index)}
            >✕</button>
          ` : nothing}
        </div>

        ${entry.error ? html`
          <div class="entry-error">${entry.error}</div>
        ` : nothing}
      </div>
    `;
  }

  private renderQueue() {
    if (this.queue.length === 0) return nothing;

    return html`
      <div class="queue">
        <div class="queue-header">
          <span class="queue-title">File Queue</span>
          <span class="queue-count">${this.queue.length} file${this.queue.length !== 1 ? 's' : ''}</span>
        </div>
        ${this.queue.map((entry, i) => this.renderQueueEntry(entry, i))}
      </div>
    `;
  }

  private renderActions() {
    if (this.queue.length === 0) return nothing;

    const allSettled = this.queue.every(
      e => e.status === 'success' || e.status === 'error'
    );

    return html`
      <div class="actions">
        <button
          class="btn clear-btn"
          @click=${this.handleClear}
          ?disabled=${this.uploading}
        >${allSettled ? 'Done' : 'Clear'}</button>
        ${!allSettled ? html`
          <button
            class="btn upload-btn"
            @click=${this.handleUpload}
            ?disabled=${!this.canUpload}
          >Upload</button>
        ` : nothing}
      </div>
    `;
  }

  render() {
    return html`
      ${this.renderDropZone()}
      ${this.renderQueue()}
      ${this.renderActions()}
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'hd-document-upload': DocumentUpload;
  }
}
```

### Step 3: Create `app/client/components/documents/document-upload.module.css`

New file — component styles using the established design token system:

```css
:host {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.drop-zone {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  padding: var(--space-8);
  border: 2px dashed var(--divider);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}

.drop-zone:hover {
  border-color: var(--color-2);
  background: var(--bg-1);
}

:host([dragover]) .drop-zone {
  border-color: var(--blue);
  background: var(--blue-bg);
}

.drop-icon {
  font-size: var(--text-2xl);
}

.drop-text {
  font-size: var(--text-sm);
  color: var(--color-2);
}

.queue {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  border: 1px solid var(--divider);
  border-radius: var(--radius-md);
  padding: var(--space-3);
}

.queue-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: var(--space-2);
  border-bottom: 1px solid var(--divider);
}

.queue-title {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color);
}

.queue-count {
  font-size: var(--text-xs);
  color: var(--color-2);
}

.queue-entry {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-2) 0;
}

.queue-entry:not(:last-child) {
  border-bottom: 1px solid var(--divider);
}

.entry-info {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.entry-filename {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
  flex: 1;
}

.entry-size {
  flex-shrink: 0;
  font-size: var(--text-xs);
  color: var(--color-2);
}

.badge {
  flex-shrink: 0;
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.badge.pending {
  color: var(--yellow);
  background: var(--yellow-bg);
}

.badge.uploading {
  color: var(--blue);
  background: var(--blue-bg);
}

.badge.success {
  color: var(--green);
  background: var(--green-bg);
}

.badge.error {
  color: var(--red);
  background: var(--red-bg);
}

.entry-fields {
  display: flex;
  align-items: flex-end;
  gap: var(--space-2);
}

.field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.field-label {
  font-size: var(--text-xs);
  color: var(--color-2);
}

.field-input {
  padding: var(--space-1) var(--space-2);
  border: 1px solid var(--divider);
  border-radius: var(--radius-sm);
  background: var(--bg);
  color: var(--color);
  font-size: var(--text-xs);
  font-family: var(--font-sans);
}

.field-input:focus {
  outline: none;
  border-color: var(--blue);
}

.field-input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.id-input {
  width: 4rem;
}

.entry-error {
  font-size: var(--text-xs);
  color: var(--red);
}

.actions {
  display: flex;
  justify-content: space-between;
}

.btn {
  padding: var(--space-1) var(--space-3);
  border: 1px solid var(--divider);
  border-radius: var(--radius-sm);
  background: var(--bg-2);
  color: var(--color);
  font-size: var(--text-xs);
  font-family: var(--font-sans);
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
}

.btn:hover:not(:disabled) {
  border-color: var(--color-2);
}

.btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.upload-btn:not(:disabled) {
  border-color: var(--blue);
  color: var(--blue);
}

.upload-btn:hover:not(:disabled) {
  background: var(--blue-bg);
}

.remove-btn {
  padding: var(--space-1) var(--space-2);
  border-color: var(--red);
  color: var(--red);
}

.remove-btn:hover:not(:disabled) {
  background: var(--red-bg);
}

.clear-btn:not(:disabled) {
  border-color: var(--yellow);
  color: var(--yellow);
}

.clear-btn:hover:not(:disabled) {
  background: var(--yellow-bg);
}
```

### Step 4: Create barrel exports

**`app/client/components/documents/index.ts`** — new file:

```typescript
export { DocumentUpload } from './document-upload';
```

**`app/client/components/index.ts`** — new file:

```typescript
export * from './documents';
```

### Step 5: Register components in `app.ts`

Add the components import to `app/client/app.ts`. After the existing `import './elements';` line, add:

```typescript
import './components';
```

The final `app.ts` will be:

```typescript
import './design/index.css';
import './elements';
import './components';
import './views';

import { Router } from '@app/router';

const router = new Router('app-content');
router.start();
```

## Validation Criteria

- [ ] `bun run build` succeeds with no errors
- [ ] `go vet ./...` passes (no Go changes but verify nothing regressed)
- [ ] Drop zone renders and responds to click (opens file picker)
- [ ] Drag-and-drop onto zone adds PDF files to queue
- [ ] Non-PDF files are filtered out
- [ ] Per-file `external_id` and `external_platform` inputs are editable
- [ ] Upload button disabled when `external_id` is 0 or `external_platform` is empty
- [ ] Remove button (✕) removes individual pending entries
- [ ] Clear button resets the entire queue
- [ ] Upload button disabled when platform fields are empty
- [ ] Upload button disabled during active upload
- [ ] Each file uploads independently via `DocumentService.upload()`
- [ ] Per-file status updates from pending → uploading → success/error
- [ ] Error messages display for failed uploads
- [ ] `upload-complete` event dispatches when at least one file succeeds
- [ ] After all uploads settle, Upload button disappears and Clear becomes "Done"
