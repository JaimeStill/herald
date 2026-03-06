# 88 — PDF Viewer Element and Storage Inline Endpoint

## Problem Context

The document review view (#61) needs to display PDFs inline so reviewers can see the document alongside its classification. This requires a backend endpoint that streams blobs with `Content-Disposition: inline` (browsers render PDFs natively in iframes) and a pure Lit element that wraps the iframe.

## Architecture Approach

The `view` handler mirrors the existing `download` handler exactly, differing only in the `Content-Disposition` value (`inline` vs `attachment`). The `hd-blob-viewer` element is a generic pure element — accepts a `src` URL and renders an iframe. It has no knowledge of storage routes, blob types, or documents. The caller constructs the URL and passes it in. The review view composes `hd-blob-viewer` by loading the document via `DocumentService.find()` and building the `/api/storage/view/` URL from `storage_key`.

## Implementation

### Step 1: Add `view` handler and route (`internal/api/storage.go`)

Add route to `routes()` between `download` and `find`:

```go
{Method: "GET", Pattern: "/view/{key...}", Handler: h.view},
```

Add `view` method after the `download` method:

```go
func (h *storageHandler) view(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	result, err := h.store.Download(r.Context(), key)
	if err != nil {
		handlers.RespondError(
			w, h.logger,
			storage.MapHTTPStatus(err), err,
		)
		return
	}
	defer result.Body.Close()

	w.Header().Set("Content-Type", result.ContentType)

	if result.ContentLength > 0 {
		w.Header().Set(
			"Content-Length",
			strconv.FormatInt(result.ContentLength, 10),
		)
	}
	w.Header().Set(
		"Content-Disposition",
		fmt.Sprintf("inline; filename=%q", path.Base(key)),
	)
	w.WriteHeader(http.StatusOK)
	io.Copy(w, result.Body)
}
```

### Step 2: Add `view` to `StorageService` (`app/client/domains/storage/service.ts`)

Add method to the service object:

```typescript
  /** Builds the inline view URL for a blob by storage key. */
  view(key: string): string {
    return `/api${base}/view/${key}`;
  },
```

This keeps the `/api/storage/view/` route construction inside the service boundary. Callers never assemble API paths.

### Step 3: Create `hd-blob-viewer` element (`app/client/ui/elements/blob-viewer.ts`)

New file:

```typescript
import { LitElement, html, nothing } from "lit";
import { customElement, property } from "lit/decorators.js";

import styles from "./blob-viewer.module.css";

@customElement("hd-blob-viewer")
export class BlobViewer extends LitElement {
  static styles = styles;

  @property() override title = "Blob viewer";
  @property() src?: string;

  render() {
    if (!this.src) return nothing;

    return html`
      <iframe
        src=${this.src}
        title=${this.title}
      ></iframe>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-blob-viewer": BlobViewer;
  }
}
```

### Step 4: Create blob viewer styles (`app/client/ui/elements/blob-viewer.module.css`)

New file:

```css
:host {
  display: flex;
  flex: 1;
  min-height: 0;
}

iframe {
  flex: 1;
  border: 1px solid var(--divider);
  border-radius: var(--radius-md);
}
```

### Step 5: Update elements barrel (`app/client/ui/elements/index.ts`)

Add between existing exports (alphabetically before `ClassifyProgress`):

```typescript
export { BlobViewer } from "./blob-viewer";
```

### Step 6: Update review view (`app/client/ui/views/review-view.ts`)

Replace the entire placeholder implementation:

```typescript
import { LitElement, html, nothing } from "lit";
import { customElement, property, state } from "lit/decorators.js";

import { navigate } from "@core/router";
import type { Document } from "@domains/documents";
import { DocumentService } from "@domains/documents";
import { StorageService } from "@domains/storage";

import buttonStyles from "@styles/buttons.module.css";
import styles from "./review-view.module.css";

@customElement("hd-review-view")
export class ReviewView extends LitElement {
  static styles = [buttonStyles, styles];

  @property() documentId?: string;
  @state() private document?: Document;
  @state() private error?: string;

  async willUpdate(changed: Map<string, unknown>) {
    if (changed.has("documentId") && this.documentId) {
      this.document = undefined;
      this.error = undefined;

      const result = await DocumentService.find(this.documentId);

      if (result.ok) {
        this.document = result.data;
      } else {
        this.error = result.error;
      }
    }
  }

  private handleBack() {
    navigate("");
  }

  render() {
    if (this.error) {
      return html`
        <div class="error">
          <p>${this.error}</p>
          <button class="button" @click=${this.handleBack}>
            Back to Documents
          </button>
        </div>
      `;
    }

    if (!this.document) {
      return html`<div class="loading">Loading document...</div>`;
    }

    return html`
      <div class="panel pdf-panel">
        <hd-blob-viewer
          .title=${this.document.filename}
          .src=${StorageService.view(this.document.storage_key)}
        ></hd-blob-viewer>
      </div>
      <div class="panel classification-panel">
        <h2>${this.document.filename}</h2>
        <p class="status">${this.document.status}</p>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "hd-review-view": ReviewView;
  }
}
```

### Step 7: Update review view styles (`app/client/ui/views/review-view.module.css`)

Replace all styles:

```css
:host {
  display: flex;
  gap: var(--space-4);
  padding: var(--space-4);
  flex: 1;
  min-height: 0;
}

.panel {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.pdf-panel {
  flex: 3;
}

.classification-panel {
  flex: 2;
  padding: var(--space-4);
  border: 1px solid var(--divider);
  border-radius: var(--radius-md);
  overflow-y: auto;
}

.classification-panel h2 {
  margin-bottom: var(--space-2);
  font-family: var(--font-mono);
  font-size: var(--text-base);
  word-break: break-all;
}

.classification-panel .status {
  color: var(--color-1);
  font-family: var(--font-mono);
  font-size: var(--text-sm);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.loading,
.error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  gap: var(--space-4);
  color: var(--color-1);
  font-family: var(--font-mono);
}
```

## Remediation

### R1: Missing route registration

The `view` handler was implemented but the route `GET /view/{key...}` was not added to `routes()`. The `/{key...}` wildcard on `find` was catching view requests, returning "blob not found". Fixed by adding the route before the catch-all.

## Validation Criteria

- [ ] `go vet ./...` passes
- [ ] `bun run build` succeeds
- [ ] `GET /api/storage/view/{key}` streams PDF with `Content-Disposition: inline`
- [ ] `hd-blob-viewer` renders content in iframe given a `src` URL
- [ ] Navigate to `/app/review/:documentId` — PDF renders in the left panel
