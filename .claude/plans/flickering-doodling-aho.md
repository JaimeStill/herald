# Phase Planning: Phase 1 Closeout → Phase 2 Objectives

## Context

Phase 1 (Service Foundation, v0.1.0) is functionally complete — all 3 objectives and their sub-issues are closed/merged. Issue #3 remains technically open on GitHub but has no remaining work. Seven `v0.1.0-dev.*` pre-releases exist and need consolidation into a final `v0.1.0` release. Phase 2 (Classification Engine, v0.2.0) is next on the roadmap.

---

## Part 1: Phase 1 Transition Closeout & v0.1.0 Release

### Step 1: Close Issue #3

```bash
gh issue close 3 --repo JaimeStill/herald
```

### Step 2: Consolidate CHANGELOG.md

Replace all 8 dev entries with a single `v0.1.0` section:

```markdown
## v0.1.0

- Establish Go project scaffolding with mise-based build tooling, Docker Compose local infrastructure (PostgreSQL + Azurite), and configuration system with TOML base, environment overlays, and env var overrides
- Add lifecycle coordinator with cold start → hot start → graceful shutdown, HTTP middleware (CORS, logging), and module-based routing
- Add database toolkit: PostgreSQL connection management, composable SQL query builder with projection mapping, generic repository helpers, domain-agnostic error mapping, and pagination types
- Add storage abstraction with Azure Blob Storage implementation, streaming blob operations, and lifecycle-coordinated container initialization
- Add infrastructure assembly, API module, and server entry point with health/readiness probes and graceful shutdown lifecycle
- Add migration CLI with embedded SQL and initial documents schema
- Add document domain: types, mapping, repository, and system interface with blob+DB atomicity, paginated filtered queries, HTTP handlers (List, Find, Search, Upload, Delete), multipart upload with pdfcpu page count extraction, and configurable max upload size
- Add read-only blob storage query endpoints (list, find, download) with marker-based pagination and wildcard path parameter routing
- Add API Cartographer — markdown-based API documentation with executable curl examples
```

### Step 3: Commit, tag, and push

```bash
git add .
git commit -m "Consolidate v0.1.0-dev entries into v0.1.0 release"
git tag v0.1.0
git push && git push --tags
```

Pushing the tag auto-generates the release with notes from CHANGELOG.md.

### Step 4: Delete dev pre-releases and tags

7 dev releases to delete (note: `v0.1.0-dev.1.4` has a CHANGELOG entry but no release/tag):

```bash
# Delete releases
for tag in v0.1.0-dev.1.5 v0.1.0-dev.1.6 v0.1.0-dev.1.7 v0.1.0-dev.2.13 v0.1.0-dev.3.16 v0.1.0-dev.3.17 v0.1.0-dev.3.21; do
  gh release delete "$tag" --repo JaimeStill/herald --yes
done

# Delete remote tags
git push origin --delete v0.1.0-dev.1.5 v0.1.0-dev.1.6 v0.1.0-dev.1.7 v0.1.0-dev.2.13 v0.1.0-dev.3.16 v0.1.0-dev.3.17 v0.1.0-dev.3.21

# Delete local tags
git tag -d v0.1.0-dev.1.5 v0.1.0-dev.1.6 v0.1.0-dev.1.7 v0.1.0-dev.2.13 v0.1.0-dev.3.16 v0.1.0-dev.3.17 v0.1.0-dev.3.21
```

### Step 5: Close v0.1.0 milestone

```bash
MILESTONE_NUMBER=$(gh api repos/JaimeStill/herald/milestones --jq '.[] | select(.title == "v0.1.0 - Service Foundation") | .number')
gh api -X PATCH "repos/JaimeStill/herald/milestones/$MILESTONE_NUMBER" -f state=closed
```

### Step 6: Clean slate

- Delete `_project/phase.md`
- Delete `_project/objective.md`

---

## Part 2: Phase 2 — Classification Engine (v0.2.0)

### Phase Scope

Build the classification engine that reads security markings from PDF documents using Azure AI Foundry GPT vision models. Adapts the sequential page-by-page context accumulation pattern from `classify-docs` (96.3% accuracy) into a 3-node state graph hosted in Herald's web service. Two new domains (prompts, classifications) provide persistence and API access.

