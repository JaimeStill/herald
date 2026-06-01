# Confidence Scoring Overhaul

Initialization brief for the next development session. Branch from `main` after the
`policy-alignment-tuning` PR (this session's classification-accuracy work) merges.

## Context

The prior session (`policy-alignment-tuning`) retuned the classification workflow prompts and
restored classification-string accuracy across `_project/marked-documents/` after the gpt-5-mini →
gpt-5.2 model change. During validation, **confidence scoring** surfaced as a separate, structural
weakness that prompt tweaks could not reliably fix — it is deferred to this session.

Concrete symptom that motivated this work: `single-secret-noforn-split.pdf` (single page, a clearly
legible top `SECRET` banner plus a near-illegible faded `NOFORN` stamp at the bottom) produced
wildly inconsistent results across identical runs — `SECRET (HIGH)`, `UNCLASSIFIED (LOW)`,
`SECRET (HIGH)`, `UNCLASSIFIED (LOW)` — and never the correct `SECRET (LOW, flag for review)`.

## Root cause (the structural problem)

Confidence is decided in the wrong stage:

- **`classify` and `enhance` are Vision calls** — they are the only stages that see the pixels, so
  they are the only stages that *can* judge legibility. Their response structs (`pageResponse`,
  `enhanceResponse`) emit **no confidence signal** — only `markings_found` + `rationale`.
- **`finalize` is a Chat call with no images** (`internal/workflow/finalize.go:59`, `a.Chat`). It is
  the *only* stage that sets confidence, and it does so purely from the serialized text. It is
  inferring legibility from prose descriptions of markings it never saw.

So confidence is a guess-about-a-guess produced by the one stage structurally blind to what
confidence measures. That is why it is noisy run-to-run and why the "a marking is present but
unreadable" signal (which we tried to carry in free-text `rationale`) gets lost — `finalize` has to
parse prose to find it. Note gpt-5.2 is a reasoning model (temperature largely fixed,
`reasoning_effort: high`), so we cannot dial variance down via sampling settings — which is exactly
why confidence must move to deterministic code aggregation.

## Goal

Ground legibility judgment in the stages that see the image, make the signals structured, and
aggregate confidence deterministically in code. `finalize` keeps the job it is genuinely good at
(assembling the banner string from text) and **stops owning confidence**.

## Design (primary work)

1. **Per-page confidence/legibility emitted by the vision stages.** Add a structured field (e.g.,
   `confidence` and/or `undetermined_markings: []`) to `pageResponse` / `enhanceResponse` and to
   `ClassificationPage` in `internal/state/state.go`. Classify/enhance populate it *while looking at
   the image* — grounded, not inferred. This is the load-bearing change.
2. **Structured enhancement outcome.** `enhance` reports, per flagged marking, `resolved` vs
   `undetermined` as a field (not buried in rationale).
3. **Deterministic confidence aggregation in code.** Document confidence = the most conservative
   per-page signal among content-bearing pages (redacted/blank pages ignored). Drop `confidence`
   from the `finalize` chat output (`finalizeResponse` keeps `classification` + `rationale`); the
   workflow computes it.

### Confidence rule (decided in the prior session)

Confidence reflects the legibility of the **security markings themselves** (banner lines, portion
marks), not page-to-page uniformity and not the state of the page body.

- **HIGH** — markings legible, classification directly determinable with no guessing. Escalation
  across pages is normal and does not lower it. **A faded marking that enhancement *clearly
  recovers* can still be HIGH.**
- **MEDIUM** — some markings partially obscured, required an educated guess, but reasonably
  confident the inferred value is correct.
- **LOW** — a marking is clearly *present* but its value **cannot be determined even after
  enhancement** (flag for human review), OR markings are largely illegible / missing / genuinely
  contradictory.
- **Redaction** (blacked-out content) is a normal, valid state — it never lowers confidence and
  never affects the classification.

Canonical test: `single-secret-noforn-split.pdf` → `SECRET` at **LOW** when the bottom stamp can't
be read; HIGH only if enhancement ever genuinely recovers it.

## Findings from the prior session to incorporate (do these properly here)

- **Enhance overwrite bug (code).** `internal/workflow/enhance.go:136-138` overwrites
  `cs.Pages[i].MarkingsFound` / `.Rationale` wholesale with the enhanced read. When a whole-page
  re-render obscures a clearly-legible banner while chasing a faint stamp, the clear marking is
  *discarded* — this is what produced `UNCLASSIFIED`. A prompt-level carry-forward stopgap currently
  lives in `enhanceInstructions` / `enhanceSpec`; **replace it with a proper code-level
  merge/reconcile** (preserve prior-confident markings, add/refine in the targeted region) and
  remove the stopgap prose once the code handles it. A blind union is wrong — it would re-introduce
  degraded tokens the resolution logic fixed (e.g., `NOFOPI` alongside `NOFORN`); the merge must be
  resolution-aware.
- **Brightness direction (kept this session).** Faint gray markings on a light background need
  *lower* brightness + *higher* contrast; raise brightness only for dark/underexposed pages. Already
  in `classifySpec`; keep/refine.
- **Faint-marking detection / enhancement trigger (kept this session, "batch 4").** `classify` flags
  enhancement when marking-position text is too faint to read confidently. This is the entry point
  that feeds the resolved/undetermined determination. Validate it does not over-trigger on clean
  documents (full-set re-run) and refine if needed.
- **present-but-unreadable → LOW (removed this session).** We implemented this in prose and stripped
  it because it is purely confidence and proved unreliable. **Reimplement it as a structured signal**
  feeding the deterministic aggregation above.

## Files likely touched (primary)

- `internal/state/state.go` — `ClassificationPage` += per-page confidence / undetermined fields.
- `internal/workflow/classify.go` — `pageResponse` += confidence; `applyPageResponse`.
- `internal/workflow/enhance.go` — `enhanceResponse` += resolution/confidence; **fix the overwrite**
  (resolution-aware merge with prior findings).
- `internal/workflow/finalize.go` — deterministic doc-confidence aggregation; drop `confidence` from
  the chat output.
- `internal/prompts/specs.go` + `instructions.go` — ask the vision stages for per-page
  confidence/resolution; reconcile the finalize confidence guidance (finalize no longer self-reports
  it); remove the carry-forward stopgap prose once code handles merge.
- `tests/` — update `tests/workflow`, `tests/state`, `tests/prompts` as the structs/behavior change.

## Minor tasks (bundle into this session)

### 1. Confidence filter select on the documents view

Add a confidence filter (`LOW` / `MEDIUM` / `HIGH`) next to the existing status filter select on the
documents list. Mirror the status filter end-to-end:

- Web: `app/client/ui/modules/document-grid.ts` — add to `DEFAULTS`, `hydrateFromQuery`,
  `syncQuery`, a handler, and a `<select>` in the toolbar template (see the status filter as the
  pattern; see the `web-development` skill for conventions).
- API/repository: `internal/documents/` — add a confidence filter to `Filters` / `FiltersFromQuery`
  and the query projection, mirroring the status filter. Confirm the documents list query exposes
  the classification `confidence` for filtering.

### 2. Page Size select does not reflect the URL param on refresh

Symptom: loading `/app/?page_size=24` paginates correctly (24 per page) but the **Page Size select
still displays 12** after refresh. The data hydrates from the URL (session #135) but the select's
displayed value is not bound to the hydrated `pageSize`.

- Fix: bind the size `<select>` value to the hydrated state (`.value=${String(this.size)}`) in
  `app/client/ui/elements/pagination-controls.ts`, and ensure `app/client/ui/modules/document-grid.ts`
  hydrates `pageSize` from the query in `hydrateFromQuery` (apply the same to `prompt-list.ts` for
  parity). See sessions `134-page-size-select.md` and `135-url-query-state-persistence.md`.

## Validation

- Re-run the full `_project/marked-documents/` set: classification strings must remain correct
  (no regression from this session's accuracy work) **and** confidence must now follow the rule
  above — `single-secret-noforn-split.pdf` is the canonical LOW case.
- Confirm both minor UI tasks in the local app (`mise run dev`): confidence filter narrows the list;
  `?page_size=N` is reflected by the select after refresh.
- `mise run vet`, `mise run test`, `mise run web:build` clean.
