# Objective Planning — #99 Web Client MSAL.js Integration

## Context

Objective 4 (#98) added backend JWT validation middleware with OIDC discovery, user context extraction, and `validated_by` population. Now the web client needs to acquire tokens from Azure Entra ID via MSAL.js and attach them to API requests. This is the browser-side complement: login flow, token management, and authenticated requests. Auth remains opt-in — when `auth_mode: "none"` (default), the client runs with zero MSAL overhead.

## Transition Closeout

- **Objective #98**: 3/3 sub-issues closed (100%) — closed issue, updated phase.md status to Complete, deleted objective.md.
- **Objective #99**: Marked Active in phase.md.

## Architecture Decisions

1. **Config injection via `<script>` tag** — `<script id="herald-config" type="application/json">` in the HTML template provides MSAL config synchronously at page load. No config API endpoint needed. Go template conditionally renders it only when auth mode is azure.

2. **Browser-safe config subset** — The app module receives a filtered struct (`Mode`, `TenantID`, `ClientID`, `Authority`). `ClientSecret` never reaches the client. `RedirectURI` derived from `BasePath`.

3. **Plain DOM hydration for user menu** — The header is server-rendered HTML. A `<div id="user-menu">` placeholder is hydrated with vanilla JS in `app.ts` rather than creating a Lit element for a static name + logout button.

4. **`acquireTokenSilent` + force-refresh on 401** — `api.ts` calls `Auth.getToken()` before requests (fast from MSAL cache). On 401, force-refresh once, then redirect to login if still unauthorized.

5. **SPA scope convention** — Default scope `api://<client_id>/access_as_user` derived from `client_id` in config. No separate scope config field.

## Sub-Issues (3, strictly sequential: 1 → 2 → 3)

### Sub-Issue 1: Inject auth config into web app HTML template

**Labels:** `feature`, `infrastructure`
**Milestone:** v0.4.0 - Security and Deployment
**Dependencies:** None

**Scope:**
- `app/app.go` — Change `NewModule(basePath)` to accept auth config struct (browser-safe subset: Mode, TenantID, ClientID, Authority). Store on `TemplateSet` or pass through to page handler.
- `pkg/web/views.go` — Add `PageHandlerWithData(layout, view, data)` method (or modify `TemplateSet` to hold persistent data) so `ViewData.Data` gets populated.
- `app/server/layouts/app.html` — Conditionally render `<script id="herald-config" type="application/json">` with `{tenant_id, client_id, redirect_uri, authority}` when `Data` is present. Add `<div id="user-menu"></div>` placeholder in header.
- `cmd/server/modules.go` — Pass auth config subset from `cfg.Auth` to `app.NewModule`.

**Files:** `app/app.go`, `pkg/web/views.go`, `app/server/layouts/app.html`, `cmd/server/modules.go`

### Sub-Issue 2: Add MSAL auth service with login gate

**Labels:** `feature`, `infrastructure`
**Milestone:** v0.4.0 - Security and Deployment
**Dependencies:** Sub-issue 1

**Scope:**
- `app/package.json` — Add `@azure/msal-browser`.
- `app/client/core/auth.ts` — Create `Auth` service (PascalCase singleton): `readConfig()`, `init()`, `getToken(forceRefresh?)`, `getAccount()`, `login()`, `logout()`, `isEnabled()`, `isAuthenticated()`. No-op when config absent.
- `app/client/app.ts` — Async bootstrap: `await Auth.init()`, gate on auth before `router.start()`.
- `app/client/core/index.ts` — Re-export Auth.

**Files:** `app/package.json`, `app/client/core/auth.ts`, `app/client/core/index.ts`, `app/client/app.ts`

### Sub-Issue 3: Authenticated API requests, 401 retry, and user menu

**Labels:** `feature`, `infrastructure`
**Milestone:** v0.4.0 - Security and Deployment
**Dependencies:** Sub-issue 2

**Scope:**
- `app/client/core/api.ts` — `request()` and `stream()` call `Auth.getToken()` and attach `Authorization: Bearer <token>`. Skip when `Auth.isEnabled()` is false. On 401: force-refresh token, retry once, redirect to login on second failure.
- `app/client/app.ts` — After auth init, hydrate `#user-menu` with account name + logout button via plain DOM.
- `app/client/design/app/app.css` — Styles for `#user-menu` (flex, alignment, logout button matching nav link style).

**Files:** `app/client/core/api.ts`, `app/client/app.ts`, `app/client/design/app/app.css`

## Verification

- `go vet ./...` passes after sub-issue 1
- `bun run build` succeeds after sub-issues 2 and 3
- With `auth_mode: "none"`: app loads normally, no config script tag in page source, no MSAL initialization, API calls have no Authorization header
- With `auth_mode: "azure"` + valid Entra config: config script tag present, unauthenticated visit triggers redirect to Entra login, after login the header shows user name with logout button, API calls include Authorization header
