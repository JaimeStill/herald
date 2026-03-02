# 64 — Go Web App Module, Server Integration, and Dev Experience

## Summary

Wired the Go web app module that embeds and serves the Lit 3.x client built in #63, using the `pkg/web/` infrastructure from #62. The app module mounts at `/app` alongside the existing API module at `/api`. Configured Air for Go hot reload and added mise tasks for the two-terminal dev workflow.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Static asset serving | `web.DistServer()` helper | Reuses existing `pkg/web/static.go` infrastructure instead of raw `http.FileServer` |
| SPA fallback | `web.Router.SetFallback()` | Cleaner than registering `/{path...}` — uses the built-in unmatched route mechanism |
| No public/ embed | Omitted | Herald has no favicon/manifest assets yet — can add later without structural changes |
| Air exclude strategy | `exclude_dir` for client subdirs | `include_dir` only supports top-level dirs; excluding `app/client`, `app/scripts`, `app/plugins`, `app/node_modules` prevents unnecessary Go rebuilds on client source changes |

## Files Modified

- `app/app.go` — Created: Go embedding module with `NewModule` factory
- `app/server/layouts/app.html` — Created: HTML shell template with base href, nav, content container
- `app/server/views/shell.html` — Created: Empty content block for client-side routing
- `cmd/server/modules.go` — Modified: Added App module creation and mounting
- `.air.toml` — Created: Air hot reload configuration
- `.mise.toml` — Modified: Added `web:build` and `web:watch` tasks
- `tests/app/app_test.go` — Created: Tests for module creation, template rendering, asset serving, SPA fallback

## Patterns Established

- **Web app module pattern**: `app/app.go` with `//go:embed` directives + `NewModule(basePath)` factory, `app/server/layouts/` for Go HTML templates, `app/server/views/` for content blocks
- **Two-terminal dev workflow**: `bun run watch` (client rebuild) + `air` (Go rebuild on dist/ changes)
- **Module.Serve test pattern**: Tests must include the module prefix in request URLs (e.g., `/app/dist/app.js` not `/dist/app.js`) since `Module.Serve` strips the prefix

## Validation Results

- `go build ./cmd/server` — compiles successfully
- `go vet ./...` — passes
- `go test ./tests/...` — 20/20 packages pass (4 new app tests)
- Manual verification: shell template renders at `/app/`, static assets serve, SPA fallback works, Air hot reload triggers on `dist/` changes
