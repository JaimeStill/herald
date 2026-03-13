# Phase 4 Review and v0.4.0 Release Plan

## Context

Phase 4 (Security and Deployment) is the final phase of Herald's initial development roadmap. All 6 objectives and 20 sub-issues are complete. This plan covers the final project review, phase closeout, and v0.4.0 release — the initial MVP milestone.

## Phase 1: Project Review

Conduct a comprehensive review and fix all discrepancies found. The audit identified the following issues:

### 1A. Version Alignment Fix

- **`config.json` line 3**: Update `"version": "0.2.0"` to `"version": "0.4.0"`

### 1B. `_project/README.md` Project Structure Update

The documented project structure is stale. Fix:

- **`web/app/`** → **`app/`** (web client lives at project root `app/`, not `web/app/`)
- **Add `pkg/auth/`** — Auth config, User context, JWT middleware (added in Phase 4)
- **Add `pkg/formatting/`** — Formatting utilities (added in Phase 3)
- **Add `deploy/`** — Bicep IaC deployment manifests (added in Phase 4)
- **Remove `web/`** — does not exist; `pkg/web/` is already listed correctly

### 1C. `_project/README.md` Content Updates

- Add `pkg/auth/` to the Infrastructure Layer section or a new Auth section
- Ensure the Entra ID integration point reflects the actual implementation (No OBO, managed identity for service-to-service, JWT claims for audit)
- Verify the Azure Entra ID section under Integration Points matches Phase 4 implementation

### 1D. Remove Ephemeral Phase/Objective Documents

- **Delete `_project/phase.md`** — ephemeral document scoped to the active phase, no longer needed
- **Delete `_project/objective.md`** — ephemeral document scoped to the active objective, no longer needed

### 1E. Review Report

Create `.claude/context/reviews/2026-03-13-herald.md` with the full review findings.

## Phase 2: Phase Closeout

Create a branch for all closeout work: `git checkout -b phase-4-closeout`

### 2A. Close GitHub Issues and Milestone

1. Close issue #100 (Deployment Configuration objective)
2. Close the `v0.4.0 - Security and Deployment` milestone
3. Verify all 20 Phase 4 issues are closed

### 2B. Consolidate CHANGELOG

Consolidate the 13 `v0.4.0-dev.*` entries into a single `v0.4.0` section organized by subsystem (matching v0.3.0's structure):
- **Authentication** — pkg/auth, JWT middleware, MSAL.js, user identity propagation
- **Managed Identity** — credential providers, token-based DB/storage/agent auth
- **Deployment** — Dockerfile hardening, Bicep IaC, Container Apps, migration job
- **Configuration** — AgentScope configurability, auth config pipeline

### 2C. Clean Up Dev Releases and Tags

Delete the 13 dev releases from GitHub, their GHCR container images, and the corresponding git tags:
- `v0.4.0-dev.95.101` through `v0.4.0-dev.100.126`

### 2D. Commit and PR

- Commit all changes (config version, README updates, CHANGELOG consolidation, objective/phase status, review report)
- Create a PR for the phase closeout
- Merge to main

## Phase 3: Tag v0.4.0 Release

After the closeout PR is merged:

1. `git tag v0.4.0` on the merged commit
2. `git push origin v0.4.0`
3. The release workflow will auto-generate the GitHub release and push the Docker image

Do NOT use `gh release create` — dev release tags are pushed manually per project conventions.

## Critical Files

| File | Change |
|------|--------|
| `config.json` | Version `0.2.0` → `0.4.0` |
| `_project/README.md` | Project structure, auth package, deploy dir |
| `_project/phase.md` | Delete (ephemeral, phase complete) |
| `_project/objective.md` | Delete (ephemeral, objective complete) |
| `CHANGELOG.md` | Consolidate dev entries → single v0.4.0 section |
| `.claude/context/reviews/2026-03-13-herald.md` | New review report |

## Verification

1. `go vet ./...` — clean
2. `go test ./tests/... -count=1` — all pass
3. `git log --oneline -5` — verify clean history after closeout merge
4. `gh issue list --milestone "v0.4.0 - Security and Deployment" --state open` — should return 0
5. `gh release list` — v0.4.0 present as Latest after tag push
