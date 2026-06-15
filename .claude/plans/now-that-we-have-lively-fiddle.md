# Herald CLI — Bulk Upload & Classification Client

> **Implementation Progress** is tracked at the bottom of this file. On resume after a
> context clear, read that section first, then `git log --oneline main..feat/herald-cli`.

## Context

Herald is now deployed on IL6. The next operational need is **bulk ingest**: a DBA must be
able to push many documents through Herald (upload → classify), then pull the inferred
classification data back out so it can be loaded into an external program-of-record database.
Doing this by hand through the web client does not scale, and a naive shell loop hitting the
API in parallel would trip Azure OpenAI token/rate limits and Herald's own throughput ceilings.

This plan introduces a new `cmd/herald` CLI — a sibling to `cmd/migrate` — that authenticates
to the Herald API with Entra and exposes scriptable, stateless commands for bulk operations.

### Decisions locked in (from planning Q&A)

- **Auth**: reuse `pkg/auth` `Config.TokenCredential()` (service-principal client secret, else
  `DefaultAzureCredential` chain). Non-interactive, scriptable.
- **Shape**: Azure-CLI-style. Separate one-shot commands, **no intermediary state files**.
  Inputs come from flags / stdin; results go to **stdout as JSON**. The consumer decides how
  to feed data in and process data out.
- **Concurrency is a CLI responsibility**: batch commands cap in-flight concurrency and retry on
  `429`/`5xx` (honoring `Retry-After`) *internally*, so the DBA cannot accidentally overload Azure
  or Herald by fanning out a shell loop. A concurrency cap plus 429-aware retry is the throttle —
  no separate client-side rate limiter (see Throttle & Concurrency Design).
- **Keep it lean**: this CLI orchestrates a fixed, small set of endpoints and returns structured
  JSON. Favor the simplest thing that meets the throttle requirement over general infrastructure.
- **Name**: `cmd/herald`, binary `herald`, release tag `herald-v*`.

## Goals / Scope

In scope:
- `herald documents upload` — batch upload (concurrency-capped, retried)
- `herald documents list` / `get` — query documents (for retrieval/extraction)
- `herald classify` — batch-trigger classification, consume the SSE stream to completion,
  emit the resulting classification JSON (concurrency-capped, retried)
- `herald classifications list` / `get` / `by-document` — pull classification results for ingest
- Thin HTTP client with Entra token acquisition (SDK-cached), concurrency cap, and 429-aware retry
- `herald-release.yml` workflow + `.mise.toml` tasks

Out of scope (note as future): `validate` / `update` commands (human review happens in the web
UI), `delete`, and any combined pipeline command.

## Throttle & Concurrency Design (the core concern)

A single `classify` request is **not** a single unit of Azure load: the server's workflow fans
out across document pages internally (`internal/workflow/classify.go` uses
`errgroup.SetLimit(core.WorkerCount(len(pages)))`) and each page is a vision-model call. So N
concurrent CLI classify requests ≈ N × pages concurrent Azure OpenAI calls. Uploads are lighter
(blob + DB) but still bounded by Herald throughput. The CLI defends with **two** simple layers:

1. **Concurrency cap** — max in-flight requests for a batch, via `errgroup.WithContext` +
   `g.SetLimit(n)` (the house pattern, see `internal/workflow/classify.go`). This is the primary,
   intuitive throttle knob. Per-command defaults: `documents upload` → `4`; `classify` → `2`
   (conservative because of the page fan-out multiplication). Overridable via `--concurrency`.
2. **429-aware retry** — on `429`/`5xx`, honor the `Retry-After` header when present, otherwise
   exponential backoff (`retry_base_delay`..`retry_max_delay`) with full jitter, capped by
   `--max-retries` (default 5). `4xx` other than 429 fails fast. This is what makes the CLI
   rate-limit-aware: when the server pushes back, the client waits as instructed and retries.

This deliberately drops a separate client-side token-bucket rate limiter (`golang.org/x/time/rate`)
and the manual token cache that earlier drafts carried. The concurrency cap governs how hard we
push; the 429 retry handles overrun per the server's own guidance; `azidentity` already caches
tokens internally. Net effect: meaningfully less client machinery, one fewer dependency, same
rate-limit-aware behavior.

