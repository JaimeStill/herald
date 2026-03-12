@description('Azure Container Registry name (globally unique, 5-50 alphanumeric)')
param name string

@description('Azure region for the resource')
param location string

@description('ACR SKU')
@allowed(['Basic', 'Standard', 'Premium'])
param skuName string = 'Basic'

@description('Resource tags')
param tags object = {}

resource registry 'Microsoft.ContainerRegistry/registries@2025-11-01' = {
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
