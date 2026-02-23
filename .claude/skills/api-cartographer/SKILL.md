---
name: api-cartographer
description: >
  REQUIRED for API endpoint documentation and testing reference generation.
  Use when the user asks to "document API", "create API spec", "generate endpoint docs",
  "update API reference", "add endpoint documentation", "api cartographer",
  or when creating new HTTP handlers that need corresponding API documentation.
  Triggers: api docs, endpoint spec, curl reference, api reference, route documentation,
  api cartographer, document endpoints, api spec.

  When this skill is invoked, check for existing _project/api/ configuration.
  If none exists, run the initialization flow to collect project-specific settings.
  Then generate or update endpoint specifications following the established templates.
---

# API Cartographer

Markdown-based API specification with executable curl examples. Replaces OpenAPI/Scalar with a lightweight, terminal-native approach.

## When This Skill MUST Be Used

**ALWAYS invoke this skill when the request involves ANY of these:**

- Creating or modifying HTTP handler endpoints
- Documenting API routes for a new or existing domain
- Generating curl examples for API testing
- Updating endpoint specs after handler changes
- Initializing API documentation for a new project
- Adding a new route group to an existing spec

## Initialization

On first use in a project, collect three settings before generating any files:

### 1. API Base Variable

The shell variable name used in all curl examples. Points to the server root URL.

**Example values:** `HERALD_API_BASE`, `AL_API_BASE`, `APP_BASE_URL`

The root README instructs the user to export this variable before running examples:

```bash
export HERALD_API_BASE="http://localhost:8080"
```

### 2. Organizational Mechanism

A description of how endpoints are grouped into files. This is recorded in the root README so future spec authors understand the grouping rationale.

**Examples:**
- "Route groups organized by URL path prefix"
- "Logical tags matching domain boundaries"
- "Feature modules aligned with internal packages"

### 3. Auth Configuration

Current authentication mechanism. Can be updated as the project evolves.

| Type | Config | Effect on Examples |
|------|--------|--------------------|
| `none` | No token variable | Auth header omitted |
| `bearer` | Token variable name (e.g., `HERALD_TOKEN`) | Auth section in root README, examples omit for brevity |
| `api-key` | Key variable name + header name | Same pattern as bearer |

## File Organization

```
_project/api/
├── README.md              # Config, setup, auth, root endpoints, group index
├── documents.md           # Documents route group
├── classifications.md     # Classifications route group
└── prompts.md             # Prompts route group
```

- One markdown file per organizational group
- Root `README.md` contains project config, setup instructions, root endpoints, and a group index table

## Root README Structure

The root `_project/api/README.md` follows this structure:

1. **Title**: `# API Reference`
2. **Configuration table**: base variable, default value, organization, auth
3. **Setup section**: export command for the base variable (and auth token when configured)
4. **Auth section** (when applicable): how to include auth headers (omitted from individual examples for brevity)
5. **Route Groups table**: links to each group file with path prefix and description
6. **Root Endpoints**: endpoints that live outside any group (health checks, version, etc.)

## Group File Structure

Each group markdown file follows this structure:

1. **Title**: group name (e.g., `# Documents`)
2. **Base path**: the URL prefix for all endpoints in the group
3. **Description**: what the group covers
4. **Endpoints**: separated by horizontal rules (`---`), each following the endpoint template

## Endpoint Template

Every endpoint section follows this pattern:

```
## [Action Name]

`[METHOD] [full path]`

[One-line description.]

### [Parameters Section]

[Table of parameters — see templates.md for type-specific sections]

### Responses

| Status | Description |
|--------|-------------|
| [code] | [description] |

### Example

```bash
curl -s ... | jq .
```

### Full Example (optional — only for endpoints with 3+ optional parameters)

```bash
curl -s ... | jq .
```
```

See [references/templates.md](references/templates.md) for complete templates covering each request type (query params, path params, JSON body, multipart form, etc.).

## Conventions

### Endpoint Documentation

- **Horizontal rules** (`---`) separate endpoints within a group file
- **Full path** in the endpoint header (e.g., `/api/documents/{id}`, not `/{id}`)
- **Full Example** section only appears when an endpoint has 3+ optional parameters
- **Response bodies** are not documented inline — only status codes and descriptions
- **Path parameters** use `{name}` syntax in the endpoint path

### Curl Examples

- Use `-s` (silent) to suppress the progress meter
- Pipe JSON responses through `jq .` for pretty-printed output
- Every curl command uses the base variable: `"$HERALD_API_BASE/api/documents"`
- Auth headers are omitted for brevity — the root README explains how to append them
- POST/PUT/PATCH examples include `-H "Content-Type: application/json"` where applicable
- Multipart examples use `-F` flags
- DELETE examples include `-X DELETE` (no `jq` pipe since 204 has no body)
- JSON bodies use `-d` with single-quoted JSON strings
- Multi-line JSON bodies use `-d` with a properly formatted JSON block

## Maintenance

### When to Update

- **New endpoint added**: add the endpoint section to the group file
- **Endpoint modified**: update parameters, responses, and examples
- **Endpoint removed**: remove the section
- **New group created**: create the markdown file and add to root group table
- **Auth configuration changes**: update root README auth section

### AI Responsibility

API Cartographer maintenance is an AI responsibility. When HTTP handlers are created or modified during task execution:

1. Generate or update the corresponding `_project/api/<group>.md`
2. Keep the root README route group table current
3. Ensure all curl examples use the project's configured base variable

This replaces the former OpenAPI schema maintenance responsibility.

### Ordering

Endpoints within a group file should follow a consistent order:

1. List (GET collection)
2. Find (GET single resource)
3. Search (POST with filters)
4. Create / Upload (POST)
5. Update (PUT/PATCH)
6. Delete (DELETE)

Custom endpoints follow after the standard CRUD operations.
