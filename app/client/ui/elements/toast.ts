import { LitElement, html, nothing } from "lit";
import { customElement, state } from "lit/decorators.js";

import styles from "./toast.module.css";

/** Semantic category of a toast, controlling color and auto-dismiss duration. */
export type ToastKind = "success" | "error" | "warning" | "info";

/** A single toast notification as rendered by {@link ToastContainer}. */
export interface ToastItem {
  id: string;
  kind: ToastKind;
  message: string;
}

type Listener = (toasts: ToastItem[]) => void;

const DURATIONS: Record<ToastKind, number> = {
  success: 3000,
  info: 3000,
  warning: 5000,
  error: 6000,
};

const MAX_VISIBLE = 5;

class ToastBus {
  private toasts: ToastItem[] = [];
  private listeners = new Set<Listener>();
  private timers = new Map<string, number>();

  subscribe(fn: Listener): () => void {
    this.listeners.add(fn);
    fn(this.toasts);
    return () => {
      this.listeners.delete(fn);
    };
  }

  push(kind: ToastKind, message: string): string {
    const id = crypto.randomUUID();
    this.toasts = [...this.toasts, { id, kind, message }];

    while (this.toasts.length > MAX_VISIBLE) {
      const oldest = this.toasts[0];
      this.clearTimer(oldest.id);
      this.toasts = this.toasts.slice(1);
    }

    this.timers.set(
      id,
      window.setTimeout(() => this.dismiss(id), DURATIONS[kind]),
    );
    this.emit();
    return id;
  }

  dismiss(id: string): void {
    this.clearTimer(id);
    const next = this.toasts.filter((t) => t.id !== id);
    if (next.length === this.toasts.length) return;
    this.toasts = next;
    this.emit();
  }

  private clearTimer(id: string) {
    const timer = this.timers.get(id);
    if (timer !== undefined) {
      clearTimeout(timer);
      this.timers.delete(id);
    }
  }

  private emit() {
    for (const fn of this.listeners) fn(this.toasts);
  }
}

const bus = new ToastBus();

/**
 * Global toast notification service. Call from any component to surface
 * transient feedback for a command execution. Kind-specific helpers push
 * onto the shared stack rendered by {@link ToastContainer}; `dismiss` and
 * `subscribe` are lower-level hooks (the element uses them internally).
 *
 * Auto-dismiss durations: success/info 3s, warning 5s, error 6s.
 * Stack cap: 5 visible — oldest is evicted.
 */
export const Toast = {
  /** Push a success toast. Returns the toast id (rarely needed). */
  success(message: string): string {
    return bus.push("success", message);
  },
  /** Push an error toast. Returns the toast id (rarely needed). */
  error(message: string): string {
    return bus.push("error", message);
  },
  /** Push a warning toast. Returns the toast id (rarely needed). */
  warning(message: string): string {
    return bus.push("warning", message);
  },
  /** Push an info toast. Returns the toast id (rarely needed). */
  info(message: string): string {
    return bus.push("info", message);
  },
  /** Remove a specific toast early (cancels its auto-dismiss timer). */
  dismiss(id: string): void {
    bus.dismiss(id);
  },
  /** Subscribe to the toast stack. Returns an unsubscribe function. */
  subscribe(fn: Listener): () => void {
    return bus.subscribe(fn);
  },
};

/**
 * Global toast stack. Subscribes to the {@link Toast} service and renders
 * the active queue. Mount once at the shell (see `app.ts`); additional
 * instances will each render their own view of the same stack.
 * Toasts dismiss on click or after their kind-specific duration.
 */
@customElement("hd-toast-container")
export class ToastContainer extends LitElement {
  static styles = [styles];

  @state() private toasts: ToastItem[] = [];

  private unsubscribe?: () => void;

  connectedCallback() {
    super.connectedCallback();
    this.setAttribute("popover", "manual");
    this.showPopover();
    this.unsubscribe = Toast.subscribe((toasts) => {
      this.toasts = toasts;
    });
  }

  disconnectedCallback() {
    this.unsubscribe?.();
    this.unsubscribe = undefined;
    if (this.matches(":popover-open")) this.hidePopover();
    super.disconnectedCallback();
  }

  private handleDismiss(id: string) {
    Toast.dismiss(id);
  }

  render() {
    if (this.toasts.length === 0) return nothing;

    return html`
      <div class="stack" role="status" aria-live="polite">
        ${this.toasts.map(
          (t) => html`
            <button
              type="button"
              class="toast ${t.kind}"
              @click=${() => this.handleDismiss(t.id)}
            >
              ${t.message}
            </button>
          `,
        )}
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-toast-container": ToastContainer;
  }
}
