# 125 — Bicep Deployment Manifests

## Context

Herald needs declarative IaC for Azure Container Apps deployment. The existing `scripts/` directory handles Cognitive Services via bash + az CLI — this adds Bicep as the primary IaC approach in a new `deploy/` directory, covering **all** Azure infrastructure: Container Apps, PostgreSQL, Blob Storage, Cognitive Services, managed identity, and role assignments.

## Directory Structure

```
deploy/
├── main.bicep              # Orchestrator — parameters → module calls → outputs
├── main.bicepparam         # Parameter values (commercial cloud example)
└── modules/
    ├── identity.bicep      # User-Assigned Managed Identity
    ├── logging.bicep       # Log Analytics Workspace
    ├── postgres.bicep      # PostgreSQL Flexible Server + database + Entra admin + firewall
    ├── storage.bicep       # Storage Account + blob container
    ├── cognitive.bicep     # Cognitive Services (OpenAI) + model deployment
    ├── registry.bicep      # Azure Container Registry (optional — IL6 deployments)
    ├── environment.bicep   # Container App Environment
    ├── app.bicep           # Container App (Herald server)
    ├── migration-job.bicep # Container Apps Job (herald-migrate -up)
    └── roles.bicep         # Role assignments for managed identity
```

## Dependency Graph

```
identity ─────────────────────────────────┐
    │                                     │
logging    postgres   storage   cognitive   registry?
    │                    │          │          │
environment              │          │          │
    │                    │          │          │
    ├────────────────────┤──────────┤──────────┤
    v                    v          v          v
         roles (identity → storage + cognitive + registry?)
                │
        ┌───────┴───────┐
        v               v
       app         migration-job
```

## Module Design

### identity.bicep
- `Microsoft.ManagedIdentity/userAssignedIdentities`
- Outputs: `id`, `principalId`, `clientId`
- **Why user-assigned**: Breaks the chicken-and-egg problem — identity exists before the app, so roles can be assigned first

### logging.bicep
- `Microsoft.OperationalInsights/workspaces`
- Outputs: `customerId`, `sharedKey` (via `listKeys()`)

### postgres.bicep
- `Microsoft.DBforPostgreSQL/flexibleServers` — both password + Entra auth enabled
- `flexibleServers/databases` — the `herald` database
- `flexibleServers/firewallRules` — allow Azure services
- `flexibleServers/administrators` — managed identity as AAD admin
- Outputs: `fqdn`, `databaseName`
- **Dual auth**: Password for migration job (DSN format), Entra tokens for the app at runtime

### storage.bicep
- `Microsoft.Storage/storageAccounts` — `allowBlobPublicAccess: false`
- `storageAccounts/blobServices/containers` — `documents` container
- Outputs: `id`, `blobEndpoint`

### cognitive.bicep
- `Microsoft.CognitiveServices/accounts` (kind: OpenAI)
- `accounts/deployments` — model deployment (gpt-5-mini)
- Outputs: `id`, `endpoint`, `deploymentName`

### environment.bicep
- `Microsoft.App/managedEnvironments`
- Inputs: Log Analytics customer ID + shared key
- Outputs: `id`

### registry.bicep (conditional — IL6 deployments)
- `Microsoft.ContainerRegistry/registries` — Basic or Standard SKU
- Deployed only when `useAcr` parameter is true (IL6 environments without GHCR access)
- Outputs: `id`, `loginServer`
- On commercial cloud, GHCR is used directly (no ACR needed)

### roles.bicep
- Storage Blob Data Contributor (`ba92f5b4-2d11-453d-a403-e96b0029c9fe`) → storage account
- Cognitive Services OpenAI User (`5e0bd9bd-7b93-4f28-af87-19fc36ad61bd`) → cognitive account
- AcrPull (`7f951dda-4ed3-4680-a7ca-43fe172d538d`) → ACR (conditional, when `useAcr` is true)
- All in one file for security visibility

