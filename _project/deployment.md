# Deployment Guide

Herald deploys as a single Azure Container App with managed identity connecting to PostgreSQL, Blob Storage, and AI Foundry. Infrastructure is defined as modular Bicep templates in the `deploy/` directory.

## Prerequisites

- [Azure CLI](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli) with Bicep (`az bicep install`)
- An Azure subscription with permission to create resources
- A GHCR personal access token with `read:packages` scope (commercial deployments)
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

A Container Apps Job configured for manual trigger. Uses the same container image as the app but overrides the command to `/usr/local/bin/migrate -up`. The database DSN is stored as a Container Apps secret. Migrations are idempotent — safe to run on every deployment.

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
    ghcrPassword='<ghcr-pat>'
```

To enable authentication, add:

```bash
    authEnabled=true \
    tenantId='<entra-tenant-id>' \
    entraClientId='<entra-client-id>'
```

### 4. Run Migrations

```bash
az containerapp job start \
  --name herald-migrate \
  --resource-group HeraldResourceGroup
```

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

## IL6 Deployment

IL6 environments have no access to GHCR. Herald uses Azure Container Registry with managed identity pull instead.

### Transfer via CDS

A GitHub Enterprise proxy repo connected to the cross-domain solution handles the transfer:

1. GHE workflow pulls the GHCR image and `deploy/` directory
2. Bundles everything into a `.tar` uploaded to CDS blob storage
3. IL6 side retrieves the `.tar`

### IL6 Side

**Import the image to ACR:**

```bash
az acr login -n <acr-name>
docker load -i herald-<tag>.tar
docker tag ghcr.io/jaimestill/herald:<tag> <acr-name>.azurecr.us/herald:<tag>
docker push <acr-name>.azurecr.us/herald:<tag>
```

**Deploy:**

```bash
az deployment group create \
  --resource-group HeraldResourceGroup \
  --template-file deploy/main.bicep \
  --parameters deploy/main.parameters.json \
  --parameters \
    postgresAdminPassword='<password>' \
    useAcr=true \
    containerImage='<acr-name>.azurecr.us/herald:<tag>'
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
| `HERALD_AUTH_AGENT_SCOPE` | `https://cognitiveservices.azure.com/.default` | Scope for AI Foundry access tokens |

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
  --name herald-ai \
  --location eastus
```

## Environment Variables Reference

All `HERALD_*` environment variables injected into the Container App are composed in `main.bicep` from module outputs. They map directly to the constants defined in `internal/config/config.go`. See that file for the authoritative list of supported variables and their defaults.
