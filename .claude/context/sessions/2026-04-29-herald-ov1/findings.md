# Findings — Herald OV-1 leadership briefing

Produced by the technical-writer subagent (run via general-purpose) on 2026-04-29.

## Purpose

Herald is a service that reads security classification markings on Department of Defense PDF documents and turns those visual markings into structured records that downstream systems can rely on. It exists to remove a manual, page-by-page bottleneck for organisations holding millions of classified documents, replacing the human-eyes step with vision-capable AI under human validation. Herald sits between the systems that own the documents, the AI service that interprets the markings, and the secure environment in which the answers are used.

## External integrations

- **Azure AI Foundry** — the AI service that examines page images and reports the markings it sees. Every classification depends on it.
- **Azure Blob Storage** — durable home for every uploaded document; Herald never modifies a document once stored.
- **Azure PostgreSQL** — system of record for document registration, classification results, and human-validation history.
- **Azure Entra ID** — identity used by both the people signing in and the service itself when reaching Azure resources.
- **External document sources** — upstream systems that hand documents to Herald via its upload API; Herald does not pull from them.
- **GitHub Container Registry** — public-internet distribution point for built Herald images.
- **Cross-domain transfer service** — the carrier that promotes a published image across security boundaries into the secure environment where Herald operates.
- **Secure storage destination** — the landing zone on the secure-environment side of the cross-domain transfer.

## Per-diagram findings

### readme — Architecture + external integrations

**Entities**
- *Herald service* (system) — focal capability, central, visually heaviest.
- *Upload client / external document source* (actor) — origin of all documents.
- *Reviewer / validator* (actor) — confirms or adjusts a classification.
- *Azure AI Foundry* (external service) — vision-capable AI.
- *Azure Blob Storage* (external service) — durable document store.
- *Azure PostgreSQL* (external service) — registration, classification, prompt records.
- *Azure Entra ID* (external service) — identity for users and service-to-service calls.

**Relationships**
- Document source → Herald: *upload* (handoff, inbound).
- Herald → Azure Blob Storage: *store document bytes* (write, outbound).
- Herald → Azure PostgreSQL: *register, query, persist results* (read/write, outbound).
- Herald → Azure AI Foundry: *page image → markings* (request/response, outbound).
- Reviewer → Herald: *validate or adjust* (inbound, human).
- Azure Entra ID → Herald: *sign-in tokens* (inbound, identity).
- Herald → Azure Entra ID: *managed-identity tokens for Azure resources* (outbound, identity).

**Layout intent.** Hub-and-spoke. Herald centred and visually heaviest; external services arrange around it; the document-source actor enters from one edge and the reviewer enters from another, so the human flow reads as a U-shape across Herald.

**Visual approach.** *Compose fresh.* This is a context-positioning diagram, not a flow — none of the three flow patterns fits.

**Prose draft.** Herald is a Go web service that reads security classification markings on uploaded PDFs and turns them into records that humans confirm. It leans on Azure AI Foundry to interpret each page, Azure Blob Storage to hold the documents, and Azure PostgreSQL to keep the classification record — the picture above shows where Herald sits in that broader system.

**Source citations.** `_project/README.md:1-22`, `_project/README.md:107-118`, `_project/README.md:294-323`, `internal/infrastructure/infrastructure.go:42-52`.

### upload — How a document enters Herald

**Entities**
- *Upload client* (actor) — caller (human or upstream system).
- *Herald service* (system) — receives the upload, orchestrates registration.
- *Azure Blob Storage* (external service) — receives the document bytes.
- *Azure PostgreSQL* (external service) — receives the registration record.
- *Document* (artifact, optional named element) — the thing flowing through.

**Relationships**
- Upload client → Herald: *send document* (inbound handoff, synchronous).
- Herald → Azure Blob Storage: *store bytes immutably* (outbound write).
- Herald → Azure PostgreSQL: *register document with metadata* (outbound write).
- Herald → Upload client: *return registered identifier* (outbound response).

**Layout intent.** Left-to-right journey. Client on the left, Herald centred, the two stores on the right; the response arrow returns underneath.

**Visual approach.** *Step-and-actor flow.* The actor handoff *is* the concept.

