# Documents

`/api/documents`

Document upload and management.

---

## List Documents

`GET /api/documents`

Returns a paginated list of documents with optional filters.

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| page | integer | no | Page number (1-indexed) |
| page_size | integer | no | Results per page |
| search | string | no | Search across filename and external_platform |
| sort | string | no | Comma-separated sort fields, prefix `-` for descending |
| status | string | no | Filter by status (pending, review, complete) |
| filename | string | no | Filter by filename (contains, case-insensitive) |
| external_id | integer | no | Filter by external ID (exact match) |
| external_platform | string | no | Filter by external platform (exact match) |
| content_type | string | no | Filter by content type (exact match) |
| classification | string | no | Filter by classification level (exact match) |
| confidence | string | no | Filter by confidence (exact match: HIGH, MEDIUM, LOW) |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Paginated document list |

### Example

```bash
curl -s "$HERALD_API_BASE/api/documents" | jq .
```

### Full Example

```bash
curl -s "$HERALD_API_BASE/api/documents?page=1&page_size=20&search=report&sort=-uploaded_at&status=pending&filename=quarterly&external_platform=HQ&content_type=application/pdf&classification=SECRET&confidence=HIGH" | jq .
```

---

## Find Document

`GET /api/documents/{id}`

Returns a single document by UUID.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Document UUID |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Document found |
| 404 | Document not found |

### Example

```bash
curl -s "$HERALD_API_BASE/api/documents/550e8400-e29b-41d4-a716-446655440000" | jq .
```

---

## Search Documents

`POST /api/documents/search`

Search documents with a JSON body containing pagination and filter criteria.

### Request

Content-Type: `application/json`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| page | integer | no | Page number (1-indexed) |
| page_size | integer | no | Results per page |
| search | string | no | Search across filename and external_platform |
| sort | string | no | Comma-separated sort fields, prefix `-` for descending |
| status | string | no | Filter by status |
| filename | string | no | Filter by filename (contains) |
| external_id | integer | no | Filter by external ID |
| external_platform | string | no | Filter by external platform |
| content_type | string | no | Filter by content type |
| storage_key | string | no | Filter by storage key (contains) |
| classification | string | no | Filter by classification level |
| confidence | string | no | Filter by confidence (HIGH, MEDIUM, LOW) |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Paginated search results |
| 400 | Invalid request body |

### Example

```bash
curl -s -X POST "$HERALD_API_BASE/api/documents/search" \
  -H "Content-Type: application/json" \
  -d '{"page": 1, "page_size": 20}' | jq .
```

### Full Example

```bash
curl -s -X POST "$HERALD_API_BASE/api/documents/search" \
  -H "Content-Type: application/json" \
  -d '{
    "page": 1,
    "page_size": 20,
    "search": "quarterly",
    "sort": "-uploaded_at",
    "status": "pending",
    "filename": "report",
    "external_id": 12345,
    "external_platform": "HQ",
    "content_type": "application/pdf",
    "storage_key": "documents/",
    "classification": "SECRET",
    "confidence": "HIGH"
  }' | jq .
```

---

## Upload Document

`POST /api/documents`

Upload a single document file with external system metadata. The server detects the content type and validates it against the registered format handlers (PDF, PNG, JPEG, WEBP). PDF uploads have their page count extracted automatically via pdfcpu; image uploads record a null `page_count`.

### Request

Content-Type: `multipart/form-data`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| file | file | yes | Document to upload. Accepted content types: `application/pdf`, `image/png`, `image/jpeg`, `image/webp` |
| external_id | string | yes | External system record ID |
| external_platform | string | yes | External system platform identifier |

### Responses

| Status | Description |
|--------|-------------|
| 201 | Document created |
| 400 | Invalid request (missing fields, bad external_id, or unsupported content type — response body cites the supported set) |
| 413 | File exceeds maximum upload size |

### Example

```bash
curl -s -X POST "$HERALD_API_BASE/api/documents" \
  -F "file=@_project/marked-documents/single-secret.pdf" \
  -F "external_id=12345" \
  -F "external_platform=HQ" | jq .
```

### Upload Raw Image

```bash
curl -s -X POST "$HERALD_API_BASE/api/documents" \
  -F "file=@_project/marked-documents/images/marked-document.1.png" \
  -F "external_id=12346" \
  -F "external_platform=HQ" | jq .
```

### Rejected Upload (unsupported content type)

```bash
curl -s -X POST "$HERALD_API_BASE/api/documents" \
  -F "file=@some-file.docx" \
  -F "external_id=12347" \
  -F "external_platform=HQ"
# HTTP/1.1 400 Bad Request
# {"error":"unsupported content type: application/msword (supported: application/pdf, image/jpeg, image/png, image/webp)"}
```

---

## Delete Document

`DELETE /api/documents/{id}`

Deletes a document and its associated blob from storage.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| id | uuid | Document UUID |

### Responses

| Status | Description |
|--------|-------------|
| 204 | Document deleted |
| 404 | Document not found |

### Example

```bash
curl -s -X DELETE "$HERALD_API_BASE/api/documents/550e8400-e29b-41d4-a716-446655440000"
```
