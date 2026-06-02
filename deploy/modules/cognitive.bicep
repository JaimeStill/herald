@description('Cognitive Services account name')
param name string

@description('Azure region for the resource')
param location string

@description('Custom subdomain name for the account endpoint')
param customSubDomainName string

@description('Cognitive Services SKU')
param skuName string = 'S0'

@description('Cognitive Services kind')
param kind string = 'AIServices'

@description('Model deployment name')
param deploymentName string = 'gpt-5.2'

@description('Model name')
param modelName string = 'gpt-5.2'

@description('Model version')
param modelVersion string = '2025-12-11'

@description('Model format')
param modelFormat string = 'OpenAI'

@description('Deployment SKU name')
param deploymentSkuName string = 'GlobalStandard'

@description('Deployment SKU capacity (TPM in thousands)')
param deploymentSkuCapacity int = 1000

@description('Responsible AI content-filter policy name applied to the model deployment')
param raiPolicyName string = 'herald-content-filter'

@description('Resource tags')
param tags object = {}

resource account 'Microsoft.CognitiveServices/accounts@2025-09-01' = {
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

// Custom Responsible AI content filter. Harm categories block only High severity
// (Low/Medium pass) so legitimate DoD classification-marking vocabulary — NOFORN
// ("Not Releasable to Foreign Nationals"), REL TO <countries>, AEA/RD, and nuclear/WMD
// declassification-exemption terms — is not misclassified as harmful and rejected.
// Mirrors the manually-configured herald-content-filter policy. Raising thresholds to
// High is self-service (no modified-content-filter approval); filters are not turned off.
resource raiPolicy 'Microsoft.CognitiveServices/accounts/raiPolicies@2025-09-01' = {
  parent: account
  name: raiPolicyName
  properties: {
    basePolicyName: 'Microsoft.DefaultV2'
    mode: 'Default'
    contentFilters: [
      { name: 'Violence', severityThreshold: 'High', blocking: true, enabled: true, source: 'Prompt' }
      { name: 'Hate', severityThreshold: 'High', blocking: true, enabled: true, source: 'Prompt' }
      { name: 'Sexual', severityThreshold: 'High', blocking: true, enabled: true, source: 'Prompt' }
      { name: 'Selfharm', severityThreshold: 'High', blocking: true, enabled: true, source: 'Prompt' }
      { name: 'Violence', severityThreshold: 'High', blocking: true, enabled: true, source: 'Completion' }
      { name: 'Hate', severityThreshold: 'High', blocking: true, enabled: true, source: 'Completion' }
      { name: 'Sexual', severityThreshold: 'High', blocking: true, enabled: true, source: 'Completion' }
      { name: 'Selfharm', severityThreshold: 'High', blocking: true, enabled: true, source: 'Completion' }
      { name: 'Jailbreak', blocking: true, enabled: true, source: 'Prompt' }
      { name: 'Indirect Attack', blocking: false, enabled: false, source: 'Prompt' }
      { name: 'Indirect Attack Spotlighting', blocking: false, enabled: false, source: 'Prompt' }
      { name: 'Protected Material Text', blocking: true, enabled: true, source: 'Completion' }
      { name: 'Protected Material Code', blocking: false, enabled: true, source: 'Completion' }
    ]
  }
}

resource deployment 'Microsoft.CognitiveServices/accounts/deployments@2025-09-01' = {
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
    raiPolicyName: raiPolicy.name
  }
}

@description('Cognitive Services account resource ID (for role assignment scope)')
output id string = account.id

@description('Cognitive Services endpoint URL')
output endpoint string = account.properties.endpoint

@description('Model deployment name')
output modelDeploymentName string = deployment.name
