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

## Herald CLI

`cmd/herald` is a standalone, scriptable client for bulk operations against the Herald API — uploading documents, triggering classification, and pulling results back out for ingest into an external program-of-record. It is a stateless, Azure-CLI-style **primitive**: each command makes exactly one API call and emits the response as JSON to stdout. Orchestration — batching, concurrency, retries — is deliberately left to your shell scripts.

**Build:**

```bash
go build -o bin/herald ./cmd/herald
```

The release version is injected at build time (the `herald-release` workflow does this automatically from the tag; the CLI is versioned independently of the server, starting at `v0.1.0`):

```bash
go build -ldflags "-X github.com/JaimeStill/herald/internal/cli.version=v0.1.0" -o bin/herald ./cmd/herald
```

### Configuration

Configuration is resolved by merging layers from lowest to highest precedence:

1. built-in defaults (API `http://localhost:8080`, output `json`, timeout `10m`)
2. **profile** — `~/.herald/settings.json` then `~/.herald/secrets.json` (`%USERPROFILE%\.herald` on Windows)
3. **working directory** — `settings.json`, the `settings.<HERALD_CLI_ENV>.json` overlay, then `secrets.json`
4. `HERALD_CLI_*` environment variables
5. command-line flags

The profile is the base and the working directory overrides it (**local wins**), so an operator can configure `~/.herald` once and run `herald` from anywhere, while a project directory can still override per run. All variables use the `HERALD_CLI_` prefix so they never collide with the server's `HERALD_*`, and these files are distinct from the server's `config.json`/`secrets.json`. Run `herald settings show` to print the fully resolved configuration from where you stand (the client secret is redacted unless `--show-secrets` is passed).

| Global flag | Env | Description |
|-------------|-----|-------------|
| `--api <url>` | `HERALD_CLI_API` | Herald API base URL |
| `--scope <scope>` | `HERALD_CLI_SCOPE` | full Entra token scope (default `api://<client-id>/.default`) |
| `--timeout <dur>` | `HERALD_CLI_TIMEOUT` | per-request timeout (e.g. `10m`) |
| `--output json\|jsonl` | `HERALD_CLI_OUTPUT` | output format |

Auth settings use the `HERALD_CLI_AUTH_*` namespace (`_MODE`, `_TENANT_ID`, `_CLIENT_ID`, `_CLIENT_SECRET`, `_MANAGED_IDENTITY`, `_AUTHORITY`).

### Authentication

When `auth_mode` is `azure`, the CLI acquires an Entra bearer token and sends it on every request. The token's audience must be the Herald API app (`api://<client-id>`); the server validates the audience and signature only. The `--scope` / `HERALD_CLI_SCOPE` value is the **full** outbound token scope — derived as `api://<client-id>/.default` from the auth client ID when unset, and operator-overridable to handle environment quirks (e.g. the IL6 trailing-slash or `.default` vs. delegated-scope differences).

The credential is selected by `pkg/auth`: if tenant ID, client ID, and **client secret** are all present, a service-principal `ClientSecretCredential` is used; otherwise the `DefaultAzureCredential` chain (`az login`, managed identity, environment).

A `settings.auth.json` overlay supplies the azure mode, tenant, and client ID:

```json
{
  "auth": {
    "auth_mode": "azure",
    "tenant_id": "<tenant-id>",
    "client_id": "<client-id>"
  }
}
```

**Option A — interactive (`az login`, delegated token).** Uses your signed-in user identity. The universal Azure CLI client (`04b07795-8ddb-461a-bbee-02f9e1bf7b46`) must be authorized on the Herald API app registration:

- App registration → **Expose an API** → add a delegated scope (e.g. `access_as_user`).
- Under **Authorized client applications**, add the Azure CLI client ID with that scope, and grant admin consent.

```bash
az login --tenant <tenant-id>
HERALD_CLI_ENV=auth ./bin/herald documents list
```

**Option B — service principal (client secret, app-only token).** The credential the production operator uses. Put the azure `auth` block in `~/.herald/settings.json`, then store the secret with `settings secret` so it never lands in shell history — it accepts the secret on stdin, which pairs well with Azure Key Vault:

```bash
az keyvault secret show --vault-name <vault> --name <secret> --query value -o tsv | herald settings secret
# writes ~/.herald/secrets.json (0600) → { "auth": { "client_secret": "..." } }

herald documents list
```

The secret can also be passed as a positional argument (`herald settings secret <value>`) or written to a gitignored `secrets.json` by hand: `{ "auth": { "client_secret": "<value>" } }`.

If the secret belongs to the **same** app registration as the API, the derived `api://<client-id>/.default` scope is already correct. If it belongs to a **separate** client app, set `client_id` to that client and override the scope to the API's audience: `HERALD_CLI_SCOPE=api://<api-client-id>/.default`. App-only `.default` token issuance may require an Application-type **app role** on the API granted to the client SP.

### Commands

```
herald <command> [flags] [args]
```

| Command | Description |
|---------|-------------|
| `documents upload --file <pdf> --external-id <n> --platform <p>` | Upload one file with its external-system linkage; emits the created `Document` |
| `documents list [filters]` | One page of documents; emits the raw paginated result (`--status`, `--platform`, `--external-id`, `--classification`, `--page`, `--page-size`, …) |
| `documents get <id>` | Fetch a single document |
| `classify <document-id>` | Trigger classification, consume the SSE stream to completion, emit the resulting `Classification` |
| `classifications list [filters]` | One page of classifications (`--classification`, `--confidence`, `--document-id`, `--page`, …) |
| `classifications get <id>` | Fetch a single classification |
| `classifications by-document <document-id>` | Fetch a document's classification |
| `settings show [--show-secrets]` | Print the fully resolved settings from all sources; the client secret is redacted unless `--show-secrets` |
| `settings secret [<secret>\|-]` | Write the Entra client secret to `~/.herald/secrets.json`; reads stdin when the argument is omitted or `-` |
| `version` | Print the CLI version |

The `external_id` + `external_platform` pair set on upload round-trips on every document response, serving as the join key back to the program-of-record — no separate correlation file is needed.

### Scripting

Because each command is a single API call, batching and concurrency live in your scripts. Example bulk upload → classify → retrieve, capping concurrency with `xargs -P`:

```bash
# Upload a batch (4 at a time), capturing each created document id
jq -c '.[]' batch.json | xargs -P4 -I{} bash -c '
  item={}
  ./bin/herald documents upload \
    --file "$(jq -r .file <<<"$item")" \
    --external-id "$(jq -r .external_id <<<"$item")" \
    --platform "$(jq -r .external_platform <<<"$item")"
' | jq -r .id > ids.txt

# Classify each (2 at a time — one request fans out across pages server-side)
xargs -P2 -a ids.txt -I{} ./bin/herald classify {} > classifications.jsonl

# Pull results for ingest
./bin/herald documents list --status review --output jsonl
```
