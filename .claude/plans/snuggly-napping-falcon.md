# 40 — Classify Node

## Context

The classify node processes pages sequentially so each page's findings inform the next page's prompt (context accumulation). This session also introduces the 4-node workflow topology: `init → classify → enhance? → finalize`. The classify node only populates per-page `ClassificationPage` data. A new finalize node (issue #41) will synthesize document-level `ClassificationState` from all page findings in a single inference.

The hard-coded specs were written before type optimization and need alignment. The `finalize` stage needs to be introduced across the prompts domain (stages, specs, instructions, migration).

## Files

| File | Action | Purpose |
|------|--------|---------|
| `internal/prompts/stages.go` | Modify | Add `StageFinalize` |
| `internal/prompts/specs.go` | Modify | Page-only classify spec, add finalize spec |
| `internal/prompts/instructions.go` | Modify | Add finalize instructions |
| `cmd/migrate/migrations/000005_prompts_add_finalize_stage.up.sql` | Create | Add `finalize` to CHECK constraint |
| `cmd/migrate/migrations/000005_prompts_add_finalize_stage.down.sql` | Create | Reverse migration |
| `workflow/classify.go` | Create | ClassifyNode implementation |

## Step 1: Add `StageFinalize` (`internal/prompts/stages.go`)

Add `StageFinalize Stage = "finalize"` constant and include it in the `stages` slice.

## Step 2: Update classify spec, add finalize spec (`internal/prompts/specs.go`)

**Classify spec** — page-only response aligned with `ClassificationPage` fields:
```json
{
  "markings_found": ["<marking1>", "<marking2>"],
  "rationale": "<explanation>",
  "enhance": false,
  "enhancements": ""
}
```

Field constraints describe per-page analysis. Behavioral constraints emphasize: analyze one page, report what you find, flag quality issues via `enhance`/`enhancements` for downstream processing. Context accumulation note: prior page findings are provided in the prompt but the model does not produce document-level classification.

**Finalize spec** — document-level synthesis aligned with `ClassificationState` fields:
```json
{
  "classification": "<marking>",
  "confidence": "<HIGH|MEDIUM|LOW>",
  "rationale": "<explanation>"
}
```

Field constraints describe document-level synthesis from all page findings. Behavioral constraints: apply highest classification across all pages, never downgrade, assess confidence based on clarity and consistency of all markings found.

Add both to the `specs` map.

## Step 3: Add finalize instructions (`internal/prompts/instructions.go`)

Add `finalizeInstructions` — instructs the model to review all per-page analysis results and produce the authoritative document-level classification, confidence, and rationale. Add to `instructions` map.

## Step 4: Migration (`cmd/migrate/migrations/000005_*`)

**Up:**
```sql
ALTER TABLE prompts DROP CONSTRAINT prompts_stage_check;
ALTER TABLE prompts ADD CONSTRAINT prompts_stage_check
  CHECK (stage IN ('classify', 'enhance', 'finalize'));
```

**Down:**
```sql
DELETE FROM prompts WHERE stage = 'finalize';
ALTER TABLE prompts DROP CONSTRAINT prompts_stage_check;
ALTER TABLE prompts ADD CONSTRAINT prompts_stage_check
  CHECK (stage IN ('classify', 'enhance'));
```

## Step 5: Create `workflow/classify.go`

### Exported function

`ClassifyNode(rt *Runtime) state.StateNode` — returns `state.NewFunctionNode` closure.

### Flow

1. Extract `ClassificationState` from state bag via `KeyClassState`
2. Create agent via `agent.New(&rt.Agent)` (per-request creation)
3. Classify all pages sequentially via `classifyPages`
4. Store updated `ClassificationState` in state bag
5. Log node completion

### Unexported types and helpers

**`pageResponse`** — intermediate parsing type matching the page-only spec:
```go
type pageResponse struct {
    MarkingsFound []string `json:"markings_found"`
    Rationale     string   `json:"rationale"`
    Enhance       bool     `json:"enhance"`
    Enhancements  string   `json:"enhancements"`
}
```

**`classifyPages(ctx, agent, rt, *ClassificationState) error`** — sequential loop. For each page: encode image, compose prompt, call vision, parse, apply response.

**`classifyPage(ctx, agent, rt, *ClassificationState, pageIndex) error`** — single page processing:
1. Read PNG from `page.ImagePath` via `os.ReadFile`
2. Encode to data URI via `encoding.EncodeImageDataURI(data, document.PNG)`
3. Compose prompt: first page passes `nil` to `ComposePrompt`, subsequent pages pass `&classState` (model sees prior page findings as context)
4. Call `agent.Vision(ctx, prompt, []string{dataURI})`
5. Parse via `formatting.Parse[pageResponse](resp.Content())`
6. Apply response to `ClassificationPage` fields only

**`encodePageImage(imagePath string) (string, error)`** — read + encode, bytes released after.

**`applyPageResponse(page *ClassificationPage, resp pageResponse)`** — maps response to page fields (markings_found, rationale, enhance, enhancements).

### Key dependencies

| Package | Import | Usage |
|---------|--------|-------|
| `go-agents/pkg/agent` | `agent.New`, `agent.Vision` | Per-request agent creation, vision calls |
| `document-context/pkg/document` | `document.PNG` | Image format constant |
| `document-context/pkg/encoding` | `encoding.EncodeImageDataURI` | Base64 data URI encoding |
| `go-agents-orchestration/pkg/state` | `state.StateNode`, `state.NewFunctionNode` | Node interface |
| `herald/internal/prompts` | `prompts.StageClassify` | Stage constant |
| `herald/pkg/formatting` | `formatting.Parse[pageResponse]` | JSON parsing with code fence fallback |

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
