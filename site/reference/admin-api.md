# Admin API Reference

The Admin API is a REST API served by the `admin` binary on port 4200 (configurable via `ADMIN_PORT`). It provides CRUD operations for memories, search, export/import, and statistics.

Base URL: `http://localhost:4200`

All endpoints accept and return JSON. The `Content-Type: application/json` header is set by default.

## Error Format

All errors return a JSON object with an `error` field:

```json
{
  "error": "description of what went wrong"
}
```

HTTP status codes:

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created (POST /api/memories) |
| 400 | Bad request (invalid JSON, missing fields) |
| 404 | Not found |
| 500 | Internal server error |

---

## GET /api/stats

Get memory statistics.

### Request

No parameters.

```bash
curl http://localhost:4200/api/stats
```

### Response

```json
{
  "total": 42,
  "by_scope": {
    "global": 20,
    "project": 22
  },
  "by_project": {
    "match": 13,
    "mememory": 9
  },
  "by_type": {
    "fact": 12,
    "rule": 10,
    "decision": 8,
    "feedback": 7,
    "context": 3,
    "bootstrap": 2
  }
}
```

---

## GET /api/memories

List memories with optional filters.

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `scope` | string | — | Filter by scope: `global`, `project` |
| `project` | string | — | Filter by project name |
| `type` | string | — | Filter by type: `fact`, `rule`, `decision`, `feedback`, `context`, `bootstrap` |
| `limit` | integer | `50` | Maximum results |

### Request

```bash
# All memories (limit 50)
curl http://localhost:4200/api/memories

# Project rules
curl "http://localhost:4200/api/memories?scope=project&project=match&type=rule"

# Global feedback
curl "http://localhost:4200/api/memories?scope=global&type=feedback&limit=10"
```

### Response

```json
[
  {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "content": "Never commit .env files",
    "scope": "global",
    "type": "rule",
    "tags": ["security"],
    "weight": 1.0,
    "created_at": "2025-01-10T08:00:00Z",
    "updated_at": "2025-01-10T08:00:00Z"
  }
]
```

Results are ordered by `updated_at` descending. Expired memories (TTL past) are excluded.

---

## GET /api/memories/:id

Get a single memory by UUID.

### Request

```bash
curl http://localhost:4200/api/memories/a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

### Response

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "content": "Never commit .env files",
  "scope": "global",
  "type": "rule",
  "tags": ["security"],
  "weight": 1.0,
  "created_at": "2025-01-10T08:00:00Z",
  "updated_at": "2025-01-10T08:00:00Z"
}
```

Returns `404` if the memory does not exist.

---

## POST /api/memories

Create a new memory. The content is embedded and stored with its vector.

### Request Body

```json
{
  "content": "Uses PostgreSQL 17 with pgvector",
  "scope": "project",
  "project": "mememory",
  "type": "fact",
  "tags": ["database"],
  "weight": 1.0,
  "ttl": "30d",
  "supersedes": ""
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `content` | string | **yes** | — | Text to store |
| `scope` | string | no | `"global"` | `"global"`, `"project"` |
| `project` | string | no | — | Project name |
| `type` | string | no | `"fact"` | `"fact"`, `"rule"`, `"decision"`, `"feedback"`, `"context"`, `"bootstrap"` |
| `tags` | string[] | no | — | Array of tag strings |
| `weight` | number | no | `1.0` | 0.1 to 1.0 |
| `ttl` | string | no | — | Duration string, e.g. `"24h"`, `"7d"` |
| `supersedes` | string | no | — | UUID of memory to replace |

### Request

```bash
curl http://localhost:4200/api/memories \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Uses PostgreSQL 17 with pgvector",
    "scope": "project",
    "project": "mememory",
    "type": "fact"
  }'
```

### Response (201)

```json
{
  "memory": {
    "id": "a1b2c3d4-...",
    "content": "Uses PostgreSQL 17 with pgvector",
    "scope": "project",
    "project": "mememory",
    "type": "fact",
    "weight": 1.0,
    "created_at": "2025-03-15T10:30:00Z",
    "updated_at": "2025-03-15T10:30:00Z"
  },
  "contradictions": [
    {
      "memory": {
        "id": "old-uuid-...",
        "content": "Uses SQLite for storage",
        "scope": "project",
        "project": "mememory",
        "type": "fact"
      },
      "similarity": 0.82
    }
  ]
}
```

The `contradictions` array is only present when similar memories exist (similarity > 0.75).

---

## PUT /api/memories/:id

Update a memory's content. Re-embeds the content for updated search.

### Request Body

```json
{
  "content": "Updated content here"
}
```

### Request

```bash
curl http://localhost:4200/api/memories/a1b2c3d4-e5f6-7890-abcd-ef1234567890 \
  -X PUT \
  -H "Content-Type: application/json" \
  -d '{"content": "Uses PostgreSQL 17 with pgvector and HNSW indexing"}'
