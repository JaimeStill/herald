# 62 — pkg/web/ Template and Static File Infrastructure

## Summary

Ported the `pkg/web/` package from agent-lab to Herald, providing template pre-parsing, static file serving, and SPA-compatible routing. Removed the placeholder `doc.go` and created three files: `views.go`, `static.go`, and `router.go`.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Port approach | Direct port with import path change only | agent-lab patterns are proven; the issue explicitly calls for this approach |
| Test scope | Router, ServeEmbeddedFile, type construction | TemplateSet/DistServer/PublicFile require embed.FS which can't be constructed in tests; real integration tested when web/app uses the package |

## Files Modified

- `pkg/web/doc.go` — removed (placeholder)
- `pkg/web/views.go` — created (TemplateSet, ViewDef, ViewData, NewTemplateSet, PageHandler, ErrorHandler, Render)
- `pkg/web/static.go` — created (DistServer, PublicFile, PublicFileRoutes, ServeEmbeddedFile)
- `pkg/web/router.go` — created (Router, NewRouter, SetFallback, Handle, HandleFunc, ServeHTTP)
- `tests/web/router_test.go` — created (4 tests)
- `tests/web/views_test.go` — created (2 tests)
- `tests/web/static_test.go` — created (2 tests)

## Patterns Established

- `pkg/web/` is the Go-side web infrastructure package — template rendering, static assets, and routing live here
- Package doc comment is on `views.go` (the primary file) since `doc.go` was removed

## Validation Results

- `go vet ./...` passes
- `go build ./...` compiles cleanly
- All 19 test packages pass (8 web tests added)
