# Prompts

`/api/prompts`

Named prompt instruction overrides for workflow stages. Each prompt targets a specific stage (init, classify, enhance) and provides tunable instructions. At most one prompt per stage can be active.

---

## List Prompts

`GET /api/prompts`

Returns a paginated list of prompts with optional filters.

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| page | integer | no | Page number (1-indexed) |
| page_size | integer | no | Results per page |
| search | string | no | Search across name and description |
| sort | string | no | Comma-separated sort fields, prefix `-` for descending |
| stage | string | no | Filter by stage (init, classify, enhance) |
| name | string | no | Filter by name (contains, case-insensitive) |
| active | boolean | no | Filter by active status |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Paginated prompt list |

### Example

```bash
curl -s "$HERALD_API_BASE/api/prompts" | jq .
```

### Full Example

```bash
curl -s "$HERALD_API_BASE/api/prompts?page=1&page_size=20&search=detailed&sort=name&stage=classify&active=true" | jq .
```

---

## List Stages

`GET /api/prompts/stages`

Returns the authoritative list of valid workflow stages.

### Responses

| Status | Description |
|--------|-------------|
| 200 | Array of stage values |

### Example

```bash
curl -s "$HERALD_API_BASE/api/prompts/stages" | jq .
```

---

## Find Prompt

`GET /api/prompts/{id}`

Returns a single prompt by UUID.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Prompt UUID |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Prompt found |
| 404 | Prompt not found |

### Example

```bash
curl -s "$HERALD_API_BASE/api/prompts/550e8400-e29b-41d4-a716-446655440000" | jq .
```

---

## Search Prompts

`POST /api/prompts/search`

Search prompts with a JSON body containing pagination and filter criteria.

### Request

Content-Type: `application/json`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| page | integer | no | Page number (1-indexed) |
| page_size | integer | no | Results per page |
| search | string | no | Search across name and description |
| sort | string | no | Comma-separated sort fields, prefix `-` for descending |
| stage | string | no | Filter by stage (init, classify, enhance) |
| name | string | no | Filter by name (contains) |
| active | boolean | no | Filter by active status |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Paginated search results |
| 400 | Invalid request body |

### Example

```bash
curl -s -X POST "$HERALD_API_BASE/api/prompts/search" \
  -H "Content-Type: application/json" \
  -d '{"page": 1, "page_size": 20}' | jq .
```

### Full Example

```bash
curl -s -X POST "$HERALD_API_BASE/api/prompts/search" \
  -H "Content-Type: application/json" \
  -d '{
    "page": 1,
    "page_size": 20,
    "search": "detailed",
    "sort": "name",
    "stage": "classify",
    "name": "verbose",
    "active": true
  }' | jq .
```

---

## Create Prompt

`POST /api/prompts`

Create a new prompt instruction override.

### Request

Content-Type: `application/json`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | yes | Unique name for the prompt |
| stage | string | yes | Workflow stage (init, classify, enhance) |
| instructions | string | yes | Prompt instruction text |
| description | string | no | Description of the prompt's purpose |

### Responses

| Status | Description |
|--------|-------------|
| 201 | Prompt created |
| 400 | Invalid request body or invalid stage |
| 409 | Prompt name already exists |

### Example

```bash
curl -s -X POST "$HERALD_API_BASE/api/prompts" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "detailed-classify",
    "stage": "classify",
    "instructions": "Analyze each page thoroughly, noting all security markings including portion markings, banner lines, and classification authority blocks.",
    "description": "Detailed classification instructions with emphasis on portion markings"
  }' | jq .
```

---

## Update Prompt

`PUT /api/prompts/{id}`

Update an existing prompt.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Prompt UUID |

### Request

Content-Type: `application/json`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | yes | Unique name for the prompt |
| stage | string | yes | Workflow stage (init, classify, enhance) |
| instructions | string | yes | Prompt instruction text |
| description | string | no | Description of the prompt's purpose |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Prompt updated |
| 400 | Invalid request body or invalid stage |
| 404 | Prompt not found |
| 409 | Prompt name already exists |

### Example

```bash
curl -s -X PUT "$HERALD_API_BASE/api/prompts/550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "detailed-classify-v2",
    "stage": "classify",
    "instructions": "Analyze each page thoroughly. Pay special attention to banner lines, portion markings, and classification authority blocks. Note any discrepancies between portion and banner markings.",
    "description": "Enhanced classification instructions v2"
  }' | jq .
```

---

## Delete Prompt

`DELETE /api/prompts/{id}`

Deletes a prompt.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Prompt UUID |

### Responses

| Status | Description |
|--------|-------------|
| 204 | Prompt deleted |
| 404 | Prompt not found |

### Example

```bash
curl -s -X DELETE "$HERALD_API_BASE/api/prompts/550e8400-e29b-41d4-a716-446655440000"
```

---

## Activate Prompt

`POST /api/prompts/{id}/activate`

Activates a prompt for its stage. Atomically deactivates the currently active prompt for the same stage (if any) and activates this one.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Prompt UUID |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Prompt activated |
| 404 | Prompt not found |

### Example

```bash
curl -s -X POST "$HERALD_API_BASE/api/prompts/550e8400-e29b-41d4-a716-446655440000/activate" | jq .
```

---

## Deactivate Prompt

`POST /api/prompts/{id}/deactivate`

Deactivates a prompt. The stage falls back to hard-coded default instructions.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Prompt UUID |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Prompt deactivated |
| 404 | Prompt not found |

### Example

```bash
curl -s -X POST "$HERALD_API_BASE/api/prompts/550e8400-e29b-41d4-a716-446655440000/deactivate" | jq .
```
