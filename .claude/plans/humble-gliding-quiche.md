# 126 — Validate Deployment Infrastructure and Finalize Documentation

## Context

This is the final sub-issue of Objective 6 (Deployment Configuration). The Bicep infrastructure (#125) and AgentScope configurability (#124) are complete. This task validates the deployment through live commercial Azure deployment, fixes infrastructure issues discovered during validation, finalizes documentation, and sets up the CDS proxy repo for IL6 transfers.

## Execution Sequence

### Phase 1: Fix Bicep Infrastructure

**File:** `deploy/main.bicep`

**Issue found:** `HERALD_AUTH_AGENT_SCOPE` is hardcoded to `https://cognitiveservices.azure.com/.default` (line 234). For Gov cloud deployments this needs to be `https://cognitiveservices.usgovcloudapi.net/.default`. The fix follows the same pattern as `postgresTokenScope` — parameterize with a commercial default.

**Changes:**

1. Add `cognitiveTokenScope` parameter (after `postgresTokenScope`, ~line 38):
   ```bicep
   @description('Cognitive Services Entra token scope')
   #disable-next-line no-hardcoded-env-urls
   param cognitiveTokenScope string = 'https://cognitiveservices.azure.com/.default'
   ```

2. Update `HERALD_AUTH_AGENT_SCOPE` in `baseEnvVars` (line 234) to reference `cognitiveTokenScope` instead of the hardcoded URL.

3. Validate: `az bicep build -f deploy/main.bicep`

### Phase 2: Live Commercial Azure Deployment

Interactive — deploy to a test Azure subscription and verify:

- All resources provision correctly
- Migration job completes
- Health endpoints respond (`/healthz`, `/readyz`)
- Managed identity connects to PostgreSQL, Blob Storage, and Cognitive Services
- Auth flow works with existing Entra app registration (`authEnabled=true`)
- Document any issues or corrections needed

### Phase 3: Documentation Finalization

**File:** `_project/deployment.md`

Based on findings from Phase 2:

- **Azure Government section**: Parameter override table (`postgresTokenScope`, `cognitiveTokenScope`, authority URL)
- **IL6 section rewrite**: Replace manual `docker load`/`docker push` with reference to `s2va/herald` CDS workflow
- **Operational runbook**: Container App logs, rollback procedure, troubleshooting (managed identity, migrations, health probes)
- **Corrections**: Any issues discovered during live deployment
- **Placeholder audit**: Verify no real resource names or secrets

### Phase 4: Proxy Repo Setup

**Location:** `~/code/_s2va/herald/` (local init — GHE push handled separately)

**Structure:**
```
~/code/_s2va/herald/
├── .github/workflows/cds-release.yaml
└── README.md
```

**Workflow design** (single consolidated bundle for CDS auditing):
```
Trigger: tag push matching 'v*'
Runner: s2va-runners
Environment: production

Steps:
1. Checkout JaimeStill/herald at the tagged version
   → Gets deploy/ directory without mirroring

2. Pull GHCR image, save as tarball
   → docker pull + docker save to staging dir

3. Stage deploy/ manifests alongside image tarball

4. Create single herald-<tag>.tar.gz bundle
   → Contains: image.tar + deploy/ (Bicep + parameters)

5. Generate sha256 checksum

6. Azure Login → Upload bundle to CDS blob storage

7. Request CDS transfer via s2va/cds-manifest
```

Reuses existing s2va runner infrastructure, CDS secrets, and `s2va/cds-manifest` action.

## Validation Criteria

- [ ] `az bicep build -f deploy/main.bicep` succeeds after `cognitiveTokenScope` parameterization
- [ ] Commercial deployment provisions and runs successfully
- [ ] Migration job completes without errors
- [ ] Health endpoints respond correctly
- [ ] Managed identity connects to all backing services
- [ ] Auth flow works with Entra app registration
- [ ] `_project/deployment.md` reflects corrections from live testing
- [ ] No real resource names or secrets in documentation
- [ ] Azure Government parameter overrides documented
- [ ] IL6 section references `s2va/herald` CDS workflow
- [ ] Operational runbook covers logs, rollback, troubleshooting
- [ ] CDS workflow YAML is syntactically valid
- [ ] Proxy repo initialized at `~/code/_s2va/herald/`
