package cli

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// BatchResult pairs a single batch item's output with its error. Exactly one of
// Value (on success) or Err (on failure) is meaningful. Results preserve input
// order.
type BatchResult[R any] struct {
	Value R
	Err   error
}

// RunBatch runs fn over items with at most concurrency executions in flight,
// using the house errgroup.SetLimit idiom (mirrors internal/workflow/classify.go).
// Item failures are isolated: a failing item never cancels the batch, its error
// is captured in the corresponding BatchResult, and the remaining items proceed.
// Cancelling ctx (e.g. on SIGINT) stops new work from launching and records
// ctx.Err() for items that had not completed, yielding a partial-but-usable
// result slice. concurrency below 1 is treated as 1.
func RunBatch[T any, R any](
	ctx context.Context,
	items []T,
	concurrency int,
	fn func(context.Context, T) (R, error),
) []BatchResult[R] {
	if concurrency < 1 {
		concurrency = 1
	}

	results := make([]BatchResult[R], len(items))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	for i, item := range items {
		g.Go(func() error {
			if err := gctx.Err(); err != nil {
				results[i] = BatchResult[R]{Err: err}
				return nil
			}
			v, err := fn(gctx, item)
			results[i] = BatchResult[R]{Value: v, Err: err}
			return nil
		})
	}

	// fn's error is recorded per item and never returned to the group, so Wait
	// only unblocks once every item has run (or the context was cancelled).
	_ = g.Wait()
	return results
}
