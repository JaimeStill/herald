# 113 — Add Auth Context Package and JWT Validation Middleware

## Problem Context

Objective #98 (API Authentication Middleware) needs a foundation layer of JWT validation infrastructure before the middleware can be wired into the API module (#114) and user identity can flow to domain handlers (#115). This sub-issue establishes `pkg/auth/` as the unified auth package and adds JWT validation middleware to `pkg/middleware/`.

## Architecture Approach

**Unified auth config in `pkg/auth/`** — Move `internal/config/auth.go` → `pkg/auth/config.go` following the established pattern where packages own their config (`database.Config` in `pkg/database/`, `storage.Config` in `pkg/storage/`). The middleware receives `*auth.Config` directly and checks `cfg.Mode == ModeAzure` — no separate middleware AuthConfig type, no mapping step.

**`go-oidc` for token verification** — Use `github.com/coreos/go-oidc/v3` instead of hand-rolling JWKS discovery, key caching, and JWT validation. `go-oidc` handles OIDC discovery, JWKS fetching with automatic key rotation, and token signature/claims verification. It's maintained by the CoreOS team (Red Hat/IBM), used by Kubernetes for cluster OIDC auth, and is a CNCF ecosystem dependency via Dex. This eliminates ~150 lines of security-sensitive code (RSA key decoding, thread-safe cache, discovery logic) in favor of a battle-tested library.

This follows Herald's dependency criteria: purpose-built for a specific use, authoritative source from a reputable maintainer, actively maintained (v3.17.0, November 2025), and production-ready. Not a supply chain risk.

## Implementation

### Step 1: Create `pkg/auth/config.go`

New file. Move and adapt `internal/config/auth.go` with these changes: package rename, type renames (`AuthConfig` → `Config`, `AuthMode` → `Mode`), `Authority` field, `Env` injection pattern, `deriveDefaults()` phase.

```go
package auth

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

const AgentScope = "https://cognitiveservices.azure.com/.default"

type Mode string

const (
	ModeNone  Mode = "none"
	ModeAzure Mode = "azure"

	DefaultAuthorityBase = "https://login.microsoftonline.com/"
	DefaultAuthorityPath = "/v2.0"
)

type Config struct {
	Mode            Mode   `json:"auth_mode"`
	ManagedIdentity bool   `json:"managed_identity"`
	TenantID        string `json:"tenant_id"`
	ClientID        string `json:"client_id"`
	ClientSecret    string `json:"client_secret"`
	Authority       string `json:"authority"`
}

type Env struct {
	Mode            string
	ManagedIdentity string
	TenantID        string
	ClientID        string
	ClientSecret    string
	Authority       string
}

func (c *Config) Finalize(env *Env) error {
	c.loadDefaults()
	if env != nil {
		c.loadEnv(env)
	}
	c.deriveDefaults()
	return c.validate()
}

func (c *Config) Merge(overlay *Config) {
	if overlay.Mode != "" {
		c.Mode = overlay.Mode
	}
	if overlay.ManagedIdentity {
		c.ManagedIdentity = true
	}
	if overlay.TenantID != "" {
		c.TenantID = overlay.TenantID
	}
	if overlay.ClientID != "" {
		c.ClientID = overlay.ClientID
	}
	if overlay.ClientSecret != "" {
		c.ClientSecret = overlay.ClientSecret
	}
	if overlay.Authority != "" {
		c.Authority = overlay.Authority
	}
}

func (c *Config) TokenCredential() (azcore.TokenCredential, error) {
	switch c.Mode {
	case ModeNone:
		return nil, nil
	case ModeAzure:
		return c.azureCredential()
	default:
		return nil, fmt.Errorf("unsupported auth mode: %s", c.Mode)
	}
}

func (c *Config) azureCredential() (azcore.TokenCredential, error) {
	if c.TenantID != "" && c.ClientID != "" && c.ClientSecret != "" {
		cred, err := azidentity.NewClientSecretCredential(
			c.TenantID, c.ClientID, c.ClientSecret, nil,
		)
		if err != nil {
			return nil, fmt.Errorf("create client secret credential: %w", err)
		}
		return cred, nil
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("create default azure credential: %w", err)
	}
	return cred, nil
}

func (c *Config) loadDefaults() {
	if c.Mode == "" {
		c.Mode = ModeNone
	}
}

func (c *Config) loadEnv(env *Env) {
	if env.Mode != "" {
		if v := os.Getenv(env.Mode); v != "" {
			c.Mode = Mode(v)
		}
	}
	if env.ManagedIdentity != "" {
		if v := os.Getenv(env.ManagedIdentity); v != "" {
			if b, err := strconv.ParseBool(v); err == nil && b {
				c.ManagedIdentity = true
			}
		}
	}
	if env.TenantID != "" {
		if v := os.Getenv(env.TenantID); v != "" {
			c.TenantID = v
		}
	}
	if env.ClientID != "" {
		if v := os.Getenv(env.ClientID); v != "" {
			c.ClientID = v
		}
	}
	if env.ClientSecret != "" {
		if v := os.Getenv(env.ClientSecret); v != "" {
			c.ClientSecret = v
		}
	}
	if env.Authority != "" {
		if v := os.Getenv(env.Authority); v != "" {
			c.Authority = v
		}
	}
}

// deriveDefaults sets computed defaults that depend on values which may
// have been populated by environment variable overrides during loadEnv.
func (c *Config) deriveDefaults() {
	if c.Authority == "" && c.TenantID != "" {
		c.Authority = DefaultAuthorityBase + c.TenantID + DefaultAuthorityPath
	}
}

func (c *Config) validate() error {
	switch c.Mode {
	case ModeNone, ModeAzure:
		return nil
	default:
		return fmt.Errorf(
			"invalid auth_mode %q: must be %q or %q",
			c.Mode, ModeNone, ModeAzure,
		)
	}
}
```

