# MCP Tools Reference

mememory exposes 7 tools via the Model Context Protocol. This is the complete reference for all tools, parameters, and response formats.

## remember

Store a new memory. The content is embedded into a vector and persisted in PostgreSQL.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `content` | string | **yes** | — | The text to remember. Should be self-contained and specific. |
| `scope` | string | no | `"global"` | Visibility: `"global"` or `"project"` |
| `project` | string | no | — | Project name. Required when scope is `project`. |
| `type` | string | no | `"fact"` | Classification: `"fact"`, `"rule"`, `"decision"`, `"feedback"`, `"context"`, `"bootstrap"` |
| `tags` | string | no | — | Comma-separated tags. E.g. `"frontend, performance"` |
| `ttl` | string | no | — | Time-to-live duration. E.g. `"24h"`, `"7d"`, `"30d"`. Omit for permanent. |
| `weight` | number | no | `1.0` | Confidence/priority from 0.1 to 1.0. |
| `supersedes` | string | no | — | UUID of existing memory to replace. Old memory is auto-downgraded to weight 0.1. |

### Response

On success, returns the stored memory object:

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "content": "Uses PostgreSQL 17 with pgvector",
  "scope": "project",
  "project": "mememory",
  "type": "fact",
  "tags": ["database"],
  "weight": 1.0,
  "created_at": "2025-03-15T10:30:00Z",
  "updated_at": "2025-03-15T10:30:00Z"
}
```

### Contradiction Warning

If similar memories exist (similarity > 75%), the response includes a warning:

```
CONTRADICTION DETECTED

The memory was stored, but similar existing memories were found that may conflict.
Ask the user to clarify before proceeding.

New memory:
  [a1b2c3d4] Uses PostgreSQL 17 with pgvector

Potentially conflicting memories:
  [e5f6g7h8] (similarity: 82%) Uses SQLite for local storage

Options:
  1. Keep both — if they are complementary, not contradictory
  2. Update old — call update(id=<old_id>, content=<new_content>)
  3. Supersede — call remember(content=..., supersedes=<old_id>)
  4. Delete old — call forget(id=<old_id>)

Stored memory details:
{ ... full JSON ... }
```

### Bootstrap Size Warning

When storing a memory with `type="bootstrap"`, the server recomputes the combined bootstrap output for the relevant scope hierarchy. If the result exceeds **10KB** (`MaxBootstrapBytes`), the response is prefixed with a warning. The memory is still stored, but MCP clients may truncate the bootstrap output on the next session start. Remove or shorten some bootstrap memories to stay under the limit.

### Examples

```
# Global rule
remember(content="Never commit .env files", type="rule")

# Project fact
remember(content="React 19 + Vite + Tailwind", scope="project", project="match", type="fact")

# Bootstrap directive loaded at session start
remember(content="Always respond in Russian", type="bootstrap", scope="global")

# Temporary context with TTL
remember(content="Demo on Friday", scope="project", project="match", type="context", ttl="3d")

# Supersede old belief
remember(content="Zustand > Redux for this app", type="decision", supersedes="old-uuid-here")
```

---

## recall

Search memories by semantic similarity. Returns the most relevant memories matching the natural language query.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `query` | string | **yes** | — | Natural language search query |
| `scope` | string | no | — | Filter to specific scope. Omit for hierarchical search. |
| `project` | string | no | — | Project name. Enables hierarchical search (global + project). |
| `limit` | number | no | `5` | Maximum results to return |

### Hierarchical Search Behavior

| Parameters provided | Scopes searched |
|-------------------|-----------------|
| (none) | global only |
| `scope="global"` | global only |
| `project="X"` | global + project:X |
| `scope="project"` (no project name) | scope=project only |

### Response

Array of `{memory, score}` sorted by relevance:

```json
[
  {
    "memory": {
      "id": "a1b2c3d4-...",
      "content": "Uses PostgreSQL 17 with pgvector for vector storage",
      "scope": "project",
      "project": "mememory",
      "type": "fact",
      "weight": 1.0,
      "created_at": "2025-03-15T10:30:00Z",
      "updated_at": "2025-03-15T10:30:00Z"
    },
    "score": 0.897
  },
  {
    "memory": { ... },
    "score": 0.534
  }
]
```

The `score` is the final composite score (similarity x scope_weight x weight x temporal_decay), not raw cosine similarity.

### Examples

```
# Search globally
recall(query="database architecture")

