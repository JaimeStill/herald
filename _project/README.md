# Herald

**One who reads and announces markings.**

Herald is a Go web service with an embedded Lit web client for classifying approximately 1,000,000 Department of Defense PDF documents' security markings using Azure AI Foundry GPT models. It reads security classification markings from document pages, interprets them through vision-capable LLMs, and produces structured classification records suitable for database association and downstream system integration.

## Vision

Organizations managing large corpora of classified documents face a persistent bottleneck: manually reading and recording security markings across millions of pages. Herald eliminates this bottleneck by applying vision-capable LLMs to interpret markings at scale, producing structured classification records that humans validate and downstream systems consume. The service is designed for rapid delivery, drawing heavily from proven patterns established during R&D in agent-lab, go-agents, and document-context.

## Core Premise

Herald occupies a deliberate middle ground between two existing implementations:

- **classify-docs** (`go-agents/tools/classify-docs`): A CLI tool using pure sequential page-by-page processing with context accumulation. Accurate (96.3%) but no web interface, no persistence, no batch processing.

- **agent-lab**: A full workflow experimentation platform with 5-node classification graphs, parallel detection, observer infrastructure, profile management, and image caching. Powerful but far more infrastructure than Herald needs.

Herald takes the sequential context accumulation pattern from classify-docs (proven for accuracy), wraps it in a simplified 3-node state graph from go-agents-orchestration (init, classify, conditional enhance), and hosts it in a streamlined Go web service adapted from agent-lab's infrastructure patterns. Documents flow in through the service API to Azure Blob Storage, classification results flow out through PostgreSQL, and humans validate results through an embedded Lit web client.

The architecture deliberately excludes: image caching (images are ephemeral during classification), observer/checkpoint infrastructure (results self-contain context), multi-workflow registries (single classification workflow), provider/agent CRUD (single externally-configured agent), and OpenAPI/Scalar documentation (deferred for velocity).

## Phases

| Phase | Focus Area | Version Target |
|-------|-----------|----------------|
| Phase 1 - Service Foundation | Go project scaffolding, Azure PostgreSQL schema/migrations, Azure Blob Storage integration, configuration system, module/routing infrastructure, document domain (upload single + batch, registration, metadata), storage abstraction, lifecycle coordination | v0.1.0 |
| Phase 2 - Classification Engine | Agent configuration from external config, classification workflow state graph (init -> classify -> enhance?), sequential page-by-page processing with context accumulation, named prompt overrides (DB + API), single document classification endpoint, classification result schema with flattened workflow metadata | v0.2.0 |
| Phase 3 - Web Client | Lit 3.x SPA with Bun + Vite embedded in Go binary, document management UI (upload single + batch, browse, search, filter), classification result viewing/validation/manual adjustment, PDF viewer alongside classification result, processing status and monitoring, prompt modification management, batch processing controls | v0.3.0 |
| Phase 4 - Security and Deployment | Azure Entra authentication (service + OBO for web client), AI Foundry token management (Key Vault or container secrets), Docker containerization with ImageMagick 7.0+, Azure deployment configuration, IL4/IL6 environment configuration | v0.4.0 |

## Architecture

### Project Structure

