# Memory Model

## What is a Memory

A memory is a piece of knowledge stored as text with a 768-dimensional embedding vector for semantic search. Each memory has metadata that controls its visibility, priority, and lifecycle.

```
Memory {
  id          UUID
  content     text          — the knowledge itself
  embedding   vector(768)   — computed by Ollama (nomic-embed-text)
  scope       enum          — global | project | persona
  project     text?         — project name (for project/persona scope)
  persona     text?         — agent persona name (for persona scope)
  type        enum          — fact | rule | decision | feedback | context
  tags        text[]        — free-form labels for filtering
  weight      float 0.1-1.0 — priority/confidence (default 1.0)
  supersedes  UUID?         — ID of memory this one replaces
  created_at  timestamp
  updated_at  timestamp
  ttl         timestamp?    — auto-expire time (null = permanent)
}
```

## Scopes and Hierarchy

Scopes control who sees what. They form a hierarchy — higher scopes inherit lower:

```
┌─────────────────────────────────────────────┐
│  global                                      │
│  Visible to all projects, all personas       │
│                                              │
│  ┌────────────────────────────────────────┐  │
│  │  project: "match"                      │  │
│  │  Visible only within project "match"   │  │
│  │                                        │  │
│  │  ┌─────────────────────────────────┐   │  │
│  │  │  persona: "architect"           │   │  │
│  │  │  Visible only to this persona   │   │  │
│  │  └─────────────────────────────────┘   │  │
│  └────────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

When an agent recalls memories with `project="match"` and `persona="architect"`, the search covers all three levels. Project-scoped results automatically outrank global ones through scope weighting.

**SQL filter for hierarchical search:**

```sql
WHERE (scope = 'global')
   OR (scope = 'project' AND project = 'match')
   OR (scope = 'persona' AND persona = 'architect' AND project = 'match')
```

## Memory Types

Types classify the nature of knowledge. They affect bootstrap ordering (rules load first) and help agents decide how to act on a memory.

| Type | Purpose | Example |
|------|---------|---------|
| `rule` | Imperative — must be followed | "Never add Co-Authored-By to commits" |
| `feedback` | Correction to agent behavior | "User prefers terse responses, no summaries" |
| `fact` | Objective information | "User's name is Scott" |
| `decision` | A choice with reasoning | "Chose pgvector over Qdrant for simpler infrastructure" |
| `context` | Temporal/situational info | "Code freeze starts March 5" (use with TTL) |

## Recall Scoring

When memories are retrieved, each result gets a composite score:

```
score = similarity × scope_weight × memory_weight × temporal_decay
```

### Semantic Similarity

Cosine similarity between the query embedding and the memory embedding, computed by PostgreSQL's pgvector:

```sql
1 - (embedding <=> query_vector)  -- result ∈ [0, 1]
```

### Scope Weight

More specific scopes score higher — a project rule outranks a global rule for the same topic:

| Scope | Weight |
|-------|--------|
| persona | 1.0 |
| project | 0.8 |
| global | 0.6 |

### Memory Weight

User-defined priority (0.1 to 1.0). Default is 1.0. Superseded memories drop to 0.1 automatically. Useful for:

- Downgrading outdated beliefs without deleting them
- Expressing uncertainty: `weight=0.5` for "might be the case"

### Temporal Decay

Gentle exponential decay — older memories score slightly lower:

```
decay = e^(-0.005 × days_old)
```

| Age | Decay Factor |
|-----|-------------|
| 1 day | 0.995 |
| 30 days | 0.861 |
| 90 days | 0.638 |
| 365 days | 0.161 |

This means recent memories are mildly preferred. The decay is gentle enough that important old memories still surface if their semantic match is strong.

### Recall Pipeline

1. Embed the query via Ollama
2. Fetch 3× the requested limit from PostgreSQL (compensates for filtering)
3. Filter out expired memories (TTL < now)
4. Filter out superseded memories (any memory whose ID appears in another's `supersedes` field)
5. Apply composite score formula
6. Sort by score descending
7. Return top N

## Contradiction Detection

When storing a new memory, the server searches for existing memories with similarity > 0.75 in the same scope hierarchy. If found, it returns a warning:

```
⚠ CONTRADICTION DETECTED

New memory:
  [abc123] Use PostgreSQL for everything

Potentially conflicting memories:
  [def456] (similarity: 82%) Use Qdrant for vector search

Options:
  1. Keep both — if complementary
  2. Update old — fix the old memory's content
  3. Supersede — remember(supersedes="def456")
  4. Delete old — forget("def456")
```

The new memory is still stored — the warning is informational, not blocking. The agent (or user) decides how to resolve it.

## Belief Evolution

Memories are not immutable. Knowledge changes over time:

- **Supersede**: `remember(content="new truth", supersedes="old_id")` — old memory drops to weight 0.1, new one takes over. History is preserved.
- **Update**: `update(id, content)` — re-embeds and overwrites content. Used for factual corrections.
- **Forget**: `forget(id)` — permanent deletion. Used for removing noise.
- **TTL**: `remember(content="...", ttl="7d")` — auto-expires. Used for temporary context (sprint goals, deadlines, incidents).

## Session Bootstrap

At the start of every session, a Claude Code `SessionStart` hook runs `memory-server --bootstrap`. This reads all global memories from PostgreSQL, formats them as Markdown, and prints to stdout. The hook injects this output into the agent's context automatically.

- Rules and feedback are in the agent's context from the first message
- No explicit `recall` needed for base principles
- New rules take effect in the next session automatically
- No size limits — hook output is not truncated like MCP `instructions`

Bootstrap groups memories by type in priority order: **Rules > Feedback > Facts > Decisions > Context**.

Configure the hook in Claude Code settings (`settings.json`):

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "docker exec mememory-admin memory-server --bootstrap"
          }
        ]
      }
    ]
  }
}
```
