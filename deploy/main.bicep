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
param cognitiveCustomDomain string

@description('AI model deployment name')
param cognitiveDeploymentName string = 'gpt-5-mini'

@description('AI model name')
param cognitiveModelName string = 'gpt-5-mini'

@description('AI model version')
param cognitiveModelVersion string = '2025-08-07'

@description('AI model deployment SKU')
@allowed(['GlobalStandard', 'DataZoneStandard', 'DataZoneProvisionedManaged', 'GlobalProvisionedManaged'])
param cognitiveDeploymentSku string = 'GlobalStandard'

@description('AI model deployment capacity (TPM in thousands, e.g., 1000 = 1M TPM)')
param cognitiveDeploymentCapacity int = 1000

@description('Cognitive Services Entra token scope (override for Azure Government)')
#disable-next-line no-hardcoded-env-urls
param cognitiveTokenScope string = 'https://cognitiveservices.azure.com/.default'

// --- Compute Target ---

@description('Compute target: containerapp (Container Apps) or appservice (App Service for Containers)')
@allowed(['containerapp', 'appservice'])
param computeTarget string = 'containerapp'

@description('App Service Plan SKU (only used when computeTarget is appservice)')
param appServiceSkuName string = 'P1v3'

// --- Container ---

@description('Container image reference (e.g., ghcr.io/jaimestill/herald:v0.4.0)')
param containerImage string

@description('CPU cores for the Container App')
param containerCpu string = '2.0'

@description('Memory for the Container App (ImageMagick workloads need headroom)')
param containerMemory string = '4Gi'

@description('Minimum replica count (0 enables scale-to-zero)')
param minReplicas int = 0

@description('Maximum replica count')
param maxReplicas int = 3

// --- Registry ---

@description('Pre-existing ACR name in this resource group')
param acrName string

@description('ACR authentication mode (managed_identity or acr_admin)')
@allowed(['managed_identity', 'acr_admin'])
param acrAuthMode string = 'managed_identity'

// --- Auth ---

@description('Enable Azure Entra authentication for the web client')
param authEnabled bool = false

@description('Azure Entra tenant ID (required when authEnabled is true)')
param tenantId string = ''

@description('Entra app registration client ID (required when authEnabled is true)')
param entraClientId string = ''

@description('Entra authority base URL (override for Azure Government, e.g., https://login.microsoftonline.us)')
param authAuthority string = ''

// ============================================================================
// Modules
// ============================================================================

// Deployment order:
//   Shared:        identity → logging → postgres → storage → cognitive
//   containerapp:  → environment → roles → app
//   appservice:    → appServicePlan → roles → appService
//
// Explicit dependsOn chain prevents ARM from parallelizing module deployments,
// which avoids race conditions where a resource reports "provisioned" before it
// is fully ready to accept child operations.

var isContainerApp = computeTarget == 'containerapp'
var isAppService = computeTarget == 'appservice'

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
    deploymentSkuCapacity: cognitiveDeploymentCapacity
    tags: tags
  }
}

var useAcrAdmin = acrAuthMode == 'acr_admin'
var useAcrManagedIdentity = acrAuthMode == 'managed_identity'

resource acr 'Microsoft.ContainerRegistry/registries@2025-11-01' existing = {
  name: acrName
}

var acrId = acr.id
var acrLoginServer = acr.properties.loginServer
var acrAdminUsername = useAcrAdmin ? acr.listCredentials().username : ''
var acrAdminPassword = useAcrAdmin ? acr.listCredentials().passwords[0].value : ''

