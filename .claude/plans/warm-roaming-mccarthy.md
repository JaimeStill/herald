# Sitrep Draft — Herald MVP

## Context
Writing a situation report for government leadership on the Herald MVP. Announcement post style per writing profile: brief, direct, factual, impact-framed for leadership.

## Topics to Cover

### 1. What Herald Does
- Go web service that classifies DoD PDF documents' security markings using Azure AI vision models
- 4-node workflow: render PDF pages → parallel per-page AI vision analysis → conditional image enhancement → document-level synthesis
- Produces structured classification records: marking level, confidence, rationale, per-page findings
- High-confidence results require no human intervention; web client available for optional validation of low-confidence edge cases

### 2. Technologies
- **Backend**: Go 1.26, PostgreSQL 17, Azure Blob Storage
- **AI**: Azure AI Foundry (GPT vision models on IL6), go-agents library for LLM abstraction
- **Frontend**: Lit 3.x web components (TypeScript), MSAL.js for Azure Entra auth
- **Infrastructure**: Azure Container Apps (horizontal scaling), Bicep IaC (10 modular templates), managed identity for all service-to-service auth
- **PDF Processing**: ImageMagick 7 + Ghostscript for page rendering, pdfcpu for metadata extraction
- **Local Dev**: Docker Compose (PostgreSQL + Azurite), simple config overlay system — same codebase runs locally or in Azure with config changes only

### 3. Cloud Deployment Strategy
- Modular Bicep IaC: 10 templates composing identity, logging, PostgreSQL, storage, AI, container registry, container environment, app, migration job, and RBAC roles
- Serialized deployment chain prevents ARM race conditions
- Managed identity eliminates credential management — RBAC for storage, database, and AI services
- Container App: 2.0 CPU / 4Gi memory, 1-3 replica scaling, health probes at `/healthz` and `/readyz`
- Supports IL4/IL6 environments with token scope overrides for Azure Government

### 4. Local Portability
- Config overlay system: `config.json` (base) → `config.<env>.json` (overlay) → `secrets.json` → `HERALD_*` env vars
- `docker compose up` provides PostgreSQL + Azurite locally
- Same Go binary, same Dockerfile — only configuration differs between local dev and cloud
- Auth is opt-in: runs without Entra locally, toggles on via config overlay or env var

### 5. Time/Cost Savings Estimate
- **Manual**: Trained reviewer ~5-10 min/document (read pages, identify markings, record classification, cross-reference caveats)
- **Herald**: ~30-60 sec/document (PDF rendering + parallel vision API calls + synthesis)
- **Horizontal scaling**: Azure Container Apps scales to multiple replicas, CLI can drive concurrent classifications — throughput multiplies linearly
- **At 1M documents**: Manual = ~83K-167K person-hours (~40-80 FTE-years). Herald = fraction of that time with minimal human oversight limited to low-confidence results
- **Net**: 10-20x speed improvement per document, compounding with horizontal concurrency. Human effort shifts from full manual review to exception-based validation.

## Output
- Write to `~/Documents/shadow-clone/sitrep/sitrep.md`
- Announcement post format: opening statement, what it means, supporting detail
- 1-2 pages max, markdown formatted
