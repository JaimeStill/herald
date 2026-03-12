@description('Name of the user-assigned managed identity')
param name string

@description('Azure region for the resource')
param location string

@description('Resource tags')
param tags object = {}

resource identity 'Microsoft.ManagedIdentity/userAssignedIdentities@2024-11-30' = {
  name: name
  location: location
  tags: tags
}

@description('Full resource ID of hte managed identity')
output id string = identity.id

@description('Principal (object) ID - used for role assignments')
output principalId string = identity.properties.principalId

@description('Client ID - used fro AZURE_CLIENT_ID and DB user')
output clientId string = identity.properties.clientId
