# IL6 Deployment Diagnostics Report

Field report documenting issues encountered during the initial Herald deployment to an IL6 Azure Government environment. Prepared for Microsoft support escalation.

## Environment

| Detail | Value |
|--------|-------|
| Azure CLI version | 2.84.0 |
| PowerShell version | 7.5.4 |
| Bicep version | Standalone exe (air-gapped; version TBD) |
| OS | Windows |
| IL6 cloud | Custom registered cloud (not standard AzureUSGovernment) |
| Target region | `<region>` |

## Deployment Overview

Herald is a Go web service deployed as a single Azure Container App with managed identity connecting to PostgreSQL Flexible Server, Blob Storage, Azure OpenAI (Cognitive Services), and Azure Container Registry. Infrastructure is defined as modular Bicep templates compiled to ARM JSON.

### Successfully Provisioned Resources

| Resource | Type | Status |
|----------|------|--------|
| `herald-identity` | User-Assigned Managed Identity | Succeeded |
| `herald-logs` | Log Analytics Workspace | Succeeded |
| `herald-db` | PostgreSQL Flexible Server + database | Succeeded |
| `heraldstorage` | Storage Account + blob container | Succeeded |
| `herald-ai` | Cognitive Services (OpenAI) + model deployment | Succeeded |
| `heraldregistry` | Azure Container Registry (Standard SKU) | Succeeded |
| `herald-env` | Container App Environment | Succeeded (after fixes) |
| RBAC role assignments | AcrPull, Storage Blob Data Contributor, Cognitive Services OpenAI User | Succeeded |

### Failing Resource

| Resource | Type | Status |
|----------|------|--------|
| `herald-app` | Container App | **Failed** — revision provisioning times out |

---

## Issue 1: Unsupported API Versions (Resolved)

### Symptom

Deployment fails with `"code":"Unsupported API version"` for `Microsoft.OperationalInsights/workspaces` and `Microsoft.App` resources.

### Details

The commercial Bicep templates use API versions not yet available on IL6:

| Resource | Commercial Version | IL6 Available |
|----------|-------------------|---------------|
| `Microsoft.OperationalInsights/workspaces` | `2025-07-01` | `2025-02-01` |
| `Microsoft.App/managedEnvironments` | `2026-01-01` | `2025-07-01` |
| `Microsoft.App/containerApps` | `2026-01-01` | `2025-07-01` |
| `Microsoft.App/jobs` | `2026-01-01` | `2025-07-01` |

Note: `Microsoft.App` resources at `2026-01-01` appear in `az provider show` output but fail at deployment with "Unsupported API version." The versions are registered but not functional in this region.

### Resolution

Downgraded `Microsoft.OperationalInsights/workspaces` to `2025-02-01` on the IL6 Bicep copy. Reverted `Microsoft.App` resources to `2025-07-01` (the commercial default, which is supported on IL6).

---

## Issue 2: Workload Profile Required (Resolved)

### Symptom

Container App Environment creation fails with `"code":"OnlyWorkloadProfileEnvironmentSupported","message":"Only workload profile based environment creation is supported"`.

### Details

IL6 does not support consumption-only Container App Environments. A workload profile must be explicitly declared.

### Resolution

Added `Consumption` workload profile to the environment resource:

```bicep
properties: {
  workloadProfiles: [
    {
      name: 'Consumption'
      workloadProfileType: 'Consumption'
    }
  ]
  // ...
}
```

This is backwards-compatible with commercial Azure.

---

## Issue 3: Container App Revision Provisioning Timeout (Unresolved)

### Symptom

Container App resource is created but revision fails to provision:

```
"code": "ContainerAppOperationError"
"message": "Failed to provision revision for container app 'herald-app'. Error details: Operation expired."
```

### Observed Behavior — Bicep Deployment

1. The Container App resource is created with status `Succeeded` initially, then transitions to `Failed`
2. No revisions are created (`az containerapp revision list` returns `[]`)
3. No container logs are produced (no revision = no container = no logs)
4. The Container App resource shows the correct environment variables on `properties.template.containers[0].env`
5. **However**, on one occasion where a revision was briefly created, its `properties.template.containers[0].env` was **empty** — the revision did not inherit the env vars from the app template

### Container App Log Analysis

Log Analytics captured console output from one early failed revision (2026-03-30):

| Field | Value |
|-------|-------|
| RevisionName | `herald-app--<tag>` |
| ContainerAppName | `herald-app` |
| Stream | `stderr` |
| Log | `config load failed:finalize config: database: name required` |
| ContainerImage | `heraldregistry.azurecr.<il6-domain-root>/herald:0.4.0` |