### Key Architectural Decisions (resolved)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Workflow topology | 3-node (init → classify → enhance?) | Init prepares images, classify runs sequential page-by-page, enhance conditionally runs as a final stage when confidence is not HIGH and image adjustments could improve clarity |
| Classification approach | Sequential page-by-page with context accumulation | Proven at 96.3% in classify-docs; each page updates running state |
| Confidence scoring | Categorical (HIGH/MEDIUM/LOW) | Aligns with classify-docs; more interpretable for human validators |
| Agent placement | Field on `Infrastructure` | Agent is stateless (no lifecycle hooks); workflow package needs it outside API module |
| Workflow registry | None — direct `Execute()` function | Single workflow, no factory pattern needed |
| Observer/checkpoint | None — noop observer | Results self-contain provenance via flattened metadata columns |
| No batch endpoint | Clients orchestrate parallel single-document classifications | Same pattern as document uploads — deterministic per-document behavior, client coordinates via `Promise.allSettled` |
| Metadata storage | Flattened columns on `classifications` table | No JSONB; direct columns for all metadata fields |
| Table naming | `prompts` (not `prompt_modifications`) | Only prompt-related table; no need for overly specific name |

### Workflow Topology Detail

```
init ──→ classify ──→ [confidence != HIGH && image quality factor?] ──→ enhance ──→ exit
                              │
                              v (confidence == HIGH or no quality improvement possible)
                             exit
```

- **init**: Purely image preparation — download PDF from blob storage, open via document-context, render all pages to base64 data URIs
- **classify**: Sequential page-by-page classification with context accumulation. Returns classification, confidence, and whether image quality was a limiting factor
- **enhance** (conditional final stage): Triggered when classify reports confidence != HIGH AND image adjustments could improve visibility. Re-renders affected pages with adjusted ImageMagick settings AND performs its own classification assessment on the enhanced images to produce the final result. Does not loop back to classify — enhance is the terminal node when triggered, even if the result is still not HIGH confidence.

This separates concerns cleanly: init handles preparation, classify handles initial analysis, enhance handles remediation and final reassessment when needed.

### Deferred to objective planning

- Image rendering parameters (DPI, format — start with document-context defaults)
- Classification prompt template text (adapt from classify-docs)
- Exact flattened metadata columns for `classifications` table (define when workflow output shape is concrete)
- Enhance trigger heuristics (classify node must report whether image quality was a factor)

### Objectives

#### Objective 1: Agent Configuration and Database Schema

**What**: Extend configuration for a single externally-configured agent, add `go-agents`/`go-agents-orchestration`/`document-context` dependencies, create migrations for `classifications` and `prompts` tables.

**Scope**:
- `[agent]` section in `config.toml` with model, provider config, token via `HERALD_AGENT_TOKEN`
- Agent config struct in `internal/config/` following three-phase finalize pattern
- `go get` for go-agents, go-agents-orchestration, document-context
- Agent creation at startup, stored on `Infrastructure`
- Migration `000002_classification_engine` with `classifications` (flattened metadata columns) and `prompts` tables

**Key files**: `internal/config/config.go`, `internal/infrastructure/infrastructure.go`, `cmd/migrate/migrations/`, `config.toml`, `go.mod`

**Dependencies**: None

---

#### Objective 2: Prompts Domain

**What**: Full CRUD domain for named prompt overrides, following the documents domain pattern.

**Scope**:
- `internal/prompts/` — types, mapping, errors, repository, system interface
- Stage validation (init/classify/enhance)
- Handler with routes: `GET /prompts`, `GET /prompts/{id}`, `POST /prompts`, `PUT /prompts/{id}`, `DELETE /prompts/{id}`, `POST /prompts/search`
- Wire into `api.Domain` and route registration
- API Cartographer docs at `_project/api/prompts.md`
- Tests in `tests/prompts/`

**Key files**: `internal/documents/` (reference pattern), `internal/api/domain.go`, `internal/api/routes.go`

**Dependencies**: Objective 1 (schema)

---

#### Objective 3: Classification Workflow

