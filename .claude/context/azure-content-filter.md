# Azure Content Filter Configuration (required for the gpt-5.2 deployment)

## Problem

During prompt validation, the **finalize** chat call was hard-rejected by Azure's Responsible-AI
content filter:

```
HTTP 400 ResponsibleAIPolicyViolation
content_filter_result: { "hate": { "filtered": true, "severity": "medium" } }
```

The filter flagged the **prompt** (`"param": "prompt"`) on the **hate** category at **medium**
severity. Azure's default content filter blocks everything at **medium or high** severity.

## Root cause — not a prompt defect

This is an inherent mismatch between a generic Responsible-AI classifier and a **legitimate,
official-policy classification-marking workload**. Herald's prompts and the documents it reads
necessarily contain DoD marking vocabulary that a generic "hate"/"violence" classifier misreads as
discriminatory or harmful content:

- `NOFORN` = "Not Releasable to Foreign Nationals" — nationality-based access control.
- `REL TO <countries>` / `RELIDO` / `DISPLAY ONLY` — release restrictions by nationality.
- AEA / `RESTRICTED DATA`, and declass exemption categories (`50X2-WMD`, nuclear command/control).

The denser, policy-complete `finalizeSpec` (DoDM 5200.01 full rule set) crossed the medium
threshold where the shorter prior prompt did not. **This is expected to recur on genuinely
release-control-heavy documents regardless of prompt wording** — the markings themselves carry the
flagged language. We deliberately did NOT dumb down the policy-accurate prompts; the correct fix is
at the content-filter configuration layer.

## Fix (self-service on commercial Azure — no approval required)

Source: [Configure content filters — Microsoft Learn](https://learn.microsoft.com/en-us/azure/ai-foundry/openai/how-to/content-filters).

All customers can raise a category's severity threshold to **"High"** (block only high-severity;
allow low + medium). Only **"No filters"** and **"Annotate only"** require the gated
[Limited Access: Modified Content Filters](https://ncv.microsoft.com/uEfCgnITdR) approval — we do
**not** need that.

Threshold options per category (Microsoft's exact names):

| Setting | Effect |
|---|---|
| Low, medium, high | strictest — filters low+medium+high |
| Medium, high | **default** — filters medium+high |
| **High** | filters only high; allows low + medium ← **use this** |
| No filters / Annotate only | requires approval — not needed here |

### Steps (Foundry classic portal)

1. Sign in to [Microsoft Foundry](https://ai.azure.com/) (New Foundry toggle **off** → classic).
   Navigate to the project for the `herald-ai-prod` resource.
2. **Guardrails + controls** → **Content filters** tab → **+ Create content filter**.
3. **Basic information:** name it (e.g., `herald-classification-filter`); select the connection for
   the `herald-ai-prod` resource. Next.
4. **Input filters (prompts):** set **Hate → High** at minimum. Recommended for this workload: set
   **all four harm categories (Hate, Violence, Sexual, Self-harm) → High**, because classification
   content can also brush Violence (nuclear/WMD/weapons exemption categories). Leave Prompt Shields
   (jailbreak) **on**. Next.
5. **Output filters (completions):** set the same thresholds (**High**). The finalize rationale and
   per-page rationales describe release controls, so the output can trip the same classifiers. Next.
6. **Connection:** associate the filter with the **`gpt-5.2`** deployment (or assign later via
   **Models + endpoints → select gpt-5.2 → Edit → choose the filter → Save**). Create.

### Important caveats

- **Deployment-level only for our use.** Azure also supports a per-request `x-policy-id` header to
  pick a filter at request time, **but it is not available for image-input (vision) scenarios** —
  Herald's classify/enhance calls send images, so they always use the deployment-level filter.
  Therefore the custom filter **must be assigned at the deployment level**, not per request.
- **"High" still blocks genuinely high-severity content** — this preserves real safety while
  allowing legitimate medium-severity marking text. It is not "turning off" the filter.
- **Justification (for change records):** Herald processes official DoD security-classification
  markings per DoDM 5200.01; the flagged terminology is legitimate marking vocabulary, not harmful
  content. Raising thresholds to High is the documented, self-service mitigation for this class of
  false positive.
- **Production reach:** apply the same configuration to any other deployment/region Herald uses
  (and the IL6/Gov deployment separately — Azure Government uses a different modified-filter form:
  https://aka.ms/AOAIGovModifyContentFilter — though threshold raises there are likewise expected to
  be self-service; confirm at deploy time).

## Status

- **Codified in IaC as of v0.6.0.** The policy is now provisioned by
  `deploy/modules/cognitive.bicep` as the `herald-content-filter` RAI policy (`basePolicyName:
  Microsoft.DefaultV2`, harm categories at `severityThreshold: High`/`blocking: true` on Prompt and
  Completion) and assigned to the model deployment via `raiPolicyName`. The manual portal steps above
  remain the **portal-equivalent reference** and the **fallback** for any environment where the
  `raiPolicies` resource type is unavailable.
- **Commercial deployment** (`herald-ai` / `heraldgroup`): the live `herald-content-filter` policy was
  configured manually first (this session) and now matches the Bicep; a Bicep deploy is an idempotent
  reconcile.
- **IL6 / Azure Government:** `raiPolicies@2025-09-01` availability in the Gov region must be verified
  at the next bicep update (per `deploy/update.md`); if unsupported, configure manually via the steps
  above and disable the `raiPolicy` resource locally.
- **Future enhancement:** none outstanding — the deferred "codify in Bicep" item is now done.
