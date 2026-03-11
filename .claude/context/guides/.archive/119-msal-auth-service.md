# 119 — Add MSAL Auth Service with Login Gate

## Problem Context

Sub-issue 2 of Objective #99 (Web Client MSAL.js Integration). With auth config injected into the HTML template (#118), the client needs an `Auth` service wrapping `@azure/msal-browser` that handles MSAL initialization, login redirect, and token acquisition. The app bootstrap must gate on authentication before starting the router when auth is enabled.

## Architecture Approach

The Auth service is framework infrastructure in `core/`, not a domain service. It follows the PascalCase singleton pattern with module-scoped `let` state for the MSAL instance. When the `<script id="herald-config">` element is absent (auth disabled), all methods are safe no-ops. The cache location is configurable via the server-side auth config, flowing from `pkg/auth/config.go` through `ClientAuthConfig` to the client.

## Implementation

### Step 1: Add `CacheLocation` to `pkg/auth/config.go`

Add the `CacheLocation` type, constants, field on `Config` and `Env`, and wire into `loadDefaults`, `loadEnv`, `Merge`, and `validate`:

**Add `CacheLocation` type and constants** (alongside existing `Mode` type):

```go
type CacheLocation string

const (
	LocalStorage  CacheLocation = "localStorage"
	SessionStorage CacheLocation = "sessionStorage"
)
```

**In the `Config` struct** (add after `Authority`):

```go
type Config struct {
	Mode            Mode          `json:"auth_mode"`
	ManagedIdentity bool          `json:"managed_identity"`
	TenantID        string        `json:"tenant_id"`
	ClientID        string        `json:"client_id"`
	ClientSecret    string        `json:"client_secret"`
	Authority       string        `json:"authority"`
	CacheLocation   CacheLocation `json:"cache_location"`
}
```

**In the `Env` struct** (add after `Authority`):

```go
type Env struct {
	Mode            string
	ManagedIdentity string
	TenantID        string
	ClientID        string
	ClientSecret    string
	Authority       string
	CacheLocation   string
}
```

**In `loadDefaults()`** (add after mode default):

```go
func (c *Config) loadDefaults() {
	if c.Mode == "" {
		c.Mode = ModeNone
	}
	if c.CacheLocation == "" {
		c.CacheLocation = LocalStorage
	}
}
```

**In `loadEnv()`** (add after Authority block):

```go
	if env.CacheLocation != "" {
		if v := os.Getenv(env.CacheLocation); v != "" {
			c.CacheLocation = CacheLocation(v)
		}
	}
```

**In `Merge()`** (add after Authority block):

```go
	if overlay.CacheLocation != "" {
		c.CacheLocation = overlay.CacheLocation
	}
```

**Replace `validate()`** with both checks:

```go
func (c *Config) validate() error {
	switch c.Mode {
	case ModeNone, ModeAzure:
	default:
		return fmt.Errorf(
			"invalid auth_mode %q: must be %q or %q",
			c.Mode, ModeNone, ModeAzure,
		)
	}
	switch c.CacheLocation {
	case LocalStorage, SessionStorage:
	default:
		return fmt.Errorf(
			"invalid cache_location %q: must be %q or %q",
			c.CacheLocation, LocalStorage, SessionStorage,
		)
	}

	return nil
}
```

### Step 2: Add env var mapping in `internal/config/config.go`

Add `CacheLocation` to the `authEnv` var:

```go
var authEnv = &auth.Env{
	Mode:            "HERALD_AUTH_MODE",
	ManagedIdentity: "HERALD_AUTH_MANAGED_IDENTITY",
	TenantID:        "HERALD_AUTH_TENANT_ID",
	ClientID:        "HERALD_AUTH_CLIENT_ID",
	ClientSecret:    "HERALD_AUTH_CLIENT_SECRET",
	Authority:       "HERALD_AUTH_AUTHORITY",
	CacheLocation:   "HERALD_AUTH_CACHE_LOCATION",
}
```

### Step 3: Add `CacheLocation` to `ClientAuthConfig` in `app/app.go`

```go
type ClientAuthConfig struct {
	TenantID      string `json:"tenant_id"`
	ClientID      string `json:"client_id"`
	RedirectURI   string `json:"redirect_uri"`
	Authority     string `json:"authority"`
	CacheLocation string `json:"cache_location"`
}
```

### Step 4: Pass `CacheLocation` through in `cmd/server/modules.go`

Update the `ClientAuthConfig` construction:

