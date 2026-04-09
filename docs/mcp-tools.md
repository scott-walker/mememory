# MCP Tools Reference

## remember

Store a new memory with semantic embedding.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `content` | string | yes | — | The text to remember |
| `scope` | string | no | `global` | `global` or `project` |
| `project` | string | no | — | Project name (required for project scope) |
| `type` | string | no | `fact` | `fact`, `rule`, `decision`, `feedback`, `context`, `bootstrap` |
| `tags` | string | no | — | Comma-separated tags |
| `ttl` | string | no | — | Auto-expire duration: `24h`, `7d`, `30d` |
| `weight` | number | no | 1.0 | Priority 0.1–1.0 |
| `supersedes` | string | no | — | UUID of memory this replaces |

Returns the stored memory. Warns if contradictions detected (similarity > 75%). When storing a `bootstrap`-type memory, also warns if the total bootstrap output would exceed `MaxBootstrapTokens` (30_000 tokens, ≈15% of a 200K-token context window).

## recall

Semantic search with hierarchical scope inheritance.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `query` | string | yes | — | Natural language search query |
| `scope` | string | no | — | Filter by scope (omit for hierarchical search) |
| `project` | string | no | — | Enable project-level search |
| `limit` | number | no | 5 | Max results |

Returns scored results. Score = similarity x scope_weight x memory_weight x temporal_decay.

Hierarchical behavior:
- `recall(query, project="X")` searches global + project:X
- `recall(query)` searches global only

## forget

Delete a memory permanently.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | string | yes | Memory UUID |

## update

Replace content of an existing memory. Re-embeds the new content, preserves all metadata.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | string | yes | Memory UUID |
| `content` | string | yes | New content |

## list

Browse memories with exact filters. No semantic search — returns all matching records.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `scope` | string | no | — | Filter by scope |
| `project` | string | no | — | Filter by project |
| `type` | string | no | — | Filter by type |
| `limit` | number | no | 20 | Max results |

## stats

Memory count breakdown. No parameters.

Returns: `{total, by_scope, by_project, by_type}`

## help

Usage documentation. Optional `topic` parameter: `overview`, `tools`, `scopes`, `types`, `examples`, `best-practices`.

## MCP Resources

| URI | Description |
|-----|-------------|
| `mememory://bootstrap` | Global memories with `type=bootstrap`. Loaded at session start. |
| `mememory://bootstrap/{project}` | Global + project-scoped `bootstrap` memories for a specific project. |

Only memories with `type=bootstrap` are returned by these resources. All other types must be retrieved with `recall`.

## Admin API

REST API at `http://localhost:4200/api` for the web admin UI.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/stats` | Memory statistics |
| GET | `/api/memories` | List with filters (?scope=&project=&type=&limit=) |
| GET | `/api/memories/:id` | Get by ID |
| POST | `/api/memories` | Create (same as remember) |
| PUT | `/api/memories/:id` | Update content |
| DELETE | `/api/memories/:id` | Delete one |
| DELETE | `/api/memories` | Bulk delete with filters |
| POST | `/api/memories/search` | Semantic search (same as recall) |
| POST | `/api/memories/export` | Export all as JSON |
| POST | `/api/memories/import` | Bulk import from JSON |
