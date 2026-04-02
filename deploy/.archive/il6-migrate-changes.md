# IL6 Migration Infrastructure Changes

Comprehensive guide for updating the IL6 environment to use the standalone `migrate.exe` binary instead of the in-container migration infrastructure. Apply these changes to the IL6 Bicep files and deployment procedures.

## Overview

The `migrate` binary is no longer bundled in the Herald container image. It is now published as a standalone release artifact under `migrate-v*` tags and included in the CDS bundle as `migrate.exe`. Operators run it directly from the deployment machine against PostgreSQL.

---

## Step 1: Remove Migration App Service (if deployed)

If `herald-migrate` was deployed as an App Service during triage:

```powershell
az webapp delete `
  --name herald-migrate `
  --resource-group heraldgroup `
  --yes
```

Verify it's gone:

```powershell
az resource list `
  --resource-group heraldgroup `
  --query "[?name=='herald-migrate']" `
  --output table
```

---

## Step 2: Remove Migration Infrastructure from Bicep

### 2a. Delete module files

Delete the following files from `deploy\modules\`:

- `migration-job.bicep`
- `appservice-migration.bicep`

### 2b. Update `main.bicep`

**Remove the `migrationDsn` variable.** Find and delete:

```bicep
var migrationDsn = 'postgres://${postgresAdminLogin}:${postgresAdminPassword}@${postgres.outputs.fqdn}:5432/${postgres.outputs.databaseName}?sslmode=require'
```

**Remove the `migrationJob` module.** Find and delete the entire block:

```bicep
module migrationJob 'modules/migration-job.bicep' = if (isContainerApp) {
  name: '${prefix}-migration-job'
  dependsOn: [app]
  params: {
    name: '${prefix}-migrate'
    location: location
    environmentId: environment.?outputs.?id ?? ''
    identityId: identity.outputs.id
    containerImage: containerImage
    registries: registries
    registrySecrets: useAcr ? [] : ghcrSecrets
    databaseDsn: migrationDsn
    tags: tags
  }
}
```

**Remove the `appServiceMigration` module.** Find and delete the entire block:

```bicep
module appServiceMigration 'modules/appservice-migration.bicep' = if (isAppService) {
  name: '${prefix}-appservice-migration'
  dependsOn: [appService]
  params: {
    name: '${prefix}-migrate'
    location: location
    appServicePlanId: appServicePlan.?outputs.?id ?? ''
    identityId: identity.outputs.id
    containerImage: containerImage
    useAcr: useAcr
    ghcrDockerSettings: useAcr ? [] : ghcrDockerSettings
    databaseDsn: migrationDsn
    tags: tags
  }
}
```

**Update the deployment order comment.** Change:

```bicep
// Deployment order:
//   Shared:        identity → logging → postgres → storage → cognitive
//   containerapp:  → environment → roles → app → migrationJob
//   appservice:    → appServicePlan → roles → appService → appServiceMigration
```

To:

```bicep
// Deployment order:
//   Shared:        identity → logging → postgres → storage → cognitive
//   containerapp:  → environment → roles → app
//   appservice:    → appServicePlan → roles → appService
```

### 2c. Recompile

```powershell
bicep.exe build deploy\main.bicep
```

### 2d. Redeploy (optional)

The migration modules were already removed from the deployment. A redeploy will clean up the ARM deployment history but isn't required for functionality. The existing `herald-migrate` resource (if any) was deleted in Step 1.

---

## Step 3: Running Migrations with the Standalone Binary

The `migrate.exe` binary is included in the CDS bundle. It embeds all SQL migrations and connects directly to PostgreSQL.

### 3a. Add a temporary firewall rule

The deployment machine needs network access to PostgreSQL:

```powershell
$myIp = (Invoke-RestMethod -Uri "https://ifconfig.me/ip").Trim()

az postgres flexible-server firewall-rule create `
  --resource-group heraldgroup `
  --name herald-db `
  --rule-name MigrateAccess `
  --start-ip-address $myIp `
  --end-ip-address $myIp
```

> **Note:** If `ifconfig.me` is not reachable on IL6, determine your public IP through your network team or Azure Portal.

### 3b. Construct the DSN

```
postgres://<admin-login>:<admin-password>@herald-db.postgres.database.azure.<il6-domain-root>:5432/herald?sslmode=require
```

Replace:
- `<admin-login>` — the `postgresAdminLogin` value from `main.parameters.json`
- `<admin-password>` — the `postgresAdminPassword` value from `main.parameters.json`
- `<il6-domain-root>` — the IL6 domain root discovered during initial setup

### 3c. Run the migration

```powershell
.\migrate.exe -dsn "postgres://<admin-login>:<admin-password>@herald-db.postgres.database.azure.<il6-domain-root>:5432/herald?sslmode=require" -up
```

### 3d. Verify

```powershell
.\migrate.exe -dsn "postgres://<admin-login>:<admin-password>@herald-db.postgres.database.azure.<il6-domain-root>:5432/herald?sslmode=require" -version
```

This prints the current migration version and dirty state.

### 3e. Remove the firewall rule

```powershell
az postgres flexible-server firewall-rule delete `
  --resource-group heraldgroup `
  --name herald-db `
  --rule-name MigrateAccess `
  --yes
```

---

## Step 4: Update IL6 Deployment Guide

Update the deployment steps in your IL6 documentation to replace any references to `herald-migrate` (Container Apps Job or App Service) with the standalone binary approach documented above.

---

## Troubleshooting

### Dirty Migration State

If a migration fails partway through, the `schema_migrations` table will be in a dirty state. Force the version to recover:

```powershell
.\migrate.exe -dsn "<dsn>" -force <version>
```

Then re-run:

```powershell
.\migrate.exe -dsn "<dsn>" -up
```

### Network Access

If the migration binary cannot connect to PostgreSQL:

1. Verify the firewall rule was created: `az postgres flexible-server firewall-rule list --resource-group heraldgroup --name herald-db --output table`
2. Verify your public IP matches the rule
3. Verify the DSN hostname resolves: `Resolve-DnsName herald-db.postgres.database.azure.<il6-domain-root>`
4. If DNS fails, run `Clear-DnsClientCache` and retry
