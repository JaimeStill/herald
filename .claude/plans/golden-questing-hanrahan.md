# Plan: Add Blob Storage Query Task to Objective #3

## Context

Objective #3 (Document Domain) currently has two closed sub-issues (#16, #17) covering SQL-backed document CRUD. However, there are no endpoints for querying Azure Blob Storage directly — clients can't list what's in the store, inspect blob metadata, or download files. Before closing Objective #3 and moving to Phase 2 planning, this gap should be addressed with one new Task issue.

This session is **project management only** — create the issue, link it to the objective, update the project board, and update `_project/objective.md`. The actual implementation will happen in a separate task execution session.

## Step 1: Create the Task Issue

```bash
CHILD_URL=$(gh issue create \
  --repo JaimeStill/herald \
  --title "Blob storage query endpoints — list, metadata, and download" \
  --label "feature" --label "task" \
  --milestone "v0.1.0 - Service Foundation" \
  --body "$(cat <<'EOF'
## Context

The document domain (Objective #3) provides CRUD operations against PostgreSQL for document metadata. However, there are no endpoints for querying Azure Blob Storage directly. This gap prevents clients from listing what's in the store, inspecting blob metadata, or downloading files without going through the SQL layer.

## Scope

Add read-only HTTP endpoints under a new `/storage` route group that query Azure Blob Storage directly:

| Method | Path | Purpose |
|--------|------|---------|
| `GET /api/storage` | List blobs with prefix filtering and marker-based pagination |
| `GET /api/storage/properties?key=...` | Get blob metadata by storage key |
| `GET /api/storage/download?key=...` | Download a file by storage key |

No mutating operations — document upload and deletion remain exclusively through the documents API.

**Key constraint**: Azure Blob Storage uses marker-based pagination (opaque continuation tokens), not offset-based pages. The list endpoint uses `prefix`, `marker`, and `max_results` query parameters — fundamentally different from the `PageRequest/PageResult` pattern used for SQL queries.

## Approach

1. Extend `storage.System` interface (`pkg/storage/storage.go`) with `List` and `GetProperties` methods
2. Add blob listing types (`BlobItem`, `BlobProperties`, `ListResult`) to `pkg/storage/`
3. Implement `List` using azblob's `NewListBlobsFlatPager` with prefix and marker support
4. Implement `GetProperties` using the blob client's `GetProperties` method
5. Create a storage handler in `internal/api/` with `List`, `Properties`, and `Download` endpoints
6. Register the `/storage` route group in `internal/api/routes.go`
7. Generate API Cartographer documentation at `_project/api/storage.md`

## Acceptance Criteria

- [ ] `storage.System` interface extended with `List` and `GetProperties` methods
- [ ] `GET /api/storage` returns blob listing with marker-based pagination
- [ ] `GET /api/storage/properties?key=...` returns blob metadata JSON
- [ ] `GET /api/storage/download?key=...` streams the blob with correct Content-Type and Content-Disposition headers
- [ ] All endpoints work against Azurite in local development
- [ ] `go vet ./...` passes
- [ ] Tests pass (`mise run test`)
- [ ] API Cartographer documentation generated at `_project/api/storage.md`
EOF
)" --json url --jq '.url')
```

## Step 2: Link as Sub-Issue of Objective #3

Parent issue ID: `I_kwDORUCvQs7sRlmf`

```bash
CHILD_ID=$(gh issue view "$CHILD_URL" --json id --jq '.id')

gh api graphql \
  -H "GraphQL-Features: sub_issues" \
  -f query='mutation($parentId: ID!, $childUrl: URI!) {
    addSubIssue(input: {issueId: $parentId, subIssueUrl: $childUrl}) {
      subIssue { number title url }
    }
  }' -f parentId="I_kwDORUCvQs7sRlmf" -f childUrl="$CHILD_URL"
```

> Note: Skip `updateIssueIssueType` — not available on personal repos per project memory.

## Step 3: Add to Project Board

```bash
# Add to project board
ITEM_ID=$(gh project item-add 7 --owner JaimeStill --url "$CHILD_URL" --format json | jq -r '.id')

# Assign Phase 1
gh project item-edit --id "$ITEM_ID" --project-id "PVT_kwHOANcww84BPoG9" \
  --field-id "PVTSSF_lAHOANcww84BPoG9zg9-08c" \
  --single-select-option-id "1fc02e12"
```

## Step 4: Update `_project/objective.md`

Add the new sub-issue to the sub-issues table (after #17) and mark #16 and #17 as Complete:

**File:** `_project/objective.md`

Update the sub-issues table to:

```markdown
| # | Title | Labels | Status | Dependencies |
|---|-------|--------|--------|--------------|
| [#16](...) | Document domain core — types, mapping, repository, and system | `feature`, `task` | Complete | — |
| [#17](...) | Document HTTP handlers and API wiring | `feature`, `task` | Complete | #16 |
| [#N](...) | Blob storage query endpoints — list, metadata, and download | `feature`, `task` | Open | — |
```

## Verification

- [ ] Issue created with correct labels, milestone, and body
- [ ] Issue linked as sub-issue of #3 (visible in GitHub sub-issues UI)
- [ ] Issue appears on project board #7 with Phase 1 assignment
- [ ] `_project/objective.md` reflects all three sub-issues with correct statuses
