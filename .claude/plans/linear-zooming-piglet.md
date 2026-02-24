# Objective Planning: #25 — Prompts Domain

## Context

Objective #25 decomposes the Prompts Domain into executable sub-issues. The prompts domain provides CRUD management for named prompt instruction overrides targeting specific workflow stages (init/classify/enhance). This is a standard domain following the documents pattern — no blob storage, no multipart uploads, just JSON CRUD.

**Dependency:** Objective #24 (Agent Configuration and Database Schema) is 100% complete — both sub-issues (#29, #30) are closed. The `prompts` table migration exists in `000002_classification_engine.up.sql`.

## Step 0: Transition Closeout

Objective #24 is fully complete. Transition actions:

1. Close issue #24 on GitHub
2. Update `_project/phase.md` — mark #24 status as `Complete`
3. Delete `_project/objective.md` (currently documents #24)

No incomplete sub-issues to disposition.

## Architecture Decisions

### Two-Layer Prompt Composition

The final system prompt sent to the LLM is composed of two layers:

- **Instructions** (tunable) — stored in the `prompts` table, managed via CRUD API. Controls *how* the model reasons about classification.
- **Return format** (hardcoded) — defined in `workflow/prompts.go`. Controls *what shape* the model's response takes. Never exposed through the API.

At workflow execution time, the workflow assembles: `instructions + return format → final system prompt`. The prompts domain only manages the instruction layer.

### Column Naming

Rename `system_prompt` → `instructions` in the migration (`000002_classification_engine.up.sql`). This makes the two-layer architecture self-documenting: the column stores instructions, the workflow composes `instructions + output_format → system prompt`. Go field: `Instructions` / JSON: `"instructions"`.

**Migration change** (in `cmd/migrate/migrations/000002_classification_engine.up.sql`):
```sql
-- before
system_prompt TEXT NOT NULL,
-- after
instructions TEXT NOT NULL,
```

No down migration change needed (already `DROP TABLE IF EXISTS prompts`).

### Stage Validation

The `stage` column has a DB CHECK constraint (`init`, `classify`, `enhance`). The domain will also validate stage values at the application layer before hitting the DB, providing clearer error messages.

## Sub-Issue Decomposition

**One sub-issue** covers the full prompts domain. Rationale: this is a well-scoped CRUD domain following an established pattern (documents), with no external service integration. Splitting would create unnecessary overhead.

### Sub-Issue #1: Prompts Domain Implementation

**Scope:** Full CRUD domain + API wiring + tests + API docs

**Files to create:**
- `internal/prompts/prompt.go` — `Prompt` entity, `CreateCommand`, `UpdateCommand`
- `internal/prompts/errors.go` — `ErrNotFound`, `ErrDuplicate`, `ErrInvalidStage` + `MapHTTPStatus`
- `internal/prompts/mapping.go` — `ProjectionMap`, `Filters`, `FiltersFromQuery`, `scanPrompt`
- `internal/prompts/repository.go` — `repo` struct implementing `System` (List, Find, Create, Update, Delete)
- `internal/prompts/system.go` — `System` interface
- `internal/prompts/handler.go` — `Handler` struct with `Routes()` returning all endpoints
- `tests/prompts/` — black-box tests mirroring the documents test pattern

**Files to modify:**
- `cmd/migrate/migrations/000002_classification_engine.up.sql` — rename `system_prompt` → `instructions`
- `internal/api/domain.go` — add `Prompts prompts.System` to `Domain` struct + instantiation
- `internal/api/routes.go` — register prompts routes

**Files to remove:**
- `internal/prompts/doc.go` — replaced by actual implementation files

**API endpoints:**
| Method | Pattern | Handler | Description |
|--------|---------|---------|-------------|
| `GET` | `/prompts` | List | Paginated list with filters |
| `GET` | `/prompts/{id}` | Find | Single prompt by ID |
| `POST` | `/prompts` | Create | Create new prompt override |
| `PUT` | `/prompts/{id}` | Update | Update existing prompt |
| `DELETE` | `/prompts/{id}` | Delete | Remove prompt override |
| `POST` | `/prompts/search` | Search | Filtered search via POST body |

**Labels:** `feature`, `task`
**Milestone:** v0.2.0 - Classification Engine

## Additional Updates

- **`_project/README.md`** — update the `prompts` table schema to show `instructions` instead of `system_prompt`, and add a note about the two-layer composition (instructions + output format)

## Execution Plan

After approval:

1. **Transition closeout** — close #24, update phase.md, delete objective.md
2. **Create sub-issue** on GitHub with full context body
3. **Link sub-issue** to objective #25 via GraphQL
4. **Add to project board** and assign phase
5. **Create `_project/objective.md`** documenting the objective and sub-issues
