# IL6 Deployment Update Recipe

Until a CI/CD process for IaC substitution is in place, each deploy-touching
PR must be re-applied by hand to the IL6-side deploy tree. `deploy/main.json`
is a gitignored build artifact — the IL6 operator regenerates it locally with
`bicep.exe build deploy\main.bicep` (prerequisite per [il6.md](il6.md)). The
operator also maintains their own `deploy/main.parameters.json` with
IL6-specific sensitive values that must not be overwritten.

This file is the canonical delta log. It is rewritten on every PR that
modifies `deploy/` so a single authoritative patch is always available to the
operator. When an older delta ships (e.g., the operator is catching up from
`0.4.2.0` → `0.5.1.0`), apply each intermediate `contentVersion` in order.

## Target Version

`0.5.0.0` — ships with the Phase 5 release (`v0.5.0`). Introduced alongside
issue [#140](https://github.com/JaimeStill/herald/issues/140) (model name
missing from review footer).

## What Changed

### `deploy/main.bicep`

One new env var added to the `baseEnvVars` list so the Container App injects
`HERALD_AGENT_MODEL_NAME` into the running container. Without it, the classify
workflow populates an empty `model_name` column and the review-view footer
renders `/ azure  Classified …` with no model segment.

Anchor lines (before / after in the `baseEnvVars` block):

```bicep
  { name: 'HERALD_AGENT_DEPLOYMENT', value: cognitive.outputs.modelDeploymentName }
  { name: 'HERALD_AGENT_MODEL_NAME', value: cognitiveModelName }    // <-- NEW
  { name: 'HERALD_AGENT_API_VERSION', value: '2025-04-01-preview' }
```

`cognitiveModelName` is the existing parameter (default `gpt-5-mini`) already
wired into the cognitive module. It is semantically distinct from
`cognitiveDeploymentName` — they coincide in value today but are bound
separately because the Azure deployment name and the model identifier are
independent concerns.

### `deploy/main.parameters.json` (commercial reference)

```diff
- "contentVersion": "0.4.2.0",
+ "contentVersion": "0.5.0.0",
```

**IL6 operators:** apply the same `contentVersion` bump to your IL6-local
parameters file. Do **not** copy any other field from the commercial file —
values like `tenantId`, `entraClientId`, `cognitiveCustomDomain`,
`authAuthority`, `postgresTokenScope`, `cognitiveTokenScope`, and
`containerImage` are IL6-specific and must remain as you have them configured.

**No new parameters** are introduced in this delta, so no additions to the
IL6 parameters file are required.

### `deploy/main.json`

Gitignored build artifact. Regenerate locally after pulling the updated
`main.bicep`:

```powershell
bicep.exe build deploy\main.bicep
```

Expected: two new `HERALD_AGENT_MODEL_NAME` entries (one per Container App
path, one per App Service path) bound to `parameters('cognitiveModelName')`.

## Application Impact

None. The compiled `/herald` binary is unchanged — no Go source, frontend, or
Dockerfile edits. The existing image already reads `HERALD_AGENT_MODEL_NAME`
via `os.Getenv` during `FinalizeAgent`; adding the env var simply lets the
runtime see a value instead of an empty default. A fresh image is **not**
required — redeploying the updated ARM template against the current image is
sufficient.

## Verification

Before deploy, on the IL6 host:

```powershell
Select-String -Path deploy\main.json -Pattern 'HERALD_AGENT_MODEL_NAME'
# expect two matches

(Get-Content deploy\main.parameters.json | ConvertFrom-Json).contentVersion
# expect 0.5.0.0
```

After deploy, confirm the Container App picked up the env var:

```powershell
az containerapp show `
  --resource-group <resource-group> `
  --name herald-app `
  --query "properties.template.containers[0].env[?name=='HERALD_AGENT_MODEL_NAME']"
```

Re-classify a document from the Herald web UI and confirm the classification
panel footer renders `<model> / azure  Classified …` (where `<model>` matches
the `cognitiveModelName` parameter value).

## Rollback

If the review footer regresses, redeploy the prior `0.4.2.0` ARM template.
The running binary is unaffected by this change — only the Container App env
block moves — so no image rollback is required.
