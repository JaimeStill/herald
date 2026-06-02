# PDF Init Render Optimizations — Contrast Normalization + Render Parameterization + Cost/Fidelity Instrumentation

## Context

Follow-up to the confidence overhaul. `single-secret-noforn-split.pdf` (legible top `SECRET`
banner + near-illegible faded `NOFORN` stamp) intermittently drops the faded marking. The
optimization notes (`.claude/context/pdf-init-optimizations.md`) framed this as a resolution
problem: they assumed Azure downsamples our 300 DPI render to ~768×990px (the GPT-4o tile
algorithm) and faint markings get averaged away below that threshold.

**Research correction (resolved this session):** Herald's deployment is `gpt-5.2`, which is in the
GPT-5 family and uses **32px patch-based tokenization**, *not* the GPT-4o 512px-tile / 768px-short-
side algorithm. For `detail: high`, patch-based models allow up to ~2,500 patches (or 2,048px max
dimension), resizing to fit. A portrait Letter page is therefore processed at **~1,380×1,785px**, not
~768×990px. Resolution is already adequate; the recall failure is a **contrast/perception** problem,
not a resolution problem. The `detail` value in `config.json` (`capabilities.vision.vision_options.detail`)
is confirmed to be the same `image_url.detail` parameter Azure accepts (traced through the tau
`format/openai` marshaler), but the behavior shown in the Microsoft "Configure image detail level"
docs screenshot describes the GPT-4o algorithm and does **not** apply to gpt-5.2.

Intended outcome: close the split.pdf recall gap via contrast normalization (the correctly-targeted
lever), keep the render pipeline clean so `Enhance` stays a deterministic function of source + settings,
and add token-usage instrumentation so cost and the gpt-5.2 patch cap are measured rather than assumed.

## Scope decisions

- **Opt #1 — adaptive contrast normalization: IMPLEMENT.** Primary recall lever for gpt-5.2.
- **Opt #2 — pre-size to Azure target: DROP.** Confirmed to yield **$0** Azure-token savings (gpt-5.2
  resizes to its patch budget regardless of what we send; image-token count is identical at 300 DPI or
  pre-sized). Only saves upload bandwidth + local CPU (negligible at scale), and dropping source DPI to
  capture CPU would sit at/below the model's effective resolution and hurt recall. Recorded as a possible
  future infra-efficiency task, not part of this release.
- **Opt #3 — parameterize `Render`: IMPLEMENT.** Keeps the always-on contrast-stretch out of `Enhance`.
- **Token-usage logging: ADD.** Permanent cost observability + empirically confirms gpt-5.2's high-detail
  patch cap (docs are ambiguous for gpt-5.2: 1,536 vs 2,500) and true effective resolution.
- **Detail-level exploration: VALIDATION ONLY (no code).** Compare `detail: high` vs `detail: original`
  via the `config.json` toggle, measuring recall vs cost. Confirm gpt-5.2 even accepts `original`.

## Source changes (code-only; docs/context handled post-execution)

### 1. Introduce `RenderOptions` and rework `Render` — `internal/format/imagemagick.go`

