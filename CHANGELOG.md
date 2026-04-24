# Changelog

## v0.5.0

### Classification Workflow

- Introduce a format-handler registry (`internal/format/`) with a `Handler` interface, a `SourceReader` abstraction over blob storage, and an explicit `Registry` composed once at `internal/api/domain.go` — decouples init / enhance / upload-validation from any single format's rasterizer. PDF and raw-image (PNG, JPEG, WEBP) land as the two initial handlers; future formats (DOCX, PPTX, TIFF) drop in as additional registrations without threading branches through the workflow or web client (#149)
- Extract the shared classification types (`ClassificationState`, `ClassificationPage`, `EnhanceSettings`, `Confidence`, state keys) into a new top-level `internal/state/` leaf package consumed by both `internal/format/` and `internal/workflow/` — breaks the prior workflow-centric ownership so format handlers and upload validation can depend on the types without a workflow-package import (#149)
- Replace the `github.com/JaimeStill/document-context` dependency with direct `pdfcpu` (page count) + `magick` calls using ImageMagick's native PDF page-selector syntax (`source.pdf[N]`). Drops one transitive-dependency tree and consolidates rendering on a single shared `format.Render` helper that both PDF and image handlers share (#149)
- Upload validation via `internal/documents/handler.go` rejects unsupported content types with HTTP 400 at ingress — error body enumerates the supported set from `format.Registry.SupportedContentTypes()` so clients self-correct without reading source. Adds `ErrUnsupportedContentType` sentinel mapped in `MapHTTPStatus` (#149)
- Image normalization pipeline (`internal/format/image.go`) — PNG sources stream through verbatim; JPEG and WEBP sources are normalized to PNG via magick at init time so downstream vision calls see uniform bytes regardless of source encoding. Enhance re-applies filters from the normalized PNG rather than re-fetching the source blob (#149)

### Dependencies

- Migrate from `github.com/JaimeStill/go-agents` and `go-agents-orchestration` to the `tailored-agentic-units` tau module graph (`agent`, `orchestrate`, `protocol`, `format`, `format/openai`, `provider`, `provider/azure`, `provider/ollama`) — includes rewrites of the Vision/Chat call sites to typed `protocol.Message` + `format.Image` and an `infrastructure.New` sync.Once that registers tau provider/format factories (#136)
- Remove `github.com/JaimeStill/document-context` in favor of direct `pdfcpu` + `magick` calls; `go mod tidy` drops its transitive deps (#149)

### Backend Infrastructure

- Promote `pkg/formatting/` to `pkg/core/` — the package had accumulated helpers (`FormatBytes`/`ParseBytes`, `Parse[T]`/`FromMap[T]`, and the new `WorkerCount`) that outgrew the original formatting-only scope. Package godoc now describes "stateless, cross-cutting primitives" with a guardrail: additions must be stateless and broadly applicable (#149)
- Add `pkg/core.WorkerCount` exported helper for bounded errgroup sizing — replaces three duplicated one-liners across `internal/format/pdf.go`, `internal/workflow/classify.go`, and `internal/workflow/enhance.go` (#149)
- Alias tau's `orchestrate/state` package as `taustate` throughout `internal/workflow/*.go` — resolves the package-name collision with Herald's new `internal/state`. External packages take the alias; local Herald identifiers stay natural (`state.*` refers to Herald, `taustate.*` refers to tau) (#149)

### Web Client

- Add a frontend format registry at `app/client/domains/formats/` mirroring the backend abstraction — `DocumentFormat` interface, PDF + image implementations, `findFormat`/`isSupported`/`acceptAttribute`/`dropZoneText`/`allSupportedContentTypes` helpers. Drives the upload widget's `accept` attribute, MIME filter, drop-zone text, and rejection toast; drives the blob viewer's render strategy (iframe vs img) with a generic iframe fallback for unregistered types (#149)
- `hd-blob-viewer` dispatches by content type — iframe for PDFs (browser's built-in viewer handles paging/zoom/search), `<img>` for PNG/JPEG/WEBP. Adds a `min-width: 0` / `min-height: 0` cascade across `blob-viewer` and `review-view` so `<img>` elements with large intrinsic dimensions (300 DPI page scans ~2550×3300 px) scale to the panel via `object-fit: contain` instead of bursting the flex container (#149)
- `hd-document-upload` reads its `accept` attribute, file filter, drop-zone label, and rejection toast content from the format registry — uploading an unsupported type (e.g. `.docx`) surfaces a warning toast and never reaches the HTTP boundary (#149)
- Persist list-view pagination, search, filter, and sort state in the URL query string — `document-grid` and `prompt-list` hydrate their `@state` from `queryParams()` on mount, write back via `updateQuery()` on every filter change, and omit default values so shareable URLs stay clean; navigating into `/review/:id` and returning (or reloading the page) restores the prior grid state (#135)
- Add `queryParams()` and `updateQuery()` helpers to `@core/router` — `updateQuery` merges a patch into `location.search`, deletes keys on empty/undefined/null values, and uses `history.replaceState` so filter changes don't remount the active view (#135)
- Retire the router's query-attribute splat on mount — query state now flows through the explicit helpers, keeping filter fields as `@state` (internal) rather than `@property` (parent-input) and preserving that semantic boundary (#135)
- Simplify `hd-pagination` to a compact, container-agnostic layout — chevron buttons (`‹` / `›`) with accessible labels replace "Prev"/"Next", the "Page X of N" caption collapses to `[input] / N`, page-size selector (12 / 24 / 48 / 96) on the left of the footer, editable page-number input commits on blur/Enter with `Math.trunc` clamp to `[1, totalPages]`, native number-input spinners hidden so the input auto-sizes to its digit count (forward-compat for ~750k-1M document counts), and prev/next `align-self: stretch` to match sibling input/select height. `document-grid` and `prompt-list` promote hardcoded `page_size: 12` to a `pageSize` `@state()` field and refetch with `page = 1` on change (#134, #135)
- Fix `hd-document-upload` queue overflowing the viewport with many queued files — host/drop-zone flex sized so the module participates in the view's flex column, new `.queue-list` wrapper scrolls the entries while the File Queue header (title, count, Clear/Upload) stays pinned (#133)
- Harden MSAL token-refresh flow against stale/corrupted cache after long idle — wrap `handleRedirectPromise()` in `init()` so nonce-mismatch-after-idle and other stale redirect-return errors clear the cache instead of rejecting `init()` and leaving the shell unmounted; broaden the `getToken()` catch to treat any non-`InteractionRequiredAuthError` as cache corruption (`clearCache()` + `acquireTokenRedirect`), short-circuit on `BrowserAuthError` with code `interaction_in_progress` so in-flight redirects are not stacked, and switch the silent-failure fallback from `loginRedirect` to the canonical MSAL SPA pattern `acquireTokenRedirect(request)` to preserve account/login-hint context (#137)
- Introduce a minimal `Toast` service + `<hd-toast-container>` element in `app/client/ui/elements/toast.ts` — module-level singleton bus with `success`/`error`/`warning`/`info` helpers, kind-specific auto-dismiss (3s/6s/5s/3s), stack capped at 5 visible, click-to-dismiss cancels the timer. Stack positioned bottom-center at `min(72ch, 100dvw - var(--space-8))` width via `margin-inline: auto`, monospace styling with semantic border-left accent. Mounts from `app.ts` via `document.body.appendChild` after the Router starts (#139)
- Wire every existing mutation call site through the toast service so every command execution surfaces success and failure feedback: single-document delete, SSE classify `onComplete`/`onError`, bulk classify (per-document outcome via the classify callbacks), classification validate + update, prompt create/update/delete, prompt activate/deactivate, and batch document upload (summary toasts after `Promise.allSettled`). Forms (`classification-panel`, `prompt-form`) keep their inline `this.error` display for form-local context; toasts are additive (#139)
- Add bulk-delete action to the document grid — a "Delete N Documents" button renders adjacent to the existing bulk classify button when ≥1 document is selected, opens `hd-confirm-dialog` with "Are you sure you want to delete {N} documents?", and on confirm runs `DocumentService.delete(id)` in parallel across the selection. Successful IDs drop out of the selection, failed IDs remain selected for retry, and each failure emits an error toast with the filename + error (#139)
- Migrate `hd-confirm-dialog` to a native `<dialog>.showModal()` overlay — drops the manual `position: fixed; z-index: 100` scrim + dialog div in favor of the top layer. Wires the native `cancel` event (Escape), backdrop click (`event.target === dialogEl`), and `autofocus` on the Confirm button; `:host { display: contents }` so the custom-element host contributes no box to the caller's layout tree. Adds `confirmKind: "danger" | "primary" | "neutral"` property (default `"danger"`) mapping caller intent to a button color class (`btn-red`/`btn-green`/`btn`) without exposing raw class names (#145)
- Migrate `hd-toast-container` to `popover="manual"` — host enters the top layer via `setAttribute("popover", "manual")` + `this.showPopover()` in `connectedCallback`, and `hidePopover()` is guarded by `this.matches(":popover-open")` in `disconnectedCallback`. Fixed positioning lives on the inner `.stack` div (UA `[popover]` defaults fight `:host` rules). `z-index: 200` is gone — top-layer stacking handles ordering (#145)
- Introduce `hd-tooltip` at `app/client/ui/elements/tooltip.ts` — hover- and focus-triggered primitive using `popover="hint"` (does not close open `auto` popovers; closes other hints) with CSS Anchor Positioning (`anchor-name` / `position-anchor` / `position-try-fallbacks: flip-block` so the tip flips above when below lacks room) and a 150ms show delay to avoid flicker on quick mouse travel. Wraps `document-card` filenames and `prompt-card` names so users can reveal the full value on hover or keyboard focus when the card layout truncates it (#145)

### Design System

- Add `design/styles/scroll.module.css` with `.scroll-y` / `.scroll-x` utilities — bundle `overflow-*: auto`, `scrollbar-gutter: stable`, and axis padding so scroll containers share consistent ergonomics across shadow DOM. Migrate every scroll container (`document-grid`, `prompt-list`, `classification-panel`, `prompt-form` body and defaults, `review-view` classification panel, `document-upload` queue list) to the utility (#133, #145)
- Four-layer cascade stack finalized as `tokens, reset, theme, app`. `scrollbar-gutter: stable` is scoped to the `.scroll-y` / `.scroll-x` utilities rather than a universal `*` rule — the broad selector matches on `overflow: hidden` as well and leaks a phantom ~15px gutter onto every hidden-overflow container in the app shell (#133, #145)
- Add `declare module "*.css"` to `client/css.d.ts` so side-effect imports of non-module CSS (e.g. `@design/index.css`) type-check under `tsc --noEmit` (#133)

### Conventions

- Establish the Overlay Convention across `.claude/CLAUDE.md` and `.claude/skills/web-development/SKILL.md`: any UI that overlays the page (tooltips, menus, popovers, toasts, confirmation dialogs, modal forms) must use native `<dialog>` + `.showModal()` or the Popover API (`popover="auto"` / `popover="hint"` / `popover="manual"`). No manual `position: fixed; z-index: ...` overlay divs. Includes a decision matrix, per-primitive patterns, anti-patterns, and reference components. Expand `.claude/skills/web-development/references/components.md` with an Overlay Elements section covering modal dialog, toast stack, and anchored tooltip patterns (#139, #145)

## v0.4.2

### Authentication

- Fix Azure Government auth by composing authority URL from base + tenant ID — `HERALD_AUTH_AUTHORITY` now accepts the base URL (e.g., `https://login.microsoftonline.us`) and derives the full OIDC authority
- Add `knownAuthorities` to MSAL config to prevent instance discovery against commercial endpoints
- Add `authAuthority` Bicep parameter for injecting `HERALD_AUTH_AUTHORITY` on non-commercial clouds

## v0.4.1

### Deployment

- Add `computeTarget` parameter to Bicep infrastructure for selecting between Container Apps (`containerapp`) and App Service for Containers (`appservice`) compute targets
- Add App Service modules: `appservice-plan.bicep`, `appservice.bicep`
- Add `Consumption` workload profile to Container App Environment for IL6 compatibility
- Decouple ACR from Bicep — ACR is now created externally and referenced via `acrName` parameter
- Rename container app from `{prefix}` to `{prefix}-app`
- Remove migration infrastructure from container image and Bicep — migrations now use standalone `migrate` binary published as a separate release artifact

### Container

- Remove `migrate` binary from Docker image — image now only contains the `herald` server binary
- Revert Dockerfile to `ENTRYPOINT ["herald"]`

### CI/CD

- Add `migrate-release.yml` workflow for independent migration binary releases triggered by `migrate-v*` tags
- Produce `migrate-linux-amd64` and `migrate-windows-amd64.exe` release artifacts

## migrate-v0.1.0

Initial standalone migration binary release. Embeds all SQL migrations from `cmd/migrate/migrations/` (000001 through 000005). Replaces the in-image migration infrastructure from v0.4.0.

## v0.4.0

### Authentication

- Add `pkg/auth/` package with unified auth config, User context helpers, error sentinels, and JWT validation middleware using `go-oidc` for OIDC discovery and token verification (#113)
- Wire auth middleware into API module between CORS and Logger with backward-compatible fallback when auth is disabled (#114)
- Populate `validated_by` from authenticated JWT user identity in classifications Validate and Update handlers (#115)
- Inject browser-safe auth config into web app HTML template for MSAL.js initialization via `ClientAuthConfig` struct and conditional `<script id="herald-config">` rendering (#118)
- Add MSAL auth service with login gate wrapping `@azure/msal-browser` for initialization, redirect login, and token acquisition (#119)
- Wire bearer token injection into API `request()` and `stream()` with 401 retry, silent token refresh, authenticated blob download for iframe viewing, and user menu with display name and logout (#120)

### Managed Identity

- Add AuthConfig with typed AuthMode enum, Azure Identity credential provider factory, and Infrastructure/Runtime credential propagation (#103)
- Add credential-based storage constructor with `NewWithCredential` for managed identity auth and `ServiceURL` config field (#105)
- Add token-based database authentication with `NewWithCredential` constructor, pgx `BeforeConnect` hook for Entra token injection, and configurable `TokenLifetime` (#106)
- Add agent factory closure to `workflow.Runtime` for auth-mode-aware agent creation, replacing direct `agent.New` calls with `rt.NewAgent(ctx)` (#107)
- Wire `ManagedIdentity` flag into infrastructure for database, storage, and agent services (#108)
- Upgrade go-agents to v0.4.0 for native managed identity support (#126)

### Deployment

- Harden Dockerfile with ImageMagick, Ghostscript, non-root user, and HEALTHCHECK; add Docker Compose production overlay (#101)
- Add modular Bicep infrastructure-as-code for Azure Container Apps deployment — ten modules orchestrated by `main.bicep` with GHCR/ACR registry switching; build both `herald` and `migrate` binaries (#125)
- Add deployment guide at `deploy/README.md` with prerequisites, architecture, module chain, and troubleshooting runbook (#125, #126)
- Add configurable `cognitiveDeploymentCapacity` parameter for token rate limits (#126)

### Configuration

- Make `AgentScope` and `TokenScope` configurable for Azure Government with commercial defaults and `HERALD_AUTH_AGENT_SCOPE` / `HERALD_DB_TOKEN_SCOPE` env var overrides (#124)
- Fix `HERALD_AGENT_BASE_URL` to append `/openai` for OpenAI-kind Cognitive Services accounts (#126)
- Fix `HERALD_DB_USER` to use Entra admin principal name instead of managed identity client ID (#126)

### Observability

- Add centralized error logging to `StreamingObserver` via logger injection (#126)

### Web Client

- Remove `PageRequest` from `@core` — all domains now define their own `SearchRequest` with domain-specific filters; `toQueryString` retained as generic utility

## v0.3.0

### Web Infrastructure

- Add `pkg/web/` template and static file infrastructure — TemplateSet with layout inheritance, embedded filesystem serving, and SPA-compatible router with fallback (#62)
- Add Go web app module with embedded client assets, SPA shell template, server integration alongside API module, Air hot reload configuration, and mise dev workflow tasks (#64)

### Client Build System and Design

- Add client-side web application foundation — Bun build pipeline with CSS module plugin (`*.module.css` → `CSSStyleSheet`), CSS cascade layer design system with dark/light theme, History API router, `Result<T>` API layer with SSE streaming, and placeholder views for all routes (#63)
- Restructure `app/client/` into four top-level packages (`core/`, `domains/`, `ui/`, `design/`) with path aliases and View → Module → Element component vocabulary; consolidate design system with `light-dark()` tokens, shared button/badge styles, monospace interactive elements, and View Transitions API progressive enhancement (#77)
- Flatten `ui/` tier directories removing domain subdirectories; invert router route dependency via constructor injection; consolidate badge CSS variants by color (#82)
- Extract duplicated CSS into shared style modules (inputs, labels, cards, button color variants), enforce named barrel exports, and align web-development skill documentation with final codebase state (#90)

### Data Layer and Services

- Add web client data layer — enhance SSE `stream()` with event type parsing and POST support, TypeScript domain types mirroring Go API shapes, and stateless service objects mapping all handler endpoints across documents, classifications, prompts, and storage domains (#74)

### SSE Classification Streaming

- Add workflow streaming observer with channel-based event emission, lean event type filtering (node.start, node.complete, complete, error), generic `FromMap[T]` map decoder, and `NewGraphWithDeps` observer injection (#70)
- Add SSE classification streaming — modify `POST /api/classifications/{documentId}` to return an SSE event stream with real-time workflow progress and persisted classification result on completion (#71)

### Document Management View

- Add document card and classify progress pure elements with `WorkflowStage` domain type and shared formatting utilities (#75)
- Add document upload component with drag-and-drop file selection, per-file metadata inputs, concurrent `Promise.allSettled` upload coordination, and per-file status tracking (#76)
- Add document grid module with search, filtering, sorting, pagination, SSE classify orchestration, bulk select + classify, and delete with confirmation dialog; compose documents view with upload toggle and grid refresh wiring (#77)

### Prompt Management View

- Add prompt card pure element with stage badge, active indicator, and toggle/delete events; add prompts `SearchRequest` type with domain-owned pagination fields (#82)
- Add prompt list module with search (300ms debounce), stage filtering, sorting, pagination, activate/deactivate lifecycle, and delete with confirmation dialog (#83)
- Add prompt form module with create/edit modes, prompts view composition with split-panel layout, collapsible default prompt reference panel, `?default=true` query param for instructions endpoint; remove unused `@lit-labs/signals` and `@lit/context` dependencies (#84)

### Document Review View

- Add `GET /api/storage/view/{key}` inline blob streaming endpoint, generic `hd-blob-viewer` pure element, `StorageService.view()` URL builder, and review view composition with two-panel layout (#88)
- Add markings list pure element, classification panel module with view/validate/update modes and event-driven state refresh (#89)
- Disable classification panel actions when validated, show external_id/platform on document cards (#90)

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