Failure isolation: a batch never aborts on a single item's terminal failure. Each item's outcome
(success payload **or** error string) is captured and emitted in the JSON results array, so a
partially-failed run is still usable and re-runnable for just the failures. On `SIGINT`/`SIGTERM`
the shared context is cancelled: no new work is launched, in-flight work is cancelled, and
partial results are flushed.

## Command Surface (Azure-CLI style)

```
herald [global flags] <group> <command> [flags] [args]

global flags (also env, HERALD_CLI_ prefix):
  --api            HERALD_CLI_API           base URL, e.g. https://herald.<il6-host>
  --scope          HERALD_CLI_SCOPE         Entra token scope (default api://{client-id}/.default)
  --concurrency    HERALD_CLI_CONCURRENCY   max in-flight requests (per-command default)
  --max-retries    HERALD_CLI_MAX_RETRIES   retry cap on 429/5xx (default 5)
  --timeout        HERALD_CLI_TIMEOUT       per-request timeout (default 10m)
  --output         json (default) | jsonl   array vs newline-delimited streaming
  (retry backoff: HERALD_CLI_RETRY_BASE_DELAY / _MAX_DELAY; auth via HERALD_CLI_AUTH_* — see below)
```

- `herald documents upload` — input is a JSON array on **stdin** (or `--items @file.json`):
  `[{"file":"a.pdf","external_id":1,"external_platform":"PLATFORM"}, ...]`. For the trivial
  single case, `--file a.pdf --external-id 1 --platform P`. Builds the `multipart/form-data`
  request (`file`, `external_id`, `external_platform`) per
  `internal/documents/handler.go:Upload`. Emits a JSON array of `{external_id, file, document,
  error}` where `document` is the created `Document` (201 body).
- `herald documents list` — filters as flags (`--status`, `--platform`, `--classification`,
  `--external-id`, …) mapped to `GET /api/documents` query params; auto-paginates and emits the
  flattened JSON array (or `--page`/`--page-size` for a single page).
- `herald documents get <id>` — `GET /api/documents/{id}`.
- `herald classify <documentId>...` — document IDs from args **or** stdin (JSON array / newline
  list), **or** `--status pending [--platform P]` to query candidates then classify them. For
  each, `POST /api/classifications/{documentId}`, consume the `text/event-stream`, and resolve on
  the terminal `complete` event (carries the full `Classification`) or fail on `error`. Emits a
  JSON array of `{document_id, classification, error}`.
- `herald classifications list` / `get <id>` / `by-document <id>` — `GET /api/classifications`,
  `/{id}`, `/document/{id}`. The retrieval/extraction surface; the `Classification` JSON already
  carries everything the program-of-record needs: `classification`, `confidence`,
  `markings_found`, `rationale`, `classified_at`, `model_name`, `provider_name`, `document_id`
  (join key back to `external_id`/`external_platform` via the document).

The `external_id` + `external_platform` round-trip (set on upload, returned on document
get/list) is the join key for ingest — no separate correlation file is needed.

## Package Layout — `internal/cli/` (+ thin `cmd/herald/main.go`)

Logic lives in the importable `internal/cli` package (black-box testable); `cmd/herald/main.go`
is a thin `package main` that calls `cli.Run(os.Args[1:])`, mirroring how `cmd/server` stays thin.

