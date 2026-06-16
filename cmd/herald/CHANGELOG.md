# Herald CLI Changelog

The `herald` CLI is versioned independently of the Herald server. Release tags are
prefixed `herald-v*` (e.g. `herald-v0.1.0`) and draw their notes from this file.

## v0.1.0

Initial release of the `herald` command-line client — a stateless, Azure-CLI-style primitive over the Herald API for scriptable bulk operations (upload, classify, retrieve). Each command makes a single API call and emits the response as JSON; orchestration (batching, concurrency, retries) is left to the caller's scripts.

### Commands

- `documents upload --file --external-id --platform` — upload one file with its external-system linkage; emits the created `Document`
- `documents list` / `documents get <id>` — query documents (status/platform/external-id/classification filters, pagination); list emits the raw paginated result for script-side iteration
- `classify <document-id>` — trigger classification, consume the Server-Sent Events stream to completion, and emit the resulting `Classification`
- `classifications list` / `get <id>` / `by-document <document-id>` — retrieve classification results for ingest; the `external_id`/`external_platform` pair is the join key back to the program-of-record
- `settings show [--show-secrets]` — print the fully resolved settings from all sources; the client secret is redacted unless `--show-secrets`
- `settings secret [<secret>|-]` — write the Entra client secret to `~/.herald/secrets.json` (`0600`); reads stdin when the argument is omitted or `-`, so it can be piped from `az keyvault secret show`
- `version`, `help`

### Authentication

- Azure Entra bearer-token auth (`auth_mode: azure`), acquired per request and served from the `azidentity` token cache
- Credential resolution via `pkg/auth`: a service-principal `ClientSecretCredential` when tenant, client, and secret are all set, otherwise the `DefaultAzureCredential` chain (`az login`, managed identity, environment)
- Operator-overridable token scope (`--scope` / `HERALD_CLI_SCOPE`), derived as `api://<client-id>/.default` when unset

### Configuration

- Per-user profile at `~/.herald/` (`%USERPROFILE%\.herald` on Windows), layered profile-base then working-directory (local wins), then `HERALD_CLI_*` environment variables, then flags
- `json` (default) and `jsonl` output formats
- All variables use the `HERALD_CLI_` prefix, distinct from the server's `HERALD_*`