```
herald/
├── cmd/
│   ├── server/           # HTTP server entry point and composition
│   └── migrate/          # Database migration CLI
├── internal/
│   ├── api/              # API module: Runtime, Domain, route registration
│   ├── config/           # Configuration management (JSON + env overlays)
│   ├── infrastructure/   # Infrastructure assembly (lifecycle, logger, database, storage, agent)
│   ├── documents/        # Document domain (upload, registration, metadata, blob management)
│   ├── classifications/  # Classification result domain (store, query, validate, adjust)
│   └── prompts/          # Named prompt override domain (CRUD, per-stage loading)
├── workflow/             # Classification workflow definition
│   ├── workflow.go       # State graph assembly: init -> classify -> enhance?
│   ├── init.go           # Init node: open PDF, extract pages, render images
│   ├── classify.go       # Classify node: sequential page-by-page context accumulation
│   ├── enhance.go        # Enhance node: conditional final stage, re-render + reassess
│   ├── types.go          # Shared types: PageImage, ClassificationState, QualityAssessment
│   ├── prompts.go        # Template-based prompt generation with running state
│   └── parse.go          # JSON response parsing with markdown code fence fallback
├── pkg/
│   ├── database/         # PostgreSQL connection management (pgx driver)
│   ├── lifecycle/        # Startup/shutdown coordination
│   ├── middleware/       # HTTP middleware (CORS, logging)
│   ├── module/           # Modular HTTP routing (prefix-based modules)
│   ├── pagination/       # PageRequest/PageResult pattern
│   ├── query/            # SQL query builder with projections and sorting
│   ├── repository/       # Database helpers (QueryOne, QueryMany, WithTx)
│   ├── handlers/         # HTTP response utilities (RespondJSON, RespondError)
│   ├── storage/          # Blob storage interface (Azure Blob Storage implementation)
│   ├── routes/           # Route registration types
│   └── web/              # Template set, router, embedded asset serving
├── web/
│   └── app/              # Lit 3.x SPA
│       ├── app.go        # Go module: embed dist/*, shell template, NewModule()
│       ├── client/       # TypeScript source (Bun + Vite build)
│       └── server/       # Go HTML templates (shell.html)
├── config.json           # Base configuration
├── Makefile
├── Dockerfile
└── _project/
    └── README.md         # This document
```

### Layered Composition Architecture

Herald follows the Layered Composition Architecture (LCA) established in agent-lab:

**Cold Start** (initialization, no connections):
1. `config.Load()` -- Read JSON base + environment overlay + env var overrides
2. `infrastructure.New(cfg)` -- Create lifecycle coordinator, logger, database system, storage system, validate agent config
3. `NewModules(infra, cfg)` -- Create API module (runtime + domain assembly) and web app module

**Hot Start** (connections established, ready to serve):
1. `infra.Start()` -- Database connect + pool, storage container initialization
2. `http.Start(lifecycle)` -- Begin listening, register shutdown hook
3. `lifecycle.WaitForStartup()` -- All subsystems report ready

**Shutdown** (coordinated teardown):
1. Signal received (SIGINT/SIGTERM)
2. `lifecycle.Shutdown(timeout)` -- Reverse-order hook execution within deadline

### Infrastructure Layer

```go
type Infrastructure struct {
    Lifecycle *lifecycle.Coordinator
    Logger    *slog.Logger
    Database  database.System
    Storage   storage.System          // Azure Blob Storage implementation
    Agent     gaconfig.AgentConfig    // go-agents config for per-request agent creation
}
```

### API Module Composition

```go
type Runtime struct {
    *infrastructure.Infrastructure
    Pagination pagination.Config
}

type Domain struct {
    Documents       documents.System
    Classifications classifications.System
    Prompts         prompts.System
}
```

### Domain Systems

Each domain follows the handler pattern: a System interface, repository with query builder, Handler struct with `Routes()` method.

**documents** -- Upload (single + batch), registration, metadata, blob lifecycle. Documents are immutable after upload.

**classifications** -- 1:1 relationship with documents. Stores the classification result with flattened workflow metadata columns. Supports manual validation and adjustment.

**prompts** -- Named prompt overrides stored in PostgreSQL with CRUD API endpoints. Each override targets a specific workflow stage. Hard-coded defaults used when no override is loaded.

### Classification Workflow

A simplified 3-node state graph using go-agents-orchestration:

```
init --> classify --> [confidence != HIGH && image quality factor?] --> enhance --> exit
                              |
                              v (confidence == HIGH or no quality improvement possible)
                             exit
```

**init node**: Opens PDF via document-context, extracts pages, renders to images in parallel. Images are encoded as base64 data URIs and held in memory (no caching). Cache parameter is nil: `page.ToImage(renderer, nil)`. This node purely handles image preparation.

**classify node**: Sequential page-by-page classification inspired by classify-docs' `ProcessWithContext[TContext]` pattern. Each page is sent to the vision-capable GPT model with the running classification state as context. The model returns an updated `ClassificationState` that accumulates across pages. Reports whether image quality was a limiting factor in the confidence assessment.

**enhance node** (conditional final stage): Triggered only when classify reports confidence != HIGH AND image adjustments could improve visibility. Re-renders affected pages with adjusted ImageMagick settings (brightness, contrast, saturation) and performs its own classification reassessment on the enhanced images to produce the final result. Does not loop back to classify — enhance is the terminal node when triggered, even if the result is still not HIGH confidence. Trigger conditions TBD through experimentation during Phase 2.