| File | Responsibility |
|------|----------------|
| `cmd/herald/main.go` | Thin entry: `os.Exit(cli.Main())` / `cli.Run(args)` |
| `cli.go` | Subcommand dispatch via `flag.FlagSet` per command (no cobra — matches `cmd/migrate` minimalism); binds global flags into a `*Settings` overlay; signal-cancel context; `version` |
| `config.go` | `Settings` (API, scope, concurrency, retry knobs, `auth.Config`) via `settings.json`+overlay+`secrets.json`+`HERALD_CLI_*` env+flags; `Load`/`Finalize`/`Merge`/`validate` **(done)** |
| `client.go` | `Client` wrapping `*http.Client` + base URL + `azcore.TokenCredential`; `do()` injects `Authorization: Bearer` (token via `cred.GetToken`, SDK-cached) and runs 429-aware retry/backoff; `getJSON` + `postMultipart` + `postStream` helpers. No rate limiter, no manual token cache. Methods own a per-call `context.WithTimeout` and `classify` consumes the SSE stream to completion internally (no streaming-body escape) |
| `batch.go` | `RunBatch[T,R]` — bounded-concurrency runner via `errgroup.WithContext` + `SetLimit`, per-item result-or-error in input order **(done)** |
| `documents.go` | `upload`, `list`, `get` commands + their `*Client` API methods; resolves per-command concurrency default |
| `classify.go` | `classify` command + SSE consumer that resolves on `complete`/`error`; classification `list`/`get`/`by-document` |
| `output.go` | Emit `json` array or `jsonl` to stdout; per-item error envelope; human-readable errors to stderr |

### Auth resolution (reuse, with one addition)

- Credential: `cfg.Auth.TokenCredential()` — service-principal (`HERALD_AUTH_CLIENT_ID` /
  `HERALD_AUTH_CLIENT_SECRET` / `HERALD_AUTH_TENANT_ID`) or `DefaultAzureCredential` chain
  (`az login` / managed identity / env). Reused verbatim from `pkg/auth/config.go`.
- API token **scope**: the outbound token's audience must be the Herald API app
  (`api://{herald-api-client-id}`). Add a `--scope` / `HERALD_API_SCOPE` setting (default derived
  as `api://{cfg.Auth.ClientID}/.default`) so the operator can point the SP credential at the
  Herald API resource. The client calls
  `cred.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{scope}})` per request and relies
  on `azidentity`'s internal token cache (no manual caching layer). Note the IL6 **trailing-slash**
  scope gotcha (see `il6_deployment_lessons` memory) — make the exact scope string
  operator-overridable, don't hardcode the suffix. Scope env var is `HERALD_CLI_SCOPE`.

### Reused building blocks (do not reinvent)

- `pkg/auth` `Config` / `TokenCredential()` / `Env` — credential + env override pattern.
- Token acquisition pattern: `cred.GetToken(ctx, policy.TokenRequestOptions{Scopes: ...})` as in
  `pkg/database/database.go:NewWithCredential`.
- `internal/workflow/classify.go` + `pkg/core/workers.go` (`WorkerCount`) — the
  `errgroup.SetLimit` bounded-fan-out idiom to mirror in `batch.go`.
- Domain models `internal/documents/document.go` (`Document`) and
  `internal/classifications/classification.go` (`Classification`) — decode targets for responses.
- SSE contract from `internal/classifications/handler.go:Classify` — `event: <type>` /
  `data: <json>`; resolve on the `complete` event, fail on `error`.

### Dependencies

- No new dependencies. `golang.org/x/sync` (errgroup) and `azidentity`/`azcore` are already
  present. The `golang.org/x/time/rate` dependency added in an earlier draft is **removed** (the
  token-bucket limiter is gone) — drop it from `go.mod` and `go mod tidy`.

## Release & Tooling

- `.github/workflows/herald-release.yml` — clone `migrate-release.yml`: trigger on `herald-v*`,
  build `CGO_ENABLED=0` for `linux/amd64` and `windows/amd64` (add `darwin/arm64` if the DBA runs
  macOS), `create-gh-release-action` with `prefix: herald-`, `gh release upload`.
- `.mise.toml` — add `herald:build` (`go build -o bin/herald ./cmd/herald`) and convenience
  passthroughs (`herald:upload`, `herald:classify`) following the existing `migrate:*` style.
- Versioning per project convention — align `herald version` output and any release tag with the
  current `_project/phase.md` target.

## Testing (AI responsibility, `tests/cli/`, black-box `package cli_test`)

