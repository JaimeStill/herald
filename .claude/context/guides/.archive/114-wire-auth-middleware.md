# 114 + 115 — Wire Auth Middleware and Populate validated_by

## Problem Context

Objective #98 (API Authentication Middleware) has three sub-issues. #113 (complete) added the `pkg/auth` package and `pkg/middleware` auth middleware. This session wires the middleware into the API module (#114) and uses the authenticated identity in classifications handlers (#115) to complete the objective.

## Architecture Approach

The auth middleware already accepts `*auth.Config` directly and handles `ModeNone` pass-through internally. No mapping layer or `APIConfig.Auth` field is needed — the middleware reads from the existing `Config.Auth`. The classifications handlers conditionally override `ValidatedBy`/`UpdatedBy` from the JWT context, preserving backward compatibility when auth is disabled.

## Implementation

### Step 1: Register auth middleware in API module

**File:** `internal/api/api.go`

Add `middleware.Auth` between CORS and Logger in `NewModule()`:

```go
m.Use(middleware.CORS(&cfg.API.CORS))
m.Use(middleware.Auth(&cfg.Auth, runtime.Infrastructure.Logger))
m.Use(middleware.Logger(runtime.Infrastructure.Logger))
```

### Step 2: Override validated_by from authenticated user

**File:** `internal/classifications/handler.go`

Add `"github.com/JaimeStill/herald/pkg/auth"` to imports.

In the `Validate` handler, after JSON decode and before `h.sys.Validate()`:

```go
if user := auth.UserFromContext(r.Context()); user != nil {
    cmd.ValidatedBy = user.Name
}
```

In the `Update` handler, after JSON decode and before `h.sys.Update()`:

```go
if user := auth.UserFromContext(r.Context()); user != nil {
    cmd.UpdatedBy = user.Name
}
```

## Validation Criteria

- [ ] `go vet ./...` passes
- [ ] `go build ./cmd/server/` succeeds
- [ ] Auth middleware registered between CORS and Logger in `api.NewModule()`
- [ ] `auth_mode: "none"` (default) — middleware is pass-through, no behavioral change
- [ ] Validate/Update handlers override identity from JWT when user is in context
- [ ] Validate/Update handlers preserve request body value when no user in context
