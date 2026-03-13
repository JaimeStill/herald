# Deployment Guide

Herald deploys as a single Azure Container App with managed identity connecting to PostgreSQL, Blob Storage, and AI Foundry. Infrastructure is defined as modular Bicep templates in the `deploy/` directory.

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

### Managed Identity

Herald uses a **user-assigned managed identity** rather than system-assigned. This breaks the circular dependency between the Container App and its role assignments — the identity is created first, roles are assigned, then the app references it. The identity receives:

- **Storage Blob Data Contributor** — read/write documents in Blob Storage
- **Cognitive Services OpenAI User** — call AI Foundry models
- **AcrPull** — pull container images from ACR (IL6 only)

### Container App

- Listens on port 8080 (TLS terminated at the platform level)
- Liveness probe: `GET /healthz`
- Readiness probe: `GET /readyz`
- Default resources: 1.0 CPU, 2Gi memory (ImageMagick workloads)
- Scale: 1–3 replicas (configurable)

### Migration Job

A Container Apps Job configured for manual trigger. Uses the same container image as the app with `/usr/local/bin/migrate` as the entrypoint and `-up` as the default argument. The database DSN is stored as a Container Apps secret. Migrations are idempotent — safe to run on every deployment. The command can be overridden at execution time for other operations (force version, rollback, etc.).

### PostgreSQL Authentication

The Flexible Server enables both password and Entra authentication:

- **Password auth** — used by the migration job (golang-migrate requires a standard DSN)
- **Entra token auth** — used by the Container App at runtime via the managed identity

## Parameters

Non-secret parameters are stored in `deploy/main.parameters.json`. Secret values are supplied at deploy time via the CLI.

