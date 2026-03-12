# Notes

Deployment quick ref:

1. Create Resource Group:

```bash
az group create \
  --name HeraldDeploymentGroup \
  --location centralus
```

2. Deploy:

```bash
az deployment group create \
  --resource-group HeraldDeploymentGroup \
  --template-file deploy/main.bicep \
  --parameters deploy/main.parameters.json \
  --parameters \
    location='centralus' \
    postgresAdminPassword='Sh4d0wCl0n34dm1n!!' \
    ghcrUsername='jaimestill' \
    ghcrPassword="$(gh auth token)" \
    authEnabled=true \
    tenantId='64819121-d17e-4216-a81e-fa8528635fb8' \
    entraClientId='52e63088-b087-4d03-8b61-0152660e963d'
```

3. Run Migrations:

```bash
az containerapp job start \
  --name herald-migrate \
  --resource-group HeraldDeploymentGroup
```

4. Verify:

``` bash
APP_URL=$(az deployment group show \
  --resource-group HeraldDeploymentGroup \
  --name herald \
  --query 'properties.outputs.appUrl.value' \
  --output tsv)

curl -s "$APP_URL/healthz"
curl -s "$APP_URL/readyz"
```

5. Delete Resource Group:

```bash
az group delete \
  --resource-group HeraldDeploymentGroup -y
```

6. Purge Cognitive Services Account:

```bash
az cognitiveservices account purge \
  --resource-group HeraldDeploymentGroup \
  --name herald-ai-prod \
  --location centralus
```