```

### Response

```json
{
  "id": "a1b2c3d4-...",
  "content": "Uses PostgreSQL 17 with pgvector and HNSW indexing",
  "scope": "project",
  "project": "mememory",
  "type": "fact",
  "weight": 1.0,
  "created_at": "2025-03-15T10:30:00Z",
  "updated_at": "2025-03-20T14:00:00Z"
}
```

---

## DELETE /api/memories/:id

Delete a single memory by UUID.

### Request

```bash
curl http://localhost:4200/api/memories/a1b2c3d4-e5f6-7890-abcd-ef1234567890 \
  -X DELETE
```

### Response

```json
{
  "ok": true
}
```

---

## DELETE /api/memories

Bulk delete memories matching filter criteria.

### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `scope` | string | Filter by scope |
| `project` | string | Filter by project |
| `type` | string | Filter by type |

### Request

```bash
# Delete all context memories for a project
curl "http://localhost:4200/api/memories?scope=project&project=old-project&type=context" \
  -X DELETE
```

### Response

```json
{
  "deleted": 5
}
```

::: danger
Bulk delete with no filters will delete ALL memories (up to 1000). Use filters carefully.
:::

---

## POST /api/memories/search

Semantic search via the Admin API. Same as the `recall` MCP tool but accessible over HTTP.

### Request Body

```json
{
  "query": "database architecture",
  "scope": "",
  "project": "mememory",
  "limit": 5
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `query` | string | **yes** | — | Natural language search query |
| `scope` | string | no | — | Scope filter |
| `project` | string | no | — | Project filter (enables hierarchical search) |
| `limit` | integer | no | `5` | Maximum results |

### Request

```bash
curl http://localhost:4200/api/memories/search \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"query": "database architecture", "project": "mememory"}'
```

### Response

```json
[
  {
    "memory": {
      "id": "a1b2c3d4-...",
      "content": "Uses PostgreSQL 17 with pgvector",
      "scope": "project",
      "project": "mememory",
      "type": "fact",
      "weight": 1.0,
      "created_at": "2025-03-15T10:30:00Z",
      "updated_at": "2025-03-15T10:30:00Z"
    },
    "score": 0.718
  }
]
```

---

## POST /api/memories/export

Export all memories as JSON. Returns the full list of memories without vector embeddings.

### Request

```bash
curl http://localhost:4200/api/memories/export \
  -X POST \
  -o memories.json
```

### Response

The response includes a `Content-Disposition: attachment; filename=memories.json` header.

```json
[
  {
    "id": "a1b2c3d4-...",
    "content": "Never commit .env files",
    "scope": "global",
    "type": "rule",
    "tags": ["security"],
    "weight": 1.0,
    "created_at": "2025-01-10T08:00:00Z",
    "updated_at": "2025-01-10T08:00:00Z"
  }
]
```

Exports up to 10,000 memories.

---

## POST /api/memories/import

Import memories from a JSON array. Each memory is re-embedded through the current embedding provider and stored with a new UUID.

### Request Body

Array of memory input objects:

```json
[
  {
    "content": "Never commit .env files",
    "scope": "global",
    "type": "rule",
    "tags": ["security"],
    "weight": 1.0
  },
  {
    "content": "Uses React 19 + Vite",
    "scope": "project",
    "project": "match",
    "type": "fact"
  }
]
```

### Request

```bash
curl http://localhost:4200/api/memories/import \
  -X POST \
  -H "Content-Type: application/json" \
  -d @memories.json
```

### Response

```json
{
  "imported": 42
}
```

::: warning
Import re-embeds every memory. For cloud embedding providers (OpenAI), this incurs token costs proportional to the number of memories. For Ollama, it takes time proportional to memory count.
:::

Old IDs and `supersedes` references are not preserved during import. Each memory gets a new UUID.
