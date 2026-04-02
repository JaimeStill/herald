# IL6 Container App Triage

Triage plan for isolating the Container App provisioning failure on IL6. The goal is to determine whether the issue is in the Bicep deployment pipeline or the Container App platform itself, and to gather precise diagnostic data for Microsoft escalation.

## Phase 1: Analyze Existing Logs

Before tearing anything down, capture diagnostic data from the current failed state.

### 1. Query Log Analytics for Container App Logs

Use the Azure Portal or `az monitor log-analytics query` to check the Log Analytics workspace for container app console logs:

```powershell
az monitor log-analytics query `
  --workspace herald-logs `
  --analytics-query "ContainerAppConsoleLogs_CL | where ContainerAppName_s == 'herald-app' | order by TimeGenerated desc | take 50" `
  --output table
```

If the CLI command doesn't work (IL6 may not support the query subcommand), query via the Azure Portal:

1. Navigate to `herald-logs` Log Analytics workspace
2. Open **Logs** blade
3. Run:
   ```kusto
   ContainerAppConsoleLogs_CL
   | where ContainerAppName_s == "herald-app"
   | order by TimeGenerated desc
   | take 50
   ```

**What to look for:**
- Is the error still `config load failed:finalize config: database: name required`?
- Are there any other errors (image pull failures, crash loops, permission denied)?
- Are there any logs at all from the most recent deployment attempt?

### 2. Verify Revision State

Confirm the current state of the container app and its revisions:

```powershell
# App provisioning state
az containerapp show `
  --name herald-app `
  --resource-group heraldgroup `
  --query "{name:name, state:properties.provisioningState, image:properties.template.containers[0].image}" `
  --output json

# List revisions
az containerapp revision list `
  --name herald-app `
  --resource-group heraldgroup `
  --output json

# Check env vars on the app resource
az containerapp show `
  --name herald-app `
  --resource-group heraldgroup `
  --query "properties.template.containers[0].env" `
  --output json
```

Record whether:
- The app shows env vars but revisions are empty (same as previous observation)
- A revision exists this time and whether it has env vars
- Any new error states

### 3. Capture Deployment Operations

```powershell
az deployment operation group list `
  --resource-group heraldgroup `
  --name main `
  --output json > deployment-operations.json
```

This captures the full operation log for the deployment, including per-resource status and error details.

---

## Phase 2: Clean Up Failed Resources

Delete the Container App and Environment so we can recreate them manually with full control.

### 1. Delete Container App

```powershell
az containerapp delete `
  --name herald-app `
  --resource-group heraldgroup `
  --yes
```

### 2. Delete Container App Environment

```powershell
az containerapp env delete `
  --name herald-env `
  --resource-group heraldgroup `
  --yes
```

### 3. Verify Cleanup

```powershell
az resource list `
  --resource-group heraldgroup `
  --output table
```

Confirm only the following remain: `herald-identity`, `herald-logs`, `herald-db`, `heraldstorage`, `herald-ai`, `heraldregistry`.

---

## Phase 3: Manual Provisioning

Recreate the environment and container app manually via `az` CLI to isolate whether the issue is Bicep-specific or platform-level.

### 1. Gather Resource Values

These values are needed for the manual provisioning commands. Retrieve them from the existing resources:

```powershell
# Managed identity resource ID and client ID
$identityId = az identity show `
  --name herald-identity `
  --resource-group heraldgroup `
  --query "id" --output tsv

$identityClientId = az identity show `
  --name herald-identity `
  --resource-group heraldgroup `
  --query "clientId" --output tsv

# Log Analytics workspace customer ID and shared key
$workspaceCustomerId = az monitor log-analytics workspace show `
  --workspace-name herald-logs `
  --resource-group heraldgroup `
  --query "customerId" --output tsv

$workspaceKey = az monitor log-analytics workspace get-shared-keys `
  --workspace-name herald-logs `
  --resource-group heraldgroup `
  --query "primarySharedKey" --output tsv

# PostgreSQL FQDN
$pgFqdn = az postgres flexible-server show `
  --name herald-db `
  --resource-group heraldgroup `
  --query "fullyQualifiedDomainName" --output tsv

# Storage blob endpoint
$blobEndpoint = az storage account show `
  --name heraldstorage `
  --resource-group heraldgroup `
  --query "primaryEndpoints.blob" --output tsv

# Cognitive Services endpoint and deployment name
$aiEndpoint = az cognitiveservices account show `
  --name herald-ai `
  --resource-group heraldgroup `
  --query "properties.endpoint" --output tsv

$aiDeployment = az cognitiveservices account deployment list `
  --name herald-ai `
  --resource-group heraldgroup `
  --query "[0].name" --output tsv
```

### 2. Create Container App Environment

```powershell
az containerapp env create `
  --name herald-env `
  --resource-group heraldgroup `
  --location <region> `
  --logs-workspace-id $workspaceCustomerId `
  --logs-workspace-key $workspaceKey `
  --enable-workload-profiles
