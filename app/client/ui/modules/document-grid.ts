import { LitElement, html, nothing } from "lit";
import { customElement, state } from "lit/decorators.js";

import type { PageResult } from "@core";
import { navigate } from "@core/router";
import { ClassificationService } from "@domains/classifications";
import type { WorkflowStage } from "@domains/classifications";
import { DocumentService } from "@domains/documents";
import type { Document, SearchRequest } from "@domains/documents";

import buttonStyles from "@styles/buttons.module.css";
import styles from "./document-grid.module.css";

interface ClassifyProgress {
  currentNode: WorkflowStage | null;
  completedNodes: WorkflowStage[];
}

/**
 * Stateful module that manages the document browsing experience.
 * Owns search, filtering, sorting, pagination, SSE classify orchestration,
 * bulk selection, and delete confirmation.
 */
@customElement("hd-document-grid")
export class DocumentGrid extends LitElement {
  static styles = [buttonStyles, styles];

  @state() private documents: PageResult<Document> | null = null;
  @state() private page = 1;
  @state() private search = "";
  @state() private status = "";
  @state() private sort = "-UploadedAt";
  @state() private classifying = new Map<string, ClassifyProgress>();
  @state() private selectedIds = new Set<string>();
  @state() private deleteDocument: Document | null = null;

  private searchTimer = 0;
  private abortControllers = new Map<string, AbortController>();

  connectedCallback() {
    super.connectedCallback();
    this.fetchDocuments();
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    clearTimeout(this.searchTimer);
    for (const controller of this.abortControllers.values()) {
      controller.abort();
    }
  }

  async refresh() {
    this.page = 1;
    await this.fetchDocuments();
  }

  private async fetchDocuments() {
    const req: SearchRequest = {
      page: this.page,
      page_size: 12,
      sort: this.sort,
    };

    if (this.search) req.search = this.search;

    if (this.status) req.status = this.status;

    const result = await DocumentService.search(req);

    if (result.ok) this.documents = result.data;
  }

  private handleSearchInput(e: Event) {
    const input = e.target as HTMLInputElement;
    this.search = input.value;

    clearTimeout(this.searchTimer);
    this.searchTimer = window.setTimeout(() => {
      this.page = 1;
      this.fetchDocuments();
    }, 300);
  }

  private handleStatusFilter(e: Event) {
    const select = e.target as HTMLSelectElement;
    this.status = select.value;
    this.page = 1;
    this.fetchDocuments();
  }

  private handleSort(e: Event) {
    const select = e.target as HTMLSelectElement;
    this.sort = select.value;
    this.page = 1;
    this.fetchDocuments();
  }

  private handlePageChange(e: CustomEvent<{ page: number }>) {
    this.page = e.detail.page;
    this.fetchDocuments();
  }

  private handleSelect(e: CustomEvent<{ id: string }>) {
    const id = e.detail.id;
    const next = new Set(this.selectedIds);

    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }

