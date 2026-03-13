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

// -- PostgreSQL ---

@description('PostgreSQL administrator login name')
param postgresAdminLogin string

@secure()
@description('PostgreSQL administrator password (supply at deploy time, never in param files)')
param postgresAdminPassword string

@description('PostgreSQL SKU name')
param postgresSkuName string = 'Standard_B1ms'

@description('PostgreSQL SKU tier')
@allowed(['Burstable', 'GeneralPurpose', 'MemoryOptimized'])
param postgesSkuTier string = 'Burstable'

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

@description('AI model deployment SKU')
@allowed(['GlobalStandard', 'DataZoneStandard', 'DataZoneProvisionedManaged', 'GlobalProvisionedManaged'])
param cognitiveDeploymentSku string = 'GlobalStandard'

@description('Cognitive Services Entra token scope (override for Azure Government)')
#disable-next-line no-hardcoded-env-urls
param cognitiveTokenScope string = 'https://cognitiveservices.azure.com/.default'

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

// Deployment order: identity → logging → postgres → storage → cognitive
//                   → registry (conditional) → environment → roles → app / migrationJob
//
// Explicit dependsOn chain prevents ARM from parallelizing module deployments,
// which avoids race conditions where a resource reports "provisioned" before it
// is fully ready to accept child operations.

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
  dependsOn: [identity]
  params: {
    name: '${prefix}-logs'
    location: location
    tags: tags
  }
}

module postgres 'modules/postgres.bicep' = {
  name: '${prefix}-postgres'
  dependsOn: [logging]
  params: {
    name: '${prefix}-db'
    location: location
    administratorLogin: postgresAdminLogin
    administratorPassword: postgresAdminPassword
    skuName: postgresSkuName
    skuTier: postgesSkuTier
    storageSizeGB: postgresStorageSizeGB
    entraAdminPrincipalId: identity.outputs.principalId
    entraAdminPrincipalName: '${prefix}-identity'
    tags: tags
  }
}

module storage 'modules/storage.bicep' = {
  name: '${prefix}-storage'
  dependsOn: [postgres]
  params: {
    name: replace('${prefix}storage', '-', '')
    location: location
    tags: tags
  }
}

module cognitive 'modules/cognitive.bicep' = {
  name: '${prefix}-cognitive'
  dependsOn: [storage]
  params: {
    name: '${prefix}-ai'
    location: location
    customSubDomainName: cognitiveCustomDomain
    deploymentName: cognitiveDeploymentName
    modelName: cognitiveModelName
    modelVersion: cognitiveModelVersion
    deploymentSkuName: cognitiveDeploymentSku
    tags: tags
  }
}

module registry 'modules/registry.bicep' = if (useAcr) {
  name: '${prefix}-registry'
  dependsOn: [cognitive]
  params: {
    name: replace('${prefix}registry', '-', '')
    location: location
    tags: tags
  }
}

var acrId = registry.?outputs.?id ?? ''
var acrLoginServer = registry.?outputs.?loginServer ?? ''

module environment 'modules/environment.bicep' = {
  name: '${prefix}-environment'
  dependsOn: [cognitive]
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
  dependsOn: [environment]
  params: {
    principalId: identity.outputs.principalId
    storageAccountId: storage.outputs.id
    cognitiveAccountId: cognitive.outputs.id
    assignAcrPull: useAcr
    acrId: useAcr ? acrId : ''
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
    server: useAcr ? acrLoginServer : ''
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
  { name: 'HERALD_AGENT_PROVIDER_NAME', value: 'azure' }
  { name: 'HERALD_AGENT_BASE_URL', value: cognitive.outputs.endpoint }
  { name: 'HERALD_AGENT_DEPLOYMENT', value: cognitive.outputs.modelDeploymentName }
  { name: 'HERALD_AGENT_API_VERSION', value: '2025-04-01-preview' }
  { name: 'HERALD_AGENT_AUTH_TYPE', value: 'managed_identity' }
  { name: 'HERALD_AGENT_RESOURCE', value: cognitiveTokenScope }
  { name: 'HERALD_AGENT_CLIENT_ID', value: identity.outputs.clientId }
  { name: 'AZURE_CLIENT_ID', value: identity.outputs.clientId }
]

var authEnvVars = authEnabled
  ? [
      { name: 'HERALD_AUTH_TENANT_ID', value: tenantId }
      { name: 'HERALD_AUTH_CLIENT_ID', value: entraClientId }
    ]
  : []

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
  dependsOn: [app]
  params: {
    name: '${prefix}-migrate'
    location: location
    environmentId: environment.outputs.id
    identityId: identity.outputs.id
    containerImage: containerImage
    registries: registries
    registrySecrets: useAcr ? [] : ghcrSecrets
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
output acrLoginServer string = useAcr ? acrLoginServer : ''
