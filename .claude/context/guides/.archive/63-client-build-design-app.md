# 63 - Web Client Build System, Design, and Client Application

## Problem Context

Herald Phase 3 requires a Lit 3.x web client embedded in the Go binary. This sub-issue (#63) establishes the complete client-side foundation: Bun build pipeline with a CSS module plugin that discriminates `*.module.css` component styles from plain CSS global styles, a design system using CSS cascade layers, a History API router, a core API layer with `Result<T>` error handling, and placeholder views for all routes.

The root directory is `app/` (with Go `package app` in issue #64). Patterns adapted from agent-lab's web client with Bun replacing Vite.

## Architecture Approach

**CSS module plugin**: Bun plugin intercepts `*.module.css` imports via `onResolve`/`onLoad` hooks, emitting JavaScript modules that construct `CSSStyleSheet` objects. Lit 3+ accepts `CSSStyleSheet` directly in `static styles`. Plain `*.css` imports fall through to Bun's default pipeline, which extracts them to `dist/app.css`. The file itself declares its intent: `my-view.module.css` → CSSStyleSheet, `index.css` → extracted to app.css.

**Build output**: Fixed filenames (`app.js`, `app.css`) in `dist/` for stable `go:embed` globs in issue #64.

**Router**: Pure JavaScript History API router with `<base href>` awareness, `:param` pattern matching, and dynamic custom element mounting. Adapted from agent-lab.

**API layer**: `Result<T>` discriminated union for error handling. Herald-specific SSE streaming with simple `onMessage`/`onError`/`onComplete` callbacks (not OpenAI chat completion format).

## Implementation

### Step 1: Package and TypeScript Setup

**`app/package.json`**:

```json
{
  "name": "herald-app",
  "private": true,
  "type": "module",
  "scripts": {
    "build": "bun scripts/build.ts",
    "watch": "bun scripts/watch.ts"
  },
  "devDependencies": {
    "@types/bun": "^1.2.0"
  },
  "dependencies": {
    "@lit-labs/signals": "^0.2.0",
    "@lit/context": "^1.1.6",
    "lit": "^3.3.2"
  }
}
```

**`app/tsconfig.json`**:

```json
{
  "compilerOptions": {
    "target": "ES2024",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "noEmit": true,
    "skipLibCheck": true,
    "esModuleInterop": true,
    "allowSyntheticDefaultImports": true,
    "experimentalDecorators": true,
    "useDefineForClassFields": false,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "paths": {
      "@app/*": ["./client/*"]
    }
  },
  "include": ["client/**/*"],
  "exclude": ["node_modules", "dist"]
}
```

**`app/client/css.d.ts`**:

```typescript
declare module '*.module.css' {
  const sheet: CSSStyleSheet;
  export default sheet;
}
```

### Step 2: CSS Module Plugin

**`app/plugins/css-modules.ts`**:

```typescript
import type { BunPlugin } from 'bun';

export const litCSSModulePlugin: BunPlugin = {
  name: 'lit-css-module',
  setup(build) {
    build.onResolve({ filter: /\.module\.css$/ }, (args) => {
      return {
        path: Bun.resolveSync(args.path, args.resolveDir),
        namespace: 'lit-css',
      };
    });

    build.onLoad({ filter: /\.module\.css$/, namespace: 'lit-css' }, async (args) => {
      const css = await Bun.file(args.path).text();
      const escaped = css.replace(/`/g, '\\`').replace(/\$/g, '\\$');

      return {
        contents: `
const sheet = new CSSStyleSheet();
sheet.replaceSync(\`${escaped}\`);
export default sheet;
`,
        loader: 'js',
      };
    });
  },
};
```

### Step 3: Build and Watch Scripts

**`app/scripts/build.ts`**:

```typescript
import { litCSSModulePlugin } from '../plugins/css-modules';

const result = await Bun.build({
  entrypoints: ['client/app.ts'],
  outdir: 'dist',
  naming: 'app.[ext]',
  plugins: [litCSSModulePlugin],
  minify: false,
});

if (!result.success) {
  console.error('Build failed:');
  for (const log of result.logs) {
    console.error(log);
  }
  process.exit(1);
}

console.log('Build complete: dist/app.js, dist/app.css');
```

**`app/scripts/watch.ts`**:

```typescript
import { watch } from 'fs';
import { join } from 'path';

const CLIENT_DIR = join(import.meta.dir, '..', 'client');
const BUILD_SCRIPT = join(import.meta.dir, 'build.ts');

let timeout: ReturnType<typeof setTimeout> | null = null;

