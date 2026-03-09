# 101 - Harden Dockerfile and Add Production Compose Overlay

## Problem Context

The current Dockerfile produces a minimal Alpine image that lacks ImageMagick and Ghostscript (required for PDF→image rendering in the classification workflow), runs as root, and has no health check. There is also no way to run the full stack (app + postgres + azurite) entirely in Docker for local integration testing.

## Architecture Approach

The image stays environment-agnostic — config files are mounted at runtime, not baked in. The compose overlay is opt-in via `-f` flag to preserve the existing dev workflow where only infrastructure runs in Docker and the Go server runs on the host.

## Implementation

### Step 1: Harden the Dockerfile

Replace the runtime stage in `Dockerfile`:

```dockerfile
FROM alpine:3.21
RUN apk add --no-cache ca-certificates curl imagemagick ghostscript
RUN addgroup -S herald && adduser -S herald -G herald
COPY --from=build /herald /usr/local/bin/herald
WORKDIR /app
USER herald
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=5s --retries=5 \
  CMD ["curl", "-f", "http://localhost:8080/healthz"]
ENTRYPOINT ["herald"]
```

Key changes from current:
- `imagemagick` and `ghostscript` packages added (ImageMagick delegates PDF rasterization to Ghostscript)
- `curl` added for the HEALTHCHECK probe
- Non-root `herald` user/group created and set as the runtime user
- `WORKDIR /app` ensures predictable working directory for config file loading
- `HEALTHCHECK` probes the existing `/healthz` endpoint

### Step 2: Update `.dockerignore`

Add `secrets.json` to the existing `.dockerignore`:

```
secrets.json
```

This prevents secrets from being copied into the Docker build context.

### Step 3: Create `config.docker.json`

Create `config.docker.json` in the project root. This is the environment overlay loaded when `HERALD_ENV=docker`. It overrides only the values that differ inside Docker Compose networking:

```json
{
  "database": {
    "host": "herald-postgres"
  },
  "storage": {
    "connection_string": "DefaultEndpointsProtocol=http;AccountName=heraldstore;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://herald-azurite:10000/heraldstore;"
  }
}
```

The only differences from `config.json`:
- `database.host`: `localhost` → `herald-postgres` (postgres container name)
- `storage.connection_string`: `127.0.0.1:10000` → `herald-azurite:10000` (azurite container name)

### Step 4: Create `compose/app.yml`

Create `compose/app.yml`:

```yaml
services:
  herald:
    build:
      context: ..
      dockerfile: Dockerfile
    container_name: herald-server
    environment:
      HERALD_ENV: docker
    volumes:
      - ../config.json:/app/config.json:ro
      - ../config.docker.json:/app/config.docker.json:ro
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      azurite:
        condition: service_healthy
    networks:
      - herald

networks:
  herald:
    name: herald
    driver: bridge
```

Notes:
- `HERALD_ENV=docker` triggers config overlay loading of `config.docker.json`
- Config files are bind-mounted read-only into `/app/` (the WORKDIR)
- `depends_on` with health conditions ensures postgres and azurite are ready before herald starts
- Joins the same `herald` network declared by the infrastructure compose files

Usage: `docker compose -f docker-compose.yml -f compose/app.yml up --build`

## Remediation

### R1: Compose build context and volume paths

The guide specified `context: ..` and `../config.json` paths in `compose/app.yml`, assuming paths resolve relative to the compose file's location. When using `-f docker-compose.yml -f compose/app.yml`, Docker sets the project directory to the first `-f` file's location (project root). All paths in overlay compose files resolve relative to the project root, not their own directory. Changed `..` to `.` for build context and volume mounts.

## Validation Criteria

- [ ] `docker build -t herald .` succeeds
- [ ] Container runs as non-root user (`docker run --rm herald whoami` → `herald`)
- [ ] `docker run --rm herald magick -version` shows ImageMagick 7.x
- [ ] `docker run --rm herald gs --version` shows Ghostscript version
- [ ] Docker HEALTHCHECK passes after startup
- [ ] Full stack starts via `docker compose -f docker-compose.yml -f compose/app.yml up --build`
- [ ] Web client accessible at `http://localhost:8080/app`
