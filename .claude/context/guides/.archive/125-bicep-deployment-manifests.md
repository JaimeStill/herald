# 125 - Add Bicep Deployment Manifests for Container Apps Infrastructure

## Problem Context

Herald needs declarative IaC to deploy its full Azure infrastructure: Container Apps, PostgreSQL, Blob Storage, Cognitive Services (AI Foundry), managed identity with role assignments, and optionally Azure Container Registry for IL6 air-gapped environments. The existing `scripts/` directory (bash + az CLI) is preserved as an alternative — this adds Bicep as the primary IaC approach in a new `deploy/` directory.

## Architecture Approach

Modular Bicep with `main.bicep` orchestrating 10 self-contained modules. A `useAcr` boolean parameter toggles between GHCR (commercial) and ACR (IL6) registry modes. User-assigned managed identity avoids the chicken-and-egg problem with role assignments. The `deploy/` directory is fully self-contained for IL6 CDS bundling.

## Implementation

### Step 1: Create `deploy/modules/identity.bicep`

```bicep
@description('Name of the user-assigned managed identity')
param name string

@description('Azure region for the resource')
param location string

@description('Resource tags')
param tags object = {}

resource identity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: name
  location: location
  tags: tags
}

@description('Full resource ID of the managed identity')
output id string = identity.id

@description('Principal (object) ID — used for role assignments')
output principalId string = identity.properties.principalId

@description('Client ID — used for AZURE_CLIENT_ID and DB user')
output clientId string = identity.properties.clientId
```

### Step 2: Create `deploy/modules/logging.bicep`

```bicep
@description('Name of the Log Analytics workspace')
param name string

@description('Azure region for the resource')
param location string

@description('Number of days to retain logs')
param retentionInDays int = 30

@description('Resource tags')
param tags object = {}

resource workspace 'Microsoft.OperationalInsights/workspaces@2023-09-01' = {
  name: name
  location: location
  tags: tags
  properties: {
    sku: {
      name: 'PerGB2018'
    }
    retentionInDays: retentionInDays
  }
}

@description('Workspace customer ID (needed by Container App Environment)')
output customerId string = workspace.properties.customerId

@description('Workspace name (used by main.bicep to call listKeys() directly)')
output name string = workspace.name
```

### Step 3: Create `deploy/modules/postgres.bicep`

```bicep
@description('PostgreSQL Flexible Server name (globally unique)')
param name string

@description('Azure region for the resource')
param location string

@description('Administrator login name')
param administratorLogin string

@secure()
@description('Administrator login password')
param administratorPassword string

@description('Database name to create')
param databaseName string = 'herald'

@description('PostgreSQL SKU name')
param skuName string = 'Standard_B1ms'

@description('PostgreSQL SKU tier')
@allowed(['Burstable', 'GeneralPurpose', 'MemoryOptimized'])
param skuTier string = 'Burstable'

@description('Storage size in GB')
param storageSizeGB int = 32

@description('PostgreSQL major version')
@allowed(['14', '15', '16', '17'])
param version string = '16'

@description('Principal ID of the managed identity to configure as Entra admin')
param entraAdminPrincipalId string

@description('Display name for the Entra admin')
param entraAdminPrincipalName string

@description('Principal type for the Entra admin')
@allowed(['ServicePrincipal', 'User', 'Group'])
param entraAdminPrincipalType string = 'ServicePrincipal'

@description('Allow Azure services to access the server')
param allowAzureServices bool = true

@description('Resource tags')
param tags object = {}

resource server 'Microsoft.DBforPostgreSQL/flexibleServers@2024-08-01' = {
  name: name
  location: location
  tags: tags
  sku: {
    name: skuName
    tier: skuTier
  }
  properties: {
    version: version
    administratorLogin: administratorLogin
    administratorLoginPassword: administratorPassword
    authConfig: {
      activeDirectoryAuth: 'Enabled'
      passwordAuth: 'Enabled'
    }
    storage: {
      storageSizeGB: storageSizeGB
    }
    backup: {
      backupRetentionDays: 7
      geoRedundantBackup: 'Disabled'
    }
    highAvailability: {
      mode: 'Disabled'
    }
  }
}

resource database 'Microsoft.DBforPostgreSQL/flexibleServers/databases@2024-08-01' = {
  parent: server
  name: databaseName
  properties: {
    charset: 'UTF8'
    collation: 'en_US.utf8'
  }
}

resource entraAdmin 'Microsoft.DBforPostgreSQL/flexibleServers/administrators@2024-08-01' = {
  parent: server
  name: entraAdminPrincipalId
  properties: {
    principalName: entraAdminPrincipalName
    principalType: entraAdminPrincipalType
    tenantId: tenant().tenantId
  }
}

resource allowAzure 'Microsoft.DBforPostgreSQL/flexibleServers/firewallRules@2024-08-01' = if (allowAzureServices) {
  parent: server
  name: 'AllowAzureServices'
  properties: {
    startIpAddress: '0.0.0.0'
    endIpAddress: '0.0.0.0'
  }
}

@description('Fully qualified domain name of the PostgreSQL server')
output fqdn string = server.properties.fullyQualifiedDomainName

@description('Database name')
output databaseName string = database.name
```