```

The `--enable-workload-profiles` flag creates the environment with a Consumption workload profile, which IL6 requires.

Verify:

```powershell
az containerapp env show `
  --name herald-env `
  --resource-group heraldgroup `
  --query "{name:name, state:properties.provisioningState}" `
  --output json
```

### 3. Create Container App

Create the container app with all environment variables inline. This is the critical test — if this succeeds, the issue is in the Bicep deployment pipeline. If it fails with the same error, it's a platform issue.

```powershell
az containerapp create `
  --name herald-app `
  --resource-group heraldgroup `
  --environment herald-env `
  --image heraldregistry.azurecr.<il6-domain-root>/herald:0.4.0 `
  --registry-server heraldregistry.azurecr.<il6-domain-root> `
  --registry-identity $identityId `
  --user-assigned $identityId `
  --target-port 8080 `
  --ingress external `
  --cpu 2.0 --memory 4Gi `
  --min-replicas 1 --max-replicas 3 `
  --env-vars `
    HERALD_ENV=azure `
    HERALD_SERVER_PORT=8080 `
    HERALD_DB_HOST=$pgFqdn `
    HERALD_DB_PORT=5432 `
    HERALD_DB_NAME=herald `
    HERALD_DB_USER=herald-identity `
    HERALD_DB_SSL_MODE=require `
    "HERALD_DB_TOKEN_SCOPE=https://ossrdbms-aad.database.<il6-domain-root>/.default" `
    HERALD_STORAGE_SERVICE_URL=$blobEndpoint `
    HERALD_STORAGE_CONTAINER_NAME=documents `
    HERALD_AUTH_MODE=azure `
    HERALD_AUTH_MANAGED_IDENTITY=true `
    HERALD_AGENT_PROVIDER_NAME=azure `
    "HERALD_AGENT_BASE_URL=${aiEndpoint}openai" `
    HERALD_AGENT_DEPLOYMENT=$aiDeployment `
    HERALD_AGENT_API_VERSION=2025-04-01-preview `
    HERALD_AGENT_AUTH_TYPE=managed_identity `
    "HERALD_AGENT_RESOURCE=https://cognitiveservices.azure.<il6-domain-root>/.default" `
    HERALD_AGENT_CLIENT_ID=$identityClientId `
    AZURE_CLIENT_ID=$identityClientId `
    HERALD_AUTH_TENANT_ID=<tenant-id> `
    HERALD_AUTH_CLIENT_ID=<entra-client-id>
```

> **Note:** The `--registry-identity` flag may reject non-`.azurecr.io` servers (see diagnostics report). If it does, use credential-based auth instead:
> ```powershell
> $token = az acr login -n heraldregistry --expose-token --query "accessToken" --output tsv
>
> # Replace the --registry-identity line with:
> --registry-username 00000000-0000-0000-0000-000000000000 `
> --registry-password $token `
> ```

### 4. Verify Container App

```powershell
# Check provisioning state
az containerapp show `
  --name herald-app `
  --resource-group heraldgroup `
  --query "{name:name, state:properties.provisioningState}" `
  --output json

# Check revision exists and has env vars
az containerapp revision list `
  --name herald-app `
  --resource-group heraldgroup `
  --output json

# Check logs
az containerapp logs show `
  --name herald-app `
  --resource-group heraldgroup `
  --tail 50
```

### 5. Health Check (if provisioning succeeds)

```powershell
$appFqdn = az containerapp show `
  --name herald-app `
  --resource-group heraldgroup `
  --query "properties.configuration.ingress.fqdn" `
  --output tsv

Invoke-RestMethod -Uri "https://$appFqdn/healthz"
Invoke-RestMethod -Uri "https://$appFqdn/readyz"
```

---

## Phase 4: Assess Results

### If Manual Provisioning Succeeds

The issue is in how Bicep/ARM creates the Container App — likely the revision template not receiving the env vars. Next steps:
- Compare the manually-created app's revision template against the Bicep-created one
- Investigate whether ARM deployment ordering or timing causes the revision to be created before env vars are applied
- Consider restructuring the Bicep to create the Container App without env vars first, then update it in a second deployment step

### If Manual Provisioning Fails with "Operation expired"

The issue is at the Container App platform level on IL6. Escalate to Microsoft with the diagnostics report (`deploy/il6-diagnostics.md`). Key questions:
- Is there a known issue with Container Apps and managed identity ACR pull in this IL6 region?
- Are there additional network or configuration requirements for Container Apps in this environment?
- Is the `2025-07-01` API version fully supported for `containerApps` in this region?

### If Manual Provisioning Fails with "database: name required"

The env vars are not being injected into the container at runtime despite being set on the resource. Escalate to Microsoft — this is a platform bug. Capture:
- The revision template (`az containerapp revision show --query "properties.template"`)
- The app template (`az containerapp show --query "properties.template"`)
- Compare the two to see if env vars are present in both
