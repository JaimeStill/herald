# Changelog

## v0.3.0-dev.61.90
- Extract duplicated CSS into shared style modules (inputs, labels, cards, button color variants), fix `querySelector<any>` type erasure, enforce named barrel exports, disable classification panel actions when validated, show external_id/platform on document cards, and align web-development skill documentation with final codebase state (#90)

## v0.3.0-dev.61.89
- Add markings list pure element rendering security markings as badge tags, classification panel module with view/validate/update modes and event-driven state refresh, and integrate panel into review view replacing placeholder content (#89)

## v0.3.0-dev.61.88
- Add `GET /api/storage/view/{key}` inline blob streaming endpoint, generic `hd-blob-viewer` pure element with caller-driven `src` URL, `StorageService.view()` URL builder, and review view composition with document loading and two-panel layout (#88)

## v0.3.0-dev.60.84
- Add prompt form module with create/edit modes, FormData extraction, and error display; compose prompts view with split-panel layout coordinating list and form via custom events; add collapsible default prompt reference panel showing stage specification and hardcoded default instructions; add `?default=true` query param to instructions endpoint to bypass active DB overrides; establish custom event naming convention (simple action verbs, no domain prefix); remove unused `@lit-labs/signals` and `@lit/context` dependencies (#84)

## v0.3.0-dev.60.83
- Add prompt list module with search (300ms debounce), stage filtering, sorting, pagination, activate/deactivate lifecycle, and delete with confirmation dialog; clean up document-grid filter/sort/search handlers to delegate to `refresh()` (#83)

## v0.3.0-dev.60.82
- Add prompt card pure element with stage badge, active indicator, and toggle/delete events; add prompts `SearchRequest` type with domain-owned pagination fields; flatten `ui/` tier directories removing domain subdirectories; invert router route dependency via constructor injection with routes at `app/client/routes.ts`; consolidate badge CSS variants by color; fix app test base href assertion for format-agnostic self-closing tags (#82)

## v0.3.0-dev.59.77
- Add document grid module with search, filtering, sorting, pagination, SSE classify orchestration, bulk select + classify, and delete with confirmation dialog; compose documents view with upload toggle and grid refresh wiring; restructure `app/client/` into four top-level packages (`core/`, `domains/`, `ui/`, `design/`) with path aliases and View → Module → Element component vocabulary; consolidate design system with `light-dark()` tokens, shared button/badge styles, monospace interactive elements, and View Transitions API progressive enhancement (#77)

## v0.3.0-dev.59.76
- Add document upload stateful component with drag-and-drop file selection, per-file metadata inputs, concurrent `Promise.allSettled` upload coordination, and per-file status tracking (#76)

## v0.3.0-dev.59.75
- Add document card and classify progress pure elements with `WorkflowStage` domain type, shared formatting utilities, and corrected streaming orchestration in web-development skill references (#75)

## v0.3.0-dev.59.74
- Add web client data layer — enhance SSE `stream()` with event type parsing and POST support, TypeScript domain types mirroring Go API shapes, and stateless service objects mapping all handler endpoints across documents, classifications, prompts, and storage domains (#74)

## v0.3.0-dev.58.71
- Add SSE classification streaming — modify `POST /api/classifications/{documentId}` to return an SSE event stream with real-time workflow progress (node.start, node.complete, complete, error events) and persisted classification result on completion (#71)

## v0.3.0-dev.58.70
- Add workflow streaming observer with channel-based event emission, lean event type filtering (node.start, node.complete, complete, error), generic `FromMap[T]` map decoder, and `NewGraphWithDeps` observer injection (#70)

## v0.3.0-dev.57.64
- Add Go web app module with embedded client assets, SPA shell template, server integration alongside API module, Air hot reload configuration, and mise dev workflow tasks (#64)

## v0.3.0-dev.57.63
- Add client-side web application foundation — Bun build pipeline with CSS module plugin (`*.module.css` → `CSSStyleSheet`), CSS cascade layer design system with dark/light theme, History API router, `Result<T>` API layer with SSE streaming, and placeholder views for all routes (#63)

## v0.3.0-dev.57.62
- Add `pkg/web/` template and static file infrastructure — TemplateSet with layout inheritance, embedded filesystem serving, and SPA-compatible router with fallback (#62)

## v0.2.0

### Agent Configuration

- Migrate config format from TOML to JSON, add go-agents AgentConfig to Config and Infrastructure with env var overrides and startup validation (#29)

### Database Schema

- Add classification engine database migration with classifications and prompts tables, CHECK constraints, and indexes (#30)

### Prompts Domain

- Add prompts domain with full CRUD, typed stage enum with JSON validation, partial unique index for single active prompt per stage, and atomic activation swap (#34)
- Restructure API Cartographer to subdirectory-per-group layout with Kulala-compatible `.http` test files (#34)
- Add hardcoded default instructions and specifications per workflow stage, `Instructions` and `Spec` methods on the prompts System interface, stage content API endpoints, `ParseStage` validation, and remove init as a prompt stage (#37)

### Classification Workflow

- Add workflow foundation — classification types, runtime dependency struct, sentinel errors, generic JSON parser with markdown code fence fallback, and prompt composition (#38)
- Add init node with concurrent page rendering to temp storage, state key constants, and document-context/go-agents-orchestration dependencies (#39)
- Add classify node with per-page-only analysis and just-in-time image encoding (#40)
- Introduce 4-node workflow topology (init → classify → enhance? → finalize) with dedicated finalize stage for document-level classification synthesis (#40)
- Align classify and enhance specs with optimized workflow types, add finalize stage to prompts domain (stages, specs, instructions, migration) (#40)
- Add enhance node (re-render flagged pages with structured ImageMagick settings, reclassify via vision), finalize node (document-level classification synthesis via Chat), state graph assembly with conditional enhancement edge, and top-level Execute function with temp directory lifecycle (#41)
- Replace `Enhance bool` + `Enhancements string` with `EnhanceSettings` struct carrying typed rendering parameters (brightness, contrast, saturation), enabling programmatic re-rendering without LLM interpretation (#41)
- Parallelize classify and enhance workflow nodes with bounded errgroup concurrency, removing sequential context accumulation in favor of independent per-page analysis (#51)

### Classifications Domain

- Move `workflow/` to `internal/workflow/` to formalize internal dependency graph (#47)
- Add classifications domain — types, system interface, sentinel errors with HTTP mapping, query projection with JSONB handling, and repository with workflow integration, upsert semantics, and transactional document status transitions (#47)
- Add classifications handler with 8 HTTP endpoints (list, find, find-by-document, search, classify, validate, update, delete), API module wiring, and route registration (#48)
- Internalize workflow runtime construction in `classifications.New` — API composition root passes raw infrastructure deps, no longer imports workflow (#48)
- Add `secrets.json` config pipeline stage for local secret storage, merged after overlay and before env vars (#48)
- Fix missing `Agent` field in `api.NewRuntime` that caused nil pointer dereference on classify (#48)
- Add Azure AI Foundry provisioning scripts for resource group, cognitive services, and model deployment (#48)
- Add API Cartographer documentation for classifications endpoints (#48)

### Query Builder

- Add query builder JOIN support with context-switching ProjectionMap pattern, ordered JoinClause map, and From() method for multi-table queries (#53)
- Extend Document struct with classification metadata (classification, confidence, classified_at) via LEFT JOIN on classifications table (#53)
- Add classification and confidence filters to document queries (#53)

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
