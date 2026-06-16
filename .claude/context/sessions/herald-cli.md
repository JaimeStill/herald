# Herald CLI — Bulk Upload & Classification Client

> One-off session (not issue-driven). Plan: `.claude/plans/now-that-we-have-lively-fiddle.md`.

## Summary

Added `cmd/herald`, a standalone command-line client for bulk operations against the Herald API — upload documents, trigger classification, and retrieve results for ingest into an external program-of-record. Logic lives in the importable `internal/cli` package (black-box testable) with a thin `cmd/herald/main.go` entry point.

The CLI is a stateless, Azure-CLI-style **primitive**: each command makes exactly one API call and emits the response as JSON to stdout. Orchestration (batching, concurrency, retries) is deliberately left to the caller's scripts. Shipped with CI (test + lint), a dedicated release workflow, and an independent version line starting at `v0.1.0`.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| CLI shape | 1:1 API primitive, no orchestration | Scripts own batching/concurrency; auto-retrying mutating POSTs risks duplicate documents/re-triggered classifications; the model-call rate-limit surface is already retried server-side |
| Package layout | `internal/cli` + thin `cmd/herald/main.go` | Black-box testable; mirrors how `cmd/server` stays thin |
| Env prefix | `HERALD_CLI_*` (incl. `HERALD_CLI_AUTH_*`) | Never collides with the server's `HERALD_*` |
| Config precedence | profile-base, local-wins | `~/.herald` configured once works from anywhere; a project dir can still override per run |
| Auth | reuse `pkg/auth` `TokenCredential()` | SP `ClientSecretCredential` when tenant+client+secret set, else `DefaultAzureCredential` chain |
| Token scope | `--scope`/`HERALD_CLI_SCOPE`, derived `api://<client-id>/.default` | Operator-overridable for the IL6 trailing-slash / `.default`-vs-delegated quirks |
| `settings show` secret | `<redacted>` unless `--show-secrets` | Avoid leaking secrets to logs/screens; redaction centralized in `redactSecrets` |
| `settings secret` input | positional arg **or** stdin | Pipe from `az keyvault secret show` without touching shell history |
| Versioning | independent `herald-v*` line, start `v0.1.0`, own `cmd/herald/CHANGELOG.md` | The CLI ships on its own cadence, separate from the server's `v0.6.0` line (deliberate exception to phase-target alignment) |
| Release targets | `linux/amd64` + `windows/amd64` (amd64 only) | DBA runs it from Windows 11; pure-Go cross-compile, no cgo |
| Lint findings | fix all, no `.golangci.yml` | Clean repo under golangci-lint v2.12.2 defaults rather than suppressing |

## Files Modified

- **New CLI**: `cmd/herald/{main.go,settings.json,settings.auth.json,CHANGELOG.md}`, `internal/cli/{cli,config,client,documents,classify,settings,output}.go`
- **Tests**: `tests/cli/{helpers,config,client,settings,dispatch}_test.go`
- **Workflows**: `.github/workflows/{ci.yml,herald-release.yml}`
- **Docs**: `README.md` (Herald CLI section), `CLAUDE.md` (versioning exception note)
- **Lint cleanup** (behavior-preserving errcheck/QF1008): `internal/api/{api,storage}.go`, `internal/{classifications,documents,prompts}/handler.go`, `internal/format/format.go`, `internal/workflow/workflow.go`, `pkg/{handlers/handlers,middleware/auth,repository/repository,web/static}.go`, plus several `tests/*` files

## Patterns Established

- **Local (non-API) command group** (`settings`) alongside the API command groups — a scoped exception to the primitive ethos for config introspection/management.
- **Per-user profile config** at `~/.herald` layered under working-directory config via the existing `Merge` machinery.
- **Independent tool versioning** within the repo: `herald-v*` tags + `cmd/herald/CHANGELOG.md`, decoupled from the server's phase version.

## Validation Results

- **Local stack (non-auth)**: upload → classify → retrieve round-trip verified; surfaced + fixed the classify SSE bug (server streams the full `ExecutionEvent` envelope; the classification is in the nested `data`).
- **`dev:auth`**: both credential flows validated end-to-end (delegated `az login` after pre-authorizing the Azure CLI client + `access_as_user` consent; SP client-secret app-only token) across GET, multipart upload, and SSE classify. Server validates audience + signature only.
- **Tests**: 29 black-box tests in `tests/cli`, all pass; full repo suite passes with no infra.
- **Quality gates**: `golangci-lint run ./...` = 0 issues; `go vet ./...`, `go build ./...` clean; release cross-compile verified (Windows = PE32+ x86-64), `herald version` → `herald v0.1.0`.
- **Not yet exercised**: commercial Azure (final pre-release check); IL6 (post-release); first CI/release workflow runs on GitHub.