async function rebuild() {
  console.log('Rebuilding...');
  const proc = Bun.spawn(['bun', BUILD_SCRIPT], {
    cwd: join(import.meta.dir, '..'),
    stdout: 'inherit',
    stderr: 'inherit',
  });
  await proc.exited;
}

function debounceRebuild() {
  if (timeout) clearTimeout(timeout);
  timeout = setTimeout(rebuild, 150);
}

await rebuild();

console.log('Watching client/ for changes...');
watch(CLIENT_DIR, { recursive: true }, (event, filename) => {
  if (filename && (filename.endsWith('.ts') || filename.endsWith('.css'))) {
    debounceRebuild();
  }
});
```

### Step 4: Design System

**`app/client/design/index.css`**:

```css
@layer tokens, reset, theme;

@import url(./core/tokens.css);
@import url(./core/reset.css);
@import url(./core/theme.css);

@import url(./app/app.css);
```

**`app/client/design/core/tokens.css`**:

```css
@layer tokens {
  :root {
    color-scheme: dark light;

    --font-sans: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
    --font-mono: ui-monospace, "Cascadia Code", "Source Code Pro", Menlo, Consolas, "DejaVu Sans Mono", monospace;

    --space-1: 0.25rem;
    --space-2: 0.5rem;
    --space-3: 0.75rem;
    --space-4: 1rem;
    --space-5: 1.25rem;
    --space-6: 1.5rem;
    --space-8: 2rem;
    --space-10: 2.5rem;
    --space-12: 3rem;
    --space-16: 4rem;

    --text-xs: 0.75rem;
    --text-sm: 0.875rem;
    --text-base: 1rem;
    --text-lg: 1.125rem;
    --text-xl: 1.25rem;
    --text-2xl: 1.5rem;
    --text-3xl: 1.875rem;
    --text-4xl: 2.25rem;

    --radius-sm: 0.25rem;
    --radius-md: 0.5rem;
    --radius-lg: 0.75rem;

    --shadow-sm: 0 1px 2px hsl(0 0% 0% / 0.05);
    --shadow-md: 0 4px 6px hsl(0 0% 0% / 0.1);
    --shadow-lg: 0 10px 15px hsl(0 0% 0% / 0.15);
  }

  @media (prefers-color-scheme: dark) {
    :root {
      --bg: hsl(0, 0%, 7%);
      --bg-1: hsl(0, 0%, 12%);
      --bg-2: hsl(0, 0%, 18%);
      --color: hsl(0, 0%, 93%);
      --color-1: hsl(0, 0%, 80%);
      --color-2: hsl(0, 0%, 65%);
      --divider: hsl(0, 0%, 25%);

      --blue: hsl(210, 100%, 70%);
      --blue-bg: hsl(210, 50%, 20%);
      --green: hsl(140, 70%, 55%);
      --green-bg: hsl(140, 40%, 18%);
      --red: hsl(0, 85%, 65%);
      --red-bg: hsl(0, 50%, 20%);
      --yellow: hsl(45, 90%, 60%);
      --yellow-bg: hsl(45, 50%, 18%);
      --orange: hsl(25, 95%, 65%);
      --orange-bg: hsl(25, 50%, 20%);
    }
  }

  @media (prefers-color-scheme: light) {
    :root {
      --bg: hsl(0, 0%, 100%);
      --bg-1: hsl(0, 0%, 96%);
      --bg-2: hsl(0, 0%, 92%);
      --color: hsl(0, 0%, 10%);
      --color-1: hsl(0, 0%, 30%);
      --color-2: hsl(0, 0%, 45%);
      --divider: hsl(0, 0%, 80%);

      --blue: hsl(210, 90%, 45%);
      --blue-bg: hsl(210, 80%, 92%);
      --green: hsl(140, 60%, 35%);
      --green-bg: hsl(140, 50%, 90%);
      --red: hsl(0, 70%, 50%);
      --red-bg: hsl(0, 70%, 93%);
      --yellow: hsl(45, 80%, 40%);
      --yellow-bg: hsl(45, 80%, 88%);
      --orange: hsl(25, 85%, 50%);
      --orange-bg: hsl(25, 75%, 90%);
    }
  }
}
```

**`app/client/design/core/reset.css`**:

```css
@layer reset {

  *,
  *::before,
  *::after {
    box-sizing: border-box;
  }

  * {
    margin: 0;
  }

  body {
    min-height: 100svh;
    line-height: 1.5;
  }

  img,
  picture,
  video,
  canvas,
  svg {
    display: block;
    max-width: 100%;
  }

  @media (prefers-reduced-motion: no-preference) {
    :has(:target) {
      scroll-behavior: smooth;
    }
  }
}
```

**`app/client/design/core/theme.css`**:

```css
@layer theme {
  body {
    font-family: var(--font-sans);
    background-color: var(--bg);
    color: var(--color);
  }

  pre,
  code {
    font-family: var(--font-mono);
  }
}
```

**`app/client/design/app/app.css`**:

```css
body {
  display: flex;
  flex-direction: column;
  height: 100dvh;
  margin: 0;
  overflow: hidden;
}