**What**: Implement the `workflow/` package — the 3-node state graph with all node implementations, prompt generation, and response parsing.

**Scope**:
- `workflow/types.go` — PageImage, ClassificationState (with quality factor flag), WorkflowResult
- `workflow/workflow.go` — StateGraph assembly (init → classify → enhance?)
- `workflow/init.go` — Download PDF from blob storage, open via document-context, render pages to base64 data URIs
- `workflow/classify.go` — Sequential page-by-page processing with context accumulation (adapted from classify-docs' `ProcessWithContext`). Reports whether image quality limited confidence.
- `workflow/enhance.go` — Conditional final stage: re-renders affected pages with adjusted ImageMagick settings and performs its own classification reassessment on the enhanced images to produce the final result (no loop back to classify)
- `workflow/prompts.go` — Template-based prompt generation with hard-coded defaults + optional prompt overrides from prompts system
- `workflow/parse.go` — JSON response parsing with markdown code fence fallback
- Top-level `Execute(ctx, runtime, documentID) → WorkflowResult` function

**Key references**:
- `~/code/go-agents/tools/classify-docs/pkg/classify/` — ProcessWithContext pattern, prompt templates, JSON parsing
- `~/code/agent-lab/workflows/classify/` — State graph topology, node patterns, data URI encoding

**Dependencies**: Objective 1 (agent + dependencies), Objective 2 (prompts system for loading overrides)

---

#### Objective 4: Classifications Domain

**What**: Persistence layer wrapping the workflow — stores, queries, validates, and adjusts classification results. Manages document status transitions.

**Scope**:
- `internal/classifications/` — types (Classification, ClassifyCommand, ValidateCommand, AdjustCommand), mapping, errors, repository, system interface
- `Classify` method: calls `workflow.Execute()`, stores result with flattened metadata columns, transitions document status `pending` → `review`
- `Validate` method: marks validated, transitions document status `review` → `complete`
- `Adjust` method: records adjusted classification + rationale
- Upsert semantics (re-classification overwrites, resets validation fields)
- Tests in `tests/classifications/`

**Key files**: `internal/documents/` (reference pattern), `workflow/` (Objective 3 output)

**Dependencies**: Objective 1 (schema), Objective 3 (workflow)

---

#### Objective 5: Classification HTTP Endpoints

**What**: Wire classifications domain into API with HTTP endpoints for single-document classification, result queries, validation, and adjustment.

**Scope**:
- Classifications handler with routes:
  - `GET /classifications` — paginated list with filters
  - `GET /classifications/{id}` — find by ID
  - `GET /classifications/document/{documentId}` — find by document
  - `POST /classifications/{documentId}` — classify single document
  - `POST /classifications/{id}/validate` — mark validated
  - `POST /classifications/{id}/adjust` — adjust classification
  - `POST /classifications/search` — search with JSON body
  - `DELETE /classifications/{id}` — delete classification
- No batch endpoint — clients orchestrate parallel single-document classifications
- Workflow runtime assembly in `api.Domain`
- Wire into `api.Domain` and route registration
- API Cartographer docs at `_project/api/classifications.md`
- Tests

**Dependencies**: Objective 4 (classifications domain), Objective 2 (prompts already wired)

### Dependency Graph

```
Objective 1: Agent Config + DB Schema
    │
    ├──→ Objective 2: Prompts Domain
    │        │
    │        ├──→ Objective 3: Classification Workflow
    │                 │
    │                 ├──→ Objective 4: Classifications Domain
    │                          │
    │                          └──→ Objective 5: Classification HTTP Endpoints
```

Linear chain: 1 → 2 → 3 → 4 → 5

---

## Execution Plan

1. **Transition closeout** (Steps 1–6 from Part 1)
2. **Create Phase 2 field option** on project board (if needed)
3. **Create all 5 Objective issues** on GitHub with `objective` label and `v0.2.0 - Classification Engine` milestone
4. **Add objectives to project board** and assign to Phase 2
5. **Update `_project/README.md`** — revise workflow topology description (init → classify → enhance?) and remove batch classification endpoint, update `prompt_modifications` → `prompts`
6. **Write `_project/phase.md`** with phase scope, objectives table, and constraints
