import { LitElement, html, nothing } from "lit";
import { customElement, property, state } from "lit/decorators.js";

import { formatDate } from "@core/formatting";
import { navigate } from "@core/router";
import { ClassificationService } from "@domains/classifications";
import type { Classification } from "@domains/classifications";

import badgeStyles from "@styles/badge.module.css";
import buttonStyles from "@styles/buttons.module.css";
import styles from "./classification-panel.module.css";

type PanelMode = "view" | "validate" | "update";

/**
 * Stateful module that loads and displays a document's classification result.
 * Supports three modes: view (read-only display), validate (confirm AI result),
 * and update (manually revise classification). Dispatches `validate` and `update`
 * custom events with the updated classification on successful submission.
 */
@customElement("hd-classification-panel")
export class ClassificationPanel extends LitElement {
  static styles = [badgeStyles, buttonStyles, styles];

  @property() documentId = "";

  @state() private classification: Classification | null = null;
  @state() private loading = true;
  @state() private error = "";
  @state() private mode: PanelMode = "view";
  @state() private submitting = false;

  updated(changed: Map<string, unknown>) {
    if (changed.has("documentId") && this.documentId) {
      this.loadClassification();
    }
  }

  private async loadClassification() {
    this.loading = true;
    this.error = "";
    this.classification = null;

    const result = await ClassificationService.findByDocument(this.documentId);

    this.loading = false;

    if (!result.ok) {
      this.error = result.error;
      return;
    }

    this.classification = result.data;
  }

  private async handleValidate(e: SubmitEvent) {
    e.preventDefault();
    this.submitting = true;

    const form = e.target as HTMLFormElement;
    const data = new FormData(form);
    const validated_by = (data.get("validated_by") as string).trim();

    const result = await ClassificationService.validate(
      this.classification!.id,
      { validated_by },
    );

    this.submitting = false;

    if (!result.ok) {
      this.error = result.error;
      return;
    }

    this.classification = result.data;
    this.mode = "view";
    this.error = "";

    this.dispatchEvent(
      new CustomEvent("validate", {
        detail: { classification: result.data },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private async handleUpdate(e: SubmitEvent) {
    e.preventDefault();
    this.submitting = true;

    const form = e.target as HTMLFormElement;
    const data = new FormData(form);

    const command = {
      classification: (data.get("classification") as string).trim(),
      rationale: (data.get("rationale") as string).trim(),
      updated_by: (data.get("updated_by") as string).trim(),
    };

    const result = await ClassificationService.update(
      this.classification!.id,
      command,
    );

    this.submitting = false;

    if (!result.ok) {
      this.error = result.error;
      return;
    }

    this.classification = result.data;
    this.mode = "view";
    this.error = "";

    this.dispatchEvent(
      new CustomEvent("update", {
        detail: { classification: result.data },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private handleBack() {
    navigate("");
  }

  private handleCancel() {
    this.mode = "view";
    this.error = "";
  }

  private renderError() {
    if (!this.error) return nothing;
    return html`<div class="error">${this.error}</div>`;
  }

  private renderValidated() {
    const c = this.classification;
    if (!c?.validated_by) return nothing;

    return html`
      <div class="validated">
        <span>Validated by ${c.validated_by}</span>
        ${c.validated_at
          ? html`<span>on ${formatDate(c.validated_at)}</span>`
          : nothing}
      </div>
    `;
  }

  private renderViewMode() {
    const c = this.classification!;

    return html`
      <div class="panel-body">
        <div class="section">
          <span class="section-label">Classification</span>
          <div class="classification-value">
            <span class="classification-name">${c.classification}</span>
            <span class="badge confidence ${c.confidence.toLowerCase()}">
              ${c.confidence.toLowerCase()}
            </span>
          </div>
        </div>

        <div class="section">
          <span class="section-label">Markings Found</span>
          <hd-markings-list .markings=${c.markings_found}></hd-markings-list>
        </div>

        <div class="section">
          <span class="section-label">Rationale</span>
          <pre class="rationale">${c.rationale}</pre>
        </div>

        ${this.renderValidated()}

        <div class="meta">
          <span>${c.model_name} / ${c.provider_name}</span>
          <span>Classified ${formatDate(c.classified_at)}</span>
        </div>
      </div>

      <div class="actions">
        <button
          class="btn validate-btn"
          @click=${() => (this.mode = "validate")}
        >
          Validate
        </button>
        <button class="btn update-btn" @click=${() => (this.mode = "update")}>
          Update
        </button>
      </div>
    `;
  }

  private renderValidateMode() {
    return html`
      <form class="panel-body" @submit=${this.handleValidate}>
        <p>Confirm that the classification is correct.</p>

        <div class="field">
          <label for="validated_by">Your Name</label>
          <input
            id="validated_by"
            name="validated_by"
            type="text"
            required
            .value=${this.classification?.validated_by ?? ""}
          />
        </div>

        ${this.renderError()}

        <div class="actions">
          <button
            type="submit"
            class="btn validate-btn"
            ?disabled=${this.submitting}
          >
            ${this.submitting ? "Validating..." : "Validate"}
          </button>
          <button
            type="button"
            class="btn cancel-btn"
            @click=${this.handleCancel}
            ?disabled=${this.submitting}
          >
            Cancel
          </button>
        </div>
      </form>
    `;
  }

  private renderUpdateMode() {
    const c = this.classification!;

    return html`
      <form class="panel-body" @submit=${this.handleUpdate}>
        <p>Manually update the classification result.</p>

        <div class="field">
          <label for="classification">Classification</label>
          <input
            id="classification"
            name="classification"
            type="text"
            required
            .value=${c.classification}
          />
        </div>

        <div class="field">
          <label for="rationale">Rationale</label>
          <textarea
            id="rationale"
            name="rationale"
            required
            .value=${c.rationale}
          ></textarea>
        </div>

        <div class="field">
          <label for="updated_by">Your Name</label>
          <input id="updated_by" name="updated_by" type="text" required />
        </div>

        ${this.renderError()}

        <div class="actions">
          <button
            type="submit"
            class="btn update-btn"
            ?disabled=${this.submitting}
          >
            ${this.submitting ? "Updating..." : "Update"}
          </button>
          <button
            type="button"
            class="btn cancel-btn"
            @click=${this.handleCancel}
            ?disabled=${this.submitting}
          >
            Cancel
          </button>
        </div>
      </form>
    `;
  }

  render() {
    if (this.loading) {
      return html`<span class="loading">Loading classifications...</span>`;
    }

    if (!this.classification) {
      return html`
        <div class="empty-state">
          <span>No classification found for this document.</span>
          <button class="button" @click=${this.handleBack}>
            Back to Documents
          </button>
        </div>
      `;
    }

    return html`
      <div class="panel-header">
        <h2>Classification</h2>
      </div>

      ${this.mode === "view"
        ? this.renderViewMode()
        : this.mode === "validate"
          ? this.renderValidateMode()
          : this.renderUpdateMode()}
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-classification-panel": ClassificationPanel;
  }
}