# Search within a project (hierarchical: global + project)
recall(query="state management", project="match")

# Limit results
recall(query="deployment", limit=3)
```

---

## forget

Delete a memory permanently by its UUID.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | string | **yes** | Memory UUID to delete |

### Response

```
Memory a1b2c3d4-e5f6-7890-abcd-ef1234567890 deleted
```

### Example

```
forget(id="a1b2c3d4-e5f6-7890-abcd-ef1234567890")
```

---

## update

Update the content of an existing memory. The content is re-embedded for updated semantic search. All other fields (scope, type, tags, etc.) remain unchanged.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | string | **yes** | Memory UUID to update |
| `content` | string | **yes** | New content for the memory |

### Response

Returns the updated memory object:

```json
{
  "id": "a1b2c3d4-...",
  "content": "Updated content here",
  "scope": "project",
  "project": "mememory",
  "type": "fact",
  "weight": 1.0,
  "created_at": "2025-03-15T10:30:00Z",
  "updated_at": "2025-03-20T14:00:00Z"
}
```

Note that `updated_at` is refreshed, which resets the [temporal decay](/guide/scoring#temporal-decay).

### Example

```
update(id="a1b2c3d4-...", content="Uses PostgreSQL 17 with pgvector and HNSW indexing")
```

---

## list

Browse memories with exact metadata filters. No semantic search — returns all matching memories ordered by `updated_at` descending.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `scope` | string | no | — | Filter by scope: `"global"`, `"project"` |
| `project` | string | no | — | Filter by project name |
| `type` | string | no | — | Filter by type: `"fact"`, `"rule"`, `"decision"`, `"feedback"`, `"context"`, `"bootstrap"` |
| `limit` | number | no | `20` | Maximum results |

### Response

Array of memory objects:

```json
[
  {
    "id": "a1b2c3d4-...",
    "content": "Never commit .env files",
    "scope": "global",
    "type": "rule",
    "weight": 1.0,
    "created_at": "2025-01-10T08:00:00Z",
    "updated_at": "2025-01-10T08:00:00Z"
  }
]
```

### Examples

```
# All feedback
list(type="feedback")

# Project rules
list(scope="project", project="match", type="rule")

# All bootstrap directives
list(type="bootstrap")

# Recent 5 memories
list(limit=5)
```

---

## stats

Get memory statistics: total count and breakdown by scope, project, and type.

### Parameters

None.

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

### Example

```
stats()
```

---

## help

Get documentation on how to use the memory system. Returns usage guides, tool reference, scope taxonomy, and examples.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `topic` | string | no | full guide | Specific topic: `"overview"`, `"tools"`, `"scopes"`, `"types"`, `"examples"`, `"best-practices"` |

### Topics

| Topic | Content |
|-------|---------|
| (default) | Full guide — all sections combined |
| `"tools"` | Detailed reference for all 7 tools |
| `"scopes"` | Scope hierarchy explanation with examples |
| `"types"` | Memory type taxonomy with examples |
| `"examples"` | Usage examples for common scenarios |
| `"best-practices"` | Best practices for effective memory usage |

### Response

Markdown-formatted documentation text.

### Examples

```
# Full guide
help()

# Just tools reference
help(topic="tools")

# Best practices
help(topic="best-practices")
```

---

## MCP Resources

In addition to tools, mememory exposes two MCP resources for session bootstrap:

| Resource URI | Description |
|-------------|-------------|
| `mememory://bootstrap` | Global memories with `type=bootstrap`, formatted as Markdown |
| `mememory://bootstrap/{project}` | Global + project-scoped memories with `type=bootstrap`, formatted as Markdown |

Only memories with `type=bootstrap` are returned. Resources are read-only and return the same output format as the [bootstrap command](/guide/bootstrap#output-format), including the 10KB size limit. Client support for MCP resources varies.