- `httptest.Server` standing in for the Herald API. Table-driven coverage of:
  - **Retry/backoff**: server returns `429` with `Retry-After`, then `200` — assert the client
    waits and succeeds; `5xx` exhausts `--max-retries` and surfaces a terminal error; non-429
    `4xx` fails fast without retry.
  - **Concurrency cap**: instrument the test server to record max simultaneous in-flight requests;
    assert it never exceeds `--concurrency` (`RunBatch`).
  - **Batch isolation**: one item errors, others succeed → results carry both, exit is non-fatal.
  - **SSE consumer**: a streamed `node.start`/`complete` sequence resolves to the embedded
    `Classification`; an `error` event surfaces as the item error.
  - **Config precedence**: defaults → file → env → flags override order; scope derivation.
  - **Output**: `json` array vs `jsonl` shape; error-envelope fields.
- Godoc on all exported `internal/cli` types/functions added post-implementation.

## Verification (end-to-end)

Testing progression (precedent): **local docker compose stack first → commercial Azure → IL6.**
Get the CLI fully working locally before any cloud test; commercial Azure is the final
pre-release check; IL6 is tested last, after release. Never test IL6-first.

1. Build: `mise run herald:build` and `mise run vet`; run `mise run test`.
2. Local stack (primary test bed): `mise run stack:reset` → `mise run stack:up` →
   `mise run dev:auth` (exercises the Entra token path against the local Azurite/Postgres stack).
3. Upload a small batch:
   `echo '[{"file":"_project/.../sample.pdf","external_id":1,"external_platform":"TEST"}]' | bin/herald documents upload --api-url http://localhost:8080`
   → expect a JSON array with a created `Document` (status `pending`).
4. Classify: `bin/herald classify --status pending --api-url http://localhost:8080 --concurrency 2`
   → expect classification JSON per document; confirm documents move to `review`.
5. Retrieve for ingest: `bin/herald classifications by-document <id>` and
   `bin/herald documents list --status review` → confirm `external_id`/`external_platform` join
   key plus `classification`/`confidence`/`markings_found` are present.
6. Throttle check: confirm `--concurrency` caps in-flight work and that a `429` with `Retry-After`
   is honored (covered in tests; spot-check manually).
7. Auth path: against `dev:auth`, set `HERALD_CLI_AUTH_*` + `HERALD_CLI_SCOPE` and confirm a bearer
   token is acquired and accepted.
8. Commercial Azure: repeat the upload → classify → retrieve flow against the commercial
   deployment as the final pre-release check.
9. IL6: only after release, validate on IL6 last.

## Open considerations (flag during implementation, not blockers)

- Exact API **scope** string for IL6 (trailing slash / `.default` vs `access_as_user`) — confirm
  against the deployed app registration; keep it operator-overridable.
- Whether the SP used by the DBA is the **same** app registration as the Herald API (ClientID
  serves both) or a distinct client (separate credential vs scope) — the `--scope` split above
  supports either, but the default assumes same-app.

---

## Implementation Progress

Resume protocol after a context clear: read this section, then `git log --oneline main..feat/herald-cli`,
`git diff --stat main...feat/herald-cli`, and the current `internal/cli/*.go`. Pick up at **Next**.

### Layout (refines the approved plan)

The implementation lives in an importable **`internal/cli`** package (config, client, batch,
commands, output, version) with a thin `cmd/herald/main.go` entry point — mirrors how `cmd/server`
stays thin with logic in `internal/`, and makes the code black-box testable (`tests/cli/`,
`package cli_test`). This supersedes the plan's "everything in `cmd/herald` package main".

### Decisions made during implementation

- Package `internal/cli`; thin `cmd/herald/main.go` calls into it.
- **Env prefix `HERALD_CLI_`** for every variable (incl. auth: `HERALD_CLI_AUTH_*`) so nothing
  collides with the server's `HERALD_*`. Field↔var mapping via an `Env` struct (`settingsEnv`,
  `authEnv`).
- Config files: **`settings.json`** base → `settings.<HERALD_CLI_ENV>.json` overlay →
  `secrets.json`, distinct from the server's `config.json`. `Load(flags)` tolerates missing files.
