# Objective Planning: #26 — Classification Workflow

## Context

Objective #26 is the core of Phase 2 — the classification engine that reads security markings from PDF documents using Azure AI Foundry GPT vision models. It implements the `workflow/` package containing a 3-node state graph (init → classify → enhance?), all node implementations, prompt composition, and response parsing. The workflow adapts classify-docs' sequential page-by-page context accumulation pattern (96.3% accuracy) into a go-agents-orchestration state graph.

Dependencies #24 (Agent Configuration) is complete and #25 (Prompts Domain) is 100% complete (1/1 sub-issues closed). A clean transition closeout is needed before starting #26.

## Step 0: Transition Closeout

#25 (Prompts Domain) has 1/1 sub-issues closed (100% complete). Clean transition:

1. Close issue #25
2. Update `_project/phase.md` — mark #25 status as `Complete`
3. Delete `_project/objective.md`

## Sub-Issue Decomposition

### Sub-issue 1: Prompts domain extensions — instructions, specifications, and hardcoded defaults

**Labels**: `task`, `feature`
**Milestone**: `v0.2.0 - Classification Engine`

#### Context

The prompts domain becomes the single owner of all prompt content: tunable instructions (DB overrides or hardcoded defaults) and immutable specifications (the output schema and behavioral constraints per stage). This gives prompt authors read-only access to specifications when crafting instructions, and guarantees the workflow always gets a non-null result from both `Instructions()` and `Spec()`.

#### Scope

**New methods on `prompts.System` interface** (`internal/prompts/system.go`):
- `Instructions(ctx, stage) (string, error)` — returns active DB override's instructions, or hardcoded default if no active override exists for the stage. Never null.
- `Spec(ctx, stage) (string, error)` — returns the hardcoded specification for the stage. Always read-only, never null.

**Hardcoded prompt data** (new file, e.g. `internal/prompts/defaults.go`):
- Default instructions per stage (init, classify, enhance) — the tunable reasoning guidance
- Specifications per stage — the immutable output format + behavioral constraints the workflow parser depends on

**New handler routes** (`internal/prompts/handler.go`):
- `GET /{stage}/instructions` → calls `system.Instructions(ctx, stage)`, returns the instruction text
- `GET /{stage}/spec` → calls `system.Spec(ctx, stage)`, returns the specification text

**API Cartographer**: Update `_project/api/prompts/` docs with new endpoints.

#### Acceptance Criteria

- [ ] `Instructions(ctx, stage)` returns active DB override or hardcoded default
- [ ] `Spec(ctx, stage)` returns hardcoded specification (read-only)
- [ ] Both methods validate stage and never return empty
- [ ] Hardcoded defaults defined for all three stages (init, classify, enhance)
- [ ] Handler routes exposed and documented
- [ ] Existing prompts CRUD behavior unchanged

#### Key Files

- `internal/prompts/system.go` — interface additions
- `internal/prompts/repository.go` — method implementations
- `internal/prompts/defaults.go` — new, hardcoded instructions + specifications
- `internal/prompts/handler.go` — new routes
- `_project/api/prompts/` — API cartographer updates

---

### Sub-issue 2: Workflow foundation — types, runtime, errors, and parsing

**Labels**: `task`, `feature`
**Milestone**: `v0.2.0 - Classification Engine`

#### Scope

- `go get github.com/JaimeStill/go-agents-orchestration` + `go get github.com/JaimeStill/document-context`
- **types.go** — All shared types:
  - `PageImage` — `PageNumber int`, `ImagePath string` (file path in temp directory, not in-memory data URI)
  - `ClassificationState` — Running state accumulated across pages: classification, confidence, markings found, rationale, quality factor flag, page-level tracking
  - `WorkflowResult` — Final output from Execute (wraps ClassificationState with document metadata)
  - `QualityAssessment` — Quality metadata for enhance evaluation
  - Confidence constants: `HIGH`, `MEDIUM`, `LOW`
- **runtime.go** — `Runtime` struct bundling workflow dependencies: `gaconfig.AgentConfig`, `storage.System`, `documents.System`, `prompts.System`, `*slog.Logger`. Follows `internal/api/runtime.go` pattern.
- **errors.go** — Sentinel errors: `ErrDocumentNotFound`, `ErrRenderFailed`, `ErrClassifyFailed`, `ErrEnhanceFailed`, `ErrParseFailed`
- **parse.go** — Generic JSON parsing with markdown code fence fallback (adapted from classify-docs `parser.go` and agent-lab `parse.go`)
- **prompts.go** — Prompt composition for workflow consumption:
  - Calls `prompts.System.Instructions(ctx, stage)` for the tunable instruction layer
  - Calls `prompts.System.Spec(ctx, stage)` for the immutable specification layer
  - Builds the final system prompt by combining instructions + spec + running ClassificationState context

