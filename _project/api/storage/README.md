# Storage

`/api/storage`

Read-only Azure Blob Storage queries. Lists blobs, retrieves metadata, and downloads files directly from blob storage without going through the SQL layer.

---

## List Blobs

`GET /api/storage`

Returns a page of blobs with optional prefix filtering and marker-based pagination.

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| prefix | string | no | Filter blobs by name prefix |
| marker | string | no | Opaque continuation token from a previous response |
| max_results | integer | no | Maximum items per page (default 50, max 5000) |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Blob listing with optional next_marker for pagination |
| 400 | Invalid max_results parameter |

### Example

```bash
curl -s "$HERALD_API_BASE/api/storage" | jq .
```

### Full Example

```bash
curl -s "$HERALD_API_BASE/api/storage?prefix=documents/&max_results=20&marker=abc123" | jq .
```

---

## Find Blob

`GET /api/storage/{key...}`

Returns metadata for a single blob by storage key.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| key | string (wildcard) | Blob storage key (may contain slashes) |

### Responses

| Status | Description |
|--------|-------------|
| 200 | Blob metadata |
| 400 | Invalid key |
| 404 | Blob not found |

### Example

```bash
curl -s "$HERALD_API_BASE/api/storage/documents/550e8400-e29b-41d4-a716-446655440000/report.pdf" | jq .
```

---

## Download Blob

`GET /api/storage/download/{key...}`

Downloads a file by storage key. Streams the blob with appropriate Content-Type and Content-Disposition headers.

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| key | string (wildcard) | Blob storage key (may contain slashes) |

### Responses

| Status | Description |
|--------|-------------|
| 200 | File stream with Content-Type, Content-Length, and Content-Disposition headers |
| 400 | Invalid key |
| 404 | Blob not found |

### Example

```bash
curl -s "$HERALD_API_BASE/api/storage/download/documents/550e8400-e29b-41d4-a716-446655440000/report.pdf" -o report.pdf
```
