# 41 — Enhance Node, Finalize Node, Graph Assembly, and Execute Function

## Problem Context

The workflow package has init and classify nodes but no way to run them. This issue completes the classification workflow by adding the remaining nodes (enhance with re-rendering + reclassification, finalize for document-level synthesis), wiring the state graph, and exposing a top-level `Execute` function that manages the full lifecycle including temp directory creation and cleanup.

## Architecture Approach

The graph follows agent-lab's assembly pattern: register nodes, wire edges with predicates, set entry/exit points. Herald's conditional edge evaluates `ClassificationState.NeedsEnhance()` via a custom `TransitionPredicate` that extracts the classification state from the state bag.

**Type change:** The `Enhance bool` + `Enhancements string` fields on `ClassificationPage` are replaced with a single `Enhancements *EnhanceSettings` pointer. `EnhanceSettings` carries structured rendering parameters (brightness, contrast, saturation) that map directly to document-context's `ImageMagickConfig` filter fields. `NeedsEnhance()` checks for any page with non-nil `Enhancements` — single source of truth, no bool/data inconsistency.

**Enhance node:** A hybrid rendering + inference node. For each flagged page: re-opens the PDF from temp dir, re-renders with adjusted ImageMagick settings derived from `EnhanceSettings`, sends the enhanced image through a vision call to update the per-page classification, then clears the `Enhancements` pointer.

**Finalize node:** Uses `agent.Chat` (not Vision) to synthesize document-level classification from all per-page findings. The `Execute` function owns temp directory lifecycle via defer.

## Implementation

### Step 1: Add `EnhanceSettings` and update `ClassificationPage` in `workflow/types.go`

Add the new type and update the page struct. Replace `Enhance bool` + `Enhancements string` with `Enhancements *EnhanceSettings`:

```go
type EnhanceSettings struct {
	Brightness *int `json:"brightness,omitempty"`
	Contrast   *int `json:"contrast,omitempty"`
	Saturation *int `json:"saturation,omitempty"`
}
```

Updated `ClassificationPage` (replaces both `Enhance` and `Enhancements` fields):

```go
type ClassificationPage struct {
	PageNumber    int              `json:"page_number"`
	ImagePath     string           `json:"image_path"`
	MarkingsFound []string         `json:"markings_found"`
	Rationale     string           `json:"rationale"`
	Enhancements  *EnhanceSettings `json:"enhancements,omitempty"`
}
```

Add `Enhance` convenience method to `ClassificationPage`:

```go
func (p *ClassificationPage) Enhance() bool {
	return p.Enhancements != nil
}
```

Updated `NeedsEnhance` and `EnhancePages` to use it:

```go
func (s *ClassificationState) NeedsEnhance() bool {
	return slices.ContainsFunc(s.Pages, func(p ClassificationPage) bool {
		return p.Enhance()
	})
}

func (s *ClassificationState) EnhancePages() []int {
	var indices []int
	for i, p := range s.Pages {
		if p.Enhance() {
			indices = append(indices, i)
		}
	}
	return indices
}
```

### Step 2: Update `classify.go` response type and apply function

Update the log line in `classifyPages` — replace `"enhance", cs.Pages[i].Enhance` with:

```go
"enhance", cs.Pages[i].Enhance(),
```

Update `pageResponse` to match the new type shape:

```go
type pageResponse struct {
	MarkingsFound []string         `json:"markings_found"`
	Rationale     string           `json:"rationale"`
	Enhancements  *EnhanceSettings `json:"enhancements,omitempty"`
}
```

Update `applyPageResponse`:

```go
func applyPageResponse(page *ClassificationPage, resp pageResponse) {
	page.MarkingsFound = resp.MarkingsFound
	page.Rationale = resp.Rationale
	page.Enhancements = resp.Enhancements
}
```

### Step 3: Update classify spec in `internal/prompts/specs.go`

Replace the `classifySpec` constant with structured `enhancements` output:

```go
const classifySpec = `Respond with a JSON object matching this exact structure:

{
  "markings_found": ["<marking1>", "<marking2>"],
  "rationale": "<explanation>",
  "enhancements": null
}

Field constraints:
- markings_found: Array of distinct marking strings found on this page,
  exactly as they appear in the document. Include the full marking text
  with any caveats (e.g., "SECRET//NOFORN" not just "SECRET").
- rationale: Brief explanation of what security markings were found on
  this page and their significance. Note any conflicts or ambiguities
  with prior page findings if a classification state is provided.
- enhancements: Set to null when the image is clear enough to read all
  markings confidently. When image quality prevents confident reading of
  any markings, provide an object with rendering adjustments:
  {
    "brightness": <80-200, 100=neutral, increase for faded/dark pages>,
    "contrast": <-50 to 50, 0=neutral, increase to sharpen faded markings>,
    "saturation": <80-200, 100=neutral, adjust for color-related issues>
  }
  Only include fields that need adjustment; omit fields that should stay
  at their neutral values.

Behavioral constraints:
- Always respond with valid JSON, no markdown fencing
- Process exactly one page per response
- Report only what you observe on this page
- If prior page findings are provided in the prompt, use them as context
  to identify consistency or conflicts, but do not repeat prior findings
  in markings_found — only include markings visible on the current page`
```

Replace the `enhanceSpec` constant for reclassification (just markings_found + rationale):

```go
const enhanceSpec = `Respond with a JSON object matching this exact structure:

{
  "markings_found": ["<marking1>", "<marking2>"],
  "rationale": "<explanation>"
}

Field constraints:
- markings_found: Array of distinct marking strings found on this
  enhanced page, exactly as they appear. Include the full marking text
  with any caveats.
- rationale: Brief explanation of what the enhanced image reveals compared
  to the original assessment. Note any new markings discovered or prior
  findings confirmed by the improved image quality.

Behavioral constraints:
- Always respond with valid JSON, no markdown fencing
- Focus analysis on the enhanced image with improved rendering settings
- Compare findings against the prior page analysis provided in the prompt
- Report only what you observe on the current enhanced page`
```

### Step 4: Add `ErrFinalizeFailed` sentinel to `workflow/errors.go`

Add to the existing `var` block:

```go
ErrFinalizeFailed = errors.New("finalize failed")
```

### Step 5: Create `workflow/enhance.go`

Re-renders flagged pages with adjusted ImageMagick settings from `EnhanceSettings`, then reclassifies each enhanced page via vision call to update per-page findings. Clears `Enhancements` after processing.

