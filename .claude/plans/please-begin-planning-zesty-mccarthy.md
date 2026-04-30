# B3d — `core/release.typ` (full distribution: tag → IL6 secure storage)

## Context

Fourth diagram in the OV-1 leadership briefing. B3a (`readme.typ`), B3b (`upload.typ`), B3c (`classification.typ`) complete. **Scope revised**: this step authors a SINGLE consolidated `core/release.typ` covering the full CI/CD chain — public release through cross-domain transfer to IL6 secure storage. The previously-planned `core/cds-release.typ` (B3e) is **retired** as a separate diagram; its content is absorbed here.

**Why one diagram, not two.** The public release is genuinely thin (release.yml is one job, two terminal artifacts). Standalone, it would feel like padding next to the architecture/upload/classification diagrams. The full chain — tag in Herald repo → public image → manual proxy mirror → cross-domain transfer → IL6 secure storage — is the load-bearing leadership story. Splitting it across two diagrams loses the causal chain that matters most: *"how does Herald reach the secure environment where it actually runs?"*

Concept (one diagram, one concept): **Herald's full distribution path, from a tag push to the secure environment.** The focal element is the cross-domain boundary crossing — visually the most consequential edge in the entire briefing for a leadership audience.

Source semantics verified against:
- `~/code/herald/.github/workflows/release.yml` — public release (release.yml:3-6 trigger; 41-50 build-and-push; 52-56 create release).
- `~/code/_s2va/herald/.github/workflows/image-release.yaml` — proxy CDS release (image-release.yaml:3-6 trigger; 31-34 pull; 36-41 bundle+checksum; 49-57 stage to gov-cloud blob; 59-66 invoke CDS).
- `~/code/_s2va/herald/README.md:30-34` — proxy tag is **pushed manually** by a maintainer; not auto-triggered from Herald's release. This is the "human gate" — stakeholder-meaningful.

Findings open-question Q6 (resolved): diagram terminates at the **delivery point** (image bundle in IL6 secure storage post-CDS); IL6-side ACR import by an operator is prose-only, not in the diagram.

## Design decisions

**Composition: three security-domain bands, horizontal flow.** Three colored band enclosures — Public / IL4 transit / IL6 — give the diagram its skeleton. The bands are visual anchors for "where each step happens"; gaps between bands implicitly mark security boundaries. Edges crossing the gaps tell the boundary-crossing story.

**Hue palette** (extending established conventions):
- **Orange** — maintainer (actor). Two maintainer nodes, one above each pipeline trigger. Same hue/glyph/role; visual repetition signals "two distinct trigger events, performed by the maintainer in each repo."
- **Purple** — pipelines (Public pipeline, CDS pipeline). Workflow-state hue from B3c.
- **Coral** — public services (GHCR, Release page) and the IL6 destination. Consistent throughout.
- **Green** stroke — forward edges. Consistent throughout.

**Band fills** (signals security domain, not threat):
- Public: `palette.surface-muted` (neutral gray).
- IL4 transit: `palette.yellow.fill` (caution / restricted in-transit).
- IL6: deeper tint — first attempt with `palette.purple.fill`; iterate if it competes with the purple pipeline nodes inside the IL4 band. Fallbacks: `palette.coral.fill` (warmer) or `palette.red.fill` (alarming — likely too strong).

**Focal element: the cross-domain transfer edge.** The CDS-pipeline → IL6-storage edge gets:
- `tokens.stroke-emphasis` (2pt) instead of `stroke-default`.
- A prominent label: `cross-domain transfer`.

The IL6 storage node also gets `stroke-emphasis` outline. The `cross-domain transfer` edge + the sealed IL6 destination together carry the diagram's focal weight.

**Sensitivity boundary applied within one diagram.** The IL4/IL6 portion follows the public-abstractions-only rule from the brief; the public-side portion stays open. Concrete redactions for the IL4/IL6 portion:
- Runner identity (`s2va-runners`) → omitted; pipeline node carries no runner reference.
- Internal action (`s2va/cds-manifest@main`) → rendered as the abstraction `cross-domain transfer service` in prose, never named.
- Storage account / container vars (`CDS_STORAGE_ACCOUNT`, `SOFTWARE_FLOW_CONTAINER`) → rendered as `government cloud blob storage` (class abstraction).
- Service principal / signing key (`CDS_APP_ID`, `CDS_APP_PRIVATE_KEY`) → omitted entirely; CDS authentication is implementation chrome at OV-1 level.
- Image path (`ghcr.io/jaimestill/herald`) → rendered as just `published Herald image` in prose; the GHCR node is class-named only (`GitHub Container Registry`).
- IL4 / IL6 labels — DoD impact-level standards, public terminology. Acceptable to use.

