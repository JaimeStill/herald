# Project Review — Herald Post-Deployment QoL (2026-04-21)

## Context

Herald reached v0.4.0 on 2026-03-13 (Phase 4 — Security and Deployment) and was successfully deployed to Azure IL6. With the service now in real use, five quality-of-life issues have surfaced. They were captured by the user in issue [#132](https://github.com/JaimeStill/herald/issues/132):

1. `document-upload` module overflows without a scroll container.
2. Page size is not adjustable (fixed at 12) — UI needs a selector.
3. Pagination, search, filter, and sort state is lost when navigating to review and back — needs URL query-param persistence.
4. Migrate from `github.com/JaimeStill/go-agents@v0.4.0` and `github.com/JaimeStill/go-agents-orchestration@v0.3.3` to `github.com/tailored-agentic-units/agent` and `github.com/tailored-agentic-units/orchestrate` (the user forked the tau libraries directly from the JaimeStill originals; diff is expected to be trivial).
5. Stale/cached Entra credentials don't refresh cleanly after a long idle — users must logout or clear storage+cookies to recover.

The desired outcome of this review session is a clean project-management structure (phase → objective → sub-issues) and a concrete technical approach per item so that implementation can begin without further discovery.

## Recommended Phase / Objective Structure

| Layer | Proposal |
|-------|----------|
| Phase | **Phase 5 — Post-Deployment Polish**, version target **v0.5.0** |
| Objective | Repurpose **#132** as the parent objective (add `objective` label, assign `Objective` issue type, link to Phase 5) |
| Sub-issues | Five `Task`/`Bug` sub-issues (one per QoL item) — each maps to exactly one branch and one PR |
| Milestone | New milestone `v0.5.0 - Post-Deployment Polish` on JaimeStill/herald |

Rationale: Phase 4 is closed, v0.4.0 is released, and these five items are post-deployment work that doesn't fit the prior phase scopes. A thin Phase 5 keeps infrastructure consistent (milestone, project board column, CHANGELOG section) without inventing new conventions. #132 already enumerates the work — converting it to an Objective preserves authorship and conversation history.

### Proposed Sub-Issues

| # | Type | Title | Scope |
|---|------|-------|-------|
| T1 | Bug | document-upload scroll container | CSS-only fix to `document-upload.module.css` |
| T2 | Task | Configurable page size selector | New `<hd-page-size-select>` element; wire into `document-grid` and `prompt-list` |
| T3 | Task | URL query-parameter state persistence | Router helper + wire `page`, `page_size`, `search`, `sort`, filter fields through URL |
| T4 | Task | Migrate to tau/agent and tau/orchestrate | Import-path find/replace + go.mod/go.sum + doc touchpoints |
| T5 | Bug | MSAL stale credential refresh | Broaden catch in `Auth.getToken`, cache-corruption recovery, proactive expiry on resume |

Suggested execution order: **T4 → T1 → T2 → T3 → T5**. T4 is the largest change surface but mechanical; doing it first keeps subsequent branches from re-touching agent infrastructure. T1 is trivial and unblocks visual fixes. T2 precedes T3 because the page-size field needs to exist before being persisted in the URL. T5 is last since it benefits from the API retry surface being stable.

## Per-Task Technical Approach

### T1 — document-upload scroll container

**File:** `/home/jaime/code/herald/app/client/ui/modules/document-upload.module.css`

**Change:** Apply the same overflow pattern already used by `prompt-list.ts` and `document-grid.ts`:

```css
:host { flex: 1; min-height: 0; }
.queue { flex: 1; min-height: 0; overflow-y: auto; }
/* .drop-zone and .queue-actions remain flex-shrink: 0 */
```

Verify in the browser by queueing ≥ 20 files; drag-drop, progress rows, and completion indicators must remain functional.

### T2 — Configurable page size selector

**New element:** `/home/jaime/code/herald/app/client/ui/elements/page-size-select.ts` (`<hd-page-size-select>`), presentational only. `@property() size: number`, `@property() options: number[] = [12, 24, 48, 96]`, emits `page-size-change` with `{ detail: { size: number } }`. Mirror the shape of `ui/elements/pagination-controls.ts`.

**Wiring:**
- `app/client/ui/modules/document-grid.ts` — promote the hardcoded `12` on line 62 to a `@state() page_size = 12`; render `<hd-page-size-select>` next to `<hd-pagination>`; on `page-size-change`, reset `page = 1` and refetch.
- `app/client/ui/modules/prompt-list.ts` — same pattern on line 50.

No server changes — both endpoints already accept `page_size` (see `SearchRequest` shapes in `app/client/core/types/document.ts:27–35` and `prompt.ts:40–48`).

### T3 — URL query-parameter state persistence

**Router helper:** Extend `/home/jaime/code/herald/app/client/core/router/router.ts` (and its `index.ts` export) with an `updateQuery(params: Record<string, string | undefined>): void` function that:
- Builds a new URL from `location.pathname` + the merged query object (empty/undefined values omitted).
- Calls `history.replaceState(null, "", newUrl)` — does NOT remount the view.

This is required because `navigate()` always remounts (router.ts:108 `this.container.innerHTML = ""`), which would destroy the list view on every filter/page change. `history.replaceState` sidesteps remount while still preserving state across a subsequent view transition (e.g., clicking into `/review/:id` then using Back).

**Reuse:** The existing `toQueryString` utility in `/home/jaime/code/herald/app/client/core/api.ts:164–172` is already used for API calls. It should also power `updateQuery`.

**View wiring:**
- `document-grid.ts`: change `@state` to `@property({ attribute: 'page', type: Number })`, `@property({ attribute: 'page_size', type: Number })`, `@property({ attribute: 'search' })`, `@property({ attribute: 'status' })`, `@property({ attribute: 'sort' })`. On any state change (debounce search ≥250ms), call `updateQuery({ page, page_size, search, status, sort })` and refetch. Router already splats query params onto the view element as attributes (router.ts:115–117), so initial load is automatic.
- `prompt-list.ts`: same treatment with its filter fields (`stage`, etc.).

**Edge cases to cover in the guide:**
- Omit default values from the URL (e.g., don't serialize `page=1` or `page_size=12`) to keep URLs clean.
- Reset `page` to 1 whenever search/filter/sort/page_size changes.
- `willUpdate()` must distinguish attribute-driven changes (initial mount, popstate) from user-driven changes to avoid refetch loops.

### T4 — Migrate to tau/agent and tau/orchestrate

**Import-path find/replace (mechanical):**

| From | To |
|------|----|
| `github.com/JaimeStill/go-agents/pkg/agent` | `github.com/tailored-agentic-units/agent/pkg/agent` (or top-level `agent` — verify package layout) |
| `github.com/JaimeStill/go-agents/pkg/config` | `github.com/tailored-agentic-units/agent/pkg/config` |
| `github.com/JaimeStill/go-agents-orchestration/pkg/config` | `github.com/tailored-agentic-units/orchestrate/pkg/config` |
| `github.com/JaimeStill/go-agents-orchestration/pkg/state` | `github.com/tailored-agentic-units/orchestrate/pkg/state` |
| `github.com/JaimeStill/go-agents-orchestration/pkg/observability` | `github.com/tailored-agentic-units/orchestrate/pkg/observability` |

**Files touched (confirmed via import audit):**
- `internal/workflow/{workflow,runtime,init,classify,enhance,finalize,observer}.go`
- `internal/config/{config,agent}.go`
- `internal/infrastructure/infrastructure.go`
- `internal/classifications/repository.go`
- `tests/infrastructure/infrastructure_test.go`
- `tests/api/api_test.go`
- `tests/workflow/observer_test.go`
- `go.mod`, `go.sum` — remove JaimeStill modules, add tau modules. Target version TBD from tau library's latest tag; if tau is pre-release and uses local development, consider a `replace` directive for local build during initial wiring, removed before PR.

**Doc touchpoints:**
- `_project/README.md` — Dependencies section (lines mentioning go-agents / go-agents-orchestration), Architecture diagrams/captions.
- `.claude/skills/web-development/SKILL.md` and `.claude/skills/api-cartographer/SKILL.md` — verify no references.
- `CHANGELOG.md` — new v0.5.0-dev.X.Y entry.

**Verification:**
- `mise run vet` clean.
- `mise run test` passes (all existing tests, especially `tests/workflow/observer_test.go` which is the most exercised integration point).
- `mise run dev` — perform one end-to-end classification against a test PDF to confirm the workflow graph still executes.

**Note on tau packages:** `~/tau/agent` and `~/tau/orchestrate` use top-level module paths (`module github.com/tailored-agentic-units/agent`) with `request/`, `registry/`, `client/`, etc. as direct subpackages — not under `pkg/`. During implementation, the first step is to confirm the actual subpackage paths (agent may live at `github.com/tailored-agentic-units/agent` rather than `.../pkg/agent`). If the structure differs, update the find/replace table above accordingly.

### T5 — MSAL stale credential refresh

**File:** `/home/jaime/code/herald/app/client/core/auth.ts`

**Root-cause hypothesis:** `Auth.getToken()` (line 99–116) only handles `InteractionRequiredAuthError` — any other MSAL error path (nonce mismatch after long idle, corrupted localStorage entries, interaction-in-progress conflict) silently returns `null`. The caller gets no token, no redirect happens, and the app appears broken until the user manually clears storage.

**Proposed fix — three layers of defense:**

1. **Broaden the catch.** Treat any non-`InteractionRequiredAuthError` as a cache-corruption signal: clear the MSAL cache (`msalInstance.clearCache()` or remove the active account), then call `login()`.
2. **Proactive expiry check.** In `getToken`, if `acquireTokenSilent` returns a result but `result.expiresOn` is within a small threshold (≤60s) of now, call `acquireTokenSilent({ forceRefresh: true })` once before returning.
3. **Resume-time validation.** In `init()`, register a `document.addEventListener("visibilitychange", ...)` that, on return-to-visible, calls `getToken(true)` in the background to prime the cache. Silent on failure (the next real API call will trigger the full flow).

**File:** `/home/jaime/code/herald/app/client/core/api.ts` — verify the existing 401 retry flow (at api.ts:42, 48) remains correct after the auth.ts changes. The retry path `getToken(true)` → `login()` on persistent failure should continue to work unchanged.

**Verification:**
- Manually: log in, leave the tab idle past the refresh-token lifetime (simulate by manually expiring localStorage entries), return to the app, confirm a clean re-authentication without needing to clear storage.
- Confirm the fix doesn't introduce redirect loops by leaving the app open on a stable connection for 5–10 minutes.

## Review Report Deliverable

Create `/home/jaime/code/herald/.claude/context/reviews/2026-04-21-herald.md` using the template from `commands/review.md`. Key content:

- **Infrastructure Audit:** Project #7 board, Phases 1–4 closed, no open issues other than #132. Milestones clean.
- **Codebase Assessment:** Unchanged from 2026-03-13 review (A grades across the board). No new technical debt introduced by IL6 deployment — the IL6 fixes (token scope trailing slash, TLS cert, authority) were already merged and documented.
- **Context Health:** 125+ archived guides, 36 session summaries, CLAUDE.md current.
- **Vision Alignment:** Phase 5 added to roadmap; remaining open questions (enhance triggers, GPT model benchmarking, bulk ingestion) deferred to future phases.
- **Actions Taken:** Phase 5 created; #132 converted to Objective; 5 sub-issues T1–T5 created; `v0.5.0` milestone created; sub-issues assigned to milestone and project board.
- **Recommendations:** Execute T4 → T1 → T2 → T3 → T5. Re-review after T5 merges to tag v0.5.0.

## Critical Files for Implementation

| Purpose | Path |
|---------|------|
| Scroll container fix | `app/client/ui/modules/document-upload.module.css` |
| New page-size element | `app/client/ui/elements/page-size-select.ts` (new) |
| Pagination controls (reference) | `app/client/ui/elements/pagination-controls.ts` |
| Document grid wiring | `app/client/ui/modules/document-grid.ts` |
| Prompt list wiring | `app/client/ui/modules/prompt-list.ts` |
| Router + query helper | `app/client/core/router/router.ts`, `app/client/core/router/index.ts` |
| Query-string utility (reuse) | `app/client/core/api.ts:164–172` (`toQueryString`) |
| MSAL service | `app/client/core/auth.ts` |
| API fetch wrapper | `app/client/core/api.ts` |
| Go agent imports | `internal/workflow/*.go`, `internal/config/*.go`, `internal/infrastructure/infrastructure.go`, `internal/classifications/repository.go`, `tests/{infrastructure,api,workflow}/*.go` |
| Go module declarations | `go.mod`, `go.sum` |
| Project docs | `_project/README.md` (Dependencies, Architecture) |

## Verification (End-to-End)

Each sub-issue has its own PR-scoped verification (above). At the objective level, after all five tasks merge:

1. `mise run vet` — clean.
2. `mise run test` — all tests pass.
3. `mise run dev` — manual smoke test:
   - Upload 20+ files, confirm scroll works in document-upload module.
   - Change page size; confirm URL updates with `?page_size=N` and list reloads.
   - Search, filter by status, sort by a non-default column; confirm URL reflects state.
   - Navigate into a document (`/review/:id`) and back; confirm grid state restored from URL.
   - Let the tab idle past the token lifetime, return, confirm seamless re-auth.
   - Run one classification end-to-end to confirm tau agent/orchestrate integration.
4. Close milestone `v0.5.0`, tag `v0.5.0`, update CHANGELOG per dev-workflow release session.