`Render` currently takes `(ctx, src, dst, density bool, settings *state.EnhanceSettings)`. Adding an
always-on contrast operator must apply to the **Extract/classify** render only, not to `Enhance`
(which applies the model's targeted brightness/contrast and must stay clean). Per the project
convention (>2 params → struct), replace the positional `density`/`settings` with an options struct:

```go
type RenderOptions struct {
    Density   bool                    // -density 300 (PDF rasterization)
    Normalize bool                    // always-on contrast operator for the extract pass
    Settings  *state.EnhanceSettings  // targeted enhance filters (brightness/contrast/saturation)
}
```

Operator order in the `magick` arg list: `-density` → `src` → (Normalize ? contrast operator) →
(Settings ? `-brightness-contrast` / `-modulate`) → `dst`. In practice Normalize and Settings are
never both set (Extract normalizes; Enhance applies settings), but the order is fixed for safety.

**Contrast operator — A/B in validation, do not hardcode blindly.** Candidates, in order of preference
to test:
1. `-contrast-stretch 0.5%x0.5%` — self-limiting global stretch (notes' suggestion).
2. `-sigmoidal-contrast 5x50%` — non-linear mid-tone boost; likely better at lifting a small faded
   mid-gray marking on a page that already has full-range black/white content (see Risk below).
3. `-normalize` / `-auto-level` — fallback global operators.

Keep `brightnessContrastArg` as-is (used by `Settings`).

### 2. Update the four call sites

- `internal/format/pdf.go` `Extract` (line 72): `RenderOptions{Density: true, Normalize: true}`.
- `internal/format/pdf.go` `Enhance` (line 96): `RenderOptions{Density: true, Settings: settings}` —
  **no Normalize**, preserving the "renders fresh from source.pdf with only targeted settings" property.
- `internal/format/image.go` `Extract` (line 58, jpeg/webp normalization): `RenderOptions{Density: false}`
  — **no Normalize.** Per the notes' scope, do not apply the always-on stretch to native image uploads.
- `internal/format/image.go` `Enhance` (line 78): `RenderOptions{Density: false, Settings: settings}`.

### 3. Token-usage logging — `internal/workflow/`

The tau response layer already parses `prompt_tokens`/`completion_tokens`/`total_tokens` into
`resp.Usage` (`*response.TokenUsage`, fields `InputTokens`/`OutputTokens`/`TotalTokens`); Herald simply
never reads it. Add `resp.Usage` fields to the existing `Logger.DebugContext` calls:
- `classify.go` (the `"classify page complete"` log, ~line 110): add `input_tokens`, `output_tokens`.
- `enhance.go` (~line 125–154): same.
- `finalize.go` (~line 62–75): same (this is the `Chat` call; no image tokens).

Guard for `resp.Usage == nil`. This is the empirical instrument for both the patch-cap confirmation and
the cost model.

## Open implementation choices (settle during execution via validation, not in advance)

- Final contrast operator + parameters (candidates above).
- Whether `detail: original` is worth its ~+36%/page cost (~+$39k corpus at 5 pages avg) — decided by the
  recall-vs-cost comparison, not assumed.

## Validation

No automated sweep harness exists; validation is manual via the API + the new token logs.

1. **Run locally:** `docker compose up -d`, `mise run dev`. Watch server logs for the new
   `input_tokens`/`output_tokens` lines.
2. **Confirm the patch cap empirically:** classify one page, then re-classify the same page rendered at a
   much smaller source size; compare `input_tokens`. Equal counts ⇒ we're at the cap ⇒ derive gpt-5.2's
   true high-detail patch budget (1,536 vs 2,500) and effective resolution. Subtract the text-prompt
   tokens (measured from a text-only call) to isolate image tokens.
3. **Contrast A/B + recall:** for each candidate operator, classify `single-secret-noforn-split.pdf`
   **×5**. Success = the faded `NOFORN` is now *perceived* — read into `markings_found`, or at minimum
   surfaced as an `undetermined_marking` driving `SECRET (LOW, flag for review)` rather than silently
   dropped. Pick the operator with the most consistent perception across runs.
4. **Detail-level comparison:** repeat the split.pdf sweep with `config.json`
   `vision_options.detail: "high"` vs `"original"` (confirm `original` is accepted by gpt-5.2). Record
   recall delta and the `input_tokens` (cost) delta from the logs.
5. **Regression guard (clean docs):** re-classify `single-secret`, `single-confidential`,
   `single-unclassified`, `uniform-*` — classification strings and confidence must not degrade; the
   contrast operator must not fabricate or distort crisp banners.
6. **Full set:** run the complete `_project/marked-documents/` set once more to confirm no
   classification-string or confidence regressions before release.

## Risks

- **Global histogram operators may not move a small faint marking.** A page with crisp black text + white
  background already spans the full tonal range, so `-contrast-stretch`/`-normalize` may do nearly nothing
  to a faded mid-gray stamp. This is why `-sigmoidal-contrast` (mid-tone-targeted) is a first-class
  candidate and why #1 is validated empirically rather than assumed to work.
- **gpt-5.2 patch cap uncertainty** (1,536 vs 2,500) — resolved by step 2; does not block #1/#3.
- **`detail: original` support** on gpt-5.2 is unconfirmed — verified in step 4 before relying on it.

## Files

- `internal/format/imagemagick.go` — `RenderOptions`, reworked `Render`.
- `internal/format/pdf.go` — `Extract`/`Enhance` call sites.
- `internal/format/image.go` — `Extract`/`Enhance` call sites.
- `internal/workflow/classify.go`, `enhance.go`, `finalize.go` — usage logging.
- `config.json` — `vision_options.detail` toggle (validation only).
