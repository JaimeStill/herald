# Core API Retrospective — Presenterm Deck Plan

## Context

The Herald architecture retrospective (`~/code/herald/_project/presentation/retrospective.md`) is complete. The next deliverable is a sibling retrospective for the **Core API** project — the first containerized service effort, which preceded Herald — following the same presenterm formatting conventions, kanagawa-themed Typst diagrams, and mise render pipeline. Skeleton notes exist at `~/notes/core-api-retrospective.md`; no diagrams exist yet for this project.

The deck will be authored **one slide at a time**: draft → user review/refine → next slide. This plan establishes the scaffold, the slide outline, and the per-slide workflow — not final slide content.

## Decisions (confirmed with user)

| Decision | Value |
|---|---|
| Location | Standalone directory: `~/code/_s2va/core-api-retrospective/` |
| Diagrams | One: current-state architecture (`assets/overview.typ` → `overview.png`) |
| ETL framing | Frame Dapper services as a live **transformation/projection layer** (extract + transform per-request, no load); position true ETL (sync + cache) as the *evolution* in the pivot slide — matches the personnel-service-demo sync-worker pattern. User may tune wording during slide drafting. |
| Lineage | Yes — closing context: Core API seeded the patterns matured through Herald, SHF Tracker, and the SAHRMS engagement scenario |

## Key References

- **Format source of truth**: `~/code/herald/_project/presentation/retrospective.md` (frontmatter, directives, slide rhythm) and `~/code/herald/_project/presentation/mise.toml` (render pipeline)
- **Design assets to copy**: `~/code/herald/_project/presentation/assets/design/{tokens.typ, theme.typ, README.md}` — kanagawa palette + CaskaydiaMono NFP tokens
- **Diagram skills**: `~/tau/diagrams/.claude/skills/` (diagram-authoring, diagram-ingredients, typst-diagrams, diagram-design-system) — process and primitives; **palette stays kanagawa** (not tau's GitHub Primer) to match the terminal-dark deck
- **Content sources**: `~/notes/core-api-retrospective.md` (skeleton), `~/code/_s2va/capabilities/servers/core/{spec.md, readme.md, deploy.md}`, `~/code/_s2va/capabilities/databases/crm/`, `~/code/_s2va/capabilities/guides/core-api-testing/readme.md`, `~/code/_s2va/personnel-service-demo/` (SAHRMS pivot context)
- **presenterm source**: `~/presenterm` (directive reference if needed)

## Phase 0 — Scaffold

Create `~/code/_s2va/core-api-retrospective/`:

```
core-api-retrospective/
├── README.md            # the deck
├── mise.toml            # render / export / export:pdf / clean (copied from Herald, paths unchanged)
└── assets/
    └── design/
        ├── tokens.typ   # copied verbatim from Herald
        ├── theme.typ    # copied from Herald; accent-mapping comments updated for Core API entities
        └── README.md    # design philosophy, updated for this deck
```

- **mise.toml**: Herald's tasks verbatim — Typst → SVG → `resvg --width 2560 --background "#1f1f28"` → PNG; `export` / `export:pdf` target `README.md`.
- **README.md frontmatter**: Herald retrospective pattern — `terminal-dark` theme, pinned `background: "1f1f28"` / `foreground: "dcd7ba"` (PDF export compat), `slide_title.font_size: 3`, footer center `"Core API Retrospective"`, right `"{current_slide} / {total_slides}"`.
- Scaffold ends with the title slide only (slide 1), then pause for review.

## Proposed Slide Outline (8 slides)

Mapped from the skeleton, following Herald's retrospective rhythm (`font_size: 2` body, `list_item_newlines`, `new_lines` spacing, `<!-- end_slide -->`):

1. **Title** — `jump_to_middle`, centered, colored span title, "Jaime Still"
2. **The Problem** — CRM is the only authoritative source of unit personnel data; the only interface is tight coupling to SQL Server; the systemic gap: no strategy for scoping data domains behind modern service contracts
3. **Design Goals** — critical "core" subset of the CRM schema; transformation/projection service layer; containerized read-only REST API; Entra OAuth/OIDC; build organizational understanding of modernization
4. **Architecture** — prose + `![image:width:90%](assets/overview.png)` — the diagram designed in this session
5. **Current State** — deployed to IL6 AKS app platform (now unmaintained); AD service account + Kerberos service container to on-prem SQL Server; pivot plan: native managed-identity Entra auth (SQL Server 2022+) → App Service interim
6. **The Pivot** — stop-gap until SAHRMS replaces CRM; internal source shift to SAHRMS endpoints **without altering the public contract**; evolution into the cache + integration point for external authoritative data (true ETL — the pattern proven in the SAHRMS engagement scenario)
7. **Why It Matters** — systemic problem beyond the organization; opportunity to set the modernization standard; domain-driven service architecture cleans up internal + hierarchical data flow; controlled access; foundation for operationalizing AI against the data platform
8. **Lineage & Close** — first containerized service effort; patterns matured through Herald, SHF Tracker, and the SAHRMS engagement scenario; "Open for discussion."

Outline is a starting proposal — the user refines per-slide; slides may merge/split during review.

## Diagram: Current-State Architecture (`assets/overview.typ`)

Designed when we reach slide 4, using the diagram-authoring flow (gather → textual sketch → compose → render → critique, 2–3 iterations with the user):

- **Entities (sketch)**: consumer applications → Core API (ASP.NET Core 9 minimal API, containerized; Entra JWT validation; ProjectionMap/QueryBuilder/Dapper projection layer) → Kerberos service container + AD service account → on-prem SQL Server (CRM). Azure Entra as the token authority.
- **Conventions**: kanagawa palette, outline-only shapes (fill: none), accent = stroke + title color, body text `ink`, neutral edges with muted italic labels, `#import "@preview/fletcher:0.5.8"`.
- **Accent mapping (proposed, confirm during design)**: Core API = violet (focal service, matching Herald's focal slot); consumers/actors = cyan; Azure services (Entra) = blue; legacy CRM/SQL Server = yellow or neutral `border` (legacy de-emphasis); decide red for the unmaintained-platform callout if needed.
- If the diagram is wide-and-shallow, apply Herald's local scale bump (`#let scale = 1.7` in the diagram file, not shared tokens).
- Render via `mise run render`; embed as `![image:width:90%](assets/overview.png)`.

## Per-Slide Workflow (Phases 1–8)

For each slide, in order:
1. Draft the slide into `README.md` from the skeleton + repo facts (verify any technical claims against the source repos before writing them).
2. Tell the user it's ready; they review in presenterm (`presenterm README.md`) and give refinements.
3. Iterate until accepted, then move to the next slide.

Slide 4 additionally runs the diagram design sub-workflow above before the slide is finalized.

## Verification

- `mise run render` in the new directory compiles the diagram cleanly (typst + resvg on PATH — proven by the Herald deck on this machine).
- `presenterm README.md` renders each slide for the user's live review (user-driven; it's an interactive TUI).
- After the final slide: `mise run export` and `mise run export:pdf` to confirm the deck exports cleanly with the pinned background colors.

## Out of Scope

- No changes to the Herald repo, the capabilities repo, or tau/diagrams.
- Evolution/pivot and query-pipeline diagrams were considered and deferred — slide 6 carries the pivot in prose.
- No git init/commit unless the user asks once content exists.
