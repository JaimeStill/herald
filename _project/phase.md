# Phase 2 — Classification Engine

**Version Target:** v0.2.0

## Scope

Build the classification engine that reads security markings from PDF documents using Azure AI Foundry GPT vision models. Adapts the sequential page-by-page context accumulation pattern from classify-docs (96.3% accuracy) into a 3-node state graph (init → classify → enhance?) hosted in Herald's web service. Two new domains (prompts, classifications) provide persistence and API access.

## Goals

- Single externally-configured agent (go-agents) connected to Azure AI Foundry
- Classification workflow state graph using go-agents-orchestration
- Sequential page-by-page processing with context accumulation via document-context
- Named prompt overrides with CRUD API (prompts domain)
- Classification result persistence with flattened workflow metadata (classifications domain)
- Single-document classification HTTP endpoint
- Document status transitions through the classification lifecycle (pending → review → complete)
- Manual validation and adjustment of classification results

## Objectives

| # | Objective | Status | Dependencies |
|---|-----------|--------|--------------|
| [#24](https://github.com/JaimeStill/herald/issues/24) | Agent Configuration and Database Schema | Complete | — |
| [#25](https://github.com/JaimeStill/herald/issues/25) | Prompts Domain | Open | #24 |
| [#26](https://github.com/JaimeStill/herald/issues/26) | Classification Workflow | Open | #24, #25 |
| [#27](https://github.com/JaimeStill/herald/issues/27) | Classifications Domain | Open | #24, #26 |
| [#28](https://github.com/JaimeStill/herald/issues/28) | Classification HTTP Endpoints | Open | #25, #27 |

## Constraints

- **No batch classification endpoint** — clients orchestrate parallel single-document classifications (same pattern as document uploads)
- **No web client** — Lit SPA is Phase 3
- **No authentication** — Azure Entra ID is Phase 4
- **No observer/checkpoint infrastructure** — results self-contain provenance via flattened metadata columns
- **No workflow registry** — single workflow, direct `Execute()` function
- **Confidence scoring is categorical** — HIGH/MEDIUM/LOW (aligns with classify-docs)
- **Enhance node starts as placeholder** — trigger conditions require experimentation; initially always skips

## Cross-Cutting Decisions

- **Workflow topology**: init → classify → enhance? — init purely prepares images, classify runs sequential page-by-page analysis, enhance conditionally re-renders and reassesses as a terminal stage
- **Agent lifetime**: Stateless, created during cold start, stored on `Infrastructure` (no lifecycle hooks)
- **Metadata storage**: Flattened columns on `classifications` table, no JSONB for workflow metadata
- **Table naming**: `prompts` (not `prompt_modifications`) — single prompt-related table
- **Prompt overrides**: Hard-coded defaults in `workflow/prompts.go` + optional per-stage overrides from the prompts domain
