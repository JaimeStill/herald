# Endpoint Templates

Templates for each request type. Replace `$BASE` with the project's configured base variable (e.g., `$HERALD_API_BASE`).

## GET with Query Parameters

Used for list/collection endpoints with optional filtering.

```markdown
## List [Resources]

`GET [full path]`

[Description.]

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| page | integer | no | Page number (1-indexed) |
| page_size | integer | no | Results per page |
| search | string | no | [Search description] |
| sort | string | no | Comma-separated sort fields, prefix `-` for descending |
| [filter] | [type] | no | [Filter description] |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Paginated [resource] list |

### Example

​```bash
curl -s "$BASE/[path]" | jq .
​```

### Full Example

​```bash
curl -s "$BASE/[path]?page=1&page_size=20&search=term&sort=-created_at&[filter]=value" | jq .
​```
```

## GET with Path Parameter

Used for single-resource retrieval by ID.

```markdown
## Find [Resource]

`GET [full path]/{id}`

[Description.]

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | [Resource] UUID |

### Responses

| Status | Description |
|--------|-------------|
| 200 | [Resource] found |
| 404 | [Resource] not found |

### Example

​```bash
curl -s "$BASE/[path]/550e8400-e29b-41d4-a716-446655440000" | jq .
​```
```

## POST with JSON Body

Used for search endpoints and create/update operations with structured data.

```markdown
## [Action] [Resource]

`POST [full path]`

[Description.]

### Request

Content-Type: `application/json`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| [field] | [type] | [yes/no] | [Description] |

### Responses

| Status | Description |
|--------|-------------|
| [200/201] | [Success description] |
| 400 | Invalid request body |

### Example

​```bash
curl -s -X POST "$BASE/[path]" \
  -H "Content-Type: application/json" \
  -d '{"field": "value"}' | jq .
​```

### Full Example

​```bash
curl -s -X POST "$BASE/[path]" \
  -H "Content-Type: application/json" \
  -d '{
    "field1": "value1",
    "field2": "value2",
    "field3": 42
  }' | jq .
​```
```

## POST with Multipart Form Data

Used for file upload endpoints.

```markdown
## Upload [Resource]

`POST [full path]`

[Description.]

### Request

Content-Type: `multipart/form-data`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| file | file | yes | [File description] |
| [field] | string | [yes/no] | [Metadata description] |

### Responses

| Status | Description |
|--------|-------------|
| 201 | [Resource] created |
| 400 | Invalid request |
| 413 | File exceeds maximum upload size |

### Example

​```bash
curl -s -X POST "$BASE/[path]" \
  -F "file=@path/to/file.pdf" \
  -F "field=value" | jq .
​```
```

## PUT/PATCH with JSON Body

Used for update operations.

```markdown
## Update [Resource]

`PUT [full path]/{id}`

[Description.]

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | [Resource] UUID |

### Request

Content-Type: `application/json`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| [field] | [type] | [yes/no] | [Description] |

### Responses

| Status | Description |
|--------|-------------|
| 200 | [Resource] updated |
| 400 | Invalid request body |
| 404 | [Resource] not found |

### Example

​```bash
curl -s -X PUT "$BASE/[path]/550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{"field": "new value"}' | jq .
​```
```

## DELETE with Path Parameter

Used for resource deletion.

```markdown
## Delete [Resource]

`DELETE [full path]/{id}`

[Description.]

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | [Resource] UUID |

### Responses

| Status | Description |
|--------|-------------|
| 204 | [Resource] deleted |
| 404 | [Resource] not found |

### Example

​```bash
curl -s -X DELETE "$BASE/[path]/550e8400-e29b-41d4-a716-446655440000"
​```
```

## Auth Header Appendage

When authentication is enabled for a project, any curl example can have the auth header appended. The root README documents the auth mechanism. Examples omit auth for brevity.

**Bearer token:**

```bash
curl -s -H "Authorization: Bearer $TOKEN_VAR" "$BASE/[path]" | jq .
```

**API key:**

```bash
curl -s -H "X-API-Key: $KEY_VAR" "$BASE/[path]" | jq .
```
