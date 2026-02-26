# 39 — Init Node: Concurrent Page Rendering with Temp Storage

## Summary

Implemented the init node for the classification workflow. The node downloads a PDF from blob storage, opens it via document-context, and renders all pages to PNG images concurrently using ImageMagick with bounded worker count. Page images are written to a request-bound temp directory. The initial `ClassificationState` (with `ClassificationPage` entries containing page numbers and image paths) is stored in the workflow state bag for downstream nodes.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Node function convention | Single exported `*Node` function with clean body composed of helper calls | Each helper encapsulates one stage; the node body reads as an orchestration sequence |
| Worker count | `max(min(runtime.NumCPU(), pageCount), 1)` | Caps at CPU count (ImageMagick is CPU-heavy) and page count (no excess goroutines), floor of 1; adapted from agent-lab |
| Temp PDF filename | `sourcePDF` constant in init.go | Avoids hardcoded strings; scoped to the file since only `downloadPDF` and `renderPages` use it |
| No new tests | InitNode requires mocking two domain systems + real PDF + ImageMagick | Integration-level testing; not worth the mocking overhead for unit tests |
| Exported InitNode | Uppercase `InitNode` | Graph assembly in #41 needs to reference it |

## Files Modified

- `workflow/init.go` — new: `InitNode`, `downloadPDF`, `extractInitState`, `renderPages`, `renderWorkerCount`
- `workflow/types.go` — added state key constants (`KeyDocumentID`, `KeyTempDir`, `KeyFilename`, `KeyPageCount`, `KeyClassState`)
- `go.mod` / `go.sum` — added `document-context`, `go-agents-orchestration`; promoted `golang.org/x/sync` to direct

## Patterns Established

- **Workflow node file convention**: Each node gets its own file (`init.go`, `classify.go`, `enhance.go`). A single exported `*Node` function with a clean body composed of unexported helper calls that encapsulate each stage.
- **State key constants**: `Key*` constants in `types.go` for the `state.State` data map keys shared across nodes.
- **Bounded render concurrency**: `renderWorkerCount` helper using `max(min(NumCPU, pageCount), 1)`.

## Validation Results

- `go vet ./...` — passes
- `go test ./tests/...` — all 17 packages pass
- `go mod tidy` — clean after tidy
