# Plan: Issue #34 — Prompts Domain Implementation

## Context

Sub-issue of #25 (Prompts Domain). Implements the full CRUD domain for named prompt instruction overrides following the documents domain pattern. Each prompt targets a specific workflow stage (init/classify/enhance) and provides tunable instructions. An `active` boolean with a partial unique index enforces at most one active prompt per stage, allowing the workflow to resolve instructions as: active prompt > hard-coded default.

## Migration

New migration `000003_prompts_active.up.sql` / `.down.sql` in `cmd/migrate/migrations/`:

- Add `active BOOLEAN NOT NULL DEFAULT false` column to `prompts` table
- Add `CREATE UNIQUE INDEX idx_prompts_active_stage ON prompts(stage) WHERE active = true`

Down migration: drop the index, drop the column.

## Files to Create

### `internal/prompts/prompt.go`

Entity and command types:

- `Prompt` struct: ID (uuid.UUID), Name (string), Stage (string), Instructions (string), Description (*string), Active (bool) — JSON-tagged
- `CreateCommand`: Name, Stage, Instructions, Description
- `UpdateCommand`: Name, Stage, Instructions, Description

Stage validation helper: `validStages` map, `validateStage(stage string) error` returning `ErrInvalidStage`

### `internal/prompts/errors.go`

- `ErrNotFound = errors.New("prompt not found")`
- `ErrDuplicate = errors.New("prompt name already exists")`
- `ErrInvalidStage = errors.New("stage must be init, classify, or enhance")`
- `MapHTTPStatus(err) int` — maps domain errors to HTTP status codes

### `internal/prompts/mapping.go`

- `projection` var: ProjectionMap for `public.prompts` with alias `p`, mapping all 6 columns (id, name, stage, instructions, description, active)
- `defaultSort`: SortField on "Name" ascending
- `Filters` struct: Stage (*string), Name (*string), Active (*bool) with `Apply(*query.Builder)` method
- `FiltersFromQuery(url.Values) Filters`
- `scanPrompt(repository.Scanner) (Prompt, error)`

### `internal/prompts/system.go`

System interface:

```
Handler() *Handler
List(ctx, page, filters) (*PageResult[Prompt], error)
Find(ctx, id) (*Prompt, error)
Create(ctx, cmd) (*Prompt, error)
Update(ctx, id, cmd) (*Prompt, error)
Delete(ctx, id) error
Activate(ctx, id) (*Prompt, error)
Deactivate(ctx, id) (*Prompt, error)
```

### `internal/prompts/repository.go`

Unexported `repo` struct with `db *sql.DB`, `logger *slog.Logger`, `pagination pagination.Config`. Public constructor `New(db, logger, pagination) System`.

Key implementation details:
- `Create`/`Update`: call `validateStage()` before DB operations
- `Activate`: within a transaction, deactivate current active for same stage (`UPDATE prompts SET active = false WHERE stage = $1 AND active = true`), then activate target (`UPDATE prompts SET active = true WHERE id = $1 RETURNING ...`)
- `Deactivate`: `UPDATE prompts SET active = false WHERE id = $1 RETURNING ...`, returns ErrNotFound if no rows affected
- All queries use `repository.QueryOne`/`QueryMany`/`WithTx`/`ExecExpectOne` + `repository.MapError`

### `internal/prompts/handler.go`

Handler struct with `sys System`, `logger *slog.Logger`, `pagination pagination.Config`.

8 endpoints via `Routes() routes.Group`:

| Method | Pattern | Handler | Description |
|--------|---------|---------|-------------|
| GET | `/prompts` | List | Paginated list with query param filters |
| GET | `/prompts/{id}` | Find | Single prompt by UUID |
| POST | `/prompts` | Create | Create new prompt |
| PUT | `/prompts/{id}` | Update | Update existing prompt |
| DELETE | `/prompts/{id}` | Delete | Remove prompt |
| POST | `/prompts/search` | Search | Filtered search via POST body |
| POST | `/prompts/{id}/activate` | Activate | Swap active prompt for stage |
| POST | `/prompts/{id}/deactivate` | Deactivate | Clear active flag |

`SearchRequest` struct embeds `PageRequest` + `Filters` (same pattern as documents).

## Files to Modify

### `internal/api/domain.go`

- Add import `"github.com/JaimeStill/herald/internal/prompts"`
- Add `Prompts prompts.System` field to `Domain`
- In `NewDomain`: construct `prompts.New(runtime.Database.Connection(), runtime.Logger, runtime.Pagination)`

### `internal/api/routes.go`

- Add `promptsRoutes := domain.Prompts.Handler().Routes()`
- Register in `routes.Register(mux, documentsRoutes, storageRoutes, promptsRoutes)`

## Files to Remove

- `internal/prompts/doc.go` — replaced by implementation files

## Key Patterns (from documents domain)

- **Constructor**: `New()` returns `System` interface, not concrete type
- **Handler factory**: `repo.Handler()` creates the Handler with the repo's own logger/pagination
- **Error mapping**: `repository.MapError(err, ErrNotFound, ErrDuplicate)` for all DB operations
- **Query building**: `query.NewBuilder(projection, defaultSort)` → WhereSearch → filters.Apply → BuildCount/BuildPage
- **Scan function**: unexported `scanPrompt` in mapping.go, used by QueryOne/QueryMany
- **JSON decode for POST bodies**: `json.NewDecoder(r.Body).Decode(&req)` in Create/Update/Search

## Differences from Documents Domain

- No blob storage dependency (simpler constructor, no compensating deletes)
- Has Update endpoint (documents are immutable)
- Has Activate/Deactivate endpoints (unique to prompts)
- Stage validation at application layer
- `active` boolean with partial unique index

## Future Note

The classification endpoint (Objectives #26/#28) should accept optional prompt ID overrides per stage for transient/exploratory use without changing the active prompt. Not in scope for this task.

## Validation Criteria

- [ ] Migration applies cleanly (`000003_prompts_active`)
- [ ] All 8 endpoints operational (List, Find, Create, Update, Delete, Search, Activate, Deactivate)
- [ ] Stage validation rejects invalid stages with 400 response
- [ ] Unique name constraint produces 409 response
- [ ] Partial unique index enforces at most one active prompt per stage
- [ ] Activate atomically swaps active prompt within a transaction
- [ ] Deactivate clears active flag, allowing fallback to hard-coded defaults
- [ ] Pagination, search, and filtering work correctly (including active filter)
- [ ] Domain wired into api.Domain and routes registered
- [ ] `go vet ./...` passes
- [ ] `go mod tidy` produces no changes
