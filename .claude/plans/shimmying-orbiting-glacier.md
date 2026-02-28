# Phase 2 Close-Out Plan

## Context

Phase 2 (Classification Engine, v0.2.0) is functionally complete. All 4 objectives (#24–#27) and their sub-issues have been implemented and merged to main. The final two sub-issues (#51, #53) were merged today. What remains is administrative closeout: closing the objective issue, clearing tracking files, consolidating 12 dev-release changelog entries into a single v0.2.0 entry, cleaning up dev releases/tags/GHCR images, and tagging the final v0.2.0 release.

The `phase-closeout` branch already exists (no commits ahead of main) and will be used for the file changes.

## Current State

- **Issues #51, #53**: Already closed
- **Issue #27** (Objective: Classifications Domain): Still open — close it
- **Milestone #2** (v0.2.0 - Classification Engine): Open, 1 open issue (#27) — close after #27
- **12 v0.2.0-dev.\* releases** and associated GHCR container images — delete all
- **12 v0.2.0-dev.\* tags** (local + remote) — delete all

## Steps

### 1. Close Issue #27 and Milestone

```bash
gh issue close 27 --comment "All sub-issues merged. Phase 2 complete."
gh api --method PATCH repos/JaimeStill/herald/milestones/2 -f state=closed
```

### 2. Delete v0.2.0-dev Releases (12 releases)

```bash
gh release delete v0.2.0-dev.27.53 --yes
gh release delete v0.2.0-dev.27.51 --yes
gh release delete v0.2.0-dev.27.48 --yes
gh release delete v0.2.0-dev.27.47 --yes
gh release delete v0.2.0-dev.26.41 --yes
gh release delete v0.2.0-dev.26.40 --yes
gh release delete v0.2.0-dev.26.39 --yes
gh release delete v0.2.0-dev.26.38 --yes
gh release delete v0.2.0-dev.26.37 --yes
gh release delete v0.2.0-dev.25.34 --yes
gh release delete v0.2.0-dev.24.30 --yes
gh release delete v0.2.0-dev.24.29 --yes
```

### 3. Delete GHCR Container Image Versions

Query the GHCR packages API for all v0.2.0-dev image version IDs, then delete each. The v0.1.0 image (tagged `latest`, `0.1`, `0.1.0`) is preserved.

```bash
# List version IDs for v0.2.0-dev images
gh api user/packages/container/herald/versions --jq '.[] | select(.metadata.container.tags | any(startswith("0.2.0-dev"))) | .id'

# Delete each version by ID
gh api --method DELETE user/packages/container/herald/versions/{id}
```

### 4. Delete v0.2.0-dev Tags (Local + Remote)

```bash
git tag -d v0.2.0-dev.24.29 v0.2.0-dev.24.30 v0.2.0-dev.25.34 \
  v0.2.0-dev.26.37 v0.2.0-dev.26.38 v0.2.0-dev.26.39 \
  v0.2.0-dev.26.40 v0.2.0-dev.26.41 v0.2.0-dev.27.47 \
  v0.2.0-dev.27.48 v0.2.0-dev.27.51 v0.2.0-dev.27.53

git push origin --delete v0.2.0-dev.24.29 v0.2.0-dev.24.30 v0.2.0-dev.25.34 \
  v0.2.0-dev.26.37 v0.2.0-dev.26.38 v0.2.0-dev.26.39 \
  v0.2.0-dev.26.40 v0.2.0-dev.26.41 v0.2.0-dev.27.47 \
  v0.2.0-dev.27.48 v0.2.0-dev.27.51 v0.2.0-dev.27.53
```

### 5. Clear `_project/phase.md`

Replace contents with:

```markdown
# Current Phase

No active phase.
```

### 6. Clear `_project/objective.md`

Replace contents with:

```markdown
# Current Objective

No active objective.
```

### 7. Update `_project/README.md`

**7a. Fix stale Core Premise** (line 19): The text says "sequential context accumulation" and "simplified 3-node state graph". Update to reflect the actual 4-node topology with parallel per-page analysis and a dedicated finalize node.

**7b. Add Status column to Phases table** (lines 25–30):

```markdown
| Phase | Focus Area | Version Target | Status |
|-------|-----------|----------------|--------|
| Phase 1 - ... | ... | v0.1.0 | Complete |
| Phase 2 - ... | ... | v0.2.0 | Complete |
| Phase 3 - ... | ... | v0.3.0 | |
| Phase 4 - ... | ... | v0.4.0 | |
```

### 8. Consolidate `CHANGELOG.md`

Replace all 12 `## v0.2.0-dev.*` sections with a single `## v0.2.0` section, grouped thematically:

```markdown
## v0.2.0

### Agent Configuration
- Config format migration TOML→JSON, AgentConfig integration (#29)

### Database Schema
- Classification engine migration — classifications and prompts tables (#30)

### Prompts Domain
- Full CRUD, typed stage enum, atomic activation swap (#34)
- Hardcoded default instructions/specs, stage content endpoints (#37)
- API Cartographer subdirectory-per-group restructure (#34)

### Classification Workflow
- Workflow foundation types, runtime, sentinel errors, prompt composition (#38)
- Init node — concurrent page rendering to temp storage (#39)
- Classify node — 4-node topology (init → classify → enhance? → finalize) (#40)
- Finalize stage added to prompts domain (#40)
- Enhance node, finalize node, state graph assembly, Execute function (#41)
- EnhanceSettings struct for typed rendering parameters (#41)
- Parallel classify and enhance with bounded errgroup concurrency (#51)

### Classifications Domain
- Move workflow/ to internal/workflow/ (#47)
- Domain types, system interface, repository with upsert semantics (#47)
- Handler with 8 HTTP endpoints, API module wiring (#48)
- Layered runtime convention — workflow encapsulated in classifications.New (#48)
- secrets.json config pipeline stage (#48)
- Azure AI Foundry provisioning scripts (#48)
- API Cartographer documentation (#48)

### Query Builder
- JOIN support with context-switching ProjectionMap pattern (#53)
- Document struct extended with classification metadata via LEFT JOIN (#53)
- Classification and confidence filters for document queries (#53)
```

### 9. Review `.claude/CLAUDE.md`

No changes needed — the project instructions accurately describe the current codebase and conventions. Phase 3-specific instructions (Lit conventions, web patterns) will be added when that phase begins.

### 10. Commit, Push, and Create PR

Use the existing `phase-closeout` branch. Commit message:

```
Close out Phase 2 — Classification Engine

Consolidate 12 v0.2.0-dev changelog entries into a single v0.2.0
release section organized by subsystem. Clear phase and objective
tracking files. Update Core Premise to reflect 4-node parallel
workflow topology. Mark Phase 2 complete in the phases table.
```

PR title: `Close out Phase 2 — Classification Engine`

PR body includes `Closes #27` (redundant safety net since we close it in step 1).

### 11. After PR Merge — Tag v0.2.0

```bash
git checkout main
git pull origin main
git tag v0.2.0
git push origin v0.2.0
```

This triggers the release workflow which builds/pushes the Docker image (tagged `0.2.0`, `0.2`, `latest`) and creates the GitHub Release from the consolidated `## v0.2.0` changelog section.

## Files Modified

| File | Change |
|------|--------|
| `_project/phase.md` | Clear to empty placeholder |
| `_project/objective.md` | Clear to empty placeholder |
| `_project/README.md` | Fix Core Premise, add Status column to Phases table |
| `CHANGELOG.md` | Consolidate 12 dev sections → single v0.2.0 |

## Verification

- `gh issue view 27 --json state` → CLOSED
- `gh api repos/JaimeStill/herald/milestones/2 --jq .state` → closed
- `gh release list` → no v0.2.0-dev.\* releases, v0.1.0 intact
- `git tag -l 'v0.2.0*'` → empty (until v0.2.0 is tagged in step 11)
- `gh api user/packages/container/herald/versions --jq '.[].metadata.container.tags'` → only v0.1.0 tags remain
- After step 11: `gh release view v0.2.0` confirms release exists with consolidated changelog
