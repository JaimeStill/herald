# 51 - Parallelize classify and enhance workflow nodes

## Summary

Replaced sequential processing in the classify and enhance workflow nodes with bounded `errgroup` concurrency. Removed context accumulation from classify (each page is now classified independently). Both nodes compose prompts once before launching goroutines and create per-goroutine agents. The enhance node opens per-goroutine PDF handles for concurrency safety. Reuses the existing `workerCount` function (renamed from `renderWorkerCount`) for all bounded concurrency scenarios.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Concurrency limit | Reuse CPU-based `workerCount` | Azure rate limits (800 RPM / 800K TPM) are generous; image encoding is CPU-bound; no new config needed |
| Context accumulation | Removed (promptState always nil) | Shifts synthesis entirely to finalize node; enables independent parallel page classification |
| `workerCount` location | Moved to `workflow.go` | Shared across init, classify, and enhance nodes — no longer render-specific |
| Prompt composition | Once before errgroup | Avoids data races from concurrent reads/writes to ClassificationState |
| Per-goroutine PDF handles | Each enhance goroutine opens its own | PDF handles are not concurrency-safe |

## Files Modified

- `internal/workflow/workflow.go` — Added `workerCount` (renamed from `renderWorkerCount`)
- `internal/workflow/init.go` — Removed `renderWorkerCount`, updated call to `workerCount`
- `internal/workflow/classify.go` — Parallel errgroup, removed context accumulation, removed `classifyPage`
- `internal/workflow/enhance.go` — Parallel errgroup, pre-composed prompt, per-goroutine PDF + agent, removed `enhancePage`
- `_project/README.md` — Updated classify/enhance node descriptions, classification approach decision

## Patterns Established

- **Shared `workerCount`**: All bounded concurrency in the workflow package uses `workerCount(n)` from `workflow.go`
- **Pre-composed prompts**: When goroutines share the same prompt, compose it once before the errgroup to avoid data races

## Validation Results

- `go vet ./...` — pass
- `go test ./tests/...` — all 18 suites pass
- `go mod tidy` — no changes
- Manual testing: 6-page document classified correctly in ~16s (classify node) vs ~78s estimated sequential
