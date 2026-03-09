# Objective: Azure Identity Credential Infrastructure

**Issue:** [#96](https://github.com/JaimeStill/herald/issues/96)
**Phase:** Phase 4 — Security and Deployment (v0.4.0)

## Scope

Add the Azure Identity SDK and build the credential provider foundation for managed identity support. Introduce `AuthConfig` with the three-phase finalize pattern and wire credential creation into `infrastructure.New()`. The credential is optional — nil when `auth_mode` is `none` (default), preserving current behavior.

## Sub-Issues

| # | Title | Issue | Status |
|---|-------|-------|--------|
| 1 | Add AuthConfig and credential provider infrastructure | [#103](https://github.com/JaimeStill/herald/issues/103) | Open |

## Architecture Decisions

- **Auth mode naming** — `none` / `azure` rather than `connection_string` / `managed_identity`. Describes the identity provider, not the connection mechanism. More intuitive and extensible.
- **Factory on config** — `TokenCredential()` method on `AuthConfig` rather than a separate `pkg/credential/` package. The factory is trivial (~10 lines) and config owns the mode decision. Consumers access the credential via `Infrastructure.Credential`.
- **`DefaultAzureCredential`** — wraps the full Azure credential chain (managed identity, workload identity, Azure CLI) rather than specific credential types. Handles all deployment scenarios automatically.
- **No separate package** — single sub-issue covers config + factory + infrastructure wiring. The scope is tightly coupled and small (~140 lines of new code).
