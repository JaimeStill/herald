# Herald Briefing — presenterm Presentation

## Context

A 30-minute briefing on the Herald project is scheduled for next Wednesday. Herald is a Go web
service that classifies DoD PDF security markings using Azure AI Foundry vision models, built to
replace manual classification of ~750k classified documents from a legacy program of record.

The deliverable is a terminal-native [presenterm](https://github.com/mfontanini/presenterm)
deck driven live from the terminal, mirroring the proven structure and visual language of the
`~/code/_s2va/personnel-service-demo/presentation` deck (front matter, `terminal-dark` theme,
kanagawa-palette Typst/Fletcher diagrams rendered to PNG). The deck must stay **lean (~14–16
slides)** and keep **Herald as the focus** — the TAU library ecosystem is the foundation Herald
is built on, not the subject. Two of the six sections are **live demos**; their slides are
command cue-cards, not screenshots.

Tooling is confirmed present: `presenterm 0.16.1`, `typst`, `resvg`, `magick`. Fonts: diagrams
use `CaskaydiaMono NFP` (already used by the OV-1 diagrams, so installed).

### Confirmed decisions
- **Location:** `_project/presentation/`
- **Diagrams:** all four, authored fresh in the **kanagawa terminal palette** (not the OV-1
  Primer palette). They may reuse OV-1 diagram *structure* as a starting point — the objection
  is purely visual. Diagrams are **dark-only** (canvas == terminal background).
- **Numbers (use these, not the repo README's 1M/96.3%):** ~750k documents; 25 documents
  validated at **100% accuracy** in IL6; projected cost **$27k inference + ~1 month of managed
  services** for the full 750k run.
- **Demo slides:** command cue-cards only.
- **Process:** build **incrementally, one layer at a time**. We design and review each layer
  together and do **not** advance to the next until both of us are aligned on the current one.

## Build sequence (incremental — align before advancing)

Each layer is its own review checkpoint. Nothing in a later layer is written until the prior
layer is reviewed and agreed.

- **Layer 1 — Foundation & design system.** Scaffold `_project/presentation/`; copy/adapt the
  kanagawa `design/` layer; lock Herald's accent conventions; stand up the render task and
  prove it with one throwaway render. *(current focus)*
- **Layer 2 — Diagrams ✓ DONE.** All three authored + approved: (a) `overview` ✓,
  (b) `classification` ✓, (c) `release` ✓. Three diagrams, not four — the supply-chain
  footprint is told with real config (code-snippet slides 8–9), not a diagram. `release` is a
  two-row composition: artifacts/pipeline/storage spine across the top (Service image → Migrate
  tool → CDS proxy → CDS Destination, collinear with the cross-domain transfer edge), each
  band's maintainer centred at the bottom with 90° L-edges up into the shapes it triggers.
- **Layer 3 — Deck skeleton ✓ DONE.** `README.md` built: front matter (title/sub_title/author,
  terminal-dark, footer), 3 `jump_to_middle` section dividers, 13 content slides with headers +
  `<!-- LAYER 4: … -->` placeholder comments marking where body content lands. Verified the
  intro slide renders in presenterm.
- **Layer 4 — Slide content ✓ DONE.** All slides filled: Overview, Scope, Classification
  workflow, Demo cue-card (1 image + PDF + live Azure step), footprint divider, Dependencies
  (3-col), No framework (2-col: component layering + design system), TAU agent (Bedrock demo +
  database/sql analogy + 3-col driver configs), Release to IL6, Close (REST-first; 25 @ 100%;
  $27k + 1mo for 750k).
- **Layer 5 — End-to-end verification.** Render, page through in presenterm, dry-run both demos.
  (User handles validation/review directly — do not run presenterm export/render checks to verify.)

## STATUS: COMPLETE (2026-05-30)

Deck approved complete by the user. Final state:
- `_project/presentation/README.md` — 14 slides (intro + 13 content + 2 jump_to_middle dividers).
- 3 diagrams (`overview`, `classification`, `release`), kanagawa palette, rendered to PNG.
- Render task caps output at `--width 2560` (1440p-appropriate; was `--zoom 3` ≈ 7000px).
- `classification.typ` + `release.typ` carry a local `scale = 1.7` font multiplier (NOT in shared
  tokens.typ) because they are the widest diagrams and shrink most when fit to slide width.
  `overview.typ` uses base token sizes. If editing these, keep the scale local.
- LOCKED accent map: Herald=violet · actors=cyan · Azure=blue · TAU=green · third-party=neutral.

The sections below are reference detail for Layers 2–4; treat them as a starting proposal, not
a locked spec — each gets reviewed when its layer comes up.

## Design system (do this first)

Copy the personnel-demo terminal-native design layer into the new deck, then adapt accents for
Herald's entities:

1. Create `_project/presentation/assets/design/` and copy verbatim from
   `~/code/_s2va/personnel-service-demo/presentation/assets/design/`:
   - `tokens.typ` (colourless — typography/spacing/geometry/stroke; copy unchanged)
   - `theme.typ` (flat kanagawa palette: `surface #1f1f28`, `ink #dcd7ba`, `border #54546d`,
     accents `red green yellow blue violet cyan orange`)
   - `README.md` (the terminal-native philosophy doc) — adapt its "authoring conventions"
     section to Herald's accent assignments below.
2. **Herald accent conventions** (LOCKED — one accent per entity kind, applied consistently
   across all four diagrams):
   - **Herald service = red** (focal node)
   - **Actors** (document source, reviewer) = **orange**
   - **Azure managed services = blue** (AI Foundry, PostgreSQL, Blob, Container Apps)
   - **TAU libraries = green**; **third-party deps = neutral `border`**; **edges = `ink-subtle`**
   - Services/libraries are outline-only rectangles (`fill: none`, accent stroke + accent
     title, `ink` body); databases are neutral cylinders. Edges neutral, italic labels on a
     `surface` label-fill.

## Diagrams (4, kanagawa, dark-only PNG)

Author each as `assets/<name>.typ` → render to `assets/<name>.png`. Reuse the Fletcher
`kinded()` shape helper pattern from `_project/ov-1/core/readme.typ:40` (icon-left + stacked
title / (kind) / divider / description), re-pointed at the kanagawa `palette`.

Accents follow the LOCKED map: Herald = violet (focal) · actors = cyan · Azure managed
services = blue · TAU libraries = green · third-party = neutral border · edges = ink-subtle.
(There is no supply-chain diagram — that story is told with real config on slides 8–9.)

1. **`overview.typ`** ✓ DONE — Architecture overview (anchor visual, Overview slide). Herald
   (violet, focal) between the two actors (cyan: document source, reviewer); the four Azure
   managed services (blue) grouped in an enclosing container titled "Azure Managed Services /
   user-managed identity", with a stealth `-|>` deployment edge docking Container Apps.
2. **`classification.typ`** — Classification workflow state graph (primes the demo). Nodes:
   `init` (format dispatch / rasterize) → `classify` (parallel per-page vision) → conditional
   `enhance` (re-render flagged pages) → `finalize` (document-level synthesis), with low
   confidence routed to human review. Re-skin of `_project/ov-1/core/classification.typ`.
3. **`release.typ`** — Release → CDS → IL6 pipeline. Tag → GHCR image + migrate binary; human
   gate → IL4 proxy repo → cross-domain transfer service → IL6 secure storage. Re-skin of
   `_project/ov-1/core/release.typ`.

### Render task
Create `_project/presentation/mise.toml` with a `render` task (adapt the personnel-demo
pipeline — single-theme, since kanagawa is dark-only):
```
typst compile --root . assets/<name>.typ assets/<name>.svg
resvg --zoom 3 --background "#1f1f28" assets/<name>.svg assets/<name>.png
```
Loop over all `assets/*.typ`. Commit the `.png` outputs (presenterm's image crate has no SVG
decoder, so PNG is required).

## Slide deck — `_project/presentation/README.md` (~15 slides)

Front matter mirrors the personnel deck, retitled:
```yaml
---
title: Herald
sub_title: "AI-Driven Security Classification of DoD Documents"
author: "<author/team>"
theme:
  name: terminal-dark
  override:
    intro_slide:
      author: { positioning: below_title }
    footer:
      style: template
      center: "Herald"
      right: "{current_slide} / {total_slides}"
---
```
Proven directives (same presenterm build as the personnel deck): `<!-- end_slide -->`,
`<!-- jump_to_middle -->` for section dividers, `<!-- newlines: N -->`, image embed
`![image:width:90%](assets/overview.png)`.

Slide sequence:
1. **(auto) intro slide** — from front matter.
2. **Overview** — the 750k-document driver: a legacy program of record has ~750k classified
   documents needing classification records; manual verification is prohibitively costly;
   Herald analyzes the markings and returns a classification + confidence (low/medium/high) +
   reasoning. `![](assets/overview.png)`.
3. **Scope** — numbered agenda of the six topics (Overview → Demo → Minimal footprint → TAU
   agent → Distribution → Close).
4. **Divider:** `Herald, live` (`jump_to_middle`).
5. **Classification workflow** — `![](assets/classification.png)` + one-line description of
   what the demo will show.
6. **Demo — cue-card** — exact commands and the walkthrough checklist (no screenshots):
   ```
   mise run stack:up      # Postgres + Azurite, migrations applied
   mise run dev:auth      # Go server (air) + web client (bun watch), Entra overlay
   ```
   Walkthrough at `http://localhost:8080/app`: (1) upload ONE image
   `_project/marked-documents/images/marked-document.12.png` → (2) upload PDF
   `_project/marked-documents/escalation-secret-to-noforn.pdf` → (3) classification workflow →
   (4) review view → (5) prompts view → (6) Container App: show Herald live in Azure at the
   deployed Container Apps URL.
   NOTE: image files use DOTS not dashes (`marked-document.12.png`), and live in the `images/`
   subdirectory. PDFs live directly under `_project/marked-documents/`. Only ONE image is
   demoed (marked-document.18.png was cut). Step 6 (live Azure deployment) was added.
7. **Divider:** `A minimal footprint` (`jump_to_middle`). The divider (or a short lead-in
   beneath the centred title) must carry the SECTION MOTIVATION, not just announce the topic:
   this architecture applies lessons learned from prior classified-project work. The driving
   question — what does sustainable, cloud-native architecture look like when you build against
   open standards and refuse the frameworks/shortcuts that come back to bite you? The minimal
   footprint (slides 8–9) is the answer to that question, not a stat for its own sake.
8. **Dependencies — "A small, isolated footprint" (3-column, real config).** Told with actual
   `go.mod` / `package.json`, not a diagram. Layout `<!-- column_layout: [1, 1, 1] -->`.
   - **Col 0 — Libraries, isolated.** The architectural argument (the headline). A compact tree
     of `tailored-agentic-units/*` showing third-party SDKs do NOT propagate upward:
     ```text
     protocol       0 deps  ← contracts
     provider       0 deps  ← interface
      ├ azure        Azure SDK
      ├ bedrock      AWS SDK
      └ ollama       0 deps
     format         0 deps  ← interface
      ├ converse     0 deps
      └ openai       0 deps
     agent          uuid
     orchestrate    uuid
     ```
     Sub-modules verified present: provider/{azure,bedrock,ollama}, format/{converse,openai}.
     Direct external deps: provider/azure→Azure SDK, provider/bedrock→AWS SDK; ALL of
     provider/ollama, format/converse, format/openai are pure Go (0 external). agent +
     orchestrate only pull google/uuid. Herald imports provider/azure + format/openai, so the
     AWS SDK never enters its build.
     Caption: a base module defines the **interface** and stays dependency-free; a vendor SDK
     lives in a **sub-module** and only enters the build when you import it — it never
     propagates up the dependency layers.
   - **Col 1 — Service `go.mod`.** 9 direct third-party, `github.com/` prefix stripped, the
     three Azure SDK packages grouped: `azcore · azidentity · azblob`, then `coreos/go-oidc/v3`,
     `golang-migrate/migrate/v4`, `google/uuid`, `jackc/pgx/v5`, `pdfcpu/pdfcpu`,
     `golang.org/x/sync`. Note: everything else is first-party.
   - **Col 2 — Client `package.json`.** `dependencies`: `@azure/msal-browser`, `lit`;
     `devDependencies`: `@types/bun`. Labeled "2 runtime + 1 build-time". One-liner: single Go
     binary — the Lit client is embedded (`go:embed`) and served at `/app`.
   - Verified facts (from go.mod inspection): protocol/provider/format carry 0 external deps;
     provider/azure is the ONLY TAU module with a vendor SDK (azcore + azidentity); format/openai
     and provider/ollama are pure Go (0 external); agent + orchestrate only pull google/uuid.
9. **No framework, by design.** The client leans on the web platform, not a framework stack.
   - **No frontend framework** (no React/Vue/Angular/Svelte) — `lit` is used only to emit native
     Web Components, standard custom elements the browser runs directly.
   - **No design library** (no Tailwind/Bootstrap/MUI) — a token-based, native-CSS design system
     we own (CSS custom properties + cascade layers), nothing to install.
   - Close: the modern web platform is mature enough to build powerful apps on standard features
     alone; skipping the framework sheds its **complexity**, runtime **inefficiency**,
     **vulnerability surface**, and ongoing **maintenance burden**. (Consider a `<!-- pause -->`
     before this list — decide when viewed in presenterm.)
10. **TAU agent example — cue-card.** Theme: the TAU libraries are REUSABLE components; Herald is
    not a disposable monolith. Lead paragraph ends with the `database/sql` analogy — the agent is
    a configurable client over a driver registry, like database/sql interfacing any SQL server via
    a swappable driver. Demo via `prompt-agent` using **AWS Bedrock + Claude + `converse`** (ties
    to the footprint slide's bedrock sub-module); provider/format swap by CONFIG alone. Keep BOTH
    commands (chat + vision). `aws login` first.
    ```
    cd ~/tau/examples && aws login
    go run ./cmd/prompt-agent -config ./cmd/prompt-agent/config.bedrock.json \
      -prompt "What is infrastructure as code? 300 words or less" -stream
    # vision: -protocol vision -images <url>
    ```
    Config verified: config.bedrock.json → provider=bedrock, format=converse, model=Claude Haiku
    4.5. Bottom: 3-column [1,1,1] condensed driver-selection slices (provider/format/model only)
    from config.{ollama,azure,bedrock}.json — same CLI, three drivers. Call out Azure OpenAI +
    Ollama as drop-in (skip live Ollama: time / VRAM warmup). Point to `~/tau/examples`
    (prompt-agent + orchestrate; also on github.com/tailored-agentic-units/examples).
11. **Divider:** `Distribution` (`jump_to_middle`).
12. **Release → IL6** — `![](assets/release.png)` + bullets: GHCR image + migrate binary on
    tag; IL4 proxy repo (version input) packages the resource and pushes through CDS to IL6.
13. **Close** — Herald is primarily a **REST API**; the client exists to demonstrate the
    service and provide the review interface. Deployed to **IL6 Azure**; **25 documents tested
    at 100% accuracy**; projected **$27k inference + ~1 month managed services** for 750k docs.
    "Open for questions."

(Optionally add a one-line `presentation-plan.md` / speaker-notes file as the personnel deck
does — skip unless wanted, to stay lean.)

## Critical files / references
- New: `_project/presentation/README.md`, `_project/presentation/mise.toml`,
  `_project/presentation/assets/design/{tokens.typ,theme.typ,README.md}`,
  `_project/presentation/assets/{overview,classification,release}.{typ,svg,png}`
- Pattern sources (read, adapt, re-skin): `_project/ov-1/core/readme.typ`,
  `classification.typ`, `release.typ`; `_project/ov-1/design/{tokens,theme}.typ`
- Deck structure source: `~/code/_s2va/personnel-service-demo/presentation/README.md` and
  `assets/design/README.md`
- Demo facts: `.mise.toml` (`stack:up`, `dev:auth`), `app/app.go` (`go:embed` + `/app` mount),
  `go.mod` / `app/package.json` (dependency counts), `~/tau/{provider,format}` (module
  isolation), `~/tau/examples/cmd/prompt-agent` (CLI flags)

## Verification
1. `cd _project/presentation && mise run render` — confirm all four `assets/*.png` produced,
   crisp, kanagawa surface background.
2. `presenterm _project/presentation/README.md` — page through all slides; confirm diagrams
   render inline (terminal with image support), section dividers center, footer shows
   `Herald N / total`, no overflow/clipping on diagram slides.
3. Dry-run the two demo cue-cards end to end: `mise run stack:up` then `mise run dev:auth`,
   upload the three demo files, walk upload → classify → review → prompts; then run the
   `prompt-agent` vision command. Confirm every command on the cue-card slides is accurate.
4. Spot-check headline numbers on the Close slide against the agreed figures (750k / 25 @ 100%
   / $27k + 1 month).
