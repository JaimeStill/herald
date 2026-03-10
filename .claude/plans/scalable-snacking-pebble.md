# Objective Planning: #97 — Managed Identity for Azure Services

## Context

Objective #96 (Azure Identity Credential Infrastructure) is complete — it added `AuthConfig`, `TokenCredential()` factory, and `Infrastructure.Credential` field. Objective #97 wires that credential into all three Azure service clients (Storage, Database, AI Foundry) as an opt-in alternative to connection strings and API keys.

### Three-Stage Auth Model

`auth_mode` and `managed_identity` are independent concerns:

| Stage | `auth_mode` | `managed_identity` | User Auth | Service Connections |
|-------|-------------|---------------------|-----------|---------------------|
| 1 | `none` | `false` (ignored) | None | Connection strings from env vars |
| 2 | `azure` | `false` | Entra JWT | Connection strings via Key Vault secrets |
| 3 | `azure` | `true` | Entra JWT | Managed identity tokens |

- `auth_mode` controls user authentication (HTTP middleware — Objective #98)
- `managed_identity` controls how Herald authenticates to Azure services (this objective)
- When `auth_mode: "none"`, `managed_identity` is ignored (no credential exists)
- `TokenCredential()` still creates a credential whenever `auth_mode: "azure"` — the `managed_identity` flag only controls whether it's passed to service client constructors

## Transition Closeout

**Previous Objective:** #96 — Azure Identity Credential Infrastructure
- Sub-issue #103: Closed (merged via PR #104)
- 1/1 complete (100%)
- **Action:** Close #96, delete `_project/objective.md`, update `_project/phase.md` status to Complete

## Sub-Issue Decomposition

### Dependency Graph

```
Sub-issue 1 (Storage)  ──┐
Sub-issue 2 (Database) ──┼──▶ Sub-issue 4 (AuthConfig + Infrastructure Wiring)
Sub-issue 3 (Agent)    ──┘
```

Sub-issues 1, 2, 3 are independent — can be developed in parallel. Sub-issue 4 depends on all three.

---

### Sub-Issue 1: Credential-based storage constructor

**Scope:** Add `NewWithCredential` constructor to `pkg/storage/` that accepts `azcore.TokenCredential` + service URL instead of connection string.

**Approach:**
- Add `ServiceURL` field to `storage.Config` + `HERALD_STORAGE_SERVICE_URL` env var
- Relax `validate()` — remove `ConnectionString` requirement (let constructors validate their own needs)
- Add `NewWithCredential(cfg *Config, cred azcore.TokenCredential, logger *slog.Logger) (System, error)` that calls `azblob.NewClient(cfg.ServiceURL, cred, nil)`
- Existing `New()` unchanged — validates ConnectionString internally

**Files:**
- `pkg/storage/config.go` — add ServiceURL, env var, relax validate
- `pkg/storage/storage.go` — add NewWithCredential constructor
- `internal/config/config.go` — add ServiceURL to storageEnv

**Labels:** `infrastructure`

---

### Sub-Issue 2: Token-based database authentication

**Scope:** Add `NewWithCredential` constructor to `pkg/database/` with pgx `BeforeConnect` hook for Entra token injection and forced 45-minute `ConnMaxLifetime`.

**Approach:**
- Add `NewWithCredential(cfg *Config, cred azcore.TokenCredential, logger *slog.Logger) (System, error)`
- Use `pgx.ParseConfig()` to build `pgx.ConnConfig` from DSN (without password)
- Set `BeforeConnect` hook that calls `cred.GetToken()` with scope `https://ossrdbms-aad.database.windows.net/.default` and sets `cc.Password = token.Token`
- Use `stdlib.OpenDB(*connConfig)` to get `*sql.DB` (preserves existing interface)
- Force `ConnMaxLifetime` to 45 minutes regardless of config — tokens expire ~1 hour
- Relax `validate()` — remove implicit password requirement (password not needed in credential mode)

**Files:**
- `pkg/database/database.go` — add NewWithCredential constructor
- `pkg/database/config.go` — add DsnWithoutPassword helper, relax validate if needed

**Labels:** `infrastructure`

---

### Sub-Issue 3: Agent token provider for AI Foundry bearer auth

**Scope:** Add `TokenProvider` to `workflow.Runtime` and `rt.NewAgent(ctx)` method so each agent creation gets a fresh Entra bearer token.

**Approach:**
- Add `TokenProvider func(ctx context.Context) (string, error)` field to `workflow.Runtime`
- Add `rt.NewAgent(ctx context.Context) (*agent.Agent, error)` method on Runtime that encapsulates the clone+inject+create pattern:
  - If `TokenProvider` is nil: `agent.New(&rt.Agent)` (existing path)
  - If `TokenProvider` is non-nil: clone `rt.Agent`, call provider, set `Options["token"]` and `Options["auth_type"] = "bearer"`, then `agent.New()`
- Replace all `agent.New(&rt.Agent)` calls in classify.go, enhance.go, finalize.go with `rt.NewAgent(ctx)`
- Token provider is a closure created by infrastructure: `cred.GetToken(ctx, {Scopes: ["https://cognitiveservices.azure.com/.default"]})` → returns `tok.Token`
- Thread TokenProvider from `Infrastructure` through `api.Runtime` → `NewDomain` → `classifications.New()` → `workflow.Runtime`

**Files:**
- `internal/workflow/runtime.go` — add TokenProvider field + NewAgent method
- `internal/workflow/classify.go` — replace agent.New with rt.NewAgent
- `internal/workflow/enhance.go` — same
- `internal/workflow/finalize.go` — same
- `internal/classifications/repository.go` — accept and store TokenProvider, pass to Runtime
- `internal/api/domain.go` — pass TokenProvider to classifications.New
- `internal/api/runtime.go` — expose TokenProvider from Infrastructure
- `internal/infrastructure/infrastructure.go` — add TokenProvider field

**Labels:** `infrastructure`

---

### Sub-Issue 4: AuthConfig managed_identity flag and infrastructure wiring

**Scope:** Add `ManagedIdentity` field to `AuthConfig`, wire credential-based constructors into `infrastructure.New()` based on the flag.

**Approach:**
- Add `ManagedIdentity bool` field to `AuthConfig` (default: `false`) + `HERALD_AUTH_MANAGED_IDENTITY` env var
- When `auth_mode: "none"`, `ManagedIdentity` is ignored (no credential exists)
- When `auth_mode: "azure"` + `ManagedIdentity: true`:
  - Call `storage.NewWithCredential()` instead of `storage.New()`
  - Call `database.NewWithCredential()` instead of `database.New()`
  - Build token provider closure for agent: `func(ctx) (string, error) { tok, err := cred.GetToken(ctx, {Scopes: ["https://cognitiveservices.azure.com/.default"]}); return tok.Token, err }`
- When `auth_mode: "azure"` + `ManagedIdentity: false`: existing `New()` constructors, no token provider (connection strings via env vars)
- No IL4/IL6 config overlay files — deployed instances configure via `HERALD_*` env vars on Azure Container Apps

**Files:**
- `internal/config/auth.go` — add ManagedIdentity field, env var, Merge, loadEnv, loadDefaults
- `internal/infrastructure/infrastructure.go` — branching logic based on `cfg.Auth.ManagedIdentity`

**Labels:** `infrastructure`

---

## Scope Changes from Original Issue

- IL4/IL6 config overlay files dropped — deployed instances configure via `HERALD_*` env vars on Azure Container Apps (env vars, Key Vault references)
- `auth_mode` semantics clarified — controls user auth only, not service identity
- New `managed_identity` bool field on `AuthConfig` — controls service client constructor selection

## GitHub Operations

After plan approval:
1. Update #97 issue body — remove config overlay bullet, clarify managed_identity as opt-in alongside connection strings
2. Close #96 (all sub-issues complete)
3. Create 4 sub-issues on JaimeStill/herald with `infrastructure` label and milestone `v0.4.0 - Security and Deployment`
4. Link each as sub-issue of #97
5. Update `_project/phase.md` — mark #96 as Complete, #97 as Active
6. Create `_project/objective.md` for #97