#### Acceptance Criteria

- [ ] go-agents-orchestration and document-context added to go.mod
- [ ] All workflow types defined with JSON tags
- [ ] `PageImage` uses file path (not in-memory data URI)
- [ ] Runtime struct defined with all dependency fields
- [ ] JSON parser handles both direct JSON and markdown-fenced JSON
- [ ] Prompt composition calls `Instructions()` + `Spec()` from prompts system
- [ ] Sentinel errors defined with descriptive messages

#### Key Files

- `workflow/types.go`, `workflow/runtime.go`, `workflow/errors.go`, `workflow/parse.go`, `workflow/prompts.go`
- `go.mod` — new dependencies

---

### Sub-issue 3: Init node — concurrent page rendering with temp storage

**Labels**: `task`, `feature`
**Milestone**: `v0.2.0 - Classification Engine`

#### Scope

- `initNode(runtime *Runtime) state.StateNode` — returns `state.NewFunctionNode` closure
- **Request-bound temp storage**: Images are rendered to a temp directory rather than held in memory. The temp directory path is stored in state for downstream nodes and cleaned up by `Execute()`.
- **Concurrent page rendering**: Pages are rendered in parallel using bounded concurrency (`errgroup` with `SetLimit`). ImageMagick rendering is CPU-heavy and benefits from parallelism.
- Node logic:
  1. Extract `document_id` and `temp_dir` from state
  2. Look up document record via `runtime.Documents.Find(ctx, documentID)` to get storage key, filename, page count
  3. Download PDF blob via `runtime.Storage` to a temp file in the temp directory
  4. Open PDF via `document.OpenPDF(tempPath)` (document-context)
  5. Create renderer via `image.NewImageMagickRenderer(config.DefaultImageConfig())`
  6. Extract all pages, render concurrently with bounded concurrency:
     - Each goroutine: `page.ToImage(renderer, nil)` → write PNG to `{tempDir}/page-{N}.png`
  7. Build `[]PageImage` (with file paths) and store in state
  8. Store document metadata in state for downstream nodes
  9. Close document (defer), temp PDF cleaned up with temp dir by Execute

#### Acceptance Criteria

- [ ] Init node downloads PDF blob from storage
- [ ] Pages rendered concurrently with bounded concurrency (errgroup)
- [ ] Rendered images saved to temp directory as individual files
- [ ] `[]PageImage` with file paths stored in state
- [ ] Document metadata stored in state
- [ ] Errors wrapped with context using sentinel errors

#### Key Files

- `workflow/init.go`

#### Reference

- `document.OpenPDF(path)` — `/home/jaime/code/document-context/pkg/document/pdf.go`
- `image.NewImageMagickRenderer(cfg)` — `/home/jaime/code/document-context/pkg/image/imagemagick.go`
- `encoding.EncodeImageDataURI(data, format)` — `/home/jaime/code/document-context/pkg/encoding/image.go`
- `page.ToImage(renderer, nil)` — nil cache, ephemeral rendering

---

### Sub-issue 4: Classify node — sequential page-by-page context accumulation

**Labels**: `task`, `feature`
**Milestone**: `v0.2.0 - Classification Engine`

#### Scope

- `classifyNode(runtime *Runtime) state.StateNode` — returns `state.NewFunctionNode` closure
- **Sequential processing**: Pages are classified one at a time so each page's result feeds into the next page's prompt (context accumulation). This is the pattern that achieved 96.3% accuracy in classify-docs.
- **Just-in-time image loading**: Each page's image is read from disk and encoded to base64 data URI only when that page is being classified, then discarded. Optimizes memory to one page image at a time.
- Node logic:
  1. Extract `[]PageImage` from state
  2. Initialize empty `ClassificationState`
  3. Create agent via `agent.New(&runtime.AgentConfig)` (per-request creation)
  4. For each page sequentially:
     - Read image from `pageImage.ImagePath`, encode to data URI via `encoding.EncodeImageDataURI`
     - Build prompt with current ClassificationState + page context using `prompts.go` composition
     - Call `agent.Vision(ctx, systemPrompt, []string{dataURI})`
     - Parse JSON response via `parse.go`
     - Update running ClassificationState with page results
  5. After all pages: evaluate whether image quality limited confidence
  6. Set `needs_enhancement` flag in state (initially always `false` per phase constraints)
  7. Store final `ClassificationState` in state

#### Acceptance Criteria

- [ ] Pages processed sequentially with context accumulation
- [ ] Each page's classification result feeds into the next page's prompt
- [ ] Images loaded from disk and encoded just-in-time (one at a time)
- [ ] Vision API called with base64 data URI per page
- [ ] JSON response parsed with code fence fallback
- [ ] `needs_enhancement` flag set in state (always false initially)
- [ ] ClassificationState stored in state for downstream consumption

