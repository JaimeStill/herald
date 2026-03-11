# 118 — Inject auth config into web app HTML template

## Summary

Added browser-safe auth config injection from the Go server into the HTML template for MSAL.js initialization. When auth mode is `azure`, a `<script id="herald-config" type="application/json">` tag and `<div id="user-menu">` placeholder are conditionally rendered. When auth mode is `none`, neither element appears.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| `template.JS` wrapping | JSON marshaled to `template.JS` | Prevents `html/template` from HTML-escaping JSON inside `<script>` tag |
| `PageHandlerWithData` | New method on `TemplateSet` | Preserves existing `PageHandler` behavior; explicit about persistent data |
| `ClientAuthConfig` location | `app` package | Browser-safe subset is an app concern; avoids coupling `pkg/auth` to template rendering |
| `RedirectURI` derivation | Set from `basePath + "/"` in `NewModule` | Server knows the base path; client prepends `window.location.origin` |
| Conditional user-menu | Rendered only when auth enabled | No user to display in `none` mode; keeps DOM clean |

## Files Modified

- `app/app.go` — Added `ClientAuthConfig` struct, updated `NewModule` signature and `buildRouter`
- `pkg/web/views.go` — Added `PageHandlerWithData` method
- `app/server/layouts/app.html` — Conditional config script tag and user-menu div
- `cmd/server/modules.go` — Constructs `ClientAuthConfig` from `cfg.Auth` when mode is azure
- `tests/app/app_test.go` — Updated existing tests for new signature, added `TestAuthConfigInjection` and `TestNoAuthConfigWhenNil`

## Patterns Established

- **Browser-safe config subset**: Define a dedicated struct in the consuming package with only client-safe fields, construct from full config at the composition root
- **Conditional template data**: Use `ViewData.Data` with `{{ if .Data }}` for optional template blocks; `template.JS` for safe JSON embedding in `<script>` tags

## Validation Results

- `go vet ./...` — clean
- `go build ./cmd/server/` — compiles
- `go test ./tests/...` — 20/20 packages pass
- New tests: `TestAuthConfigInjection` (verifies all JSON fields + user-menu), `TestNoAuthConfigWhenNil` (verifies absence)