**Prose draft.** A document enters Herald through a single upload step. Herald saves the file in durable storage and registers a record for it in the database, so every later step has a stable identity to attach results to. Documents are immutable once stored; nothing else writes them.

**Source citations.** `internal/documents/handler.go:120-185`, `internal/documents/repository.go:93-130`, `_project/README.md:198-208`.

### classification — How Herald analyses a document

**Entities**
- *Herald service* (system) — orchestrates the workflow.
- *Init step* (state) — pulls the document from storage and renders one image per page.
- *Classify step* (state) — analyses every page in parallel.
- *Enhance step* (state, conditional) — re-renders flagged pages with adjusted brightness/contrast/saturation and re-analyses them.
- *Finalize step* (state) — synthesises the document-level answer.
- *Azure Blob Storage* (external service) — source for the document bytes.
- *Azure AI Foundry* (external service) — answers per-page and document-level inference.
- *Azure PostgreSQL* (external service) — destination for the classification record.

**Relationships**
- Init → Classify: *unconditional transition* (handoff).
- Classify → Enhance: *conditional, when any page flagged* (branch).
- Classify → Finalize: *conditional, when no page flagged* (branch).
- Enhance → Finalize: *unconditional transition* (rejoin).
- Each step ↔ Azure AI Foundry: *vision or chat call* (outbound, repeated).
- Init ← Azure Blob Storage: *fetch document* (inbound to Herald).
- Finalize → Azure PostgreSQL: *persist classification* (outbound write, post-workflow).

**Layout intent.** Left-to-right state graph with a labelled fork-and-rejoin between Classify and Finalize. External services hover above or below the band.

**Visual approach.** *State-graph card.* The fork-and-rejoin is the focal feature.

**Prose draft.** Herald examines a document one page at a time: it renders each page to an image, asks the AI service what markings it sees, and sometimes re-renders pages with image adjustments to ask again when a first look was inconclusive. A final synthesis step reviews everything together and produces the document-level classification, confidence, and rationale that a human reviewer then validates.

**Source citations.** `internal/workflow/workflow.go:45-98`, `internal/workflow/init.go:15-60`, `internal/workflow/classify.go:32-117`, `internal/workflow/enhance.go:32-149`, `internal/workflow/finalize.go:26-74`, `internal/state/state.go:73-88`, `_project/README.md:144-165`.

### release — How a tagged version becomes a published image

**Entities**
- *Maintainer* (actor) — pushes the version tag.
- *Herald source repository* (system, public) — holds code and workflow.
- *Build step* (state) — compiles Go binary + web client into a container image.
- *Publish step* (state) — pushes image to public registry.
- *Release step* (state) — creates public release notes.
- *GitHub Container Registry* (external service) — destination for the image.
- *Public release page* (external service) — destination for the release notes.

**Relationships**
- Maintainer → Source repository: *push version tag* (handoff trigger).
- Source repository → Build: *trigger* (event).
- Build → Publish: *image artifact* (handoff).
- Publish → GitHub Container Registry: *push image* (outbound write).
- Build → Release: *changelog excerpt* (handoff).
- Release → Public release page: *publish notes* (outbound write).

**Layout intent.** Horizontal pipeline. Tag on the left, terminal artifacts on the right.

**Visual approach.** *Labelled milestone strip.* Pipeline reads "what happens at each stop"; published image is the focal terminal.

**Prose draft.** When a maintainer marks a Herald version with a tag, an automated pipeline builds the service into a container image and publishes it to a public registry, alongside human-readable release notes. The published image is the unit of distribution every downstream environment consumes.

**Source citations.** `.github/workflows/release.yml:1-57`, `Dockerfile:1-24`.

### cds-release — Cross-domain release

**Entities**
- *Maintainer* (actor) — pushes a version tag in the proxy repository.
- *Proxy repository* (system, public — kept as *cross-domain transfer proxy*).
- *GitHub Container Registry* (external service, public) — source of the image.
- *Bundle step* (state) — pulls the image, packages it as a tarball with a checksum.
- *Stage step* (state) — uploads the bundle to a staging blob area.
- *Cross-domain transfer service* (external service) — performs the boundary crossing.
- *Secure storage destination* (external service) — landing zone on the secure side.
- *Operator on the secure side* (actor) — installs the image into the secure registry once delivered (out of scope for the diagram, prose-only).

