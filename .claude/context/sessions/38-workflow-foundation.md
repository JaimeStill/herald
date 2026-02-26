# 38 — Workflow Foundation: Types, Runtime, Errors, and Parsing

## Summary

Established the foundational `workflow/` package with shared types, a runtime dependency struct, sentinel errors, generic JSON parsing, and prompt composition. Added `go-agents-orchestration` and `document-context` dependencies. All subsequent workflow sub-issues (#39–#41) build on these definitions.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Type consolidation | Two core types (`ClassificationState`, `ClassificationPage`) instead of four | Reduces token consumption per LLM response; per-page classification/confidence are intermediate values that don't need persistence |
| Enhancement methods | `NeedsEnhance()` and `EnhancePages()` on `ClassificationState` | Replaces standalone `QualityAssessment` struct; derives enhancement decisions from page data |
| `NeedsEnhance` implementation | `slices.ContainsFunc` | Idiomatic Go; short-circuits on first match |
| `EnhancePages` implementation | Range loop collecting indices | No standard library function for index-collecting filter |
| JSON parser location | `pkg/formatting/parse.go` | Generic utility alongside existing `bytes.go`; not workflow-specific |
| Workflow Runtime struct | Separate from Infrastructure | Infrastructure represents cold-start systems; workflow Runtime bundles the specific subset of dependencies nodes need, including domain systems created after Infrastructure |

## Files Modified

- `workflow/errors.go` — new: 4 sentinel errors, package doc comment
- `workflow/types.go` — new: `Confidence`, `ClassificationPage`, `ClassificationState` (with methods), `WorkflowResult`
- `workflow/runtime.go` — new: `Runtime` struct (AgentConfig, Storage, Documents, Prompts, Logger)
- `workflow/prompts.go` — new: `ComposePrompt` combining instructions + spec + state
- `pkg/formatting/parse.go` — new: `Parse[T any]` with markdown code fence fallback, `ErrParseFailed`
- `go.mod` / `go.sum` — added `go-agents-orchestration` and `document-context` dependencies
- `tests/formatting/parse_test.go` — new: 10 test cases for `Parse`
- `tests/workflow/types_test.go` — new: tests for `NeedsEnhance`, `EnhancePages`, JSON round-trip, confidence constants
- `tests/workflow/prompts_test.go` — new: tests for `ComposePrompt` with mock `prompts.System`

## Patterns Established

- **Workflow types are lean** — only persist what's needed; intermediate LLM response fields are consumed by nodes but not stored per-page
- **Per-page rationale** — `ClassificationPage.Rationale` captures model notes that feed into overall assessment
- **Enhancement as per-page flag** — `Enhance` bool + `Enhancements` string on `ClassificationPage` replaces a separate quality assessment struct
- **Mock prompts.System** — test pattern for workflow tests needing a prompts dependency

## Validation Results

- `go vet ./...` — passes
- `go test ./tests/...` — all 28 new tests pass, full suite green (17 packages)
- `go mod tidy` — no changes
