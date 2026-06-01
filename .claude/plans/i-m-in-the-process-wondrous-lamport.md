# Mise tasks for demo workflow

## Context

Preparing for a capability demonstration after time away from the project. The
existing `.mise.toml` covers building, testing, and the individual moving parts
(`migrate:up`, `web:watch`, plus a server-only `dev` that just runs
`go run ./cmd/server` with no live reload), but there's no single-command way to
bring the local stack up with migrations applied, or to run the full dev loop
(Go server via `air` + web client via `bun run watch`) together. This adds
ergonomic lifecycle tasks so the demo can be driven with a handful of memorable
commands.

Verified facts that shape the design:
- Both `air` (`go/1.26.1/bin/air`) and `bun` (`bun/1.3.14/bin/bun`) are already
  installed via mise tools — no new tool entries needed.
- Both compose services have healthchecks (`compose/postgres.yml` →
  `pg_isready`, `compose/azurite.yml` → `nc -z … 10000`), so
  `docker compose up -d --wait` blocks until the stack is actually ready before
  migrations run.
- `docker-compose.yml` is an aggregator (`include: compose/postgres.yml,
  compose/azurite.yml`), so plain `docker compose …` from the repo root drives
  the whole stack.
- `cmd/migrate/main.go` takes flags (`-up`/`-down`/`-version`) and defaults its
  DSN to `postgres://herald:herald@localhost:5432/herald?sslmode=disable` —
  matching the compose Postgres — so migration is env-agnostic (auth mode only
  changes auth config, not the DB).
- `HERALD_ENV=auth` selects the `config.auth.json` overlay; `air` and the bun
  watcher both inherit the task's environment, so setting it once at task level
  propagates to the spawned server and client.
- **Yes — a single mise task can run two long-lived processes**: a shell `run`
  body that backgrounds both and `trap`s cleanup on exit is the reliable
  pattern (mise's `depends` is designed for prerequisites that *complete*, which
  doesn't fit never-exiting watchers).

## Approach

Edit `/home/jaime/code/herald/.mise.toml`. Keep all existing tasks except the
server-only `dev`, which gets **replaced** by the combined watcher (per the
chosen naming). Add the `stack:*` lifecycle tasks and the `dev:*` watcher tasks.

### Stack lifecycle (requirements 1–3)

```toml
[tasks."stack:up"]
description = "Start the Docker stack (Postgres + Azurite) and apply migrations"
run = '''
set -e
docker compose up -d --wait
go run ./cmd/migrate -up
'''

[tasks."stack:down"]
description = "Stop the Docker stack"
run = "docker compose down"

[tasks."stack:reset"]
description = "Purge the Docker stack (containers, volumes, networks, images, orphans)"
run = "docker compose down -v --rmi all --remove-orphans"
```

`--wait` guarantees Postgres is healthy before `migrate -up` runs, so no sleep
or retry loop is needed. `-up` is idempotent (`cmd/migrate` swallows
`migrate.ErrNoChange`), so re-running `stack:up` is safe.

### Dev loop (requirements 4–5)

Replace the existing `[tasks.dev]` (server-only `go run`) with a combined
watcher, plus an auth variant. The `trap 'kill 0' …` line tears down both child
processes (and air's spawned `bin/server`) when the task is interrupted with
Ctrl-C.

```toml
[tasks.dev]
description = "Run the full dev loop: Go server (air live-reload) + web client (bun watch)"
run = '''
set -e
trap 'kill 0' EXIT INT TERM
(cd app && bun run watch) &
air &
wait
'''

[tasks."dev:auth"]
description = "Run the full dev loop with HERALD_ENV=auth (Entra auth overlay)"
env = { HERALD_ENV = "auth" }
run = '''
set -e
trap 'kill 0' EXIT INT TERM
(cd app && bun run watch) &
air &
wait
'''
```

Note: `air` (`.air.toml`) watches `cmd/internal/pkg/app` for `go/html/js/css`,
so when `bun run watch` regenerates the client bundle, air rebuilds and
restarts the embedded server — the two watchers compose correctly.

### Stop watchers (requirement 6)

Safety-net task to kill any running dev processes regardless of which variant
started them (covers orphans left if a terminal was closed instead of Ctrl-C'd):

```toml
[tasks."dev:stop"]
description = "Stop air, the bun web watcher, and the air-built server for any running dev loop"
run = '''
pkill -x air || true
pkill -f 'scripts/watch.ts' || true
pkill -f 'bin/server' || true
true
'''
```

`pkill -x air` matches the `air` process by exact name; `scripts/watch.ts` is
what `bun run watch` actually execs; `bin/server` is air's compiled child. Each
`|| true` keeps the task green when a given process isn't running, and the
trailing `true` guarantees a zero exit.

## Files

- `/home/jaime/code/herald/.mise.toml` — only file changed: remove the
  server-only `[tasks.dev]`, add `stack:up`/`stack:down`/`stack:reset`,
  `dev` (combined), `dev:auth`, `dev:stop`. Existing `build`, `test`, `vet`,
  `migrate:*`, `web:*` tasks are left untouched.

## Verification

1. `mise tasks` — confirm the new tasks are listed with descriptions.
2. `mise run stack:up` — Postgres + Azurite come up healthy and
   `migrations applied successfully` prints. Re-run to confirm idempotency
   (`no change`-style clean exit). Check `mise run migrate:version`.
3. `mise run dev` — both processes start; the web bundle rebuilds and the server
   serves on its port. Ctrl-C cleanly stops both (verify with `pgrep -x air`
   and `pgrep -f scripts/watch.ts` → nothing).
4. In a separate terminal while `dev` runs, `mise run dev:stop` — confirms both
   watchers and `bin/server` are killed.
5. `mise run dev:auth` — server starts under the auth overlay (look for the
   auth/Entra config taking effect in startup logs vs. plain `dev`).
6. `mise run stack:down` then `mise run stack:reset` — stack stops; reset leaves
   no `herald-*` containers, `herald-postgres`/`herald-azurite` volumes, or
   images (`docker ps -a`, `docker volume ls`).
