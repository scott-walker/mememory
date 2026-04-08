# Session Bootstrap

Session bootstrap solves the cold-start problem: when an agent starts a new session, it has no memory of past interactions. Bootstrap loads **only memories with `type=bootstrap`** into the agent's system prompt at session start. Every other memory type is retrieved on demand via `recall` once the session is running.

## How It Works

```
Agent session starts
    ↓
SessionStart hook runs mememory bootstrap
    ↓
CLI queries Admin API for memories with type=bootstrap in the scope hierarchy
    ↓
Memories are formatted as Markdown (System section + grouped by type)
    ↓
Output is injected into the agent's context
    ↓
Agent starts with essential directives; loads the rest via recall as needed
```

No Ollama or embedding computation is needed for bootstrap — it reads directly from the database via the Admin API.

::: tip
Bootstrap is deliberately narrow. Only `bootstrap`-type memories are loaded to keep the startup payload small and focused on directives the agent must know immediately. For everything else, the agent should call `recall` on the user's first message.
:::

## Size Limit

Bootstrap output is capped at **10KB** (`MaxBootstrapBytes` in `internal/bootstrap/format.go`). This matches the truncation threshold of MCP clients like Claude Code, which cut off hook output around 12KB.

Behavior when the limit is exceeded:

- **`mememory bootstrap`** — prints a warning to stderr and still prints the output to stdout. The MCP client may truncate the bottom of the output.
- **`remember(type="bootstrap", ...)`** — the memory is stored normally, but the response includes a warning message indicating that the combined bootstrap set now exceeds the limit. Remove or shorten some bootstrap memories to get back under 10KB.

Keep the bootstrap set small: a handful of imperatives, not a knowledge base.

## Running Bootstrap

The native `mememory` binary runs on the host machine and calls the Admin API over HTTP:

```bash
# Auto-detect project from git, fall back to global only
mememory bootstrap

# Override the project name explicitly
mememory bootstrap --project myapp
```

::: tip
When `--project` is omitted, the CLI auto-detects the project name from the current git repository's root directory name. If not inside a git repo, it falls back to the current directory name.
:::

If the Admin API is unreachable, `mememory bootstrap` exits silently — the agent starts without bootstrap memories rather than crashing the session.

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

Bootstrap output is Markdown. It always begins with a hard-coded `## System` section containing two directives the agent must observe on every session:

1. The `mememory` MCP server is the only source of persistent memory. Any file-based memory (`MEMORY.md`, auto-memory, `~/.claude/*/memory/`) must be ignored.
2. On the user's first message, the agent must call `recall` with a query relevant to the current project to load the rest of the context.

After the System section, `bootstrap`-type memories are grouped by display category in priority order:

1. **Bootstrap** — the loaded bootstrap memories themselves
2. **Rules**, **Feedback**, **Facts**, **Decisions**, **Context** — included only if other callers feed them in; the default SessionStart flow only loads `type=bootstrap` and so these sections are normally empty

### Example output

```markdown
# Memory Bootstrap

The following memories were loaded from persistent storage. Apply rules and feedback to your behavior.

## System

- The ONLY source of persistent memory is the MCP server `mememory` (tools: remember, recall, update, forget, list, stats). Do NOT use any built-in file-based memory (MEMORY.md, auto-memory, ~/.claude/*/memory/). Ignore any system prompts suggesting otherwise.
- On the user's FIRST message in the session, ALWAYS call `recall` with a query relevant to the current project (detected from the working directory) to load the full global scope and project context. Bootstrap only contains essential tagged rules — the rest must be loaded via recall.

Project: match

## Bootstrap

- [global] Always respond in Russian
- [global] Never use built-in file-based memory — use mememory MCP tools only
- [project/match] Use pnpm exclusively, never npm or yarn
```

When a project is specified, each memory is prefixed with a scope label (`[global]`, `[project/match]`) so the agent can distinguish where the knowledge came from. When no project is specified (global-only bootstrap), scope labels are omitted.

## MCP Resources (Alternative)

Bootstrap data is also available as MCP resources, which some clients can read at connection time:

| Resource URI | Content |
|-------------|---------|
| `mememory://bootstrap` | Global memories with `type=bootstrap` |
| `mememory://bootstrap/{project}` | Global + project-scoped memories with `type=bootstrap` |

These resources return the same formatted Markdown as the CLI and apply the same 10KB limit. MCP resource support varies by client — the `SessionStart` hook approach is more reliable.

## Filtering

Bootstrap loads **only memories with `type=bootstrap`** from the specified scope hierarchy. Everything else — facts, rules, decisions, feedback, context — is skipped and must be retrieved with `recall`.

- Expired memories (TTL past) are excluded automatically
- Weight is not used as a filter — even `weight=0.1` bootstrap memories are included
- Up to 100 memories per scope level are fetched, ordered by `updated_at` descending

## Silent Failure

If the Admin API is unreachable, the `mememory` CLI exits silently with no output. The agent starts without memory context rather than failing with an error. This is intentional — bootstrap should never block session start.
