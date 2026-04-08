# Scoring & Recall

When you call `recall`, mememory does not simply return the most similar vectors. Results go through a multi-stage pipeline that considers scope, confidence, and recency to produce a ranked list of the most relevant memories.

## Scoring Formula

Every recall result is scored by:

```
final_score = similarity x scope_weight x memory_weight x temporal_decay
```

Each component:

| Component | Range | Source |
|-----------|-------|--------|
| `similarity` | 0.0 - 1.0 | Cosine similarity between query embedding and memory embedding |
| `scope_weight` | 0.8 - 1.0 | Determined by the memory's scope level |
| `memory_weight` | 0.1 - 1.0 | Explicitly set by the user via `weight` parameter |
| `temporal_decay` | 0.0 - 1.0 | Exponential decay based on time since last update |

## Similarity

The raw similarity score comes from PostgreSQL's pgvector cosine distance:

```sql
1 - (embedding <=> query_vector) AS score
```

This measures how semantically close the memory content is to the query. A score of 1.0 means identical meaning, 0.0 means completely unrelated.

## Scope Weight

More specific memories are prioritized over general ones:

| Scope | Weight | Effect |
|-------|--------|--------|
| `project` | 1.0 | No penalty — project memories get full score |
| `global` | 0.8 | 20% reduction — global knowledge yields to local |

This ensures that a project-level architecture decision outranks a loosely related global fact, even if the global fact has slightly higher vector similarity.

## Memory Weight

The user-specified `weight` parameter (default 1.0) acts as a confidence multiplier:

| Weight | Use case |
|--------|----------|
| 1.0 | Confident, current knowledge (default) |
| 0.5 | Uncertain or tentative beliefs |
| 0.3 | Partially outdated but still relevant |
| 0.1 | Auto-downgraded by `supersedes` mechanism |

Weight is the only scoring component directly controlled by the user. Use it to express confidence levels without deleting or hiding memories.

## Temporal Decay

Newer memories score slightly higher than older ones. The decay function is a gentle exponential:

```
decay = e^(-lambda * days_since_update)
```

Where `lambda = 0.005`. This produces:

| Age | Decay factor | Effect |
|-----|-------------|--------|
| 1 day | 0.995 | Negligible — virtually full score |
| 7 days | 0.966 | ~3% reduction |
| 30 days | 0.861 | ~14% reduction |
| 90 days | 0.638 | ~36% reduction |
| 180 days | 0.407 | ~59% reduction |
| 365 days | 0.161 | ~84% reduction |

::: tip
The decay is intentionally gentle. A 30-day-old memory still retains 86% of its score. This prevents recent but irrelevant memories from drowning out older but highly relevant ones.
:::

The decay is based on `updated_at`, not `created_at`. Calling `update` on a memory resets its temporal decay.

## Recall Pipeline

The full recall process, step by step:

### Step 1: Embed the query

The query string is converted to a vector embedding using the configured [embedding provider](/guide/embedding-providers).

### Step 2: Fetch candidates

A vector similarity search runs against PostgreSQL with a hierarchical WHERE clause (see [Scopes](/guide/scopes#sql-filter-implementation)). The system fetches `3x` the requested limit (minimum 15) to have enough candidates for re-ranking.

### Step 3: Filter expired

Memories with a TTL that has passed are removed from results.

### Step 4: Filter superseded

Memories whose ID appears as the `supersedes` target of another result are removed. This implements belief evolution — only the latest belief in a chain is shown.

### Step 5: Score and rank

Each remaining memory gets its final score:

```
final_score = similarity x scope_weight(memory.scope) x memory.weight x temporal_decay(now - memory.updated_at)
```

Results are sorted by `final_score` descending.

### Step 6: Trim to limit

The top N results (default 5) are returned.

## Worked Example

Suppose you query `recall(query="state management", project="match", limit=3)`:

| Memory | Similarity | Scope | Weight | Age | Final Score |
|--------|-----------|-------|--------|-----|-------------|
| "Uses Zustand for stores" (project:match) | 0.92 | project (1.0) | 1.0 | 5d (0.975) | **0.897** |
| "Prefer Redux for large apps" (global) | 0.95 | global (0.8) | 1.0 | 60d (0.741) | **0.563** |
| "State management is complex" (global) | 0.88 | global (0.8) | 0.5 | 2d (0.990) | **0.349** |

The project-specific Zustand memory ranks first despite lower raw similarity, because scope weight and recency push it ahead.

## Contradiction Detection

When calling `remember`, the system checks for potential conflicts before storing the new memory.

### How it works

1. The new content is embedded
2. A hierarchical search is run against existing memories (same scope hierarchy)
3. Any existing memory with similarity > **0.75** (75%) is flagged as a potential contradiction

### What happens on detection

The memory is still stored — contradictions do not block storage. But the response includes:

- A warning message
- The list of conflicting memories with similarity percentages
- Resolution options: keep both, update old, supersede, or delete old

```
CONTRADICTION DETECTED

New memory:
  [a1b2c3d4] Zustand is better than Redux for small apps

Potentially conflicting memories:
  [e5f6g7h8] (similarity: 82%) Redux is the best state manager

Options:
  1. Keep both — if they are complementary
  2. Update old — call update(id=<old_id>, content=...)
  3. Supersede — call remember(content=..., supersedes=<old_id>)
  4. Delete old — call forget(id=<old_id>)
```

::: warning
When an agent receives a contradiction warning, it should always ask the user for clarification before choosing a resolution. Never silently keep contradicting memories.
:::

### Threshold

The contradiction threshold is **0.75** (75% cosine similarity). This value balances between catching real conflicts and avoiding false positives for merely related content.
