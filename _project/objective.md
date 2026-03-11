# Objective: Web Client MSAL.js Integration

**Issue:** [#99](https://github.com/JaimeStill/herald/issues/99)
**Phase:** Phase 4 — Security and Deployment (v0.4.0)

## Scope

Add browser-side Azure Entra authentication to the Lit SPA using MSAL.js. Provide login flow, token acquisition, and authenticated API requests. Conditionally skipped when auth is disabled — zero-friction local development preserved.

### What's Covered

- Inject MSAL config from Go server into HTML template via `<script>` tag
- Create `Auth` service wrapping `@azure/msal-browser` (login, logout, token acquisition)
- Login gate — redirect to Entra login before rendering the SPA when auth enabled
- Attach `Authorization: Bearer <token>` to all API requests
- Handle 401 responses with silent token refresh and retry
- User display in app header with logout option

### What's Not Covered

- OBO flow — Herald uses managed identity for downstream calls
- Server-side JWT validation (Objective 4, #98 — complete)

## Sub-Issues

| # | Title | Issue | Status | Dependencies |
|---|-------|-------|--------|--------------|
| 1 | Inject auth config into web app HTML template | [#118](https://github.com/JaimeStill/herald/issues/118) | Open | None |
| 2 | Add MSAL auth service with login gate | [#119](https://github.com/JaimeStill/herald/issues/119) | Open | #118 |
| 3 | Wire token injection into API layer and add header user menu | [#120](https://github.com/JaimeStill/herald/issues/120) | Open | #119 |

Sub-issues are strictly sequential: 1 → 2 → 3.

## Architecture Decisions

- **Config injection via `<script>` tag** — `<script id="herald-config" type="application/json">` in the HTML template provides MSAL config synchronously at page load. Go template conditionally renders it only when auth mode is azure. No config API endpoint needed.
- **Browser-safe config subset** — The app module receives a filtered struct (Mode, TenantID, ClientID, Authority). `ClientSecret` never reaches the client. `RedirectURI` derived from `BasePath`.
- **Plain DOM hydration for user menu** — The header is server-rendered HTML. A `<div id="user-menu">` placeholder is hydrated with vanilla JS in `app.ts` rather than creating a Lit element for a static name + logout button.
- **`acquireTokenSilent` + force-refresh on 401** — `api.ts` calls `Auth.getToken()` before requests (fast from MSAL cache). On 401, force-refresh once, then redirect to login if still unauthorized.
- **SPA scope convention** — Default scope `api://<client_id>/access_as_user` derived from `client_id` in config. No separate scope config field.
