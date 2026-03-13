@description('Cognitive Services account name')
param name string

@description('Azure region for the resource')
param location string

@description('Custom subdomain name for the account endpoint')
param customSubDomainName string

@description('Cognitive Services SKU')
param skuName string = 'S0'

@description('Cognitive Services kind')
param kind string = 'OpenAI'

@description('Model deployment name')
param deploymentName string = 'gpt-5-mini'

@description('Model name')
param modelName string = 'gpt-5-mini'

@description('Model version')
param modelVersion string = '2025-08-07'

@description('Model format')
param modelFormat string = 'OpenAI'

@description('Deployment SKU name')
param deploymentSkuName string = 'GlobalStandard'

@description('Deployment SKU capacity (TPM in thousands)')
param deploymentSkuCapacity int = 1000

@description('Resource tags')
param tags object = {}

resource account 'Microsoft.CognitiveServices/accounts@2025-06-01' = {
  name: name
  location: location
  tags: tags
  kind: kind
  sku: {
    name: skuName
  }
  properties: {
    customSubDomainName: customSubDomainName
    publicNetworkAccess: 'Enabled'
  }
}

resource deployment 'Microsoft.CognitiveServices/accounts/deployments@2025-06-01' = {
  parent: account
  name: deploymentName
  sku: {
    name: deploymentSkuName
    capacity: deploymentSkuCapacity
  }
  properties: {
    model: {
      format: modelFormat
      name: modelName
      version: modelVersion
    }
  }
}

@description('Cognitive Services account resource ID (for role assignment scope)')
output id string = account.id

@description('Cognitive Services endpoint URL')
output endpoint string = account.properties.endpoint

@description('Model deployment name')
output modelDeploymentName string = deployment.name
