# Memory Model

A memory is the fundamental unit of knowledge in mememory. Each memory is a piece of text that gets embedded as a vector and stored in PostgreSQL with pgvector, making it searchable by semantic similarity.

## Memory Structure

Every memory has these fields:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `id` | UUID | auto | generated | Unique identifier |
| `content` | string | yes | — | The knowledge to store. Should be self-contained. |
| `scope` | string | no | `"global"` | Visibility level: `global`, `project` |
| `project` | string | no | — | Project name. Required when scope is `project`. |
| `type` | string | no | `"fact"` | Content classification: `fact`, `rule`, `decision`, `feedback`, `context`, `bootstrap` |
| `tags` | string[] | no | — | Free-form labels for additional filtering |
| `weight` | float | no | `1.0` | Confidence/priority from 0.1 to 1.0 |
| `supersedes` | UUID | no | — | ID of the memory this one replaces |
| `ttl` | duration | no | — | Time-to-live. Memory auto-expires after this period. |
| `created_at` | timestamp | auto | now | When the memory was first stored |
| `updated_at` | timestamp | auto | now | When the memory was last modified |

## Memory Types

The `type` field classifies what kind of knowledge the memory represents. This affects how memories are grouped during [session bootstrap](/guide/bootstrap) and helps agents understand how to use the information.

### fact

Objective, verifiable information about the world, the project, or the user.

```
remember(
  content="The database uses PostgreSQL 17 with pgvector extension",
  type="fact",
  scope="project",
  project="mememory"
)
```

```
remember(
  content="User's name is Scott",
  type="fact",
  scope="global"
)
```

**When to use:** storing technical facts, architecture details, user identity, tool versions, repository structure.

### rule

Imperatives that must be followed. Include the reasoning when possible, so agents understand why the rule exists.

```
remember(
  content="Never use native <select> elements — only custom dropdowns with search",
  type="rule",
  scope="project",
  project="match"
)
```

```
remember(
  content="pnpm only, no npm or yarn — CI enforces lockfile consistency",
  type="rule",
  scope="global"
)
```

**When to use:** coding standards, tool restrictions, workflow requirements, mandatory patterns. Rules are loaded first during bootstrap and carry the highest behavioral weight.

### decision

A choice that was made, with reasoning. Future agents need to understand not just _what_ was decided, but _why_.

```
remember(
  content="Chose Zustand over Redux for state management — simpler API, less boilerplate, sufficient for app size",
  type="decision",
  scope="project",
  project="match"
)
```

```
remember(
  content="Sequential data collection (OU first, then BU) because OFI knows OU but BU contact comes from OU. Prevents routing errors.",
  type="decision",
  scope="project",
  project="remide"
)
```

**When to use:** architecture choices, library selections, design tradeoffs, process decisions. Always include the "why".

### feedback

User corrections to agent behavior. This is the most valuable memory type — it prevents agents from repeating mistakes across all future sessions.

```
remember(
  content="Don't refactor code without explicit permission. Minimal diffs only.",
  type="feedback",
  scope="global"
)
```

```
remember(
  content="Stop summarizing what you just did — user can read the diff",
  type="feedback",
  scope="global"
)
```

**When to use:** every time the user corrects the agent's behavior, style, or approach. Feedback is loaded automatically during bootstrap, right after rules.

### context

Temporal or situational information that is relevant now but may not be permanent. Often used with TTL.

```
remember(
  content="Preparing for investor demo on April 5. All UIs must be production-ready.",
  type="context",
  scope="project",
  project="match",
  ttl="7d"
)
```

```
remember(
  content="Currently refactoring auth flow — don't touch auth-store.ts",
  type="context",
  scope="project",
  project="match",
  ttl="3d"
)
```

**When to use:** sprint goals, deadlines, temporary constraints, active refactoring warnings, feature flags. Set `ttl` so context auto-expires when it becomes irrelevant.

### bootstrap

Essential directives that must be present in the agent's context from the first message of every session. Only memories with `type=bootstrap` are loaded by the [session bootstrap](/guide/bootstrap) mechanism — all other types are retrieved on demand via `recall`.

```
remember(
  content="Always respond in Russian",
  type="bootstrap",
  scope="global"
)
```

```
remember(
  content="Use pnpm exclusively. Never run npm install.",
  type="bootstrap",
  scope="project",
  project="match"
)
```

**When to use:** rules and directives that the agent must know immediately on session start, before it can safely take any action. Keep the set small — the combined bootstrap output is bounded by `MaxBootstrapTokens` (30,000 tokens, ≈15% of a 200K-token context window) and exists specifically to ensure session-start memory never crowds out the actual conversation. Overflow is reported in the in-payload `## Bootstrap Stats` block but is not truncated.

## Weight

The `weight` field (0.1 to 1.0, default 1.0) controls how much influence a memory has during [recall scoring](/guide/scoring). Use it for:

- **Uncertain knowledge** — store with `weight=0.5`, increase as confidence grows
- **Partially outdated beliefs** — lower to 0.3 instead of deleting, preserving history
- **Auto-downgraded memories** — when a memory is superseded, its weight drops to 0.1 automatically

```
remember(
  content="GraphQL might be better than REST for this project — not sure yet",
  type="decision",
  scope="project",
  project="match",
  weight=0.5,
  tags="tentative"
)
```

## Supersedes (Belief Evolution)

When the user's understanding changes, use `supersedes` to create a clean chain of belief evolution rather than adding conflicting memories.

```
# Old memory exists: id="abc123", content="Redux is the best state manager"

remember(
  content="Zustand is better than Redux for small-medium apps — simpler API, less boilerplate",
  type="decision",
  scope="global",
  supersedes="abc123"
)
```

What happens:
1. The new memory is stored normally
2. The old memory (`abc123`) is auto-downgraded to `weight=0.1`
3. The old memory is excluded from recall results (superseded IDs are filtered out)
4. History is preserved — the old memory still exists in the database

## TTL (Auto-Expiry)

Set a time-to-live duration to auto-expire temporary knowledge. Supported formats:

| Format | Duration |
|--------|----------|
| `"24h"` | 24 hours |
| `"7d"` | 7 days |
| `"30d"` | 30 days |
| `"720h"` | 720 hours (30 days) |

Expired memories are filtered from recall and list results, and periodically cleaned from the database (every hour).

## Tags

Free-form labels for cross-cutting concerns. Tags enable filtering without changing scope or type.

```
remember(
  content="All grays must be metallic (R < G < B) — brand requirement",
  type="rule",
  scope="project",
  project="match",
  tags="frontend, design, brand"
)
```

Tags are stored as an array and can be used in `list` filters. They are comma-separated when passed via MCP tools.

## Contradiction Detection

When you call `remember`, the system automatically checks for semantically similar existing memories. If similarity exceeds 75%, a contradiction warning is returned with the conflicting memories and resolution options.

See [Scoring & Recall](/guide/scoring#contradiction-detection) for details on the threshold and algorithm.
