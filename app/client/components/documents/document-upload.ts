import { LitElement, html, nothing } from 'lit';
import { customElement, state } from 'lit/decorators.js';
import { DocumentService } from '@app/documents';
import type { UploadEntry } from '@app/documents';
import { formatBytes } from '@app/formatting';
import styles from './document-upload.module.css';

/**
 * Stateful component that coordinates multi-file PDF uploads.
 * Provides drag-and-drop and file picker selection, per-file metadata
 * inputs, and concurrent upload via {@link DocumentService.upload}.
 * Dispatches `upload-complete` when at least one file uploads successfully.
 */
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
    const pdfs = Array
      .from(files)
      .filter(f => f.type === 'application/pdf');

    if (pdfs.length < 1)
      return;

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
    this.renderRoot
      .querySelector<HTMLInputElement>('#file-input')
      ?.click();
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
    if (e.dataTransfer?.files)
      this.addFiles(e.dataTransfer.files);
  }

  private handleIdChange(index: number, e: Event) {
    const input = e.target as HTMLInputElement;
    const value = parseInt(input.value, 10);
    if (Number.isNaN(value))
      return;

    this.queue[index].externalId = value;
    this.requestUpdate();
  }

  private handlePlatformChange(index: number, e: Event) {
    const input = e.target as HTMLInputElement;

    this.queue[index].platform = input.value;
    this.requestUpdate();
  }

  private handleRemove(index: number) {
    this.queue = this.queue
      .filter((_, i) => i !== index);
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
    if (this.queue.length < 1)
      return nothing;

    return html`
      <div class="queue">
        <div class="queue-header">
          <span class="queue-title">File Queue</span>
          <span class="queue-count">
            ${this.queue.length} file${this.queue.length > 1 ? 's' : ''}
          </span>
        </div>
        ${this.queue.map((entry, i) => this.renderQueueEntry(entry, i))}
      </div>
    `;
  }

  private renderActions() {
    if (this.queue.length < 1)
      return nothing;

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