### Step 4: Create `deploy/modules/storage.bicep`

```bicep
@description('Storage account name (globally unique, 3-24 lowercase alphanumeric)')
param name string

@description('Azure region for the resource')
param location string

@description('Blob container name for documents')
param containerName string = 'documents'

@description('Storage account SKU')
@allowed(['Standard_LRS', 'Standard_GRS', 'Standard_ZRS'])
param skuName string = 'Standard_LRS'

@description('Resource tags')
param tags object = {}

resource storageAccount 'Microsoft.Storage/storageAccounts@2023-05-01' = {
  name: name
  location: location
  tags: tags
  kind: 'StorageV2'
  sku: {
    name: skuName
  }
  properties: {
    accessTier: 'Hot'
    allowBlobPublicAccess: false
    minimumTlsVersion: 'TLS1_2'
    supportsHttpsTrafficOnly: true
  }
}

resource blobService 'Microsoft.Storage/storageAccounts/blobServices@2023-05-01' = {
  parent: storageAccount
  name: 'default'
}

resource container 'Microsoft.Storage/storageAccounts/blobServices/containers@2023-05-01' = {
  parent: blobService
  name: containerName
  properties: {
    publicAccess: 'None'
  }
}

@description('Storage account resource ID (for role assignment scope)')
output id string = storageAccount.id

@description('Storage account name')
output storageAccountName string = storageAccount.name

@description('Primary blob endpoint URL')
output blobEndpoint string = storageAccount.properties.primaryEndpoints.blob
```

### Step 5: Create `deploy/modules/cognitive.bicep`

```bicep
@description('Cognitive Services account name')
param name string

@description('Azure region for the resource')
param location string

@description('Custom subdomain name for the account endpoint')
param customSubDomainName string

@description('Cognitive Services SKU')
param skuName string = 'S0'

@description('Cognitive Services kind')
param kind string = 'OpenAI'

@description('Model deployment name')
param deploymentName string = 'gpt-5-mini'

@description('Model name')
param modelName string = 'gpt-5-mini'

@description('Model version')
param modelVersion string = '2025-08-07'

@description('Model format')
param modelFormat string = 'OpenAI'

@description('Deployment SKU name')
param deploymentSkuName string = 'GlobalStandard'

@description('Deployment SKU capacity (TPM in thousands)')
param deploymentSkuCapacity int = 10

@description('Resource tags')
param tags object = {}

resource account 'Microsoft.CognitiveServices/accounts@2024-10-01' = {
  name: name
  location: location
  tags: tags
  kind: kind
  sku: {
    name: skuName
  }
  properties: {
    customSubDomainName: customSubDomainName
    publicNetworkAccess: 'Enabled'
  }
}

resource deployment 'Microsoft.CognitiveServices/accounts/deployments@2024-10-01' = {
  parent: account
  name: deploymentName
  sku: {
    name: deploymentSkuName
    capacity: deploymentSkuCapacity
  }
  properties: {
    model: {
      format: modelFormat
      name: modelName
      version: modelVersion
    }
  }
}

@description('Cognitive Services account resource ID (for role assignment scope)')
output id string = account.id

@description('Cognitive Services endpoint URL')
output endpoint string = account.properties.endpoint

@description('Model deployment name')
output modelDeploymentName string = deployment.name
```