- **Precedence:** built-in defaults → settings.json → overlay → secrets.json → env → **CLI flags**.
  Flags are bound into a zero-valued `*Settings` overlay and merged inside `Finalize(env, aenv, flags)`
  right before `validate()` — single validation pass, flags reuse the same `Merge` (zero = unset, at
  every layer consistently).
- `Output` is a typed `OutputFormat` enum (`OutputJSON`/`OutputJSONL`), house-consistent with
  `LogLevel`/`auth.Mode`.
- **Simplification (this revision):** dropped the client-side token-bucket rate limiter
  (`golang.org/x/time/rate`, `Rate`/`Burst` settings, `NewLimiter`) and the manual token cache.
  Throttle = concurrency cap + 429-aware retry only; tokens are SDK-cached. This cuts `client.go`
  roughly in half and removes a dependency. Rationale: fixed-endpoint orchestration CLI, keep it lean.
- **Per-command concurrency default lives in the command**, not in `Settings` (no `throttle`/`resolve`
  helper). Unset `Concurrency` (0) → command default (upload 4 / classify 2; classify lower because
  one request fans out across pages server-side). `RunBatch` clamps concurrency < 1 to 1.
- `NewClient(s *Settings)` — no limiter param. `do()` uses the call's ctx directly (no per-attempt
  context, no `cancelReadCloser`); each public method wraps a single `context.WithTimeout`, and
  `Classify` consumes the SSE stream to completion internally (returns `*Classification`, never an
  open body).
- `RunBatch[T,R]` (batch.go): `errgroup.WithContext` + `SetLimit`; item failures isolated (recorded
  per `BatchResult`, never cancel the batch); ctx cancel → partial-but-usable results.
- Reuse `internal/documents.Document` and `internal/classifications.Classification` as decode
  targets (same module). `complete` SSE event's `data` is the full `Classification`.
- Testing precedent: **local docker stack (`stack:reset`/`stack:up`/`dev:auth`) → commercial Azure
  → IL6**, never IL6-first. (memory `feedback_cli_test_progression`)

### Status

- [x] Branch `feat/herald-cli` (checkpoint #1: `6f594d6`; checkpoint #2: simplification)
- [x] `internal/cli/batch.go` — `RunBatch` + `BatchResult` (reviewed; unchanged by simplification)
- [x] `internal/cli/config.go` — trimmed: `Rate`/`Burst` removed from struct/`Env`/`settingsEnv`/
      `Merge`/`loadEnv`. Rest intact (API, Scope, Concurrency, MaxRetries, RetryBaseDelay/MaxDelay,
      Timeout, Output, Auth). Reviewed; builds + vets.
- [x] `internal/cli/client.go` — leaner: no limiter/token-cache/per-attempt-ctx/`cancelReadCloser`;
      `NewClient(s)` only; `authorize` calls `cred.GetToken` directly. Transport helpers
      (`getJSON`/`postMultipart`/`postStream`) stay **unexported** — public API is the typed
      per-endpoint methods (decided: `UploadDocument`/`Classify`/`ListDocuments`/`GetDocument`/
      classification getters) added in documents.go/classify.go. Reviewed; builds + vets.
- [x] Removed `golang.org/x/time/rate` from go.mod (`go mod tidy`)
- [ ] **Next:** `internal/cli/output.go` — emit json array / jsonl to stdout
- [ ] `internal/cli/documents.go` — upload (stdin JSON array or single-flag), list, get; + exported
      `*Client` domain methods; per-command concurrency default
- [ ] `internal/cli/classify.go` — classify (SSE consumer) + classifications list/get/by-document
- [ ] `internal/cli/cli.go` + `cmd/herald/main.go` — subcommand dispatch, global flags→overlay,
      signal-cancel context, `version`
- [ ] `.github/workflows/herald-release.yml` (`herald-v*`) + `.mise.toml` `herald:build`
- [ ] Tests in `tests/cli/` (retry/backoff, concurrency cap, batch isolation, SSE, config
      precedence, output); godoc pass
- [ ] Verify against local stack → commercial Azure

Checkpoint cadence: update this Status block + commit (WIP, squash at PR) at each reviewed unit.