## Layout

Coordinates (Fletcher grid, positive y = down):

```
            [Maintainer-public]                                [Maintainer-il4]
                  │ push tag                                         │ manually mirror tag
                  ▼                                                  ▼
      ┌─ Public ──────────────────────────┐  ┌─ IL4 transit ───────┐  ┌─ IL6 ────────┐
      │  [Public pipeline] ──image──▶ [GHCR] │──pull──▶ [CDS pipeline] │──XDOM──▶ [IL6  │
      │       │                             │                          │   secure      │
      │       └─notes──▶ [Release page]     │                          │   storage]    │
      │                                     │                          │  (FOCAL)      │
      └─────────────────────────────────────┘  └──────────────────────┘  └──────────────┘
```

| Node | Pos | Hue | Glyph | Title | Kind | Stroke |
|---|---|---|---|---|---|---|
| `<m-pub>` | (-3, -1.5) | orange | `\u{F007}` | Maintainer | release author | default |
| `<pub>` | (-3, 0) | purple | `\u{F085}` (cogs) | Public release pipeline | automation | default |
| `<ghcr>` | (-1, 0) | coral | `\u{F49E}` (box-open) | GitHub Container Registry | container registry | default |
| `<release>` | (-3, 1) | coral | `\u{F1EA}` (newspaper) | GitHub Releases page | release notes | default |
| `<m-il4>` | (1, -1.5) | orange | `\u{F007}` | Maintainer | release author | default |
| `<cds>` | (1, 0) | purple | `\u{F085}` (cogs) | Cross-domain release pipeline | automation | default |
| `<il6>` | (3, 0) | coral | `\u{F023}` (lock) | IL6 secure storage | classified destination | **emphasis** |

Glyph rationale:
- Maintainers share `\u{F007}` (user) — actor convention; visual repetition is intentional ("same role, two trigger events").
- Pipelines share `\u{F085}` (cogs) — both are automated GitHub Actions workflows; band membership and descriptions differentiate them.
- GHCR: `\u{F49E}` (box-open) — generic image-as-deliverable; avoids brand glyphs and overlap with `\u{F0A0}` (Blob) / `\u{F1B2}` (ACA).
- Release page: `\u{F1EA}` (newspaper).
- IL6 storage: `\u{F023}` (lock) — signals sealed/classified destination.

## Band enclosures

Three single-node `enclose:` containers, no headers (consistent with B3c pattern):

```typst
// Public band — encloses pipeline, GHCR, release page.
node(enclose: (<pub>, <ghcr>, <release>),
     [], shape: rect, fill: palette.surface-muted, stroke: none,
     inset: tokens.pad-inside-container,
     corner-radius: tokens.radius-container, snap: -1)

// IL4 transit band — encloses CDS pipeline only.
node(enclose: (<cds>,),
     [], shape: rect, fill: palette.yellow.fill, stroke: none,
     inset: tokens.pad-inside-container,
     corner-radius: tokens.radius-container, snap: -1)

// IL6 band — encloses IL6 storage only.
node(enclose: (<il6>,),
     [], shape: rect, fill: palette.purple.fill, stroke: none,
     inset: tokens.pad-inside-container,
     corner-radius: tokens.radius-container, snap: -1)
```

Maintainer nodes float ABOVE the bands (not enclosed). Their tag-push edges cross the public/IL4 band tops downward — visually consistent with "trigger event from outside the band."

## Edge inventory

| # | Edge | Mark | Stroke | Label |
|---|---|---|---|---|
| 1 | `<m-pub.south>` → `<pub.north>` | `->` | solid green default | `push tag (v*)` |
| 2 | `<pub.east>` → `<ghcr.west>` | `->` | solid green default | `publish image` |
| 3 | `<pub.south>` → `<release.north>` | `->` | solid green default | `publish notes from CHANGELOG` |
| 4 | `<ghcr.east>` → `<cds.west>` | `->` | solid green default | `pull image` |
| 5 | `<m-il4.south>` → `<cds.north>` | `->` | solid green default | `manually mirror tag` |
| 6 | `<cds.east>` → `<il6.west>` | `->` | solid green **emphasis** | `cross-domain transfer` |

Total: 6 edges, 7 nodes, 3 band enclosures. Edge #6 is the focal — heavier stroke + key label. Edge #5's label (`manually mirror tag`) is the load-bearing one for the human-gate concept; it differentiates from the automated trigger semantics of edge #1.