### Step 6: Create `deploy/modules/registry.bicep`

```bicep
@description('Azure Container Registry name (globally unique, 5-50 alphanumeric)')
param name string

@description('Azure region for the resource')
param location string

@description('ACR SKU')
@allowed(['Basic', 'Standard', 'Premium'])
param skuName string = 'Basic'

@description('Resource tags')
param tags object = {}

resource registry 'Microsoft.ContainerRegistry/registries@2023-07-01' = {
  name: name
  location: location
  tags: tags
  sku: {
    name: skuName
  }
  properties: {
    adminUserEnabled: false
  }
}

@description('ACR resource ID (for role assignment scope)')
output id string = registry.id

@description('ACR login server (e.g., myregistry.azurecr.io)')
output loginServer string = registry.properties.loginServer
```

### Step 7: Create `deploy/modules/environment.bicep`

```bicep
@description('Container App Environment name')
param name string

@description('Azure region for the resource')
param location string

@description('Log Analytics workspace name (used to resolve listKeys internally)')
param logAnalyticsWorkspaceName string

@description('Log Analytics workspace customer ID')
param logAnalyticsCustomerId string

@description('Resource tags')
param tags object = {}

// Resolve the workspace here so listKeys() works —
// an existing resource referenced via module output name
// cannot call listKeys() from the parent template (BCP307)
resource workspace 'Microsoft.OperationalInsights/workspaces@2023-09-01' existing = {
  name: logAnalyticsWorkspaceName
}

resource environment 'Microsoft.App/managedEnvironments@2024-03-01' = {
  name: name
  location: location
  tags: tags
  properties: {
    appLogsConfiguration: {
      destination: 'log-analytics'
      logAnalyticsConfiguration: {
        customerId: logAnalyticsCustomerId
        sharedKey: workspace.listKeys().primarySharedKey
      }
    }
  }
}

@description('Container App Environment resource ID')
output id string = environment.id
```

### Step 8: Create `deploy/modules/roles.bicep`

```bicep
@description('Principal ID of the managed identity to assign roles to')
param principalId string

@description('Storage account resource ID (scope for Storage Blob Data Contributor)')
param storageAccountId string

@description('Cognitive Services account resource ID (scope for OpenAI User)')
param cognitiveAccountId string

@description('Whether to assign AcrPull role')
param assignAcrPull bool = false

@description('ACR resource ID (scope for AcrPull, required when assignAcrPull is true)')
param acrId string = ''

// Well-known role definition IDs
var storageBlobDataContributor = 'ba92f5b4-2d11-453d-a403-e96b0029c9fe'
var cognitiveServicesOpenAIUser = '5e0bd9bd-7b93-4f28-af87-19fc36ad61bd'
var acrPull = '7f951dda-4ed3-4680-a7ca-43fe172d538d'

resource storageBlobRole 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(storageAccountId, principalId, storageBlobDataContributor)
  scope: storageAccount
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', storageBlobDataContributor)
    principalId: principalId
    principalType: 'ServicePrincipal'
  }
}

resource storageAccount 'Microsoft.Storage/storageAccounts@2023-05-01' existing = {
  name: last(split(storageAccountId, '/'))
}

resource cognitiveRole 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(cognitiveAccountId, principalId, cognitiveServicesOpenAIUser)
  scope: cognitiveAccount
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', cognitiveServicesOpenAIUser)
    principalId: principalId
    principalType: 'ServicePrincipal'
  }
}

resource cognitiveAccount 'Microsoft.CognitiveServices/accounts@2024-10-01' existing = {
  name: last(split(cognitiveAccountId, '/'))
}

resource acrRole 'Microsoft.Authorization/roleAssignments@2022-04-01' = if (assignAcrPull && acrId != '') {
  name: guid(acrId, principalId, acrPull)
  scope: containerRegistry
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', acrPull)
    principalId: principalId
    principalType: 'ServicePrincipal'
  }
}

resource containerRegistry 'Microsoft.ContainerRegistry/registries@2023-07-01' existing = if (assignAcrPull && acrId != '') {
  name: last(split(acrId, '/'))
}
```

