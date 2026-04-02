# Deployment Guide

Herald deploys as a containerized Azure service with managed identity connecting to PostgreSQL, Blob Storage, and AI Foundry. Infrastructure is defined as modular Bicep templates in the `deploy/` directory. Container images are stored in Azure Container Registry.

## Prerequisites

- [Azure CLI](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli) with Bicep (`az bicep install`)
- An Azure subscription with permission to create resources
- A pre-existing Azure Container Registry with the Herald image pushed
- An Entra app registration (if enabling authentication — see [Entra Configuration](#entra-configuration))

## Architecture

```
deploy/
├── main.bicep                 # Orchestrator — parameters → module calls → outputs
├── main.parameters.json       # Non-secret parameter values
└── modules/
    ├── identity.bicep         # User-Assigned Managed Identity
    ├── logging.bicep          # Log Analytics Workspace
    ├── postgres.bicep         # PostgreSQL Flexible Server + database + Entra admin
    ├── storage.bicep          # Storage Account + documents blob container
    ├── cognitive.bicep        # Cognitive Services (OpenAI) + model deployment
    ├── environment.bicep      # Container App Environment (containerapp target)
    ├── app.bicep              # Container App (containerapp target)
    ├── appservice-plan.bicep  # App Service Plan (appservice target)
    ├── appservice.bicep       # Web App for Containers (appservice target)
    └── roles.bicep            # Role assignments for managed identity
```

All modules are orchestrated by `main.bicep`. Resource names follow a `{prefix}-{component}` pattern (e.g., `herald-db`, `herald-identity`).

### Deployment Order

Modules deploy in a serialized chain to avoid ARM race conditions:

```
Shared:        identity → logging → postgres → storage → cognitive
containerapp:  → environment → roles → app
appservice:    → app service plan → roles → app service
```

The `computeTarget` parameter selects which branch deploys.

### Compute Targets

**Container Apps** (`computeTarget=containerapp`, default) — serverless containers with scale-to-zero, revision-based rollback, and consumption billing. Best for cost-efficiency.

**App Service** (`computeTarget=appservice`) — dedicated Linux container hosting on an App Service Plan. Fixed-cost, always-on. Used as an alternative when Container Apps is not available.

### Managed Identity

Herald uses a **user-assigned managed identity** rather than system-assigned. This breaks the circular dependency between the compute resource and its role assignments. The identity receives:

- **Storage Blob Data Contributor** — read/write documents in Blob Storage
- **Cognitive Services OpenAI User** — call AI Foundry models
- **AcrPull** — pull container images from ACR (when `acrAuthMode=managed_identity`)

### ACR Authentication

The `acrAuthMode` parameter controls how the compute target authenticates to ACR:

- **`managed_identity`** (default) — the managed identity pulls images via the AcrPull role. No credentials stored.
- **`acr_admin`** — ACR admin credentials are injected as secrets (Container Apps) or Docker registry settings (App Service). Use as a fallback if managed identity pull is not working.

### Migrations

Database migrations use the standalone `migrate` binary, published as a [GitHub Release](https://github.com/JaimeStill/herald/releases) artifact under `migrate-v*` tags. The binary embeds all SQL migrations and connects directly to PostgreSQL via a DSN. See [Running Migrations](#running-migrations).

### PostgreSQL Authentication

The Flexible Server enables both password and Entra authentication:

- **Password auth** — used by the migration binary (golang-migrate requires a standard DSN)
- **Entra token auth** — used by the compute target at runtime via the managed identity

## Configuration

### Parameters

Non-secret parameters are stored in `deploy/main.parameters.json`. The `postgresAdminPassword` is stored in `deploy/main.secrets.json` (gitignored).

Create `deploy/main.secrets.json`:

```json
{
  "$schema": "https://schema.management.azure.com/schemas/2019-04-01/deploymentParameters.json#",
  "contentVersion": "0.4.1.0",
  "parameters": {
    "postgresAdminPassword": {
      "value": "<your-password>"
    }
  }
}
```

**Infrastructure:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `location` | — | Azure region (e.g., `centralus`) |
| `prefix` | `herald` | Naming prefix for all resources |
| `tags` | `{}` | Resource tags applied to all resources |
| `computeTarget` | `containerapp` | Compute platform: `containerapp` or `appservice` |
| `appServiceSkuName` | `P1v3` | App Service Plan SKU (only when `computeTarget` is `appservice`) |

**PostgreSQL:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `postgresAdminLogin` | — | PostgreSQL admin username |
| `postgresAdminPassword` | — | PostgreSQL admin password (**secure**) |
| `postgresSkuName` | `Standard_B1ms` | PostgreSQL SKU |
| `postgresSkuTier` | `Burstable` | PostgreSQL tier |
| `postgresStorageSizeGB` | `32` | PostgreSQL storage |
| `postgresTokenScope` | `https://ossrdbms-aad.database.windows.net/.default` | Entra token scope for PostgreSQL |

**Cognitive Services:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `cognitiveCustomDomain` | — | Cognitive Services subdomain (**globally unique**) |
| `cognitiveDeploymentName` | `gpt-5-mini` | Model deployment name |
| `cognitiveModelName` | `gpt-5-mini` | Model name |
| `cognitiveModelVersion` | `2025-08-07` | Model version |
| `cognitiveDeploymentSku` | `GlobalStandard` | Deployment SKU |
| `cognitiveDeploymentCapacity` | `1000` | Deployment capacity (TPM in thousands) |
| `cognitiveTokenScope` | `https://cognitiveservices.azure.com/.default` | Entra token scope for Cognitive Services |

**Container:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `containerImage` | — | Full image reference (e.g., `heraldregistry.azurecr.io/herald:0.4.1`) |
| `containerCpu` | `2.0` | CPU cores (Container Apps only) |
| `containerMemory` | `4Gi` | Memory (Container Apps only) |
| `minReplicas` | `0` | Minimum replica count (0 enables scale-to-zero, Container Apps only) |
| `maxReplicas` | `3` | Maximum replica count (Container Apps only) |

**Registry:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `acrName` | — | Pre-existing ACR name in this resource group |
| `acrAuthMode` | `managed_identity` | ACR auth: `managed_identity` or `acr_admin` |

**Authentication:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `authEnabled` | `false` | Enable Entra authentication |
| `tenantId` | — | Entra tenant ID (required when `authEnabled=true`) |
| `entraClientId` | — | Entra app registration client ID (required when `authEnabled=true`) |

### Entra Configuration

Entra authentication is opt-in via the `authEnabled` parameter. When enabled, the compute target receives `HERALD_AUTH_MODE=azure` and the Entra tenant/client IDs as environment variables.

#### Prerequisites

An Entra app registration must exist before deployment. See the [Entra section in the root README](../README.md#entra) for setup steps.

#### What Bicep Configures

When `authEnabled=true`, these additional environment variables are injected:

| Variable | Source |
|----------|--------|
| `HERALD_AUTH_TENANT_ID` | `tenantId` param |
| `HERALD_AUTH_CLIENT_ID` | `entraClientId` param |

#### What Bicep Cannot Configure

- **App registration** — create via Azure Portal or `az ad app create`
- **Redirect URIs** — the compute FQDN is auto-generated; add it post-deployment
- **Admin consent** — must be granted manually in the portal

## Deployment

### 1. Create Resource Group and ACR

```bash
az group create --name <resource-group> --location <region>

az acr create \
  --resource-group <resource-group> \
  --name <acr-name> \
  --sku Standard \
  --admin-enabled false
```

### 2. Push Image to ACR

```bash
az acr login -n <acr-name>
docker tag ghcr.io/jaimestill/herald:<tag> <acr-name>.azurecr.io/herald:<tag>
docker push <acr-name>.azurecr.io/herald:<tag>
```

### 3. Deploy

```bash
az deployment group create \
  --resource-group <resource-group> \
  --template-file deploy/main.bicep \
  --parameters deploy/main.parameters.json \
  --parameters deploy/main.secrets.json
```

To deploy with App Service instead of Container Apps:

```bash
az deployment group create \
  --resource-group <resource-group> \
  --template-file deploy/main.bicep \
  --parameters deploy/main.parameters.json \
  --parameters deploy/main.secrets.json \
  --parameters computeTarget=appservice
```

### 4. Run Migrations

See [Running Migrations](#running-migrations).

### 5. Verify

```bash
APP_URL=$(az deployment group show \
  --resource-group <resource-group> \
  --name main \
  --query 'properties.outputs.appUrl.value' \
  --output tsv)

curl -s $APP_URL/healthz
curl -s $APP_URL/readyz
```

### 6. Post-Deploy: Entra Redirect URI

If authentication is enabled, add the auto-generated FQDN to the Entra app registration:

1. Get the FQDN from the deployment output
2. In Azure Portal → App registrations → your app → Authentication
3. Add `https://<app-fqdn>/app/` as a SPA redirect URI

## Running Migrations

Migrations use the standalone `migrate` binary, published as a GitHub Release artifact under `migrate-v*` tags. The binary embeds all SQL migrations and connects directly to PostgreSQL via a DSN.

### Prerequisites

The deployment machine must have network access to the PostgreSQL server. Add a temporary firewall rule:

```bash
MY_IP=$(curl -s ifconfig.me)
az postgres flexible-server firewall-rule create \
  --resource-group <resource-group> \
  --name <prefix>-db \
  --rule-name MigrateAccess \
  --start-ip-address $MY_IP \
  --end-ip-address $MY_IP
```

### Run

```bash
./migrate -dsn 'postgres://<admin-login>:<admin-password>@<prefix>-db.postgres.database.azure.com:5432/herald?sslmode=require' -up
```

Available flags: `-up`, `-down`, `-force <version>`, `-steps <n>`, `-version`.

On Windows (IL6), use `migrate.exe` instead of `./migrate`.

### Cleanup

```bash
az postgres flexible-server firewall-rule delete \
  --resource-group <resource-group> \
  --name <prefix>-db \
  --rule-name MigrateAccess \
  --yes
```

## Operations

### Updating a Deployment

Deployments are idempotent. To update the container image or change parameters:

```bash
az deployment group create \
  --resource-group <resource-group> \
  --template-file deploy/main.bicep \
  --parameters deploy/main.parameters.json \
  --parameters deploy/main.secrets.json
```

If the new image includes schema changes, run migrations after the deployment completes.

### Teardown

```bash
az group delete --name <resource-group> --yes
```

Cognitive Services soft-delete may retain the account. Check and purge:

```bash
az cognitiveservices account list-deleted --output table

az cognitiveservices account purge \
  --resource-group <resource-group> \
  --name <prefix>-ai \
  --location <region>
```

## Diagnostics

### Container App

```bash
# Status and FQDN
az containerapp show \
  --name <prefix>-app \
  --resource-group <resource-group> \
  --query "{state: properties.runningStatus, fqdn: properties.configuration.ingress.fqdn}" \
  --output json

# Logs
az containerapp logs show \
  --name <prefix>-app \
  --resource-group <resource-group> \
  --tail 50

# Environment variables
az containerapp show \
  --name <prefix>-app \
  --resource-group <resource-group> \
  --query "properties.template.containers[0].env" \
  --output json

# Revision history
az containerapp revision list \
  --name <prefix>-app \
  --resource-group <resource-group> \
  --output table
```

### App Service

```bash
# Status
az webapp show \
  --name <prefix>-app \
  --resource-group <resource-group> \
  --query "{state: state, defaultHostName: defaultHostName}" \
  --output json

# Logs
az webapp log config \
  --name <prefix>-app \
  --resource-group <resource-group> \
  --docker-container-logging filesystem

az webapp log tail \
  --name <prefix>-app \
  --resource-group <resource-group>
```

### PostgreSQL

```bash
az postgres flexible-server show \
  --resource-group <resource-group> \
  --name <prefix>-db \
  --query "{state: state, fqdn: fullyQualifiedDomainName, version: version}" \
  --output json
```

### Cognitive Services

```bash
az cognitiveservices account show \
  --resource-group <resource-group> \
  --name <prefix>-ai \
  --query "{endpoint: properties.endpoint, state: properties.provisioningState}" \
  --output json

az cognitiveservices account deployment list \
  --resource-group <resource-group> \
  --name <prefix>-ai \
  --output table
```

### Deployment Outputs

```bash
az deployment group show \
  --resource-group <resource-group> \
  --name main \
  --query "properties.outputs" \
  --output json
```

## Environment Variables Reference

All `HERALD_*` environment variables injected into the compute target are composed in `main.bicep` from module outputs and parameters.

| Variable | Source | Always Set |
|----------|--------|------------|
| `HERALD_ENV` | `azure` | Yes |
| `HERALD_SERVER_PORT` | `8080` | Yes |
| `HERALD_DB_HOST` | `postgres.outputs.fqdn` | Yes |
| `HERALD_DB_PORT` | `5432` | Yes |
| `HERALD_DB_NAME` | `postgres.outputs.databaseName` | Yes |
| `HERALD_DB_USER` | `${prefix}-identity` | Yes |
| `HERALD_DB_SSL_MODE` | `require` | Yes |
| `HERALD_DB_TOKEN_SCOPE` | `postgresTokenScope` param | Yes |
| `HERALD_STORAGE_SERVICE_URL` | `storage.outputs.blobEndpoint` | Yes |
| `HERALD_STORAGE_CONTAINER_NAME` | `documents` | Yes |
| `HERALD_AUTH_MODE` | `azure` or `none` | Yes |
| `HERALD_AUTH_MANAGED_IDENTITY` | `true` | Yes |
| `HERALD_AGENT_PROVIDER_NAME` | `azure` | Yes |
| `HERALD_AGENT_BASE_URL` | `cognitive.outputs.endpoint` + `openai` | Yes |
| `HERALD_AGENT_DEPLOYMENT` | `cognitive.outputs.modelDeploymentName` | Yes |
| `HERALD_AGENT_API_VERSION` | `2025-04-01-preview` | Yes |
| `HERALD_AGENT_AUTH_TYPE` | `managed_identity` | Yes |
| `HERALD_AGENT_RESOURCE` | `cognitiveTokenScope` param | Yes |
| `HERALD_AGENT_CLIENT_ID` | `identity.outputs.clientId` | Yes |
| `AZURE_CLIENT_ID` | `identity.outputs.clientId` | Yes |
| `HERALD_AUTH_TENANT_ID` | `tenantId` param | Auth only |
| `HERALD_AUTH_CLIENT_ID` | `entraClientId` param | Auth only |