All labels use the `step-label` helper. All labelled edges use native `label-fill: palette.surface`.

## Shape descriptions (kinded body)

Stakeholder-voice, 2–3 short lines each:

- **Maintainer-public** — *Tags a Herald version to release.*
- **Public release pipeline** — *Triggered by a version-tag push. Builds the container image and publishes release notes from the CHANGELOG.*
- **GitHub Container Registry** — *Public destination for the tagged Herald image. Bridge between the public release and the cross-domain transfer.*
- **GitHub Releases page** — *Public release page rendered from the CHANGELOG entry for the tagged version.*
- **Maintainer-il4** — *Pushes a matching tag in the proxy repository to initiate the cross-domain transfer. This is a deliberate human gate — the secure-environment promotion does not happen automatically.*
- **Cross-domain release pipeline** — *Pulls the published image, packages it as a checksummed bundle, stages it to government cloud storage, and submits it to the cross-domain transfer service.*
- **IL6 secure storage** — *Landing zone on the secure-environment side after the cross-domain transfer is delivered. Operators on the secure side import the image into the secure environment from here.*

The Maintainer-il4 description carries the stakeholder-meaningful "human gate" framing. The CDS pipeline description packages the four-step proxy workflow into one stakeholder sentence (pull → bundle → stage → submit), with all sensitive specifics abstracted.

## Helper reuse + new addition

Reused verbatim from `classification.typ`:
- `kinded(...)` — copy from `classification.typ:44-65`.
- `step-label(s)` — copy from `classification.typ:76-82`.
- `edge-stroke` (solid green default) — copy from `classification.typ:69`.

**New helper** (introduced in B3d):
```typst
#let edge-stroke-emphasis = tokens.stroke-emphasis + palette.green.stroke
```

Parallels `edge-stroke` and `edge-stroke-dashed`. Used on edge #6 (cross-domain transfer).

**`kinded` extension** — add an optional `stroke-weight: tokens.stroke-default` parameter so the IL6 storage node can take a heavier outline:

```typst
#let kinded(pos, hue, glyph, title, kind, description,
             extras: none, ref: none, stroke-weight: tokens.stroke-default) = node(...)
  ...
  stroke: stroke-weight + hue.stroke,
  ...
)
```

Backwards-compatible (defaults to current `stroke-default`); applies to `<il6>` with `stroke-weight: tokens.stroke-emphasis`. This is a small extension; subsequent diagrams (not in scope) inherit the option.

## Critical files

| Path | Role |
|---|---|
| `/home/jaime/code/herald/_project/ov-1/core/release.typ` | **NEW** — file to author |
| `/home/jaime/code/herald/_project/ov-1/core/classification.typ` | Source of `kinded` (with signature extension), `step-label`, edge-stroke helpers |
| `/home/jaime/code/herald/_project/ov-1/design/tokens.typ` | `tokens.stroke-emphasis` (2pt) — already exists |
| `/home/jaime/code/herald/_project/ov-1/design/theme.typ` | `palette.purple`, `palette.coral`, `palette.orange`, `palette.green`, `palette.yellow`, `palette.surface-muted` |
| `/home/jaime/code/herald/.github/workflows/release.yml` | Source-of-truth for public-side semantics |
| `/home/jaime/code/_s2va/herald/.github/workflows/image-release.yaml` | Source-of-truth for IL4/IL6 chain |
| `/home/jaime/code/_s2va/herald/README.md` | Source-of-truth for the human-gate fact |
| `/home/jaime/code/herald/CHANGELOG.md` | Public release notes source |
| `/home/jaime/code/herald/_project/ov-1/mise.toml` | `mise run render-file` task |

Output renders: `core/release-light.svg` and `core/release-dark.svg`.

## Iteration plan

This diagram is denser than B3a–B3c; expect ~2–3 iterations.

