# Plan: Parallelize classify and enhance workflow nodes (#51)

## Context

Testing the classify endpoint on a 1-page document took ~40 seconds across 3 LLM round-trips. Each page's vision call takes ~13 seconds, so sequential processing would take 2+ minutes for a 10-page document. LLM calls are network-bound, not CPU-bound, so they parallelize well within Azure API rate limits. The init node already uses `errgroup` bounded concurrency for CPU-bound ImageMagick rendering — both classify and enhance should follow the same pattern for inference calls.

## Approach

Reuse the existing `workerCount` pattern (`max(min(NumCPU, pageCount), 1)`) for inference concurrency. Even though LLM calls are network-bound, the image encoding work (file I/O, base64) is CPU-bound, and Azure rate limits (800 RPM / 800K TPM) are generous enough that CPU cores remain the practical bottleneck. No new config or Runtime fields needed.

### 1. Rename `workerCount` to `workerCount` (`internal/workflow/init.go`)

The function now applies to all bounded concurrency scenarios (rendering, classify, enhance), not just rendering. Rename it and update the call site in `renderPages`.

### 2. Parallelize classify node (`internal/workflow/classify.go`)

- **Remove context accumulation**: `promptState` is always `nil` — no prior page findings injected. This shifts synthesis responsibility entirely to the finalize node (which already handles it).
- **Compose prompt once** before the errgroup (same prompt for all pages since state is nil).
- **Each goroutine**: create agent via `agent.New` → encode image → vision call → write to `pages[i]`.
- **Bounded concurrency**: `errgroup.SetLimit(workerCount(len(cs.Pages)))`.
- Pre-allocated slice indices — each goroutine writes to its own index (no mutex needed).

### 2. Parallelize enhance node (`internal/workflow/enhance.go`)

- **Compose prompt once** before the errgroup (snapshot of current `cs` state — avoids data race from concurrent reads/writes during goroutine execution).
- **Each goroutine**: open its own PDF handle → rerender → encode → vision call with pre-composed prompt → update `pages[i]` → close PDF handle.
- **Each goroutine creates its own agent** via `agent.New`.
- **Bounded concurrency**: `errgroup.SetLimit(workerCount(len(enhanced)))`.
- PDF handles opened per-goroutine since they are not concurrency-safe.

## Files Modified

| File | Change |
|------|--------|
| `internal/workflow/init.go` | Rename `renderWorkerCount` → `workerCount` and update call site |
| `internal/workflow/classify.go` | Replace sequential loop with errgroup bounded concurrency, remove context accumulation, compose prompt once, per-goroutine agent creation |
| `internal/workflow/enhance.go` | Replace sequential loop with errgroup bounded concurrency, compose prompt once, per-goroutine PDF handle + agent creation |

## Validation

- `go vet ./...` passes
- `go test ./tests/...` passes
- Manual testing with multi-page documents to verify correctness matches sequential results
