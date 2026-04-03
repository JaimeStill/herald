@description('App Service site name')
param name string

@description('Azure region for the resource')
param location string

@description('App Service Plan resource ID')
param appServicePlanId string

@description('User-assigned managed identity resource ID')
param identityId string

@description('User-assigned managed identity client ID (for ACR pull)')
param identityClientId string

@description('Container image reference')
param containerImage string

@description('Application environment variables')
param envVars array

@description('Use ACR with managed identity pull')
param useAcrManagedIdentity bool

@description('Use ACR with admin credentials')
param useAcrAdmin bool

@description('ACR login server')
param acrLoginServer string

@description('ACR admin username')
param acrAdminUsername string = ''

@secure()
@description('ACR admin password')
param acrAdminPassword string = ''

@description('Resource tags')
param tags object = {}

var platformSettings = [
  {
    name: 'WEBSITES_PORT'
    value: '8080'
  }
  {
    name: 'WEBSITE_LOAD_CERTIFICATES'
    value: '*'
  }
  {
    name: 'SSL_CERT_DIR'
    value: '/var/ssl/certs'
  }
]

var acrAdminDockerSettings = [
  {
    name: 'DOCKER_REGISTRY_SERVER_URL'
    value: 'https://${acrLoginServer}'
  }
  {
    name: 'DOCKER_REGISTRY_SERVER_USERNAME'
    value: acrAdminUsername
  }
  {
    name: 'DOCKER_REGISTRY_SERVER_PASSWORD'
    value: acrAdminPassword
  }
]

var dockerSettings = useAcrAdmin ? acrAdminDockerSettings : []
var appSettings = concat(envVars, platformSettings, dockerSettings)

resource site 'Microsoft.Web/sites@2024-04-01' = {
  name: name
  location: location
  tags: tags
  kind: 'app,linux,container'
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${identityId}': {}
    }
  }
  properties: {
    serverFarmId: appServicePlanId
    clientAffinityEnabled: false
    httpsOnly: true
    siteConfig: {
      linuxFxVersion: 'DOCKER|${containerImage}'
      healthCheckPath: '/healthz'
      appSettings: appSettings
      acrUseManagedIdentityCreds: useAcrManagedIdentity
      acrUserManagedIdentityID: useAcrManagedIdentity ? identityClientId : ''
    }
  }
}

@description('App Service default hostname')
output defaultHostName string = site.properties.defaultHostName
