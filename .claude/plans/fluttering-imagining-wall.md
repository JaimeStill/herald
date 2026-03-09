# Phase 4 Planning: Security and Deployment (v0.4.0)

## Context

Phases 1–3 are complete. Herald has a working Go web service with embedded Lit SPA, classification workflow, and all domain systems. However, the service has **no authentication** (all API endpoints are open), uses **connection strings and API keys** for Azure services, and the **Docker image lacks ImageMagick** (required for PDF rendering). Phase 4 closes these gaps to make Herald deployable to IL4/IL6 Azure Government environments.

## Infrastructure State

- **Milestone**: `v0.4.0 - Security and Deployment` (#4) — already exists
- **Project board**: Phase 4 option (`fe8cd544`) — already exists
- **No open issues** — clean slate

## Proposed Objectives (6)

### Objective 1: Docker Production Image

Harden the Dockerfile for production: add ImageMagick 7.0+ and Ghostscript to the alpine runtime stage, add non-root user, health check, `.dockerignore`, and a production compose overlay for local integration testing.

**Dependencies:** None

### Objective 2: Azure Identity Credential Infrastructure

Add `azidentity` SDK and build the credential provider foundation. New `AuthConfig` in `internal/config/` with `auth_mode` (connection_string | managed_identity), tenant/client IDs, env var overrides. Wire credential creation into `infrastructure.New()` as an optional field. Backward compatible — `connection_string` mode preserves current behavior.

**Dependencies:** None (can parallel with Objective 1)

### Objective 3: Managed Identity for Azure Services

Wire managed identity credentials into all three Azure service clients:
- **Storage**: Add credential-based constructor alongside connection string
- **Database**: Token-based pgx auth via `BeforeConnect` hook + 45-min `ConnMaxLifetime` safety net
- **AI Foundry**: Token provider that supplies fresh bearer tokens at agent creation time

Add `config.il4.json` and `config.il6.json` environment overlays.

**Dependencies:** Objective 2

### Objective 4: API Authentication Middleware (Entra ID)

JWT bearer token validation middleware in `pkg/middleware/auth.go`. Validates against Entra JWKS, verifies audience/issuer/expiry. Extracts user identity into request context (`pkg/auth/` context helpers). Updates `validated_by` in classifications from authenticated user. Auth disabled by default for local dev.

**Dependencies:** Objective 2

### Objective 5: Web Client MSAL.js Integration

Add `@azure/msal-browser` for browser-side auth. MSAL config injected from Go server via template data. Token acquisition in `core/api.ts` for all requests. Login gate when auth enabled. User display in app header. Conditionally skipped when `auth.enabled: false`.

**Dependencies:** Objective 4

### Objective 6: Deployment Configuration

Azure Container Apps deployment manifests (Bicep), managed identity role assignments, production environment overlays, GitHub Actions workflow for container builds, database migration strategy (init container).

**Dependencies:** Objectives 1, 2, 3, 4

## Dependency Graph

```
Obj 1: Docker Image ─────────────────────────┐
                                              │
Obj 2: Identity Infrastructure ──┬────────────┤
                                 │            │
                    ┌────────────┤            │
                    │            │            │
Obj 3: Managed ID   Obj 4: Auth Middleware    │
for Services             │                   │
        │                 │                   │
        │          Obj 5: Web Client MSAL     │
        │                 │                   │
        └────────┬────────┘                   │
                 │                            │
          Obj 6: Deployment Config ───────────┘
```

Objectives 1 & 2 parallel. Objectives 3 & 4 parallel (after 2). Objective 5 after 4. Objective 6 is terminal.

## Key Architectural Decisions

### 1. No OBO — Managed identity for service-to-service

Herald's server calls Azure services (Storage, PostgreSQL, AI Foundry) using a shared managed identity, not user-delegated tokens. User identity is captured from JWT claims and stored in `validated_by` for audit. OBO would only be needed if downstream services required per-user authorization, which they don't — Herald owns all its data.

### 2. Container Apps (not AKS)

Single-service deployment. Container Apps supports managed identity, auto-scaling, and requires no cluster management. AKS is overkill.

### 3. Database token refresh via BeforeConnect + ConnMaxLifetime

pgx `BeforeConnect` hook acquires a fresh Entra token for each new connection. `ConnMaxLifetime` set to 45 minutes (tokens expire ~1 hour) ensures pooled connections don't outlive their tokens.

### 4. Auth-disabled local development

`auth.enabled: false` (default) preserves the current zero-friction dev experience. No MSAL.js init, no token injection, no login gates. The web client checks server-injected config at runtime.

## Verification

After all objectives are complete:
1. `docker compose up` runs Herald with ImageMagick, classifies a PDF successfully
2. With `auth.enabled: true`, unauthenticated API requests return 401
3. Web client login flow acquires token, all API calls succeed with Bearer header
4. With `auth_mode: managed_identity`, services connect via Entra credentials (testable with Azure CLI credential fallback)
5. Deployment manifests validate (`az deployment group validate`)
