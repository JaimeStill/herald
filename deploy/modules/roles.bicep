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

resource storageAccount 'Microsoft.Storage/storageAccounts@2025-06-01' existing = {
  name: last(split(storageAccountId, '/'))
}

resource cognitiveAccount 'Microsoft.CognitiveServices/accounts@2025-06-01' existing = {
  name: last(split(cognitiveAccountId, '/'))
}

resource containerRegistry 'Microsoft.ContainerRegistry/registries@2025-11-01' existing = if (assignAcrPull && acrId != '') {
  name: last(split(acrId, '/'))
}

resource storageBlobRole 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(storageAccountId, principalId, storageBlobDataContributor)
  scope: storageAccount
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', storageBlobDataContributor)
    principalId: principalId
    principalType: 'ServicePrincipal'
  }
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

resource acrRole 'Microsoft.Authorization/roleAssignments@2022-04-01' = if (assignAcrPull && acrId != '') {
  name: guid(acrId, principalId, acrPull)
  scope: containerRegistry
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', acrPull)
    principalId: principalId
    principalType: 'ServicePrincipal'
  }
}
