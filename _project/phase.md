# Phase 1 — Service Foundation

**Version Target:** v0.1.0

## Scope

Establish the Go service foundation for Herald — from `go.mod` to a running web service that accepts document uploads and manages them through Azure Blob Storage and PostgreSQL. All infrastructure patterns are adapted from agent-lab's proven Layered Composition Architecture.

## Goals

- Running Go web service with cold start → hot start → graceful shutdown lifecycle
- Configuration system with TOML base, environment overlays, and env var overrides
- PostgreSQL integration with migrations, query builder, and repository helpers
- Azure Blob Storage integration for document persistence
- Module-based HTTP routing with middleware and response utilities
- Complete document domain: upload (single + batch), registration, metadata, CRUD queries
- Local development environment via Docker Compose (PostgreSQL + Azurite)
- mise-based task runner for build, test, and development workflows

## Objectives

| # | Objective | Status | Dependencies |
|---|-----------|--------|--------------|
| [#1](https://github.com/JaimeStill/herald/issues/1) | Project Scaffolding, Configuration, and Service Skeleton | Complete | — |
| [#2](https://github.com/JaimeStill/herald/issues/2) | Database Schema and Migration Tooling | Open | #1 |
| [#3](https://github.com/JaimeStill/herald/issues/3) | Document Domain | Open | #1, #2 |

## Constraints

- **No classification logic** — workflow, agents, and prompt management are Phase 2
- **No web client** — Lit SPA is Phase 3
- **No authentication** — Azure Entra ID is Phase 4
- **No OpenAPI/Scalar** — excluded per concept for velocity
- **mise over Makefile** — all task automation via `.mise.toml`
- **Azurite for local dev** — Azure Blob Storage emulator via Docker Compose

## Cross-Cutting Decisions

- **Storage implementation**: Azure Blob Storage via azblob SDK; Azurite emulator for local development
- **Database driver**: pgx with connection pooling
- **Document status**: `pending` is the only status set in Phase 1; transitions (`review`, `complete`) added in Phase 2
- **Batch upload**: First-class `POST /documents/batch` endpoint for the 1M-document ingestion use case
- **Test pattern**: Black-box tests in `tests/` directory mirroring `internal/` (e.g., `tests/documents/`)