### app.bicep
- `Microsoft.App/containerApps` with user-assigned identity
- Ingress on port 8080 (external)
- Liveness probe: `GET /healthz`
- Readiness probe: `GET /readyz`
- Resource limits: 1.0 CPU, 2Gi memory (ImageMagick workloads)
- Scale: min/max replicas from parameters
- Registry config passed as parameter (supports both GHCR and ACR, see below)
- All `HERALD_*` env vars composed in `main.bicep` from module outputs

### migration-job.bicep
- `Microsoft.App/jobs` — manual trigger
- Command override: `['/usr/local/bin/herald-migrate', '-up']`
- `HERALD_DB_DSN` stored as Container Apps secret
- DSN composed in `main.bicep`: `postgres://<admin>:<pass>@<fqdn>:5432/<db>?sslmode=require`

## main.bicep Parameters

| Parameter | Type | Default | Notes |
|-----------|------|---------|-------|
| `location` | string | — | Azure region |
| `prefix` | string | `'herald'` | Naming prefix for all resources |
| `postgresAdminLogin` | string | — | DB admin username |
| `postgresAdminPassword` | @secure string | — | Supplied at deploy time |
| `containerImage` | string | — | Full image reference (GHCR or ACR) |
| `useAcr` | bool | `false` | `true` = deploy ACR + managed identity pull; `false` = GHCR with PAT |
| `ghcrUsername` | string | `''` | GitHub username (GHCR only, ignored when useAcr) |
| `ghcrPassword` | @secure string | `''` | GitHub PAT (GHCR only, ignored when useAcr) |
| `cognitiveCustomDomain` | string | `'herald-ai'` | Cognitive Services custom subdomain |
| `cognitiveModelName` | string | `'gpt-5-mini'` | Model name + deployment name |
| `cognitiveModelVersion` | string | `'2025-08-07'` | Model version |
| `authEnabled` | bool | `false` | Enable Entra auth for web client |
| `tenantId` | string | `''` | Entra tenant (when auth enabled) |
| `entraClientId` | string | `''` | App registration client ID |
| `containerCpu` | string | `'1.0'` | CPU cores |
| `containerMemory` | string | `'2Gi'` | Memory |
| `minReplicas` | int | `1` | Scale min |
| `maxReplicas` | int | `3` | Scale max |
| `tags` | object | `{}` | Resource tags |

## Registry Strategy (GHCR vs ACR)

Two registry modes controlled by the `useAcr` parameter:

**Commercial (`useAcr: false`, default)**:
- Container App pulls from `ghcr.io/jaimestill/herald:<tag>`
- GHCR credentials passed as `ghcrUsername` + `ghcrPassword` (secure)
- Container App's `registries` block uses `passwordSecretRef`
- No ACR module deployed

**IL6 (`useAcr: true`)**:
- `registry.bicep` deploys an Azure Container Registry
- AcrPull role assigned to the managed identity in `roles.bicep`
- Container App's `registries` block uses `identity` (managed identity pull, no passwords)
- `containerImage` points to the ACR login server: `<prefix>registry.azurecr.us/herald:<tag>`
- Image is pushed to ACR before deployment (see IL6 Deployment Bundle below)

`main.bicep` builds the appropriate registry configuration object and passes it to `app.bicep` and `migration-job.bicep`.

## Container App Environment Variables

Composed in `main.bicep` from module outputs. Maps to `internal/config/config.go` env var names:

```
HERALD_ENV=azure
HERALD_SERVER_PORT=8080
HERALD_DB_HOST=<postgres.fqdn>
HERALD_DB_NAME=herald
HERALD_DB_USER=<identity.clientId>
HERALD_DB_SSL_MODE=require
HERALD_DB_TOKEN_SCOPE=https://ossrdbms-aad.database.windows.net/.default
HERALD_STORAGE_SERVICE_URL=<storage.blobEndpoint>
HERALD_STORAGE_CONTAINER_NAME=documents
HERALD_AUTH_MODE=<azure|none>
HERALD_AUTH_MANAGED_IDENTITY=true
HERALD_AUTH_TENANT_ID=<tenantId>
HERALD_AUTH_CLIENT_ID=<entraClientId>
HERALD_AUTH_AGENT_SCOPE=https://cognitiveservices.azure.com/.default
HERALD_AGENT_PROVIDER_NAME=azure
HERALD_AGENT_BASE_URL=<cognitive.endpoint>
HERALD_AGENT_DEPLOYMENT=<cognitive.deploymentName>
HERALD_AGENT_API_VERSION=2025-04-01-preview
HERALD_AGENT_AUTH_TYPE=managed_identity
AZURE_CLIENT_ID=<identity.clientId>
```

