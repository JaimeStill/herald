# Classifications

`/api/classifications`

Classification results for documents. Stores, queries, validates, and updates classification results produced by the workflow engine.

---

## List Classifications

`GET /api/classifications`

Returns a paginated list of classifications with optional filters.

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| page | integer | no | Page number (1-indexed) |
| page_size | integer | no | Results per page |
| search | string | no | Search across classification and rationale |
| sort | string | no | Comma-separated sort fields, prefix `-` for descending |
| classification | string | no | Filter by classification (exact match) |
| confidence | string | no | Filter by confidence (exact match: HIGH, MEDIUM, LOW) |
| document_id | uuid | no | Filter by document ID (exact match) |
| validated_by | string | no | Filter by validator (exact match) |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Paginated classification list |

### Example

```bash
curl -s "$HERALD_API_BASE/api/classifications" | jq .
```

### Full Example

```bash
curl -s "$HERALD_API_BASE/api/classifications?page=1&page_size=20&search=SECRET&sort=-classified_at&classification=SECRET&confidence=HIGH&validated_by=admin" | jq .
```

---

## Find Classification

`GET /api/classifications/{id}`

Returns a single classification by UUID.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Classification UUID |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Classification found |
| 404 | Classification not found |

### Example

```bash
curl -s "$HERALD_API_BASE/api/classifications/550e8400-e29b-41d4-a716-446655440000" | jq .
```

---

## Find Classification by Document

`GET /api/classifications/document/{id}`

Returns the classification associated with a document.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Document UUID |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Classification found |
| 404 | No classification for this document |

### Example

```bash
curl -s "$HERALD_API_BASE/api/classifications/document/660e8400-e29b-41d4-a716-446655440000" | jq .
```

---

## Search Classifications

`POST /api/classifications/search`

Search classifications with a JSON body containing pagination and filter criteria.

### Request

Content-Type: `application/json`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| page | integer | no | Page number (1-indexed) |
| page_size | integer | no | Results per page |
| search | string | no | Search across classification and rationale |
| sort | string | no | Comma-separated sort fields, prefix `-` for descending |
| classification | string | no | Filter by classification |
| confidence | string | no | Filter by confidence |
| document_id | uuid | no | Filter by document ID |
| validated_by | string | no | Filter by validator |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Paginated search results |
| 400 | Invalid request body |

### Example

```bash
curl -s -X POST "$HERALD_API_BASE/api/classifications/search" \
  -H "Content-Type: application/json" \
  -d '{"page": 1, "page_size": 20}' | jq .
```

### Full Example

```bash
curl -s -X POST "$HERALD_API_BASE/api/classifications/search" \
  -H "Content-Type: application/json" \
  -d '{
    "page": 1,
    "page_size": 20,
    "search": "SECRET",
    "sort": "-classified_at",
    "classification": "SECRET",
    "confidence": "HIGH",
    "validated_by": "admin"
  }' | jq .
```

---

## Classify Document

`POST /api/classifications/{documentId}`

Executes the classification workflow for a single document. Runs the full workflow graph (init, classify, enhance?, finalize), upserts the classification result, and transitions the document status to `review`. Re-classification overwrites any existing result and resets validation fields.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| documentId | uuid | Document UUID to classify |

### Responses

| Status | Description |
|--------|-------------|
| 201 | Classification created |
| 404 | Document not found |

### Example

```bash
curl -s -X POST "$HERALD_API_BASE/api/classifications/660e8400-e29b-41d4-a716-446655440000" | jq .
```

---

## Validate Classification

`POST /api/classifications/{id}/validate`

Marks a classification as human-validated. The human agrees with the AI-produced classification. Transitions the associated document status from `review` to `complete`.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Classification UUID |

### Request

Content-Type: `application/json`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| validated_by | string | yes | Identifier of the human validator |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Classification validated |
| 404 | Classification not found |
| 409 | Document is not in review status |

### Example

```bash
curl -s -X POST "$HERALD_API_BASE/api/classifications/550e8400-e29b-41d4-a716-446655440000/validate" \
  -H "Content-Type: application/json" \
  -d '{"validated_by": "admin"}' | jq .
```

---

## Update Classification

`PUT /api/classifications/{id}`

Manually overwrites a classification's result. The human corrects the AI-produced classification and rationale. Transitions the associated document status from `review` to `complete`.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Classification UUID |

### Request

Content-Type: `application/json`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| classification | string | yes | Corrected classification marking |
| rationale | string | yes | Corrected rationale |
| updated_by | string | yes | Identifier of the human reviewer |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Classification updated |
| 404 | Classification not found |
| 409 | Document is not in review status |

### Example

```bash
curl -s -X PUT "$HERALD_API_BASE/api/classifications/550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{
    "classification": "TOP SECRET",
    "rationale": "Banner markings on pages 1 and 3 indicate TOP SECRET//SCI.",
    "updated_by": "reviewer"
  }' | jq .
```

---

## Delete Classification

`DELETE /api/classifications/{id}`

Deletes a classification.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Classification UUID |

### Responses

| Status | Description |
|--------|-------------|
| 204 | Classification deleted |
| 404 | Classification not found |

### Example

```bash
curl -s -X DELETE "$HERALD_API_BASE/api/classifications/550e8400-e29b-41d4-a716-446655440000"
```
