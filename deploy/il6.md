# IL6 Deployment Guide

IL6-specific adjustments for deploying Herald on an air-gapped Azure Government environment. This guide covers Bicep infrastructure changes, CDS artifact handling, and deployment procedures. All commands target **PowerShell 7.5+** on Windows.

For general deployment steps and parameter reference, see [deploy/README.md](README.md). For troubleshooting, see [deploy/troubleshooting.md](troubleshooting.md).

## Prerequisites

- [ ] PowerShell 7.5+ (`$PSVersionTable.PSVersion`)
- [ ] Azure CLI installed and authenticated (`az login --use-device-code`)
- [ ] Azure CLI configured for the IL6 cloud (`az cloud set --name <cloud>`)
- [ ] Standalone `bicep.exe` available (IL6's `az` CLI cannot fetch Bicep over the air gap)
- [ ] Docker Desktop available for image load/tag/push operations
- [ ] Target subscription selected (`az account set --subscription <id>`)

## Bicep Adjustments

### Domain Root Discovery

IL6 uses different service endpoints than commercial Azure. Discover the domain root:

```powershell
az cloud show --query "endpoints" --output json
```

Identify the domain root shared across endpoints. This replaces `windows.net` and `azure.com` in token scopes.

### Token Scope Overrides

Verify the token scopes are reachable, then use them in the parameters file:

```powershell
# PostgreSQL
az account get-access-token `
  --resource "https://ossrdbms-aad.database.<il6-domain-root>" `
  --output json

# Cognitive Services
az account get-access-token `
  --resource "https://cognitiveservices.azure.<il6-domain-root>" `
  --output json
```

| Parameter | Commercial Default | IL6 Override |
|-----------|-------------------|--------------|
| `postgresTokenScope` | `https://ossrdbms-aad.database.windows.net/.default` | `https://ossrdbms-aad.database.<il6-domain-root>//.default` |
| `cognitiveTokenScope` | `https://cognitiveservices.azure.com/.default` | `https://cognitiveservices.azure.<il6-domain-root>/.default` |

> **Important:** The PostgreSQL token scope requires a **trailing slash** on the resource URL before `/.default` (resulting in `//`). Without it, the token audience doesn't match what PostgreSQL expects. Cognitive Services does not require the trailing slash.

> **Note:** The PostgreSQL scope uses `ossrdbms-aad.database.<root>` while Cognitive Services uses `cognitiveservices.azure.<root>` — the patterns differ.

### API Version Discovery

IL6 lags behind commercial Azure on API versions. Query available versions for each resource provider:

```powershell
# Log Analytics Workspaces
az provider show --namespace Microsoft.OperationalInsights `
  --query "resourceTypes[?resourceType=='workspaces'].apiVersions" --output json

# Container Apps, Environments
az provider show --namespace Microsoft.App `
  --query "resourceTypes[?resourceType=='containerApps'].apiVersions" --output json

az provider show --namespace Microsoft.App `
  --query "resourceTypes[?resourceType=='managedEnvironments'].apiVersions" --output json

# PostgreSQL Flexible Server
az provider show --namespace Microsoft.DBforPostgreSQL `
  --query "resourceTypes[?resourceType=='flexibleServers'].apiVersions" --output json

# Storage Accounts
az provider show --namespace Microsoft.Storage `
  --query "resourceTypes[?resourceType=='storageAccounts'].apiVersions" --output json

# Cognitive Services
az provider show --namespace Microsoft.CognitiveServices `
  --query "resourceTypes[?resourceType=='accounts'].apiVersions" --output json

# Managed Identity
az provider show --namespace Microsoft.ManagedIdentity `
  --query "resourceTypes[?resourceType=='userAssignedIdentities'].apiVersions" --output json

# Role Assignments
az provider show --namespace Microsoft.Authorization `
  --query "resourceTypes[?resourceType=='roleAssignments'].apiVersions" --output json

# Container Registry
az provider show --namespace Microsoft.ContainerRegistry `
  --query "resourceTypes[?resourceType=='registries'].apiVersions" --output json

# App Service Plans and Sites
az provider show --namespace Microsoft.Web `
  --query "resourceTypes[?resourceType=='serverfarms'].apiVersions" --output json

az provider show --namespace Microsoft.Web `
  --query "resourceTypes[?resourceType=='sites'].apiVersions" --output json
```

Compare against the versions in `deploy\modules\` and downgrade where needed. Known IL6 downgrades:

| Module | Resource | Commercial Version | IL6 Version |
|--------|----------|-------------------|-------------|
| `logging.bicep` | `workspaces` | `2025-07-01` | `2025-02-01` |
| `environment.bicep` | `workspaces` (existing ref) | `2025-07-01` | `2025-02-01` |

> **Note:** API versions may appear in the provider list but not be fully deployed in your region. If a listed version fails, try the next version down. Prefer non-preview versions.

### Model Availability

Commercial defaults (`gpt-5-mini`, `GlobalStandard`) may not be available in IL6. Check:

```powershell
az cognitiveservices model list `
  --location <region> `
  --query "[].{format:model.format, name:model.name, version:model.version, skus:model.skus[*].name}" `
  --output table
```

Herald requires a vision-capable model. Record the model name, version, and deployment SKU for the parameters file.

### Region Availability

Confirm required services are available in your target region:

```powershell
az provider show --namespace Microsoft.App `
  --query "resourceTypes[?resourceType=='containerApps'].locations" --output json

az provider show --namespace Microsoft.DBforPostgreSQL `
  --query "resourceTypes[?resourceType=='flexibleServers'].locations" --output json

az provider show --namespace Microsoft.CognitiveServices `
  --query "resourceTypes[?resourceType=='accounts'].locations" --output json
```

### Compiling Bicep

Use the standalone `bicep.exe` — the `az` CLI cannot fetch Bicep on an air-gapped network:

```powershell
bicep.exe build deploy\main.bicep
```

This produces `deploy\main.json` for deployment. Validation warnings about missing type definitions are expected on air-gapped networks.

## CDS Artifacts

Herald artifacts are transferred to IL6 via CDS using two independent workflows from the [Herald CDS Proxy](https://github.com/s2va/herald):

### Image Bundle (`herald-v<tag>.tar.gz`)

Triggered by `v*` tags. Contains the container image tarball.

Extract, tag, and push to ACR:

```powershell
tar xzf herald-v<tag>.tar.gz
az acr login -n <acr-name>
docker load -i image.tar
docker tag ghcr.io/jaimestill/herald:<tag> <acr-name>.azurecr.<il6-domain-root>/herald:<tag>
docker push <acr-name>.azurecr.<il6-domain-root>/herald:<tag>
```

### Migrate Bundle (`migrate-herald-migrate-v<tag>.tar.gz`)

Triggered by `migrate-v*` tags. Contains versioned binaries for linux-amd64 and windows-amd64.

Extract and run:

```powershell
tar xzf migrate-herald-migrate-v<tag>.tar.gz
.\migrate-windows-amd64-migrate-v<tag>.exe -dsn "<dsn>" -up
```

## Deployment

### 1. Create Resource Group and ACR

```powershell
az group create `
  --name <resource-group> `
  --location <region>

az acr create `
  --resource-group <resource-group> `
  --name <acr-name> `
  --sku Standard `
  --admin-enabled false
```

### 2. Push Container Image

Extract and push the CDS image bundle (see [CDS Artifacts](#cds-artifacts)).

### 3. Create Entra App Registration

1. Azure Portal → App registrations → New registration
   - Name: `herald`
   - Supported account types: **Single tenant**
   - Redirect URI: leave blank (added post-deploy)

2. Expose an API → Set Application ID URI → Add scope `access_as_user`

3. API permissions → Add delegated permission `api://<client-id>/access_as_user` → Grant admin consent

4. Record **Directory (tenant) ID** and **Application (client) ID**

### 4. Prepare Parameters

Create `deploy\main.parameters.json` with IL6-specific values:

```json
{
  "$schema": "https://schema.management.azure.com/schemas/2019-04-01/deploymentParameters.json#",
  "contentVersion": "0.4.1.0",
  "parameters": {
    "location": {
      "value": "<region>"
    },
    "prefix": {
      "value": "herald"
    },
    "acrName": {
      "value": "heraldregistry"
    },
    "containerImage": {
      "value": "heraldregistry.azurecr.<il6-domain-root>/herald:0.4.1"
    },
    "postgresAdminLogin": {
      "value": "heraldadmin"
    },
    "postgresAdminPassword": {
      "value": "<strong-password>"
    },
    "postgresTokenScope": {
      "value": "https://ossrdbms-aad.database.<il6-domain-root>//.default"
    },
    "cognitiveCustomDomain": {
      "value": "<unique-subdomain>"
    },
    "cognitiveModelName": {
      "value": "<model-name>"
    },
    "cognitiveDeploymentName": {
      "value": "<model-name>"
    },
    "cognitiveModelVersion": {
      "value": "<model-version>"
    },
    "cognitiveDeploymentSku": {
      "value": "<sku>"
    },
    "cognitiveDeploymentCapacity": {
      "value": 1000
    },
    "cognitiveTokenScope": {
      "value": "https://cognitiveservices.azure.<il6-domain-root>/.default"
    },
    "authEnabled": {
      "value": true
    },
    "tenantId": {
      "value": "<tenant-id>"
    },
    "entraClientId": {
      "value": "<client-id>"
    },
    "authAuthority": {
      "value": "https://login.microsoftonline.<il6-domain-root>"
    }
  }
}
```

### 5. Deploy

```powershell
az deployment group create `
  --resource-group <resource-group> `
  --template-file deploy\main.json `
  --parameters deploy\main.parameters.json
```

### 6. Run Migrations

Add a temporary firewall rule, run the migration binary, then remove the rule:

```powershell
$myIp = (Invoke-RestMethod -Uri "https://ifconfig.me/ip").Trim()

az postgres flexible-server firewall-rule create `
  --resource-group <resource-group> `
  --name herald-db `
  --rule-name MigrateAccess `
  --start-ip-address $myIp `
  --end-ip-address $myIp

.\migrate-windows-amd64-migrate-v<tag>.exe `
  -dsn "postgres://<admin-login>:<admin-password>@herald-db.postgres.database.azure.<il6-domain-root>:5432/herald?sslmode=require" `
  -up

az postgres flexible-server firewall-rule delete `
  --resource-group <resource-group> `
  --name herald-db `
  --rule-name MigrateAccess `
  --yes
```

### 7. Upload TLS Certificate

IL6 uses DISA CA certificates not included in the default Alpine CA bundle. Upload the root CA certificate to enable TLS connections to Entra endpoints (OIDC discovery, token acquisition):

1. In Azure Portal → App Service (`herald-app`) → Settings → Certificates
2. Public key certificates (.cer) → Add certificate
3. Upload the DISA root CA `.cer` file

The Bicep template automatically sets `WEBSITE_LOAD_CERTIFICATES=*` and `SSL_CERT_DIR=/var/ssl/certs` to make the certificate available to the container.

### 8. Verify

```powershell
$appFqdn = az containerapp show `
  --name herald-app `
  --resource-group <resource-group> `
  --query "properties.configuration.ingress.fqdn" `
  --output tsv

Invoke-RestMethod -Uri "https://$appFqdn/healthz"
Invoke-RestMethod -Uri "https://$appFqdn/readyz"
```

For App Service:

```powershell
$appFqdn = az webapp show `
  --name herald-app `
  --resource-group <resource-group> `
  --query "defaultHostName" `
  --output tsv

Invoke-RestMethod -Uri "https://$appFqdn/healthz"
Invoke-RestMethod -Uri "https://$appFqdn/readyz"
```

### 9. Post-Deploy: Entra Redirect URI

1. In Azure Portal → App registrations → `herald` → Authentication
2. Add platform → **Single-page application**
3. Add redirect URI: `https://<app-fqdn>/app/`
4. Grant admin consent if required