This collapses agent-lab's 5-node graph (init, detect, enhance, classify, score) into 3 nodes by merging detection, classification, and confidence scoring into the classify node, with enhance as an optional remediation stage. This reduces LLM round-trips per page from potentially 3 to 1.

### Agent Configuration

Single agent definition from external configuration (not CRUD-managed). Uses go-agents' `config.AgentConfig` type directly — no Herald-specific agent config structs:

```json
{
  "agent": {
    "name": "herald-classifier",
    "provider": {
      "name": "azure",
      "base_url": "https://...",
      "options": {
        "deployment": "gpt-5-mini",
        "api_version": "2024-12-01-preview",
        "auth_type": "api_key"
      }
    },
    "model": {
      "name": "gpt-5-mini",
      "capabilities": {
        "vision": {
          "max_tokens": 4096,
          "temperature": 0.1
        }
      }
    }
  }
}
```

Configured at startup from JSON config + env var overrides. Token injected via `HERALD_AGENT_TOKEN` env var. Provider, model, and provider options are also overridable via `HERALD_AGENT_*` env vars.

### Storage Architecture

Azure Blob Storage with flat layout:

```
documents/{document-id}/{filename}.pdf
```

Documents are immutable after upload. Processing status is tracked exclusively in the PostgreSQL database. All documents enter through the Go service API (single or batch upload) to ensure database record + blob atomicity.

### Database Schema

Azure PostgreSQL with golang-migrate. Core tables:

**documents** -- Registration records with metadata and blob reference. Status tracks a document's position in the classification lifecycle, not operational concerns. `pending` means inference hasn't completed; `review` means inference produced a result awaiting human validation; `complete` means the classification has been validated or adjusted. Errors during inference leave the document in `pending` — error handling and observability are separate concerns.
- id, external_id, external_platform, filename, content_type, size_bytes, page_count, storage_key, status (pending/review/complete), uploaded_at, updated_at

**classifications** -- 1:1 with documents, overwritten on re-classification. Workflow metadata is stored as flattened columns rather than JSONB.
- id, document_id (unique FK), classification, confidence, markings_found (JSONB), rationale, agent_config, classified_at
- Flattened workflow metadata columns (exact schema TBD during Phase 2 when workflow output shape is concrete)
- validated_by, validated_at, adjusted_classification, adjustment_rationale

**prompts** -- Named overrides per workflow stage.
- id, name (unique), stage (init/classify/enhance), system_prompt, description

### Web Client

Lit 3.x SPA following agent-lab patterns:

- **Build**: Bun + Vite + TypeScript (CI only)
- **Embedding**: `go:embed dist/*`, shell template pattern
- **Routing**: Client-side History API router, component prefix `hd-`
- **Styling**: CSS cascade layers with design tokens
- **State**: Signal-based reactivity via `@lit-labs/signals`
- **Services**: `@lit/context` for dependency injection

Key views: document upload/management, classification results with PDF viewer, batch processing controls, prompt modification editor, processing status dashboard.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Workflow topology | 3-node (init -> classify -> enhance?) | Reduces LLM round-trips from 3 to 1 per page. Detection, classification, and confidence scoring collapsed into single sequential classify pass. Enhance runs as optional final remediation stage when image quality limits confidence. |
| Classification approach | Sequential page-by-page with context accumulation | Validated at 96.3% accuracy in classify-docs. Each page updates running classification state. |
| Image lifecycle | Ephemeral (render, encode, discard) | 1M documents would generate enormous image storage. Images serve only as Vision API input. |
| Classification result model | 1:1 with document, overwritten on re-classification | Simpler than run/stage/decision tracking. Flattened workflow metadata columns preserve provenance. |
| Agent configuration | Single externally-configured agent | Herald serves one purpose with one or two GPT models. External config matches deployment patterns. |
| Prompt management | Hard-coded defaults + named DB overrides | Lighter than agent-lab's profile + profile_stages system. |
| Blob storage layout | Flat with DB-driven status | Avoids expensive blob-move operations across 1M documents. DB is single source of truth. |
| Database | Azure PostgreSQL | Maximizes code reuse from agent-lab (pgx, query builder, repository patterns). |
| Observability | No observer; results self-contain workflow context | Observer infrastructure adds complexity. Single workflow with deterministic topology doesn't benefit from generic execution tracking. |
| Batch classification | Client-orchestrated parallel single-document classifications | Same pattern as document uploads — deterministic per-document behavior. Clients coordinate via `Promise.allSettled`. |
| Bulk upload | Sequential single-file uploads (no batch endpoint) | `ParseMultipartForm(maxMemory)` caps total request memory, making a batch endpoint's per-file size limit unpredictable. Single-file uploads give deterministic per-file size limits. The web client coordinates multi-file uploads via `<input multiple>` with `Promise.allSettled`, providing per-file progress, retry, and error handling. |
| Web client scope | Full management MVP | Upload, browse, classify, validate, monitor, manage prompts. Complete operational interface. |

