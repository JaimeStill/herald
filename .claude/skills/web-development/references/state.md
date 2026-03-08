# Component State

Views and modules manage state with Lit's `@state()` decorator. Modules own their data (fetched from services), filters, pagination, and UI state. Views manage view-level concerns and coordinate between modules. Data flows down via `@property()`, events flow up via `CustomEvent`.

## `@state()` — Primary State Tool

`@state()` is the primary mechanism for all component-owned reactive state. When a `@state()` field changes, Lit automatically triggers a re-render.

### Module State Pattern

Modules own their data. They call services directly, store results in `@state()` fields, and pass data to child elements via `@property()` bindings.

```typescript
@customElement("hd-document-grid")
export class DocumentGrid extends LitElement {
  // Data state — fetched from services
  @state() private documents: PageResult<Document> | null = null;

  // Filter/pagination state — drives fetch parameters
  @state() private page = 1;
  @state() private search = "";
  @state() private status = "";
  @state() private sort = "-UploadedAt";

  // UI state — local concerns
  @state() private classifying = new Map<string, ClassifyProgress>();
  @state() private selectedIds = new Set<string>();
  @state() private deleteDocument: Document | null = null;

  connectedCallback() {
    super.connectedCallback();
    this.fetchDocuments();
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
}
```

### View State Pattern

Views manage view-level UI toggles and coordinate between modules.

```typescript
@customElement("hd-documents-view")
export class DocumentsView extends LitElement {
  @state() private showUpload = false;

  private handleUploadComplete() {
    this.showUpload = false;
    this.renderRoot.querySelector("hd-document-grid")?.refresh();
  }

  render() {
    return html`
      <div class="view">
        ${this.showUpload
          ? html`<hd-document-upload
              @upload-complete=${this.handleUploadComplete}
            ></hd-document-upload>`
          : nothing}
        <hd-document-grid></hd-document-grid>
      </div>
    `;
  }
}
```

## State Initialization

- `null` means "not yet loaded" — show loading indicator
- Empty `PageResult` (`data: []`) means "loaded but empty" — show empty state
- Pagination metadata comes from the server, never hardcoded on the client

## View-to-Module Coordination

Views coordinate modules through two mechanisms:

### `querySelector` + Public Methods

Views call public methods on modules to trigger refreshes or state changes:

```typescript
// View refreshes the grid after an upload
private handleUploadComplete() {
  this.showUpload = false;
  this.renderRoot.querySelector("hd-document-grid")?.refresh();
}
```

### `@property()` for Parent-to-Child Data

Views pass data to modules via `@property()` when modules need external input:

```typescript
// Module accepts selectedId from parent view for highlighting
@property() selectedId = "";
```

## Events for Child-to-Parent Communication

Children dispatch `CustomEvent` with `bubbles: true, composed: true` to notify parents. Parents listen in templates:

```typescript
// Child dispatches
this.dispatchEvent(
  new CustomEvent("select", {
    detail: { prompt: this.prompt },
    bubbles: true,
    composed: true,
  }),
);

// Parent listens
html`<hd-prompt-list
  @select=${this.handleSelect}
></hd-prompt-list>`;
```

## Streaming Orchestration

SSE operations are owned by the **module** managing the collection — not the pure element that triggered the action. The module:

1. Calls the streaming service
2. Tracks per-item progress via `@state()` Map
3. Passes progress data to pure elements as `@property()` bindings
4. Dispatches a completion event upward when done

Pure elements receive all streaming state as properties and dispatch intent events. They have no knowledge of services, SSE, or `AbortController`.

## Conventions

- **`@state()` for all component-owned data**: Fetched results, filters, pagination, progress, errors, UI toggles
- **`@property()` for parent-provided data**: Selected ID, mode flags, data objects from parent
- **Modules call services directly**: No orchestration middleman between services and modules
- **Events up, data down**: Children dispatch `CustomEvent`, parents bind `@property()`
- **Views coordinate, modules own data**: Views manage UI toggles and relay between modules; modules own their domain data
- **Public `refresh()` methods**: Modules expose refresh for parent views to call after cross-module events
- **`null` vs empty**: Use `null` for "not loaded", empty results for "loaded but nothing found"
