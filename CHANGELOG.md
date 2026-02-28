# Changelog

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
