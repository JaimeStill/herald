# 62 — pkg/web/ Template and Static File Infrastructure

## Context

Issue #62 is the first sub-issue of Objective #57 (Web Client Foundation). It establishes the Go-side web infrastructure that the web app module (#64) will depend on. The `pkg/web/` package provides template pre-parsing, static file serving, and a router wrapper — all ported from the proven `~/code/agent-lab/pkg/web/` package.

## Approach

Direct port of three files from agent-lab with Herald-specific import path adjustments. The API surface, behavior, and structure remain identical — agent-lab's patterns are proven and Herald's issue explicitly calls for this approach.

## Files

### Remove
- `pkg/web/doc.go` — placeholder, replaced by the real package

### Create
- **`pkg/web/views.go`** — `TemplateSet` (pre-parsed template cache with layout inheritance), `ViewDef`, `ViewData`, `PageHandler()`, `ErrorHandler()`, `Render()`
- **`pkg/web/static.go`** — `DistServer()`, `PublicFile()`, `PublicFileRoutes()`, `ServeEmbeddedFile()`
- **`pkg/web/router.go`** — `Router` wrapper around `http.ServeMux` with optional fallback handler

### Adaptations from agent-lab
1. **Import path**: `github.com/JaimeStill/agent-lab/pkg/routes` → `github.com/JaimeStill/herald/pkg/routes`
2. **No godoc comments** in the guide (added in Phase 7)
3. Herald's `pkg/routes/route.go` already has the identical `Route` type — no changes needed there

## Verification

- `go vet ./...` passes
- `go build ./...` compiles cleanly
- Package is importable and types are accessible
