# Plan: Simplify Bicep to ACR-Only Registry with Dual Auth Mode

## Context

The GHCR registry path in the Bicep infrastructure has been causing ARM template evaluation issues — both branches of conditional expressions get evaluated, resulting in empty GHCR credentials polluting App Service deployments even when ACR is configured. Additionally, aligning commercial and IL6 deployments on a single ACR-based approach creates a single source of truth and eliminates an entire category of configuration complexity. Another IL6 capability uses managed identity + AcrPull with ACR successfully, suggesting the managed identity path should work once GHCR concerns are removed.

## Changes

### 1. `deploy/main.bicep` — Remove all GHCR infrastructure

**Remove parameters:**
- `ghcrUsername`
- `ghcrPassword`

**Remove variables:**
- `ghcrRegistries`
- `ghcrSecrets`

**Change `acrName` from optional to required** — remove default empty string. ACR is now always required.

**Remove `useAcr` variable** — ACR is always used. Simplify `useAcrAdmin` and `useAcrManagedIdentity` to derive directly from `acrAuthMode`:
```bicep
var useAcrAdmin = acrAuthMode == 'acr_admin'
var useAcrManagedIdentity = acrAuthMode == 'managed_identity'
```

**Simplify registry configuration** — no more GHCR fallback:
```bicep
var registries = useAcrAdmin ? acrAdminRegistries : acrManagedIdentityRegistries
var containerAppSecrets = useAcrAdmin ? acrAdminSecrets : []
```

**Simplify roles module call:**
```bicep
assignAcrPull: useAcrManagedIdentity
acrId: useAcrManagedIdentity ? acrId : ''
```

**Simplify App Service module call** — remove `ghcrUsername`/`ghcrPassword` params.

**ACR `existing` resource** — remove the `if (useAcr)` condition since ACR is always present:
```bicep
resource acr 'Microsoft.ContainerRegistry/registries@2025-11-01' existing = {
  name: acrName
}
```

**Simplify ACR variable access** — no more safe-access operators for ACR since it's always present:
```bicep
var acrId = acr.id
var acrLoginServer = acr.properties.loginServer
```

`listCredentials()` still needs the `useAcrAdmin` guard.

### 2. `deploy/modules/appservice.bicep` — Remove GHCR settings

**Remove parameters:**
- `ghcrUsername`
- `ghcrPassword`

**Remove `ghcrDockerSettings` variable.**

**Simplify `dockerSettings`:**
```bicep
var dockerSettings = useAcrAdmin ? acrAdminDockerSettings : []
```

### 3. `deploy/modules/app.bicep` — No changes needed

The Container App module receives `registries` and `secrets` arrays from main.bicep. The simplification happens upstream.

### 4. `deploy/main.parameters.json` — Update for ACR

Replace GHCR image reference with ACR. Add `acrName`:
```json
{
  "location": { "value": "centralus" },
  "prefix": { "value": "herald" },
  "acrName": { "value": "heraldregistry" },
  "containerImage": { "value": "heraldregistry.azurecr.io/herald:0.4.1" },
  "postgresAdminLogin": { "value": "heraldadmin" },
  "cognitiveCustomDomain": { "value": "herald-ai-prod" },
  "authEnabled": { "value": true },
  "tenantId": { "value": "<existing-tenant-id>" },
  "entraClientId": { "value": "<existing-client-id>" }
}
```

### 5. `deploy/main.secrets.json` — Simplify

Only `postgresAdminPassword` remains. No more GHCR credentials:
```json
{
  "parameters": {
    "postgresAdminPassword": { "value": "<password>" }
  }
}
```

### 6. `deploy/README.md` — Significant cleanup

- Remove all GHCR references (parameter table, auth section, deployment commands)
- Remove `ghcrUsername`/`ghcrPassword` from parameter tables
- Change `acrName` description from "leave empty for GHCR" to required
- Add `acrAuthMode` to parameter table
- Update deployment commands to remove `ghcrPassword` CLI parameter
- Update secrets template (remove GHCR credentials)
- Simplify IL6 section (no more GHCR vs ACR distinction)

### 7. `deploy/il6.md` — Rewrite with updated Bicep details

Archive existing il6-*.md content, rewrite il6.md with:
- Updated parameters template (no GHCR, `acrAuthMode` included)
- Updated troubleshooting reflecting current state
- Standalone migration binary workflow
- Updated API version discovery table

### 8. Archive il6-*.md files

Move `il6-triage.md`, `il6-diagnostics.md`, `il6-migrate-changes.md` content to an archive or delete — these are transient operational docs. The user has already been updating IL6 independently and will delete these once finalized.

## Files to Modify

| File | Changes |
|------|---------|
| `deploy/main.bicep` | Remove GHCR params/vars, make `acrName` required, simplify conditionals |
| `deploy/modules/appservice.bicep` | Remove GHCR params/vars, simplify docker settings |
| `deploy/main.parameters.json` | ACR image reference, add `acrName` |
| `deploy/README.md` | Remove GHCR docs, add `acrAuthMode`, update commands |
| `deploy/il6.md` | Rewrite with updated infrastructure |

## Files Unchanged

| File | Reason |
|------|--------|
| `deploy/modules/app.bicep` | Receives arrays from main.bicep, no GHCR-specific logic |
| `deploy/modules/appservice-plan.bicep` | No registry concerns |
| `deploy/modules/roles.bicep` | Already correct — AcrPull only assigned for managed_identity mode |
| `deploy/modules/*.bicep` (shared) | identity, logging, postgres, storage, cognitive — no registry concerns |

## Verification

1. `az bicep build -f deploy/main.bicep` — compiles with no errors/warnings (except BCP422 for listCredentials)
2. Purge commercial deployment: `az group delete --name HeraldDeploymentGroup --yes`
3. Create fresh commercial deployment with `heraldgroup`:
   - Create resource group + ACR
   - Push image to ACR  
   - Deploy with `computeTarget=containerapp` (default) and `acrAuthMode=managed_identity` (default)
4. Verify `/healthz` and `/readyz`
5. Test `computeTarget=appservice` on commercial
6. Transfer to IL6 and deploy fresh
