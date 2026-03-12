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

resource workspace 'Microsoft.OperationalInsights/workspaces@2025-07-01' existing = {
  name: logAnalyticsWorkspaceName
}

resource environment 'Microsoft.App/managedEnvironments@2025-07-01' = {
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
