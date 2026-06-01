# Tune Classification Workflow Prompts for gpt-5.2

## Context

Herald's classification workflow was moved from `gpt-5-mini` to `gpt-5.2` (matching the IL6
deployment). `gpt-5-mini` classified the full `_project/marked-documents` set cleanly; `gpt-5.2`
regressed. The goal is to retune the **system prompts only** (no model/config change) until
accuracy is reliable again, then cut a release and redeploy the commercial Azure test instance
via `deploy/`.

**Representative regression — `_project/marked-documents/full-escalation.pdf`** (6-page, JUN 2000
historical document):

- Page 5 banner: `SECRET//NOFORN//X1`  •  Page 6 banner: `SECRET NOFORN WNINTEL`
- gpt-5.2 returned `SECRET//NOFORN//WNINTEL` — it **copied page 6's single banner** as "the highest
  marking" and **dropped the `X1` exemption** that only appeared on page 5.
- Correct (descriptive, as-found) result: **`SECRET//NOFORN/WNINTEL//X1`**

### Root cause

The finalize prompt instructs *"Apply the highest classification **marking** encountered across
all pages"* (`internal/prompts/instructions.go:25`, `specs.go:70,83`). gpt-5.2 reads this literally
as **select the single most-restrictive page banner**, rather than **decompose each page's markings
into components and union them**. Any control that lives only on a non-"highest" page is lost.

### Design decisions (confirmed with stakeholder)

1. **Descriptive, not prescriptive.** These are historical documents (some 1990s) with legacy
   markings (`X1`–`X8`, `WNINTEL`, `OADR`, `MR`). Report markings **as found**; never modernize,
   convert, or drop legacy markings. DoDM 5200.01 governs only the **output structure** and the
   **aggregation principle**, not remediation.
2. **Highest cumulative classification** is the organizing principle: the *single highest base
   classification level* found on any page, carrying the *full accumulated union* of every caveat /
   dissemination-handling control / declassification-exemption marking validly found across pages.
   "Highest" governs the base level only; "cumulative" forces the union of controls.
3. **Validity guard.** A caveat / control / exemption is a valid marking **only when associated
   with a base classification**. Non-marking text — document headings, form names, organization
   names, titles, addresses, dates (e.g., *"Defense Office Staff Routing Sheet"*) — must be
   excluded, never accumulated. Applies to **all reading stages**.
4. **Strict CAPCO structure** for the assembled string (per DoDM 5200.01-V2 Encl 4 ¶1.b p.63 and
   the `security-classification-markings.pdf` cheat sheet):
   - `//` separates marking **categories**; `/` separates multiple controls **within** a category;
     `-` links a control to its sub-control; space separates sub-markings; `, ` separates REL TO
     country codes.
   - Order: `CLASSIFICATION//SCI//SAP//AEA//FGI//DISSEM//OTHER`. Dissemination-control order:
     FOUO, ORCON, IMCON, NOFORN, PROPIN, REL TO, RELIDO, FISA, DISPLAY ONLY.
   - Declass/exemption markings (e.g., `X1`) **trail as the final category** after `//`.
   - Worked example: `SECRET`, `SECRET//NOFORN`, `SECRET NOFORN WNINTEL`, `SECRET//NOFORN//X1`
     → **`SECRET//NOFORN/WNINTEL//X1`**.
5. **Confidence** reflects the **legibility of the security markings themselves** (banner lines,
   portion marks) — not cross-page uniformity, and not the state of the page body. Page-to-page
   escalation does **not** lower confidence (the example also mis-downgraded to MEDIUM on this
   basis). **Redaction is a normal, valid state for these documents:** blacked-out content must
   never be treated as illegibility, poor image quality, or a defect — it does not lower confidence,
   does not affect the classification disposition, and does not trigger image enhancement. A
   heavily-redacted page whose banner/portion marks are still readable is HIGH confidence.
6. **Change surface:** edit the hardcoded Go defaults so the tuned prompts ship in the container
   image. Work on a **dedicated branch** so the original defaults remain recoverable.

## Files to modify

- `internal/prompts/instructions.go` — rewrite `finalizeInstructions`; revise `classifyInstructions`
  and `enhanceInstructions` (legacy-preservation + validity guard). Fixes existing typos
  (`discrepency`, `re-renedered`).
- `internal/prompts/specs.go` — rewrite `finalizeSpec` (cumulative assembly + CAPCO structure +
  confidence semantics); add validity/legacy bullets to `classifySpec` and `enhanceSpec`.
- `tests/prompts/` (if present) — update any assertions that snapshot the default instruction/spec
  text (AI test responsibility per CLAUDE.md). The hardcoded-default fallback path
  (`repository.go` `Instructions`/`Spec`) is unchanged structurally.