**Relationships**
- Maintainer → Proxy repository: *push version tag* (handoff trigger).
- Proxy repository → Bundle: *trigger* (event).
- Bundle ← GitHub Container Registry: *pull image* (inbound to bundle step).
- Bundle → Stage: *bundle artifact* (handoff).
- Stage → Secure storage destination *(via cross-domain transfer)*: *upload + transfer request* (outbound write, then async transfer).
- Cross-domain transfer service → Secure storage destination: *delivery* (async, dashed edge).

**Layout intent.** Horizontal pipeline that visibly crosses a boundary roughly two-thirds along; the transfer step is the visually heaviest element. Public-side and secure-side regions are distinct background bands.

**Visual approach.** *Labelled milestone strip,* with the cross-domain transfer rendered as the focal step (heaviest stroke, hue accent).

**Prose draft.** A separate, public-facing release path takes a published Herald image and promotes it across security boundaries into the secure environment where Herald actually runs. The image is bundled, staged, and submitted to a cross-domain transfer that delivers it to a secure-side storage destination; from there, operators on the secure side load it into their own registry.

**Source citations.** `_s2va/herald/.github/workflows/image-release.yaml:1-67`, `_s2va/herald/README.md:1-50`.

## Sensitivity check

- **`readme`** — no sensitive content. Class-named external services only.
- **`upload`** — no sensitive content.
- **`classification`** — no sensitive content. Four state names appear in public `_project/README.md`.
- **`release`** — no sensitive content. Public Herald repo and GHCR are openly linked.
- **`cds-release`** — items the user should review before authoring:
  - `s2va-runners` (`image-release.yaml:14`) → **redact** to "self-hosted runner inside the secure boundary" or omit.
  - `s2va/cds-manifest@main` (`image-release.yaml:60`) → **redact** to "cross-domain transfer service".
  - `CDS_SP_CRED`, `CDS_APP_ID`, `CDS_APP_PRIVATE_KEY`, `SOFTWARE_FLOW_CONTAINER`, `CDS_STORAGE_ACCOUNT` (`image-release.yaml:46-65`) → **never named**; abstract to "cross-domain transfer credentials" / "staging blob area" / "secure storage destination".
  - Azure US Government environment (`image-release.yaml:48`) → **abstract** to "secure environment".
  - The `_s2va` directory path itself signals the org name; refer only to "the proxy repository".
  - **Borderline (user judgement):** GHCR image path `ghcr.io/jaimestill/herald` (`image-release.yaml:10`) is publicly visible from Herald's own README. Could appear as "public registry" in stakeholder prose.

## Open questions — resolved 2026-04-29

1. **Reviewer actor in `readme`.** **YES — include reviewer.** Reviewer enters from one edge, document source from the other; U-shape across Herald. Five spokes total: 3 external services + 1 source actor + 1 reviewer actor. Entra renders as a compact badge alongside Herald (see Q4).
2. **Document source actor naming.** **Default: "Document source"** (clean, codebase's `external_id` / `external_platform` is implementation detail).
3. **Cross-diagram visual cohesion.** **YES — shared visual vocabulary** for `release` + `cds-release`. Same node shape, same edge weight, similar tokens. The pair reads as one continuous distribution story across two pictures (public-side, then secure-side).
4. **Azure Entra ID prominence.** **Compact badge alongside Herald.** Renders as a small "secured by Entra" or identity glyph paired with Herald's focal shape. Not a spoke.
5. **Public registry naming in `cds-release` prose.** **Default: "public registry" abstraction** (symmetric with secure-side abstractions).
6. **`cds-release` diagram boundary.** **Stop at secure storage destination on the IL6 side, after the CDS action is approved.** The cross-domain transfer's approval gate IS part of the diagram (or implied by the focal step). Prose articulates: "we are working with Microsoft engineers to build out the IL6-side GitHub Actions workflow that picks up from the secure storage destination" — the post-delivery story is in-progress; the diagram intentionally stops at the delivery point.