### Step 9: Create `deploy/modules/app.bicep`

```bicep
@description('Container App name')
param name string

@description('Azure region for the resource')
param location string

@description('Container App Environment resource ID')
param environmentId string

@description('User-assigned managed identity resource ID')
param identityId string

@description('Container image reference (GHCR or ACR)')
param containerImage string

@description('Registry configuration array')
param registries array

@description('Container App secrets array')
param secrets array = []

@description('Environment variables for the container')
param envVars array

@description('CPU cores allocated to the container')
param cpu string = '1.0'

@description('Memory allocated to the container')
param memory string = '2Gi'

@description('Minimum number of replicas')
param minReplicas int = 1

@description('Maximum number of replicas')
param maxReplicas int = 3

@description('Target port for ingress')
param targetPort int = 8080

@description('Resource tags')
param tags object = {}

resource app 'Microsoft.App/containerApps@2024-03-01' = {
  name: name
  location: location
  tags: tags
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${identityId}': {}
    }
  }
  properties: {
    managedEnvironmentId: environmentId
    configuration: {
      ingress: {
        external: true
        targetPort: targetPort
        transport: 'http'
      }
      registries: registries
      secrets: secrets
    }
    template: {
      containers: [
        {
          name: name
          image: containerImage
          resources: {
            cpu: json(cpu)
            memory: memory
          }
          env: envVars
          probes: [
            {
              type: 'Liveness'
              httpGet: {
                path: '/healthz'
                port: targetPort
              }
              initialDelaySeconds: 5
              periodSeconds: 10
            }
            {
              type: 'Readiness'
              httpGet: {
                path: '/readyz'
                port: targetPort
              }
              initialDelaySeconds: 5
              periodSeconds: 10
            }
          ]
        }
      ]
      scale: {
        minReplicas: minReplicas
        maxReplicas: maxReplicas
      }
    }
  }
}

@description('Container App FQDN')
output fqdn string = app.properties.configuration.ingress.fqdn
```

### Step 10: Create `deploy/modules/migration-job.bicep`

```bicep
@description('Container Apps Job name')
param name string

@description('Azure region for the resource')
param location string

@description('Container App Environment resource ID')
param environmentId string

@description('User-assigned managed identity resource ID')
param identityId string

@description('Container image reference (same image as the app)')
param containerImage string

@description('Registry configuration array')
param registries array

@secure()
@description('Full PostgreSQL DSN for migrations')
param databaseDsn string

@description('CPU cores allocated to the job container')
param cpu string = '0.5'

@description('Memory allocated to the job container')
param memory string = '1Gi'

@description('Resource tags')
param tags object = {}

resource job 'Microsoft.App/jobs@2024-03-01' = {
  name: name
  location: location
  tags: tags
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${identityId}': {}
    }
  }
  properties: {
    environmentId: environmentId
    configuration: {
      triggerType: 'Manual'
      replicaTimeout: 300
      replicaRetryLimit: 0
      registries: registries
      secrets: [
        {
          name: 'db-dsn'
          value: databaseDsn
        }
      ]
    }
    template: {
      containers: [
        {
          name: name
          image: containerImage
          command: [
            '/usr/local/bin/migrate'
            '-up'
          ]
          resources: {
            cpu: json(cpu)
            memory: memory
          }
          env: [
            {
              name: 'HERALD_DB_DSN'
              secretRef: 'db-dsn'
            }
          ]
        }
      ]
    }
  }
}
```

