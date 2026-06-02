# Herald Prompt Infrastructure Review

A holistic, policy-grounded review of Herald's classification prompts, conducted after the
render-fidelity work (contrast normalization + token instrumentation) revealed that the one
remaining unreliable scenario — `single-secret-noforn-split.pdf` (faded NOFORN) — is a
**prompt/perception** problem, not a render problem. Goal: prompts that are clear, reliable,
policy-aligned, and **universal** (correct for any document), not tuned to our scenario set.

Policy basis: **DoDM 5200.01, Volume 2 — "Marking of Information"** (Feb 24 2012, Change 4, eff.
Jul 28 2020), staged at `/home/jaime/code/dodm-5200.01.txt`. Citations below are to Enclosure
(Encl) → paragraph. Quick-reference cross-check: USSOCOM "Security Classification Markings"
(21 MAR 2024), `~/Documents/security-classification-markings.pdf`.

---

## 1. Infrastructure map

Prompts are composed per workflow stage by `workflow.ComposePrompt` as:

```
<instructions>  +  \n\n  +  <spec>  +  (optional) "\n\nCurrent classification state:\n\n" + <state JSON>
```

Three stages run in sequence: **classify** (per page, vision) → **enhance** (conditional, per
flagged page, vision) → **finalize** (once, chat — no image).

| Component | Source | Mutability |
|---|---|---|
| **instructions** (`internal/prompts/instructions.go`) | DB `active` row per stage, else hardcoded default | **Tunable** — a Prompts-UI override shadows the default |
| **spec** (`internal/prompts/specs.go`) | always hardcoded | **Immutable** — never DB-backed (`repo.Spec` returns the constant) |
| **state JSON** | accumulated `ClassificationState` | runtime |

