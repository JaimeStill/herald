# 37 - Prompts Domain Extensions

## Problem Context

The classification workflow (#26) needs a single source of truth for all prompt content. The workflow composes each stage's system prompt from two layers: tunable instructions (active DB override or hardcoded default) and immutable specifications (output JSON schema + behavioral constraints). The prompts domain currently only handles CRUD for named overrides — it has no way to resolve effective instructions for a stage or expose specifications. This issue adds both capabilities plus API endpoints for prompt authors to view specs when crafting instructions.

## Architecture Approach

Extend the existing prompts domain in-place. Hardcoded content lives in two new files (`instructions.go`, `specs.go`) with named constants and map-based accessors. Two new methods on `System` (`Instructions`, `Spec`) provide the workflow's entry points. Two new handler routes expose the content via API.

Only the classify and enhance stages carry prompt content — init is purely image preparation with no LLM interaction. `StageInit` is removed from the prompts domain entirely (including a migration to tighten the DB CHECK constraint) since it was never a valid prompt stage.

## Implementation

### Step 1: Remove `StageInit` from `internal/prompts/stages.go`

Remove the `StageInit` constant and its entry in the `stages` slice. Update the file to only contain classify and enhance.

The full updated file:

```go
package prompts

import (
	"encoding/json"
	"slices"
)

type Stage string

const (
	StageClassify Stage = "classify"
	StageEnhance  Stage = "enhance"
)

var stages = []Stage{
	StageClassify,
	StageEnhance,
}

func Stages() []Stage {
	return stages
}

func (s *Stage) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	v := Stage(raw)
	if !slices.Contains(stages, v) {
		return ErrInvalidStage
	}
	*s = v
	return nil
}

func ParseStage(s string) (Stage, error) {
	v := Stage(s)
	if !slices.Contains(stages, v) {
		return "", ErrInvalidStage
	}
	return v, nil
}
```

### Step 2: Update `internal/prompts/errors.go`

Update the `ErrInvalidStage` message to only list classify and enhance:

```go
ErrInvalidStage = errors.New("stage must be classify or enhance")
```

### Step 3: Add migration `000004_prompts_remove_init_stage`

**`cmd/migrate/migrations/000004_prompts_remove_init_stage.up.sql`**:

```sql
DELETE FROM prompts WHERE stage = 'init';

ALTER TABLE prompts
  DROP CONSTRAINT prompts_stage_check;

ALTER TABLE prompts
  ADD CONSTRAINT prompts_stage_check
  CHECK (stage IN ('classify', 'enhance'));
```

**`cmd/migrate/migrations/000004_prompts_remove_init_stage.down.sql`**:

```sql
ALTER TABLE prompts
  DROP CONSTRAINT prompts_stage_check;

ALTER TABLE prompts
  ADD CONSTRAINT prompts_stage_check
  CHECK (stage IN ('init', 'classify', 'enhance'));
```

### Step 4: Create `internal/prompts/instructions.go`

New file with per-stage instruction constants, the lookup map, and accessor function.

```go
package prompts

const classifyInstructions = `You are a security classification analyst reviewing a document page by page.

For each page, examine all visible security markings including:
- Banner lines (top and bottom of page)
- Portion markings (paragraph-level classification indicators)
- Classification authority blocks
- Declassification instructions
- Caveats and dissemination controls (e.g., NOFORN, REL TO, FOUO)

Accumulate your findings across pages to build a complete classification picture of the
document. When markings on different pages conflict, note the discrepancy and apply the
highest classification encountered. Your confidence assessment should reflect the clarity
and consistency of the markings found.`

const enhanceInstructions = `You are re-assessing a document's security classification using enhanced page images.

Previous classification analysis was limited by image quality. The affected pages have been
re-rendered with adjusted brightness, contrast, and saturation settings to improve marking
visibility. Focus your analysis on the enhanced pages, looking for markings that may have
been obscured in the original rendering.

Compare your findings against the prior classification state. If the enhanced images reveal
additional or different markings, update the classification accordingly. If the enhanced
images confirm the prior assessment, maintain the existing classification with increased
confidence.`

var instructions = map[Stage]string{
	StageClassify: classifyInstructions,
	StageEnhance:  enhanceInstructions,
}

func Instructions(stage Stage) (string, error) {
	text, ok := instructions[stage]
	if !ok {
		return "", ErrInvalidStage
	}
	return text, nil
}
```

### Step 5: Create `internal/prompts/specs.go`

New file with per-stage specification constants, the lookup map, and accessor function.

```go
package prompts

const classifySpec = `Respond with a JSON object matching this exact structure:

{
  "classification": "<marking>",
  "confidence": "<HIGH|MEDIUM|LOW>",
  "markings_found": ["<marking1>", "<marking2>"],
  "rationale": "<explanation>",
  "page_number": <number>,
  "image_quality_limiting": <true|false>
}

Field constraints:
- classification: The overall security classification marking for the document
  as assessed through this page (e.g., UNCLASSIFIED, CONFIDENTIAL, SECRET,
  TOP SECRET, or with caveats like SECRET//NOFORN).
- confidence: Categorical assessment of classification certainty.
  HIGH = markings are clear, consistent, and unambiguous.
  MEDIUM = markings are present but partially obscured or inconsistent.
  LOW = markings are unclear, missing, or contradictory.
- markings_found: Array of distinct marking strings found on this page,
  exactly as they appear in the document.
- rationale: Brief explanation of how the classification was determined
  from the visible markings. Note any conflicts or ambiguities.
- page_number: The 1-indexed page number being analyzed.
- image_quality_limiting: Whether image quality prevented confident
  reading of any markings on this page. true triggers potential
  enhancement processing.

Behavioral constraints:
- Always respond with valid JSON, no markdown fencing
- Process exactly one page per response
- Accumulate classification state: if a prior state is provided in the
  prompt, update it based on this page's findings
- Apply the highest classification encountered across all pages
- Never downgrade a classification based on a subsequent page`

const enhanceSpec = `Respond with a JSON object matching this exact structure:

{
  "classification": "<marking>",
  "confidence": "<HIGH|MEDIUM|LOW>",
  "markings_found": ["<marking1>", "<marking2>"],
  "rationale": "<explanation>",
  "page_number": <number>,
  "image_quality_limiting": <true|false>,
  "enhanced": true,
  "prior_confidence": "<HIGH|MEDIUM|LOW>"
}

Field constraints:
- All fields from the classify specification apply, plus:
- enhanced: Always true in enhancement responses.
- prior_confidence: The confidence level from the classify stage that
  triggered enhancement. Used to track whether enhancement improved
  the assessment.

Behavioral constraints:
- Always respond with valid JSON, no markdown fencing
- Focus analysis on re-rendered pages with improved image settings
- Compare findings against the prior classification state
- Maintain or upgrade classification; never downgrade
- If enhanced images reveal no new information, preserve the prior
  classification and note this in the rationale`

var specs = map[Stage]string{
	StageClassify: classifySpec,
	StageEnhance:  enhanceSpec,
}

func Spec(stage Stage) (string, error) {
	text, ok := specs[stage]
	if !ok {
		return "", ErrInvalidStage
	}
	return text, nil
}
```

### Step 6: Extend `internal/prompts/system.go`

Add two methods to the `System` interface, after `Deactivate`:

```go
Instructions(ctx context.Context, stage Stage) (string, error)
Spec(ctx context.Context, stage Stage) (string, error)
```

### Step 7: Implement methods on `internal/prompts/repository.go`

Add these two methods to the `repo` type. `"database/sql"` and `"errors"` are already imported.

```go
func (r *repo) Instructions(ctx context.Context, stage Stage) (string, error) {
	var text string
	err := r.db.QueryRowContext(ctx,
		"SELECT instructions FROM prompts WHERE stage = $1 AND active = true",
		stage,
	).Scan(&text)

	if errors.Is(err, sql.ErrNoRows) {
		return Instructions(stage)
	}
	if err != nil {
		return "", fmt.Errorf("query active instructions: %w", err)
	}
	return text, nil
}

func (r *repo) Spec(ctx context.Context, stage Stage) (string, error) {
	return Spec(stage)
}
```

### Step 8: Add handler routes to `internal/prompts/handler.go`

Add the `StageContent` response type after `SearchRequest`:

```go
type StageContent struct {
	Stage   Stage  `json:"stage"`
	Content string `json:"content"`
}
```

Add two new handler methods:

```go
func (h *Handler) Instructions(w http.ResponseWriter, r *http.Request) {
	stage, err := ParseStage(r.PathValue("stage"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	text, err := h.sys.Instructions(r.Context(), stage)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, StageContent{Stage: stage, Content: text})
}

func (h *Handler) Spec(w http.ResponseWriter, r *http.Request) {
	stage, err := ParseStage(r.PathValue("stage"))
	if err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	text, err := h.sys.Spec(r.Context(), stage)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	handlers.RespondJSON(w, http.StatusOK, StageContent{Stage: stage, Content: text})
}
```

Add two new routes to the `Routes()` method's slice:

```go
{Method: "GET", Pattern: "/{stage}/instructions", Handler: h.Instructions},
{Method: "GET", Pattern: "/{stage}/spec", Handler: h.Spec},
```

### Step 9: Update API Cartographer docs

Add two new sections to `_project/api/prompts/README.md` after the **Deactivate Prompt** section:

```markdown
---

## Get Stage Instructions

`GET /api/prompts/{stage}/instructions`

Returns the effective instructions for a workflow stage. If an active prompt override exists for the stage, returns its instructions. Otherwise, returns the hardcoded default instructions. Only classify and enhance stages have prompt content.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| stage | string | Workflow stage (classify, enhance) |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Stage instructions |
| 400 | Invalid stage |

### Response Body

| Field | Type | Description |
|-------|------|-------------|
| stage | string | The requested stage |
| content | string | The effective instruction text |

### Example

```bash
curl -s "$HERALD_API_BASE/api/prompts/classify/instructions" | jq .
```

---

## Get Stage Specification

`GET /api/prompts/{stage}/spec`

Returns the hardcoded specification for a workflow stage. Specifications define the expected output format and behavioral constraints that the workflow parser depends on. Always read-only. Only classify and enhance stages have prompt content.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| stage | string | Workflow stage (classify, enhance) |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Stage specification |
| 400 | Invalid stage |

### Response Body

| Field | Type | Description |
|-------|------|-------------|
| stage | string | The requested stage |
| content | string | The specification text |

### Example

```bash
curl -s "$HERALD_API_BASE/api/prompts/classify/spec" | jq .
```
```

Update the **List Prompts** and **Search Prompts** sections to reference stages as `(classify, enhance)` instead of `(init, classify, enhance)`.

Update the **List Stages** description to note that it returns the valid prompt workflow stages.

Update the **Create Prompt** and **Update Prompt** stage field description to `(classify, enhance)`.

Add to `_project/api/prompts/prompts.http` after the Delete Prompt section:

```
### Get Stage Instructions
GET {{HOST}}/api/prompts/classify/instructions

### Get Stage Specification
GET {{HOST}}/api/prompts/classify/spec
```

## Validation Criteria

- [ ] `StageInit` removed from prompts package — only classify and enhance are valid
- [ ] Migration `000004` tightens CHECK constraint to `('classify', 'enhance')`
- [ ] `Instructions(ctx, stage)` returns active DB override when one exists
- [ ] `Instructions(ctx, stage)` returns hardcoded default when no active override exists
- [ ] `Spec(ctx, stage)` returns hardcoded specification
- [ ] Both methods return `ErrInvalidStage` for unknown stages and never return empty
- [ ] `ParseStage` validates string input against known stages
- [ ] Handler routes return `StageContent` JSON with 200 OK
- [ ] Handler routes return 400 Bad Request for invalid stage
- [ ] `go vet ./...` passes
- [ ] All existing tests pass
- [ ] API docs updated with new endpoints and stage references
