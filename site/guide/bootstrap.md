# Session Bootstrap

Session bootstrap solves the cold-start problem: when an agent starts a new session, it has no memory of past interactions. Bootstrap loads accumulated rules, feedback, facts, decisions, and context into the agent's system prompt at session start.

## How It Works

```
Agent session starts
    ↓
SessionStart hook runs mememory bootstrap
    ↓
CLI queries Admin API for memories matching scope hierarchy
    ↓
Memories are formatted as Markdown, grouped by type
    ↓
Output is injected into the agent's context
    ↓
Agent starts with full memory from the first message
```

No Ollama or embedding computation is needed for bootstrap — it reads directly from the database via the Admin API.

## Bootstrap Methods

There are two ways to run bootstrap:

### 1. mememory CLI (recommended)

The native `mememory` binary runs on the host machine and calls the Admin API over HTTP:

```bash
# Global memories only
mememory bootstrap

# Global + project-scoped memories (auto-detect project from git)
mememory bootstrap

# Global + specific project
mememory bootstrap --project myapp

# Global + project + persona
mememory bootstrap --project myapp --persona reviewer
```

::: tip
When `--project` is omitted, the CLI auto-detects the project name from the current git repository's root directory name. If not inside a git repo, it falls back to the current directory name.
:::

### 2. memory-server --bootstrap (legacy)

The MCP server binary also supports bootstrap mode. It connects directly to PostgreSQL (no Admin API needed):

```bash
# Global only
memory-server --bootstrap

# Global + project
memory-server --bootstrap --project myapp

# Global + project + persona
memory-server --bootstrap --project myapp --persona dev
```

This mode is useful inside Docker or when the Admin API is not running. However, the `mememory` CLI is preferred because it auto-detects the project and handles API connectivity gracefully (silent exit if unreachable).

## Hook Configuration

### Claude Code

Add a `SessionStart` hook in your Claude Code settings (`~/.claude/settings.json`):

```json
{
  "hooks": {
    "SessionStart": [
      {
        "type": "command",
        "command": "mememory bootstrap"
      }
    ]
  }
}
```

This runs `mememory bootstrap` every time a new Claude Code session starts. The project is auto-detected from the working directory's git root.

For a specific persona:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "type": "command",
        "command": "mememory bootstrap --persona architect"
      }
    ]
  }
}
```

### Custom URL

If the Admin API runs on a non-default port or host:

```bash
mememory bootstrap --url http://localhost:9000
```

Or set the `MEMORY_URL` environment variable:

```bash
export MEMORY_URL=http://my-server:4200
```

## Output Format

Bootstrap output is Markdown, grouped by type in priority order:

1. **Rules** — imperatives that must be followed
2. **Feedback** — user corrections to agent behavior
3. **Facts** — objective information
4. **Decisions** — choices with reasoning
5. **Context** — temporal/situational information

### Example output

```markdown
# Memory Bootstrap

The following memories were loaded from persistent storage. Apply rules and feedback to your behavior.

Project: match

## Rules

- [global] Never commit .env files to version control
- [global] pnpm only, no npm or yarn
- [project/match] Never use native <select> elements — only custom dropdowns

## Feedback

- [global] Don't refactor without explicit permission
- [global] Stop summarizing what you just did — user can read the diff

## Facts

- [project/match] Uses React 19 + Vite + Tailwind CSS 4
- [project/match] SQLite with better-sqlite3, no ORM

## Decisions

- [project/match] Chose Zustand over Redux — simpler API, sufficient for app size

## Context

- [project/match] Preparing for investor demo on April 5
```

When a project is specified, each memory is prefixed with a scope label (`[global]`, `[project/match]`, `[persona/match/reviewer]`) so the agent can distinguish where the knowledge came from.

When no project is specified (global-only bootstrap), scope labels are omitted.

## MCP Resources (Alternative)

Bootstrap data is also available as MCP resources, which some clients can read at connection time:

| Resource URI | Content |
|-------------|---------|
| `memory://bootstrap` | Global memories only |
| `memory://bootstrap/{project}` | Global + project-scoped memories |

These resources return the same formatted Markdown as the CLI. However, MCP resource support varies by client — the `SessionStart` hook approach is more reliable.

## Filtering

Bootstrap loads all non-expired memories for the specified scope hierarchy. Memories with:

- Expired TTL are excluded
- Weight below any threshold are still included (even `weight=0.1`)

There is no semantic search during bootstrap — all memories matching the scope are loaded, sorted by `updated_at` descending, up to 100 per scope level.

## Silent Failure

If the Admin API is unreachable, the `mememory` CLI exits silently with no output. The agent starts without memory context rather than failing with an error. This is intentional — bootstrap should never block session start.