### Step 2: Create `pkg/auth/errors.go`

New file.

```go
package auth

import "errors"

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrTokenExpired = errors.New("token expired")
	ErrInvalidToken = errors.New("invalid token")
)
```

### Step 3: Create `pkg/auth/user.go`

New file.

```go
package auth

import "context"

type contextKey struct{}

var userKey = contextKey{}

type User struct {
	ID    string
	Name  string
	Email string
}

func ContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

func UserFromContext(ctx context.Context) *User {
	user, _ := ctx.Value(userKey).(*User)
	return user
}
```

### Step 4: Create `pkg/middleware/auth.go`

New file. Uses `go-oidc` for OIDC discovery, JWKS management, and token verification. The provider is created lazily on the first request to avoid blocking server startup with a network call.

```go
package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/JaimeStill/herald/pkg/auth"
	"github.com/coreos/go-oidc/v3/oidc"
)

func Auth(cfg *auth.Config, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if cfg.Mode != auth.ModeAzure {
			return next
		}

		var (
			once     sync.Once
			verifier *oidc.IDTokenVerifier
			initErr  error
		)

		initVerifier := func() {
			provider, err := oidc.NewProvider(context.Background(), cfg.Authority)
			if err != nil {
				initErr = err
				return
			}

			verifier = provider.Verifier(&oidc.Config{
				ClientID: cfg.ClientID,
			})
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, ok := extractBearer(r)
			if !ok {
				respondUnauthorized(w, auth.ErrUnauthorized)
				return
			}

			once.Do(initVerifier)
			if initErr != nil {
				logger.Error("oidc provider init failed", "error", initErr)
				respondUnauthorized(w, auth.ErrInvalidToken)
				return
			}

			idToken, err := verifier.Verify(r.Context(), tokenString)
			if err != nil {
				logger.Debug("token verification failed", "error", err)
				respondUnauthorized(w, mapVerifyError(err))
				return
			}

			var claims struct {
				OID               string `json:"oid"`
				Name              string `json:"name"`
				PreferredUsername string `json:"preferred_username"`
				Email             string `json:"email"`
				UPN               string `json:"upn"`
			}

			if err := idToken.Claims(&claims); err != nil {
				logger.Error("claim extraction failed", "error", err)
				respondUnauthorized(w, auth.ErrInvalidToken)
				return
			}

			user := &auth.User{
				ID:    claims.OID,
				Name:  firstNonEmpty(claims.Name, claims.PreferredUsername),
				Email: firstNonEmpty(claims.Email, claims.UPN),
			}

			ctx := auth.ContextWithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearer(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return "", false
	}
	return strings.TrimPrefix(header, "Bearer "), true
}

func mapVerifyError(err error) error {
	if strings.Contains(err.Error(), "token is expired") {
		return auth.ErrTokenExpired
	}
	return auth.ErrInvalidToken
}

func respondUnauthorized(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
```

### Step 5: Update `internal/config/config.go`

Add `auth` import, add `authEnv` variable, change `Auth` field type, update finalize/merge calls.

**Add import:**

```go
"github.com/JaimeStill/herald/pkg/auth"
```

**Add `authEnv` variable** (alongside existing `databaseEnv` and `storageEnv`):

```go
var authEnv = &auth.Env{
	Mode:            "HERALD_AUTH_MODE",
	ManagedIdentity: "HERALD_AUTH_MANAGED_IDENTITY",
	TenantID:        "HERALD_AUTH_TENANT_ID",
	ClientID:        "HERALD_AUTH_CLIENT_ID",
	ClientSecret:    "HERALD_AUTH_CLIENT_SECRET",
	Authority:       "HERALD_AUTH_AUTHORITY",
}
```

**Change `Config.Auth` field:**

```go
Auth auth.Config `json:"auth"`
```

**Update `finalize()`:**

```go
if err := c.Auth.Finalize(authEnv); err != nil {
```

**Update `Merge()`:**

No signature change needed — `c.Auth.Merge(&overlay.Auth)` still works with the new type.

