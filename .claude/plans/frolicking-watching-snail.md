# Phase 3 Closeout — Web Client (v0.3.0)

## Context

Phase 3 (Web Client) is functionally complete. All 5 objectives (#57–#61) have their sub-issues closed and merged. The remaining work is mechanical: close tracking infrastructure, consolidate the changelog, clean up dev releases/tags/container images, fix stale documentation, produce a review report, and tag the v0.3.0 release.

## Review Findings

### Infrastructure
- **Issue #61**: OPEN on GitHub, but all 3 sub-issues (#88, #89, #90) are closed — needs closing
- **Milestone v0.3.0**: open — needs closing
- **15 dev releases/tags**: `v0.3.0-dev.57.62` through `v0.3.0-dev.61.90` — need deletion
- **15 GHCR container images**: `0.3.0-dev.*` versions on `ghcr.io/jaimestill/herald` — need deletion
- **No orphaned/stale issues**: #61 is the only open issue

### Stale Documentation (`_project/README.md`)
- Phase 3 row missing `Complete` status (line 29)
- Lines 224-225: reference `@lit-labs/signals` (State) and `@lit/context` (Services) — both removed in #84
- Line 264: frontend deps list includes `@lit/context, @lit-labs/signals` — only `lit` remains in package.json

### Project Documents
- `_project/phase.md`: Objective 5 shown as "Open" — needs clearing (phase complete)
- `_project/objective.md`: still has #61 details — needs clearing (objective complete)

### Context Infrastructure
- All implementation guides archived (`.claude/context/guides/` empty)
- 36 session summaries present and complete
- Previous review at `2026-02-28-herald.md` — kept in place (reviews are historical records, not archived)

## Execution Plan

### Step 1: GitHub Infrastructure Cleanup

Close issue #61 and milestone v0.3.0:

```bash
gh issue close 61 --comment "All sub-issues complete. Closing as part of Phase 3 closeout."

MILESTONE_NUMBER=$(gh api repos/{owner}/{repo}/milestones --jq '.[] | select(.title | startswith("v0.3.0")) | .number')
gh api --method PATCH repos/{owner}/{repo}/milestones/$MILESTONE_NUMBER -f state=closed
```

### Step 2: Create Branch

```bash
git checkout -b phase-3-closeout
```

### Step 3: Create Review Report

Create `.claude/context/reviews/2026-03-08-herald.md` with infrastructure audit, codebase assessment, context health, vision alignment, and actions taken.

### Step 4: Consolidate CHANGELOG

Replace all 15 `## v0.3.0-dev.*` sections with a single `## v0.3.0` section organized by subsystem:

- **Web Infrastructure** — pkg/web/ templates, Go web app module, Air hot reload (#62, #64)
- **Client Build System and Design** — Bun build pipeline, CSS modules, design system, client restructure, shared styles (#63, #77, #82, #90)
- **Data Layer and Services** — TypeScript domain types, stateless service objects, SSE stream (#74)
- **SSE Classification Streaming** — Workflow streaming observer, SSE endpoint (#70, #71)
- **Document Management View** — Document card/progress elements, upload component, grid module, view composition (#75, #76, #77)
- **Prompt Management View** — Prompt card, list module, form module, view composition (#82, #83, #84)
- **Document Review View** — Storage inline endpoint, blob viewer, markings list, classification panel, review view (#88, #89, #90)

### Step 5: Update `_project/README.md`

Three edits:
1. **Line 29**: Add `Complete` to Phase 3 Status column
2. **Lines 224-225**: Replace signal/context references with `@state()` decorator pattern and stateless service objects
3. **Line 264**: Change `Lit 3.x (lit, @lit/context, @lit-labs/signals)` to `Lit 3.x (lit)`

### Step 6: Clear Project Documents

- `_project/phase.md` → empty placeholder (no active phase)
- `_project/objective.md` → empty placeholder (no active objective)

### Step 7: Commit, Push, PR

```bash
git add .
git commit -m "Close out Phase 3 — Web Client (v0.3.0)

Consolidate 15 dev changelog entries into a single v0.3.0 release
organized by subsystem. Mark Phase 3 complete in project README.
Fix stale web client documentation removing references to
@lit-labs/signals and @lit/context. Clear phase and objective
tracking documents. Create Phase 3 review report."

git push -u origin phase-3-closeout

gh pr create --title "Phase 3 closeout — Web Client v0.3.0" --body "..."
# PR body includes: Closes #61
```

### Step 8: After PR Merge — Delete Dev Releases, Container Images, and Tags

```bash
# Delete 15 GitHub releases
gh release list --json tagName --jq '.[].tagName' \
  | grep '^v0.3.0-dev\.' \
  | while read tag; do gh release delete "$tag" --yes; done

# Delete GHCR container images
gh api user/packages/container/herald/versions \
  --jq '[.[] | select(.metadata.container.tags | any(startswith("0.3.0-dev"))) | .id] | .[]' \
  | while read id; do gh api --method DELETE "user/packages/container/herald/versions/$id"; done

# Delete local tags
git tag -l 'v0.3.0-dev.*' | xargs git tag -d

# Delete remote tags (may already be gone after release deletion — errors are harmless)
git tag -l 'v0.3.0-dev.*' | xargs -I{} git push origin --delete {}
```

### Step 9: Tag v0.3.0

```bash
git checkout main
git pull origin main
git branch -d phase-3-closeout
git remote prune origin
git tag v0.3.0
git push origin v0.3.0
```

Triggers the GitHub Actions release workflow to create the v0.3.0 GitHub release and push the `ghcr.io/jaimestill/herald:0.3.0` container image (plus `0.3` and `latest` tags).

## Critical Files

| File | Action |
|------|--------|
| `CHANGELOG.md` | Consolidate 15 dev entries → single v0.3.0 |
| `_project/README.md` | Mark Phase 3 complete, fix stale web client docs |
| `_project/phase.md` | Clear to empty placeholder |
| `_project/objective.md` | Clear to empty placeholder |
| `.claude/context/reviews/2026-03-08-herald.md` | Create (new review report) |

## Verification

1. `go vet ./...` passes
2. `go test ./tests/...` passes
3. Issue #61 closed on GitHub
4. Milestone v0.3.0 closed
5. PR merged with `Closes #61`
6. All 15 dev releases deleted
7. All 15 dev GHCR container images deleted
8. All 15 dev tags deleted (local + remote)
9. `v0.3.0` tag exists and release workflow triggered
10. `_project/README.md` Phase 3 shows `Complete`, no stale signal/context references
