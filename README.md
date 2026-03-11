# Herald

One who reads and announces markings.

Go web service for classifying PDF documents' security markings using Azure AI Foundry GPT vision models. See [`_project/README.md`](_project/README.md) for full architecture and roadmap.

## Prerequisites

- [Go](https://go.dev/) 1.26+
- [Bun](https://bun.sh/)
- [ImageMagick](https://imagemagick.org/) 7.0+ with Ghostscript (for PDF rendering)
- [Docker](https://www.docker.com/) and Docker Compose
- [Air](https://github.com/air-verse/air) (for Go hot reload in development)
- [mise](https://mise.jdx.dev/) (optional, for task runner shortcuts)

## Development

Development runs the Go server on the host with infrastructure (PostgreSQL, Azurite) in Docker.

**Start infrastructure:**

```bash
docker compose up -d
```

**Run database migrations:**

```bash
go run ./cmd/migrate -up
```

**Start the dev server (two terminals):**

```bash
# Terminal 1 — watch and rebuild the web client
cd app && bun run watch

# Terminal 2 — hot reload the Go server
air

# Terminal 2 — hot reload the Go server with Azure Entra auth
HERALD_ENV=auth air
```

The web client is available at `http://localhost:8080/app`.

## Containerized

Run the full stack entirely in Docker (app + PostgreSQL + Azurite):

```bash
docker compose -f docker-compose.yml -f compose/app.yml up --build
```

This builds the Herald Docker image and starts all services with health-conditioned dependencies. The app loads `config.docker.json` via the `HERALD_ENV=docker` overlay to resolve container hostnames.

To stop:

```bash
docker compose -f docker-compose.yml -f compose/app.yml down
```

The web client is available at `http://localhost:8080/app`.

## Tasks

Herald uses [mise](https://mise.jdx.dev/) as a task runner. All tasks can also be run directly with the underlying commands.

| Task | Command | Description |
|------|---------|-------------|
| `mise run dev` | `go run ./cmd/server` | Run the server in development mode |
| `mise run build` | `go build -o bin/server ./cmd/server` | Build the server binary |
| `mise run test` | `go test ./tests/...` | Run all tests |
| `mise run vet` | `go vet ./...` | Run go vet |
| `mise run migrate:up` | `go run ./cmd/migrate -up` | Run all up migrations |
| `mise run migrate:down` | `go run ./cmd/migrate -down` | Run all down migrations |
| `mise run migrate:version` | `go run ./cmd/migrate -version` | Print current migration version |
| `mise run web:fmt` | `cd app && bunx prettier --write client/` | Format web client source files |
| `mise run web:build` | `cd app && bun run build` | Build the web client |
| `mise run web:watch` | `cd app && bun run watch` | Watch and rebuild the web client |

## Configuration

Config loading follows a layered overlay pattern:

1. `config.json` — base configuration
2. `config.<HERALD_ENV>.json` — environment overlay (e.g., `config.docker.json`)
3. `secrets.json` — gitignored secrets
4. `HERALD_*` environment variables — final overrides

All environment variables use the `HERALD_` prefix (e.g., `HERALD_SERVER_PORT`, `HERALD_DB_HOST`).

### Entra

Azure Entra authentication is opt-in. To enable it locally, create a `config.auth.json` overlay and run with `HERALD_ENV=auth`.

**App registration setup:**

1. Register an app in Azure Entra ID (portal → App registrations → New registration)
   - Name: `herald`
   - Supported account types: Single tenant
   - Redirect URI: **SPA** platform → `http://localhost:8080/app/`

2. Expose an API (left sidebar)
   - Set Application ID URI: `api://<client-id>` (default)
   - Add a scope (e.g., `access`) — Admin and users can consent

3. API permissions (left sidebar)
   - Add your app's scope as a delegated permission (e.g., `api://<client-id>/access`)
   - Grant admin consent

4. Note the **Directory (tenant) ID** and **Application (client) ID** from the Overview page

**Create `config.auth.json`:**

```json
{
  "auth": {
    "auth_mode": "azure",
    "tenant_id": "<tenant-id>",
    "client_id": "<client-id>",
    "scope": "<scope-name>"
  }
}
```

The `scope` field is the bare scope name (e.g., `access`). The client composes the full `api://<client-id>/<scope>` format at runtime. When omitted, defaults to `access_as_user`.
