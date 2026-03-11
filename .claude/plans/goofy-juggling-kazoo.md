# Plan: #118 — Inject auth config into web app HTML template

## Context

The Lit SPA needs MSAL configuration at page load to initialize `@azure/msal-browser` (sub-issue #119). The Go server has all auth config in `pkg/auth.Config`, but `app.NewModule` currently receives only a `basePath` string and `ViewData.Data` is always empty. This task injects a browser-safe subset of auth config into the HTML template via a `<script type="application/json">` tag, conditionally rendered only when auth mode is azure.

## Approach

Four files, each with a small targeted change:

### 1. `app/app.go` — Define `ClientAuthConfig`, update `NewModule` signature

- Add `ClientAuthConfig` struct with browser-safe fields only: `TenantID`, `ClientID`, `RedirectURI`, `Authority` (JSON-tagged as snake_case)
- Change `NewModule(basePath string)` → `NewModule(basePath string, authCfg *ClientAuthConfig)`
- When `authCfg` is non-nil: set `RedirectURI = basePath + "/"`, marshal to JSON, wrap in `template.JS` (prevents html/template from HTML-escaping the JSON inside `<script>`)
- Pass the `template.JS` value to `buildRouter` → `PageHandlerWithData`
- When `authCfg` is nil (auth mode "none"): pass `nil` as data — template skips the config block

### 2. `pkg/web/views.go` — Add `PageHandlerWithData` method

- New method: `PageHandlerWithData(layout string, view ViewDef, data any) http.HandlerFunc`
- Identical to `PageHandler` but sets `ViewData.Data = data`
- Existing `PageHandler` unchanged — no impact on other consumers

### 3. `app/server/layouts/app.html` — Conditional config script + user-menu placeholder

- Inside `<head>`, before the closing `</head>`: `{{ if .Data }}<script id="herald-config" type="application/json">{{ .Data }}</script>{{ end }}`
- In `<header>`, after `<nav>`: `{{ if .Data }}<div id="user-menu"></div>{{ end }}` — only rendered when auth is enabled (no user to display otherwise)

### 4. `cmd/server/modules.go` — Pass auth config from `cfg.Auth`

- Import `auth` package, construct `*app.ClientAuthConfig` when `cfg.Auth.Mode == auth.ModeAzure`
- Pass `nil` when mode is `"none"`
- Pass the config to `app.NewModule(basePath, authCfg)`

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| `template.JS` for JSON | Wrap marshaled JSON in `template.JS` | `html/template` applies JS escaping inside `<script>` tags — `template.JS` marks it as safe |
| `PageHandlerWithData` | New method, not modify existing | No impact on existing `PageHandler` callers; explicit about intent |
| `RedirectURI` from `basePath` | Set in `NewModule` as `basePath + "/"` | Server knows the base path; client (in #119) prepends `window.location.origin` |
| `ClientAuthConfig` in `app` pkg | Not in `pkg/auth` | Browser-safe subset is an app concern; avoids coupling `pkg/auth` to template rendering |

## Files Modified

- `app/app.go`
- `pkg/web/views.go`
- `app/server/layouts/app.html`
- `cmd/server/modules.go`

## Validation

- `go vet ./...` passes
- `go build ./cmd/server/` compiles
- With auth mode `none` (default): no `<script id="herald-config">` in rendered HTML
- With auth mode `azure` + tenant/client config: `<script id="herald-config">` contains valid JSON with `tenant_id`, `client_id`, `redirect_uri`, `authority`
- `<div id="user-menu"></div>` present in header regardless of auth mode
