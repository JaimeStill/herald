# 140 - Model name missing from review footer in production

## Summary

Deployment-layer fix for the review-view footer rendering `/ azure  Classified …` on IL6 (model segment empty) while local `mise run dev` rendered `gpt-5-mini / azure …` correctly. Root cause: the Dockerfile ships only the compiled binary — `config.json` is not bundled — so production relies entirely on `HERALD_*` env vars, and `deploy/main.bicep` omitted `HERALD_AGENT_MODEL_NAME`. Added the env var to the Container App's `baseEnvVars`, bound to the existing `cognitiveModelName` parameter, and authored `deploy/update.md` as the canonical manual-patch recipe for the air-gapped IL6 host.

No application code changed. The running container picks up the fix on the next ARM redeploy — no new image required.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Scope | Deployment-only (no Go changes) | Full `HERALD_*` env-var audit showed `HERALD_AGENT_MODEL_NAME` was the only variable without a tangible code default that was missing from Bicep. The Go config ordering is already correct and covered by `TestAgentEnvOverrides`. |
| Env var source | `cognitiveModelName` param (not `cognitive.outputs.modelDeploymentName`) | The model identifier and the Azure deployment name are semantically distinct. `HERALD_AGENT_DEPLOYMENT` already binds to the deployment output; keeping `HERALD_AGENT_MODEL_NAME` bound to the parameter maintains the separation even though both values happen to equal `gpt-5-mini` today. |
| Parameter content version | `0.4.2.0` → `0.5.0.0` | Rolls with the Phase 5 release rather than cutting a Phase-5 dev iteration. |
| Dev tag / CHANGELOG | Skipped | Container image is byte-identical, so a `v0.5.0-dev.132.140` tag would orphan itself. Captured as a memory rule for future deploy-only sessions. |
| `deploy/update.md` scope | Target compile-and-merge workflow, not hand-edit main.json | `deploy/main.json` is gitignored (build artifact); IL6 operators have `bicep.exe` as a prerequisite and regenerate it locally. The recipe focuses on what changed in `main.bicep` + `main.parameters.json` so the operator knows what to re-compile and what to merge into their IL6-local parameters file. |

## Files Modified

- `deploy/main.bicep` — added one `HERALD_AGENT_MODEL_NAME` entry to `baseEnvVars` (bound to `cognitiveModelName`).
- `deploy/main.parameters.json` — bumped `contentVersion` from `0.4.2.0` to `0.5.0.0`.
- `deploy/README.md` — added one row to the env var reference table.
- `deploy/update.md` — **new**, canonical manual-patch recipe for the air-gapped IL6 deploy host.
- `.claude/plans/immutable-squishing-forest.md` — session plan (preserved per project convention).
- `.claude/context/guides/.archive/140-model-name-missing-review-footer.md` — archived implementation guide.
- `.claude/context/sessions/140-model-name-missing-review-footer.md` — this summary.

`deploy/main.json` is a gitignored build artifact regenerated locally via `az bicep build`; not tracked by git.

## Patterns Established

- **Deploy-only PRs skip the dev release tag.** When every changed path is under `deploy/` (or docs/session artifacts) and the container image is byte-identical, Phase 8c (CHANGELOG bump) and the dev tag are skipped — the Container App picks up the fix on the next ARM redeploy with no new image. Saved to memory as `feedback_deploy_only_no_dev_tag.md`.
- **`deploy/update.md` as canonical delta log.** Overwritten on every deploy-touching PR. The IL6 operator applies the listed `main.bicep` / parameter edits, runs `bicep.exe build`, merges the contentVersion bump into their IL6-local parameters file, and redeploys. Sensitive values (tenant/client IDs, token scopes, custom domains, container image path) remain IL6-local and are explicitly excluded from the recipe.
- **Full env-var audit before deployment fixes.** Rather than patching only the reported symptom, walk every `HERALD_*` env var consumed by the code against `baseEnvVars` in `main.bicep`, categorizing each as (a) injected, (b) backed by a production-viable code default, (c) mode-incompatible (e.g., token-based auth when running with managed identity), or (d) missing. Only category (d) requires a fix.

## Validation Results

- `mise run vet` — clean.
- `mise run test ./tests/...` — all packages pass (existing `TestAgentEnvOverrides` at `tests/config/config_test.go:477-516` already covers the env-override path; no new tests required).
- `grep -c HERALD_AGENT_MODEL_NAME deploy/main.{bicep,json} deploy/README.md deploy/update.md` — `1 / 2 / 1 / 6` (expected: `main.json` has one entry per emitted ARM path, Container App + App Service).
- `jq -r '.contentVersion' deploy/main.parameters.json` — `0.5.0.0`.
- Local `mise run dev` footer behavior unchanged — no hot path touched.
- Post-deploy verification (out of scope for merge): IL6 review footer renders `<model> / azure  Classified …` for a freshly classified document.