.app-header {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-6);
  background: var(--bg-1);
  border-bottom: 1px solid var(--divider);
}

.app-header .brand {
  font-size: var(--text-lg);
  font-weight: 600;
  color: var(--color);
  text-decoration: none;
}

.app-header .brand:hover {
  color: var(--blue);
}

.app-header nav {
  display: flex;
  gap: var(--space-4);
}

.app-header nav a {
  color: var(--color-1);
  text-decoration: none;
  font-size: var(--text-sm);
}

.app-header nav a:hover {
  color: var(--blue);
}

#app-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

#app-content > * {
  flex: 1;
  min-height: 0;
}
```

**`app/client/design/app/elements.css`**:

```css
/* Base element styles for Shadow DOM components.
   Import in component CSS via:
   import styles from './my-component.module.css'; */
```

### Step 5: Client-Side Router

**`app/client/router/types.ts`**:

```typescript
export interface RouteConfig {
  component: string;
  title: string;
}

export interface RouteMatch {
  config: RouteConfig;
  params: Record<string, string>;
  query: Record<string, string>;
}
```

**`app/client/router/routes.ts`**:

```typescript
import type { RouteConfig } from './types';

export const routes: Record<string, RouteConfig> = {
  '': { component: 'hd-documents-view', title: 'Documents' },
  'prompts': { component: 'hd-prompts-view', title: 'Prompts' },
  'review/:documentId': { component: 'hd-review-view', title: 'Review' },
  '*': { component: 'hd-not-found-view', title: 'Not Found' },
};
```

**`app/client/router/router.ts`**:

```typescript
import { routes } from './routes';
import type { RouteMatch } from './types';

let routerInstance: Router | null = null;

export function navigate(path: string): void {
  routerInstance?.navigate(path);
}

export class Router {
  private container: HTMLElement;
  private basePath: string;

  constructor(containerId: string) {
    const el = document.getElementById(containerId);
    if (!el) throw new Error(`Container #${containerId} not found`);

    this.container = el;
    this.basePath = document
      .querySelector('base')
      ?.getAttribute('href')
      ?.replace(/\/$/, '')
      ?? '/app';

    routerInstance = this;
  }

  navigate(path: string, pushState: boolean = true): void {
    const [pathPart, queryPart] = path.split('?');
    const normalized = this.normalizePath(pathPart);
    const query = this.parseQuery(queryPart);
    const match = this.match(normalized, query);

    if (pushState) {
      let fullPath = `${this.basePath}/${normalized}`.replace(/\/+/g, '/');
      if (queryPart) fullPath += `?${queryPart}`;
      history.pushState(null, '', fullPath);
    }

    document.title = `${match.config.title} - Herald`;
    this.mount(match);
  }

  start(): void {
    this.navigate(this.currentPath(), false);

    window.addEventListener('popstate', () => {
      this.navigate(this.currentPath(), false);
    });
  }

