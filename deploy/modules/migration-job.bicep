@description('Container Apps Job name')
param name string

@description('Azure region for the resource')
param location string

@description('Container App Environment resource ID')
param environmentId string

@description('User-assigned managed identity resource ID')
param identityId string

@description('Container image reference (same image as the app)')
param containerImage string

@description('Registry configuration array')
param registries array

@secure()
@description('Full PostgreSQL DSN for migrations')
param databaseDsn string

@description('CPU cores allocated to the job container')
param cpu string = '0.5'

@description('Memory allocated to the job container')
param memory string = '1Gi'

@description('Resource tags')
param tags object = {}

resource job 'Microsoft.App/jobs@2025-07-01' = {
  name: name
  location: location
  tags: tags
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${identityId}': {}
    }
  }
  properties: {
    environmentId: environmentId
    configuration: {
      triggerType: 'Manual'
      replicaTimeout: 300
      replicaRetryLimit: 0
      registries: registries
      secrets: [
        {
          name: 'db-dsn'
          value: databaseDsn
        }
      ]
    }
    template: {
      containers: [
        {
          name: name
          image: containerImage
          command: [
            '/usr/local/bin/migrate'
            '-up'
          ]
          resources: {
            cpu: json(cpu)
            memory: memory
          }
          env: [
            {
              name: 'HERALD_DB_DSN'
              secretRef: 'db-dsn'
            }
          ]
        }
      ]
    }
  }
}
