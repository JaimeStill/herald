# 51 - Parallelize classify and enhance workflow nodes

## Problem Context

The classify and enhance workflow nodes process pages sequentially. Each page's vision model call takes ~13 seconds, so a 10-page document takes over 2 minutes for classify alone. Both nodes should be parallelized using bounded `errgroup` concurrency, matching the pattern already established in the init node's `renderPages`. With Azure rate limits at 800 RPM / 800K TPM, CPU cores remain the practical bottleneck — reuse the existing worker count calculation.

## Architecture Approach

Reuse the `renderWorkerCount` function (renamed to `workerCount`) for all bounded concurrency scenarios. Remove context accumulation from classify (promptState always nil). Compose prompts once before the errgroup to avoid data races. Each goroutine creates its own agent and writes to its own pre-allocated slice index.

## Implementation

### Step 1: Rename `renderWorkerCount` to `workerCount` in `internal/workflow/init.go`

Rename the function and update the call site in `renderPages`:

```go
func workerCount(n int) int {
	return max(min(runtime.NumCPU(), n), 1)
}
```

Update the call in `renderPages`:

```go
g.SetLimit(workerCount(pageCount))
```

### Step 2: Parallelize classify node in `internal/workflow/classify.go`

Replace the sequential `classifyPages` function with errgroup bounded concurrency. Remove context accumulation. Compose the prompt once before launching goroutines.

Replace the existing `classifyPages` function with:

```go
func classifyPages(ctx context.Context, rt *Runtime, cs *ClassificationState) error {
	prompt, err := ComposePrompt(ctx, rt.Prompts, prompts.StageClassify, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrClassifyFailed, err)
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(workerCount(len(cs.Pages)))

	for i := range cs.Pages {
		g.Go(func() error {
			if gctx.Err() != nil {
				return gctx.Err()
			}

			a, err := agent.New(&rt.Agent)
			if err != nil {
				return fmt.Errorf("page %d: create agent: %w", i+1, err)
			}

			dataURI, err := encodePageImage(cs.Pages[i].ImagePath)
			if err != nil {
				return fmt.Errorf("page %d: %w", i+1, err)
			}

			resp, err := a.Vision(gctx, prompt, []string{dataURI})
			if err != nil {
				return fmt.Errorf("page %d: vision call: %w", i+1, err)
			}

			parsed, err := formatting.Parse[pageResponse](resp.Content())
			if err != nil {
				return fmt.Errorf("page %d: parse response: %w", i+1, err)
			}

			applyPageResponse(&cs.Pages[i], parsed)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("%w: %w", ErrClassifyFailed, err)
	}

	return nil
}
```

Remove the `classifyPage` function — its logic is inlined into the goroutine.

Add the `errgroup` import:

```go
"golang.org/x/sync/errgroup"
```

Remove unused imports that are no longer needed: the `"os"` import stays (used by `encodePageImage`), but the `"github.com/JaimeStill/go-agents/pkg/agent"` import stays (used for per-goroutine agent creation). Remove the `"github.com/JaimeStill/document-context/pkg/document"` and `"github.com/JaimeStill/document-context/pkg/encoding"` imports — they are only used by `encodePageImage` which is still in this file. Actually, keep all existing imports and add `errgroup`.

### Step 3: Parallelize enhance node in `internal/workflow/enhance.go`

Replace the `enhancePages` function with errgroup bounded concurrency. Compose the prompt once before launching goroutines. Each goroutine opens its own PDF handle.

Replace the existing `enhancePages` function with:

```go
func enhancePages(ctx context.Context, rt *Runtime, cs *ClassificationState, tempDir string) error {
	pdfPath := filepath.Join(tempDir, sourcePDF)
	enhanced := cs.EnhancePages()

	prompt, err := ComposePrompt(ctx, rt.Prompts, prompts.StageEnhance, cs)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrEnhanceFailed, err)
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(workerCount(len(enhanced)))

	for _, i := range enhanced {
		g.Go(func() error {
			if gctx.Err() != nil {
				return gctx.Err()
			}

			pdfDoc, err := document.OpenPDF(pdfPath)
			if err != nil {
				return fmt.Errorf("page %d: open pdf: %w", cs.Pages[i].PageNumber, err)
			}
			defer pdfDoc.Close()

			a, err := agent.New(&rt.Agent)
			if err != nil {
				return fmt.Errorf("page %d: create agent: %w", cs.Pages[i].PageNumber, err)
			}

			imgPath, err := rerender(pdfDoc, &cs.Pages[i], tempDir)
			if err != nil {
				return fmt.Errorf("page %d: %w", cs.Pages[i].PageNumber, err)
			}
			cs.Pages[i].ImagePath = imgPath

			dataURI, err := encodePageImage(imgPath)
			if err != nil {
				return fmt.Errorf("page %d: %w", cs.Pages[i].PageNumber, err)
			}

			resp, err := a.Vision(gctx, prompt, []string{dataURI})
			if err != nil {
				return fmt.Errorf("page %d: vision call: %w", cs.Pages[i].PageNumber, err)
			}

			parsed, err := formatting.Parse[enhanceResponse](resp.Content())
			if err != nil {
				return fmt.Errorf("page %d: parse response: %w", cs.Pages[i].PageNumber, err)
			}

			cs.Pages[i].MarkingsFound = parsed.MarkingsFound
			cs.Pages[i].Rationale = parsed.Rationale
			cs.Pages[i].Enhancements = nil

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("%w: %w", ErrEnhanceFailed, err)
	}

	return nil
}
```

Update the `enhancePage` function signature — remove `a agent.Agent` and `pdfDoc document.Document` parameters since these are now created per-goroutine. Actually, the `enhancePage` function is no longer needed — its logic is inlined into the goroutine closure. Remove it.

Add the `errgroup` import:

```go
"golang.org/x/sync/errgroup"
```

## Validation Criteria

- [ ] `renderWorkerCount` renamed to `workerCount` in `init.go` and call site updated
- [ ] `classifyPages` uses errgroup with bounded concurrency
- [ ] Context accumulation removed from classify (promptState always nil)
- [ ] Each classify goroutine creates its own agent
- [ ] `enhancePages` uses errgroup with bounded concurrency
- [ ] Each enhance goroutine opens its own PDF handle
- [ ] Each enhance goroutine creates its own agent
- [ ] Prompt composed once before errgroup in both nodes
- [ ] `go vet ./...` passes
- [ ] `go test ./tests/...` passes
