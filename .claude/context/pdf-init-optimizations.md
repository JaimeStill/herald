# PDF Init Render Optimizations

Follow-up work captured during the confidence-overhaul validation sweep. These optimize the
**input fidelity** of the PDF→PNG rasterization so the vision model can actually perceive faint
markings — the root cause behind `single-secret-noforn-split.pdf` returning `SECRET`/HIGH with the
faded NOFORN silently dropped.

## Why (the finding)

We render pages at **300 DPI** (`magick -density 300`, `internal/format/imagemagick.go`) and send the
PNG to Azure with `detail: high`. But the OpenAI/Azure vision pipeline **downsamples high-detail
images** before the model sees them — scale to fit 2048×2048, then scale so the **shortest side is
768px**. A portrait Letter page is therefore processed at roughly **768×990px** regardless of our
DPI. So:

- Our 300 DPI is already **above** what the API uses — raising DPI does nothing (the extra pixels are
  discarded).
- At ~768px page width, a **faded, low-contrast** marking falls below the model's perceptual
  threshold. That's why split.pdf "glosses over it as if it doesn't exist."

This is an **input-fidelity** problem, not a prompt-logic problem — no prompt wording recovers pixels
that were averaged away in the downsample. The recall fix lives in the render pipeline.

## Scope

- **Conversion formats only** (`internal/format/pdf.go`). Native image uploads (`image.go`) already
  carry their own resolution — operate on the native image directly and only apply *enhance*
  adjustments when flagged. Do **not** apply the always-on stretch/resize to native images.
- **No tiling or cropping** — explicitly out of scope for the available time (see item 4).
- Every change must be validated against the **full** scenario set, with the **clean** docs
  (`single-secret`, `uniform-*`, `single-confidential`, `single-unclassified`) as the regression
  guard: normalization/resize must not degrade crisp banners or fabricate markings.

## Optimizations, ordered by simplicity

### 1. Adaptive contrast normalization (simplest, do first)

Add an ImageMagick histogram operator (`-contrast-stretch 0.5%x0.5%`, `-normalize`, or `-auto-level`)
to the pass-1 render in `Render` (`imagemagick.go`). These are **self-limiting**: a clean,
high-contrast page is barely affected; a faint, low-contrast page gets its faded gray pulled toward
black so it survives the 768px downsample. No analysis/decision step needed — the operator adapts
per image. This is the highest value-to-effort change and may close split.pdf on its own.

- Validate: re-run `single-secret-noforn-split` several times — does the faint NOFORN now get
  *perceived* (read, or at least flagged `undetermined` → LOW)? Re-run the clean docs for regressions.

### 2. Pre-size to the Azure target (own the downscale)

In the same `magick` pipeline, **resize** the render to the API's effective dimensions (~768px short
side, ≤2048 long) with a quality filter (`-filter Lanczos`), applied **after** the contrast-stretch
on the still-detailed raster. Benefits: we control the resample quality instead of Azure's black-box
downscaler, Azure does little/no further downsampling, and the transmitted payload + our compute
shrink. Keep a healthy source DPI for the contrast step (test 150 vs 300 — 150 is ~4× cheaper and a
1275→768 downscale preserves plenty); do **not** rasterize directly at the target (that loses faint
detail before contrast can act). Operator order: **stretch → (enhance filters) → resize**.

- Prereq: **verify the current Azure `gpt-5.2` `detail:high` dimensions** before hardcoding `768` —
  the deployment may differ from the documented algorithm.

### 3. Parameterize `Render` so enhance stays clean

Don't bake the always-on stretch/resize into every `Render` call. `Extract` opts into
stretch+resize; `Enhance` renders fresh from the **pristine source PDF** and applies **only its
targeted brightness/contrast settings** (no always-on stretch), so enhance output stays a
deterministic function of `(source PDF + settings)` rather than compounding the always-on
processing. (PDF `Enhance` already renders from `source.pdf`, not the classify-stage PNG — this just
keeps it that way after items 1–2 land.)

### 4. Banner-region crops / page tiling (deferred — out of scope)

Highest resolution leverage: render and send the top/bottom banner *strips* as separate images. A
thin bottom strip isn't downsampled (its short side is below 768), so it reaches the model at
~2048px wide — ~2.7× the effective resolution on the marking zone. Deferred for time; revisit only
if contrast + pre-size proves insufficient. Costs extra image tokens per page.

## Validation plan

1. Verify Azure `gpt-5.2` `detail:high` scaling dims.
2. Land item 1 (contrast), re-run `single-secret-noforn-split` ×3–5 + clean docs.
3. If item 1 is insufficient, land item 2 (pre-size), re-validate.
4. Land item 3 (enhance parameterization), confirm enhance composition.
5. Full-set re-run to confirm no classification-string or confidence regressions.

## Related (separate prompt follow-ups — not PDF-init)

Tracked alongside this work but in the prompt layer, not the render pipeline:

- **X1/XI resolution** — `single-secret-noforn-xi` returns `XI` ~half the runs; normalize ambiguous
  exemption glyphs (`XI`/`Xl` → `X1`), analogous to `NOFOPI → NOFORN`.
- **Routing/cover-page fabrication** — `uniform-unclassified` occasionally fabricates a base level
  (`CONF.`) on a heavily-redacted routing sheet; reinforce read-don't-infer for unmarked redacted
  pages. The contrast/clean-render work above may *also* reduce this by giving the model less
  redaction-edge noise to hallucinate from.
