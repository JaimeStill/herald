# Phase 4 — Security and Deployment

**Version Target:** v0.4.0

## Scope

Make Herald deployable to IL4/IL6 Azure Government environments. Add Azure Entra authentication for the HTTP API and web client, managed identity for Azure service connections, production-ready Docker containerization, and Azure Container Apps deployment configuration. All security features are opt-in via configuration — local development remains zero-friction with auth disabled by default.

## Goals

- Authenticate HTTP API requests via Azure Entra ID JWT bearer tokens
- Authenticate web client users via MSAL.js with Entra login flow
- Connect Azure services (Storage, PostgreSQL, AI Foundry) via managed identity as an alternative to connection strings and API keys
- Produce a production Docker image with ImageMagick 7.0+ and security hardening
- Provide Azure Container Apps deployment manifests (Bicep) with managed identity role assignments
- Support IL4/IL6 environment configuration overlays

## Key Decisions

- **No OBO** — managed identity for service-to-service calls, user identity from JWT claims stored in `validated_by` for audit trail
- **Container Apps** over AKS — single-service deployment, no cluster management needed
- **Auth opt-in** — `auth.enabled: false` (default) preserves current dev experience; `auth_mode: connection_string` (default) preserves current service connections
- **Database token refresh** — pgx `BeforeConnect` hook + 45-minute `ConnMaxLifetime` safety net for Entra token expiry

## Objectives

| # | Objective | Issue | Status | Dependencies |
|---|-----------|-------|--------|--------------|
| 1 | Docker Production Image | [#95](https://github.com/JaimeStill/herald/issues/95) | Complete | None |
| 2 | Azure Identity Credential Infrastructure | [#96](https://github.com/JaimeStill/herald/issues/96) | Complete | None |
| 3 | Managed Identity for Azure Services | [#97](https://github.com/JaimeStill/herald/issues/97) | Complete | #96 |
| 4 | API Authentication Middleware | [#98](https://github.com/JaimeStill/herald/issues/98) | Active | #96 |
| 5 | Web Client MSAL.js Integration | [#99](https://github.com/JaimeStill/herald/issues/99) | Open | #98 |
| 6 | Deployment Configuration | [#100](https://github.com/JaimeStill/herald/issues/100) | Open | #95, #96, #97, #98 |

## Dependency Graph

```
Obj 1: Docker Image ──────────────────────────┐
                                              │
Obj 2: Identity Infrastructure ──┬────────────┤
                                 │            │
                    ┌────────────┤            │
                    │            │            │
Obj 3: Managed ID   Obj 4: Auth Middleware    │
for Services             │                    │
        │                │                    │
        │          Obj 5: Web Client MSAL     │
        │                │                    │
        └────────┬───────┘                    │
                 │                            │
          Obj 6: Deployment Config ───────────┘
```

Objectives 1 & 2 parallel. Objectives 3 & 4 parallel (after 2). Objective 5 after 4. Objective 6 is terminal.

## Constraints

- All auth features must be fully opt-in — `auth.enabled: false` and `auth_mode: connection_string` are defaults
- No breaking changes to local development workflow (Azurite + PostgreSQL containers, no IdP required)
- ImageMagick 7.0+ required in the production container for PDF rendering
- Azure Government cloud endpoints differ from commercial Azure — environment overlays must account for this
