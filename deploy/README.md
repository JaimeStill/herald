# Deployment Guide

Herald deploys as a single Azure Container App with managed identity connecting to PostgreSQL, Blob Storage, and AI Foundry. Infrastructure is defined as modular Bicep templates in the `deploy/` directory. Container images are published to [GHCR](https://github.com/JaimeStill/herald/pkgs/container/herald).

## Prerequisites

- [Azure CLI](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli) with Bicep (`az bicep install`)
- An Azure subscription with permission to create resources
- A GHCR token with `read:packages` scope (commercial deployments — see [GHCR Authentication](#ghcr-authentication))
- An Entra app registration (if enabling authentication — see [Entra Configuration](#entra-configuration))

## Architecture

```
deploy/
├── main.bicep              # Orchestrator — parameters → module calls → outputs
├── main.parameters.json    # Non-secret parameter values
└── modules/
    ├── identity.bicep      # User-Assigned Managed Identity
    ├── logging.bicep       # Log Analytics Workspace
    ├── postgres.bicep      # PostgreSQL Flexible Server + database + Entra admin
    ├── storage.bicep       # Storage Account + documents blob container
    ├── cognitive.bicep     # Cognitive Services (OpenAI) + model deployment
    ├── registry.bicep      # Azure Container Registry (IL6 only)
    ├── environment.bicep   # Container App Environment
    ├── app.bicep           # Container App (Herald server)
    ├── migration-job.bicep # Container Apps Job (database migrations)
    └── roles.bicep         # Role assignments for managed identity
```

All modules are orchestrated by `main.bicep`. Resource names follow a `{prefix}-{component}` pattern (e.g., `herald-db`, `herald-identity`).

### Deployment Order

Modules deploy in a serialized chain to avoid ARM race conditions where a resource reports "provisioned" before it is fully ready to accept child operations:

```
identity → logging → postgres → storage → cognitive
  → registry (conditional) → environment → roles → app → migration job
```

### Managed Identity

Herald uses a **user-assigned managed identity** rather than system-assigned. This breaks the circular dependency between the Container App and its role assignments — the identity is created first, roles are assigned, then the app references it. The identity receives:

- **Storage Blob Data Contributor** — read/write documents in Blob Storage
- **Cognitive Services OpenAI User** — call AI Foundry models
- **AcrPull** — pull container images from ACR (IL6 only)

### Container App

- Listens on port 8080 (TLS terminated at the platform level)
- Liveness probe: `GET /healthz`
- Readiness probe: `GET /readyz`
- Default resources: 2.0 CPU, 4Gi memory (ImageMagick workloads need headroom)
- Scale: 1–3 replicas (configurable)
- Ingress idle timeout: 240s (platform default). Active SSE streams are not affected. For longer idle periods, configure [Premium Ingress](https://learn.microsoft.com/en-us/azure/container-apps/premium-ingress) at the environment level.

### Migration Job

A Container Apps Job configured for manual trigger. Uses the same container image as the app with `/usr/local/bin/migrate` as the entrypoint and `-up` as the default argument. The database DSN is stored as a Container Apps secret. Migrations are idempotent — safe to run on every deployment. The command can be overridden at execution time for other operations (force version, rollback, etc.).

### PostgreSQL Authentication

The Flexible Server enables both password and Entra authentication:

- **Password auth** — used by the migration job (golang-migrate requires a standard DSN)
- **Entra token auth** — used by the Container App at runtime via the managed identity

The `HERALD_DB_USER` must be the Entra admin **principal name** (e.g., `herald-identity`), not the managed identity client ID. See [Troubleshooting > PostgreSQL Authentication](#postgresql-authentication-fails-with-managed-identity) for details.

## Configuration

### Parameters

Non-secret parameters are stored in `deploy/main.parameters.json`. Secret values (`postgresAdminPassword`, `ghcrUsername`) are stored in `deploy/main.secrets.json`, which is gitignored. Dynamic secrets like `ghcrPassword` that require shell expansion are supplied at deploy time via the CLI.

Create `deploy/main.secrets.json` from this template:

```json
{
  "$schema": "https://schema.management.azure.com/schemas/2019-04-01/deploymentParameters.json#",
  "contentVersion": "0.4.0.0",
  "parameters": {
    "postgresAdminPassword": {
      "value": "<your-password>"
    },
    "ghcrUsername": {
      "value": "<your-github-username>"
    }
  }
}
```

**Infrastructure:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `location` | — | Azure region (e.g., `eastus`) |
| `prefix` | `herald` | Naming prefix for all resources |
| `tags` | `{}` | Resource tags applied to all resources |

**PostgreSQL:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `postgresAdminLogin` | — | PostgreSQL admin username |
| `postgresAdminPassword` | — | PostgreSQL admin password (**secure**, supply at deploy time) |
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
| `cognitiveDeploymentCapacity` | `1000` | Deployment capacity in thousands of TPM (1000 = 1M TPM) |
| `cognitiveTokenScope` | `https://cognitiveservices.azure.com/.default` | Entra token scope for Cognitive Services |

**Container App:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `containerImage` | — | Full image reference (e.g., `ghcr.io/jaimestill/herald:v0.4.0`) |
| `containerCpu` | `2.0` | CPU cores |
| `containerMemory` | `4Gi` | Memory |
| `minReplicas` | `1` | Minimum replica count |
| `maxReplicas` | `3` | Maximum replica count |

**Registry:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `useAcr` | `false` | Deploy ACR for IL6 (see [IL6 Deployment](#il6-deployment)) |
| `ghcrUsername` | — | GitHub username for GHCR pull |
| `ghcrPassword` | — | GitHub PAT (**secure**, supply at deploy time) |

**Authentication:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `authEnabled` | `false` | Enable Entra authentication |
| `tenantId` | — | Entra tenant ID (required when `authEnabled=true`) |
| `entraClientId` | — | Entra app registration client ID (required when `authEnabled=true`) |

### GHCR Authentication

Commercial deployments pull the container image from GHCR. The `ghcrPassword` parameter accepts any token with `read:packages` scope.

**Using the GitHub CLI token:**

```bash
gh auth token
```

If the token lacks the `read:packages` scope, refresh it:

```bash
gh auth refresh --scopes read:packages
gh auth token
```

**Using a classic PAT:** Generate one at GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic) with the `read:packages` scope.

In both cases, `ghcrUsername` is your GitHub username.

### Entra Configuration

Entra authentication is opt-in via the `authEnabled` parameter. When enabled, the Container App receives `HERALD_AUTH_MODE=azure` and the Entra tenant/client IDs as environment variables. When disabled (default), `HERALD_AUTH_MODE=none` preserves the unauthenticated experience.

#### Prerequisites

An Entra app registration must exist before deployment. See the [Entra section in the root README](../README.md#entra) for app registration setup steps. The same registration works for both local development and production — just add the production redirect URI after deployment.

#### What Bicep Configures

When `authEnabled=true`, these additional environment variables are injected into the Container App:

| Variable | Source | Purpose |
|----------|--------|---------|
| `HERALD_AUTH_TENANT_ID` | `tenantId` param | Entra tenant for OIDC discovery |
| `HERALD_AUTH_CLIENT_ID` | `entraClientId` param | App registration audience for token validation |

The following identity and agent variables are always set (regardless of `authEnabled`):

| Variable | Source | Purpose |
|----------|--------|---------|
| `HERALD_AUTH_MODE` | `azure` or `none` | Enables/disables JWT middleware |
| `HERALD_AUTH_MANAGED_IDENTITY` | `true` | Uses managed identity for token acquisition |
| `HERALD_AGENT_AUTH_TYPE` | `managed_identity` | Agent provider authentication mode |
| `HERALD_AGENT_RESOURCE` | `cognitiveTokenScope` param | Token scope for AI Foundry access |
| `HERALD_AGENT_CLIENT_ID` | Managed identity client ID | User-assigned identity for agent token acquisition |
| `AZURE_CLIENT_ID` | Managed identity client ID | Azure SDK default identity selector |

#### What Bicep Cannot Configure

- **App registration** — create via Azure Portal or `az ad app create` (the Microsoft.Graph Bicep extension is preview-only and not available on IL6)
- **Redirect URIs** — the Container App FQDN is auto-generated; add it to the app registration's SPA platform after deployment
- **Admin consent** — must be granted manually in the portal

## Commercial Deployment (GHCR)

### 1. Validate Templates

```bash
az bicep build -f deploy/main.bicep
```

### 2. Create Resource Group

```bash
az group create \
  --name <resource-group> \
  --location <region>
```

### 3. Deploy

```bash
az deployment group create \
  --resource-group <resource-group> \
  --template-file deploy/main.bicep \
  --parameters deploy/main.parameters.json \
  --parameters deploy/main.secrets.json \
  --parameters ghcrPassword="$(gh auth token)"
```

Authentication is controlled by `authEnabled`, `tenantId`, and `entraClientId` in `main.parameters.json`. Set `authEnabled` to `false` for unauthenticated deployments.

### 4. Run Migrations

```bash
az containerapp job start \
  --name <prefix>-migrate \
  --resource-group <resource-group>
```

The command returns an execution name (e.g., `herald-migrate-u49u1qp`). Check the status:

```bash
az containerapp job execution show \
  --name <prefix>-migrate \
  --resource-group <resource-group> \
  --job-execution-name <execution-name> \
  --output table
```

If the status is `Failed`, inspect the logs:

```bash
az containerapp job logs show \
  --name <prefix>-migrate \
  --resource-group <resource-group> \
  --execution <execution-name> \
  --container <prefix>-migrate
```

To run a different migrate command (e.g., force a version after a dirty state), override the command:

```bash
az containerapp job start \
  --name <prefix>-migrate \
  --resource-group <resource-group> \
  --command "/usr/local/bin/migrate -force <version>"
```

Available flags: `-up`, `-down`, `-force <version>`, `-steps <n>`, `-version`.

### 5. Verify

```bash
# Get the app URL
az deployment group show \
  --resource-group <resource-group> \
  --name <prefix> \
  --query 'properties.outputs.appUrl.value' \
  --output tsv

# Test health
curl -s https://<app-fqdn>/healthz
curl -s https://<app-fqdn>/readyz
```

### 6. Post-Deploy: Entra Redirect URI

If authentication is enabled, add the Container App's auto-generated FQDN to the Entra app registration's SPA redirect URIs:

1. Get the FQDN from the deployment output
2. In Azure Portal → App registrations → your app → Authentication
3. Add `https://<app-fqdn>/app/` as a SPA redirect URI

Multiple redirect URIs are supported — the same app registration can serve local development, staging, and production.

## Azure Government

Override these parameters when deploying to Azure Government (IL4/IL6):

| Parameter | Commercial (default) | Azure Government |
|-----------|---------------------|-----------------|
| `postgresTokenScope` | `https://ossrdbms-aad.database.windows.net/.default` | `https://ossrdbms-aad.database.usgovcloudapi.net/.default` |
| `cognitiveTokenScope` | `https://cognitiveservices.azure.com/.default` | `https://cognitiveservices.usgovcloudapi.net/.default` |

The Herald server also requires the `HERALD_AUTH_AUTHORITY` environment variable for Entra OIDC discovery. In Azure Government, set this to `https://login.microsoftonline.us` (handled automatically when the auth config's `Authority` field is set — see `pkg/auth/config.go`).

## IL6 Deployment

IL6 environments have no access to GHCR. Herald uses Azure Container Registry with managed identity pull instead.

### Transfer via CDS

A proxy repo on GitHub Enterprise handles cross-domain transfers. On each tag push, its workflow:

1. Checks out the Herald source at the tagged version for the `deploy/` directory
2. Pulls the GHCR image and saves it as a tarball
3. Bundles the image tarball and `deploy/` manifests into a single `herald-<tag>.tar.gz`
4. Uploads the bundle to CDS blob storage and requests cross-domain transfer

### IL6 Side

Extract the CDS bundle and import the image to ACR:

```bash
tar xzf herald-<tag>.tar.gz
az acr login -n <acr-name>
docker load -i image.tar
docker tag ghcr.io/jaimestill/herald:<tag> <acr-name>.azurecr.us/herald:<tag>
docker push <acr-name>.azurecr.us/herald:<tag>
```

Deploy with `useAcr=true` and Azure Government token scope overrides:

```bash
az deployment group create \
  --resource-group <resource-group> \
  --template-file deploy/main.bicep \
  --parameters deploy/main.parameters.json \
  --parameters deploy/main.secrets.json \
  --parameters \
    useAcr=true \
    containerImage='<acr-name>.azurecr.us/herald:<tag>' \
    postgresTokenScope='https://ossrdbms-aad.database.usgovcloudapi.net/.default' \
    cognitiveTokenScope='https://cognitiveservices.usgovcloudapi.net/.default'
```

When `useAcr=true`:
- `registry.bicep` deploys an ACR in Herald's resource group
- AcrPull role is assigned to the managed identity
- Container App pulls via managed identity (no registry passwords)

Then run migrations and verify as in the commercial flow.

## Operations

### Updating a Deployment

Deployments are idempotent. To update the container image or change parameters:

```bash
az deployment group create \
  --resource-group <resource-group> \
  --template-file deploy/main.bicep \
  --parameters deploy/main.parameters.json \
  --parameters deploy/main.secrets.json \
  --parameters \
    ghcrPassword="$(gh auth token)" \
    containerImage='ghcr.io/jaimestill/herald:<new-tag>'
```

If the new image includes schema changes, run the migration job after the deployment completes.

> **Note:** Container Apps compares the image digest, not just the tag. Redeploying with the same tag but a different image SHA will trigger a new revision.

### Teardown

```bash
az group delete --name <resource-group> --yes
```

Cognitive Services soft-delete may retain the account. Purge if needed:

```bash
az cognitiveservices account purge \
  --resource-group <resource-group> \
  --name <prefix>-ai \
  --location <region>
```

## Diagnostics

### Container App

**Check app status and FQDN:**

```bash
az containerapp show \
  --name <prefix> \
  --resource-group <resource-group> \
  --query "{state: properties.runningStatus, fqdn: properties.configuration.ingress.fqdn}" \
  --output json
```

**View app logs (last 50 lines):**

```bash
az containerapp logs show \
  --name <prefix> \
  --resource-group <resource-group> \
  --tail 50
```

**Follow app logs in real time:**

```bash
az containerapp logs show \
  --name <prefix> \
  --resource-group <resource-group> \
  --follow
```

**Inspect environment variables:**

```bash
az containerapp show \
  --name <prefix> \
  --resource-group <resource-group> \
  --query "properties.template.containers[0].env" \
  --output json
```

**List revision history:**

```bash
az containerapp revision list \
  --name <prefix> \
  --resource-group <resource-group> \
  --output table
```

### Migration Job

**List all executions:**

```bash
az containerapp job execution list \
  --name <prefix>-migrate \
  --resource-group <resource-group> \
  --output table
```

**Check execution status:**

```bash
az containerapp job execution show \
  --name <prefix>-migrate \
  --resource-group <resource-group> \
  --job-execution-name <execution-name> \
  --output table
```

**View execution logs:**

```bash
az containerapp job logs show \
  --name <prefix>-migrate \
  --resource-group <resource-group> \
  --execution <execution-name> \
  --container <prefix>-migrate
```

### PostgreSQL

**Check server state:**

```bash
az postgres flexible-server show \
  --resource-group <resource-group> \
  --name <prefix>-db \
  --query "{state: state, fqdn: fullyQualifiedDomainName, version: version}" \
  --output json
```

**Connect via psql** (requires firewall rule for your IP):

```bash
# Add temporary firewall rule
MY_IP=$(curl -s ifconfig.me)
az postgres flexible-server firewall-rule create \
  --resource-group <resource-group> \
  --name <prefix>-db \
  --rule-name TempAccess \
  --start-ip-address $MY_IP \
  --end-ip-address $MY_IP

# Connect
PGPASSWORD='<password>' psql \
  "host=<prefix>-db.postgres.database.azure.com port=5432 dbname=herald user=<admin-login> sslmode=require"

# Remove firewall rule when done
az postgres flexible-server firewall-rule delete \
  --resource-group <resource-group> \
  --name <prefix>-db \
  --rule-name TempAccess \
  --yes
```

### Cognitive Services

**Check account and model deployment:**

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

**List soft-deleted accounts** (useful when redeployments fail with `CustomDomainInUse`):

```bash
az cognitiveservices account list-deleted --output table
```

### Deployment Outputs

**Retrieve all deployment outputs:**

```bash
az deployment group show \
  --resource-group <resource-group> \
  --name main \
  --query "properties.outputs" \
  --output json
```

## Troubleshooting

### PostgreSQL Authentication Fails with Managed Identity

**Symptom:** `failed SASL auth: FATAL: password authentication failed for user '<uuid>'`

The `HERALD_DB_USER` must be the Entra admin **principal name** (e.g., `herald-identity`), not the managed identity client ID (UUID). The Bicep template sets `entraAdminPrincipalName: '${prefix}-identity'` and `HERALD_DB_USER` to match. The managed identity client ID is used for `AZURE_CLIENT_ID` and `HERALD_AGENT_CLIENT_ID`, not the database username.

### Agent Vision Calls Return 404

**Symptom:** `HTTP 404: Resource not found` on classify workflow

The `HERALD_AGENT_BASE_URL` must include the `/openai` path segment for OpenAI-kind Cognitive Services accounts. The cognitive services endpoint output is `https://<subdomain>.openai.azure.com/` but the Azure OpenAI REST API expects paths like `/openai/deployments/{deployment}/chat/completions`. The Bicep template appends `/openai` to the endpoint output.

> **Note:** AIServices-kind accounts that expose a unified endpoint do not require the `/openai` segment. If you are using an AIServices-kind account, remove the `/openai` suffix from `HERALD_AGENT_BASE_URL` in the Bicep template.

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

PostgreSQL Burstable tier and Cognitive Services model quotas vary by subscription type and region. Visual Studio Professional subscriptions have restrictions in some regions. Try a different region or SKU. The `cognitiveDeploymentSku` parameter accepts `GlobalStandard`, `DataZoneStandard`, `DataZoneProvisionedManaged`, or `GlobalProvisionedManaged`. The `cognitiveDeploymentCapacity` parameter controls token rate limits in thousands of TPM — reduce it if your subscription has limited quota.

### Dirty Migration State

**Symptom:** Migration job fails with `dirty database version N`

A previously failed migration leaves `schema_migrations` in a dirty state. Connect via psql (see [Diagnostics > PostgreSQL](#postgresql)) and reset:

```sql
-- Check current state
SELECT * FROM schema_migrations;

-- Reset dirty flag
UPDATE schema_migrations SET dirty = false WHERE version = <N>;
```

Then re-run the migration job. Alternatively, use the force command:

```bash
az containerapp job start \
  --name <prefix>-migrate \
  --resource-group <resource-group> \
  --command "/usr/local/bin/migrate -force <N>"
```

If the schema is in an inconsistent state, you may need to manually fix it or drop `schema_migrations` and re-run all migrations from scratch.

### Rollback

Container Apps maintains a revision history. To roll back to a previous revision:

```bash
# List revisions
az containerapp revision list \
  --name <prefix> \
  --resource-group <resource-group> \
  --output table

# Activate a previous revision
az containerapp revision activate \
  --name <prefix> \
  --resource-group <resource-group> \
  --revision <revision-name>

# Route all traffic to the previous revision
az containerapp ingress traffic set \
  --name <prefix> \
  --resource-group <resource-group> \
  --revision-weight <revision-name>=100
```

To roll back by redeploying with a previous image tag, re-run the Bicep deployment with `containerImage` set to the previous tag.

## Environment Variables Reference

All `HERALD_*` environment variables injected into the Container App are composed in `main.bicep` from module outputs and parameters. They map directly to the constants defined in `internal/config/`. See those files for the authoritative list of supported variables and their defaults.

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