#### Key Files

- `workflow/classify.go`

#### Reference

- classify-docs `ProcessWithContext` pattern — `/home/jaime/code/go-agents/tools/classify-docs/pkg/classify/document.go`
- classify-docs prompt template — same file, `buildClassificationPrompt` function
- classify-docs JSON parser — `/home/jaime/code/go-agents/tools/classify-docs/pkg/classify/parser.go`

---

### Sub-issue 5: Enhance node, graph assembly, and Execute function

**Labels**: `task`, `feature`
**Milestone**: `v0.2.0 - Classification Engine`

#### Scope

- **enhance.go**: Placeholder node — reads state, returns state unchanged. Full node function signature in place (`enhanceNode(runtime *Runtime) state.StateNode`) so the conditional edge evaluates correctly. Actual re-render and reassess logic deferred until trigger conditions are defined through experimentation.
- **workflow.go**:
  - State graph assembly: `init → classify → [needs_enhancement=true] → enhance → exit` / `[needs_enhancement=false] → exit`
  - Two exit points: classify (when no enhancement needed) and enhance (when enhancement runs)
  - `Execute(ctx context.Context, runtime *Runtime, documentID uuid.UUID) (*WorkflowResult, error)` — top-level entry point:
    1. Create temp directory (`os.MkdirTemp`) and defer `os.RemoveAll` for cleanup
    2. Create initial state with document ID and temp directory path
    3. Build graph (add nodes, wire edges, set entry/exit points)
    4. Execute graph
    5. Extract `WorkflowResult` from final state

#### Acceptance Criteria

- [ ] Enhance node is a pass-through (returns state unchanged)
- [ ] State graph wired with conditional edge on `needs_enhancement`
- [ ] Execute creates temp directory and defers cleanup on all paths
- [ ] Execute function creates initial state, runs graph, returns WorkflowResult
- [ ] Graph topology matches README spec: init → classify → enhance? → exit

#### Key Files

- `workflow/enhance.go`, `workflow/workflow.go`

#### Reference

- agent-lab graph assembly — `/home/jaime/code/agent-lab/workflows/classify/classify.go` (factory function, AddNode/AddEdge/SetEntryPoint/SetExitPoint)
- `state.KeyEquals("needs_enhancement", true/false)` — conditional edge predicates

## Architecture Decisions

1. **Prompts domain owns all prompt content**: The prompts domain is the single source of truth for both tunable instructions (DB overrides or hardcoded defaults) and immutable specifications (output schema + behavioral constraints). `Instructions(ctx, stage)` always returns a non-null string. `Spec(ctx, stage)` always returns the read-only specification. The workflow composes both into the final system prompt.

2. **Specifications replace "output format"**: The immutable traits of each prompt stage are called "specifications" — they define the expected JSON output structure and behavioral constraints that the workflow parser depends on. Exposed via `GET /api/prompts/{stage}/spec` as read-only context for prompt authors crafting instructions.

3. **Request-bound temp storage**: Page images are rendered to a temp directory (created by `Execute`, cleaned up via defer) rather than held as base64 data URIs in memory. `PageImage` stores a file path. Classify/enhance nodes encode to data URI just-in-time per page, keeping memory usage proportional to one page at a time rather than all pages simultaneously.

4. **Concurrent rendering, sequential classification**: The init node renders pages concurrently (ImageMagick is CPU-heavy, bounded concurrency via `errgroup.SetLimit`). The classify node processes pages sequentially for context accumulation — each page's classification feeds the next page's prompt. This preserves the 96.3% accuracy pattern from classify-docs while optimizing the rendering bottleneck.

5. **Inline sequential processing**: Herald implements the classify-docs `ProcessWithContext` pattern inline rather than importing the generic abstraction. A simple `for range pages` loop with state accumulation is clearer for a single workflow.

6. **Graph exit points**: Two exit points — classify (no enhancement needed) and enhance (enhancement ran). The conditional edge on `needs_enhancement` determines which path executes. Initially, classify always sets `needs_enhancement = false`, so enhance never runs.

7. **Per-request agent creation**: Each `Execute` call creates a fresh `agent.Agent` from the config. This matches Herald's stateless agent design — no agent lifecycle management.

## Dependency Order

```
Sub-issue 1 (prompts extensions) → Sub-issue 2 (workflow foundation) → Sub-issue 3 (init) → Sub-issue 4 (classify) → Sub-issue 5 (enhance + Execute)
```

Sub-issue 1 extends the prompts domain that the workflow consumes. Sub-issues 3 and 4 could potentially be parallelized (both depend only on sub-issue 2), but the classify node benefits from having the init node's state output shape finalized first.