### Step 11: Create `deploy/main.bicep`

```bicep
targetScope = 'resourceGroup'

// ============================================================================
// Parameters
// ============================================================================

@description('Azure region for all resources')
param location string

@description('Naming prefix for all resources (e.g., herald)')
param prefix string = 'herald'

@description('Resource tags applied to all resources')
param tags object = {}

// --- PostgreSQL ---

@description('PostgreSQL administrator login name')
param postgresAdminLogin string

@secure()
@description('PostgreSQL administrator password (supply at deploy time, never in param files)')
param postgresAdminPassword string

@description('PostgreSQL SKU name')
param postgresSkuName string = 'Standard_B1ms'

@description('PostgreSQL SKU tier')
@allowed(['Burstable', 'GeneralPurpose', 'MemoryOptimized'])
param postgresSkuTier string = 'Burstable'

@description('PostgreSQL storage size in GB')
param postgresStorageSizeGB int = 32

@description('PostgreSQL Entra token scope (same across cloud instances)')
#disable-next-line no-hardcoded-env-urls
param postgresTokenScope string = 'https://ossrdbms-aad.database.windows.net/.default'

// --- Cognitive Services ---

@description('Cognitive Services custom subdomain (globally unique)')
param cognitiveCustomDomain string = 'herald-ai-prod'

@description('AI model deployment name')
param cognitiveDeploymentName string = 'gpt-5-mini'

@description('AI model name')
param cognitiveModelName string = 'gpt-5-mini'

@description('AI model version')
param cognitiveModelVersion string = '2025-08-07'

// --- Container App ---

@description('Container image reference (e.g., ghcr.io/jaimestill/herald:v0.4.0)')
param containerImage string

@description('CPU cores for the Container App')
param containerCpu string = '1.0'

@description('Memory for the Container App (ImageMagick workloads need headroom)')
param containerMemory string = '2Gi'

@description('Minimum replica count')
param minReplicas int = 1

@description('Maximum replica count')
param maxReplicas int = 3

// --- Registry ---

@description('Use Azure Container Registry instead of GHCR (for IL6 air-gapped environments)')
param useAcr bool = false

@description('GitHub username for GHCR authentication (ignored when useAcr is true)')
param ghcrUsername string = ''

@secure()
@description('GitHub PAT for GHCR authentication (ignored when useAcr is true)')
param ghcrPassword string = ''

// --- Auth ---

@description('Enable Azure Entra authentication for the web client')
param authEnabled bool = false

@description('Azure Entra tenant ID (required when authEnabled is true)')
param tenantId string = ''

@description('Entra app registration client ID (required when authEnabled is true)')
param entraClientId string = ''

// ============================================================================
// Modules
// ============================================================================

module identity 'modules/identity.bicep' = {
  name: '${prefix}-identity'
  params: {
    name: '${prefix}-identity'
    location: location
    tags: tags
  }
}

module logging 'modules/logging.bicep' = {
  name: '${prefix}-logging'
  params: {
    name: '${prefix}-logs'
    location: location
    tags: tags
  }
}

module postgres 'modules/postgres.bicep' = {
  name: '${prefix}-postgres'
  params: {
    name: '${prefix}-db'
    location: location
    administratorLogin: postgresAdminLogin
    administratorPassword: postgresAdminPassword
    skuName: postgresSkuName
    skuTier: postgresSkuTier
    storageSizeGB: postgresStorageSizeGB
    entraAdminPrincipalId: identity.outputs.principalId
    entraAdminPrincipalName: '${prefix}-identity'
    tags: tags
  }
}

module storage 'modules/storage.bicep' = {
  name: '${prefix}-storage'
  params: {
    name: replace('${prefix}storage', '-', '')
    location: location
    tags: tags
  }
}

module cognitive 'modules/cognitive.bicep' = {
  name: '${prefix}-cognitive'
  params: {
    name: '${prefix}-ai'
    location: location
    customSubDomainName: cognitiveCustomDomain
    deploymentName: cognitiveDeploymentName
    modelName: cognitiveModelName
    modelVersion: cognitiveModelVersion
    tags: tags
  }
}

module registry 'modules/registry.bicep' = if (useAcr) {
  name: '${prefix}-registry'
  params: {
    name: replace('${prefix}registry', '-', '')
    location: location
    tags: tags
  }
}

// Safe-dereference conditional module outputs to avoid BCP318 warnings
var acrId = registry.?outputs.?id ?? ''
var acrLoginServer = registry.?outputs.?loginServer ?? ''

module environment 'modules/environment.bicep' = {
  name: '${prefix}-environment'
  params: {
    name: '${prefix}-env'
    location: location
    logAnalyticsWorkspaceName: logging.outputs.name
    logAnalyticsCustomerId: logging.outputs.customerId
    tags: tags
  }
}

module roles 'modules/roles.bicep' = {
  name: '${prefix}-roles'
  params: {
    principalId: identity.outputs.principalId
    storageAccountId: storage.outputs.id
    cognitiveAccountId: cognitive.outputs.id
    assignAcrPull: useAcr
    acrId: acrId
  }
}

// ============================================================================
// Registry Configuration
// ============================================================================

// GHCR: password-based auth via secrets
// ACR:  managed identity pull (no passwords)
var ghcrRegistries = [
  {
    server: 'ghcr.io'
    username: ghcrUsername
    passwordSecretRef: 'ghcr-password'
  }
]

var acrRegistries = [
  {
    server: acrLoginServer
    identity: identity.outputs.id
  }
]

var registries = useAcr ? acrRegistries : ghcrRegistries

var ghcrSecrets = [
  {
    name: 'ghcr-password'
    value: ghcrPassword
  }
]

// ============================================================================
// Environment Variables
// ============================================================================

var baseEnvVars = [
  { name: 'HERALD_ENV', value: 'azure' }
  { name: 'HERALD_SERVER_PORT', value: '8080' }
  { name: 'HERALD_DB_HOST', value: postgres.outputs.fqdn }
  { name: 'HERALD_DB_PORT', value: '5432' }
  { name: 'HERALD_DB_NAME', value: postgres.outputs.databaseName }
  { name: 'HERALD_DB_USER', value: identity.outputs.clientId }
  { name: 'HERALD_DB_SSL_MODE', value: 'require' }
  { name: 'HERALD_DB_TOKEN_SCOPE', value: postgresTokenScope }
  { name: 'HERALD_STORAGE_SERVICE_URL', value: storage.outputs.blobEndpoint }
  { name: 'HERALD_STORAGE_CONTAINER_NAME', value: 'documents' }
  { name: 'HERALD_AUTH_MODE', value: authEnabled ? 'azure' : 'none' }
  { name: 'HERALD_AUTH_MANAGED_IDENTITY', value: 'true' }
  { name: 'HERALD_AUTH_AGENT_SCOPE', value: 'https://cognitiveservices.azure.com/.default' }
  { name: 'HERALD_AGENT_PROVIDER_NAME', value: 'azure' }
  { name: 'HERALD_AGENT_BASE_URL', value: cognitive.outputs.endpoint }
  { name: 'HERALD_AGENT_DEPLOYMENT', value: cognitive.outputs.modelDeploymentName }
  { name: 'HERALD_AGENT_API_VERSION', value: '2025-04-01-preview' }
  { name: 'HERALD_AGENT_AUTH_TYPE', value: 'managed_identity' }
  { name: 'AZURE_CLIENT_ID', value: identity.outputs.clientId }
]

var authEnvVars = authEnabled ? [
  { name: 'HERALD_AUTH_TENANT_ID', value: tenantId }
  { name: 'HERALD_AUTH_CLIENT_ID', value: entraClientId }
] : []

var envVars = concat(baseEnvVars, authEnvVars)

// ============================================================================
// Container App
// ============================================================================

module app 'modules/app.bicep' = {
  name: '${prefix}-app'
  dependsOn: [roles]
  params: {
    name: prefix
    location: location
    environmentId: environment.outputs.id
    identityId: identity.outputs.id
    containerImage: containerImage
    registries: registries
    secrets: useAcr ? [] : ghcrSecrets
    envVars: envVars
    cpu: containerCpu
    memory: containerMemory
    minReplicas: minReplicas
    maxReplicas: maxReplicas
    tags: tags
  }
}

// ============================================================================
// Migration Job
// ============================================================================

var migrationDsn = 'postgres://${postgresAdminLogin}:${postgresAdminPassword}@${postgres.outputs.fqdn}:5432/${postgres.outputs.databaseName}?sslmode=require'

module migrationJob 'modules/migration-job.bicep' = {
  name: '${prefix}-migration-job'
  dependsOn: [roles]
  params: {
    name: '${prefix}-migrate'
    location: location
    environmentId: environment.outputs.id
    identityId: identity.outputs.id
    containerImage: containerImage
    registries: registries
    databaseDsn: migrationDsn
    tags: tags
  }
}

// ============================================================================
// Outputs
// ============================================================================

@description('Container App public URL')
output appUrl string = 'https://${app.outputs.fqdn}'

@description('PostgreSQL server FQDN')
output postgresHost string = postgres.outputs.fqdn

@description('Storage account blob endpoint')
output storageBlobEndpoint string = storage.outputs.blobEndpoint

@description('Cognitive Services endpoint')
output cognitiveEndpoint string = cognitive.outputs.endpoint

@description('ACR login server (empty when using GHCR)')
output acrLoginServer string = acrLoginServer
```

