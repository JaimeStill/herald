# Objective: Docker Production Image

**Issue:** [#95](https://github.com/JaimeStill/herald/issues/95)
**Phase:** Phase 4 — Security and Deployment (v0.4.0)

## Scope

Harden the Dockerfile for production deployment with ImageMagick 7.0+ support, security best practices, and local integration testing infrastructure.

## Sub-Issues

| # | Title | Issue | Status |
|---|-------|-------|--------|
| 1 | Harden Dockerfile and add production compose overlay | [#101](https://github.com/JaimeStill/herald/issues/101) | Open |

## Architecture Decisions

- **Config files mounted, not baked in** — image stays environment-agnostic. Config overlay loaded at runtime via bind-mount + `HERALD_ENV` env var.
- **Opt-in compose overlay** — `compose/app.yml` is not auto-included in `docker-compose.yml` to avoid breaking the existing dev workflow (server runs on host, only infra in Docker). Full-stack usage: `docker compose -f docker-compose.yml -f compose/app.yml up --build`.
- **Alpine packages** — `imagemagick` and `ghostscript` from Alpine 3.21 repos (not compiled from source). Satisfies the 7.0+ requirement.
