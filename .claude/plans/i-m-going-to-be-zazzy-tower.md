# Offsite Briefing — Architecture & Philosophy (10 min)

## Context

Herald has been deployed to the organization's IL6 tenant. As a result, the developer
has been asked to give a **10-minute briefing** at an organizational offsite focused on
solidifying cloud architecture strategy. Unlike last week's 30-minute capability briefing
(`_project/presentation/README.md`, which *describes Herald*), this deck must articulate the
**architecture and philosophy** behind Herald and its underlying ecosystem, and **what is
valuable for the organization moving forward**.

The deck is a supplemental presentation authored at `_project/presentation/offsite.md`, built
with presenterm, sharing the existing theme, `assets/`, and `mise.toml` render pipeline.

**Audience:** the IT support element of a military organization. **Voicing:** clear, concise,
professional, objective. No sensational, aggrandized, or self-promoting language — the technical
details carry the argument.

**The core message (confirmed with the developer):**
- Three pillars build toward one climactic ask. The pillars — *minimal footprint*, *deployable
  by design*, *from a service to a standard* — are each critical to building **long-lived,
  authoritative enterprise capabilities**.
- The **climax** is **authoritative data ownership + service interoperability**: data domains
  owned and served by their authoritative sources, services interoperable through narrow
  contracts, no domain duplication. This is the central organizational roadblock to delivering
  higher-level intelligence priorities.
- A deliberate caveat, stated at the open and reinforced at the close: **this discipline is not
  one-size-fits-all.** It suits long-lived authoritative capabilities; immediate operational
  needs have different timelines and requirements, and the architecture should flex to them.
  Matching the level of investment to the capability is itself the engineering judgment.

## Approach

Author a single presenterm deck at `_project/presentation/offsite.md` (~7 content slides for a
10-minute slot, ~1–1.5 min each). Reuse the existing front-matter theme block verbatim (only
`title`/`sub_title`/footer `center` change). Match the existing deck's declarative, **bold-highlight**
prose style and its three-column dependency-table convention.

### Front matter

Copy the `theme:` block from `README.md:5-21` exactly (terminal-dark + kanagawa pin + footer
template). Change only:
- `title:` → working title **"Durable by Design"** (adjustable during execution)
- `sub_title:` → "Architecture and philosophy for long-term enterprise services"
- `author:` → "Jaime Still"
- footer `center:` → the deck title

### Slide-by-slide

**1 — Title.** Intro slide (front matter renders it).

**2 — The question (framing + scope + flexibility caveat).** Prose. The question driving the
architecture: what does software that *stays relevant* look like when you build against **open
standards** and refuse the shortcuts that come back to bite you? State the caveat up front —
this discipline targets long-lived authoritative capabilities, not every operational need; avoid
a one-size-fits-all mandate. Close with a short scope list: three pillars → one ask.

