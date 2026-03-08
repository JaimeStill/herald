# Lit Lifecycle Hooks

Herald components use Lit's lifecycle hooks for data loading, cleanup, and reactive side effects. Each hook has a specific role — using the wrong one creates bugs (e.g., async work in `updated()` causes double-fetches).

## `connectedCallback()`

Called when the element is added to the DOM. Use for initial data fetching in modules that own their data.

```typescript
connectedCallback() {
  super.connectedCallback();
  this.fetchDocuments();
}
```

Always call `super.connectedCallback()` first. This is the standard entry point for modules that load data on mount.

## `disconnectedCallback()`

Called when the element is removed from the DOM. Use for cleanup: timers, abort controllers, blob URLs, event listeners.

```typescript
disconnectedCallback() {
  super.disconnectedCallback();
  clearTimeout(this.searchTimer);
  for (const controller of this.abortControllers.values()) {
    controller.abort();
  }
}
```

Always call `super.disconnectedCallback()` first. Forgetting cleanup here causes memory leaks and orphaned network requests.

## `updated(changed)`

Called synchronously after the component re-renders. Receives a `Map<string, unknown>` of changed properties (keys are property names, values are previous values).

**Use for synchronous side effects** — host attribute reflection, DOM measurements, focus management.

```typescript
updated(changed: Map<string, unknown>) {
  if (changed.has("dragover")) {
    this.toggleAttribute("dragover", this.dragover);
  }
}
```

**Also used for property-change reactions in modules** that need to trigger data loading when a `@property()` changes:

```typescript
updated(changed: Map<string, unknown>) {
  if (changed.has("documentId") && this.documentId) {
    this.loadClassification();
  }
}
```

This pattern is used by modules like `classification-panel` that receive a property from a parent view and need to load data when it changes. The async work happens in the called method, not in `updated()` itself.

## `willUpdate(changed)`

Called before the render cycle. Can be async but Lit does not await it — the render proceeds immediately. Use sparingly.

**Use for async data loading triggered by route parameter changes** in views:

```typescript
async willUpdate(changed: Map<string, unknown>) {
  if (changed.has("documentId") && this.documentId) {
    this.document = undefined;
    this.error = undefined;
    await this.loadDocument(this.documentId);
  }
}
```

This pattern is used by `review-view` where the router sets `documentId` as an HTML attribute. The view clears state and loads data when the parameter changes.

## When to Use Which

| Scenario | Hook | Example |
|----------|------|---------|
| Initial data fetch on mount | `connectedCallback` | `document-grid`, `prompt-list` |
| Cleanup timers/controllers/URLs | `disconnectedCallback` | `document-grid`, `document-upload` |
| Host attribute reflection | `updated` | `document-upload` (dragover) |
| React to `@property()` change in a module | `updated` | `classification-panel` (documentId) |
| React to route param change in a view | `willUpdate` | `review-view` (documentId) |

## Anti-Patterns

- **Don't fetch data in `connectedCallback` when it depends on a property** — the property may not be set yet. Use `updated()` with a `changed.has()` guard instead.
- **Don't use `willUpdate` for synchronous side effects** — use `updated()` which runs after render.
- **Don't forget `super.*Callback()`** — Lit's base class needs these calls for internal bookkeeping.
