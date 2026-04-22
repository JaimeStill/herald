# Issue 140 - Model name missing from review footer in production

## Context

On the IL6 deployment the review view footer renders `/ azure  Classified [date]` — the model portion before the `/` is empty. On local (`mise run dev`) the same footer renders `gpt-5-mini / azure  Classified [date]`. The user confirmed local still targets the same Azure AI Foundry deployment, so the difference must come from configuration, not the agent backend.

## Root Cause

The Dockerfile (`Dockerfile:16-22`) copies only the compiled `/herald` binary into the final image — `config.json` is NOT bundled. The container's `WORKDIR /app` is empty, so `config.Load` (`internal/config/config.go:89`) finds no base file and no overlay, and proceeds to `finalize()` with a zero-value `tauconfig.AgentConfig`.

`FinalizeAgent` (`internal/config/agent.go:25-29`) then:
1. `loadAgentDefaults` merges `tauconfig.DefaultAgentConfig()` — which returns `Model: DefaultModelConfig()` whose `Name` is the empty string (`~/tau/protocol/config/model.go:14-18`).
2. `loadAgentEnv` would set `c.Model.Name` from `HERALD_AGENT_MODEL_NAME` (`internal/config/agent.go:53-55`), but that env var is not injected by `deploy/main.bicep`.

Result: `c.Model.Name` stays empty, flows through `runtime.Agent.Model.Name` at `internal/api/domain.go:34` → `classifications.New(..., modelName, ...)`, lands in the `model_name` column as `''`, and renders empty in `app/client/ui/modules/classification-panel.ts:204` (`${c.model_name} / ${c.provider_name}`).

Pure deployment-layer bug. The Go config layer works correctly and `tests/config/config_test.go:477-516` (`TestAgentEnvOverrides`) already exercises `HERALD_AGENT_MODEL_NAME`.

## Full Env-Var Audit

To address the broader concern ("any config settings that aren't being established in Bicep"), I walked every `HERALD_*` env var consumed by the codebase against the `baseEnvVars` list in `deploy/main.bicep:274-294`.

**Injected by Bicep (9):** `HERALD_ENV`, `HERALD_SERVER_PORT`, `HERALD_DB_HOST`, `HERALD_DB_PORT`, `HERALD_DB_NAME`, `HERALD_DB_USER`, `HERALD_DB_SSL_MODE`, `HERALD_DB_TOKEN_SCOPE`, `HERALD_STORAGE_SERVICE_URL`, `HERALD_STORAGE_CONTAINER_NAME`, `HERALD_AUTH_MODE`, `HERALD_AUTH_MANAGED_IDENTITY`, `HERALD_AGENT_PROVIDER_NAME`, `HERALD_AGENT_BASE_URL`, `HERALD_AGENT_DEPLOYMENT`, `HERALD_AGENT_API_VERSION`, `HERALD_AGENT_AUTH_TYPE`, `HERALD_AGENT_RESOURCE`, `HERALD_AGENT_CLIENT_ID` (+ conditional `HERALD_AUTH_TENANT_ID`, `HERALD_AUTH_CLIENT_ID`, `HERALD_AUTH_AUTHORITY`).

**Rely on tangible code defaults (OK to leave un-set):**
- Server: `_SERVER_HOST=0.0.0.0`, `_SERVER_READ_TIMEOUT=1m`, `_SERVER_WRITE_TIMEOUT=15m`, `_SERVER_SHUTDOWN_TIMEOUT=30s` (`internal/config/server.go:76-92`).
- Root: `_SHUTDOWN_TIMEOUT=30s`, `_VERSION=0.1.0` (`internal/config/config.go:166-173`).
- API: `_API_BASE_PATH=/api`, `_API_MAX_UPLOAD_SIZE=50MB` (`internal/config/api.go:70-77`).
- Pagination / CORS: have package-level defaults (`pkg/pagination/config.go:41`, CORS disabled by default).
- Auth: `_AUTH_SCOPE` derives to `access_as_user` (`pkg/auth/config.go:202-204`), `_AUTH_CACHE_LOCATION=localStorage` (line 144).
- Database pool/timeout knobs: all sane defaults (`pkg/database/config.go:117-145`), including `_DB_TOKEN_SCOPE` which Bicep already overrides via the `postgresTokenScope` parameter.
- Storage: `_STORAGE_MAX_LIST_SIZE=50`, `_STORAGE_CONTAINER_NAME=documents` (`pkg/storage/config.go:51-61`).

