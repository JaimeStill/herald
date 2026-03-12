@description('Name of the Log Analytics workspace')
param name string

@description('Azure region for the resource')
param location string

@description('Number of days to retain logs')
param retentionInDays int = 30

@description('Resource tags')
param tags object = {}

resource workspace 'Microsoft.OperationalInsights/workspaces@2025-07-01' = {
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

@description('Workspace name (needed by Container App Environment)')
output name string = workspace.name
