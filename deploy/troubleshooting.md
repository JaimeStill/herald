# Troubleshooting

## Deployment

### Cognitive Services `CustomDomainInUse`

**Symptom:** Redeployment fails with `CustomDomainInUse` error

Cognitive Services uses soft-delete — deleted accounts retain their subdomain for a recovery period. Purge the soft-deleted account before redeploying:

```bash
az cognitiveservices account list-deleted --output table

az cognitiveservices account purge \
  --resource-group <resource-group> \
  --name <account-name> \
  --location <region>
```

### Regional Quota and Availability

**Symptom:** `LocationIsOfferRestricted` or `InsufficientQuota` during provisioning

PostgreSQL Burstable tier and Cognitive Services model quotas vary by subscription type and region. Try a different region or SKU. The `cognitiveDeploymentSku` parameter accepts `GlobalStandard`, `DataZoneStandard`, `DataZoneProvisionedManaged`, or `GlobalProvisionedManaged`. The `cognitiveDeploymentCapacity` parameter controls token rate limits in thousands of TPM — reduce it if your subscription has limited quota.

### Cancel a Stuck Deployment

```bash
az deployment group cancel \
  --resource-group <resource-group> \
  --name main
```

ARM deployments are idempotent — rerunning after a fix will pick up where it left off, skipping already-provisioned resources.

## Application

### PostgreSQL Authentication Fails with Managed Identity

**Symptom:** `failed SASL auth: FATAL: password authentication failed for user '<uuid>'`

The `HERALD_DB_USER` must be the Entra admin **principal name** (e.g., `herald-identity`), not the managed identity client ID (UUID). The Bicep template sets `HERALD_DB_USER` to `${prefix}-identity` to match.

### Agent Vision Calls Return 404

**Symptom:** `HTTP 404: Resource not found` on classify workflow

The `HERALD_AGENT_BASE_URL` must include the `/openai` path segment for OpenAI-kind Cognitive Services accounts. The Bicep template appends `/openai` to the endpoint output automatically.

> **Note:** AIServices-kind accounts that expose a unified endpoint do not require the `/openai` segment.

### App Service Image Pull Failure

**Symptom:** `ImagePullUnauthorizedFailure` in App Service Docker logs

Verify:

1. **`acrUserManagedIdentityID` is set to the client ID** (GUID), not the resource ID (ARM path):
   ```bash
   az webapp config show \
     --name <prefix>-app \
     --resource-group <resource-group> \
     --query "{acrUseManagedIdentityCreds:acrUseManagedIdentityCreds, acrUserManagedIdentityID:acrUserManagedIdentityID}" \
     --output json
   ```

2. **AcrPull role is assigned** to the managed identity on the ACR:
   ```bash
   az role assignment list \
     --assignee <principal-id> \
     --scope <acr-resource-id> \
     --all \
     --output table
   ```

3. **ACR admin is enabled** (if using `acrAuthMode=acr_admin`):
   ```bash
   az acr show --name <acr-name> --query "adminUserEnabled" --output tsv
   ```

### TLS Certificate Errors on IL6

**Symptom:** `tls: failed to verify certificate: x509: certificate signed by unknown authority`

IL6 uses DISA CA certificates not in the default Alpine CA bundle. The container cannot verify TLS connections to Entra endpoints (OIDC discovery, PostgreSQL token acquisition) without the root CA.

1. Upload the DISA root CA `.cer` to App Service → Settings → Certificates → Public key certificates
2. Verify these app settings are present (set by Bicep automatically):
   - `WEBSITE_LOAD_CERTIFICATES` = `*`
   - `SSL_CERT_DIR` = `/var/ssl/certs`
3. Restart the App Service

### PostgreSQL Token Audience Mismatch

**Symptom:** `The access token doesn't have a valid audience claim. Acquire a new token for resource "https://ossrdbms-aad.database.<domain>/"`