**Intentionally not set (mode-incompatible):**
- `HERALD_AGENT_TOKEN`, `HERALD_DB_PASSWORD`, `HERALD_AUTH_CLIENT_SECRET`, `HERALD_STORAGE_CONNECTION_STRING` — not used when running with managed identity.

**Missing, no tangible default (1):** `HERALD_AGENT_MODEL_NAME` — `tauconfig.DefaultModelConfig()` returns an empty `Name`; the Azure deployment semantics also have no reason to guess here. This is the only env var that should be added.

Note: `cognitiveModelName` (`deploy/main.bicep:48`, default `'gpt-5-mini'`) is the semantic model identifier, distinct from `cognitiveDeploymentName` (line 45, the Azure deployment name). They happen to match today but are semantically different — `HERALD_AGENT_MODEL_NAME` should bind to `cognitiveModelName`, mirroring how `HERALD_AGENT_DEPLOYMENT` binds to `cognitive.outputs.modelDeploymentName`.

## Approach

Single-change deployment fix:
1. Add `HERALD_AGENT_MODEL_NAME` to the `baseEnvVars` list in `deploy/main.bicep` bound to `cognitiveModelName`.
2. Regenerate `deploy/main.json` via `az bicep build` — this machine has `bicep` v0.41.2 available (the IL6 air-gapped deploy host does not, which is why `main.json` is checked in).
3. Bump `deploy/main.parameters.json` `contentVersion` from `0.4.2.0` to `0.5.0.0` — this rolls with the Phase 5 release and will ship as part of the full `v0.5.0` deployment, not as a Phase-5 dev build.
4. Update `deploy/README.md` env var reference table.
5. Add `deploy/update.md` — a manual-patch recipe for the IL6 air-gapped deploy host. Until a Phase-6 (or later) CI/CD process for IaC lands, each change to `deploy/` files must be re-applied by hand on the IL6 side. This file is the canonical delta for the v0.5.0 deploy.

No Go code changes. No new tests — existing `TestAgentEnvOverrides` already covers the env override path. The bug is purely deployment-layer, so a regression test at the Go layer would not exercise the failure mode.

## Files to Modify

- `deploy/main.bicep` — add one env var entry to `baseEnvVars`.
- `deploy/main.json` — regenerated via `az bicep build`.
- `deploy/main.parameters.json` — bump `contentVersion` to `0.5.0.0`.
- `deploy/README.md` — append one row to the agent env var table.
- `deploy/update.md` — new file documenting the IL6-side manual edits.
- `CHANGELOG.md` — add `v0.5.0-dev.132.140` section.

## Implementation

### Step 1: Add env var binding in `deploy/main.bicep`

Insert after line 289 (`HERALD_AGENT_DEPLOYMENT`), keeping the deployment/model pair adjacent:

```bicep
  { name: 'HERALD_AGENT_DEPLOYMENT', value: cognitive.outputs.modelDeploymentName }
  { name: 'HERALD_AGENT_MODEL_NAME', value: cognitiveModelName }
  { name: 'HERALD_AGENT_API_VERSION', value: '2025-04-01-preview' }
```

### Step 2: Regenerate `deploy/main.json`

```bash
az bicep build --file deploy/main.bicep --outfile deploy/main.json
```

Spot-check the diff to confirm the only change is two new `HERALD_AGENT_MODEL_NAME` entries (one per Container App variant and one per App Service variant in the flattened `createArray` expression).

### Step 3: Bump parameter content version