```go
	authCfg = &app.ClientAuthConfig{
		TenantID:      cfg.Auth.TenantID,
		ClientID:      cfg.Auth.ClientID,
		Authority:     cfg.Auth.Authority,
		CacheLocation: cfg.Auth.CacheLocation,
	}
```

### Step 5: Add `@azure/msal-browser` dependency

In `app/package.json`, add to dependencies:

```json
"dependencies": {
  "@azure/msal-browser": "^5.4.0",
  "lit": "^3.3.2"
}
```

Run `bun install` from the `app/` directory.

### Step 6: Create `app/client/core/auth.ts`

New file — the Auth service:

```typescript
import {
  type AccountInfo,
  type AuthenticationResult,
  type Configuration,
  InteractionRequiredAuthError,
  PublicClientApplication,
} from "@azure/msal-browser";

interface AuthConfig {
  tenant_id: string;
  client_id: string;
  redirect_uri: string;
  authority: string;
  cache_location?: string;
}

let msalInstance: PublicClientApplication | null = null;
let config: AuthConfig | null = null;

function readConfig(): AuthConfig | null {
  const el = document.getElementById("herald-config");
  if (!el?.textContent) return null;

  return JSON.parse(el.textContent) as AuthConfig;
}

function scope(): string {
  return `api://${config!.client_id}/access_as_user`;
}

export const Auth = {
  isEnabled(): boolean {
    return config !== null;
  },

  isAuthenticated(): boolean {
    if (!msalInstance) return false;
    return msalInstance.getActiveAccount() !== null;
  },

  getAccount(): AccountInfo | null {
    return msalInstance?.getActiveAccount() ?? null;
  },

  async init(): Promise<void> {
    config = readConfig();
    if (!config) return;

    const msalConfig: Configuration = {
      auth: {
        clientId: config.client_id,
        authority: config.authority,
        redirectUri: config.redirect_uri,
      },
      cache: {
        cacheLocation: config.cache_location ?? "localStorage",
      },
    };

    msalInstance = new PublicClientApplication(msalConfig);
    await msalInstance.initialize();

    const response: AuthenticationResult | null =
      await msalInstance.handleRedirectPromise();

    if (response?.account) {
      msalInstance.setActiveAccount(response.account);
    } else {
      const accounts = msalInstance.getAllAccounts();
      if (accounts.length === 1) {
        msalInstance.setActiveAccount(accounts[0]);
      }
    }
  },

  async getToken(forceRefresh?: boolean): Promise<string | null> {
    const account = msalInstance?.getActiveAccount();
    if (!msalInstance || !account) return null;

    try {
      const result = await msalInstance.acquireTokenSilent({
        scopes: [scope()],
        account,
        forceRefresh: forceRefresh ?? false,
      });
      return result.accessToken;
    } catch (e) {
      if (e instanceof InteractionRequiredAuthError) {
        await this.login();
      }
      return null;
    }
  },

  async login(): Promise<void> {
    if (!msalInstance) return;
    await msalInstance.loginRedirect({ scopes: [scope()] });
  },

  async logout(): Promise<void> {
    if (!msalInstance) return;
    await msalInstance.logoutRedirect();
  },
};
```

### Step 7: Re-export Auth from `app/client/core/index.ts`

Add `Auth` export at the top of the barrel:

```typescript
export { Auth } from "./auth";

export { request, stream, toQueryString } from "./api";

export type {
  ExecutionEvent,
  PageRequest,
  PageResult,
  Result,
  StreamOptions,
} from "./api";
```

### Step 8: Convert `app/client/app.ts` to async bootstrap

Replace the entire file:

```typescript
import { Auth, Router } from "@core";
import "@ui/elements";
import "@ui/modules";
import "@ui/views";

import { routes } from "./routes";

import "@design/index.css";

(async () => {
  await Auth.init();

  if (Auth.isEnabled() && !Auth.isAuthenticated()) {
    await Auth.login();
    return;
  }

  const router = new Router("app-content", routes);
  router.start();
})();
```

## Validation Criteria

- [ ] `go vet ./...` passes
- [ ] `bun install` completes in `app/` with `@azure/msal-browser` resolved
- [ ] `bun run build` produces `dist/app.js` and `dist/app.css` without errors
- [ ] With auth disabled (default): app loads normally, router starts, existing functionality works
- [ ] `Auth.isEnabled()` returns `false` when no config script is present
- [ ] All Auth methods are safe no-ops when auth is disabled
