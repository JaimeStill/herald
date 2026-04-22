# 140 - Model name missing from review footer in production

## Problem Context

On the IL6 deployment the review view footer renders `/ azure  Classified [date]` — the model portion before the `/` is empty. On local (`mise run dev`) the same footer renders `gpt-5-mini / azure  Classified [date]`. The user confirmed local still targets the same Azure AI Foundry deployment, so the difference must come from configuration, not the agent backend.

## Architecture Approach

Root cause: The Dockerfile ships only the compiled binary — `config.json` is not in the production image. Production relies entirely on `HERALD_*` env vars, and `deploy/main.bicep` omits `HERALD_AGENT_MODEL_NAME`. After a full audit of every `HERALD_*` env var the code consumes, `HERALD_AGENT_MODEL_NAME` is the only missing variable that lacks a tangible code default (`tauconfig.DefaultModelConfig().Name` is the empty string). All other un-set vars either have production-viable defaults or are mode-incompatible (e.g., `HERALD_AGENT_TOKEN` when running with managed identity).

Fix: add the one env var to `deploy/main.bicep`, regenerate `deploy/main.json` via `az bicep build`, bump the parameter `contentVersion` to align with the in-progress Phase 5 target (`0.5.0.0`, deployed when Phase 5 ships), update the deploy README, and author `deploy/update.md` as the canonical manual-patch recipe for the air-gapped IL6 host.

No Go code changes. No new tests — `TestAgentEnvOverrides` (`tests/config/config_test.go:477-516`) already covers the env-override path; the bug is purely deployment-layer.

## Implementation

### Step 1: Add env var binding in `deploy/main.bicep`

In the `baseEnvVars` list (around line 287), insert a new entry immediately after the `HERALD_AGENT_DEPLOYMENT` row so the deployment/model pair stays adjacent:

```bicep
  { name: 'HERALD_AGENT_DEPLOYMENT', value: cognitive.outputs.modelDeploymentName }
  { name: 'HERALD_AGENT_MODEL_NAME', value: cognitiveModelName }
  { name: 'HERALD_AGENT_API_VERSION', value: '2025-04-01-preview' }
```

`cognitiveModelName` is the existing parameter declared at `deploy/main.bicep:48` (default `'gpt-5-mini'`) and already flows into the cognitive module at line 183. It is semantically distinct from `cognitiveDeploymentName` (the Azure deployment identifier) — they coincide in value today but are wired separately on purpose.

### Step 2: Regenerate `deploy/main.json`

```bash
az bicep build --file deploy/main.bicep --outfile deploy/main.json
```

Spot-check with:

```bash
grep -c HERALD_AGENT_MODEL_NAME deploy/main.json   # expect 2
git diff deploy/main.json | head -80
```

The diff should only add `createObject('name', 'HERALD_AGENT_MODEL_NAME', 'value', parameters('cognitiveModelName'))` to the two flattened `concat(createArray(...))` expressions (one for the Container App path, one for the App Service path).

### Step 3: Bump parameter content version

In `deploy/main.parameters.json`, change:

```diff
- "contentVersion": "0.4.2.0",
+ "contentVersion": "0.5.0.0",
```

Leave `containerImage` alone — the release workflow will bump the image tag when Phase 5's release tag is cut.

### Step 4: Update deployment README env var table

In `deploy/README.md`, insert one row between the `HERALD_AGENT_DEPLOYMENT` (line 426) and `HERALD_AGENT_API_VERSION` (line 427) rows:

```diff
  | `HERALD_AGENT_DEPLOYMENT` | `cognitive.outputs.modelDeploymentName` | Yes |
+ | `HERALD_AGENT_MODEL_NAME` | `cognitiveModelName` param | Yes |
  | `HERALD_AGENT_API_VERSION` | `2025-04-01-preview` | Yes |
```

### Step 5: Create `deploy/update.md`

New file — a manual-patch recipe for the air-gapped IL6 deploy host. The IL6 machine has no `az bicep` and no CI/CD for IaC yet, so each deploy-touching PR must be re-applied by hand there. `update.md` is the canonical delta log for the next deploy.