module environment 'modules/environment.bicep' = if (isContainerApp) {
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

module appServicePlan 'modules/appservice-plan.bicep' = if (isAppService) {
  name: '${prefix}-plan'
  dependsOn: [cognitive]
  params: {
    name: '${prefix}-plan'
    location: location
    skuName: appServiceSkuName
    tags: tags
  }
}

module roles 'modules/roles.bicep' = {
  name: '${prefix}-roles'
  dependsOn: [environment, appServicePlan]
  params: {
    principalId: identity.outputs.principalId
    storageAccountId: storage.outputs.id
    cognitiveAccountId: cognitive.outputs.id
    assignAcrPull: useAcrManagedIdentity
    acrId: acrId
  }
}

// ============================================================================
// Registry Configuration
// ============================================================================

// Container App registry configuration
var acrManagedIdentityRegistries = [
  {
    server: acrLoginServer
    identity: identity.outputs.id
  }
]

var acrAdminRegistries = [
  {
    server: acrLoginServer
    username: acrAdminUsername
    passwordSecretRef: 'acr-password'
  }
]

var registries = useAcrAdmin ? acrAdminRegistries : acrManagedIdentityRegistries

var acrAdminSecrets = [
  {
    name: 'acr-password'
    value: acrAdminPassword
  }
]

var containerAppSecrets = useAcrAdmin ? acrAdminSecrets : []


// ============================================================================
// Environment Variables
// ============================================================================

var baseEnvVars = [
  { name: 'HERALD_ENV', value: 'azure' }
  { name: 'HERALD_SERVER_PORT', value: '8080' }
  { name: 'HERALD_DB_HOST', value: postgres.outputs.fqdn }
  { name: 'HERALD_DB_PORT', value: '5432' }
  { name: 'HERALD_DB_NAME', value: postgres.outputs.databaseName }
  { name: 'HERALD_DB_USER', value: '${prefix}-identity' }
  { name: 'HERALD_DB_SSL_MODE', value: 'require' }
  { name: 'HERALD_DB_TOKEN_SCOPE', value: postgresTokenScope }
  { name: 'HERALD_STORAGE_SERVICE_URL', value: storage.outputs.blobEndpoint }
  { name: 'HERALD_STORAGE_CONTAINER_NAME', value: 'documents' }
  { name: 'HERALD_AUTH_MODE', value: authEnabled ? 'azure' : 'none' }
  { name: 'HERALD_AUTH_MANAGED_IDENTITY', value: 'true' }
  { name: 'HERALD_AGENT_PROVIDER_NAME', value: 'azure' }
  { name: 'HERALD_AGENT_BASE_URL', value: '${cognitive.outputs.endpoint}openai' }
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

var authorityEnvVars = authAuthority != ''
  ? [
      { name: 'HERALD_AUTH_AUTHORITY', value: authAuthority }
    ]
  : []

var envVars = concat(baseEnvVars, authEnvVars, authorityEnvVars)

// ============================================================================
// Container App (when computeTarget == 'containerapp')
// ============================================================================

module app 'modules/app.bicep' = if (isContainerApp) {
  name: '${prefix}-app'
  dependsOn: [roles]
  params: {
    name: '${prefix}-app'
    location: location
    environmentId: environment.?outputs.?id ?? ''
    identityId: identity.outputs.id
    containerImage: containerImage
    registries: registries
    secrets: containerAppSecrets
    envVars: envVars
    cpu: containerCpu
    memory: containerMemory
    minReplicas: minReplicas
    maxReplicas: maxReplicas
    tags: tags
  }
}

// ============================================================================
// App Service (when computeTarget == 'appservice')
// ============================================================================

module appService 'modules/appservice.bicep' = if (isAppService) {
  name: '${prefix}-appservice'
  dependsOn: [roles]
  params: {
    name: '${prefix}-app'
    location: location
    appServicePlanId: appServicePlan.?outputs.?id ?? ''
    identityId: identity.outputs.id
    identityClientId: identity.outputs.clientId
    containerImage: containerImage
    envVars: envVars
    useAcrManagedIdentity: useAcrManagedIdentity
    useAcrAdmin: useAcrAdmin
    acrLoginServer: acrLoginServer
    acrAdminUsername: acrAdminUsername
    acrAdminPassword: acrAdminPassword
    tags: tags
  }
}

// ============================================================================
// Outputs
// ============================================================================

@description('Application public URL')
var appFqdn = isContainerApp
  ? (app.?outputs.?fqdn ?? '')
  : (appService.?outputs.?defaultHostName ?? '')

output appUrl string = 'https://${appFqdn}'

@description('PostgreSQL server FQDN')
output postgresHost string = postgres.outputs.fqdn

@description('Storage account blob endpoint')
output storageBlobEndpoint string = storage.outputs.blobEndpoint

@description('Cognitive Services endpoint')
output cognitiveEndpoint string = cognitive.outputs.endpoint

@description('ACR login server')
output acrLoginServer string = acrLoginServer
