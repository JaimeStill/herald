# Confidence Scoring Overhaul

> **Execution note:** First action on approval — branch from `main`:
> `git checkout -b confidence-overhaul` (the brief's "branch after #151 merges" precondition is met).
> All work lands on this branch for a single PR at closeout.

## Context

After the `policy-alignment-tuning` work (PR #151) restored classification-string accuracy
following the gpt-5-mini → gpt-5.2 model change, **confidence scoring** remained a structural
weakness that prompt tweaks could not fix. The canonical symptom: `single-secret-noforn-split.pdf`
(clear top `SECRET` banner + near-illegible faded `NOFORN` stamp) produced wildly inconsistent
results across identical runs — `SECRET (HIGH)`, `UNCLASSIFIED (LOW)`, ... — never the correct
`SECRET (LOW, flag for review)`.

**Root cause:** confidence is decided in the wrong stage. `classify` and `enhance` are the only
Vision calls (they see the pixels, so they are the only stages that *can* judge legibility), yet
they emit no confidence signal. `finalize` is a text-only `Chat` call (`finalize.go:59`,
structurally blind to the image) and is the *only* stage that sets confidence — inferring legibility
from prose. This is a structural problem, not a sampling-noise one: gpt-5.2 is a reasoning model, so
`temperature`/`top_p` are unavailable (the API rejects them) and `seed` is only best-effort, but the
real driver is that a blind stage guessing about legibility — fed by upstream vision reads that
themselves flip (`SECRET` ↔ `UNCLASSIFIED` in `markings_found`) — cannot be stabilized by any
sampling knob. The fix is architectural: ground legibility in the vision stages and aggregate in
code. (Model-option tuning is a *complementary* lever, not a substitute — see "Model tuning" below.)

**Goal:** ground the legibility judgment in the stages that see the image, make the signals
structured, and aggregate document confidence deterministically in code. `finalize` keeps the job it
is good at (assembling the banner string from text) and **stops owning confidence**.

## Design

### 1. Per-page legibility signals from the vision stages (`internal/state/state.go`)

Add two fields to `ClassificationPage`:

```go
Confidence           Confidence `json:"confidence,omitempty"`            // per-page legibility of THIS page's markings
UndeterminedMarkings []string   `json:"undetermined_markings,omitempty"` // markings PRESENT but unreadable
```

`EnhanceSettings`/`Enhancements` is **unchanged** and remains both the render-parameter carrier and
the enhancement trigger — `handler.Enhance(...)` consumes the render params, and `Enhance()` /
`NeedsEnhance()` / `EnhancePages()` keep byte-identical semantics. `UndeterminedMarkings` is the
orthogonal *semantic* signal: at classify it names what the model could not read (informational; the
re-render still gates on `Enhancements != nil`); after enhance, a still-non-empty list means "present
but unreadable even after enhancement" → forces the page to LOW.

These per-page fields are **in-memory workflow state only** — they are NOT persisted. Verified:
`internal/classifications/repository.go` upserts only the final scalar `string(result.State.Confidence)`
plus a flattened `markings_found`. No migration, projection, or `Classification`-struct change.

### 2. Per-page confidence at classify (`internal/workflow/classify.go`)

- Add `Confidence Confidence` and `UndeterminedMarkings []string` to `pageResponse`.
- **Remove the dead `Enhance bool` field** — it is parsed but never read (the `Enhancements` pointer
  is the real trigger). Clean to delete in this overhaul.
- `applyPageResponse` (classify.go:127-131) copies the two new fields onto the page.

### 3. Resolution-aware merge at enhance — fixes the overwrite bug (`internal/workflow/enhance.go`)

The current wholesale overwrite (`enhance.go:136-138`) discards a clear banner when the enhanced
re-render obscures it — this is what produced spurious `UNCLASSIFIED`. Replace the
`enhanceResponse` and the merge with a **non-destructive delta**:

```go
type enhanceResponse struct {
    ResolvedMarkings     []string         `json:"resolved_markings"`     // faint markings now readable
    UndeterminedMarkings []string         `json:"undetermined_markings"` // still unreadable after enhancement
    Confidence           state.Confidence `json:"confidence"`
    Rationale            string           `json:"rationale"`
}
```

Merge (replacing lines 136-138):

```go
p := &cs.Pages[i]
p.MarkingsFound = unionMarkings(p.MarkingsFound, parsed.ResolvedMarkings) // preserve prior-confident verbatim, add resolved
p.UndeterminedMarkings = parsed.UndeterminedMarkings
p.Confidence = parsed.Confidence
p.Rationale = parsed.Rationale
p.Enhancements = nil
```

`unionMarkings(prior, resolved)` appends + sort-dedupes (mirror the existing `collectMarkings`
`slices.Sort`+`slices.Compact` idiom in `internal/classifications/repository.go`); define it in
`internal/workflow` (it is a workflow merge concern, not a state concern). This is *resolution-aware*
because prior-confident markings survive unchanged (no re-degradation) and only genuinely-resolved
markings are added — never a blind union of a full page re-read. **Bounded asymmetry:** the enhance
pass may ADD (`resolved_markings`) and FLAG (`undetermined_markings`) but never silently delete a
prior marking; a believed misread goes into `resolved_markings` and finalize's existing
validity/dedup rules collapse the stale token (e.g. `NOFOPI` vs `NOFORN`). A removal channel is
explicitly deferred.

### 4. Deterministic aggregation in code (`internal/state/state.go` + `internal/workflow/finalize.go`)

Add a method on `ClassificationState` (beside `NeedsEnhance`/`EnhancePages`) plus a private
`rank(Confidence) int` helper (`LOW=0 < MEDIUM=1 < HIGH=2`):

```go
func (s *ClassificationState) AggregateConfidence() Confidence
```

Logic (a page is *content-bearing* when `len(MarkingsFound) > 0 || len(UndeterminedMarkings) > 0`):

```
result := HIGH                                  // seed; blank/redacted pages never lower it
for each content-bearing page p:
    pc := p.Confidence
    if pc == ""                       { pc = LOW }   // content page with no signal → conservative, never HIGH by omission
    if len(p.UndeterminedMarkings) > 0 { pc = LOW }  // present-but-unreadable floor
    result = min(result, pc)                          // by rank
return result                                         // no content-bearing pages → stays HIGH
```

**Unmarked-document default = HIGH** (per decision this session): a document the model confidently
reads as having no markings is a confident UNCLASSIFIED. Review triggers only when markings are
actually present-but-unreadable. `AggregateConfidence` is **total** over HIGH/MEDIUM/LOW — it can
never return `""` (the DB `CHECK (confidence IN ('HIGH','MEDIUM','LOW'))` NOT NULL constraint makes
an empty return a crash; the `pc==""→LOW` and seed-HIGH guards make it impossible).

In `finalize.go`: drop `Confidence` from `finalizeResponse` (keep `Classification` + `Rationale`),
remove `cs.Confidence = parsed.Confidence` (line 70), and after `synthesize()` set
`cs.Confidence = cs.AggregateConfidence()`.

### 5. Prompt changes (`internal/prompts/specs.go` + `instructions.go`)

- **`classifySpec`** — add `confidence` and `undetermined_markings` to the JSON example + field
  constraints. Keep `enhancements` exactly as-is (still the render trigger).
- **`enhanceSpec`** — replace the `markings_found`/`rationale` example with
  `resolved_markings`/`undetermined_markings`/`confidence`/`rationale`; **delete the carry-forward
  stopgap bullets** (specs.go:93-95).
- **`enhanceInstructions`** — rewrite the final paragraph (instructions.go:30) from "your response
  replaces the prior findings... carry them all forward" to the delta contract: report only
  newly-resolved markings and still-unreadable regions; code preserves prior markings; report `LOW`
  when undetermined markings remain; never request deletion of a prior marking.
- **`finalizeSpec`** — remove the `confidence` field from the JSON example (specs.go:104-109), the
  entire `confidence:` constraint block (specs.go:140-152), and the trailing "Confidence reflects
  legibility..." bullet (specs.go:170).
- **`finalizeInstructions`** — verify no stray confidence language remains (currently none).

Encode the confidence rule in the vision-stage prompts: HIGH = legible, no guessing (escalation
across pages is normal; a faded marking enhancement *clearly recovers* is still HIGH); MEDIUM =
partially obscured, educated guess but reasonably confident; LOW = a marking is present but its value
can't be determined even after enhancement, or markings largely illegible/missing/contradictory.
Redaction never lowers confidence.

### 6. Model tuning — vision `reasoning_effort`, delivered dev + prod (model-agnostic)

Herald's `agent.model.capabilities` is a pure pass-through map (`map[string]map[string]any`, tau
`protocol/config/model.go`): every key in the `chat`/`vision` blocks is splatted into the Azure
request body. The `chat` block sets `reasoning_effort: high`, but the **`vision` block sets none** —
so the vision reads (where the run-to-run `markings_found` variance originates) run at the deployment
default. The tuning: add `reasoning_effort: high` to the vision capability. `temperature`/`top_p` are
not options for this reasoning model, and `seed` is best-effort only — neither is pursued.

**Deployment-critical:** the deployed container is config-file-free. The Dockerfile bakes only the
binary; `main.bicep` delivers all config via `HERALD_*` env vars, and `tauconfig.DefaultModelConfig()`
ships an **empty** capabilities map — so production currently sends *no* `reasoning_effort` /
`max_completion_tokens` / `vision_options` at all (a pre-existing divergence from the tuned dev
config). A `config.json`-only change would never reach prod. Fix it with a model-agnostic env-var
channel (no hardcoded option keys in Go — Herald must not assume which options a given model
supports):

- **`config.json` (dev source)** — add `reasoning_effort: high` to the `agent.model.capabilities.vision`
  block.
- **`internal/config/agent.go`** — in `loadAgentEnv`, add two env vars parsed as full JSON objects
  into `c.Model.Capabilities["chat"]` / `["vision"]` (init the map if nil; surface a parse error so a
  malformed blob fails fast at startup — `loadAgentEnv` grows an `error` return that `FinalizeAgent`
  propagates):
  - `HERALD_AGENT_CAPABILITIES_CHAT` (e.g. `{"max_completion_tokens":4096,"reasoning_effort":"high"}`)
  - `HERALD_AGENT_CAPABILITIES_VISION` (e.g. `{"max_completion_tokens":4096,"reasoning_effort":"high","vision_options":{"detail":"high"}}`)
  A single blob per capability preserves the flexible pass-through (nested `vision_options` unmarshals
  to `map[string]any`; JSON whole-numbers round-trip cleanly as ints) and keeps the code agnostic of
  any specific model's option set.
- **`deploy/main.bicep`** — add the two env vars to the `envVars` array (lines ~275-294) with the
  model-appropriate values above, alongside the existing `HERALD_AGENT_*` settings. Any other
  deployment manifest (e.g. IL6) must mirror them.

This is complementary to the structural work (steadier upstream reads → cleaner per-page confidence)
and is evaluated empirically during the marked-documents re-run — keep the vision `reasoning_effort`
only if it measurably stabilizes/improves the reads; revert the values if neutral or worse.

## Minor tasks (bundle into this session)

### A. Confidence filter select on the documents view — **frontend only**

The backend is **already fully implemented** — `internal/documents/mapping.go` has
`Filters.Confidence`, `FiltersFromQuery` parses `?confidence=`, `Apply` does
`WhereEquals("Confidence")`, the projection selects `confidence`, and the TS `SearchRequest` already
has `confidence?: string`. Only the client wiring in `app/client/ui/modules/document-grid.ts` is
missing. Mirror the existing **status** filter end-to-end: add `confidence` to `DEFAULTS`, a
`@state` field, `hydrateFromQuery`, `syncQuery`, `fetchDocuments` (`if (this.confidence)
req.confidence = this.confidence`), a `handleConfidenceFilter` handler, and a `<select>` (options
`HIGH`/`MEDIUM`/`LOW`) in the toolbar template next to the status select — using the status select's
`?selected=${this.confidence === "..."}` option pattern.

### B. Page Size select does not reflect the URL param on refresh

The brief's proposed fix (`.value=${String(this.size)}`) **already exists** at
`pagination-controls.ts:98`, and `document-grid.ts:91` already hydrates `pageSize` — yet the select
still shows 12. Root cause is the Lit `<select>`/`.value` commit-order gotcha: the `.value` binding
is applied before the `<option>` children exist, so the browser drops it to the first option. Fix by
mirroring the status select's reliable pattern — add `?selected` to each option in
`app/client/ui/elements/pagination-controls.ts:101-103`:

```ts
${this.sizeOptions.map(
  (n) => html`<option value=${n} ?selected=${n === this.size}>${n}</option>`,
)}
```

`prompt-list.ts` already hydrates `pageSize` correctly and shares `hd-pagination`, so this one fix
covers both views — no `prompt-list.ts` change needed.

## Files modified

- `internal/state/state.go` — `ClassificationPage` += `Confidence` / `UndeterminedMarkings`;
  `AggregateConfidence()` method + `rank()` helper.
- `internal/workflow/classify.go` — `pageResponse` += fields, − dead `Enhance bool`;
  `applyPageResponse`.
- `internal/workflow/enhance.go` — new `enhanceResponse`; resolution-aware merge; `unionMarkings`.
- `internal/workflow/finalize.go` — drop `Confidence` from `finalizeResponse`; compute
  `cs.Confidence = cs.AggregateConfidence()`.
- `internal/prompts/specs.go` + `instructions.go` — per-page confidence/resolution asks; remove
  finalize confidence guidance; remove carry-forward stopgap.
- `config.json` — add `reasoning_effort: high` to the `agent.model.capabilities.vision` block (dev
  source; evaluated during the re-run, kept only if it helps).
- `internal/config/agent.go` — parse `HERALD_AGENT_CAPABILITIES_CHAT` / `_VISION` JSON blobs into the
  capabilities map in `loadAgentEnv` (new env-var constants; `loadAgentEnv`/`FinalizeAgent` return a
  parse error). Model-agnostic — no hardcoded option keys.
- `deploy/main.bicep` — add the two `HERALD_AGENT_CAPABILITIES_*` env vars to the `envVars` array so
  prod receives the capabilities (closes the pre-existing empty-capabilities divergence).
- `app/client/ui/modules/document-grid.ts` — confidence filter wiring (minor task A).
- `app/client/ui/elements/pagination-controls.ts` — `?selected` on size options (minor task B).

### Tests (`tests/`, AI-authored per project conventions)

- `tests/state/state_test.go` — **PRIMARY.** New `TestAggregateConfidence` (table-driven): all-HIGH
  → HIGH; one MEDIUM → MEDIUM; content page with non-empty `UndeterminedMarkings` → LOW floor even
  if its `Confidence` is HIGH; blank pages ignored (HIGH content + blanks → HIGH); **all-blank doc →
  HIGH**; content page with `Confidence==""` → LOW; `nil` pages → HIGH; assert the result is never
  `""`. Extend `TestClassificationStateJSON` for the new per-page fields (round-trip + omitempty).
- `tests/workflow/prompts_test.go` — add an assertion that serialized state carries per-page
  `confidence`/`undetermined_markings` when set.
- `tests/prompts/prompts_test.go` — optional contract locks: `enhanceSpec` contains
  `resolved_markings`; `finalizeSpec` no longer contains `confidence`.
- `tests/config/*` — add coverage for the two new capability env vars: valid JSON blob populates
  `Capabilities["chat"]`/`["vision"]`, nested `vision_options` preserved, malformed JSON surfaces an
  error, unset env leaves config-file values intact.
- `tests/classifications/*` — no change (document-level confidence filter/column untouched).

## Verification

- `mise run vet`, `mise run test`, `mise run web:build` clean.
- Re-run the full `_project/marked-documents/` set via `mise run dev`: classification strings remain
  correct (no regression from the accuracy work) **and** confidence now follows the rule.
  `single-secret-noforn-split.pdf` is the canonical case → `SECRET` at **LOW** when the bottom stamp
  can't be read; HIGH only if enhancement genuinely recovers it.
- Evaluate the vision `reasoning_effort: high` tuning across the re-run: confirm it stabilizes/improves
  the upstream reads (less run-to-run `markings_found` drift). Keep it only if it measurably helps;
  revert the values if neutral or worse.
- Confirm the capability env-var channel: with `HERALD_AGENT_CAPABILITIES_VISION` set, the composed
  request carries `reasoning_effort`/`vision_options` (and a malformed blob fails startup cleanly).
  Verify `deploy/main.bicep` carries both vars so the deployed service no longer runs on empty
  capabilities.
- In the local app: the new confidence `<select>` narrows the documents list (HIGH/MEDIUM/LOW); load
  `/app/?page_size=24` then refresh and confirm the Page Size select now displays `24`.