```go
package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JaimeStill/document-context/pkg/config"
	"github.com/JaimeStill/document-context/pkg/document"
	"github.com/JaimeStill/document-context/pkg/image"

	"github.com/JaimeStill/go-agents/pkg/agent"

	"github.com/JaimeStill/go-agents-orchestration/pkg/state"

	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/pkg/formatting"
)

type enhanceResponse struct {
	MarkingsFound []string `json:"markings_found"`
	Rationale     string   `json:"rationale"`
}

func EnhanceNode(rt *Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		cs, err := extractClassState(s)
		if err != nil {
			return s, fmt.Errorf("enhance: %w", err)
		}

		tempDir, err := extractTempDir(s)
		if err != nil {
			return s, fmt.Errorf("enhance: %w", err)
		}

		enhanced := cs.EnhancePages()

		if err := enhancePages(ctx, rt, cs, tempDir); err != nil {
			return s, fmt.Errorf("enhance: %w", err)
		}

		rt.Logger.InfoContext(
			ctx, "enhance node complete",
			"pages_enhanced", len(enhanced),
		)

		s = s.Set(KeyClassState, *cs)
		return s, nil
	})
}

func extractTempDir(s state.State) (string, error) {
	val, ok := s.Get(KeyTempDir)
	if !ok {
		return "", fmt.Errorf("%w: missing %s in state", ErrEnhanceFailed, KeyTempDir)
	}

	tempDir, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("%w: %s is not string", ErrEnhanceFailed, KeyTempDir)
	}

	return tempDir, nil
}

func enhancePages(ctx context.Context, rt *Runtime, cs *ClassificationState, tempDir string) error {
	pdfPath := filepath.Join(tempDir, sourcePDF)
	pdfDoc, err := document.OpenPDF(pdfPath)
	if err != nil {
		return fmt.Errorf("%w: open pdf: %w", ErrEnhanceFailed, err)
	}
	defer pdfDoc.Close()

	a, err := agent.New(&rt.Agent)
	if err != nil {
		return fmt.Errorf("%w: create agent: %w", ErrEnhanceFailed, err)
	}

	for _, i := range cs.EnhancePages() {
		if err := enhancePage(ctx, a, rt, pdfDoc, cs, i, tempDir); err != nil {
			return fmt.Errorf("%w: page %d: %w", ErrEnhanceFailed, cs.Pages[i].PageNumber, err)
		}

		rt.Logger.InfoContext(
			ctx, "page enhanced",
			"page", cs.Pages[i].PageNumber,
			"markings", cs.Pages[i].MarkingsFound,
		)
	}

	return nil
}

func enhancePage(
	ctx context.Context,
	a agent.Agent,
	rt *Runtime,
	pdfDoc document.Document,
	cs *ClassificationState,
	pageIdx int,
	tempDir string,
) error {
	page := &cs.Pages[pageIdx]

	// Re-render with adjusted settings
	imgPath, err := rerender(pdfDoc, page, tempDir)
	if err != nil {
		return err
	}
	page.ImagePath = imgPath

	// Encode enhanced image for vision call
	dataURI, err := encodePageImage(imgPath)
	if err != nil {
		return err
	}

	// Compose prompt with current classification state as context
	prompt, err := ComposePrompt(ctx, rt.Prompts, prompts.StageEnhance, cs)
	if err != nil {
		return err
	}

	resp, err := a.Vision(ctx, prompt, []string{dataURI})
	if err != nil {
		return fmt.Errorf("vision call: %w", err)
	}

	parsed, err := formatting.Parse[enhanceResponse](resp.Content())
	if err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	// Update page findings and clear enhancement flag
	page.MarkingsFound = parsed.MarkingsFound
	page.Rationale = parsed.Rationale
	page.Enhancements = nil

	return nil
}

func rerender(pdfDoc document.Document, page *ClassificationPage, tempDir string) (string, error) {
	p, err := pdfDoc.ExtractPage(page.PageNumber)
	if err != nil {
		return "", fmt.Errorf("extract page %d: %w", page.PageNumber, err)
	}

	cfg := buildEnhanceConfig(page.Enhancements)
	renderer, err := image.NewImageMagickRenderer(cfg)
	if err != nil {
		return "", fmt.Errorf("create renderer: %w", err)
	}

	data, err := p.ToImage(renderer, nil)
	if err != nil {
		return "", fmt.Errorf("render page %d: %w", page.PageNumber, err)
	}

	imgPath := filepath.Join(tempDir, fmt.Sprintf("page-%d-enhanced.png", page.PageNumber))
	if err := os.WriteFile(imgPath, data, 0600); err != nil {
		return "", fmt.Errorf("write enhanced page %d: %w", page.PageNumber, err)
	}

	return imgPath, nil
}

func buildEnhanceConfig(settings *EnhanceSettings) config.ImageConfig {
	opts := map[string]any{
		"background": "white",
	}

	if settings.Brightness != nil {
		opts["brightness"] = *settings.Brightness
	}

	if settings.Contrast != nil {
		opts["contrast"] = *settings.Contrast
	}

	if settings.Saturation != nil {
		opts["saturation"] = *settings.Saturation
	}

	return config.ImageConfig{
		Format:  "png",
		DPI:     300,
		Options: opts,
	}
}

```

