# 120 — Wire Token Injection into API Layer and Add Header User Menu

## Context

Sub-issue 3 of Objective #99 (Web Client MSAL.js Integration). The auth service from #119 provides `Auth.getToken()`, `Auth.isEnabled()`, and `Auth.getAccount()`. This task connects it to the API transport layer so all requests carry bearer tokens, handles 401 retry with silent token refresh, and displays the authenticated user in the app header.

## Files Modified

1. **`app/client/core/api.ts`** — Token injection + 401 retry
2. **`app/client/app.ts`** — User menu hydration
3. **`app/client/design/app/app.css`** — `#user-menu` styles

## Implementation

### 1. `app/client/core/api.ts` — Token injection and 401 retry

**Helper: `authHeaders()`** — async function that returns `{ Authorization: Bearer <token> }` when auth is enabled, empty object otherwise. Called by both `request()` and `stream()`.

```ts
async function authHeaders(): Promise<HeadersInit> {
  if (!Auth.isEnabled()) return {};
  const token = await Auth.getToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}
```

**`request()` changes:**
- Before fetch, call `authHeaders()` and merge into `init.headers`
- After non-ok response: if status is 401, call `Auth.getToken(true)` (force refresh), rebuild headers, retry fetch once. If retry also 401, call `Auth.login()`.
- Extract the fetch + response handling into a helper to avoid duplicating the parse logic.

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

**`stream()` changes:**
- Before fetch, merge `authHeaders()` into init. Same 401 pattern but adapted for streaming: check initial response status, force-refresh + retry once, then login redirect.

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
          opts = { ...opts, headers: { ...opts.headers, Authorization: `Bearer ${token}` } };
          res = await fetch(`${BASE}${path}`, opts);
        }
        if (res.status === 401) {
          await Auth.login();
          return;
        }
      }

      if (!res.ok) { ... } // existing error handling
      // ... existing stream reading logic
    } catch (err: unknown) {
      if ((err as Error).name !== "AbortError") {
        options.onError?.((err as Error).message);
      }
    }
  })();

  return controller;
}
```

Import `Auth` at the top of `api.ts`:
```ts
import { Auth } from "./auth";
```

### 2. `app/client/app.ts` — User menu hydration

After `router.start()`, hydrate `#user-menu` with the account display name and a logout button using plain DOM manipulation:

```ts
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
```

### 3. `app/client/design/app/app.css` — `#user-menu` styles

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

Logout button styled to match nav link style (no border, no background, color transition on hover).

## Validation

- [ ] `bun run build` succeeds
- [ ] `request()` attaches `Authorization: Bearer <token>` when auth is enabled
- [ ] `stream()` attaches `Authorization: Bearer <token>` when auth is enabled
- [ ] When auth is disabled (no config script), no token acquisition occurs
- [ ] 401 responses trigger one silent token refresh retry, then redirect to login
- [ ] App header displays authenticated user's name when logged in
- [ ] Logout button calls `Auth.logout()`
- [ ] `#user-menu` remains empty when auth is disabled