The PostgreSQL token scope requires a **trailing slash** on the resource URL before `/.default`. The scope should be `https://ossrdbms-aad.database.<il6-domain-root>//.default` (note the `//`). Without the trailing slash, the token audience doesn't match what PostgreSQL expects.

## Migrations

### Dirty Migration State

**Symptom:** Migration fails with `dirty database version N`

A previously failed migration leaves `schema_migrations` in a dirty state. Force the version to recover:

```bash
./migrate -dsn '<dsn>' -force <N>
```

Then re-run:

```bash
./migrate -dsn '<dsn>' -up
```

Alternatively, connect via psql and reset manually:

```sql
SELECT * FROM schema_migrations;
UPDATE schema_migrations SET dirty = false WHERE version = <N>;
```

If the schema is in an inconsistent state and the above doesn't resolve it, drop the tracking table and re-run all migrations from scratch:

```sql
DROP TABLE schema_migrations;
```

```bash
./migrate -dsn '<dsn>' -up
```

## Rollback

### Container App

Container Apps maintains a revision history. To roll back to a previous revision:

```bash
# List revisions
az containerapp revision list \
  --name <prefix>-app \
  --resource-group <resource-group> \
  --output table

# Activate a previous revision
az containerapp revision activate \
  --name <prefix>-app \
  --resource-group <resource-group> \
  --revision <revision-name>

# Route all traffic to the previous revision
az containerapp ingress traffic set \
  --name <prefix>-app \
  --resource-group <resource-group> \
  --revision-weight <revision-name>=100
```

To roll back by redeploying with a previous image tag, re-run the Bicep deployment with `containerImage` set to the previous tag.

## IL6-Specific Issues

### DNS Resolution Failures

**Symptom:** `Failed to resolve '<hostname>' ([Errno 11001] getaddrinfo failed)`

IL6 environments experience intermittent DNS cache staleness. Clear the cache and retry:

```powershell
Clear-DnsClientCache
```

### Stale Authentication

**Symptom:** `az` commands return unexpected errors unrelated to the operation (e.g., `invalid jmespath_type value` on a valid query)

Log out and back in:

```powershell
az logout
az login --use-device-code
```

Stale tokens can produce misleading error messages that don't indicate an auth problem.

### Login Fails After Azure CLI Reinstall

**Symptom:** `AADSTS500011: The resource principal named https://management.core.windows.net/ was not found in the tenant`

Reinstalling Azure CLI wipes `$env:USERPROFILE\.azure`, including the custom IL6 cloud registration. To recover:

1. Open a Cloud Shell in the IL6 Azure Portal
2. Run `az cloud show --output json` to get the full cloud profile
3. On your local machine, re-register the cloud using `az cloud register` with the endpoints from the Cloud Shell output
4. Set the cloud: `az cloud set --name <il6-cloud-name>`
5. Log in: `az login --use-device-code`

### Bicep Cannot Fetch Updates

**Symptom:** `az bicep build` fails with `Failed to resolve 'aka.ms'`

The IL6 air gap prevents `az` CLI from downloading Bicep. Use the standalone `bicep.exe` instead:

```powershell
bicep.exe build deploy\main.bicep
```

Bicep may report warnings about missing type definitions — this is expected and will not block deployment.

### Unsupported API Version

**Symptom:** Deployment fails with `"code":"Unsupported API version"`

IL6 lags behind commercial Azure on API versions. Query available versions and downgrade as needed (see [IL6 Guide](il6.md)).

> **Note:** API versions may appear in the provider list but not yet be fully deployed in a region. If a listed version fails, try the next version down. Prefer non-preview versions for production.

### OnlyWorkloadProfileEnvironmentSupported

**Symptom:** Container App Environment creation fails with `"code":"OnlyWorkloadProfileEnvironmentSupported"`

IL6 regions require workload profile-based environments. The `environment.bicep` module includes a `Consumption` workload profile by default:

```bicep
properties: {
  workloadProfiles: [
    {
      name: 'Consumption'
      workloadProfileType: 'Consumption'
    }
  ]
}
```
