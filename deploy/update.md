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
`0.5.0.0` → `0.6.0.0`), apply each intermediate `contentVersion` in order.

## Target Version

`0.6.0.0` — ships with the Phase 6 release (`v0.6.0`).

## What Changed

### Model & platform migration (`main.bicep` + `cognitive.bicep`)

The cognitive configuration migrated to the Azure AI Foundry deployment format
since v0.5.0. This was not captured in a prior delta, so apply it now if your
IL6 tree is still on the v0.5.0 (`OpenAI` / `gpt-5-mini`) shape.

- **`cognitive.bicep` — `kind` (structural):** `OpenAI` → `AIServices`. Apply to
  your IL6 `cognitive.bicep` source — it is a hardcoded param default, not a
  value you override via the parameters file:

  ```bicep
  @description('Cognitive Services kind')
  param kind string = 'AIServices'    // was 'OpenAI'
  ```

- **Model deployment params** (defaults in both `main.bicep` and
  `cognitive.bicep`): `gpt-5-mini` → `gpt-5.2`, version `2025-08-07` →
  `2025-12-11`. These are the **commercial reference** defaults — **IL6 governs
  the actual model through its parameters file.** Confirm `gpt-5.2` /
  `2025-12-11` availability in your Gov region and set
  `cognitiveDeploymentName` / `cognitiveModelName` / `cognitiveModelVersion`
  accordingly (substitute an available model if `gpt-5.2` is not yet in Gov).
  Source-default change shown for parity:

  ```bicep
  param cognitiveDeploymentName string = 'gpt-5.2'    // was 'gpt-5-mini'
  param cognitiveModelName string = 'gpt-5.2'         // was 'gpt-5-mini'
  param cognitiveModelVersion string = '2025-12-11'   // was '2025-08-07'
  ```

### `deploy/main.bicep`

**Three** env vars were added to the `baseEnvVars` list since v0.5.0 — apply all
three. (`HERALD_AGENT_CAPABILITIES_*` arrived with the confidence-overhaul work;
`HERALD_LOG_LEVEL` is new this release.)

**1. Log level** — explicit deployed log level. The app defaults to `info` when
unset, so this is a make-it-explicit override (set `debug` only for transient
diagnostics):

```bicep
  { name: 'HERALD_SERVER_PORT', value: '8080' }
  { name: 'HERALD_LOG_LEVEL', value: 'info' }    // <-- NEW
  { name: 'HERALD_DB_HOST', value: postgres.outputs.fqdn }
```

**2. Agent capabilities** — the model-agnostic capability channel; without
these, the deployed container does not receive the configured
completion-token / reasoning-effort / vision-detail settings. Add both lines
between `HERALD_AGENT_CLIENT_ID` and `AZURE_CLIENT_ID`:

```bicep
  { name: 'HERALD_AGENT_CLIENT_ID', value: identity.outputs.clientId }
  { name: 'HERALD_AGENT_CAPABILITIES_CHAT', value: '{"max_completion_tokens":4096,"reasoning_effort":"high"}' }    // <-- NEW
  { name: 'HERALD_AGENT_CAPABILITIES_VISION', value: '{"max_completion_tokens":4096,"reasoning_effort":"high","vision_options":{"detail":"high"}}' }    // <-- NEW
  { name: 'AZURE_CLIENT_ID', value: identity.outputs.clientId }
```

### `deploy/modules/cognitive.bicep`