  private currentPath(): string {
    const pathname = location.pathname;

    if (pathname.startsWith(this.basePath))
      return pathname
        .slice(this.basePath.length)
        .replace(/^\//, '');

    return pathname.replace(/^\//, '');
  }

  private match(path: string, query: Record<string, string>): RouteMatch {
    const segments = path.split('/').filter(Boolean);

    if (routes[path])
      return { config: routes[path], params: {}, query };

    for (const [pattern, config] of Object.entries(routes)) {
      if (pattern === '*') continue;

      const patternSegments = pattern.split('/').filter(Boolean);

      if (patternSegments.length !== segments.length) continue;

      const params: Record<string, string> = {};
      let matched = true;

      for (let i = 0; i < patternSegments.length; i++) {
        const pat = patternSegments[i];
        const seg = segments[i];

        if (pat.startsWith(':')) {
          params[pat.slice(1)] = seg;
        } else if (pat !== seg) {
          matched = false;
          break;
        }
      }

      if (matched) {
        return { config, params, query };
      }
    }

    return { config: routes['*'], params: { path }, query };
  }

  private mount(match: RouteMatch): void {
    this.container.innerHTML = '';
    const el = document.createElement(match.config.component);

    for (const [key, value] of Object.entries(match.params)) {
      el.setAttribute(key, value);
    }

    for (const [key, value] of Object.entries(match.query)) {
      el.setAttribute(key, value);
    }

    this.container.appendChild(el);
  }

  private normalizePath(path: string): string {
    let normalized = path.replace(/^\//, '');
    const baseWithoutSlash = this.basePath.replace(/^\//, '');

    if (normalized.startsWith(baseWithoutSlash))
      normalized = normalized
        .slice(baseWithoutSlash.length)
        .replace(/^\//, '');

    return normalized;
  }

  private parseQuery(queryString?: string): Record<string, string> {
    if (!queryString) return {};

    const params = new URLSearchParams(queryString);
    const result: Record<string, string> = {};
    for (const [key, value] of params) {
      result[key] = value;
    }
    return result;
  }
}
```

**`app/client/router/index.ts`**:

```typescript
export { Router, navigate } from './router';
export type { RouteConfig, RouteMatch } from './types';
```

### Step 6: Core API Layer

**`app/client/core/api.ts`**:

```typescript
const BASE = '/api';

export type Result<T> =
  | { ok: true; data: T }
  | { ok: false; error: string };

export async function request<T>(
  path: string,
  init?: RequestInit,
  parse: (res: Response) => Promise<T> = (res) => res.json()
): Promise<Result<T>> {
  try {
    const res = await fetch(`${BASE}${path}`, init);
    if (!res.ok) {
      const text = await res.text();
      return { ok: false, error: text || res.statusText };
    }
    if (res.status === 204) {
      return { ok: true, data: undefined as T };
    }
    return { ok: true, data: await parse(res) };
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : String(e) };
  }
}

export interface StreamOptions {
  onMessage: (data: string) => void;
  onError?: (error: string) => void;
  onComplete?: () => void;
  signal?: AbortSignal;
}

export function stream(
  path: string,
  options: StreamOptions
): AbortController {
  const controller = new AbortController();
  const signal = options.signal ?? controller.signal;

  fetch(`${BASE}${path}`, { signal })
    .then(async (res) => {
      if (!res.ok) {
        const text = await res.text();
        options.onError?.(text || res.statusText);
        return;
      }

      const reader = res.body?.getReader();
      if (!reader) {
        options.onError?.('No response body');
        return;
      }

      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() ?? '';

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            const data = line.slice(6).trim();
            if (data === '[DONE]') {
              options.onComplete?.();
              return;
            }
            options.onMessage(data);
          }
        }
      }

      options.onComplete?.();
    })
    .catch((err: Error) => {
      if (err.name !== 'AbortError') {
        options.onError?.(err.message);
      }
    });

  return controller;
}

export interface PageResult<T> {
  data: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface PageRequest {
  page?: number;
  page_size?: number;
  search?: string;
  sort?: string;
}

export function toQueryString(params: PageRequest): string {
  const entries = Object.entries(params)
    .filter(([, v]) => v !== undefined && v !== null && v !== '')
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`);

  return entries.length > 0
    ? `?${entries.join('&')}`
    : '';
}
```

**`app/client/core/index.ts`**:

```typescript
export { request, stream, toQueryString } from './api';
export type { Result, StreamOptions, PageResult, PageRequest } from './api';
```

### Step 7: Placeholder Views

**`app/client/views/not-found-view.module.css`**:

```css
:host {
  display: flex;
  align-items: center;
  justify-content: center;
}

.container {
  text-align: center;
}

h1 {
  margin-bottom: var(--space-4);
}

p {
  color: var(--color-1);
}

a {
  color: var(--blue);
  text-decoration: none;
}

a:hover {
  text-decoration: underline;
}
```

**`app/client/views/not-found-view.ts`**:

```typescript
import { LitElement, html } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import styles from './not-found-view.module.css';

@customElement('hd-not-found-view')
export class NotFoundView extends LitElement {
  static styles = styles;

  @property({ type: String }) path?: string;

  render() {
    return html`
      <div class="container">
        <h1>404</h1>
        <p>Page not found${this.path ? html`: /${this.path}` : ''}</p>
        <a href="">Return home</a>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'hd-not-found-view': NotFoundView;
  }
}
```

**`app/client/views/documents-view.module.css`**:

```css
:host {
  display: flex;
  align-items: center;
  justify-content: center;
}

.container {
  text-align: center;
}

h1 {
  margin-bottom: var(--space-4);
}

p {
  color: var(--color-1);
}
```

**`app/client/views/documents-view.ts`**:

```typescript
import { LitElement, html } from 'lit';
import { customElement } from 'lit/decorators.js';
import styles from './documents-view.module.css';

@customElement('hd-documents-view')
export class DocumentsView extends LitElement {
  static styles = styles;

