# Plan: Remove Migration Infrastructure, Add Standalone Binary Release

## Context

The migration web app approach (`appservice-migration.bicep`) doesn't work because the Dockerfile uses `ENTRYPOINT ["herald"]` and App Service's `appCommandLine` sets CMD, not the entrypoint. Rather than work around this, we're removing all migration infrastructure from the container image and Bicep entirely. The `migrate` binary becomes a standalone release artifact that operators run directly from their deployment machine. This is cleaner — migrations are a one-shot operation, not a long-running service.

This is part of the v0.4.1 release.

## Changes to `/home/jaime/code/herald`

### 1. Dockerfile — Revert to ENTRYPOINT, remove migrate binary

- Change `CMD ["herald"]` back to `ENTRYPOINT ["herald"]`
- Remove `RUN CGO_ENABLED=0 go build -o /migrate ./cmd/migrate`
- Remove `COPY --from=build /migrate /usr/local/bin/migrate`

### 2. `.github/workflows/release.yml` — No changes needed

The existing release workflow handles the container image. Migration binaries are released independently.

### 3. `.github/workflows/migrate-release.yml` — New workflow for migrate binary releases

Triggered on tag push matching `migrate-v*` (e.g., `migrate-v0.1.0`). Builds the migrate binary for `linux-amd64` and `windows-amd64`, attaches both to a GitHub Release. Decoupled from the container image release cycle. Uses `taiki-e/create-gh-release-action` with `prefix: migrate-` so it reads from the `## migrate-v*` section of CHANGELOG.md.

### 4. `CHANGELOG.md` — Add migrate release section convention

Add a `## migrate-v0.1.0` section (or similar) below the main `## v0.4.1` section. The `migrate-release.yml` workflow uses `prefix: migrate-` with `taiki-e/create-gh-release-action` to extract the correct changelog section for migrate-tagged releases.

### 4. `deploy/main.bicep` — Remove migration modules and DSN

- Remove `migrationDsn` var
- Remove `migrationJob` module (Container Apps)
- Remove `appServiceMigration` module (App Service)
- Remove `postgresAdminPassword` from the migration DSN composition (it's still needed as a Bicep parameter for PostgreSQL provisioning)
- Remove `ghcrDockerSettings` if only used by the migration module (check if appservice.bicep also uses it — yes it does, so keep it)

### 5. Delete migration module files

- `deploy/modules/migration-job.bicep`
- `deploy/modules/appservice-migration.bicep`

### 6. `deploy/README.md` — Update migration docs

- Remove Container Apps Job migration section
- Remove App Service migration web app section
- Add "Running Migrations" section documenting the standalone binary approach
- Update architecture diagram to remove migration modules

### 7. `deploy/il6.md` — Update migration step

- Replace the `az webapp restart` migration step with running the binary directly
- Document DSN construction for the migration binary

### 8. `deploy/il6-bicep-updates.md` — Update with migration removal

- Document removal of migration modules and DSN from main.bicep
- Document the standalone binary approach

## Changes to `/home/jaime/code/_s2va/herald`

### 9. `.github/workflows/cds-release.yaml` — Remove deploy manifests, add migrate binary

- Remove the "Stage deploy manifests" step (`cp -r source/deploy/*`)
- Add a Go build step to compile `migrate` for `windows-amd64` (IL6 deployment machine is Windows)
- Stage the binary as `staging/migrate.exe`
- Bundle now contains: `image.tar` + `migrate.exe` (no more `deploy/`)

### 10. `README.md` — Update bundle contents and IL6 instructions

- Update bundle description (no more deploy manifests)
- Update extraction instructions to show migrate binary usage
- Document the DSN and how to run migrations

## New File

### `deploy/il6-migrate-changes.md`

Comprehensive document with detailed code blocks capturing all changes needed on the IL6 side:
- Remove `herald-migrate` App Service (if it was created)
- Remove migration references from the IL6 Bicep files
- Update `main.parameters.json` (remove migration-related params if any)
- Document how to run `migrate.exe` directly with DSN
- Update deployment steps in any IL6 guides

## Verification

1. `az bicep build -f deploy/main.bicep` — compiles cleanly
2. Deploy to commercial Azure with both `computeTarget` values — no migration resources created
3. Build migrate binary locally: `CGO_ENABLED=0 go build -o migrate ./cmd/migrate`
4. Run `./migrate -version` to verify it works standalone
5. Verify the release workflow compiles (check YAML syntax)

## Files Summary

**Herald repo — modify:**
- `Dockerfile`
- `CHANGELOG.md`
- `deploy/main.bicep`
- `deploy/README.md`
- `deploy/il6.md`
- `deploy/il6-bicep-updates.md`

**Herald repo — create:**
- `.github/workflows/migrate-release.yml`
- `deploy/il6-migrate-changes.md`

**Herald repo — delete:**
- `deploy/modules/migration-job.bicep`
- `deploy/modules/appservice-migration.bicep`

**CDS proxy repo — modify:**
- `.github/workflows/cds-release.yaml`
- `README.md`
