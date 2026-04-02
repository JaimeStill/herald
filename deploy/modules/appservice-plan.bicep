@description('App Service Plan name')
param name string

@description('Azure region for the resource')
param location string

@description('App Service Plan SKU name')
param skuName string = 'P1v3'

@description('Resource tags')
param tags object = {}

resource plan 'Microsoft.Web/serverfarms@2024-04-01' = {
  name: name
  location: location
  tags: tags
  kind: 'linux'
  sku: {
    name: skuName
  }
  properties: {
    reserved: true
  }
}

@description('App Service Plan resource ID')
output id string = plan.id
