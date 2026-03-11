# 114 + 115 — Wire Auth Middleware and Populate validated_by

## Context

Sub-issues 2 and 3 of Objective #98 (API Authentication Middleware). #114 connects the auth middleware to the API module. #115 uses the authenticated identity in classifications handlers. Combined because #114 is a one-line change and both are needed to complete the objective.

## Changes

### Step 1: `internal/api/api.go` — Register auth middleware (#114)

Add `middleware.Auth(&cfg.Auth, logger)` between CORS and Logger:

```go
m.Use(middleware.CORS(&cfg.API.CORS))
m.Use(middleware.Auth(&cfg.Auth, runtime.Infrastructure.Logger))
m.Use(middleware.Logger(runtime.Infrastructure.Logger))
```

No `APIConfig.Auth` field needed — middleware reads `Config.Auth` directly. No `config.json` changes — top-level `auth` section already provides the config.

### Step 2: `internal/classifications/handler.go` — Override validated_by from JWT (#115)

In the `Validate` handler, after JSON decode and before calling `h.sys.Validate()`:

```go
if user := auth.UserFromContext(r.Context()); user != nil {
    cmd.ValidatedBy = user.Name
}
```

In the `Update` handler, after JSON decode and before calling `h.sys.Update()`:

```go
if user := auth.UserFromContext(r.Context()); user != nil {
    cmd.UpdatedBy = user.Name
}
```

Add `"github.com/JaimeStill/herald/pkg/auth"` to imports.

**Behavior:**
- `auth_mode: "none"` → no user in context → request body value used (existing behavior)
- `auth_mode: "azure"` → user extracted from JWT → overrides request body value

## Files Modified

| File | Change |
|------|--------|
| `internal/api/api.go` | Add `middleware.Auth` call |
| `internal/classifications/handler.go` | Import `pkg/auth`, override `ValidatedBy`/`UpdatedBy` from context |

## Verification

- `go vet ./...` passes
- `go build ./cmd/server/` succeeds
- Server starts with `auth_mode: "none"` — middleware is pass-through, no behavioral change
- Validate/Update handlers still accept body values when no user in context
