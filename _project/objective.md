# Objective: Deployment Configuration

**Issue:** [#100](https://github.com/JaimeStill/herald/issues/100)
**Phase:** Phase 4 — Security and Deployment (v0.4.0)

## Scope

Create Azure Container Apps deployment manifests and environment-specific configuration for IL4/IL6 environments. This is the capstone objective that ties together the production Docker image, managed identity, and authentication work.

### What's Covered

- Make `AgentScope` configurable for Azure Government cloud endpoints
- Azure Container Apps deployment manifests (Bicep) in `deploy/` directory
- Managed identity role assignments (Storage Blob Data Contributor, Cognitive Services OpenAI User)
- Container Apps Job for database migrations (idempotent `migrate -up`)
- Dockerfile update to build both server and migrate binaries
- Deployment documentation in `_project/deployment/`

### What's Not Covered

- AKS manifests — Container Apps selected as the deployment target
- Application code changes (beyond AgentScope configurability)
- Config overlay files — Container Apps config is entirely env-var driven via Bicep parameters

## Sub-Issues

| # | Title | Issue | Status | Dependencies |
|---|-------|-------|--------|--------------|
| 1 | Make AgentScope configurable for Azure Government | [#124](https://github.com/JaimeStill/herald/issues/124) | Open | None |
| 2 | Add Bicep deployment manifests for Container Apps infrastructure | [#125](https://github.com/JaimeStill/herald/issues/125) | Open | #124 |
| 3 | Add deployment documentation | [#126](https://github.com/JaimeStill/herald/issues/126) | Open | #124, #125 |

Sub-issues are strictly sequential: 1 → 2 → 3.

## Architecture Decisions

- **Bicep over bash scripts** — Declarative IaC with compile-time validation, what-if preview, and automatic rollback. Compiles to ARM templates (IL6 compatible). Existing `scripts/` directory preserved as alternative.
- **No config overlay files** — Container Apps configuration is entirely env-var driven via Bicep parameters. Overlay files would be redundant documentation artifacts with maintenance burden.
- **Container Apps Job for migrations** — Independent execution with separate logging, manual trigger capability, and distinct resource allocation. `golang-migrate -up` is idempotent, safe on every deployment. Init containers share app resources and can't be triggered independently.
- **Single container image** — Both `herald` and `herald-migrate` binaries in one image. Migration Job overrides the command. Ensures migration code always matches server version.
- **No CD workflow** — `release.yml` already handles GHCR builds on tag push. Deployment is environment-specific — operators deploy via `az deployment group create`.
- **AgentScope configurable** — The only application code change. Azure Gov cognitive services uses `.azure.us` domain. Database `TokenScope` is universal across clouds.