| Parameter | Default | Description |
|-----------|---------|-------------|
| `location` | — | Azure region (e.g., `eastus`) |
| `prefix` | `herald` | Naming prefix for all resources |
| `containerImage` | — | Full image reference |
| `postgresAdminLogin` | — | PostgreSQL admin username |
| `postgresAdminPassword` | — | PostgreSQL admin password (**secure**, supply at deploy time) |
| `postgresSkuName` | `Standard_B1ms` | PostgreSQL SKU |
| `postgresSkuTier` | `Burstable` | PostgreSQL tier |
| `postgresStorageSizeGB` | `32` | PostgreSQL storage |
| `postgresTokenScope` | `https://ossrdbms-aad.database.windows.net/.default` | Entra token scope for PostgreSQL |
| `cognitiveCustomDomain` | `herald-ai-prod` | Cognitive Services subdomain (globally unique) |
| `cognitiveDeploymentName` | `gpt-5-mini` | Model deployment name |
| `cognitiveModelName` | `gpt-5-mini` | Model name |
| `cognitiveModelVersion` | `2025-08-07` | Model version |
| `cognitiveDeploymentSku` | `GlobalStandard` | Model deployment SKU (`GlobalStandard`, `DataZoneStandard`, `DataZoneProvisionedManaged`, `GlobalProvisionedManaged`) |
| `cognitiveTokenScope` | `https://cognitiveservices.azure.com/.default` | Cognitive Services Entra token scope (override for Azure Government) |
| `containerCpu` | `1.0` | CPU cores |
| `containerMemory` | `2Gi` | Memory |
| `minReplicas` | `1` | Minimum replicas |
| `maxReplicas` | `3` | Maximum replicas |
| `useAcr` | `false` | Deploy ACR for IL6 (see [IL6 Deployment](#il6-deployment)) |
| `ghcrUsername` | — | GitHub username for GHCR pull |
| `ghcrPassword` | — | GitHub PAT (**secure**, supply at deploy time) |
| `authEnabled` | `false` | Enable Entra authentication |
| `tenantId` | — | Entra tenant ID (when auth enabled) |
| `entraClientId` | — | Entra app registration client ID (when auth enabled) |
| `tags` | `{}` | Resource tags |

## GHCR Authentication

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

## Commercial Deployment (GHCR)

### 1. Validate Templates

```bash
az bicep build -f deploy/main.bicep
```

### 2. Create Resource Group

```bash
az group create \
  --name HeraldResourceGroup \
  --location eastus
```

### 3. Deploy

```bash
az deployment group create \
  --resource-group HeraldResourceGroup \
  --template-file deploy/main.bicep \
  --parameters deploy/main.parameters.json \
  --parameters \
    postgresAdminPassword='<password>' \
    ghcrUsername='<github-username>' \
    ghcrPassword='<ghcr-pat>' \ # use ="$(gh auth token)" to source from gh CLI
    authEnabled=true \
    tenantId='<entra-tenant-id>' \
    entraClientId='<entra-client-id>'
```

> If you encounter an "InsufficientQuota" message, try `location='eastus2'` or another region with quota availability.
>
> `{"code": "InsufficientQuota", "message": "This operation require 10 new capacity in quota One Thousand Tokens Per Minute - gpt-5-mini - GlobalStandard, which is bigger than the current available capacity 0. The current quota usage is 1000 and the quota limit is 1000 for quota One Thousand Tokens Per Minute - gpt-5-mini - GlobalStandard."}`

### 4. Run Migrations

```bash
az containerapp job start \
  --name herald-migrate \
  --resource-group HeraldResourceGroup
```

The command returns an execution name (e.g., `herald-migrate-u49u1qp`). Check the status:

```bash
az containerapp job execution show \
  --name herald-migrate \
  --resource-group HeraldResourceGroup \
  --job-execution-name <execution-name> \
  --output table
```

If the status is `Failed`, inspect the logs:

```bash
az containerapp job logs show \
  --name herald-migrate \
  --resource-group HeraldResourceGroup \
  --execution <execution-name> \
  --container herald-migrate
```

To run a different migrate command (e.g., force a version after a dirty state), override the command:

```bash
az containerapp job start \
  --name herald-migrate \
  --resource-group HeraldResourceGroup \
  --command "/usr/local/bin/migrate -force <version>"
```

Available flags: `-up`, `-down`, `-force <version>`, `-steps <n>`, `-version`.

### 5. Verify

```bash
# Get the app URL
az deployment group show \
  --resource-group HeraldResourceGroup \
  --name herald \
  --query 'properties.outputs.appUrl.value' \
  --output tsv

# Test health
curl -s https://<app-fqdn>/healthz
```

### 6. Post-Deploy: Entra Redirect URI

If authentication is enabled, add the Container App's auto-generated FQDN to the Entra app registration's SPA redirect URIs:

1. Get the FQDN from the deployment output
2. In Azure Portal → App registrations → Herald → Authentication
3. Add `https://<app-fqdn>/app/` as a SPA redirect URI

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

The `s2va/herald` proxy repo on GitHub Enterprise handles cross-domain transfers. On each tag push, its workflow:

1. Checks out `JaimeStill/herald` at the tagged version for the `deploy/` directory
2. Pulls the GHCR image and saves it as a tarball
3. Bundles the image tarball and `deploy/` manifests into a single `herald-<tag>.tar.gz`
4. Uploads the bundle to CDS blob storage via Portage and requests transfer via `s2va/cds-manifest`

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
  --resource-group HeraldResourceGroup \
  --template-file deploy/main.bicep \
  --parameters deploy/main.parameters.json \
  --parameters \
    postgresAdminPassword='<password>' \
    useAcr=true \
    containerImage='<acr-name>.azurecr.us/herald:<tag>' \
    postgresTokenScope='https://ossrdbms-aad.database.usgovcloudapi.net/.default' \
    cognitiveTokenScope='https://cognitiveservices.usgovcloudapi.net/.default' \
    authEnabled=true \
    tenantId='<entra-tenant-id>' \
    entraClientId='<entra-client-id>'
```

When `useAcr=true`:
- `registry.bicep` deploys an ACR in Herald's resource group
- AcrPull role is assigned to the managed identity
- Container App pulls via managed identity (no registry passwords)

Then run migrations and verify as in the commercial flow.

## Entra Configuration

Entra authentication is opt-in via the `authEnabled` parameter. When enabled, the Container App receives `HERALD_AUTH_MODE=azure` and the Entra tenant/client IDs as environment variables. When disabled (default), `HERALD_AUTH_MODE=none` preserves the unauthenticated experience.

### Prerequisites

An Entra app registration must exist before deployment. See the [Entra section in the root README](../README.md#entra) for app registration setup steps. The same registration works for both local development and production — just add the production redirect URI after deployment.

### What Bicep Configures

When `authEnabled=true`, the following environment variables are injected into the Container App:

| Variable | Value | Purpose |
|----------|-------|---------|
| `HERALD_AUTH_MODE` | `azure` | Enables JWT middleware |
| `HERALD_AUTH_MANAGED_IDENTITY` | `true` | Uses managed identity for token acquisition |
| `HERALD_AUTH_TENANT_ID` | `tenantId` param | Entra tenant for OIDC discovery |
| `HERALD_AUTH_CLIENT_ID` | `entraClientId` param | App registration audience for token validation |
| `HERALD_AUTH_AGENT_SCOPE` | `cognitiveTokenScope` param | Scope for AI Foundry access tokens |

### What Bicep Cannot Configure

- **App registration** — create via Azure Portal or `az ad app create` (the Microsoft.Graph Bicep extension is preview-only and not available on IL6)
- **Redirect URIs** — the Container App FQDN is auto-generated; add it to the app registration's SPA platform after deployment
- **Admin consent** — must be granted manually in the portal

## Updating a Deployment

Deployments are idempotent. To update the container image or change parameters:

```bash
az deployment group create \
  --resource-group HeraldResourceGroup \
  --template-file deploy/main.bicep \
  --parameters deploy/main.parameters.json \
  --parameters \
    postgresAdminPassword='<password>' \
    ghcrUsername='<github-username>' \
    ghcrPassword='<ghcr-pat>' \
    containerImage='ghcr.io/jaimestill/herald:<new-tag>'
```

If the new image includes schema changes, run migrations after the deployment completes.

## Teardown

```bash
az group delete --name HeraldResourceGroup --yes
```

Cognitive Services soft-delete may retain the account. Purge if needed:

```bash
az cognitiveservices account purge \
  --resource-group HeraldResourceGroup \
  --name <cognitive-account-name> \
  --location <region>
```

## Diagnostics

### Container App

**Check app status and FQDN:**

```bash
az containerapp show \
  --name herald \
  --resource-group HeraldResourceGroup \
  --query "{state: properties.runningStatus, fqdn: properties.configuration.ingress.fqdn}" \
  --output json
```

**View app logs (last 50 lines):**

```bash
az containerapp logs show \
  --name herald \
  --resource-group HeraldResourceGroup \
  --tail 50
```

**Follow app logs in real time:**

```bash
az containerapp logs show \
  --name herald \
  --resource-group HeraldResourceGroup \
  --follow
```

**Inspect environment variables:**

```bash
az containerapp show \
  --name herald \
  --resource-group HeraldResourceGroup \
  --query "properties.template.containers[0].env" \
  --output json
```

**List revision history:**

```bash
az containerapp revision list \
  --name herald \
  --resource-group HeraldResourceGroup \
  --output table
```

### Migration Job

**List all executions:**

```bash
az containerapp job execution list \
  --name herald-migrate \
  --resource-group HeraldResourceGroup \
  --output table
```

**Check execution status:**

```bash
az containerapp job execution show \
  --name herald-migrate \
  --resource-group HeraldResourceGroup \
  --job-execution-name <execution-name> \
  --output table
```

**View execution logs:**

```bash
az containerapp job logs show \
  --name herald-migrate \
  --resource-group HeraldResourceGroup \
  --execution <execution-name> \
  --container herald-migrate
```

### PostgreSQL

**Check server state:**

```bash
az postgres flexible-server show \
  --resource-group HeraldResourceGroup \
  --name herald-db \
  --query "{state: state, fqdn: fullyQualifiedDomainName, version: version}" \
  --output json
```

**Connect via psql** (requires firewall rule for your IP):

```bash
# Add temporary firewall rule
MY_IP=$(curl -s ifconfig.me)
az postgres flexible-server firewall-rule create \
  --resource-group HeraldResourceGroup \
  --name herald-db \
  --rule-name TempAccess \
  --start-ip-address $MY_IP \
  --end-ip-address $MY_IP

# Connect
PGPASSWORD='<password>' psql \
  "host=herald-db.postgres.database.azure.com port=5432 dbname=herald user=<admin-login> sslmode=require"

# Remove firewall rule when done
az postgres flexible-server firewall-rule delete \
  --resource-group HeraldResourceGroup \
  --name herald-db \
  --rule-name TempAccess \
  --yes
```

### Cognitive Services

**Check account and model deployment:**

```bash
az cognitiveservices account show \
  --resource-group HeraldResourceGroup \
  --name herald-ai \
  --query "{endpoint: properties.endpoint, state: properties.provisioningState}" \
  --output json

az cognitiveservices account deployment list \
  --resource-group HeraldResourceGroup \
  --name herald-ai \
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
  --resource-group HeraldResourceGroup \
  --name main \
  --query "properties.outputs" \
  --output json
```

## Troubleshooting

### PostgreSQL Authentication Fails with Managed Identity

**Symptom:** `failed SASL auth: FATAL: password authentication failed for user '<uuid>'`

The `HERALD_DB_USER` must be the Entra admin **principal name** (e.g., `herald-identity`), not the managed identity client ID (UUID). The Bicep template sets `entraAdminPrincipalName: '${prefix}-identity'` — `HERALD_DB_USER` must match this value. The managed identity client ID is used for `AZURE_CLIENT_ID` and `HERALD_AGENT_CLIENT_ID`, not the database username.

### Agent Vision Calls Return 404

**Symptom:** `HTTP 404: Resource not found` on classify workflow

The `HERALD_AGENT_BASE_URL` must include the `/openai` path segment for OpenAI-kind Cognitive Services accounts. The cognitive services endpoint output is `https://<subdomain>.openai.azure.com/` but the Azure OpenAI REST API expects `https://<subdomain>.openai.azure.com/openai/deployments/{deployment}/chat/completions`. The Bicep template appends `/openai` to the endpoint output.

> **Note:** AIServices-kind accounts do not require the `/openai` segment. This only applies to OpenAI-kind accounts.

### Cognitive Services `CustomDomainInUse`

**Symptom:** Redeployment fails with `CustomDomainInUse` error

Cognitive Services uses soft-delete — deleted accounts retain their subdomain for a recovery period. Purge the soft-deleted account before redeploying:

```bash
az cognitiveservices account list-deleted --output table

az cognitiveservices account purge \
  --resource-group <original-resource-group> \
  --name <account-name> \
  --location <region>
```

### Regional Quota and Availability

PostgreSQL Burstable tier and Cognitive Services model quotas vary by subscription type and region. Visual Studio Professional subscriptions have restrictions in some regions. If provisioning fails with `LocationIsOfferRestricted` or `InsufficientQuota`, try a different region or SKU. The `cognitiveDeploymentSku` parameter accepts `GlobalStandard`, `DataZoneStandard`, `DataZoneProvisionedManaged`, or `GlobalProvisionedManaged`.

### Dirty Migration State

**Symptom:** Migration job fails with `dirty database version N`

A previously failed migration leaves `schema_migrations` in a dirty state. Connect via psql (see [Diagnostics > PostgreSQL](#postgresql)) and reset:

```sql
-- Check current state
SELECT * FROM schema_migrations;

-- Reset dirty flag
UPDATE schema_migrations SET dirty = false WHERE version = <N>;
```

Then re-run the migration job. If the schema is in an inconsistent state, you may need to manually fix it or drop `schema_migrations` and re-run all migrations from scratch.

### Rollback

Container Apps maintains a revision history. To roll back to a previous revision:

```bash
# List revisions
az containerapp revision list \
  --name herald \
  --resource-group HeraldResourceGroup \
  --output table

# Activate a previous revision
az containerapp revision activate \
  --name herald \
  --resource-group HeraldResourceGroup \
  --revision <revision-name>

# Route all traffic to the previous revision
az containerapp ingress traffic set \
  --name herald \
  --resource-group HeraldResourceGroup \
  --revision-weight <revision-name>=100
```

To roll back by redeploying with a previous image tag, re-run the Bicep deployment with `containerImage` set to the previous tag.

## Environment Variables Reference

All `HERALD_*` environment variables injected into the Container App are composed in `main.bicep` from module outputs. They map directly to the constants defined in `internal/config/config.go`. See that file for the authoritative list of supported variables and their defaults.
