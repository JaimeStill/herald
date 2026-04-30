# Initialize: ~/tau/herald-overview-brief.md

## Context

`~/tau/herald-overview-brief.md` is an *orientation document* (a meta-plan, not the deliverable itself) for producing a leadership-facing briefing of `~/code/herald`. Its current draft scopes the briefing to the `core/` tier of TAU's diagram framework — **OV-1 stakeholder voice, multi-diagram core/, PDF for distribution**. That framing is correct and stays.

What the brief does **not** yet pin down is the diagram inventory. The user has now locked it: **5 diagrams**, all in `core/`, all in OV-1 stakeholder voice, with prose in the README articulating the technical core concepts at a leadership level.

| # | Subject | Source for findings |
|---|---|---|
| 1 | Overall architecture + external integrations (Azure AI Foundry, Azure PostgreSQL, Azure Blob Storage; optional Entra/MSI) | `~/code/herald/_project/README.md`, `internal/infrastructure/`, `internal/config/` |
| 2 | Document upload process | `internal/documents/handler.go:120`, `internal/documents/repository.go`, `internal/format/` |
| 3 | Document classification workflow (init → classify → enhance? → finalize) | `internal/classifications/handler.go:130`, `internal/workflow/` |
| 4 | Release workflow (CI → GHCR + GitHub Release) | `~/code/herald/.github/workflows/release.yml` |
| 5 | Proxy CDS release process (cross-domain transfer to IL6) | `~/code/_s2va/herald/.github/workflows/image-release.yaml` |

**Why this matters for TAU's diagram project.** Diagrams 2–5 are flow/process/pipeline shapes. TAU's existing rendered corpus (protocol, format) only contains layered architecture and decomposition — **no flow/sequence/pipeline diagrams exist yet**. This session is the first chance to surface those conventions and capture them in the phase-04 prelude artifact.

The session has two halves, in this order:

1. **Revise the brief** at `~/tau/herald-overview-brief.md` so it reflects the locked decisions (inventory, sensitivity boundary, PDF kept, output location confirmed) and seeds the new flow/process pattern category.
2. **Execute the authoring loop** end-to-end against the revised brief: technical-writer subagent → tier overview → 4 additional diagrams → README + PDF + prelude artifact → commit.

## Locked decisions (from this planning conversation)

| Decision | Value |
|---|---|
| Audience | Non-technical company leadership, OV-1 framing throughout |
| Tier | `core/` only (no operational/specification) |
| Diagram inventory | 5 fixed: `readme` (architecture), `upload`, `classification`, `release`, `cds-release` |
| Voice | Stakeholder voice; external services *named* (Azure AI Foundry, Azure PostgreSQL, etc. — they are load-bearing for the briefing) but no type/method internals |
| Output location | `~/code/herald/diagrams/core/` (colocated with herald) |
| Render pipeline | Mirror TAU's mise pipeline; copy `mise.toml` + `design/` from `~/tau/diagrams/` |
| PDF deliverable | **Required** — pandoc + typst engine; light-only; for email distribution |
| CDS sensitivity | **Public abstractions only** — show: tag push → pull from GHCR → bundle → upload to CDS storage destination → manifest action; hide: tenant IDs, account/container names, signing internals, runner identity |
| Subagent step | Run `technical-writer` against the locked inventory for per-diagram structured findings |
| Visual approach for flow/process diagrams | Brief lists 2–3 candidate ingredient combinations; author chooses per diagram (toolkit-not-ruleset) |

## Phase A — Revise the brief

Single file edited: `~/tau/herald-overview-brief.md`. Targeted revisions only; do not rewrite working sections.

### Edits

1. **Diagram inventory** (Scope: full multi-diagram core/ tier, lines ~47–59).
   Replace the "Likely candidates (technical-writer surfaces the actual set)" list with the locked 5-diagram inventory; remove "2–5 diagrams" and "Final inventory is decided per technical-writer's findings + user review" since the inventory is now pre-decided. Keep one-diagram-one-concept and stakeholder voice.