  render() {
    return html`
      <div class="container">
        <h1>Documents</h1>
        <p>Document management interface.</p>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'hd-documents-view': DocumentsView;
  }
}
```

**`app/client/views/prompts-view.module.css`**:

```css
:host {
  display: flex;
  align-items: center;
  justify-content: center;
}

.container {
  text-align: center;
}

h1 {
  margin-bottom: var(--space-4);
}

p {
  color: var(--color-1);
}
```

**`app/client/views/prompts-view.ts`**:

```typescript
import { LitElement, html } from 'lit';
import { customElement } from 'lit/decorators.js';
import styles from './prompts-view.module.css';

@customElement('hd-prompts-view')
export class PromptsView extends LitElement {
  static styles = styles;

  render() {
    return html`
      <div class="container">
        <h1>Prompts</h1>
        <p>Prompt management interface.</p>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'hd-prompts-view': PromptsView;
  }
}
```

**`app/client/views/review-view.module.css`**:

```css
:host {
  display: flex;
  align-items: center;
  justify-content: center;
}

.container {
  text-align: center;
}

h1 {
  margin-bottom: var(--space-4);
}

p {
  color: var(--color-1);
}
```

**`app/client/views/review-view.ts`**:

```typescript
import { LitElement, html } from 'lit';
import { customElement, property } from 'lit/decorators.js';
import styles from './review-view.module.css';

@customElement('hd-review-view')
export class ReviewView extends LitElement {
  static styles = styles;

  @property({ type: String }) documentId?: string;

  render() {
    return html`
      <div class="container">
        <h1>Review</h1>
        <p>Classification review for document ${this.documentId ?? 'unknown'}.</p>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'hd-review-view': ReviewView;
  }
}
```

### Step 8: Entry Point

**`app/client/app.ts`**:

```typescript
import './design/index.css';

import { Router } from '@app/router';

import './views/documents-view';
import './views/prompts-view';
import './views/review-view';
import './views/not-found-view';

const router = new Router('app-content');
router.start();
```

### Step 9: Gitignore and Initial Build

Add to `.gitignore`:

```
app/node_modules/
app/dist/
```

Then run:

```bash
cd app && bun install && bun run build
```

## Remediation

### R1: CSS module plugin — `.module.css` naming convention replaces import attributes

Bun 1.3.10 does not expose import attributes (`with { type: 'css' }`) to plugin `onResolve`/`onLoad` hooks. The `OnResolveArgs` interface only has `path`, `importer`, `namespace`, `resolveDir`, and `kind` — no `with`, `attributes`, or `importAttributes` property. This is a known gap tracked in [oven-sh/bun#7293](https://github.com/oven-sh/bun/issues/7293) and [#16147](https://github.com/oven-sh/bun/issues/16147).

**Fix**: Use the `*.module.css` naming convention to discriminate component CSS from global CSS. The file itself declares its intent — the plugin filters on `/\.module\.css$/` and emits a `CSSStyleSheet` JS module. Plain `*.css` imports fall through to Bun's default extraction pipeline. This is fail-safe: you can't accidentally import a `.module.css` file without the plugin catching it.

- Component CSS: `import styles from './my-view.module.css'` → inlined in `app.js` as `CSSStyleSheet`
- Global CSS: `import './design/index.css'` → extracted to `dist/app.css`

The `css.d.ts` declaration targets `*.module.css` to provide type safety for component style imports.

## Validation Criteria

- [ ] `bun install` succeeds with lit, @lit/context, @lit-labs/signals
- [ ] `bun run build` produces `dist/app.js` and `dist/app.css`
- [ ] CSS module plugin correctly discriminates `*.module.css` imports from plain `*.css` imports
- [ ] Component CSS inlined in `app.js` as CSSStyleSheet modules (not in `app.css`)
- [ ] Global design CSS extracted to `dist/app.css`
- [ ] `bun run watch` rebuilds on `client/**/*.{ts,css}` changes
- [ ] Router navigates between placeholder views (fully testable after #64)
- [ ] `Result<T>` API layer compiles with correct types
- [ ] All 4 placeholder views render with `hd-` prefix