`AZURE_CLIENT_ID` is the Azure SDK convention for user-assigned managed identity — tells `DefaultAzureCredential` which identity to use.

## Dockerfile Changes

Add `herald-migrate` binary build in the build stage and copy to the final image:

```dockerfile
# Build stage: add after herald build
RUN CGO_ENABLED=0 go build -o /herald-migrate ./cmd/migrate

# Final stage: add after herald COPY
COPY --from=build /herald-migrate /usr/local/bin/herald-migrate
```

## What Bicep Cannot Do

- **Entra App Registration** — `Microsoft.Graph` extension is preview-only. App registrations should be created via Azure portal or `az ad app` CLI. Bicep accepts `tenantId` and `entraClientId` as input parameters assuming the app already exists.
- **DNS / Custom domains** — separate concern
- **Image transfer to IL6** — Bicep deploys infrastructure; getting the image into an air-gapped ACR is an operational step (see bundle below)

## IL6 Deployment Transfer

IL6 environments have no access to GHCR. Transfer uses a **GitHub Enterprise proxy repo** connected to a cross-domain solution (CDS):

1. GHE proxy repo has a GitHub Actions workflow
2. Workflow pulls the GHCR image (`docker save`) + `deploy/` directory from this repo
3. Bundles everything into a `.tar` uploaded to CDS blob storage
4. IL6 side retrieves the `.tar`, pushes the image to ACR, deploys Bicep

**Design constraint**: The `deploy/` directory must be fully self-contained — no references to files outside `deploy/`, so bundling it with the image tarball is all that's needed.

The CDS workflow lives in the GHE proxy repo (out of scope for #125). The migrate CLI transfers automatically since both binaries are in the same container image.

**IL6 side after transfer:**
```bash
# Import image to ACR
az acr login -n <acr-name>
docker load -i herald-<tag>.tar
docker tag ghcr.io/jaimestill/herald:<tag> <acr-name>.azurecr.us/herald:<tag>
docker push <acr-name>.azurecr.us/herald:<tag>

# Deploy infrastructure
az deployment group create \
  -g HeraldResourceGroup \
  -f deploy/main.bicep \
  -p deploy/main.bicepparam \
  -p postgresAdminPassword='<secret>' \
  -p useAcr=true \
  -p containerImage='<acr-name>.azurecr.us/herald:<tag>'
```

## Deployment

**Commercial (GHCR):**
```bash
az deployment group create \
  -g HeraldResourceGroup \
  -f deploy/main.bicep \
  -p deploy/main.bicepparam \
  -p postgresAdminPassword='<secret>' \
  -p ghcrPassword='<github-pat>'
```

**IL6 (ACR):**
```bash
az deployment group create \
  -g HeraldResourceGroup \
  -f deploy/main.bicep \
  -p deploy/main.bicepparam \
  -p postgresAdminPassword='<secret>' \
  -p useAcr=true \
  -p containerImage='<acr>.azurecr.us/herald:<tag>'
```

**Post-deploy (both):**
```bash
# Run migrations
az containerapp job start -n herald-migrate -g HeraldResourceGroup

# Verify
curl https://<app-fqdn>/healthz
```

## Validation Criteria

- [ ] `az bicep build -f deploy/main.bicep` compiles without errors
- [ ] All parameters documented with `@description` decorators
- [ ] No secrets in parameter files (sensitive values via @secure parameters)
- [ ] Dockerfile builds both `herald` and `herald-migrate` binaries
- [ ] Migration Job uses same image with overridden command
- [ ] Existing `scripts/` directory untouched
- [ ] HERALD_* env vars match `internal/config/config.go` constants
- [ ] `useAcr=true` path deploys ACR + AcrPull role, uses managed identity pull (no registry passwords)
- [ ] `useAcr=false` path uses GHCR with PAT credentials (no ACR deployed)