In `deploy/main.parameters.json`, change `"contentVersion": "0.4.2.0"` to `"contentVersion": "0.5.0.0"`. Do not touch `containerImage` — the release workflow will bump the image tag when Phase 5's release tag is cut.

### Step 4: Update deployment README

In the env var reference table at `deploy/README.md:424-430`, insert between the `HERALD_AGENT_DEPLOYMENT` and `HERALD_AGENT_API_VERSION` rows:

```
| `HERALD_AGENT_MODEL_NAME` | `cognitiveModelName` param | Yes |
```

### Step 5: Create `deploy/update.md` — IL6 manual patch recipe

New file that captures the exact edits the IL6 air-gapped deploy host must apply by hand. Structure:

1. **Preamble** — one paragraph explaining why the file exists (air-gapped host has no `az bicep`, CI/CD for IaC hasn't been built yet, this is the canonical delta log for each deploy-touching PR).
2. **Target version** — `0.5.0.0`, introduced with issue #140 / PR associated with this session.
3. **Per-file edits**, each as a unified-diff-style block or a before/after pair so the operator can grep the right location and apply the change deterministically:
   - `deploy/main.bicep`: show the 3-line context around the new `HERALD_AGENT_MODEL_NAME` entry.
   - `deploy/main.json`: show the `createObject('name', 'HERALD_AGENT_MODEL_NAME', 'value', parameters('cognitiveModelName'))` insertion point in each of the two flattened `concat(createArray(...))` expressions — cite the surrounding `HERALD_AGENT_DEPLOYMENT` and `HERALD_AGENT_API_VERSION` entries as anchors.
   - `deploy/main.parameters.json`: `contentVersion` `"0.4.2.0"` → `"0.5.0.0"`.
4. **Parameter values that must remain air-gap-specific** — remind the operator not to blindly copy `main.parameters.json` from commercial; the `tenantId`, `entraClientId`, `cognitiveCustomDomain`, etc. differ on IL6 and the commercial file is not authoritative for those values. The operator's existing IL6 parameter file is the source of truth for sensitive values; only the `contentVersion` bump and any newly added parameters are carried across.
5. **Verification commands** (runnable on IL6 without internet): `grep -n HERALD_AGENT_MODEL_NAME deploy/main.json` (expect two hits), `jq '.contentVersion' deploy/main.parameters.json` (expect `"0.5.0.0"`), and a deployment-time check to confirm `az containerapp show` surfaces the new env var.
6. **Rollback note** — short paragraph: if the classification view regresses, re-deploy the previous `0.4.2.0` template. The app itself is unaffected (no Go code change), so a binary rollback isn't required.

## Validation Criteria

- [ ] `mise run vet` passes.
- [ ] `mise run test ./tests/config/...` passes (existing `TestAgentEnvOverrides` already covers the env-override path).
- [ ] `grep HERALD_AGENT_MODEL_NAME deploy/main.bicep deploy/main.json deploy/README.md deploy/update.md` returns expected hits in all five files.
- [ ] `deploy/main.parameters.json` `contentVersion` reads `0.5.0.0`.
- [ ] `az bicep build` completes without warnings relative to baseline.
- [ ] `deploy/update.md` opens cleanly and the per-file edits can be applied verbatim by a human operator without needing to cross-reference the bicep source.
- [ ] Local `mise run dev` footer unchanged (still renders `gpt-5-mini / azure`) — no code path touched.
- [ ] Root cause (Dockerfile doesn't ship `config.json`; Bicep omitted `HERALD_AGENT_MODEL_NAME`) summarized in the PR description.
- [ ] Post-deploy (out of scope for PR merge): IL6 review footer renders `gpt-5-mini / azure  Classified [date]` for a freshly classified document.

## Out of Scope

- Go code changes — config ordering is correct and already tested.
- Container image rebuild / release tagging — handled by the release workflow after PR merge.
- Bundling `config.json` into the Docker image — production intentionally relies on env vars; adding a config file would fight that pattern for no gain.
