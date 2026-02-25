# API Reference

## Configuration

| Setting | Value |
|---------|-------|
| Base Variable | `HERALD_API_BASE` |
| Default Value | `http://localhost:8080` |
| Organization | Route groups by URL path prefix |
| Auth | None (bearer token planned for Phase 4) |

## Setup

```bash
export HERALD_API_BASE="http://localhost:8080"
```

## Route Groups

| Group | Path Prefix | Description |
|-------|-------------|-------------|
| [Documents](documents/) | `/api/documents` | Document upload and management |
| [Prompts](prompts/) | `/api/prompts` | Prompt instruction overrides per workflow stage |
| [Storage](storage/) | `/api/storage` | Read-only blob storage queries |

## Root Endpoints

### Health Check

`GET /healthz`

Returns service health status.

#### Responses

| Status | Description |
|--------|-------------|
| 200 | Service is healthy |

#### Example

```bash
curl -s "$HERALD_API_BASE/healthz" | jq .
```

---

### Readiness Check

`GET /readyz`

Returns whether the service is ready to accept requests.

#### Responses

| Status | Description |
|--------|-------------|
| 200 | Service is ready |
| 503 | Service is not ready |

#### Example

```bash
curl -s "$HERALD_API_BASE/readyz" | jq .
```
