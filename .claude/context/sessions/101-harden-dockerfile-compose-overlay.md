# 101 - Harden Dockerfile and Add Production Compose Overlay

## Summary

Hardened the Dockerfile for production deployment by adding ImageMagick, Ghostscript, and curl packages, creating a non-root `herald` user, setting `WORKDIR /app`, and adding a `HEALTHCHECK` instruction. Created a Docker Compose overlay (`compose/app.yml`) and config overlay (`config.docker.json`) for running the full stack in Docker. Updated `README.md` with development workflow, containerized usage, mise tasks, and configuration documentation.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| HEALTHCHECK probe | `curl -f` to `/healthz` | Existing endpoint, `curl` is lightweight and already added for this purpose |
| Compose overlay approach | Separate `compose/app.yml` with `-f` flag | Preserves existing dev workflow where only infra runs in Docker |
| Config overlay | `config.docker.json` with container hostnames | Image stays environment-agnostic; config mounted at runtime |
| Build context paths | `.` not `..` in compose overlay | `-f` flag sets project dir to first file's location; overlay paths resolve from project root |

## Files Modified

- `Dockerfile` — added packages, non-root user, WORKDIR, HEALTHCHECK
- `.dockerignore` — added `secrets.json`
- `config.docker.json` — new, Docker network config overlay
- `compose/app.yml` — new, Herald server compose service
- `README.md` — added development, containerized, tasks, and configuration sections

## Patterns Established

- **Compose overlay pattern**: Infrastructure compose files in `compose/` are included by `docker-compose.yml` for dev. The app overlay is opt-in via `-f` flag for full-stack Docker usage.
- **Compose path resolution**: When using `-f` overlays, all paths resolve relative to the project root (location of the first `-f` file), not relative to each compose file's directory.

## Validation Results

- `go vet ./...` — clean
- `go test ./tests/...` — all 20 packages pass
- Docker build succeeds
- Full stack starts and serves web client at `http://localhost:8080/app`
- Classification workflow completes end-to-end in containerized environment
