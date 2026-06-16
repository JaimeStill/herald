# Herald Retrospective — presenterm deck

## Context

Jaime is giving a **10-minute retrospective** on the development *and delivery* of Herald.
A rough outline exists at `_project/herald-architecture-retrospective.md`
(Problem → Design Goals → Ideology → Result → Lessons Learned). This effort turns it into a
presenterm deck at `_project/presentation/retrospective.md` that acts as a **speaker aid** —
concise talking-point bullets that jog memory, not a prescriptive script.

A polished product-overview deck already lives beside it at `_project/presentation/README.md`.
That deck is the *demo*; this one is the *reflection*. We inherit its house style and reuse its
rendered diagrams, but the content is distinct.

**Decisions locked with Jaime:**
- **Audience:** technical peers / engineers — can go deep on architecture, TAU isolation, lessons.
- **Emphasis:** even arc — each outline section gets ~2 slides.
- **Relationship to product deck:** reuse the kanagawa theme + footer and reuse the
  `classification.png` / `release.png` diagrams; write fresh, reflective content.

## Working method

**Build one slide at a time.** Draft a slide into `retrospective.md`, confirm it with Jaime,
then move to the next — no big-bang draft-then-refine. The slide list below is the agreed
skeleton; exact copy is decided live, slide by slide.

## Deck-wide setup

- **File:** `_project/presentation/retrospective.md` (same dir as `README.md`, so
  `assets/*.png` paths resolve identically).
- **Front matter:** copy the theme override block from `README.md:1-22` (terminal-dark +
  kanagawa background/foreground pin for PDF export, `below_title` author, template footer).
  **Title:** `Herald Architecture Retrospective`. **No `sub_title`.** Footer `center`:
  `"Herald Retrospective"`.
- **Idioms** (match README): setext titles (`Title\n===`), `<!-- end_slide -->` separators,
  `![image:width:90%](assets/<name>.png)`, `<!-- column_layout -->`, centered callouts.
- **No section dividers.** A section's first slide doubles as its intro (see slide 3).
- **No new diagrams** and **no `mise` export task.** Reuse the three already in `assets/`
  (`overview.png`, `classification.png`, `release.png`); the `render` task already builds them.
- **Locked facts:** corpus is **~750,000**; production accuracy is **26 documents @ 100%**.

## Slide skeleton (~11 slides, ~50s each)

1. **Title** — `Herald Architecture Retrospective` · Jaime Still.
2. **The Problem** — a legacy program of record **contains** ~750,000 classified documents,
   each needing a classification record; manually reading/recording markings at that volume is
   the prohibitive bottleneck.
3. **Design Goals (section intro)** — headline carries the framing that introduces slides 4–5:
   13 years building capabilities across the community; use this concrete problem to prove out a
   reusable architecture that confronts recurring friction points — not a one-off tool.
4. **Design Goals (the five)** — one keyworded bullet each: right tool per job (container tooling
   + Go + inference); reusable, vendor/model-agnostic inference + orchestration, minimal
   dependency footprint; agentic workflows inside a *traditional web service*; minimal-footprint
   service + embedded client on open standards, not frameworks; **dev low, push high** CI/CD +
   CDS with containers, binaries, IaC.
5. **The computation triad** (anchored by `classification.png`) — three kinds of computation,
   each best-suited: **deterministic container tooling** (ImageMagick rasterizes pages) →
   **deterministic orchestration** (Go drives the state graph, concurrency, persistence) →
   **non-deterministic inference** (the LLM reads markings and reasons). Expands goal #1.
6. **Ideology** — reverse-plan into targeted milestones; small, digestible iterative sessions
   yielding tangible capability; only as much complexity as the problem demands, pausing to
   refine at friction points; make everything configurable and reusable (loose coupling,
   segregated dependencies).
7. **Result: TAU libraries** — Tailored Agentic Units standardize model execution +
   orchestration like `database/sql` standardizes SQL access. Reuse the 3-column dependency
   layout from `README.md:146-200`, reframed as *the reusable result*: dependency-free interfaces
   in `protocol`/`format`/`provider`; vendor SDKs isolated in sub-modules that enter the build
   only when imported.
8. **Result: Distribution** (reuse `release.png`) — tagged releases publish container image +
   standalone migrations binary; the `s2va/herald` proxy mirrors artifacts across security
   domains to IL6 behind a deliberate human gate; Bicep deploys/updates IL6 via managed services
   + managed identity. Managed services chosen for simplicity (no standard prod container
   platform for AKS/on-prem yet).
9. **SHF + SAHRMS** (placeholder — Jaime narrates) — light bullets only: the scenario reiterated
   and refined the architecture and pipeline flows; private s2va libraries built on Herald's core
   package infrastructure; automated GitHub Actions deployments.
10. **Lessons Learned** — this worked *for this problem*; stay flexible to other timelines and
    constraints. Biggest challenges were human, not technical: resisting the urge to build
    something "perfect" or chase out-of-scope work instead of delivering a service that solves
    the target problem; and confidence that the investment would pay off.
11. **Close** — deployed to IL6, running today; **26 documents tested at 100% accuracy**;
    projected ~$27k inference + ~1 month managed services for the full ~750,000; success rooted
    in being on-site and understanding the real problem. Open for discussion.

## Verification

- Preview live: `cd _project/presentation && presenterm retrospective.md` (page through; confirm
  the two reused diagrams render and no slide overflows the terminal).
- Refresh shared diagrams first if stale: `mise run render`.
- Sanity: ~11 slides paces to roughly 10 minutes.