**Notes:**
- `EnhancePages()` returns the indices of flagged pages — the enhance loop iterates only those
- `rerender` extracts the specific page from the PDF (still in temp dir as `source.pdf`), creates a renderer with the adjusted config, and writes the enhanced image to a separate file (`page-N-enhanced.png`)
- `encodePageImage` is reused from classify.go (same data URI encoding)
- `enhanceResponse` only has `markings_found` + `rationale` — no enhancement fields since the page has been enhanced
- After reclassification, `Enhancements` is set to `nil` so the page is no longer flagged
- The log message captures the count before processing via `len(cs.EnhancePages())`

### Step 6: Create `workflow/finalize.go`

```go
package workflow

import (
	"context"
	"fmt"

	"github.com/JaimeStill/go-agents/pkg/agent"

	"github.com/JaimeStill/go-agents-orchestration/pkg/state"

	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/pkg/formatting"
)

type finalizeResponse struct {
	Classification string     `json:"classification"`
	Confidence     Confidence `json:"confidence"`
	Rationale      string     `json:"rationale"`
}

func FinalizeNode(rt *Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		cs, err := extractClassState(s)
		if err != nil {
			return s, fmt.Errorf("finalize: %w", err)
		}

		if err := synthesize(ctx, rt, cs); err != nil {
			return s, fmt.Errorf("finalize: %w", err)
		}

		rt.Logger.InfoContext(
			ctx, "finalize node complete",
			"classification", cs.Classification,
			"confidence", cs.Confidence,
		)

		s = s.Set(KeyClassState, *cs)
		return s, nil
	})
}

func synthesize(ctx context.Context, rt *Runtime, cs *ClassificationState) error {
	a, err := agent.New(&rt.Agent)
	if err != nil {
		return fmt.Errorf("%w: create agent: %w", ErrFinalizeFailed, err)
	}

	prompt, err := ComposePrompt(ctx, rt.Prompts, prompts.StageFinalize, cs)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFinalizeFailed, err)
	}

	resp, err := a.Chat(ctx, prompt)
	if err != nil {
		return fmt.Errorf("%w: chat call: %w", ErrFinalizeFailed, err)
	}

	parsed, err := formatting.Parse[finalizeResponse](resp.Content())
	if err != nil {
		return fmt.Errorf("%w: parse response: %w", ErrFinalizeFailed, err)
	}

	cs.Classification = parsed.Classification
	cs.Confidence = parsed.Confidence
	cs.Rationale = parsed.Rationale

	return nil
}
```

### Step 7: Create `workflow/workflow.go`

