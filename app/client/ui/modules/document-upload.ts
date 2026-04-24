import { LitElement, html, nothing } from "lit";
import { customElement, state } from "lit/decorators.js";

import { formatBytes } from "@core/formatting";
import { DocumentService } from "@domains/documents";
import type { UploadEntry } from "@domains/documents";
import { acceptAttribute, dropZoneText, isSupported } from "@domains/formats";
import { Toast } from "@ui/elements";

import badgeStyles from "@styles/badge.module.css";
import buttonStyles from "@styles/buttons.module.css";
import scrollStyles from "@styles/scroll.module.css";
import styles from "./document-upload.module.css";

/**
 * Stateful component that coordinates multi-file PDF uploads.
 * Provides drag-and-drop and file picker selection, per-file metadata
 * inputs, and concurrent upload via {@link DocumentService.upload}.
 * Dispatches `upload-complete` when at least one file uploads successfully.
 */
@customElement("hd-document-upload")
export class DocumentUpload extends LitElement {
  static styles = [buttonStyles, badgeStyles, scrollStyles, styles];

  @state() private queue: UploadEntry[] = [];
  @state() private uploading = false;
  @state() private dragover = false;

  private get canUpload(): boolean {
    return (
      this.queue.length > 0 &&
      !this.uploading &&
      this.queue.every(
        (e) =>
          e.status === "pending" &&
          e.externalId > 0 &&
          e.platform.trim() !== "",
      )
    );
  }

  updated(changed: Map<string, unknown>) {
    if (changed.has("dragover")) {
      this.toggleAttribute("dragover", this.dragover);
    }
  }

  private addFiles(files: FileList) {
    const all = Array.from(files);
    const accepted = all.filter((f) => isSupported(f.type));
    const rejected = all.length - accepted.length;

    if (rejected > 0) {
      Toast.warning(
        `Skipped ${rejected} unsupported file${rejected === 1 ? "" : "s"}`,
      );
    }

    if (accepted.length < 1) return;

    const entries: UploadEntry[] = accepted.map((file) => ({
      file,
      status: "pending" as const,
      externalId: 0,
      platform: "",
    }));

    this.queue = [...this.queue, ...entries];
  }

  private handleFileInput(e: Event) {
    const input = e.target as HTMLInputElement;
    if (input.files) this.addFiles(input.files);
    input.value = "";
  }

  private handleClick() {
    this.renderRoot.querySelector<HTMLInputElement>("#file-input")?.click();
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

    this.queue = this.queue.map((entry) => ({
      ...entry,
      status: "uploading" as const,
    }));

    const results = await Promise.allSettled(
      this.queue.map(async (entry, index) => {
        const result = await DocumentService.upload(
          entry.file,
          entry.externalId,
          entry.platform,
        );

        this.queue[index].status = result.ok
          ? ("success" as const)
          : ("error" as const);
        this.queue[index].error = result.ok ? undefined : result.error;
        this.requestUpdate();

        if (!result.ok) throw new Error(result.error);

        return result.data;
      }),
    );

    this.uploading = false;

    const succeeded = results.filter((r) => r.status === "fulfilled").length;
    const failed = results.length - succeeded;

    if (succeeded > 0) {
      Toast.success(`Uploaded ${succeeded} file${succeeded === 1 ? "" : "s"}`);
      this.dispatchEvent(
        new CustomEvent("upload-complete", {
          bubbles: true,
          composed: true,
        }),
      );
    }

    if (failed > 0) {
      Toast.error(`Failed to upload ${failed} file${failed === 1 ? "" : "s"}`);
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
          accept=${acceptAttribute()}
          multiple
          hidden
          @change=${this.handleFileInput}
        />
        <span class="drop-icon">📄</span>
        <span class="drop-text">${dropZoneText()}</span>
      </div>
    `;
  }

  private renderQueueEntry(entry: UploadEntry, index: number) {
    const editable = entry.status === "pending";

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
              min="1"
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

          ${editable
            ? html`
                <button
                  class="btn btn-red"
                  @click=${() => this.handleRemove(index)}
                >
                  Remove
                </button>
              `
            : nothing}
        </div>

        ${entry.error
          ? html` <div class="entry-error">${entry.error}</div> `
          : nothing}
      </div>
    `;
  }

  private renderQueueActions() {
    const allSettled = this.queue.every(
      (e) => e.status === "success" || e.status === "error",
    );

    return html`
      <div class="queue-actions">
        <button
          class="btn btn-yellow"
          @click=${this.handleClear}
          ?disabled=${this.uploading}
        >
          ${allSettled ? "Done" : "Clear"}
        </button>
        ${!allSettled
          ? html`
              <button
                class="btn btn-blue"
                @click=${this.handleUpload}
                ?disabled=${!this.canUpload}
              >
                Upload
              </button>
            `
          : nothing}
      </div>
    `;
  }

  private renderQueue() {
    if (this.queue.length < 1) return nothing;

    return html`
      <div class="queue">
        <div class="queue-header">
          <span class="queue-title">File Queue</span>
          <span class="queue-count">
            ${this.queue.length} file${this.queue.length > 1 ? "s" : ""}
          </span>
          ${this.renderQueueActions()}
        </div>
        <div class="queue-list scroll-y">
          ${this.queue.map((entry, i) => this.renderQueueEntry(entry, i))}
        </div>
      </div>
    `;
  }

  render() {
    return html` ${this.renderDropZone()} ${this.renderQueue()} `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-document-upload": DocumentUpload;
  }
}