**3 — A minimal footprint.** Adapt the verbatim prose from `README.md:133-142` ("A small,
isolated footprint") and reuse the three-column dependency table (`README.md:146-200`):
Core Libraries / Herald Service / Herald Client. Anchor on the numbers — `protocol`, `format`,
`provider` roots carry **0 third-party deps**; `agent`/`orchestrate` carry **1** (`google/uuid`);
vendor SDKs are **opt-in sub-modules** (Herald pulls Azure because it imports `provider/azure`;
AWS SDK never enters its tree). Formats implement **published wire specs** (OpenAI Chat,
Bedrock Converse), not vendor SDKs. *Why it matters for IT:* smaller vulnerability and
supply-chain surface, less to patch, vendor-swappable, longer relevance. Name the **tau**
ecosystem concretely (it is the technical evidence).

**4 — Deployable by design.** Prose + reuse `assets/release.png` (the IL6 CDS distribution
pipeline — already rendered). Points: **infrastructure as code** to provision and maintain
infra is the non-negotiable; managed services are *today's* only reliable IaC path given no
standardized Kubernetes or modern on-prem container infrastructure yet — **a pragmatic current
choice, not advocacy and not lock-in**. The unit of delivery is the **container + IaC**, so the
target (cloud or on-prem) can change. What a developer actually needs: deployment that is
**simple, reliable, secure, and repeatable**, free of artificial roadblocks and bureaucracy.
`release.png` grounds "secure + repeatable" in the real cross-domain release to IL6.

**5 — From a service to a standard.** Prose + reuse `assets/overview.png` (the layered
composition architecture). This addresses the follow-on engagement points the developer
explicitly wants surfaced, kept **abstract** (do not name the external team / engagement
specifics): Herald's `pkg/` infrastructure (auth, lifecycle, database, web, middleware, module)
proved out the layered patterns; in a follow-on engagement those were **extracted into
standardized, provider/driver-neutral internal libraries** (common web infrastructure that lives
in Herald's `pkg/` today became reusable building blocks), and the **CI/CD flow for release and
deployment was standardized** into one repeatable rhythm (versioned per-module releases,
container image per binary, IaC-driven deploy). *Message:* patterns become shared infrastructure;
reuse and a common release rhythm compound, and consistency lowers maintenance and risk across
services.

**6 — Interoperability & authoritative data ownership (CLIMAX).** Prose, optionally anchored by
one new diagram (see Visuals). The same engagement demonstrated the organizational thesis on
three principles:
1. **Interoperability through narrow contracts** — a consumer depends only on a minimal,
   standards-based contract (e.g., a modified-since timestamp + REST GET), never on the
   upstream's platform or implementation.
2. **Authoritative data ownership** — data is owned and served by its authoritative source;
   consumers replicate or reference it **read-only**, never mutate it. No split-brain, no
   reconciliation cost.
3. **No domain duplication** — services **reference** authoritative records rather than copying
   them; the join composes what each domain needs.
Then the ask: this is **the central roadblock** to higher-level intelligence priorities — without
authoritative, interoperable data services, the organization duplicates, drifts, and cannot
compose capabilities. Open this as the discussion point.

**7 — What this means for us (close).** Synthesis + caveat + discussion seeds. Build long-lived
authoritative capabilities against open standards, with minimal dependency surface, IaC-driven
deployment, and standardized internal libraries — **but match the discipline to the capability's
timeline and requirements**. The unlock for the organization: authoritative, interoperable data
services owned by their sources. End with 2–3 discussion bullets (e.g., a standardized
container/deploy path and K8s/on-prem roadmap; who owns which data domains; adoption of the
standardized libraries + CI/CD rhythm), then "Open for discussion."

### Voicing rules

- Declarative, technical, matter-of-fact; **bold** for key concepts, `inline code` for
  identifiers. No marketing verbs, no superlatives, no self-promotion.
- First person is measured and architectural ("the question driving this…", "I targeted managed
  services because…"), never boastful.
- Every claim is one the source code already supports (dependency counts, opt-in SDK sub-modules,
  the IL6 release gate, the `pkg/` → standardized-library extraction).

## Visuals

**Committed (reuse, already rendered — no authoring):**
- `assets/overview.png` → slide 5 (layered composition).
- `assets/release.png` → slide 4 (IL6 distribution / repeatability).
- The three-column dependency table (text) → slide 3.

**Recommended optional (one new diagram), decide during execution:** an **authoritative data
ownership** diagram for slide 6 — there is no existing visual for the climax, so this is the
highest-value new asset. Concept: an authoritative source (owned domain) → *narrow contract*
edge → a consumer holding a **read-only synced copy** plus an **internal owned domain** whose
record **references** the synced record (foreign key, not a copy). If authored, add
`assets/data-ownership.typ` following `assets/design/theme.typ` + `assets/design/tokens.typ` and
the locked palette (actors/sources = cyan, consumer service = violet, neutral borders for
references); `mise run render` rasterizes all `assets/*.typ` automatically, so no `mise.toml`
change is needed. If skipped, a `column_layout` (owned vs synced) conveys the same idea in text.

**Not pursued:** a dependency-isolation diagram — the existing three-column table already conveys
this crisply; a diagram adds little.

## Files

- **Create:** `/home/jaime/code/herald/_project/presentation/offsite.md` (the deck).
- **Create (optional):** `/home/jaime/code/herald/_project/presentation/assets/data-ownership.typ`
  (only if the climax diagram is pursued).
- **No change required** to `mise.toml` for rendering. Note: `export`/`export:pdf` are hardcoded
  to `README.*`; to export the offsite deck, run presenterm directly (below) or add parallel
  `export:offsite` tasks if the developer wants them committed.

## Verification

1. **Render** (regenerates all diagram PNGs, including any new `.typ`):
   ```bash
   cd _project/presentation && mise run render
   ```
2. **Present live** and walk every slide; confirm column layouts and reused images render and
   that the deck fits a 10-minute delivery (~7 content slides):
   ```bash
   presenterm _project/presentation/offsite.md
   ```
3. **Export for review/sharing** (tasks are README-bound, so target offsite explicitly):
   ```bash
   cd _project/presentation && presenterm --export-html --output offsite.html offsite.md
   ```
4. Read through against the voicing rules: no sensational/self-promoting language; every claim
   traceable to the source repos; the flexibility caveat present at open and close; the deck
   builds to the data-ownership climax.
