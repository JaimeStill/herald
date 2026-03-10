# 105 — Add credential-based storage constructor

## Summary

Added a `NewWithCredential` constructor to `pkg/storage/` that accepts `azcore.TokenCredential` + service URL for managed identity connections to Azure Blob Storage. Relaxed config validation so each constructor guards its own requirements while `validate()` only checks universal invariants.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Validation split | `validate()` checks universal invariants; constructors guard their own fields | Prevents `Finalize()` from requiring fields only one auth mode needs |
| Separate `ServiceURL` field | New field rather than parsing connection string | Different formats, different deployment configurations — managed identity environments won't have connection strings |

## Files Modified

- `pkg/storage/config.go` — Added `ServiceURL` field to `Config` and `Env`, added merge/env support, relaxed `validate()`
- `pkg/storage/storage.go` — Added `NewWithCredential` constructor, added `ConnectionString` guard in `New`, added `azcore` import
- `internal/config/config.go` — Added `HERALD_STORAGE_SERVICE_URL` env var mapping
- `tests/storage/config_test.go` — Updated validation tests for relaxed `validate()`, added `ServiceURL` env and merge tests

## Patterns Established

- Dual-constructor pattern (`New` / `NewWithCredential`) for connection string vs credential auth — same pattern will apply to `pkg/database/` in #106

## Validation Results

- `go vet ./...` — clean
- `go build ./cmd/server/` — success
- `go test ./tests/...` — all pass (20/20 packages)