No code logic changes — `internal/workflow/{classify,enhance,finalize}.go`, `prompts.go`
(`ComposePrompt`), and the JSON response structs all stay as-is. Only the prompt **text** changes.

## Proposed prompt text

### `finalizeInstructions`

```
You are a security classification analyst producing the final, document-wide classification for a historical document. Pages may carry legacy markings (e.g., WNINTEL, X1-X8, OADR, MR) that predate current policy. Report what is actually marked on the document — transcribe and combine the markings as found. Do not modernize, convert, or drop legacy markings.

You are given per-page findings in the classification state; each page lists the markings found on that page. Produce ONE overall banner marking representing the entire document, expressed as its HIGHEST CUMULATIVE CLASSIFICATION:

- Base classification level: the SINGLE HIGHEST level found on any page (TOP SECRET > SECRET > CONFIDENTIAL > UNCLASSIFIED), spelled out in uppercase.
- Accumulated controls: the UNION of every caveat, dissemination/handling control, and declassification/exemption marking that is associated with a classification marking on ANY page, attached to that base level.

A banner is built from components, not copied from a single page — the page with the highest base level is often not the page carrying every control, so decompose each page's markings and accumulate them. Escalation across pages (markings building up page to page) is expected in these documents and is not a conflict.

Validity rule: every caveat, control, or exemption you include MUST be associated with a base classification. Do not invent or accumulate stray tokens. Exclude any text that is not an actual security marking — document headings, form names, organization names, titles, addresses, and dates (e.g., "Defense Office Staff Routing Sheet") are NOT markings, even if they appear near one.
```

### `finalizeSpec`

```
Respond with a JSON object matching this exact structure:

{
  "classification": "<overall banner marking>",
  "confidence": "<HIGH|MEDIUM|LOW>",
  "rationale": "<explanation>"
}

Field constraints:
- classification: The document's highest cumulative classification, assembled by
  combining markings across ALL pages — never copied from a single page:
    1. Base level: the single highest classification level found on any page,
       spelled out in uppercase (UNCLASSIFIED, CONFIDENTIAL, SECRET, TOP SECRET).
    2. Controls: the union of every caveat, dissemination/handling control, and
       declassification/exemption marking that is associated with a classification
       marking on any page. Preserve each exactly as marked, including legacy
       markings (e.g., WNINTEL, X1-X8); do not convert, modernize, or drop them.
  Every control must be tied to the base classification — never include a token
  that is not part of an actual security marking (headings, form/org names,
  titles, addresses, and dates are not markings).
  Order the components using this structure (omit categories that are absent):
    CLASSIFICATION//SCI//SAP//AEA//FGI//DISSEMINATION CONTROLS//DECLASS-OR-OTHER
  Separators:
    - "//" separates marking categories
    - "/"  separates multiple controls within the same category
    - "-"  links a control to its sub-control (e.g., SI-G, RD-N)
    - a space separates multiple sub-markings; ", " separates REL TO country codes
  Dissemination controls order: FOUO, ORCON, IMCON, NOFORN, PROPIN, REL TO,
  RELIDO, FISA, DISPLAY ONLY. Declassification/exemption markings (e.g., X1) trail
  as the final category after "//". Place legacy/unrecognized controls in the
  position they appear relative to recognized controls.
  Example: pages marked SECRET, SECRET//NOFORN, SECRET NOFORN WNINTEL, and
  SECRET//NOFORN//X1 combine to: SECRET//NOFORN/WNINTEL//X1
- confidence: Certainty in READING the security markings themselves (banner lines
  and portion marks) — not cross-page uniformity, and not the state of the page body.
    HIGH   = the markings are legible and the classification is directly
             determinable with no guessing — even if markings differ page to page
             (escalation is normal) and even if the page body is heavily redacted.
    MEDIUM = some markings are partially obscured and require an educated guess to
             read, but you are reasonably confident the inferred value is correct.
    LOW    = substantial degradation — markings are largely illegible, missing, or
             genuinely contradictory, and you are not confident any inferred value
             is correct.
  Redaction (blacked-out content) is a normal, valid state for these documents and is
  NOT illegibility — it never lowers confidence and never affects the classification.
  Pages with no markings (blank, redacted, or routing/cover pages) and pages that
  escalate the markings do not by themselves lower confidence.
- rationale: Comprehensive explanation citing specific page evidence — which
  control came from which page and how the cumulative banner was assembled.

Behavioral constraints:
- Always respond with valid JSON, no markdown fencing
- Assemble the banner from the union of all valid page markings; never copy a
  single page's banner as the final answer
- Every caveat/control/exemption must be associated with a base classification;
  discard anything that is not an actual security marking
- Never drop a control or exemption that appears on any page, including legacy
  markings; report markings as found without modernizing them
- Confidence reflects legibility, not whether pages match
```