**Design implication:** policy-critical, non-negotiable rules (banner assembly order, "every
control must be tied to a classification," transcribe-don't-derive, control-combination rules)
belong in the **immutable spec**, so a user instruction override cannot silently break them.
Softer guidance (tone, what to look for, how to reason about degradation) belongs in instructions.

---

## 2. Operating constraints (the foundation)

### 2.1 Transcribe-as-found (confirmed)

Herald is a **faithful reader** of the markings physically present on (often historical)
documents. It does **not** re-derive, modernize, or remediate. Concretely: legacy markings
(`X1`–`X8`, `OADR`, `MR`, `WNINTEL`) are transcribed **verbatim**; Herald never applies the
derivative-classifier rule "add 25 years to the source date" (Encl 3, para 9.a) — that is a
re-marking action, not a reading action. A correctly-marked legacy document literally shows
`Declassify On: X4` (Encl 3, Figure 8), and that is what Herald reports.

### 2.2 Vision processing: tile-768 (empirically confirmed)

The Azure `gpt-5.2` deployment processes each image with the **GPT-4o tile algorithm**, *not* the
32px-patch algorithm the public gpt-5 docs describe. Measured from token logs: a single-page
classify call reported `input_tokens ≈ 3,100`; calibrating text tokens against the image-free
`finalize` call (~4.2 chars/token) leaves **~765 image tokens** — exactly the tile cost for a
portrait page (`shortest side → 768px ⇒ 4× 512px tiles ⇒ 4×170 + 85 = 765`). Setting
`detail: "original"` left `input_tokens` **unchanged at 3,100** → the deployment ignores it; there
is **no resolution headroom via the `detail` knob.**

Implications for the prompts:
- The model sees the page at **~768px on its short side**. Faint, small markings (bottom-banner
  NOFORN) sit near the perceptual floor. We cannot add resolution, so the prompt must bias toward
  **flagging** a faint banner position as `undetermined` rather than concluding from the legible
  markings alone.
- The **enhance** pass is the recovery mechanism. It now renders with the same contrast baseline
  as classify plus the model's targeted settings (see render work), but it is still bound by 768px.
- The only lever that raises effective resolution on the marking zone is **banner-region crops**
  (§6, deferred) — relevant only if prompt + contrast prove insufficient.

### 2.3 Confidence integration

Per-page `confidence` is emitted by the vision stages (classify/enhance — the only stages that see
pixels). Document confidence is derived **deterministically** in code
(`state.AggregateConfidence`): the lowest per-page confidence among content-bearing pages; a page
with `undetermined_markings` is floored to LOW; a content-bearing page reporting no confidence
defaults to LOW; pages with no markings (blank/redacted/cover) are ignored; a doc with no
content-bearing pages is a confident UNCLASSIFIED (HIGH).

**Key finding:** the residual `split.pdf` failure is **upstream of aggregation**. In the unsafe
run the page returned `HIGH / [SECRET] / undetermined=[]` — the model neither read nor flagged the
faded NOFORN, so the deterministic floor had nothing to act on. **No aggregation rule can catch a
marking the model never surfaced.** The fix must live in the classify prompt: make the model
systematically inspect every banner position and flag a faint one.

---

## 3. Policy reference (cited)

### 3.1 Banner line — order & separators

Authoritative one-line template (Encl 4, para 1.b; order per Figure 25, Encl 4 para 1.a):

```
CLASSIFICATION//SCI//SAP//AEA//FGI//DISSEM//OTHER DISSEM
```

- `//` separates **categories**; `/` separates **multiple entries within a category**; `-` joins a
  control to its **sub-control/compartment** (`SI-G`, `RD-N`, `ACCM-NICKNAME`); space separates
  multiple sub-markings; `, ` separates REL TO country codes (Encl 4, para 1.b.(1)).
- Classification level is **spelled out, UPPERCASE, never abbreviated, exactly one level** — the
  **highest** of any portion (Encl 3, para 5.a). A leading `//` (nothing before it) = FGI-only or
  JOINT-only document (Encl 4, para 3.b).
- **Declassification / exemption markings do NOT appear in the banner.** They live in the
  classification **authority block** "Declassify On:" line (Encl 3, paras 8.a.(3), 8.b.(1)(d)).
- Banners appear at **both top and bottom** of the cover, title, first, and each interior page,
  usually centered (Encl 3, paras 5.b–5.c). ← the systematic-inspection hook.

### 3.2 Dissemination controls — list, order, combination rules

Category 8 order (Figure 25): FOUO → ORCON → IMCON → NOFORN → PROPIN → REL TO → RELIDO →
DISPLAY ONLY → FISA. Category 9 ("Other Dissem", trailing): SPECAT, NC2-ESI, ACCM, EXDIS, NODIS
(Encl 4, §11). Meanings per App 1 / Encl 4 §§10–11.

**Combination rules a naive union gets wrong** (the important part):
1. **NOFORN and REL TO/RELIDO are mutually exclusive in the banner; NOFORN wins.** A doc with both
   NOFORN and REL TO portions → banner shows `//NOFORN`, not both (Encl 4 REL TO para 7; App 1).
2. **REL TO is suppressed unless the *entire* document is releasable** to the listed countries; if
   any portion is uncaveated or countries don't intersect, DoD banner shows only the U.S.
   classification (IC rule instead elevates to NOFORN) (Encl 4 REL TO paras 8, 8.a–8.b).
3. **FOUO/CUI is not hoisted into a classified banner** — `(U//FOUO)` portions do not propagate
   FOUO to the overall banner; FOUO appears in a banner only on an *unclassified* page
   (`UNCLASSIFIED//FOUO`) (Encl 4, para 10.b.(3)).
4. **DISPLAY ONLY** may not combine with RELIDO or NOFORN (Encl 4, Display Only para 4).
5. **Level constraints:** IMCON is SECRET-only (elevates in a TS portion); HCS / TK-GEOCAP require
   NOFORN to co-occur (Encl 4, para 6.f).

### 3.3 SCI / SAP / AEA (RD/FRD) / FGI

- **SCI** (cat 4): `[CLASS]//SI/TK//...`, compartments via `-`, alphabetical (Encl 4 §6).
- **SAP** (cat 5): `[CLASS]//SAR-NICKNAME`, `SAR-MULTIPLE PROGRAMS`, `WAIVED` as a dissem control
  (Encl 4 §7).
- **AEA — RD/FRD** (cat 6): NOT dissemination controls; banner `[CLASS]//RESTRICTED DATA` (or `RD`)
  / `//FORMERLY RESTRICTED DATA`; **must appear in the banner if any portion contains it**;
  **suppresses the "Declassify On" date** (no automatic declass); sub-controls CNWDI (`RD-N`) and
  SIGMA (`RD-SIGMA #`) (Encl 4 §8; Encl 3 para 8.a.(3)).
- **FGI** (cat 7): `[CLASS]//FGI <codes alphabetical>`; overall level = max(US, US-equivalent of
  FGI); NOFORN may not apply to FGI-only portions; REL TO suppressed unless whole doc releasable
  (Encl 4 §§4, 9).

### 3.4 Declassification & authority block

- "Declassify On:" carries a date (`YYYYMMDD`), event, exemption (`25X1`–`25X9` with date/event;
  `50X1-HUM`, `50X2-WMD` with none), or legacy value (`OADR`, `MR`, `X1`–`X8`, `DCI Only`)
  (Encl 3 §8–9). **Authority-block content, not banner content.**
- Authority block (face of doc): original = `Classified By / Reason / Declassify On`; derivative =
  `Classified By / Derived From / Declassify On` (Encl 3 §8). FGI-only / NATO docs carry no
  authority block.

### 3.5 What is / isn't a marking; placement

- A marking must be **conspicuous and distinct from informational text** (Encl 3 paras 1, 2.b).
- A **control is meaningful only when bound to a classification** (Encl 3 para 5.f). A bare control
  word in body text, and titles/subjects/form-names/addresses/dates, are **not** markings (their
  *portion marks* are, but the text is not) (Encl 3 paras 6.e.(2), 14).

---

## 4. Gap analysis — current prompts vs policy

| # | Gap | Current behavior | Policy | Disposition |
|---|---|---|---|---|
| **G1** | **Declass exemptions in the banner** | `finalizeSpec` trails `X1`/`25X1`/`50X1-HUM` "as the final category after `//`"; `classifySpec` records them in `markings_found` | Exemptions belong in the **authority block**, never the banner (Encl 3 §8) | **Open decision (D1)** — strict banner vs. as-marked summary |
| **G2** | **Naive union of controls** | finalize takes "the UNION of every caveat/control" | NOFORN⊻REL TO (NOFORN wins); REL TO suppressed unless wholly releasable; FOUO not hoisted; DISPLAY ONLY exclusions (Encl 4) | **Encode combination rules in spec** (D2) |
| **G3** | **Final banner category mislabeled** | "DISSEMINATION CONTROLS//DECLASS-OR-OTHER" | Real trailing category is **"OTHER DISSEM"** = SPECAT/ACCM/NC2-ESI/EXDIS/NODIS; declass is not a banner category | Rename/realign category; couple with D1 |
| **G4** | **AEA (RD/FRD) not represented** | No mention of RD/FRD/CNWDI/SIGMA; no rule that RD suppresses declass | RD/FRD are a distinct banner category and suppress the declass date (Encl 4 §8) | Add to spec (universal correctness) |
| **G5** | **Synthesize vs transcribe the overall banner** | "A banner is built from components, not copied from a single page" (pure union) | Correctly-marked pages already carry top+bottom banners; union risks emitting **policy-invalid** combinations | **Open decision (D3)** — keep cumulative assembly but make it policy-aware (recommended) |
| **G6** | **FGI / SCI / SAP** | Banner order lists the categories but gives no construction rules | Detailed per category (Encl 4 §§4–8) | Add concise rules to spec; keep proportional (most Herald docs are collateral) |
| **G7** | **Banner-position inspection** | classify says "Banner lines (top and bottom of page)" but does not enforce checking the bottom when the top is already legible | Banners appear top **and** bottom of every page (Encl 3 §5) | **Strengthen classify instructions** (the split fix, §5) |

What the current prompts already get **right** (keep): closed-vocabulary resolution of degraded
tokens (`NOFOPI → NOFORN`) without inventing controls; "record only markings tied to a base
classification"; exclusion of headings/forms/dates; read-don't-infer for unmarked/redacted pages;
"redaction never lowers confidence"; legacy markings preserved verbatim; separators and category
order skeleton.

---

## 5. Residual failure: the faded-NOFORN silent-drop — universal remedy

**Observed (4 runs, all tile-768/detail-high):** SECRET//NOFORN HIGH (read) ×1, SECRET//NOFORN
MEDIUM (read) ×1, SECRET LOW (flagged undetermined → review) ×1, **SECRET HIGH (silently dropped,
no flag) ×1**. The classification string is now stable when the marking is perceived; the unsafe
case is the model not attending to the bottom banner at all.

**Root cause:** at 768px the bottom-banner NOFORN is faint; when the top banner reads cleanly the
model sometimes concludes without scrutinizing the bottom. Contrast normalization lifts it over the
threshold ~2/3 of the time but cannot guarantee perception.

**Universal remedy (policy-grounded, not scenario-specific):** because policy requires a banner at
**both top and bottom of every page** (Encl 3 §5), instruct the classify stage to **inspect every
required banner position independently** — top banner, bottom banner, and portion marks — and to
treat **any** faint/partial content in a banner position as an `undetermined_marking` (which forces
enhancement and floors confidence to LOW), rather than concluding from the clearest banner alone.
Framed as "verify each banner position," this is correct for *every* document and directly targets
the silent-drop without naming NOFORN or "the bottom of split.pdf." This belongs primarily in the
tunable **classify instructions**, with the "faint banner position ⇒ undetermined + enhance" rule
reinforced in the immutable **classify spec**.

---

## 6. Carried-forward items (from the now-removed `pdf-init-optimizations.md`)

Preserved here so they are not lost — these were never completed:

- **Render opt #4 — banner-region crops / page tiling (DEFERRED).** The only lever that raises
  effective resolution on the marking zone past the tile-768 ceiling: render the top/bottom banner
  *strips* as separate images so a thin strip's short side stays below 768 and it reaches the model
  at ~2048px wide (~2.7× resolution on the marking). Costs extra image tokens per page. **Revisit
  only if §5's prompt remedy + contrast prove insufficient** in validation.
- **Prompt follow-up — `XI`/`Xl` → `X1` normalization.** `single-secret-noforn-xi` intermittently
  returns `XI`; normalize the ambiguous exemption glyph to `X1`, analogous to `NOFOPI → NOFORN`.
  Fits the existing closed-vocabulary resolution mechanism. **Note:** if D1 removes exemption
  markings from the banner, this normalization still applies wherever exemptions are captured.
- **Prompt follow-up — routing/cover-page fabrication.** `uniform-unclassified` occasionally
  fabricates a base level (`CONF.`) on a heavily-redacted routing sheet. The read-don't-infer
  language exists; reinforce it for unmarked/redacted pages. The contrast/clean-render work may
  also reduce the redaction-edge noise that seeds the hallucination.

---

## 7. Changes (per file) — APPLIED (pending live validation)

**`internal/prompts/specs.go` (immutable — policy-critical rules):**
- `finalizeSpec`: replace naive union with **policy-aware assembly** — encode NOFORN⊻REL TO,
  REL TO suppression, no-FOUO-hoist, DISPLAY ONLY exclusions (G2); rename the trailing category to
  **OTHER DISSEM** with the correct membership (G3); add **AEA (RD/FRD)** category + declass-date
  suppression (G4); resolve D1 for declass exemptions.
- `classifySpec`: reinforce "**any faint/partial content in a banner position ⇒
  `undetermined_markings` + `enhancements`**" (§5); keep degraded-token resolution.
- `enhanceSpec`: unchanged in substance; confirm it still adds rather than replaces.

**`internal/prompts/instructions.go` (tunable defaults — guidance):**
- `classifyInstructions`: add the **"verify every banner position (top, bottom, portion marks)
  independently"** directive (§5); reinforce read-don't-infer for unmarked/redacted routing/cover
  pages (G7, §6).
- `finalizeInstructions`: align the prose with the policy-aware assembly (mirror the spec).
- Apply `XI/Xl → X1` glyph normalization guidance alongside the existing `NOFOPI → NOFORN` example.

**Tests (`tests/prompts`, `tests/workflow`, `tests/state`):** update for any structural change from
D1/D2/D3. Re-validate the scenario set after.

---

## 8. Decisions (locked)

- **D1 — Declass exemptions in the output banner? → STRICT POLICY BANNER.** `classification` =
  `CLASSIFICATION//controls` only; declassification/exemption markings (`X1`, `25X1`, `50X1-HUM`,
  `OADR`, `MR`, dates) are **excluded from the banner string**. They are still transcribed per page
  (transcribe-as-found) and reported in the finalize **rationale**. This de-overfits the `x1`/`xi`
  scenarios — their expected `classification` output changes (the banner no longer carries `X1`).
- **D1a — Where do excluded declass/exemption values surface? → RATIONALE ONLY.** Finalize reports
  the document's declassification/exemption instructions in the rationale prose; no schema/API/web
  change. A dedicated structured field is a documented future enhancement.
- **D2 — Combination-logic depth → FULL DoDM RULE SET.** Encode the complete combination logic in
  the immutable spec: NOFORN⊻REL TO (NOFORN wins), REL TO suppression unless wholly releasable,
  no FOUO/CUI hoisting, DISPLAY ONLY/RELIDO exclusions, HCS/TK→NOFORN, RD/FRD as AEA category with
  declass-date suppression, FGI interactions. Keep wording tight to limit misfire surface.
- **D3 — Banner assembly → CUMULATIVE UNION, POLICY-AWARE, DoD BASELINE.** Keep assembling from the
  union of per-page markings (correct for the escalation scenarios) but apply the D2 combination
  guardrails, using DoDM 5200.01 (the DoD baseline) as the primary rule set.
- **D4 — Rule set → DoD baseline primary** (folded into D3).

---

## 8a. Follow-ups (post-validation)

- **Retry — RESOLVED (tau `agent v0.1.2`).** The client retried `429/502/503/504` + network but NOT
  `500`; a transient Azure `500` aborted a 27-page run. Fixed in the client (retry is a transport
  concern), not Herald — `isRetryableError` now covers `408 + 429 + all 5xx`
  (`~/tau/agent/_project/retry-optimization.md`). Herald bumped `agent v0.1.1 → v0.1.2`; no Herald
  code change (inherits the client's default 3× backoff+jitter; tunable via `agent.client.retry`).
- **Minor prompt refinements (non-blocking, observed in the 27-page run):**
  - Pages 12/13/25 recorded markings as labeled prose (`"CLASSIFICATION: SECRET CAVEATS: NOFORN"`)
    instead of canonical `SECRET//NOFORN`. Finalize parsed it correctly; consider nudging
    `classifySpec` toward canonical marking strings.
  - An `enhance` pass returned `LOW` with empty `undetermined_markings` (page 18, illegible "50"
    stamp). Slightly inconsistent with the enhance confidence rules (LOW should pair with an
    unresolved/contradictory marking). Outcome was safe (document → LOW/review). Consider tightening
    the `enhanceSpec` confidence wording.

## 9. Validation plan

0. **Prerequisite (blocking):** apply the custom Azure content filter (thresholds → High) and assign
   it to the `gpt-5.2` deployment — the full-rule-set `finalizeSpec` trips the default RAI "hate"
   filter at medium. See `azure-content-filter.md`. Finalize validation is blocked until this is done.
1. Land the agreed prompt changes (spec + instructions).
2. **Split silent-drop:** re-run `single-secret-noforn-split.pdf` ×5+ — target: NOFORN is *never*
   silently dropped; it is either read (SECRET//NOFORN) or flagged `undetermined` → LOW/review.
   Watch the token logs to confirm enhance triggers when flagged.
3. **Policy-assembly regression:** re-run the escalation/uniform/caveat-accumulation scenarios —
   confirm cumulative banners assemble correctly and no policy-invalid combinations are emitted.
4. **De-overfit check:** confirm `x1`/`xi` reflect the D1 decision; `xi` resolves the glyph.
5. **Clean-doc regression:** `single-secret`, `single-confidential`, `single-unclassified`,
   `uniform-*` — no classification-string or confidence regressions.
6. `mise run vet`, `mise run test`, `mise run web:build` clean.
7. If §5 + contrast still leave an unsafe silent-drop tail, escalate to render opt #4 (banner crops).
