import { LitElement, html, nothing } from "lit";
import { customElement, state } from "lit/decorators.js";

import type { PageResult } from "@core";
import { navigate, queryParams, updateQuery } from "@core/router";
import { ClassificationService } from "@domains/classifications";
import type { WorkflowStage } from "@domains/classifications";
import { DocumentService } from "@domains/documents";
import type { Document, SearchRequest } from "@domains/documents";
import { Toast } from "@ui/elements";

import buttonStyles from "@styles/buttons.module.css";
import inputStyles from "@styles/inputs.module.css";
import scrollStyles from "@styles/scroll.module.css";
import styles from "./document-grid.module.css";

const DEFAULTS = {
  page: 1,
  pageSize: 12,
  search: "",
  status: "",
  sort: "-UploadedAt",
} as const;

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
  static styles = [buttonStyles, inputStyles, scrollStyles, styles];

  @state() private documents: PageResult<Document> | null = null;
  @state() private page: number = DEFAULTS.page;
  @state() private pageSize: number = DEFAULTS.pageSize;
  @state() private search: string = DEFAULTS.search;
  @state() private status: string = DEFAULTS.status;
  @state() private sort: string = DEFAULTS.sort;
  @state() private classifying = new Map<string, ClassifyProgress>();
  @state() private selectedIds = new Set<string>();
  @state() private deleteDocument: Document | null = null;
  @state() private deleteDocuments: Document[] | null = null;

  private searchTimer = 0;
  private abortControllers = new Map<string, AbortController>();

  connectedCallback() {
    super.connectedCallback();
    this.hydrateFromQuery();
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
    this.page = DEFAULTS.page;
    this.syncQuery();
    await this.fetchDocuments();
  }

  private async fetchDocuments() {
    const req: SearchRequest = {
      page: this.page,
      page_size: this.pageSize,
      sort: this.sort,
    };

    if (this.search) req.search = this.search;
    if (this.status) req.status = this.status;

    const result = await DocumentService.search(req);

    if (result.ok) this.documents = result.data;
  }

  private hydrateFromQuery() {
    const q = queryParams();
    if (q.page) this.page = Number(q.page) || DEFAULTS.page;
    if (q.page_size) this.pageSize = Number(q.page_size) || DEFAULTS.pageSize;
    if (q.search) this.search = q.search;
    if (q.status) this.status = q.status;
    if (q.sort) this.sort = q.sort;
  }

  private syncQuery() {
    updateQuery({
      page: this.page === DEFAULTS.page ? undefined : this.page,
      page_size:
        this.pageSize === DEFAULTS.pageSize ? undefined : this.pageSize,
      search: this.search || undefined,
      status: this.status || undefined,
      sort: this.sort === DEFAULTS.sort ? undefined : this.sort,
    });
  }

  private handleSearchInput(e: Event) {
    const input = e.target as HTMLInputElement;
    this.search = input.value;

    clearTimeout(this.searchTimer);
    this.searchTimer = window.setTimeout(() => this.refresh(), 300);
  }

  private handleStatusFilter(e: Event) {
    const select = e.target as HTMLSelectElement;
    this.status = select.value;
    this.refresh();
  }

  private handleSort(e: Event) {
    const select = e.target as HTMLSelectElement;
    this.sort = select.value;
    this.refresh();
  }

  private handlePageChange(e: CustomEvent<{ page: number }>) {
    this.page = e.detail.page;
    this.syncQuery();
    this.fetchDocuments();
  }

  private handlePageSizeChange(e: CustomEvent<{ size: number }>) {
    this.pageSize = e.detail.size;
    this.page = DEFAULTS.page;
    this.syncQuery();
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
        const filename =
          this.documents?.data.find((d) => d.id === docId)?.filename ??
          "document";
        Toast.success(`Classified ${filename}`);
        this.fetchDocuments();
      },
      onError: () => {
        this.abortControllers.delete(docId);
        const updated = new Map(this.classifying);
        updated.delete(docId);
        this.classifying = updated;
        const filename =
          this.documents?.data.find((d) => d.id === docId)?.filename ??
          "document";
        Toast.error(`Classification failed for ${filename}`);
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
    const filename = this.deleteDocument.filename;
    this.deleteDocument = null;

    const result = await DocumentService.delete(id);

    if (result.ok) {
      this.selectedIds.delete(id);
      this.fetchDocuments();
      Toast.success(`Deleted ${filename}`);
    } else {
      Toast.error(`Failed to delete ${filename}: ${result.error}`);
    }
  }

  private cancelDelete() {
    this.deleteDocument = null;
  }

  private handleBulkDelete() {
    if (!this.documents) return;

    const selected = this.documents.data.filter((d) =>
      this.selectedIds.has(d.id),
    );
    if (selected.length === 0) return;

    this.deleteDocuments = selected;
  }

  private async confirmBulkDelete() {
    const batch = this.deleteDocuments;
    this.deleteDocuments = null;
    if (!batch) return;

    const outcomes = await Promise.all(
      batch.map(async (doc) => {
        try {
          const result = await DocumentService.delete(doc.id);
          return result.ok
            ? { doc, ok: true as const }
            : { doc, ok: false as const, error: result.error };
        } catch (err) {
          return { doc, ok: false as const, error: String(err) };
        }
      }),
    );

    const failed = new Set<string>();
    let succeeded = 0;

    for (const outcome of outcomes) {
      if (outcome.ok) {
        succeeded++;
        continue;
      }
      failed.add(outcome.doc.id);
      console.error(
        `Failed to delete document ${outcome.doc.id}:`,
        outcome.error,
      );
      Toast.error(`Failed to delete ${outcome.doc.filename}: ${outcome.error}`);
    }

    if (succeeded > 0) {
      Toast.success(
        `Deleted ${succeeded} document${succeeded === 1 ? "" : "s"}`,
      );
    }

    this.selectedIds = failed;
    this.fetchDocuments();
  }

  private cancelBulkDelete() {
    this.deleteDocuments = null;
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
          class="input search-input"
          placeholder="Search documents..."
          .value=${this.search}
          @input=${this.handleSearchInput}
        />
        <select class="input filter-select" @change=${this.handleStatusFilter}>
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
        <select class="input sort-select" @change=${this.handleSort}>
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
              <button class="btn btn-blue" @click=${this.handleBulkClassify}>
                Classify ${this.selectedIds.size} Documents
              </button>
              <button class="btn btn-red" @click=${this.handleBulkDelete}>
                Delete ${this.selectedIds.size} Documents
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
      <div class="grid scroll-y">
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
        .size=${this.pageSize}
        @page-change=${this.handlePageChange}
        @page-size-change=${this.handlePageSizeChange}
      ></hd-pagination>
      ${this.deleteDocument
        ? html`
            <hd-confirm-dialog
              message="Are you sure you want to delete ${this.deleteDocument
                .filename}?"
              @confirm=${this.confirmDelete}
              @cancel=${this.cancelDelete}
            ></hd-confirm-dialog>
          `
        : nothing}
      ${this.deleteDocuments
        ? html`
            <hd-confirm-dialog
              message="Are you sure you want to delete ${this.deleteDocuments
                .length} documents?"
              @confirm=${this.confirmBulkDelete}
              @cancel=${this.cancelBulkDelete}
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