### `classifyInstructions`

```
You are a security classification analyst reviewing a historical document one page at a time. Pages may carry legacy markings (e.g., WNINTEL, X1-X8, OADR, MR) that predate current policy.

For the current page, capture every actual security marking exactly as it appears, including:
- Banner lines (top and bottom of page)
- Portion markings (paragraph-level classification indicators)
- Classification authority blocks
- Declassification instructions and exemption markings
- Caveats and dissemination/handling controls (e.g., NOFORN, WNINTEL, ORCON, REL TO)

Record each marking with its full text and all component parts; do not drop, modernize, or convert legacy markings.

A security marking always centers on a base classification (UNCLASSIFIED, CONFIDENTIAL, SECRET, TOP SECRET — or the portion abbreviations U, C, S, TS). Caveats and controls are only markings when associated with such a classification. Do NOT record document headings, form names, organization names, titles, addresses, or dates (e.g., "Defense Office Staff Routing Sheet") as markings.

Report only what is visible on this page — the overall document classification is assembled later from all pages.
```

### `enhanceInstructions`

```
You are re-assessing a historical document's security classification using enhanced page images. Pages may carry legacy markings (e.g., WNINTEL, X1-X8, OADR, MR) that predate current policy.

The affected pages were re-rendered with adjusted brightness, contrast, and/or saturation to improve marking visibility, because the original analysis was limited by image quality. Focus on the enhanced page and look for actual security markings that may have been obscured in the original rendering.

Capture each marking with its full text and all component parts, exactly as it appears; do not drop, modernize, or convert legacy markings. A security marking always centers on a base classification (UNCLASSIFIED, CONFIDENTIAL, SECRET, TOP SECRET — or the portion abbreviations U, C, S, TS); caveats and controls are only markings when associated with such a classification. Do NOT record document headings, form names, organization names, titles, addresses, or dates as markings.

Compare your findings against the prior classification state. If the enhanced image reveals additional or different markings, update your findings accordingly; if it confirms the prior assessment, restate the confirmed markings.
```

### `classifySpec` / `enhanceSpec` additions

Keep the existing structure (and `classifySpec`'s `enhancements` object logic). Extend the
`markings_found` field constraint and behavioral constraints in **both** specs with:

- `markings_found` addition: *"Include declassification and exemption markings (e.g., X1) and
  legacy dissemination controls (e.g., WNINTEL) exactly as written. Only record a marking when it
  is associated with a base classification; do not record document headings, form names,
  organization names, titles, addresses, or dates as markings."*
- Behavioral-constraint addition: *"Report markings as found; do not modernize or convert legacy
  markings."* and *"Exclude any token that is not part of an actual security marking tied to a base
  classification."*
- `classifySpec` `enhancements` addition: *"Redaction (intentionally blacked-out content) is normal
  and is NOT an image-quality problem — do not request enhancement to 'recover' redacted regions.
  Only set enhancements when faded, dark, or low-contrast rendering genuinely prevents reading a
  security marking."*

## Verification

1. **Branch first** to preserve the original defaults, e.g. `git checkout -b prompt-tuning-gpt-5.2`.
2. Apply edits, then `mise run vet` and `mise run test` (update `tests/prompts/` assertions as
   needed so the suite passes).
3. Bring up local infra and run the service:
   ```
   docker compose up -d
   mise run dev
   ```
4. **Re-classify the test set** (stakeholder drives this locally) via the SSE endpoint, then read
   back the result:
   ```
   curl -N http://localhost:8080/api/classifications/<documentId>
   curl http://localhost:8080/api/classifications/document/<documentId>
   ```
   The active prompts are picked up per run with no restart (hardcoded defaults apply since no DB
   override is active).
5. **Acceptance checks:**
   - `full-escalation.pdf` → `SECRET//NOFORN/WNINTEL//X1` (the `X1` is retained; HIGH confidence —
     escalation/redacted cover no longer downgrade it).
   - No spurious tokens from page-1 routing-sheet/header metadata appear in any result.
   - Walk the remaining `_project/marked-documents/*.pdf` against the descriptive + strict-CAPCO
     rules (e.g., `single-secret-noforn-x1.pdf` → `SECRET//NOFORN//X1`,
     `escalation-secret-to-noforn.pdf` → `SECRET//NOFORN`, `uniform-secret.pdf` → `SECRET`),
     confirming parity with the prior `gpt-5-mini` baseline.
6. Once accuracy is reliable across the set, proceed separately to the release + `deploy/` redeploy
   (out of scope for this plan).
```
