# Objective: Managed Identity for Azure Services

**Issue:** [#97](https://github.com/JaimeStill/herald/issues/97)
**Phase:** Phase 4 — Security and Deployment (v0.4.0)

## Scope

Wire managed identity credentials into all three Azure service clients (Storage, Database, AI Foundry) as an opt-in alternative to connection strings and API keys. Add `managed_identity` boolean to `AuthConfig` to control service client constructor selection independently from `auth_mode` (user authentication).

### Three-Stage Auth Model

| Stage | `auth_mode` | `managed_identity` | User Auth | Service Connections |
|-------|-------------|---------------------|-----------|---------------------|
| 1 | `none` | `false` (ignored) | None | Connection strings from env vars |
| 2 | `azure` | `false` | Entra JWT | Connection strings via Key Vault secrets |
| 3 | `azure` | `true` | Entra JWT | Managed identity tokens |

## Sub-Issues

| # | Title | Issue | Status | Dependencies |
|---|-------|-------|--------|--------------|
| 1 | Add credential-based storage constructor | [#105](https://github.com/JaimeStill/herald/issues/105) | Open | None |
| 2 | Add token-based database authentication | [#106](https://github.com/JaimeStill/herald/issues/106) | Open | None |
| 3 | Add agent token provider for AI Foundry bearer auth | [#107](https://github.com/JaimeStill/herald/issues/107) | Open | None |
| 4 | Add AuthConfig managed_identity flag and infrastructure wiring | [#108](https://github.com/JaimeStill/herald/issues/108) | Open | #105, #106, #107 |

Sub-issues 1–3 are independent and can be developed in parallel. Sub-issue 4 depends on all three.

## Architecture Decisions

- **Two-field auth model** — `auth_mode` (none/azure) controls user authentication; `managed_identity` (bool) controls service client constructor selection. Independent concerns, independently configurable.
- **Dual constructors** — Each service package (`pkg/storage/`, `pkg/database/`) gets a `NewWithCredential` constructor alongside the existing `New`. Infrastructure assembly branches based on `managed_identity` flag.
- **rt.NewAgent(ctx) method** — `workflow.Runtime` encapsulates the agent creation pattern (clone config, inject token, create agent) in a single method. Workflow nodes call `rt.NewAgent(ctx)` instead of `agent.New(&rt.Agent)` directly.
- **No config overlay files** — IL4/IL6 deployment configures via `HERALD_*` env vars on Azure Container Apps (environment variables + Key Vault references). No `config.il4.json` or `config.il6.json`.
- **Token provider closure** — `func(ctx) (string, error)` closure built in infrastructure wiring, threaded through to `workflow.Runtime.TokenProvider`. Nil when managed identity is disabled.
