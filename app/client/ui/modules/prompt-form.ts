import { LitElement, html, nothing } from "lit";
import { customElement, property, state } from "lit/decorators.js";

import { PromptService } from "@domains/prompts";
import type { Prompt } from "@domains/prompts";
import { Toast } from "@ui/elements";

import buttonStyles from "@styles/buttons.module.css";
import inputStyles from "@styles/inputs.module.css";
import labelStyles from "@styles/labels.module.css";
import scrollStyles from "@styles/scroll.module.css";
import styles from "./prompt-form.module.css";

/**
 * Stateful module for creating and editing prompts.
 * Null prompt = create mode, populated prompt = edit mode.
 * Fetches and displays the stage specification and default instructions
 * in a collapsible reference panel. Dispatches `save` and `cancel` events.
 */
@customElement("hd-prompt-form")
export class PromptForm extends LitElement {
  static styles = [
    buttonStyles,
    inputStyles,
    labelStyles,
    scrollStyles,
    styles,
  ];

  @property({ type: Object }) prompt: Prompt | null = null;

  @state() private submitting = false;
  @state() private error = "";
  @state() private defaultInstructions = "";
  @state() private spec = "";

  updated(changed: Map<string, unknown>) {
    if (changed.has("prompt") && this.prompt) {
      this.fetchDefaults(this.prompt.stage);
    }
  }

  private get isEdit() {
    return this.prompt !== null;
  }

  private async fetchDefaults(stage: string) {
    this.spec = "";
    this.defaultInstructions = "";

    const typedStage = stage as Prompt["stage"];

    const [specResult, instrResult] = await Promise.all([
      PromptService.spec(typedStage),
      PromptService.instructions(typedStage, true),
    ]);

    if (specResult.ok) this.spec = specResult.data.content;
    if (instrResult.ok) this.defaultInstructions = instrResult.data.content;
  }

  private handleStageChange(e: Event) {
    const stage = (e.target as HTMLSelectElement).value;
    if (stage) this.fetchDefaults(stage);
  }

  private async handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    this.error = "";
    this.submitting = true;

    const form = e.target as HTMLFormElement;
    const data = new FormData(form);

    const name = (data.get("name") as string).trim();
    const stage = (data.get("stage") as string) ?? this.prompt!.stage;
    const instructions = (data.get("instructions") as string).trim();
    const description = (data.get("description") as string).trim();

    const command = {
      name,
      stage: stage as Prompt["stage"],
      instructions,
      ...(description && { description }),
    };

    const result = this.isEdit
      ? await PromptService.update(this.prompt!.id, command)
      : await PromptService.create(command);

    this.submitting = false;

    if (!result.ok) {
      this.error = result.error ?? "An unexpected error occurred.";
      Toast.error(
        `Failed to ${this.isEdit ? "update" : "create"} prompt: ${this.error}`,
      );
      return;
    }

    Toast.success(`Prompt ${this.isEdit ? "updated" : "created"}`);

    this.dispatchEvent(
      new CustomEvent("save", {
        detail: { prompt: result.data },
        bubbles: true,
        composed: true,
      }),
    );
  }

  private handleCancel() {
    this.error = "";
    this.dispatchEvent(
      new CustomEvent("cancel", {
        bubbles: true,
        composed: true,
      }),
    );
  }

  private renderDefaults() {
    if (!this.spec && !this.defaultInstructions) return nothing;

    return html`
      <details class="defaults">
        <summary>Default Prompt</summary>
        <div class="defaults-content scroll-y">
          ${this.spec
            ? html`
                <h4>Specification</h4>
                <pre>${this.spec}</pre>
              `
            : nothing}
          ${this.defaultInstructions
            ? html`
                <h4>Default Instructions</h4>
                <pre>${this.defaultInstructions}</pre>
              `
            : nothing}
        </div>
      </details>
    `;
  }

  private renderError() {
    if (!this.error) return nothing;
    return html`<div class="error">${this.error}</div>`;
  }

  render() {
    const p = this.prompt;

    return html`
      <div class="form-header">
        <h2>${this.isEdit ? "Edit Prompt" : "New Prompt"}</h2>
      </div>
      <div class="form-description">
        Instructions are combined with the stage specification to form the
        complete prompt. Override the default instructions below to customize
        behavior.
      </div>
      <form class="form-body scroll-y" @submit=${this.handleSubmit}>
        <div class="field">
          <label class="label" for="name">Name</label>
          <input
            class="input"
            id="name"
            name="name"
            type="text"
            required
            .value=${p?.name ?? ""}
          />
        </div>
        <div class="field">
          <label class="label" for="stage">Stage</label>
          <select
            class="input"
            id="stage"
            name="stage"
            required
            ?disabled=${this.isEdit}
            @change=${this.handleStageChange}
          >
            <option value="" ?selected=${!p}>---</option>
            <option value="classify" ?selected=${p?.stage === "classify"}>
              Classify
            </option>
            <option value="enhance" ?selected=${p?.stage === "enhance"}>
              Enhance
            </option>
            <option value="finalize" ?selected=${p?.stage === "finalize"}>
              Finalize
            </option>
          </select>
        </div>
        ${this.renderDefaults()}
        <div class="field">
          <label class="label" for="instructions">Instructions</label>
          <textarea
            class="input instructions"
            id="instructions"
            name="instructions"
            required
            .value=${p?.instructions ?? ""}
          ></textarea>
        </div>
        <div class="field">
          <label class="label" for="description">Description</label>
          <textarea
            class="input"
            id="descripion"
            name="description"
            .value=${p?.description ?? ""}
          ></textarea>
        </div>
        ${this.renderError()}
        <div class="actions">
          <button
            type="submit"
            class="btn btn-green"
            ?disabled=${this.submitting}
          >
            ${this.submitting ? "Saving..." : "Save"}
          </button>
          <button
            type="button"
            class="btn btn-muted"
            @click=${this.handleCancel}
            ?disabled=${this.submitting}
          >
            Cancel
          </button>
        </div>
      </form>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-prompt-form": PromptForm;
  }
}