```go
package workflow

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	orchconfig "github.com/JaimeStill/go-agents-orchestration/pkg/config"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
)

func Execute(ctx context.Context, rt *Runtime, documentID uuid.UUID) (*WorkflowResult, error) {
	tempDir, err := os.MkdirTemp("", "herald-classify-*")
	if err != nil {
		return nil, fmt.Errorf("create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	graph, err := buildGraph(rt)
	if err != nil {
		return nil, fmt.Errorf("build graph: %w", err)
	}

	initialState := state.New(nil)
	initialState = initialState.Set(KeyDocumentID, documentID)
	initialState = initialState.Set(KeyTempDir, tempDir)

	finalState, err := graph.Execute(ctx, initialState)
	if err != nil {
		return nil, fmt.Errorf("execute graph: %w", err)
	}

	return extractResult(finalState)
}

func buildGraph(rt *Runtime) (state.StateGraph, error) {
	cfg := orchconfig.DefaultGraphConfig("herald-classify")
	cfg.Observer = "noop"

	graph, err := state.NewGraph(cfg)
	if err != nil {
		return nil, err
	}

	if err := graph.AddNode("init", InitNode(rt)); err != nil {
		return nil, err
	}

	if err := graph.AddNode("classify", ClassifyNode(rt)); err != nil {
		return nil, err
	}

	if err := graph.AddNode("enhance", EnhanceNode(rt)); err != nil {
		return nil, err
	}

	if err := graph.AddNode("finalize", FinalizeNode(rt)); err != nil {
		return nil, err
	}

	// init → classify (unconditional)
	if err := graph.AddEdge("init", "classify", nil); err != nil {
		return nil, err
	}

	// classify → enhance (when any page needs enhancement)
	if err := graph.AddEdge("classify", "enhance", needsEnhance); err != nil {
		return nil, err
	}

	// classify → finalize (when no enhancement needed)
	if err := graph.AddEdge("classify", "finalize", state.Not(needsEnhance)); err != nil {
		return nil, err
	}

	// enhance → finalize (unconditional)
	if err := graph.AddEdge("enhance", "finalize", nil); err != nil {
		return nil, err
	}

	if err := graph.SetEntryPoint("init"); err != nil {
		return nil, err
	}

	if err := graph.SetExitPoint("finalize"); err != nil {
		return nil, err
	}

	return graph, nil
}

func needsEnhance(s state.State) bool {
	val, ok := s.Get(KeyClassState)
	if !ok {
		return false
	}

	cs, ok := val.(ClassificationState)
	if !ok {
		return false
	}

	return cs.NeedsEnhance()
}

func extractResult(s state.State) (*WorkflowResult, error) {
	val, ok := s.Get(KeyClassState)
	if !ok {
		return nil, fmt.Errorf("missing %s in final state", KeyClassState)
	}

	cs, ok := val.(ClassificationState)
	if !ok {
		return nil, fmt.Errorf("%s is not ClassificationState", KeyClassState)
	}

	docIDVal, ok := s.Get(KeyDocumentID)
	if !ok {
		return nil, fmt.Errorf("missing %s in final state", KeyDocumentID)
	}

	documentID, ok := docIDVal.(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("%s is not uuid.UUID", KeyDocumentID)
	}

	filenameVal, ok := s.Get(KeyFilename)
	if !ok {
		return nil, fmt.Errorf("missing %s in final state", KeyFilename)
	}

	filename, ok := filenameVal.(string)
	if !ok {
		return nil, fmt.Errorf("%s is not string", KeyFilename)
	}

	pageCountVal, ok := s.Get(KeyPageCount)
	if !ok {
		return nil, fmt.Errorf("missing %s in final state", KeyPageCount)
	}

	pageCount, ok := pageCountVal.(int)
	if !ok {
		return nil, fmt.Errorf("%s is not int", KeyPageCount)
	}

	return &WorkflowResult{
		DocumentID:  documentID,
		Filename:    filename,
		PageCount:   pageCount,
		State:       cs,
		CompletedAt: time.Now(),
	}, nil
}
```

## Validation Criteria

- [ ] `go vet ./...` passes
- [ ] `go build ./...` passes
- [ ] `EnhanceSettings` struct with `Brightness`, `Contrast`, `Saturation` as `*int`
- [ ] `ClassificationPage` uses `Enhancements *EnhanceSettings` (no separate `Enhance` bool)
- [ ] `NeedsEnhance()` checks for non-nil `Enhancements`
- [ ] Classify spec outputs structured `enhancements` (object or null)
- [ ] Enhance spec outputs only `markings_found` + `rationale` (no enhancement fields)
- [ ] Enhance node re-renders flagged pages with settings-derived `ImageConfig`, reclassifies via vision, clears `Enhancements`
- [ ] Finalize node performs single Chat inference producing document-level classification, confidence, and rationale
- [ ] State graph wired with conditional edge using `ClassificationState.NeedsEnhance()`
- [ ] Single exit point (finalize) — both paths converge to finalize
- [ ] Execute creates temp directory and defers cleanup on all paths
- [ ] Execute function creates initial state, runs graph, returns WorkflowResult