## Dependencies

### Go Libraries (ecosystem)

- **go-agents**: LLM abstraction. Agent creation, Vision API calls, response parsing.
- **go-agents-orchestration**: State graph workflow engine. StateGraph, StateNode, conditional edges.
- **document-context**: PDF processing. Document/Page interfaces, ImageMagick rendering, base64 encoding.

### Go Libraries (external)

- **pgx**: PostgreSQL driver with connection pooling
- **golang-migrate**: Database migration management
- **google/uuid**: UUID generation
- **azure-sdk-for-go**: Azure Blob Storage client, Azure Identity (Entra auth)
- **pdfcpu**: PDF page count extraction on upload

### Frontend

- **Lit 3.x** (lit, @lit/context, @lit-labs/signals)
- **Bun**: JavaScript runtime and package manager (build-time)
- **Vite**: Build tool with TypeScript support (build-time)

### Runtime

- **ImageMagick 7.0+**: PDF-to-image rendering (required in deployment containers)
- **Azure PostgreSQL**: Managed database service
- **Azure Blob Storage**: Document persistence
- **Azure AI Foundry**: GPT-5-mini and GPT-5.2 deployments (both confirmed available on IL6)

## Integration Points

### Azure AI Foundry

- Single configurable agent targeting GPT-5-mini or GPT-5.2 per deployment
- Vision API for page classification (base64 data URI encoded images)
- Authentication via API key (Azure Key Vault) or access token
- Configuration: go-agents AgentConfig in config.json + HERALD_AGENT_* env var overrides

### Azure Blob Storage

- Document upload via Go service API (ensures DB + blob atomicity)
- Flat key structure: `documents/{document-id}/{filename}.pdf`
- Azure SDK azblob client with managed identity or connection string auth
- Immutable blobs (upload once, delete on removal, never modify)

### Azure PostgreSQL

- pgx driver with connection pooling
- golang-migrate for schema management
- Query builder pattern for dynamic filtering and sorting

### Azure Entra ID (Phase 4)

- Service authentication: managed identity for Blob Storage, PostgreSQL, AI Foundry access
- Web client authentication: OBO flow for user identity propagation
- Token management: Azure Identity SDK with credential chain

### External Document Sources

- Documents enter Herald exclusively through the upload API (single or batch)
- External systems push documents to Herald; Herald does not pull
- Document metadata fields for external system linkage TBD during Phase 1

## Open Questions

1. **Enhance stage trigger conditions**: What quality thresholds trigger conditional enhancement? The classify node must report whether image quality was a limiting factor. Requires experimentation during Phase 2.
2. **GPT-5-mini vs GPT-5.2 benchmarking**: Which model performs better at acceptable cost? Both confirmed available on IL6. Benchmarking during Phase 2.
3. **Azure deployment target**: Container Apps vs AKS vs App Service? Deferred to Phase 4.
4. **Bulk ingestion strategy**: Loading 1M documents requires a bulk ingestion approach feeding the upload API. Detailed strategy TBD.

## Resolved Questions

1. **External system identifier fields**: `external_id` and `external_platform` columns on the documents table provide linkage back to originating systems. Resolved in Phase 1.
2. **Confidence scoring approach**: Categorical (HIGH/MEDIUM/LOW). Aligns with classify-docs' proven approach and is more interpretable for human validators.
3. **Worker pool sizing**: Not applicable — no batch classification endpoint. Clients orchestrate parallel single-document classifications.
