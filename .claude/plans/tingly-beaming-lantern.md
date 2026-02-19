# Herald Concept Development Plan

## Context

Herald is a new project to classify ~1,000,000 DoD PDF documents' security markings using Azure AI Foundry GPT models. The project needs a Go web service with an embedded Lit web client, hosted in Azure (IL4/IL6). This concept session establishes the project vision, architecture, phases, and bootstraps the repository and project board infrastructure.

The design draws from three prior efforts:
- **agent-lab** (`~/code/agent-lab`) — architectural patterns (LCA, module routing, handler pattern, database patterns)
- **classify-docs** (`~/code/go-agents/tools/classify-docs`) — sequential page-by-page classification with context accumulation (96.3% accuracy)
- **document-context** (`~/code/document-context`) — PDF processing, rendering, and encoding library

## Concept Strategy

### Vision

Herald eliminates the bottleneck of manually reading and recording security markings across millions of documents by applying vision-capable LLMs to interpret markings at scale, producing structured classification records that humans validate and downstream systems consume.

### Architecture Summary

| Component | Choice |
|-----------|--------|
| Backend | Go web service (native net/http), adapted from agent-lab patterns |
| Frontend | Lit 3.x SPA, embedded via go:embed, built with Bun + Vite |
| Database | Azure PostgreSQL (pgx driver, golang-migrate) |
| Blob Storage | Azure Blob Storage, flat layout (`documents/{id}/{filename}.pdf`), DB-driven status |
| AI Service | Azure AI Foundry — GPT-5-mini and GPT-5.2 (both confirmed available on IL6), single configurable agent per deployment |
| Auth | Azure Entra with OBO (Phase 4) |
| Libraries | go-agents, go-agents-orchestration, document-context |

### Classification Workflow

Simplified 3-node state graph using go-agents-orchestration:

```
init → [conditional: needs_enhancement?] → enhance → classify
init → classify (no enhancement needed)
```

- **init**: Open PDF, extract pages, render to images (parallel), quality assessment
- **enhance**: Re-render low-quality pages with adjusted settings (conditional, trigger TBD through experimentation)
- **classify**: Sequential page-by-page classification with context accumulation (classify-docs pattern)

### Key Decisions

1. **Simplified state graph** (3 nodes) instead of agent-lab's 5-node graph — reduces LLM round-trips per page from 3 to 1
2. **Azure PostgreSQL** — maximizes code reuse from agent-lab
3. **Flat blob storage** with DB-driven status — avoids expensive move operations on 1M documents
4. **Batch API endpoint** for bulk upload (multipart with metadata)
5. **API-triggered batch classification** — service manages processing queue internally
6. **Database-stored prompt modifications** with CRUD API — simpler than agent-lab's full Profile system
7. **No observer infrastructure** — classification results embed workflow context metadata
8. **1:1 document-to-classification** — re-classification overwrites previous result
9. **Ephemeral images** — rendered, encoded as data URI, discarded after inference (no image caching)
10. **Full management web client MVP** — upload, browse, classify, validate, monitor, manage prompts

### Phases

| Phase | Focus Area | Version |
|-------|-----------|---------|
| Phase 1 — Service Foundation | Go scaffolding, PostgreSQL schema, Blob Storage, config, routing, document domain (upload + batch + registration) | v0.1.0 |
| Phase 2 — Classification Engine | Agent config, workflow (init → enhance? → classify), prompt modifications, single + batch classification endpoints | v0.2.0 |
| Phase 3 — Web Client | Lit 3.x SPA, document management, result viewing/validation, PDF viewer, monitoring, prompt editor, batch controls | v0.3.0 |
| Phase 4 — Security & Deployment | Azure Entra auth + OBO, Key Vault token management, Docker, Azure deployment, IL4/IL6 config | v0.4.0 |

### Open Questions (not blocking, deferred to phase planning)

1. External system identifier fields for document metadata (Phase 1)
2. Enhance stage trigger conditions (Phase 2 experimentation)
3. Worker pool sizing for batch processing (Phase 2)
4. GPT-5-mini vs GPT-5.2 benchmarking — both confirmed on IL6 Azure AI Foundry (Phase 2)
5. Azure deployment target: Container Apps vs AKS vs App Service (Phase 4)
6. Bulk ingestion strategy for initial 1M document load (Phase 1)
7. Categorical vs numeric confidence scoring (Phase 2)

## Execution Steps

### 1. Initialize Git Repository

```bash
cd ~/code/herald
git init
```

### 2. Create GitHub Repository

```bash
gh repo create JaimeStill/herald --public --description "One who reads and announces markings." --source .
```

### 3. Bootstrap Standard Labels

Apply the tau label convention to the repository using `gh label create`.

### 4. Create GitHub Project Board

Create a GitHub Projects v2 board for Herald and link the repository.

### 5. Configure Project Board Phases

Add Phase field with options:
- Phase 1 — Service Foundation
- Phase 2 — Classification Engine
- Phase 3 — Web Client
- Phase 4 — Security & Deployment

### 6. Create Milestones

Create milestones on the repository matching each phase:
- v0.1.0 — Service Foundation
- v0.2.0 — Classification Engine
- v0.3.0 — Web Client
- v0.4.0 — Security & Deployment

### 7. Create Project Directory and Concept Document

```bash
mkdir -p _project
```

Write `_project/README.md` with the full concept document (Vision, Core Premise, Phases, Architecture, Key Decisions, Dependencies, Integration Points, Open Questions).

### 8. Initial Commit

Stage all files and create the initial commit.

## Critical Files

| Source | Purpose |
|--------|---------|
| `~/code/agent-lab/internal/api/api.go` | API module assembly pattern (Runtime, Domain, routes) |
| `~/code/agent-lab/internal/infrastructure/infrastructure.go` | Infrastructure initialization pattern |
| `~/code/agent-lab/cmd/server/server.go` | Server composition and lifecycle pattern |
| `~/code/agent-lab/workflows/classify/classify.go` | State graph assembly pattern |
| `~/code/go-agents/tools/classify-docs/pkg/classify/document.go` | Sequential classification with context accumulation |
| `~/code/go-agents/tools/classify-docs/pkg/processing/sequential.go` | ProcessWithContext generic pattern |
| `~/code/document-context/pkg/document/document.go` | Document/Page interfaces |
| `~/code/document-context/pkg/encoding/image.go` | Data URI encoding for Vision API |

## Verification

1. GitHub repository exists at `https://github.com/JaimeStill/herald`
2. Project board created with Phase field and 4 phase options
3. 4 milestones created on the repository (v0.1.0 through v0.4.0)
4. Standard labels bootstrapped on the repository
5. `_project/README.md` exists with the complete concept document
6. Initial commit pushed to the repository