This confirms the image was pulled successfully in at least one early attempt, but the container did not receive its environment variables.

### Triage: Manual Provisioning (2026-04-01)

To isolate whether the issue was Bicep/ARM-specific or platform-level, we deleted `herald-app` and `herald-env`, then manually provisioned both via `az` CLI.

**Environment creation:** Succeeded.

```powershell
az containerapp env create `
  --name herald-env `
  --resource-group heraldgroup `
  --location <region> `
  --logs-workspace-id <customer-id> `
  --logs-workspace-key <shared-key> `
  --enable-workload-profiles
```

**Container App creation:** The app resource was created successfully with all 22 env vars present, and this time the revision was also created with the env vars populated. However, the revision failed with `Pending:InitImagePullBackOff`.

**Note:** The `az containerapp create` CLI rejects `--registry-identity` for non-`.azurecr.io` domains, so the initial creation used a short-lived ACR access token via `--registry-username`/`--registry-password`. The token expired before the platform attempted the pull.

### Triage: Registry Authentication Attempts

We attempted multiple registry authentication methods on the manually-provisioned Container App:

| Method | Command | Result |
|--------|---------|--------|
| Managed identity | `az containerapp registry set --identity <id>` | Revision created, `InitImagePullBackOff` |
| ACR access token | `--registry-username 00000000-... --registry-password <token>` | Revision created, `InitImagePullBackOff` (token expired) |
| ACR admin credentials | `az containerapp registry set --username <acr-name> --password <admin-password>` | Revision created, `InitImagePullBackOff` |
| Fresh delete + recreate with admin credentials | `az containerapp create` with admin creds | Revision created, `InitImagePullBackOff` |

**All authentication methods produce the same `InitImagePullBackOff` result.** The registry configuration on the Container App resource is correct in every case (verified via `az containerapp show --query "properties.configuration.registries"`).

### Key Finding

**Container App revisions on IL6 cannot pull images from ACR regardless of authentication method.** This is not an authentication issue — it is a platform-level connectivity issue between the Container App Environment infrastructure and ACR. The same Bicep configuration deploys successfully on commercial Azure.

Additional observations:
- The Container App Environment has no VNET configuration (`vnetConfiguration: null`)
- ACR has public network access enabled
- ACR is reachable from the deployment workstation (Docker pull succeeds locally)
- The ACR FQDN resolves correctly via DNS
- ACR SKU was upgraded from Basic to Standard with no effect

### Remediation Attempts Summary

| Attempt | Result |
|---------|--------|
| Bicep deployment (multiple cycles) | Revision empty or not created, "Operation expired" |
| Manual CLI provisioning with managed identity ACR pull | `InitImagePullBackOff` |
| Manual CLI provisioning with ACR access token | `InitImagePullBackOff` (token expired) |
| Manual CLI provisioning with ACR admin credentials | `InitImagePullBackOff` |
| Upgraded ACR SKU from Basic to Standard | No effect |
| Multiple full delete + redeploy cycles (Bicep and CLI) | Same result each time |

---

## Questions for Microsoft

1. **Why can Container App revisions on IL6 not pull images from ACR?** All authentication methods fail with `InitImagePullBackOff`. The same configuration works on commercial Azure. Is there a known networking or platform parity gap between Container Apps and ACR on IL6?

2. **Are there additional network configuration requirements for Container Apps on IL6?** The environment has no VNET configuration and ACR has public network access. Is VNET injection or a private endpoint required for Container Apps to reach ACR in this IL6 region?

3. **Why does the `az containerapp create` CLI reject `--registry-identity` for non-`.azurecr.io` ACR domains?** IL6 ACR uses a different domain suffix. This forces credential-based auth via CLI even when managed identity is the intended approach.

4. **Why do `Microsoft.App` API versions `2026-01-01` appear in `az provider show` output but fail at deployment with "Unsupported API version"?** Is there a way to distinguish between registered and actually-deployed API versions?

---

## Interim Mitigation

Due to the Container App ACR pull failure, we are pivoting to Azure App Service for Containers as the compute platform while the Container Apps issue is investigated. The App Service pivot uses the same ACR, managed identity, and supporting infrastructure — only the compute layer changes.

## Next Steps

1. Escalate Container Apps + ACR issue to Microsoft with this report
2. Deploy Herald on App Service for Containers as interim compute platform
3. Track IL6 Container Apps parity for future migration back once the issue is resolved