1. **Iteration 1** — author per the layout above; render dual-theme. Surface to user. Likely first-pass adjustments:
   - **Band fill colors** — `palette.purple.fill` for IL6 may compete visually with the purple pipeline nodes inside the IL4 band. Try fallback: `palette.coral.fill` deeper-tinted, or `palette.red.fill` if IL6 needs heavier signal.
   - **Maintainer-il4 placement** — at (1, -1.5) above the IL4 band, the trigger edge is short and clean. If it visually competes with the GHCR→CDS edge running just below, consider raising to (1, -1.8) or shifting horizontally.
   - **Edge label collisions** — six labelled edges in a horizontal flow; check that `publish image` (edge #2) and `pull image` (edge #4) don't collide along the y=0 baseline since they sit close together. Vertical separation comes from label-side asymmetry (one above, one below) if needed.
   - **Stroke emphasis legibility in dark theme** — verify the heavier `cross-domain transfer` edge and the IL6 outline read distinctly in dark mode.
   - **Band tightness** — `enclose:` with a single node (the IL4 band has only `<cds>`) may produce a tight box that doesn't visually balance with the wider Public band. May need additional inset or a phantom invisible node to extend the band horizontally.
2. **Iteration 2** — apply user adjustments; lock layout.
3. **Iteration 3** *(if needed)* — refinement after user critique.

## Verification

- **Sources render.** `mise run render-file core/release.typ` produces `core/release-light.svg` and `core/release-dark.svg` with no compile errors.
- **Both themes render correctly.** Open both SVGs; band fills + node hues + edge emphasis all read clearly in light AND dark.
- **Critique checklist**:
  - One concept (the full distribution path; cross-domain crossing is focal).
  - Shape carries identity (orange actor / purple automation / coral terminal / IL6 with emphasis stroke).
  - Visual weight tracks meaning (CDS edge stroke-emphasis = "this is the consequential boundary crossing").
  - Both themes render correctly.
  - Edge labels carry trigger / output / transfer names; no implementation chrome.
- **Voice holds.** Read shape descriptions and edge labels aloud — no Go type names, no GHA action repository names, no runner identity, no env-var names. Plain stakeholder terms (`tag`, `pipeline`, `image`, `release notes`, `CHANGELOG`, `cross-domain transfer`, `secure environment`).
- **Sensitivity rule held.** No `s2va-runners`, no `s2va/cds-manifest`, no `CDS_*` var names, no `SOFTWARE_FLOW_CONTAINER`, no specific tenant/subscription/account identifiers. The `ghcr.io/jaimestill/herald` path does not appear (GHCR is class-named only). IL4 / IL6 labels appear as standard DoD impact-level terminology.
- **Cross-diagram cohesion.** Hues, glyphs (where reused), edge label voice all match B3a–B3c.
- **Lock confirmation with the user** before transitioning to B4 (assemble README.md).

## README prose framing (handoff to B4)

When B4 assembles `~/code/herald/_project/ov-1/README.md`, the release-section prose must capture the **future-state** context that explains why this diagram terminates at IL6 secure storage:

> *Once GitHub Actions become available on the IL6 GitHub instance, an IL6-side release workflow will be built to complete the automated deployment update. Today, the chain stops at the secure storage destination, where IL6 operators import the image into the secure environment manually.*

This frame is load-bearing for the leadership audience: it tells them (a) why the diagram stops where it does, (b) that the manual IL6-side import is a known gap, not a permanent design, and (c) that automation will close the gap when the IL6 platform supports it. Without this frame, the diagram could read as "that's the whole story" — understating the planned end-to-end automation. The diagram body stays as designed; the future-state framing lives in the surrounding prose.

## Brief revision (post-implementation, before B7)

The brief at `~/tau/herald-overview-brief.md` currently locks **5 diagrams** including a separate `core/cds-release.typ`. After this step lands and B5 (PDF render) succeeds, the brief inventory must be revised to **4 diagrams**:

| File | Subject |
|---|---|
| `core/readme.typ` | Architecture + external integrations |
| `core/upload.typ` | Document upload |
| `core/classification.typ` | Classification workflow |
| `core/release.typ` | **Full distribution: tag → IL6 secure storage** *(merged scope)* |

The phase-04 prelude artifact (B6) should also note that "release pipeline at OV-1 level" and "cross-domain release pipeline at OV-1 level" merge into a single **multi-domain distribution-pipeline** category — the merge is itself a finding to capture for phase-04.

## Out of scope

- Authoring B3e (`core/cds-release.typ`) — **retired** as a separate diagram; its content is absorbed in this step.
- Assembling `~/code/herald/_project/ov-1/README.md` (B4).
- Rendering the briefing PDF (B5).
- Phase-04 prelude artifact (B6) — captures the merge as a finding.
- Brief revision at `~/tau/herald-overview-brief.md` — deferred to post-implementation, before B7.
- Toolkit edits to `~/tau/diagrams/.claude/skills/...` (deferred to post-B7).
- Extracting `kinded` / `step-label` / edge-stroke constants to a shared `_helpers.typ` module (deferred — helper extension here is the last anticipated signature change; module extraction can happen post-briefing).
- Any commit (B7).
