# Objective #26 — Classification Workflow

**Phase:** [Phase 2 — Classification Engine](phase.md)
**Issue:** [#26](https://github.com/JaimeStill/herald/issues/26)
**Milestone:** v0.2.0 - Classification Engine

## Scope

Implement the `workflow/` package containing the 4-node state graph (init → classify → enhance? → finalize), all node implementations, prompt composition, and response parsing. The workflow adapts classify-docs' sequential page-by-page context accumulation pattern (96.3% accuracy) into a go-agents-orchestration state graph. The classify node handles per-page analysis; a dedicated finalize node synthesizes the document-level classification from all page findings.

**Out of scope**: Database persistence of classification results, HTTP endpoints, document status transitions.

## Sub-Issues

| # | Sub-Issue | Status | Dependencies |
|---|-----------|--------|--------------|
| [#37](https://github.com/JaimeStill/herald/issues/37) | Prompts domain extensions — instructions, specifications, and hardcoded defaults | Open | — |
| [#38](https://github.com/JaimeStill/herald/issues/38) | Workflow foundation — types, runtime, errors, and parsing | Open | #37 |
| [#39](https://github.com/JaimeStill/herald/issues/39) | Init node — concurrent page rendering with temp storage | Open | #38 |
| [#40](https://github.com/JaimeStill/herald/issues/40) | Classify node — sequential page-by-page context accumulation | Open | #38 |
| [#41](https://github.com/JaimeStill/herald/issues/41) | Enhance node, finalize node, graph assembly, and Execute function | Open | #39, #40 |

## Architecture Decisions

1. **Prompts domain owns all prompt content**: The prompts domain is the single source of truth for both tunable instructions (DB overrides or hardcoded defaults) and immutable specifications (output schema + behavioral constraints). `Instructions(ctx, stage)` always returns a non-null string. `Spec(ctx, stage)` returns the read-only specification. The workflow composes both into the final system prompt.

2. **Specifications replace "output format"**: The immutable traits of each prompt stage are called "specifications" — they define the expected JSON output structure and behavioral constraints that the workflow parser depends on. Exposed via `GET /api/prompts/{stage}/spec` as read-only context for prompt authors crafting instructions.

3. **Request-bound temp storage**: Page images are rendered to a temp directory (created by `Execute`, cleaned up via defer) rather than held as base64 data URIs in memory. `PageImage` stores a file path. Classify/enhance nodes encode to data URI just-in-time per page, keeping memory usage proportional to one page at a time.

4. **Concurrent rendering, sequential classification**: The init node renders pages concurrently (ImageMagick is CPU-heavy, bounded concurrency via `errgroup.SetLimit`). The classify node processes pages sequentially for context accumulation — each page's findings feed the next page's prompt. Preserves the 96.3% accuracy pattern from classify-docs while optimizing the rendering bottleneck.

5. **Inline sequential processing**: Herald implements the classify-docs `ProcessWithContext` pattern inline. A simple `for range pages` loop with state accumulation is clearer for a single workflow.

6. **Per-page analysis, document-level synthesis**: The classify node only populates `ClassificationPage` fields per page. A dedicated finalize node performs a single inference to synthesize the document-level `ClassificationState` (classification, confidence, rationale) from all page findings. This eliminates incremental anchoring bias and ensures finalize sees all evidence — including any enhanced pages — holistically.

7. **Single exit point**: Finalize is always the terminal node. The conditional edge on `ClassificationState.NeedsEnhance()` determines whether enhance runs before finalize, but both paths converge to finalize. Initially, classify never sets `Enhance: true`, so enhance is skipped.

8. **Per-request agent creation**: Each `Execute` call creates a fresh `agent.Agent` from the config. Stateless agent design — no lifecycle management.
