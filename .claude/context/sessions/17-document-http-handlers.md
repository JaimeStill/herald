# 17 - Document HTTP Handlers

## Summary

Built the HTTP presentation layer for the document domain: Handler struct with 5 endpoints (List, Find, Search, Upload, Delete), multipart upload processing with pdfcpu PDF page count extraction, configurable max upload size, and full API route wiring. Also removed the OpenAPI/Scalar infrastructure (which had unresolvable multipart form rendering bugs) and replaced it with API Cartographer — a markdown-based API documentation convention with executable curl examples.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Batch upload endpoint | Removed in favor of single-file uploads | `ParseMultipartForm(maxMemory)` caps total request memory, making per-file limits unpredictable in a batch. Single-file uploads give deterministic per-file limits; the web client coordinates parallel uploads via `Promise.allSettled` |
| OpenAPI/Scalar | Removed entirely | Scalar 1.43.2 had unresolvable multipart form data rendering bugs. Extensive debugging (removing fields, encoding, properties) could not fix the file picker |
| API documentation | API Cartographer (markdown + curl) | Lightweight, terminal-native approach. Claude skill at `.claude/skills/api-cartographer/` generates `_project/api/` specs with `curl -s ... \| jq .` examples |
| Content type detection | Prefer multipart header, fall back to `http.DetectContentType` | Multipart headers from browsers are usually accurate; `DetectContentType` handles the `application/octet-stream` fallback |
| PDF page count | pdfcpu, non-fatal on failure | Logs warning and sets `page_count` to nil if extraction fails |

## Files Modified

**Created:**
- `pkg/formatting/bytes.go` — ParseBytes/FormatBytes with base-1024 units
- `internal/documents/handler.go` — Handler struct, 5 endpoints, helpers
- `.claude/skills/api-cartographer/SKILL.md` — API Cartographer skill definition
- `.claude/skills/api-cartographer/references/templates.md` — Endpoint templates
- `_project/api/README.md` — Root API reference
- `_project/api/documents.md` — Documents route group spec
- `tests/formatting/bytes_test.go` — 28 test cases
- `tests/documents/handler_test.go` — 16 test cases

**Modified:**
- `internal/config/api.go` — MaxUploadSize field, MaxUploadSizeBytes() method
- `internal/documents/system.go` — Handler() on System interface
- `internal/documents/repository.go` — Handler() implementation on repo
- `internal/api/routes.go` — Wire document handler routes
- `internal/api/api.go` — Remove OpenAPI spec building
- `cmd/server/modules.go` — Remove Scalar module
- `pkg/routes/group.go` — Remove OpenAPI fields
- `pkg/routes/route.go` — Remove OpenAPI field
- `config.toml` — Remove `[api.openapi]` section
- `.mise.toml` — Remove web tasks
- `Dockerfile` — Remove Bun/web build stage
- `.claude/CLAUDE.md` — Update AI responsibilities
- `_project/README.md` — Update bulk upload decision
- `tests/config/config_test.go` — Add MaxUploadSizeBytes tests
- `tests/api/api_test.go` — Remove OpenAPI references
- `tests/routes/routes_test.go` — Remove OpenAPI dependencies
- `tests/module/module_test.go` — Remove Scalar references

**Deleted:**
- `pkg/openapi/` — Entire OpenAPI package (5 files)
- `web/` — Entire Scalar UI directory (11 files)
- `tests/openapi/openapi_test.go`
- `internal/documents/openapi.go`

## Patterns Established

- **Mock System pattern for handler tests**: Create a `mockSystem` struct with function fields (e.g., `listFn`, `findFn`) implementing the System interface. Inject into `NewHandler` for isolated HTTP endpoint testing without database dependencies.
- **API Cartographer convention**: Markdown-based API specs in `_project/api/` with route group files, `curl -s | jq .` examples, and a root README with config/setup/group index. Maintained by AI as a skill responsibility.
- **Byte size formatting**: `pkg/formatting` provides `ParseBytes`/`FormatBytes` for human-readable byte sizes used in config values like `MaxUploadSize`.

## Validation Results

- `go vet ./...` — clean
- `go build ./cmd/server/` — clean
- 15 test suites, all passing (44 new test cases added across 3 files)
