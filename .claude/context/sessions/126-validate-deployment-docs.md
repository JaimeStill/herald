# 126 — Validate Deployment Infrastructure and Finalize Documentation

Final sub-issue of Objective 6 (Deployment Configuration). Live commercial Azure deployment validation with iterative infrastructure fixes, deployment guide overhaul, and CDS proxy repo initialization.

## What Was Done

### go-agents Upgrade (v0.4.0)
- Upgraded go-agents to v0.4.0 for native Azure managed identity support
- Removed manual token acquisition factory from `internal/infrastructure/infrastructure.go`
- Added `HERALD_AGENT_RESOURCE` and `HERALD_AGENT_CLIENT_ID` env var mappings in `internal/config/agent.go`
- Removed `AgentScope` from `pkg/auth` (field, defaults, env loading, merge, tests)

### Bicep Infrastructure Fixes
- Fixed `HERALD_DB_USER` — must be Entra admin principal name (`${prefix}-identity`), not managed identity client ID
- Fixed `HERALD_AGENT_BASE_URL` — appended `/openai` for OpenAI-kind Cognitive Services accounts
- Added `cognitiveDeploymentCapacity` parameter (default 1M TPM) for configurable rate limits
- Made `cognitiveCustomDomain` a required parameter (no default)
- Added `deploy/main.secrets.json` (gitignored) for `postgresAdminPassword` and `ghcrUsername`
- Added `authEnabled`, `tenantId`, `entraClientId` to `main.parameters.json`

### Observer Error Logging
- Added `*slog.Logger` to `StreamingObserver` for centralized workflow error logging
- `SendError` and `handleNodeComplete` now log errors through the observer rather than requiring duplicate logging at call sites
- Updated all test calls to pass logger

### Deployment Guide
- Moved from `_project/deployment.md` to `deploy/README.md`
- Restructured: grouped parameter tables, Configuration section, Operations section, full Environment Variables Reference table
- Added troubleshooting runbook based on live validation: PostgreSQL auth, Agent Vision 404, CustomDomainInUse, regional quota, dirty migration state, rollback
- Removed all hardcoded resource names in favor of `<prefix>-*` placeholders
- Added secrets file template and GHCR container packages link

### Housekeeping
- Added `deploy/main.json` and `deploy/main.secrets.json` to `.gitignore`
- Removed `notes.md` from tracking
- Cleaned up all accumulated plan files
- Documented Azure base URL `/openai` requirement in go-agents README

## Issues Discovered During Live Validation

1. **PostgreSQL auth**: `HERALD_DB_USER` must be principal name, not client ID UUID
2. **Agent 404**: OpenAI-kind accounts require `/openai` in base URL path
3. **Rate limiting**: Default 10K TPM too low for concurrent classifications — increased to 1M
4. **Entra scope**: App registration scope name must match config default (`access_as_user`)
5. **SSE proxy buffering**: Events visible via curl but not in browser through Container Apps — unresolved, likely platform proxy buffering

## Key Decisions

- Agent token scope handled by go-agents natively (not Herald auth config)
- Observer owns error logging (not call sites) — single responsibility
- Deployment guide lives with infrastructure (`deploy/README.md`) not project docs
- Static secrets in gitignored parameter file, dynamic secrets via CLI
- Herald repo has no knowledge of the CDS proxy infrastructure