    this.selectedIds = next;
  }

  private handleClassify(e: CustomEvent<{ id: string }>) {
    const docId = e.detail.id;
    if (this.classifying.has(docId)) return;

    const progress: ClassifyProgress = {
      currentNode: null,
      completedNodes: [],
    };

    this.classifying = new Map(this.classifying).set(docId, progress);

    const controller = ClassificationService.classify(docId, {
      onEvent: (type, data) => {
        try {
          const event = JSON.parse(data);
          const updated = new Map(this.classifying);
          const current = updated.get(docId);
          if (!current) return;

          if (type === "node.start") {
            updated.set(docId, {
              ...current,
              currentNode: event.data?.node ?? null,
            });
          } else if (type === "node.complete") {
            const node = event.data?.node as WorkflowStage;
            if (node) {
              updated.set(docId, {
                ...current,
                currentNode: null,
                completedNodes: [...current.completedNodes, node],
              });
            }
          }

          this.classifying = updated;
        } catch (err) {
          console.warn("Failed to parse SSE event:", data, err);
        }
      },
      onComplete: () => {
        this.abortControllers.delete(docId);
        const updated = new Map(this.classifying);
        updated.delete(docId);
        this.classifying = updated;
        this.fetchDocuments();
      },
      onError: () => {
        this.abortControllers.delete(docId);
        const updated = new Map(this.classifying);
        updated.delete(docId);
        this.classifying = updated;
        this.fetchDocuments();
      },
    });

    this.abortControllers.set(docId, controller);
  }

  private handleReview(e: CustomEvent<{ id: string }>) {
    navigate(`review/${e.detail.id}`);
  }

  private handleDelete(e: CustomEvent<{ document: Document }>) {
    this.deleteDocument = e.detail.document;
  }

  private async confirmDelete() {
    if (!this.deleteDocument) return;

    const id = this.deleteDocument.id;
    this.deleteDocument = null;

    const result = await DocumentService.delete(id);

    if (result.ok) {
      this.selectedIds.delete(id);
      this.fetchDocuments();
    }
  }

  private cancelDelete() {
    this.deleteDocument = null;
  }

  private handleBulkClassify() {
    const ids = [...this.selectedIds];
    this.selectedIds = new Set();

    for (const id of ids) {
      this.handleClassify(new CustomEvent("classify", { detail: { id } }));
    }
  }

  private renderToolbar() {
    return html`
      <div class="toolbar">
        <input
          type="search"
          class="search-input"
          placeholder="Search documents..."
          .value=${this.search}
          @input=${this.handleSearchInput}
        />
        <select class="filter-select" @change=${this.handleStatusFilter}>
          <option value="">---</option>
          <option value="pending" ?selected=${this.status === "pending"}>
            Pending
          </option>
          <option value="review" ?selected=${this.status === "review"}>
            Review
          </option>
          <option value="complete" ?selected=${this.status === "complete"}>
            Complete
          </option>
        </select>
        <select class="sort-select" @change=${this.handleSort}>
          <option value="-UploadedAt" ?selected=${this.sort === "-UploadedAt"}>
            Newest
          </option>
          <option value="UploadedAt" ?selected=${this.sort === "UploadedAt"}>
            Oldest
          </option>
          <option value="Filename" ?selected=${this.sort === "Filename"}>
            Name (A-Z)
          </option>
          <option value="-Filename" ?selected=${this.sort === "-Filename"}>
            Name (Z-A)
          </option>
        </select>
        ${this.selectedIds.size > 0
          ? html`
              <button class="btn bulk-btn" @click=${this.handleBulkClassify}>
                Classify ${this.selectedIds.size} Documents
              </button>
            `
          : nothing}
      </div>
    `;
  }

  private renderGrid() {
    if (!this.documents) {
      return html`<div class="empty-state">Loading...</div>`;
    }

    if (this.documents.data.length < 1) {
      return html`<div class="empty-state">No documents found.</div>`;
    }

    return html`
      <div class="grid">
        ${this.documents.data.map((doc) => {
          const progress = this.classifying.get(doc.id);
          return html`
            <hd-document-card
              .currentNode=${progress?.currentNode ?? null}
              .completedNodes=${progress?.completedNodes ?? []}
              .document=${doc}
              ?classifying=${this.classifying.has(doc.id)}
              ?selected=${this.selectedIds.has(doc.id)}
              @classify=${this.handleClassify}
              @review=${this.handleReview}
              @delete=${this.handleDelete}
              @select=${this.handleSelect}
            ></hd-document-card>
          `;
        })}
      </div>
    `;
  }

  render() {
    return html`
      ${this.renderToolbar()} ${this.renderGrid()}
      <hd-pagination
        .page=${this.documents?.page ?? 1}
        .totalPages=${this.documents?.total_pages ?? 1}
        @page-change=${this.handlePageChange}
      ></hd-pagination>
      ${this.deleteDocument
        ? html`
            <hd-confirm-dialog
              message="Are you sure you want to delete ${this.deleteDocument.filename}?"
              @confirm=${this.confirmDelete}
              @cancel=${this.cancelDelete}
            ></hd-confirm-dialog>
          `
        : nothing}
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-document-grid": DocumentGrid;
  }
}
