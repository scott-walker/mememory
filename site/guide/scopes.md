# Scopes & Hierarchy

Scopes control which memories are visible to which projects. mememory uses a two-level hierarchy that enables knowledge sharing while preserving specificity.

## The Two Scopes

### global

Visible to **all projects**. Use for universal knowledge.

| Good for | Examples |
|----------|----------|
| User identity | "User's name is Scott" |
| Universal preferences | "Respond in Russian" |
| Cross-project rules | "Never commit .env files" |
| Workflow preferences | "Don't refactor without asking" |

```
remember(
  content="Never commit .env files to version control",
  type="rule",
  scope="global"
)
```

### project

Visible only within a **named project**. Requires the `project` parameter.

| Good for | Examples |
|----------|----------|
| Architecture | "Uses SQLite with better-sqlite3, no ORM" |
| Tech stack | "React 19 + Vite + Tailwind CSS 4" |
| Project decisions | "Chose Zustand for state management" |
| Active context | "Preparing for demo on April 5" |

```
remember(
  content="Uses PostgreSQL 17 with pgvector for vector storage",
  type="fact",
  scope="project",
  project="mememory"
)
```

## Hierarchical Search

The key feature of scopes is **hierarchical inheritance** during recall. Project scope automatically includes global.

### How it works

```
recall(query="database architecture")
→ searches: global only

recall(query="database architecture", project="match")
→ searches: global + project:match
```

### Visual diagram

```
┌─────────────────────────────────────────────┐
│                  global                      │
│  "Never commit .env files"                   │
│  "User's name is Scott"                      │
│                                              │
│  ┌───────────────────────────────────────┐   │
│  │          project: match               │   │
│  │  "Uses SQLite with better-sqlite3"    │   │
│  │  "React 19 + Vite + Tailwind"         │   │
│  └───────────────────────────────────────┘   │
│                                              │
│  ┌───────────────────────────────────────┐   │
│  │         project: mememory             │   │
│  │  "Uses PostgreSQL + pgvector"         │   │
│  │  "Go monorepo with Docker stack"      │   │
│  └───────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
```

When an agent works within project `match`, it sees global + match memories. An agent in project `mememory` sees global + mememory memories. Neither sees the other's project-scoped content.

## SQL Filter Implementation

Under the hood, hierarchical search generates an OR-based WHERE clause:

```sql
-- recall(project="match")
SELECT *, 1 - (embedding <=> $1) AS score
FROM memories
WHERE (
    scope = 'global'
    OR (scope = 'project' AND project = 'match')
)
ORDER BY embedding <=> $1
LIMIT 15
```

This single query searches both scope levels simultaneously, ranked by vector similarity.

## Scope Weights

After retrieval, each result's similarity score is multiplied by a scope weight to prioritize more specific memories:

| Scope | Weight | Rationale |
|-------|--------|-----------|
| project | 1.0 | Most specific — highest priority |
| global | 0.8 | Universal — slight penalty in favor of local knowledge |

This means a project-level memory with 85% similarity can outrank a global memory with 95% similarity. Specific knowledge outranks general knowledge.

The full scoring formula is:

```
final_score = similarity x scope_weight x memory_weight x temporal_decay
```

See [Scoring & Recall](/guide/scoring) for the complete breakdown.

## Scope Override Pattern

Project-level memories can effectively override global rules. For example:

```
# Global rule
remember(content="Never use ORM", type="rule", scope="global")

# Project exception
remember(
  content="In convervox we use Drizzle ORM — complex schema justifies it",
  type="decision",
  scope="project",
  project="convervox",
  supersedes="<id-of-global-no-orm-rule>"
)
```

The supersede mechanism combined with scope weights ensures the project-specific decision wins within the `convervox` project, while the global rule remains active for all other projects.

## Choosing the Right Scope

| Question | Scope |
|----------|-------|
| Would this apply in ANY project? | `global` |
| Only in THIS project? | `project` |

::: warning
Avoid over-scoping. If a rule applies globally, store it as global. Do not duplicate the same memory across multiple projects.
:::
