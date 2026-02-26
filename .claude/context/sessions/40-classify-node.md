# 40 — Classify Node

## Summary

Implemented the classify node for sequential page-by-page analysis with context accumulation. Introduced the 4-node workflow topology (init → classify → enhance? → finalize) by separating per-page analysis from document-level classification synthesis. Updated the prompts domain to add the `finalize` stage (stages, specs, instructions, migration) and aligned the classify/enhance specs with the optimized workflow types.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| 4-node workflow topology | init → classify → enhance? → finalize | Separating per-page analysis (classify) from document-level synthesis (finalize) eliminates incremental anchoring bias and ensures finalize sees all evidence holistically, including any enhanced pages |
| Per-page-only classify response | Response schema matches `ClassificationPage` fields only | Document-level classification deferred to finalize node; classify focuses on what's visible per page |
| Enhancements field purpose | Classify node describes what adjustments the enhance node should apply | Provides actionable context for the enhance node (e.g., "faded banner markings — increase brightness and contrast") |
| Agent creation in classifyPages | Moved from ClassifyNode to classifyPages | Agent is only used within the page loop; keeps ClassifyNode focused on state extraction and storage |
| Spec alignment with types | Updated specs to use `ClassificationPage` field names | Eliminated field name mismatches (image_quality_limiting → enhance) and removed obsolete document-level fields from classify spec |
| No unit tests for ClassifyNode | Skip | Requires mocking agent, prompts.System, storage, and real Vision API calls — not worth the overhead |

## Files Modified

- `workflow/classify.go` — New: ClassifyNode, pageResponse, classifyPages, classifyPage, encodePageImage, applyPageResponse
- `internal/prompts/stages.go` — Added StageFinalize constant
- `internal/prompts/specs.go` — Rewrote classifySpec (page-only), updated enhanceSpec, added finalizeSpec
- `internal/prompts/instructions.go` — Added finalizeInstructions
- `cmd/migrate/migrations/000005_prompts_add_finalize_stage.up.sql` — New: adds finalize to CHECK constraint
- `cmd/migrate/migrations/000005_prompts_add_finalize_stage.down.sql` — New: reverse migration
- `_project/README.md` — Updated workflow topology, node descriptions, key decisions
- `_project/objective.md` — Updated #41 scope and architecture decisions for 4-node topology
- `tests/prompts/prompts_test.go` — Updated for StageFinalize
- `tests/prompts/handler_test.go` — Updated for StageFinalize

## Patterns Established

- **Per-page-only node pattern**: Classify populates `ClassificationPage` fields per page; document-level `ClassificationState` fields are set by a downstream finalize node. This separation applies to enhance as well.
- **Just-in-time image encoding**: PNG read from disk, encoded to data URI, bytes released. One page image in memory at a time.
- **Context accumulation via ComposePrompt**: First page passes nil (no context), subsequent pages pass &classState so the model sees prior page findings.

## Validation Results

- `go vet ./...` — clean
- `go build ./...` — clean
- `go test ./tests/...` — all pass (prompts tests updated for StageFinalize)
