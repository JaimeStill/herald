# 125 - Add Bicep Deployment Manifests for Container Apps Infrastructure

## Summary

Created a complete modular Bicep infrastructure-as-code deployment for Herald targeting Azure Container Apps. Ten modules cover the full Azure stack: managed identity, Log Analytics, PostgreSQL Flexible Server, Blob Storage, Cognitive Services (AI Foundry), optional ACR, Container App Environment, Container App, migration job, and role assignments. Updated the Dockerfile to build both `herald` and `migrate` binaries. Added a deployment guide at `_project/deployment.md`.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| User-assigned vs system-assigned identity | User-assigned | Breaks circular dependency — identity exists before the app, so roles can be assigned first |
| Registry strategy | `useAcr` boolean parameter | Commercial uses GHCR with PAT; IL6 uses ACR with managed identity AcrPull — single codebase, two paths |
| PostgreSQL auth | Dual (password + Entra) | Migration job needs DSN (password); Container App uses Entra tokens at runtime |
| Log Analytics shared key | `listKeys()` inside environment.bicep | Avoids exposing secrets as module outputs (BCP307 / outputs-should-not-contain-secrets) |
| Conditional module outputs | Safe-dereference (`?.` + `??`) | Resolves BCP318 warnings for nullable `registry` module outputs |
| DB token scope | Parameterized with `#disable-next-line` | Scope is same across clouds but linter flags `database.windows.net`; parameterized for flexibility |
| Parameter file format | JSON (not .bicepparam) | Universal ARM/Bicep support, guaranteed IL6 compatibility |
| ARM contentVersion | Four-part aligned with semver (`0.4.0.0`) | ARM schema requires four parts; fourth position tracks deployment iterations |
| Entra app registration | Manual/CLI (not Bicep) | Microsoft.Graph extension is preview-only, unavailable on IL6 |
| Deployment docs | Separate `_project/deployment.md` | README keeps local dev Entra setup; deployment guide covers production concerns |
| Migration binary name | `/usr/local/bin/migrate` | Shorter, no ambiguity inside the container |

## Files Modified

- `Dockerfile` — added `migrate` binary build and copy
- `.claude/CLAUDE.md` — added Versioning section (ARM four-part version convention)

## Files Created

- `deploy/main.bicep` — orchestrator
- `deploy/main.parameters.json` — non-secret parameters
- `deploy/modules/identity.bicep` — user-assigned managed identity
- `deploy/modules/logging.bicep` — Log Analytics workspace
- `deploy/modules/postgres.bicep` — PostgreSQL Flexible Server + database + Entra admin + firewall
- `deploy/modules/storage.bicep` — Storage account + documents container
- `deploy/modules/cognitive.bicep` — Cognitive Services (OpenAI) + model deployment
- `deploy/modules/registry.bicep` — ACR (conditional)
- `deploy/modules/environment.bicep` — Container App Environment
- `deploy/modules/app.bicep` — Container App
- `deploy/modules/migration-job.bicep` — Container Apps Job
- `deploy/modules/roles.bicep` — role assignments
- `_project/deployment.md` — deployment guide

## Patterns Established

- Modular Bicep with `main.bicep` orchestrating self-contained modules via parameters/outputs
- `useAcr` boolean for registry mode switching (commercial vs IL6)
- Safe-dereference pattern for conditional module outputs
- `deploy/` directory is fully self-contained for IL6 CDS bundling
- ARM parameter `contentVersion` tracks deployment iterations via fourth position

## Validation Results

- `az bicep build -f deploy/main.bicep` compiles with zero errors and zero warnings
- All parameters documented with `@description` decorators
- No secrets in parameter files
- Dockerfile builds both binaries
- Existing `scripts/` directory untouched
