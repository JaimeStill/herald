@description('Container App name')
param name string

@description('Azure region for the resource')
param location string

@description('Container App Environment resource ID')
param environmentId string

@description('User-assigned managed identity resource ID')
param identityId string

@description('COntainer image reference (GHCR or ACR)')
param containerImage string

@description('Registry configuration array')
param registries array

@description('Container App secrets array')
param secrets array = []

@description('Environment variables for the container')
param envVars array

@description('CPU cores allocated to the container')
param cpu string = '2.0'

@description('Memory allocated to the container')
param memory string = '4Gi'

@description('Minimum number of replicas')
param minReplicas int = 1

@description('Maximum number of replicas')
param maxReplicas int = 3

@description('Target port for ingress')
param targetPort int = 8080

@description('Resource tags')
param tags object = {}

resource app 'Microsoft.App/containerApps@2025-07-01' = {
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
    managedEnvironmentId: environmentId
    configuration: {
      ingress: {
        external: true
        targetPort: targetPort
        transport: 'http'
      }
      registries: registries
      secrets: secrets
    }
    template: {
      containers: [
        {
          name: name
          image: containerImage
          resources: {
            cpu: json(cpu)
            memory: memory
          }
          env: envVars
          probes: [
            {
              type: 'Liveness'
              httpGet: {
                path: '/healthz'
                port: targetPort
              }
              initialDelaySeconds: 5
              periodSeconds: 10
            }
            {
              type: 'Readiness'
              httpGet: {
                path: '/readyz'
                port: targetPort
              }
              initialDelaySeconds: 5
              periodSeconds: 10
            }
          ]
        }
      ]
      scale: {
        minReplicas: minReplicas
        maxReplicas: maxReplicas
      }
    }
  }
}

@description('Container App FQDN')
output fqdn string = app.properties.configuration.ingress.fqdn
