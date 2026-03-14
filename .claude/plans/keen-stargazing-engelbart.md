# Herald Infrastructure Performance Adjustments

## Context

Herald's classification workflow silently fails during bulk classification (3 concurrent documents) on Azure Container Apps. The Container App is provisioned at 1.0 CPU / 2Gi memory. Concurrent ImageMagick/Ghostscript renders can exceed 2Gi, triggering OOM kills that surface as silent replica restarts with no application-level error. Increasing compute resources addresses the root cause.

The original request included adding a configurable ingress timeout, but research confirmed that request timeout is **not configurable at the app level** — it requires Premium Ingress (environment-level, D4+ workload profile, min 2 nodes). Since SSE streams actively send data and shouldn't hit the idle timeout, and the silent failures are OOM kills rather than timeouts, this change is dropped.

## Changes

### 1. Increase Container App resource defaults

**`deploy/modules/app.bicep`** (lines 25-29):
- `cpu` default: `'1.0'` → `'2.0'`
- `memory` default: `'2Gi'` → `'4Gi'`

**`deploy/main.bicep`** (lines 69-73):
- `containerCpu` default: `'1.0'` → `'2.0'`
- `containerMemory` default: `'2Gi'` → `'4Gi'`

`deploy/main.parameters.json` does not override these values — no change needed there.

### 2. Update deploy README

**`deploy/README.md`**:
- Line 55: Update "Default resources: 1.0 CPU, 2Gi memory" → "2.0 CPU, 4Gi memory"
- Lines 130-131: Update parameter table defaults (`1.0` → `2.0`, `2Gi` → `4Gi`)
- Add a note about the 240s default ingress idle timeout and Premium Ingress as an option if longer idle periods are needed

## Verification

1. `az deployment group what-if` to confirm the resource changes propagate correctly
2. After deploy, check system logs for OOM evidence from previous runs:
   ```bash
   az containerapp logs show --name herald --resource-group <rg> --type system --tail 100
   ```
3. Run bulk classification of 3 documents and confirm all SSE streams complete