2. **Visual approach for process/flow diagrams** (new section, after *Conventions to apply*).
   Note that TAU's rendered corpus has no flow/sequence/pipeline examples; diagrams 2–5 will surface a new pattern category. Offer 2–3 candidate ingredient combinations the author can choose from per diagram, e.g.:
   - **Labeled milestone strip** — a horizontal sequence of node "stops" with edge labels for the transitions; reads like a journey.
   - **Step-and-actor flow** — actors (boxes) on a baseline, steps (small numbered tags) on directional edges; reads like a sequence diagram without time-axis chrome.
   - **State-graph card** — nodes as states, edges as transitions, with a conditional fork for diagrams that have a branch (classification's `enhance?` is the canonical case).
   These are descriptive starting points, not contracts. Each diagram opts in or composes fresh.

3. **Sensitivity boundary** (in *Open questions* item #3 → promote to a confirmed section, e.g., *Sensitivity*).
   State the public-abstractions-only rule for the CDS-release diagram explicitly: show flow shape and external-system-class (`GHCR`, `cross-domain transfer`, `secure storage destination`); do not show tenant IDs, account/container names, signing internals, runner identity. The same rule applies anywhere else company/customer identity could leak (none expected for diagrams 1–4).

4. **Technical-writer invocation prompt** (lines ~78–86).
   Replace the open inventory-discovery framing with a per-diagram findings request: subagent receives the locked inventory and is asked to produce, for each of the 5 diagrams, the structured findings format from `technical-writer.md` — entities, relationships, layout intent, OV-1 stakeholder prose draft, open questions. Cross-repo note: the subagent must read both `~/code/herald` and `~/code/_s2va/herald` for the CDS-release diagram.

5. **Phase-04 prelude artifact** (lines ~143–176).
   Update *Categories to watch for* to lead with the genuinely new categories this session surfaces:
   - **Process/flow rendering at OV-1 level** — what works visually for a stakeholder-voiced upload/classification flow; ingredient combinations that hold up across 2–4 process diagrams in one briefing.
   - **CI/deployment-pipeline rendering at OV-1 level** — release.yml + cds-release as a related pair; the "tag → image → distribution" framing.
   - **Multi-external-service integration-surface rendering** — herald + Azure AI Foundry + Azure PostgreSQL + Azure Blob Storage as one OV-1 picture; how external services balance the focus shape without competing for visual weight.
   - **Sensitivity layer convention** — the public-abstraction rule applied per-diagram; what's redacted vs. retained.
   Keep the existing categories (multi-diagram core/ patterns, OV-1 framing, briefing-voiced prose, header metadata, PDF chain, toolkit friction) below these.

6. **Open questions section** (lines ~195–203).
   Trim. Remove items now confirmed: output location (1), render pipeline (2), PDF format (5 — A4 default, no cover page, CaskaydiaMono default unless visual issues surface), PDF distribution (6 — single PDF), preprocessing (7 — generated `README-pdf.md` variant approach). Keep only the ones still genuinely open: sensitivity nuance per diagram if findings surface anything 1–4 raise; audience size if the technical-writer's findings imply tighter or looser context-setting; PDF font fallback if rendering issues surface.

### What stays unchanged

- Purpose, Subject, Why-this-is-a-one-off, Bootstrap, OV-1 definition, Conventions to apply, Execution loop steps 2–6, PDF rendering toolchain, Verification, Handoff. These all match the locked decisions already.

## Phase B — Execute against the revised brief

Order matters. The first three steps are setup; once setup is done, the per-diagram loop runs five times (once for `core/readme.typ`, then four content-named diagrams).

### B1. Set up the herald diagrams workspace

- Create `~/code/herald/diagrams/core/`.
- Copy `~/tau/diagrams/mise.toml` and `~/tau/diagrams/design/` into `~/code/herald/diagrams/`. Verify imports resolve (sources will import `../design/tokens.typ`, etc.).
- Replicate the technical-writer subagent symlink at `~/code/herald/.claude/agents/technical-writer.md → ~/tau/diagrams/.claude/agents/technical-writer.md` so subagent invocation from herald discovers it (per `03-core-tau-diagrams.md` line 19 convention).
- Confirm `pandoc` is installed; if not, install via `pacman -S pandoc` (with user confirmation before invoking system package manager).

### B2. Invoke `technical-writer` subagent

Single invocation, scoped to the locked inventory. The prompt explicitly:
- Names the 5 diagrams.
- Anchors voice (OV-1 stakeholder, prose carries the technical concept).
- Gives sensitivity guidance (public abstractions for cds-release; no tenant info anywhere).
- Asks for per-diagram findings: entities (with kind), relationships (with kind), layout intent, OV-1 prose draft, open questions.
- Points the subagent at both `~/code/herald` and `~/code/_s2va/herald`.

Findings are reviewed with the user before any `.typ` source is written. Open questions resolved here.

### B3. Author per the loop, in this order

1. **`core/readme.typ`** — Herald + integrations OV-1 architecture. Sets the visual tone for the briefing. Render dual-theme. Write H1 metadata block + 1–2 sentence stakeholder prose under the leading `<picture>`.
2. **`core/upload.typ`** — document upload flow (user → herald → registers + stores). Render. Write H2 + prose.
3. **`core/classification.typ`** — workflow (init → classify → enhance? → finalize → result), with conditional fork. Render. Write H2 + prose.
4. **`core/release.typ`** — release pipeline (tag → build → publish image + GitHub Release). Render. Write H2 + prose.
5. **`core/cds-release.typ`** — CDS proxy pipeline (tag → pull from GHCR → bundle → cross-domain transfer to secure environment), public abstractions only. Render. Write H2 + prose.

Each diagram: source → render → critique (`diagram-authoring/references/critique-checklist.md`) → iterate → prose. The critique checklist's *one concept*, *shape for identity, text for metadata*, *visual weight tracks meaning*, and *both themes render* checks are non-negotiable.

### B4. Assemble the README

`~/code/herald/diagrams/README.md` follows the universal pattern from `03-core-tau-diagrams.md` lines 102–162:

```
# [herald](<repo url>)

Capability: <github url without https>
Language: Go
Native dependencies:    ← if any TAU library is referenced; likely none at OV-1 level
- ...
External dependencies:  ← Azure AI Foundry, Azure PostgreSQL, Azure Blob Storage
- ...

<picture>core/readme</picture>

[H1 stakeholder prose — what herald is, in 1-2 sentences]

## Upload

<picture>core/upload</picture>

[Stakeholder prose — what the diagram says, plus the technical core concept of upload]

## Classification

<picture>core/classification</picture>

[Stakeholder prose — vision-based marking analysis, the enhancement loop framed at concept level, the validation outcome]

## Release

<picture>core/release</picture>

[Stakeholder prose — automated release on tag; what's published]

## CDS release

<picture>core/cds-release</picture>

[Stakeholder prose — cross-domain transfer to secure environments, framed at concept level]
```

Voice stays stakeholder-OV-1 throughout. Code/method/type names do not appear in prose; technical core concepts (vision analysis, security marking classification, cross-domain transfer) do.

### B5. Render the briefing PDF

Per the brief's existing PDF rendering section:
1. Generate a `README-pdf.md` variant from `README.md` by replacing each `<picture>` block with a single `<img>` referencing the light SVG (script lives in mise as `briefing-pdf` task; commit the script, not the generated file).
2. `pandoc README-pdf.md -o briefing.pdf --pdf-engine=typst`.
3. A4, no cover page, CaskaydiaMono default; surface any font issues if they appear.
4. Place at `~/code/herald/diagrams/core/briefing.pdf`.

### B6. Capture phase-04 prelude artifact

Write `~/tau/diagrams/.claude/project/04-prelude-herald-ov1.md` per the template in the (revised) brief, leading with the new categories (process/flow, CI/deployment-pipeline, multi-external-service architecture, sensitivity layer). Cite the specific diagrams that drove each captured convention.

### B7. Commit

Single commit at `~/code/herald` with `git add .` (per user preference) and a message like `diagrams: add core overview (OV-1)`. The phase-04 prelude artifact at `~/tau/` is its own commit (no `Closes #N` per memory).

## Critical files

| Path | Role |
|---|---|
| `~/tau/herald-overview-brief.md` | The brief itself (revised in Phase A; orientation for Phase B) |
| `~/tau/diagrams/.claude/project/03-core-tau-diagrams.md` | Phase-03 authoring loop, README pattern, tier model — reference throughout Phase B |
| `~/tau/diagrams/.claude/project/04-advanced-tau-diagrams.md` | Phase-04 plan; describes herald as a phase-04 subject |
| `~/tau/diagrams/.claude/agents/technical-writer.md` | Subagent contract; symlink replicated at `~/code/herald/.claude/agents/` |
| `~/tau/diagrams/.claude/skills/tau-diagrams/SKILL.md` | Skill load order + locks |
| `~/tau/diagrams/.claude/skills/tau-diagrams/references/tau-decisions.md` | Palette, font, render pipeline, single-shape native deps |
| `~/tau/diagrams/.claude/skills/diagram-authoring/references/critique-checklist.md` | Per-diagram critique gate |
| `~/tau/diagrams/.claude/skills/diagram-ingredients/references/edges-and-marks.md` | Edge/mark vocabulary (load-bearing for the 4 flow diagrams) |
| `~/tau/diagrams/.claude/skills/typst-diagrams/references/render-pipeline.md` | Compile + sed strip pattern |
| `~/tau/diagrams/protocol/core/readme.typ` | Visual reference (blue focus + capability pills + muted layers) |
| `~/tau/diagrams/format/core/readme.typ` | Visual reference (most recent; forked translates-to edge example) |
| `~/tau/diagrams/design/{tokens,theme}.typ` | Design layer (copied to herald/diagrams/design/) |
| `~/tau/diagrams/mise.toml` | Render task (copied to herald/diagrams/) |
| `~/code/herald/_project/README.md` | Source for diagram 1 architecture findings |
| `~/code/herald/internal/{documents,classifications,workflow,format,infrastructure,config}/` | Source for diagrams 1–3 |
| `~/code/herald/.github/workflows/release.yml` | Source for diagram 4 |
| `~/code/_s2va/herald/.github/workflows/image-release.yaml` | Source for diagram 5 (cross-repo read) |
| `~/code/herald/diagrams/` | Output target (created in B1) |
| `~/tau/diagrams/.claude/project/04-prelude-herald-ov1.md` | Phase-04 prelude artifact (written in B6) |

## Verification

End-to-end checks before commit:

- **Sources render.** `mise run render` from `~/code/herald/diagrams/` produces 10 SVGs (5 diagrams × 2 themes) in `core/`. No compile errors. No missing imports.
- **Light/dark theme switching works.** Open the herald repo's `diagrams/README.md` in GitHub's Markdown preview (or a local renderer that honors `<picture>` + `prefers-color-scheme`); each diagram swaps between light and dark with the system theme.
- **Each diagram passes the critique checklist** at `~/tau/diagrams/.claude/skills/diagram-authoring/references/critique-checklist.md`. Specifically:
  - One concept per diagram.
  - Shape carries identity; text carries metadata.
  - Visual weight tracks importance.
  - Both themes render correctly (no faded states going wrong-direction in dark).
  - Edge encodings differentiate the flow's distinct relationships at a glance.
- **Voice holds.** Read each prose paragraph aloud; no Go type names, no method names, no package paths. The technical core concept (vision-based marking analysis, classification confidence, cross-domain transfer) is named in plain terms.
- **Sensitivity rule held.** The CDS-release diagram and prose contain no tenant IDs, no account/container names, no manifest action internals, no runner identity.
- **PDF renders.** `briefing.pdf` opens; embedded SVGs render at the right size; typography is readable at A4; no broken images. The PDF is self-contained (no external references that fail when the file is detached from the workspace).
- **Phase-04 prelude artifact exists** at `~/tau/diagrams/.claude/project/04-prelude-herald-ov1.md` with at least one captured convention per new category (process/flow, CI/deployment-pipeline, multi-external-service architecture, sensitivity layer).
- **Commits.** Herald repo committed (single commit) with `git add .`. Tau diagrams repo prelude artifact is its own commit (separate session-end push if desired).

## Risks and notes

- **Session length.** Five diagrams + README + PDF + prelude artifact is a long session. If context pressure surfaces during Phase B, the natural break is between B3 (per-diagram authoring) iterations — checkpoint at the partial state and resume.
- **Pandoc + typst PDF chain is unverified end-to-end.** The brief assumes it works; if SVG embedding has issues, fallback paths are: (a) inline-embed via `<img>` tags pointing to relative SVG paths, (b) pre-render SVG → PDF per diagram and inline as PDF pages, (c) use `wkhtmltopdf`. Verify on the first diagram before authoring all 5.
- **Sensitivity guidance is uniform** but the technical-writer's findings may surface per-diagram nuance (e.g., diagram 1 referencing specific Azure resource types — class names, not instances). Re-confirm with user during findings review if anything ambiguous appears.
- **Flow/process pattern is genuinely new.** Expect 1–2 extra critique-iterate cycles on diagrams 2 and 3 before the visual settles. The pattern that emerges feeds directly into the phase-04 prelude artifact.