A custom Responsible AI content-filter policy (`herald-content-filter`) is now
provisioned in IaC and assigned to the model deployment. It raises the four
harm-category thresholds to **High** (block only high severity; low/medium
pass) so legitimate DoD marking vocabulary — NOFORN ("Not Releasable to Foreign
Nationals"), `REL TO <countries>`, AEA/Restricted Data, and nuclear/WMD
declassification-exemption terms — is not misclassified as harmful and rejected
by the default (medium-threshold) filter. This previously had to be configured
by hand; it is now codified. Apply the three additions below verbatim.

**(a)** New parameter — add alongside the other deployment params (after
`deploymentSkuCapacity`, before `tags`):

```bicep
@description('Responsible AI content-filter policy name applied to the model deployment')
param raiPolicyName string = 'herald-content-filter'
```

**(b)** New resource — add between the `account` resource and the `deployment`
resource (paste the entire block):

```bicep
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
```

**(c)** New deployment property — add `raiPolicyName` to the existing
`deployment` resource's `properties` block:

```bicep
  properties: {
    model: {
      format: modelFormat
      name: modelName
      version: modelVersion
    }
    raiPolicyName: raiPolicy.name    // <-- NEW
  }
```

> **IL6 / Azure Government verification REQUIRED.** Confirm the
> `Microsoft.CognitiveServices/accounts/raiPolicies@2025-09-01` resource type is
> available in your Gov region and that the deployment accepts `raiPolicyName`.
> If the Gov cloud does not support the resource at this API version, configure
> the equivalent filter **manually** in the AI Foundry portal (all four harm
> categories → "Lowest blocking"/High on input *and* output; keep Jailbreak on
> and Protected Material Text blocking), and comment out the `raiPolicy`
> resource + the `raiPolicyName` deployment property in your local `main.bicep`
> until the resource type is supported. Without one of these, release-control-
> heavy documents (NOFORN/REL TO) will be rejected with an RAI `400`.

### `deploy/main.parameters.json` (commercial reference)

```diff
- "contentVersion": "0.5.0.0",
+ "contentVersion": "0.6.0.0",
```
```diff
- "value": "heraldregistry.azurecr.io/herald:0.5.0"
+ "value": "heraldregistry.azurecr.io/herald:0.6.0"
```

**IL6 operators:** bump your IL6-local `contentVersion` to `0.6.0.0` and your
`containerImage` tag to the v0.6.0 image you push to the IL6 ACR (see
Application Impact — a fresh image is required this release). Do **not** copy
any other field from the commercial file — values like `tenantId`,
`entraClientId`, `cognitiveCustomDomain`, `authAuthority`, `postgresTokenScope`,
`cognitiveTokenScope`, and `containerImage` are IL6-specific. The new
`raiPolicyName` parameter has a default and needs no entry unless you use a
different policy name.

### `deploy/main.json`

Gitignored build artifact. Regenerate locally after pulling the updated
`main.bicep` / `cognitive.bicep`:

```powershell
bicep.exe build deploy\main.bicep
```

Expected: the new env-var entries flattened into both compute paths (Container
App + App Service) — `HERALD_LOG_LEVEL`, `HERALD_AGENT_CAPABILITIES_CHAT`, and
`HERALD_AGENT_CAPABILITIES_VISION` (two occurrences each) — plus a new
`Microsoft.CognitiveServices/accounts/raiPolicies` resource whose name the
deployment's `raiPolicyName` references.

## Application Impact

**A fresh container image IS required this release** — unlike the env-only
v0.5.0 delta. v0.6.0 changes the `/herald` binary:

- dependency bump `agent v0.1.1 → v0.1.2` (broadened HTTP retry policy: now
  retries `408`, `429`, and all `5xx` — transient Azure `500`s no longer abort
  a multi-page classification),
- classification prompt rewrite (policy-aligned banner assembly + banner-
  position scrutiny),
- render contrast normalization, per-call token-usage logging, and the new
  `HERALD_LOG_LEVEL` config field.

Build, tag, and push the v0.6.0 image to the IL6 ACR, then set the
`containerImage` tag before deploying the ARM template.

## Verification

Before deploy, on the IL6 host:

```powershell
Select-String -Path deploy\main.json -Pattern 'HERALD_LOG_LEVEL'              # expect 2 matches
Select-String -Path deploy\main.json -Pattern 'HERALD_AGENT_CAPABILITIES_VISION'  # expect 2 matches
Select-String -Path deploy\main.json -Pattern 'raiPolicies'                  # expect the RAI policy resource

(Get-Content deploy\main.parameters.json | ConvertFrom-Json).contentVersion
# expect 0.6.0.0
```

After deploy, confirm the env var and the RAI policy assignment (Gov ARM
endpoint is `management.usgovcloudapi.net`):

```powershell
az containerapp show `
  --resource-group <resource-group> `
  --name herald-app `
  --query "properties.template.containers[0].env[?name=='HERALD_LOG_LEVEL']"

az rest --method get `
  --url "https://management.usgovcloudapi.net/subscriptions/<sub>/resourceGroups/<rg>/providers/Microsoft.CognitiveServices/accounts/<account>/deployments/<deployment>?api-version=2025-09-01" `
  --query "properties.raiPolicyName"
# expect: herald-content-filter
```

Then classify a release-control-heavy document (one bearing NOFORN / REL TO) and
confirm finalize completes — no `400` `ResponsibleAIPolicyViolation` — and the
banner assembles correctly.

## Rollback

Redeploy the prior `0.5.0.0` ARM template **and** revert the `containerImage`
tag to the v0.5.0 image — because v0.6.0 ships a new image, image rollback IS
required this release (unlike the v0.5.0 env-only delta). The `0.5.0.0` template
does not define the RAI policy; the `raiPolicies` resource is additive and
harmless to leave in place on the account, but a `0.5.0.0` redeploy will not
unassign it from the deployment — verify `properties.raiPolicyName` afterward if
you need the prior filter posture restored.
