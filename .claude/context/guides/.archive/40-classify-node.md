# 40 — Classify Node

## Problem Context

The classify node is the core of the classification workflow. It processes pages sequentially so each page's findings feed into the next page's prompt (context accumulation) — adapted from the classify-docs pattern that achieved 96.3% accuracy. The classify node only populates per-page `ClassificationPage` data; document-level classification synthesis is deferred to a finalize node (#41).

The hard-coded specs and prompts domain also need alignment with the optimized workflow types and the new 4-node topology (init → classify → enhance? → finalize).

## Architecture Approach

- **Per-page analysis only**: classify populates `ClassificationPage` fields; `ClassificationState` document-level fields are left empty for the finalize node
- **Context accumulation via ComposePrompt**: first page passes `nil` (no prior context), subsequent pages pass `&classState` so the model sees prior page findings
- **Just-in-time image encoding**: PNG read from disk, encoded to data URI, bytes released after encoding — one page image in memory at a time
- **Per-request agent creation**: fresh `agent.Agent` per `Execute` call
- **Page-only classify spec**: response schema aligned with `ClassificationPage` fields
- **New finalize stage**: `StageFinalize` added to prompts domain with its own spec and instructions

## Implementation

### Step 1: Add `StageFinalize` (`internal/prompts/stages.go`)

Add the finalize constant and include it in the stages slice:

```go
const (
	StageClassify  Stage = "classify"
	StageEnhance   Stage = "enhance"
	StageFinalize  Stage = "finalize"
)

var stages = []Stage{
	StageClassify,
	StageEnhance,
	StageFinalize,
}
```

### Step 2: Update classify spec, add finalize spec (`internal/prompts/specs.go`)

Replace `classifySpec` with a page-only response format aligned with `ClassificationPage` fields:

```go
const classifySpec = `Respond with a JSON object matching this exact structure:

{
  "markings_found": ["<marking1>", "<marking2>"],
  "rationale": "<explanation>",
  "enhance": false,
  "enhancements": ""
}

Field constraints:
- markings_found: Array of distinct marking strings found on this page,
  exactly as they appear in the document. Include the full marking text
  with any caveats (e.g., "SECRET//NOFORN" not just "SECRET").
- rationale: Brief explanation of what security markings were found on
  this page and their significance. Note any conflicts or ambiguities
  with prior page findings if a classification state is provided.
- enhance: Whether image quality prevented confident reading of any
  markings on this page. Set true only when markings are visibly present
  but cannot be read with certainty due to image quality.
- enhancements: When enhance is true, describe the specific image quality
  issues and what adjustments might help (e.g., "faded banner markings —
  increase brightness and contrast"). Empty string when enhance is false.

Behavioral constraints:
- Always respond with valid JSON, no markdown fencing
- Process exactly one page per response
- Report only what you observe on this page
- If prior page findings are provided in the prompt, use them as context
  to identify consistency or conflicts, but do not repeat prior findings
  in markings_found — only include markings visible on the current page`
```

Replace `enhanceSpec` with an aligned version:

```go
const enhanceSpec = `Respond with a JSON object matching this exact structure:

{
  "markings_found": ["<marking1>", "<marking2>"],
  "rationale": "<explanation>",
  "enhance": false,
  "enhancements": ""
}

Field constraints:
- markings_found: Array of distinct marking strings found on this
  enhanced page, exactly as they appear. Include the full marking text
  with any caveats.
- rationale: Brief explanation of what the enhanced image reveals compared
  to the original. Note any new markings discovered or prior findings
  confirmed by the improved image quality.
- enhance: Whether image quality still prevents confident reading after
  enhancement. Should typically be false after enhancement processing.
- enhancements: Description of remaining quality issues if enhance is
  still true. Empty string when enhance is false.

Behavioral constraints:
- Always respond with valid JSON, no markdown fencing
- Focus analysis on the enhanced image with improved rendering settings
- Compare findings against the prior page analysis provided in the prompt
- Report only what you observe on the current enhanced page`
```

Add `finalizeSpec`:

```go
const finalizeSpec = `Respond with a JSON object matching this exact structure:

{
  "classification": "<marking>",
  "confidence": "<HIGH|MEDIUM|LOW>",
  "rationale": "<explanation>"
}

Field constraints:
- classification: The overall security classification marking for the
  document, synthesized from all page findings (e.g., UNCLASSIFIED,
  CONFIDENTIAL, SECRET, TOP SECRET, or with caveats like SECRET//NOFORN).
  Apply the highest classification encountered across all pages.
- confidence: Categorical assessment of classification certainty.
  HIGH = markings are clear, consistent, and unambiguous across pages.
  MEDIUM = markings are present but partially obscured or inconsistent.
  LOW = markings are unclear, missing, or contradictory.
- rationale: Comprehensive explanation of the document classification
  synthesized from all page findings. Reference specific page evidence,
  note any cross-page conflicts, and explain how the final classification
  was determined.

Behavioral constraints:
- Always respond with valid JSON, no markdown fencing
- Consider all page findings holistically when determining classification
- Apply the highest classification encountered across all pages
- Never downgrade based on pages with lower or missing markings
- Confidence reflects the overall clarity and consistency across all pages,
  not just the most recent page analyzed`
```

Update the `specs` map:

```go
var specs = map[Stage]string{
	StageClassify: classifySpec,
	StageEnhance:  enhanceSpec,
	StageFinalize: finalizeSpec,
}
```

### Step 3: Add finalize instructions (`internal/prompts/instructions.go`)

Add `finalizeInstructions`:

```go
const finalizeInstructions = `You are a security classification analyst producing the final document classification.

Review all per-page analysis results provided in the classification state. Each page entry contains the markings found on that page and a rationale explaining the findings. Synthesize these per-page results into a single authoritative document classification.

When determining the overall classification:
- Apply the highest classification marking encountered across all pages
- Consider the full marking text including caveats (e.g., NOFORN, REL TO, FOUO)
- Resolve any cross-page conflicts by applying the most restrictive interpretation
- Base your confidence on the overall clarity and consistency of markings across all pages`
```

Update the `instructions` map:

```go
var instructions = map[Stage]string{
	StageClassify: classifyInstructions,
	StageEnhance:  enhanceInstructions,
	StageFinalize: finalizeInstructions,
}
```

### Step 4: Database migration

Create `cmd/migrate/migrations/000005_prompts_add_finalize_stage.up.sql`:

```sql
ALTER TABLE prompts DROP CONSTRAINT prompts_stage_check;

ALTER TABLE prompts
  ADD CONSTRAINT prompts_stage_check
  CHECK (stage IN ('classify', 'enhance', 'finalize'));
```

Create `cmd/migrate/migrations/000005_prompts_add_finalize_stage.down.sql`:

```sql
DELETE FROM prompts WHERE stage = 'finalize';

ALTER TABLE prompts DROP CONSTRAINT prompts_stage_check;

ALTER TABLE prompts
  ADD CONSTRAINT prompts_stage_check
  CHECK (stage IN ('classify', 'enhance'));
```

### Step 5: Create `workflow/classify.go`

Complete new file:

```go
package workflow

import (
	"context"
	"fmt"
	"os"

	"github.com/JaimeStill/document-context/pkg/document"
	"github.com/JaimeStill/document-context/pkg/encoding"

	"github.com/JaimeStill/go-agents/pkg/agent"

	"github.com/JaimeStill/go-agents-orchestration/pkg/state"

	"github.com/JaimeStill/herald/internal/prompts"
	"github.com/JaimeStill/herald/pkg/formatting"
)

type pageResponse struct {
	MarkingsFound []string `json:"markings_found"`
	Rationale     string   `json:"rationale"`
	Enhance       bool     `json:"enhance"`
	Enhancements  string   `json:"enhancements"`
}

func ClassifyNode(rt *Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		classState, err := extractClassState(s)
		if err != nil {
			return s, fmt.Errorf("classify: %w", err)
		}

		if err := classifyPages(ctx, rt, classState); err != nil {
			return s, fmt.Errorf("classify: %w", err)
		}

		rt.Logger.InfoContext(
			ctx, "classify node complete",
			"page_count", len(classState.Pages),
		)

		s = s.Set(KeyClassState, *classState)
		return s, nil
	})
}

func extractClassState(s state.State) (*ClassificationState, error) {
	val, ok := s.Get(KeyClassState)
	if !ok {
		return nil, fmt.Errorf("%w: missing %s in state", ErrClassifyFailed, KeyClassState)
	}

	cs, ok := val.(ClassificationState)
	if !ok {
		return nil, fmt.Errorf("%w: %s is not ClassificationState", ErrClassifyFailed, KeyClassState)
	}

	return &cs, nil
}

func classifyPages(ctx context.Context, rt *Runtime, cs *ClassificationState) error {
	a, err := agent.New(&rt.Agent)
	if err != nil {
		return fmt.Errorf("%w: create agent: %w", ErrClassifyFailed, err)
	}

	for i := range cs.Pages {
		if err := classifyPage(ctx, a, rt, cs, i); err != nil {
			return fmt.Errorf("%w: page %d: %w", ErrClassifyFailed, i+1, err)
		}

		rt.Logger.InfoContext(
			ctx, "page classified",
			"page", i+1,
			"total", len(cs.Pages),
			"markings", cs.Pages[i].MarkingsFound,
			"enhance", cs.Pages[i].Enhance,
		)
	}

	return nil
}

func classifyPage(ctx context.Context, a agent.Agent, rt *Runtime, cs *ClassificationState, pageIdx int) error {
	dataURI, err := encodePageImage(cs.Pages[pageIdx].ImagePath)
	if err != nil {
		return err
	}

	var promptState *ClassificationState
	if pageIdx > 0 {
		promptState = cs
	}

	prompt, err := ComposePrompt(ctx, rt.Prompts, prompts.StageClassify, promptState)
	if err != nil {
		return err
	}

	resp, err := a.Vision(ctx, prompt, []string{dataURI})
	if err != nil {
		return fmt.Errorf("vision call: %w", err)
	}

	parsed, err := formatting.Parse[pageResponse](resp.Content())
	if err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	applyPageResponse(&cs.Pages[pageIdx], parsed)
	return nil
}

func encodePageImage(imagePath string) (string, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("read image: %w", err)
	}

	dataURI, err := encoding.EncodeImageDataURI(data, document.PNG)
	if err != nil {
		return "", fmt.Errorf("encode image: %w", err)
	}

	return dataURI, nil
}

func applyPageResponse(page *ClassificationPage, resp pageResponse) {
	page.MarkingsFound = resp.MarkingsFound
	page.Rationale = resp.Rationale
	page.Enhance = resp.Enhance
	page.Enhancements = resp.Enhancements
}
```

## Validation Criteria

- [ ] `go vet ./...` passes
- [ ] `go build ./...` passes
- [ ] Pages processed sequentially with context accumulation (prior page findings in prompt)
- [ ] Each page response populates only `ClassificationPage` fields
- [ ] Images loaded from disk and encoded just-in-time (one at a time)
- [ ] Vision API called with base64 data URI per page
- [ ] JSON response parsed with code fence fallback via `formatting.Parse`
- [ ] ClassificationState stored in state with populated pages for downstream finalize
- [ ] Classify spec is page-only, finalize spec covers document-level synthesis
- [ ] `StageFinalize` added to stages, specs, instructions, and migration