```markdown
# IL6 Deployment Update Recipe

The IL6 deployment host is air-gapped and cannot run `az bicep`. Until a CI/CD
process for IaC substitution is in place, the edits below must be re-applied by
hand to the IL6-side `deploy/` tree for each deploy-touching PR. This file is
the canonical delta log — it is overwritten on every PR that modifies
`deploy/`, so the operator always has a single authoritative patch to apply.

## Target Version

`0.5.0.0` — corresponds to the Phase 5 release (v0.5.0). Introduced alongside
issue #140 (model name missing from review footer).

## Per-File Edits

### `deploy/main.bicep`

In the `baseEnvVars` list, insert one line so the deployment/model pair stays
adjacent. Anchors: the line before is `HERALD_AGENT_DEPLOYMENT`, the line
after is `HERALD_AGENT_API_VERSION`.

```bicep
  { name: 'HERALD_AGENT_DEPLOYMENT', value: cognitive.outputs.modelDeploymentName }
  { name: 'HERALD_AGENT_MODEL_NAME', value: cognitiveModelName }           // <-- NEW
  { name: 'HERALD_AGENT_API_VERSION', value: '2025-04-01-preview' }
```

### `deploy/main.json`

The `baseEnvVars` collection appears twice as a flattened
`concat(createArray(...), variables('authEnvVars'), variables('authorityEnvVars'))`
expression — once for the Container App path and once for the App Service
path. In each, insert a new `createObject` immediately after the
`HERALD_AGENT_DEPLOYMENT` entry:

```
createObject('name', 'HERALD_AGENT_DEPLOYMENT', 'value', reference(...).outputs.modelDeploymentName.value),
createObject('name', 'HERALD_AGENT_MODEL_NAME', 'value', parameters('cognitiveModelName')),   // <-- NEW
createObject('name', 'HERALD_AGENT_API_VERSION', 'value', '2025-04-01-preview'),
```

Grep to confirm: `grep -c HERALD_AGENT_MODEL_NAME deploy/main.json` → `2`.

### `deploy/main.parameters.json`

Bump the content version only. Do **not** copy any other field from the
commercial tree — values like `tenantId`, `entraClientId`, and
`cognitiveCustomDomain` are IL6-specific and must remain as the IL6 operator
has them configured.

```diff
- "contentVersion": "0.4.2.0",
+ "contentVersion": "0.5.0.0",
```

If a future delta adds or renames a parameter, that will be called out
explicitly in this section. For this release no parameters are added.

## IL6-Specific Parameter Values

A reminder for every delta: the commercial `main.parameters.json` is not
authoritative for IL6. Only the `contentVersion` bump and any newly added
parameters propagate. Sensitive or environment-specific values (tenant ID,
client ID, cognitive custom domain, ACR name, container image path, postgres
admin login, etc.) remain IL6-local.

## Verification

Run on the IL6 host after applying the edits:

```bash
grep -n HERALD_AGENT_MODEL_NAME deploy/main.json        # expect two hits
jq -r '.contentVersion' deploy/main.parameters.json     # expect "0.5.0.0"
```

Post-deploy, confirm the Container App has the new env var:

```bash
az containerapp show -g <rg> -n <app> --query "properties.template.containers[0].env[?name=='HERALD_AGENT_MODEL_NAME']"
```

Re-classify a document from the web UI and confirm the footer renders as
`gpt-5-mini / azure  Classified [date]` (or whatever `cognitiveModelName`
resolves to on IL6).

## Rollback

If the classification view regresses after deploy, re-deploy the prior
`0.4.2.0` ARM template. The Go binary is unaffected by this change — the
review footer will revert to rendering the empty model segment but no
classification or persistence behavior is impacted.
```

## Validation Criteria

- [ ] `grep HERALD_AGENT_MODEL_NAME deploy/main.bicep` returns one hit.
- [ ] `grep -c HERALD_AGENT_MODEL_NAME deploy/main.json` returns `2`.
- [ ] `grep HERALD_AGENT_MODEL_NAME deploy/README.md deploy/update.md` returns hits in both.
- [ ] `jq -r '.contentVersion' deploy/main.parameters.json` returns `0.5.0.0`.
- [ ] `az bicep build --file deploy/main.bicep --outfile /tmp/main.check.json && diff -q deploy/main.json /tmp/main.check.json` — no differences.
- [ ] `mise run vet` passes.
- [ ] `mise run test ./tests/config/...` passes (existing coverage, no new tests expected).
- [ ] Local `mise run dev` still renders `gpt-5-mini / azure  Classified [date]` in the review footer (nothing in the hot path changed).
