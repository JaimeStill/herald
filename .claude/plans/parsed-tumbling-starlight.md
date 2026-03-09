# Objective Planning: #95 Docker Production Image

## Context

Herald's Dockerfile is a minimal multi-stage build that produces a working image but lacks runtime dependencies (ImageMagick, Ghostscript) needed for the classification workflow, has no security hardening (runs as root), no Docker HEALTHCHECK, and no compose overlay for running the full stack locally. This objective closes those gaps to produce a production-ready image.

## Previous Objective

`_project/objective.md` contains a placeholder ("No active objective. Awaiting Phase 4 planning.") — no transition closeout needed.

## Sub-Issue Decomposition

Single sub-issue. The entire scope is tightly cohesive infrastructure work across a small number of files.

### Sub-Issue 1: Harden Dockerfile and add production compose overlay

**Branch:** `95-docker-production-image`
**Labels:** `infrastructure`
**Milestone:** `v0.4.0 - Security and Deployment`

**Scope:**

Dockerfile hardening:
- Add `imagemagick`, `ghostscript` to `apk add` in the runtime stage (Alpine 3.21 has ImageMagick 7.x). Required by document-context's PDF→image rendering.
- Create non-root `herald` user/group via `addgroup -S` / `adduser -S`. Set `USER herald`.
- Add `HEALTHCHECK` instruction: `wget --spider -q http://localhost:8080/healthz` (wget ships with Alpine). Use 10s interval, 5s timeout, 5 retries (matches compose healthcheck patterns).
- Add `WORKDIR /app` so CWD is predictable for config file loading at runtime.
- Add `secrets.json` to `.dockerignore` as a safety net.

Production compose overlay:
- Create `config.docker.json` — overlay that points DB host to `herald-postgres` and storage connection string to `herald-azurite:10000`. Only host references change; ports/credentials stay at defaults matching compose services.
- Create `compose/app.yml` — herald server service definition:
  - Container name: `herald-app`
  - Build from repo root Dockerfile
  - `depends_on` postgres and azurite with `condition: service_healthy`
  - Environment: `HERALD_ENV=docker`, `HERALD_AGENT_TOKEN` passed from host
  - Bind-mount `config.json` and `config.docker.json` into `/app` (container WORKDIR)
  - Port 8080, `herald` network
- Keep `docker-compose.yml` unchanged (infrastructure-only for local dev). Full-stack usage: `docker compose -f docker-compose.yml -f compose/app.yml up --build`

**Key files:**
- `Dockerfile` — runtime deps, non-root user, HEALTHCHECK, WORKDIR
- `.dockerignore` — add `secrets.json`
- `compose/app.yml` — new, herald server service
- `config.docker.json` — new, Docker environment overlay

**Config overlay mechanism (confirmed):** `internal/config/config.go:185-193` — `HERALD_ENV=docker` triggers `os.Stat("config.docker.json")`, loads and merges onto base config.

**Acceptance criteria:**
- [ ] `docker build -t herald .` succeeds
- [ ] Container runs as non-root (`whoami` → `herald`)
- [ ] `magick -version` and `gs --version` available inside the container
- [ ] Docker HEALTHCHECK passes after startup
- [ ] `docker compose -f docker-compose.yml -f compose/app.yml up --build` starts full stack
- [ ] Herald web client accessible at `http://localhost:8080/app`
- [ ] Health endpoints respond: `/healthz` (200), `/readyz` (200 after startup)
- [ ] Document upload succeeds through the containerized service
- [ ] Classification workflow completes (requires `HERALD_AGENT_TOKEN` set on host)

## Architecture Decisions

- **Config files mounted, not baked in:** Image stays environment-agnostic. Config overlay loaded at runtime via bind-mount + `HERALD_ENV` env var.
- **Opt-in compose overlay:** `compose/app.yml` is not auto-included in `docker-compose.yml` to avoid breaking the existing dev workflow (server runs on host, only infra in Docker).
- **Alpine packages:** `imagemagick` and `ghostscript` from Alpine repos (not compiled from source). Alpine 3.21 ships ImageMagick 7.x which satisfies the 7.0+ requirement.

## `_project/objective.md` Update

After the sub-issue is created, write `_project/objective.md` with the objective reference, phase context, sub-issue table, and architecture decisions above.
