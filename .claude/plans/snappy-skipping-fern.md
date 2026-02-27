# Plan: Classifications Handler, API Wiring, and API Cartographer Docs (#48)

## Context

Issue #48 is the second sub-issue of Objective #27 (Classifications Domain). The classifications System, Repository, types, and errors were implemented in #47. This issue builds the HTTP layer on top: a Handler struct with 8 endpoints, API module wiring including workflow.Runtime assembly, and API Cartographer documentation.

## Approach

Follow the established handler pattern from `internal/documents/handler.go` and `internal/prompts/handler.go` exactly. The classifications domain already has its System interface, repository, types, mapping, and errors — this issue adds the Handler and wires everything together.

## Implementation

### Step 1: Add `Handler()` to `classifications.System` interface

**File:** `internal/classifications/system.go`

Add `Handler() *Handler` method to the System interface, matching the prompts pattern (no extra params — classifications has no domain-specific constructor config like documents' `maxUploadSize`).

Add `Handler()` method to `repo` in `internal/classifications/repository.go` returning `NewHandler(r, r.logger, r.pagination)`.

### Step 2: Create `internal/classifications/handler.go`

**New file.** Handler struct with `sys System`, `logger *slog.Logger`, `pagination pagination.Config`. SearchRequest type embedding `pagination.PageRequest` + `Filters`.

**Routes (8 endpoints):**

| Method | Pattern | Handler | Status |
|--------|---------|---------|--------|
| GET | `` | List | 200 |
| GET | `/{id}` | Find | 200 |
| GET | `/document/{id}` | FindByDocument | 200 |
| POST | `/search` | Search | 200 |
| POST | `/{documentId}` | Classify | 201 |
| POST | `/{id}/validate` | Validate | 200 |
| PUT | `/{id}` | Update | 200 |
| DELETE | `/{id}` | Delete | 204 |

**Endpoint patterns:**
- `List` — `PageRequestFromQuery` + `FiltersFromQuery`, calls `sys.List`
- `Find` — parse `{id}` UUID, calls `sys.Find`
- `FindByDocument` — parse `{id}` UUID, calls `sys.FindByDocument`
- `Search` — decode JSON `SearchRequest`, normalize, calls `sys.List`
- `Classify` — parse `{documentId}` UUID, calls `sys.Classify`, respond 201
- `Validate` — parse `{id}` UUID, decode `ValidateCommand`, calls `sys.Validate`
- `Update` — parse `{id}` UUID, decode `UpdateCommand`, calls `sys.Update`
- `Delete` — parse `{id}` UUID, calls `sys.Delete`, respond 204

Error handling: `MapHTTPStatus(err)` for domain errors, `http.StatusBadRequest` for parse/decode errors, `http.StatusInternalServerError` for list/search.

### Step 3: Wire classifications into API domain

**File:** `internal/api/domain.go`

1. Add `Classifications classifications.System` field to `Domain`
2. In `NewDomain`: construct `workflow.Runtime` from `runtime.Infrastructure.Agent`, `runtime.Storage`, docsSystem, promptsSystem, `runtime.Logger`
3. Construct `classifications.New(db, &wfRuntime, logger, pagination)` using the runtime
4. Add imports for `classifications` and `workflow` packages

### Step 4: Register classifications routes

**File:** `internal/api/routes.go`

Add `classificationsRoutes := domain.Classifications.Handler().Routes()` and include in `routes.Register(...)` call.

### Step 5: API Cartographer docs (AI responsibility — not in implementation guide)

Created by AI after developer execution. Not included in the guide.

## Key Files

| File | Action |
|------|--------|
| `internal/classifications/system.go` | Add `Handler()` to interface |
| `internal/classifications/repository.go` | Add `Handler()` implementation |
| `internal/classifications/handler.go` | Create (new) |
| `internal/api/domain.go` | Add Classifications + Runtime wiring |
| `internal/api/routes.go` | Register classifications routes |
| `_project/api/classifications/README.md` | Create (new) |
| `_project/api/classifications/classifications.http` | Create (new) |

## Verification

- `mise run vet` passes
- `mise run test` passes (existing tests unaffected)
- Server starts with `domain.Classifications` non-nil
- All 8 endpoints respond correctly when tested manually
