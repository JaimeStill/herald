# 119 — Add MSAL Auth Service with Login Gate

## Context

Sub-issue 2 of Objective #99 (Web Client MSAL.js Integration). With auth config now injected into the HTML template (#118), the client needs an `Auth` service that wraps `@azure/msal-browser`, handles MSAL initialization, login redirect, and token acquisition. The app bootstrap must gate on authentication before starting the router when auth is enabled.

## Files

| File | Action |
|------|--------|
| `pkg/auth/config.go` | Add `CacheLocation` field to `Config` |
| `internal/config/config.go` | Add `CacheLocation` env var to `authEnv` |
| `app/app.go` | Add `CacheLocation` field to `ClientAuthConfig` |
| `cmd/server/modules.go` | Pass `CacheLocation` through to `ClientAuthConfig` |
| `app/package.json` | Add `@azure/msal-browser` dependency |
| `app/client/core/auth.ts` | **New** — Auth service |
| `app/client/core/index.ts` | Re-export Auth |
| `app/client/app.ts` | Async bootstrap with auth gate |

## Implementation

### 1. Server-side: Add `CacheLocation` field

**`pkg/auth/config.go`** — Add `CacheLocation string` field to `Config` (json: `"cache_location"`). Add to `loadDefaults()` (default `"localStorage"`), `loadEnv()` (via new `Env.CacheLocation`), and `Merge()`. Valid values: `"localStorage"`, `"sessionStorage"`.

**`internal/config/config.go`** — Add `CacheLocation: "HERALD_AUTH_CACHE_LOCATION"` to the `authEnv` var.

**`app/app.go`** — Add `CacheLocation string` field to `ClientAuthConfig` (json: `"cache_location"`).

**`cmd/server/modules.go`** — Pass `cfg.Auth.CacheLocation` through when building `ClientAuthConfig`.

### 2. `app/package.json` — Add dependency

Add `@azure/msal-browser` to dependencies. Run `bun install`.

### 3. `app/client/core/auth.ts` — Auth service (new file)

PascalCase singleton object (matches service pattern) in `core/` (framework infra, not domain).

**Internal state** (module-scoped `let`):
- `msalInstance: PublicClientApplication | null`
- `config: AuthConfig | null`

**`AuthConfig` interface** (unexported, matches server JSON):
```typescript
interface AuthConfig {
  tenant_id: string;
  client_id: string;
  redirect_uri: string;
  authority: string;
  cache_location?: string; // "localStorage" | "sessionStorage", defaults to "localStorage"
}
```

**`readConfig()` (unexported helper)**:
- `document.getElementById("herald-config")`
- If not found → return `null` (auth disabled)
- `JSON.parse(el.textContent!)` → `AuthConfig`

**Exported `Auth` object methods:**

| Method | Behavior |
|--------|----------|
| `isEnabled(): boolean` | `config !== null` |
| `isAuthenticated(): boolean` | Active account exists (false when disabled) |
| `getAccount(): AccountInfo \| null` | Active account or null |
| `async init(): Promise<void>` | readConfig → if null, return. Create MSAL instance (`auth.clientId`, `auth.authority`, `auth.redirectUri`, `cache.cacheLocation: config.cache_location ?? "localStorage"`). `await instance.initialize()`. `await handleRedirectPromise()` → set active account from result or from `getAllAccounts()[0]`. |
| `async getToken(forceRefresh?): Promise<string \| null>` | If disabled/no account → null. `acquireTokenSilent({ scopes: [scope], account, forceRefresh })`. On interaction-required error → `login()`. |
| `async login(): Promise<void>` | `loginRedirect({ scopes: [scope] })`. No-op if disabled. |
| `async logout(): Promise<void>` | `logoutRedirect()`. No-op if disabled. |

**Scope**: `api://${config.client_id}/access_as_user` — derived from config, not a separate field.

**Cache location**: `localStorage` — persists tokens across browser sessions.

### 4. `app/client/core/index.ts` — Re-export

Add `export { Auth } from "./auth"` at top of barrel.

### 5. `app/client/app.ts` — Async bootstrap

Wrap in async IIFE:
1. `await Auth.init()`
2. If `Auth.isEnabled() && !Auth.isAuthenticated()` → `await Auth.login(); return`
3. Create router and start

The `return` after `login()` prevents the router from mounting during redirect. The redirect navigates away, so this is a safety guard.

## Key Decisions

- **Configurable cache location** — `CacheLocation` field on `auth.Config` (default `"localStorage"`), flows through `ClientAuthConfig` to the client. Overridable via `HERALD_AUTH_CACHE_LOCATION` env var or JSON config.
- **Async IIFE** over top-level await — more explicit bootstrap boundary
- **`init()` calls `handleRedirectPromise()`** — completes in-flight redirect on second page load, sets active account
- **No-op when disabled** — all methods are safe when config is absent; no conditional imports needed
- **Scope derived from `client_id`** — `api://<client_id>/access_as_user` convention, no extra config field

## Verification

- `bun install` succeeds with new dependency
- `bun run build` produces `dist/app.js` and `dist/app.css` without errors
- With auth disabled (no config script in DOM): app loads normally, router starts, all existing functionality works
- `go vet ./...` passes
