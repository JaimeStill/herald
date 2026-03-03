import { LitElement, html } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import { WORKFLOW_STAGES } from '@app/classifications';
import type { WorkflowStage } from '@app/classifications';
import styles from './classify-progress.module.css'

/**
 * Pure element that renders a horizontal 4-stage pipeline indicator
 * for the classification workflow. Receives progress state as properties.
 */
@customElement('hd-classify-progress')
export class ClassifyProgress extends LitElement {
  static styles = styles;

  @property() currentNode: WorkflowStage | null = null;
  @property({ type: Array }) completedNodes: WorkflowStage[] = [];

  private stageState(stage: WorkflowStage): string {
    if (this.completedNodes.includes(stage)) return 'completed';
    if (stage === this.currentNode) return 'active';
    return 'pending';
  }

  render() {
    return html`
      <div class="pipeline">
        ${WORKFLOW_STAGES.map((stage) => html`
            <div class="stage ${this.stageState(stage)}">
              <div class="indicator"></div>
              <span class="label">${stage}</span>
            </div>
        `)}
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'hd-classify-progress': ClassifyProgress;
  }
}