### Step 12: Create `deploy/main.parameters.json`

```json
{
  "$schema": "https://schema.management.azure.com/schemas/2019-04-01/deploymentParameters.json#",
  "contentVersion": "0.4.0.0",
  "parameters": {
    "location": {
      "value": "eastus"
    },
    "prefix": {
      "value": "herald"
    },
    "containerImage": {
      "value": "ghcr.io/jaimestill/herald:v0.4.0"
    },
    "postgresAdminLogin": {
      "value": "heraldadmin"
    },
    "cognitiveCustomDomain": {
      "value": "herald-ai-prod"
    }
  }
}
```

Secrets are intentionally omitted — supply at deploy time:

```bash
az deployment group create \
  -g HeraldResourceGroup \
  -f deploy/main.bicep \
  -p deploy/main.parameters.json \
  -p postgresAdminPassword='<secret>' \
  -p ghcrPassword='<github-pat>'
```

### Step 13: Update `Dockerfile`

Add the `migrate` binary build to the build stage and copy to the final image.

**In the build stage**, add after the existing `go build` line:

```dockerfile
RUN CGO_ENABLED=0 go build -o /migrate ./cmd/migrate
```

**In the final stage**, add after the existing `COPY --from=build` line:

```dockerfile
COPY --from=build /migrate /usr/local/bin/migrate
```

## Validation Criteria

- [ ] `az bicep build -f deploy/main.bicep` compiles without errors
- [ ] All parameters documented with `@description` decorators
- [ ] No secrets in parameter files (sensitive values via @secure parameters)
- [ ] Dockerfile builds both `herald` and `migrate` binaries
- [ ] Migration Job uses same image with overridden command
- [ ] Existing `scripts/` directory untouched
- [ ] HERALD_* env vars match `internal/config/config.go` constants
- [ ] `useAcr=true` path deploys ACR + AcrPull role, uses managed identity pull
- [ ] `useAcr=false` path uses GHCR with PAT credentials (no ACR deployed)
