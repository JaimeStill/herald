# Objective Planning: #100 — Deployment Configuration

## Context

Objective #100 is the capstone of Phase 4 (Security and Deployment, v0.4.0). Objectives 1-5 delivered the Docker image, identity infrastructure, managed identity for Azure services, auth middleware, and MSAL.js web client. This objective ties everything together with Azure Container Apps deployment configuration and documentation for IL4/IL6 Azure Government environments.

## Transition Closeout

**Previous objective:** #99 — Web Client MSAL.js Integration
- Status: **3/3 sub-issues complete (100%)**
- All sub-issues (#118, #119, #120) are closed
- Clean transition — no carry-forward or backlog disposition needed

**Actions:**
1. Close issue #99
2. Update `_project/phase.md` — mark Objective 5 as Complete, Objective 6 as Active
3. Delete `_project/objective.md` (recreated in Step 5)

## Sub-Issues

### 1. Make AgentScope configurable for Azure Government

**Labels:** `infrastructure`
**Milestone:** v0.4.0 - Security and Deployment

**Context:** `AgentScope` is hardcoded as `https://cognitiveservices.azure.com/.default` in `pkg/auth/config.go`. Azure Government uses `https://cognitiveservices.azure.us/.default`. This is the only application code change needed for Gov cloud support — all other config flows through `HERALD_*` env vars set by Bicep at deployment time. No config overlay files (config.il4.json, config.il6.json) are needed since Container Apps configuration is entirely env-var driven.

**Scope:**
- Convert `AgentScope` constant to a configurable field on `auth.Config` with commercial Azure default
- Add `HERALD_AUTH_AGENT_SCOPE` env var mapping in `internal/config/config.go`
- Update `internal/infrastructure/infrastructure.go` to read scope from `cfg.Auth.AgentScope` instead of the `auth.AgentScope` constant

**Key files:**
- `pkg/auth/config.go` — constant → configurable field
- `internal/config/config.go` — env var mapping
- `internal/infrastructure/infrastructure.go:133` — uses `auth.AgentScope`

**Acceptance criteria:**
- [ ] `AgentScope` is a configurable field with `https://cognitiveservices.azure.com/.default` as default
- [ ] `HERALD_AUTH_AGENT_SCOPE` env var overrides the default
- [ ] `internal/infrastructure/infrastructure.go` reads from config, not constant
- [ ] Existing behavior unchanged when env var is not set

**Dependencies:** None

---

### 2. Add Bicep deployment manifests for Container Apps infrastructure

**Labels:** `infrastructure`
**Milestone:** v0.4.0 - Security and Deployment

**Context:** Herald deploys as a single Container App with managed identity connecting to PostgreSQL, Blob Storage, and AI Foundry. The existing `scripts/` directory (bash + `az cli`) is preserved as an alternative — this sub-issue adds Bicep as the primary IaC approach in a new `deploy/` directory.

**Scope:**

**Create `deploy/` with modular Bicep templates:**
```
deploy/
  main.bicep              # Orchestrator: parameters → module calls
  main.bicepparam         # Parameter file (environment-specific values)
  modules/
    environment.bicep     # Container App Environment + Log Analytics workspace
    app.bicep             # Container App (system-assigned MI, ingress, env vars, health probe)
    migration.bicep       # Container Apps Job for database migrations
    postgres.bicep        # PostgreSQL Flexible Server with Entra auth
    storage.bicep         # Storage Account + documents container
```

**`main.bicep`** orchestrates all modules with parameters:
- `environmentName`, `location`, `containerImage`, `containerTag`
- `postgresServerName`, `postgresAdminUser`, `postgresDatabaseName`
- `storageAccountName`, `storageContainerName`
- `aiFoundryResourceName`, `aiFoundryResourceGroup`
- `tenantId`, `clientId` (for auth config)
- `cpuCores`, `memoryGi`, `minReplicas`, `maxReplicas`

**`modules/environment.bicep`:**
- Container App Environment with Log Analytics workspace

**`modules/app.bicep`:**
- Container App with system-assigned managed identity
- Ingress on port 8080, health probe at `/healthz`
- All config via `HERALD_*` env vars (maps to `internal/config/config.go` env var names)
- Resource limits for ImageMagick workloads (default 1.0 CPU, 2Gi memory)
- Scale rules (min/max replicas from parameters)
- Role assignments: Storage Blob Data Contributor, Cognitive Services OpenAI User

**`modules/migration.bicep`:**
- Container Apps Job (manual trigger type)
- Same GHCR image, overridden command to run `herald-migrate -up`
- `HERALD_DB_DSN` env var for database connection

**`modules/postgres.bicep`:**
- PostgreSQL Flexible Server with Entra-only authentication
- Firewall rules for Container App Environment

**`modules/storage.bicep`:**
- Storage Account with `documents` container

**Update Dockerfile** to build both binaries:
```dockerfile
RUN CGO_ENABLED=0 go build -o /herald ./cmd/server
RUN CGO_ENABLED=0 go build -o /herald-migrate ./cmd/migrate
```
Copy both to runtime stage.

**Key files:**
- `Dockerfile` — add migrate binary build
- `internal/config/config.go` — reference for env var names used in Bicep

**Acceptance criteria:**
- [ ] `az bicep build -f deploy/main.bicep` compiles without errors
- [ ] All parameters documented with `@description` decorators
- [ ] No secrets in parameter files (sensitive values via secure parameters)
- [ ] Dockerfile builds both `herald` and `herald-migrate` binaries
- [ ] Migration Job uses same image with overridden command
- [ ] Existing `scripts/` directory untouched

**Dependencies:** #1 (AgentScope env var must exist for Bicep to reference)

---

### 3. Add deployment documentation

**Labels:** `documentation`
**Milestone:** v0.4.0 - Security and Deployment

**Context:** Deployment documentation captures architecture decisions, setup procedures, and operational runbook for IL4/IL6 environments.

**Scope:**
- Create `_project/deployment/`:
  ```
  _project/deployment/
    README.md       # Architecture overview, prerequisites, deployment sequence
    azure-gov.md    # Azure Gov endpoint differences, env var configuration
    runbook.md      # First deploy, updates, migrations, rollback, troubleshooting
  ```
- README.md: architecture diagram, prerequisites, Bicep deployment sequence (`az deployment group create` → migration job → verify health), `HERALD_*` env var reference, image lifecycle (tag → release.yml → GHCR), Bicep module overview
- azure-gov.md: endpoint mapping (commercial → gov), AgentScope env var for Gov cloud, how Bicep parameters differ between commercial and Gov deployments
- runbook.md: first-time deploy, app updates, manual migration trigger (Container Apps Job), rollback, log access, troubleshooting

**Acceptance criteria:**
- [ ] All three documents exist with cross-references
- [ ] Complete `HERALD_*` env var reference
- [ ] Bicep deployment walkthrough with parameter examples
- [ ] No real resource names or secrets (use placeholders)
- [ ] Deployment sequence is followable step-by-step

**Dependencies:** #1, #2 (documents what was built)

---

## Dependency Graph

```
#1 AgentScope configurability
    |
    v
#2 Bicep Manifests + Migration Job + Dockerfile update
    |
    v
#3 Deployment Documentation
```

Strictly sequential: 1 → 2 → 3.

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| IaC approach | Bicep (primary), bash scripts (preserved) | Declarative, compile-time validation, what-if preview, automatic rollback. Compiles to ARM (IL6 compatible). Existing `scripts/` kept as alternative. |
| No config overlays | Env vars only | Container Apps config is entirely env-var driven via Bicep parameters. Overlay files would be redundant documentation artifacts with maintenance burden. |
| Migration strategy | Container Apps Job | Independent execution, separate logging, manual trigger, idempotent `migrate -up`. Init containers share app resources and can't be triggered independently. |
| Config delivery | `HERALD_*` env vars | Container Apps doesn't support config file volume mounts. The overlay system already supports env var overrides as the highest-precedence layer. |
| Single container image | Both binaries in one image | Avoids separate Dockerfile. Migration Job overrides command. Ensures migration code always matches server version. |
| No CD workflow | Bicep + manual deployment | release.yml already handles GHCR builds. Deployment is environment-specific — operators deploy via `az deployment group create`. |
| AgentScope configurable | Config field with commercial default | Only application code change needed. Azure Gov cognitive services uses `.azure.us` domain. Database `TokenScope` is universal (same across clouds). |
