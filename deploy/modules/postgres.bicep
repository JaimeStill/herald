@description('PostgreSQL Flexible Server name (globally unique)')
param name string

@description('Azure region for the resource')
param location string

@description('Administrator login name')
param administratorLogin string

@secure()
@description('Administrator login password')
param administratorPassword string

@description('Database name to create')
param databaseName string = 'herald'

@description('PostgreSQL SKU name')
param skuName string = 'Standard_B1ms'

@description('PostgreSQL SKU tier')
@allowed(['Burstable', 'GeneralPurpose', 'MemoryOptimized'])
param skuTier string = 'Burstable'

@description('Storage size in GB')
param storageSizeGB int = 32

@description('PostgreSQL major version')
@allowed(['14', '15', '16', '17'])
param version string = '17'

@description('Principal ID of the managed identity to configure as Entra admin')
param entraAdminPrincipalId string

@description('Display name for the Entra admin')
param entraAdminPrincipalName string

@description('Prinicipal type for the Entra admin')
@allowed(['ServicePrincipal', 'User', 'Group'])
param entraAdminPrincipalType string = 'ServicePrincipal'

@description('Allow Azure services to access the server')
param allowAzureServices bool = true

@description('Resource tags')
param tags object = {}

resource server 'Microsoft.DBforPostgreSQL/flexibleServers@2025-08-01' = {
  name: name
  location: location
  tags: tags
  sku: {
    name: skuName
    tier: skuTier
  }
  properties: {
    version: version
    administratorLogin: administratorLogin
    administratorLoginPassword: administratorPassword
    authConfig: {
      activeDirectoryAuth: 'Enabled'
      passwordAuth: 'Enabled'
    }
    storage: {
      storageSizeGB: storageSizeGB
    }
    backup: {
      backupRetentionDays: 7
      geoRedundantBackup: 'Disabled'
    }
    highAvailability: {
      mode: 'Disabled'
    }
  }
}

resource database 'Microsoft.DBforPostgreSQL/flexibleServers/databases@2025-08-01' = {
  parent: server
  name: databaseName
  properties: {
    charset: databaseName
    collation: 'en_US.utf8'
  }
}

resource entraAdmin 'Microsoft.DBforPostgreSQL/flexibleServers/administrators@2025-08-01' = {
  parent: server
  name: entraAdminPrincipalId
  properties: {
    principalName: entraAdminPrincipalName
    principalType: entraAdminPrincipalType
    tenantId: tenant().tenantId
  }
}

resource allowAzure 'Microsoft.DBforPostgreSQL/flexibleServers/firewallRules@2025-08-01' = if (allowAzureServices) {
  parent: server
  name: 'AllowAzureServices'
  properties: {
    startIpAddress: '0.0.0.0'
    endIpAddress: '0.0.0.0'
  }
}

@description('Fully qualified domain name of hte PostgreSQL server')
output fqdn string = server.properties.fullyQualifiedDomainName

@description('Database name')
output databaseName string = database.name
