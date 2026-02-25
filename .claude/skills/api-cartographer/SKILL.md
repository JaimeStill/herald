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

On first use in a project, collect settings and create the environment file before generating any documentation:

### 1. API Base Variable

The shell variable name used in all curl examples. Points to the server root URL.

**Example values:** `HERALD_API_BASE`, `AL_API_BASE`, `APP_BASE_URL`

The root README instructs the user to export this variable before running examples:

```bash
export HERALD_API_BASE="http://localhost:8080"
```

### 2. HTTP Client Environment

On initialization, create `_project/api/http-client.env.json` with the `HOST` variable matching the base URL:

```json
{
  "$schema": "https://getkulala.net/http-client.env.schema.json",
  "dev": {
    "HOST": "http://localhost:8080"
  }
}
```

Additional environments (staging, prod) can be added as needed. Auth tokens are added here when authentication is configured.

### 3. Organizational Mechanism

A description of how endpoints are grouped into files. This is recorded in the root README so future spec authors understand the grouping rationale.

**Examples:**
- "Route groups organized by URL path prefix"
- "Logical tags matching domain boundaries"
- "Feature modules aligned with internal packages"

### 4. Auth Configuration

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
├── root.http              # Root endpoint test file (health, readiness, etc.)
├── http-client.env.json   # Kulala environment variables (HOST, auth tokens)
├── documents/
│   ├── README.md          # Documents route group documentation
│   └── documents.http     # Documents endpoint test file
├── classifications/
│   ├── README.md          # Classifications route group documentation
│   └── classifications.http
└── prompts/
    ├── README.md          # Prompts route group documentation
    └── prompts.http       # Prompts endpoint test file
```

- Each route group gets a subdirectory containing its `README.md` (documentation) and `.http` (test file)
- Root `README.md` contains project config, setup instructions, root endpoints, and a group index table
- `root.http` contains test requests for root-level endpoints documented in the root README (health checks, readiness, etc.)
- `http-client.env.json` sits at the `api/` root and defines shared environment variables for all `.http` files

## Root README Structure

The root `_project/api/README.md` follows this structure:

1. **Title**: `# API Reference`
2. **Configuration table**: base variable, default value, organization, auth
3. **Setup section**: export command for the base variable (and auth token when configured)
4. **Auth section** (when applicable): how to include auth headers (omitted from individual examples for brevity)
5. **Route Groups table**: links to each group directory with path prefix and description
6. **Root Endpoints**: endpoints that live outside any group (health checks, version, etc.)

Route group links use directory paths (e.g., `[Documents](documents/)`) so GitHub renders the directory listing.

## Group Directory Structure

Each group directory contains two files:

### README.md (API Documentation)

1. **Title**: group name (e.g., `# Documents`)
2. **Base path**: the URL prefix for all endpoints in the group
3. **Description**: what the group covers
4. **Endpoints**: separated by horizontal rules (`---`), each following the endpoint template

### [group].http (REST Test File)

Kulala-compatible HTTP test file for executing requests directly from Neovim. Uses the `{{HOST}}` variable from `http-client.env.json`.

**Conventions:**

- `###` separates requests with descriptive names (e.g., `### Create Prompt`)
- `@variableName=value` defines reusable variables (e.g., `@promptId=...`) with a comment noting replacement
- Request order follows the same CRUD ordering as the README
- JSON bodies are formatted for readability
- No auth headers — same brevity convention as curl examples

## Endpoint Template

Every endpoint section follows this pattern:

````
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
````

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

- **New endpoint added**: add the endpoint section to the group README and a request block to the `.http` file
- **Endpoint modified**: update parameters, responses, examples in README and the corresponding `.http` request
- **Endpoint removed**: remove the section from README and the request block from `.http`
- **New group created**: create the group directory with `README.md` and `[group].http`, add to root group table
- **Auth configuration changes**: update root README auth section and `http-client.env.json`

### AI Responsibility

API Cartographer maintenance is an AI responsibility. When HTTP handlers are created or modified during task execution:

1. Generate or update the corresponding `_project/api/<group>/README.md`
2. Generate or update the corresponding `_project/api/<group>/[group].http`
3. Keep the root README route group table current
4. Ensure all curl examples use the project's configured base variable
5. Ensure all `.http` files use `{{HOST}}` from `http-client.env.json`

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