### Step 6: Update `internal/infrastructure/infrastructure.go`

**Add import:**

```go
"github.com/JaimeStill/herald/pkg/auth"
```

**Change `config.AgentScope` → `auth.AgentScope`:**

In the `newAgentFactory` function:

```go
Scopes: []string{auth.AgentScope},
```

### Step 7: Delete `internal/config/auth.go`

Remove the file entirely — all its contents have been moved to `pkg/auth/config.go`.

### Step 8: Update `tests/config/auth_test.go`

Replace all `config.AuthConfig` → `auth.Config`, `config.AuthModeNone` → `auth.ModeNone`, `config.AuthModeAzure` → `auth.ModeAzure`, `config.AuthMode` → `auth.Mode`.

**Add import:**

```go
"github.com/JaimeStill/herald/pkg/auth"
```

**Change `Finalize()` calls** to `Finalize(nil)` for standalone auth config tests (no env injection needed when testing directly).

**Update `TestAuthConfigEnvOverrides`** — this test uses `t.Setenv` with hardcoded env var names. Since the standalone `Config.Finalize(nil)` won't read env vars (env is nil), this test needs to pass an `Env` struct:

```go
func TestAuthConfigEnvOverrides(t *testing.T) {
	t.Setenv("HERALD_AUTH_MODE", "azure")
	t.Setenv("HERALD_AUTH_MANAGED_IDENTITY", "true")
	t.Setenv("HERALD_AUTH_TENANT_ID", "tenant-123")
	t.Setenv("HERALD_AUTH_CLIENT_ID", "client-456")
	t.Setenv("HERALD_AUTH_CLIENT_SECRET", "secret-789")

	env := &auth.Env{
		Mode:            "HERALD_AUTH_MODE",
		ManagedIdentity: "HERALD_AUTH_MANAGED_IDENTITY",
		TenantID:        "HERALD_AUTH_TENANT_ID",
		ClientID:        "HERALD_AUTH_CLIENT_ID",
		ClientSecret:    "HERALD_AUTH_CLIENT_SECRET",
	}

	cfg := &auth.Config{}
	if err := cfg.Finalize(env); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	// ... assertions unchanged ...
}
```

Same pattern for `TestAuthConfigManagedIdentityEnvValues`.

Tests that don't test env overrides (`TestAuthConfigDefaults`, `TestAuthConfigNoneCredential`, `TestAuthConfigValidation`, `TestAuthConfigMerge*`, `TestTokenCredentialUnsupportedMode`) pass `nil` to `Finalize`.

Tests that go through `config.Load()` (`TestAuthConfigFromLoad`, `TestAuthConfigFromLoadWithOverlay`, `TestAuthConfigInvalidModeFromLoad`) need their assertions updated to use `auth.ModeNone`/`auth.ModeAzure` but the `Finalize` call is handled internally by `config.Load()`.

### Step 9: Update `tests/infrastructure/infrastructure_test.go`

**Add import:**

```go
"github.com/JaimeStill/herald/pkg/auth"
```

**Change `validConfig()` Auth field:**

```go
Auth: auth.Config{Mode: auth.ModeNone},
```

Remove `"github.com/JaimeStill/herald/internal/config"` from imports if no longer used (check — it may still be used for `config.Config`).

### Step 10: Update `tests/api/api_test.go`

Same pattern as infrastructure test:

**Add import:**

```go
"github.com/JaimeStill/herald/pkg/auth"
```

**Change `validConfig()` Auth field:**

```go
Auth: auth.Config{Mode: auth.ModeNone},
```

### Step 11: Add `go-oidc` dependency

```bash
go get github.com/coreos/go-oidc/v3
go mod tidy
```

### Step 12: Document dependency criteria in `_project/README.md`

Add a **Dependency Criteria** section under **Dependencies** to codify the evaluation standards for external libraries:

```markdown
### Dependency Criteria

External dependencies are acceptable when they meet all of the following:

- **Purpose-built** — solves a specific, well-scoped problem (not a kitchen-sink framework)
- **Authoritative source** — maintained by a reputable organization or established in the ecosystem
- **Actively maintained** — recent releases, responsive to issues, not abandoned
- **Production-ready** — stable API, used in production by major projects
- **Not a supply chain risk** — minimal transitive dependencies, auditable codebase
```

## Validation Criteria

- [ ] `go vet ./...` passes
- [ ] `go build ./...` passes
- [ ] `go test ./tests/...` passes
- [ ] `pkg/auth/` exports Config, Mode constants, User, context helpers, error sentinels, AgentScope, TokenCredential
- [ ] `pkg/middleware/auth.go` compiles with correct function signature
- [ ] `Mode != ModeAzure` produces a pass-through middleware (returns `next` directly)
- [ ] `go-oidc` is a direct dependency in go.mod
- [ ] `internal/config/auth.go` is deleted
- [ ] Dependency criteria documented in `_project/README.md`
