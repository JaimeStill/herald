# 41 — Enhance Node, Finalize Node, Graph Assembly, and Execute Function

## Summary

Completed the classification workflow by adding the enhance node (re-render + reclassify), finalize node (document-level synthesis via Chat), state graph assembly with conditional enhancement edge, and the top-level `Execute` function with temp directory lifecycle management. Also introduced `EnhanceSettings` as a structured type replacing the previous `Enhance bool` + `Enhancements string` fields, and updated the classify/enhance prompt specs to match.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Enhancement signal | `Enhancements *EnhanceSettings` (nil = no enhancement) instead of separate `Enhance bool` + `Enhancements string` | Single source of truth — no risk of bool/data inconsistency. Classify agent outputs structured rendering parameters directly. |
| EnhanceSettings fields | `Brightness *int`, `Contrast *int`, `Saturation *int` | Maps directly to document-context's `ImageMagickConfig` filter fields. Pointer types distinguish "not set" from "neutral value". |
| Enhance node role | Hybrid rendering + inference | Re-renders flagged pages programmatically from structured settings (no LLM needed for rendering params), then reclassifies via vision to update per-page findings. |
| Finalize inference | `agent.Chat` (not Vision) | Finalize reviews text data (all per-page findings as serialized JSON context), not images. |
| Graph observer | `"noop"` | Herald explicitly excludes observer/checkpoint infrastructure per architectural decisions. |
| `Enhance()` convenience method | On `ClassificationPage` | Centralizes `Enhancements != nil` check — used by log lines, `NeedsEnhance()`, and `EnhancePages()`. |

## Files Modified

- `workflow/types.go` — added `EnhanceSettings`, `ClassificationPage.Enhance()`, replaced `Enhance bool` + `Enhancements string` with `Enhancements *EnhanceSettings`, updated `NeedsEnhance()` and `EnhancePages()`
- `workflow/errors.go` — added `ErrFinalizeFailed`, updated package doc
- `workflow/classify.go` — updated `pageResponse`, `applyPageResponse`, log line to use `Enhance()`
- `workflow/enhance.go` — new file: `EnhanceNode`, `enhancePages`, `enhancePage`, `rerender`, `buildEnhanceConfig`, `extractTempDir`
- `workflow/finalize.go` — new file: `FinalizeNode`, `synthesize`
- `workflow/workflow.go` — new file: `Execute`, `buildGraph`, `needsEnhance` predicate, `extractResult`
- `internal/prompts/specs.go` — updated classify spec (structured `enhancements` output), updated enhance spec (markings_found + rationale only)
- `internal/prompts/instructions.go` — updated enhance instructions wording
- `config.json` — added chat capability to model config
- `tests/workflow/types_test.go` — updated for new type shape, added `TestEnhance`, `TestEnhanceSettingsJSON`
- `tests/workflow/prompts_test.go` — added `StageFinalize` to mock, added finalize stage test

## Patterns Established

- **Structured enhancement settings**: Classify agent outputs rendering parameters as a typed struct rather than free text, eliminating an LLM interpretation step from the enhance pipeline
- **Convenience predicate method**: `ClassificationPage.Enhance()` centralizes the nil-check pattern for enhancement flagging
- **Hybrid node pattern**: Enhance node combines rendering (programmatic from structured settings) with inference (vision reclassification) in a single node

## Validation Results

- `go vet ./...` — clean
- `go build ./...` — clean
- `go test ./tests/...` — all 17 packages pass (17/17 workflow tests pass)
