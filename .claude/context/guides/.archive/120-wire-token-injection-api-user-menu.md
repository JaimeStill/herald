# 120 — Wire Token Injection into API Layer and Add Header User Menu

## Problem Context

The MSAL auth service (#119) can acquire tokens and verify authentication state, but the API transport layer (`api.ts`) sends unauthenticated requests. This sub-issue connects `Auth.getToken()` to `request()` and `stream()` so all API calls carry bearer tokens when auth is enabled, handles 401 retry with silent token refresh, and displays the authenticated user in the app header.

## Architecture Approach

Token injection is a cross-cutting concern handled at the transport layer. A module-private `authHeaders()` helper centralizes token acquisition for both `request()` and `stream()`. The 401 retry uses a single force-refresh attempt — if that fails, the session is invalid and a full re-login is appropriate. The user menu uses plain DOM hydration because the header is server-rendered HTML (a Lit element would be over-engineering for a static name + button).

## Implementation

### Step 1: Add token injection and 401 retry to `api.ts`

**File: `app/client/core/api.ts`**

Add import at the top of the file (after no existing imports — this is the first):

```ts
import { Auth } from "./auth";
```

Add `authHeaders()` helper after the `ExecutionEvent` interface (before the `request` function):

```ts
async function authHeaders(): Promise<Record<string, string>> {
  if (!Auth.isEnabled()) return {};
  const token = await Auth.getToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}
```

Replace the `request()` function with:

```ts
export async function request<T>(
  path: string,
  init?: RequestInit,
  parse: (res: Response) => Promise<T> = (res) => res.json(),
): Promise<Result<T>> {
  try {
    const headers = { ...init?.headers, ...(await authHeaders()) };
    const opts: RequestInit = { ...init, headers };
    let res = await fetch(`${BASE}${path}`, opts);

    if (res.status === 401 && Auth.isEnabled()) {
      const token = await Auth.getToken(true);
      if (token) {
        opts.headers = { ...opts.headers, Authorization: `Bearer ${token}` };
        res = await fetch(`${BASE}${path}`, opts);
      }
      if (res.status === 401) {
        await Auth.login();
        return { ok: false, error: "Authentication required" };
      }
    }

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
```

Replace the `stream()` function with:

```ts
export function stream(
  path: string,
  options: StreamOptions,
  init?: RequestInit,
): AbortController {
  const controller = new AbortController();
  const signal = options.signal ?? controller.signal;

  (async () => {
    try {
      const headers = { ...init?.headers, ...(await authHeaders()) };
      let opts: RequestInit = { ...init, headers, signal };
      let res = await fetch(`${BASE}${path}`, opts);

      if (res.status === 401 && Auth.isEnabled()) {
        const token = await Auth.getToken(true);
        if (token) {
          opts = {
            ...opts,
            headers: { ...opts.headers, Authorization: `Bearer ${token}` },
          };
          res = await fetch(`${BASE}${path}`, opts);
        }
        if (res.status === 401) {
          await Auth.login();
          return;
        }
      }

      if (!res.ok) {
        const text = await res.text();
        options.onError?.(text || res.statusText);
        return;
      }

      const reader = res.body?.getReader();
      if (!reader) {
        options.onError?.("No response body");
        return;
      }

      const decoder = new TextDecoder();
      let buffer = "";
      let currentEvent = "message";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() ?? "";

        for (const line of lines) {
          if (line.startsWith("event: ")) {
            currentEvent = line.slice(7).trim();
          } else if (line.startsWith("data: ")) {
            const data = line.slice(6).trim();
            options.onEvent(currentEvent, data);
            currentEvent = "message";
          }
        }
      }

      options.onComplete?.();
    } catch (err: unknown) {
      if ((err as Error).name !== "AbortError") {
        options.onError?.((err as Error).message);
      }
    }
  })();

  return controller;
}
```

The key structural change to `stream()` is wrapping the body in an async IIFE so `authHeaders()` can be awaited before the initial fetch. The 401 retry pattern mirrors `request()`.

### Step 2: Hydrate user menu in `app.ts`

**File: `app/client/app.ts`**

Add user menu hydration after `router.start()`, inside the existing async IIFE:

```ts
(async () => {
  await Auth.init();

  if (Auth.isEnabled() && !Auth.isAuthenticated()) {
    await Auth.login();
    return;
  }

  const router = new Router("app-content", routes);
  router.start();

  if (Auth.isEnabled()) {
    const account = Auth.getAccount();
    const menu = document.getElementById("user-menu");
    if (account && menu) {
      const name = document.createElement("span");
      name.className = "user-name";
      name.textContent = account.name ?? account.username;

      const logout = document.createElement("button");
      logout.className = "user-logout";
      logout.textContent = "Logout";
      logout.addEventListener("click", () => Auth.logout());

      menu.append(name, logout);
    }
  }
})();
```

No new imports needed — `Auth` is already imported.

### Step 3: Style `#user-menu` in `app.css`

**File: `app/client/design/app/app.css`**

Add inside the `@layer app` block, after the `#app-content` rule:

```css
  #user-menu {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  #user-menu .user-name {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    color: var(--color-1);
  }

  #user-menu .user-logout {
    font-family: var(--font-mono);
    font-size: var(--text-sm);
    color: var(--color-1);
    background: none;
    border: none;
    cursor: pointer;
    padding: 0;

    &:hover {
      color: var(--blue);
    }
  }
```

Logout button matches the nav link style: no border, no background, color transition on hover. Both elements use `--font-mono` per the design convention for interactive elements.

## Remediation

### R1: Configurable OAuth scope

The original design hardcoded the scope name as `access_as_user` in `auth.ts`. The Entra app registration used `access` instead, causing `AADSTS65005` (scope not found). Fix: added `Scope` field to `pkg/auth/Config`, `app.ClientAuthConfig`, and `AuthConfig` in `auth.ts`. The server stores just the scope claim name (e.g., `access`); the client composes the full `api://<client_id>/<scope>` format. Default derived in `deriveDefaults()` is `access_as_user` when not configured.

**Files:** `pkg/auth/config.go`, `internal/config/config.go`, `app/app.go`, `cmd/server/modules.go`, `app/client/core/auth.ts`, `config.auth.json`

### R2: OIDC verifier audience and issuer mismatch

The auth middleware's OIDC verifier expected `aud` = raw client ID GUID, but Entra access tokens for custom scopes use `aud` = `api://<client_id>`. Additionally, Entra access tokens use the v1 issuer format (`https://sts.windows.net/<tenant>/`) even from the v2.0 endpoint, but OIDC discovery returns the v2.0 issuer. Fix: set verifier audience to `"api://" + cfg.ClientID` and `SkipIssuerCheck: true`. Signature and expiry are still validated.

**File:** `pkg/middleware/auth.go`

### R3: Header nav centering with three flex children

Adding `#user-menu` as a third child of `.app-header` (with `justify-content: space-between`) pushed nav to the center. Fix: replaced `space-between` with `gap`, added `margin-left: auto` on nav to push it right. Added `border-left` divider and `padding-left` on `#user-menu` for visual separation.

**File:** `app/client/design/app/app.css`

### R4: PDF iframe unauthorized when auth enabled

The `hd-blob-viewer` iframe loaded PDFs via direct URL (`/api/storage/view/:key`), which can't carry a bearer token. Fix: when auth is enabled, the review view fetches the PDF through `StorageService.download()` (which goes through the authenticated `request()` path), creates a `blob:` URL, and passes that to `hd-blob-viewer`. Falls back to direct URL when auth is disabled. Blob URLs are revoked on document change and disconnect.

**File:** `app/client/ui/views/review-view.ts`

## Validation Criteria

- [ ] `bun run build` succeeds
- [ ] `request()` attaches `Authorization: Bearer <token>` when auth is enabled
- [ ] `stream()` attaches `Authorization: Bearer <token>` when auth is enabled
- [ ] When auth is disabled (no config script), no token acquisition occurs
- [ ] 401 responses trigger one silent token refresh retry, then redirect to login
- [ ] App header displays authenticated user's name when logged in
- [ ] Logout button calls `Auth.logout()` and redirects to Entra
- [ ] `#user-menu` remains empty when auth is disabled
