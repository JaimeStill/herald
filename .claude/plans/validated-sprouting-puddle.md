# Plan: Harden Dockerfile and Add Production Compose Overlay (#101)

## Context

Issue #101 under Objective #95 (Docker Production Image). The current Dockerfile produces a minimal Alpine image without ImageMagick/Ghostscript (required for PDF rendering), runs as root, and has no health check. There's no way to run the full stack (app + postgres + azurite) entirely in Docker for integration testing.

## Changes

### 1. Harden Dockerfile (`Dockerfile`)

Current runtime stage is bare `alpine:3.21` with only `ca-certificates`. Update to:

- Add `imagemagick` and `ghostscript` packages (Alpine 3.21 repos provide ImageMagick 7.x)
- Create non-root `herald` user/group with `addgroup`/`adduser`
- Set `WORKDIR /app` for predictable config file loading
- Set `USER herald`
- Add `HEALTHCHECK` instruction: `curl` to `/healthz` (10s interval, 5s timeout, 5 retries) — need to add `curl` package too

### 2. Update `.dockerignore`

Add `secrets.json` to prevent secrets from being copied into the build context.

### 3. Create `config.docker.json`

Environment overlay for Docker Compose networking. Overrides only what differs from `config.json`:

- `database.host` → `herald-postgres` (container name from `compose/postgres.yml`)
- `storage.connection_string` → replace `127.0.0.1:10000` with `herald-azurite:10000`

### 4. Create `compose/app.yml`

Herald server service compose file:

- Build from project root Dockerfile
- `depends_on` postgres and azurite with `condition: service_healthy`
- `HERALD_ENV=docker` environment variable (triggers `config.docker.json` overlay)
- Bind-mount `config.json` and `config.docker.json` into `/app/`
- Port 8080
- Joins the `herald` network

## Files Modified

| File | Action |
|------|--------|
| `Dockerfile` | Edit — add packages, non-root user, WORKDIR, HEALTHCHECK |
| `.dockerignore` | Edit — add `secrets.json` |
| `config.docker.json` | Create — Docker network overlay |
| `compose/app.yml` | Create — Herald server service |

## Verification

1. `docker build -t herald .` succeeds
2. `docker run --rm herald magick -version` shows ImageMagick 7.x
3. `docker run --rm herald gs --version` shows Ghostscript version
4. `docker run --rm herald whoami` shows `herald` (non-root)
5. Full stack: `docker compose -f docker-compose.yml -f compose/app.yml up --build` starts all services
6. Health check passes (visible in `docker ps` HEALTH column)
7. Web client accessible at `http://localhost:8080/app`
