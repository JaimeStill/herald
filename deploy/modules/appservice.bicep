@description('App Service site name')
param name string

@description('Azure region for the resource')
param location string

@description('App Service Plan resource ID')
param appServicePlanId string

@description('User-assigned managed identity resource ID')
param identityId string

@description('Container image reference (GHCR or ACR)')
param containerImage string

@description('Application environment variables')
param envVars array

@description('Use ACR with managed identity pull')
param useAcr bool

@description('GHCR Docker registry settings (required when useAcr is false)')
param ghcrDockerSettings array = []

@description('Resource tags')
param tags object = {}

var websitesPort = [
  {
    name: 'WEBSITES_PORT'
    value: '8080'
  }
]

var appSettings = concat(envVars, websitesPort, useAcr ? [] : ghcrDockerSettings)

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
      acrUseManagedIdentityCreds: useAcr
      acrUserManagedIdentityID: useAcr ? identityId : ''
    }
  }
}

@description('App Service default hostname')
output defaultHostName string = site.properties.defaultHostName
