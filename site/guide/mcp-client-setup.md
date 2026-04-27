# MCP Client Setup

mememory uses the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) to communicate with AI agents. The MCP server runs via stdio transport — the client launches the server process and communicates over stdin/stdout.

## Claude Code

Claude Code is the recommended client. It supports both MCP tools and SessionStart hooks for automatic bootstrap.

### Step 1: Add MCP server

Add to `~/.claude/.mcp.json` (global) or `.mcp.json` (project-level):

```json
{
  "mcpServers": {
    "mememory": {
      "type": "stdio",
      "command": "docker",
      "args": ["exec", "-i", "mememory", "server"],
      "env": {}
    }
  }
}
```

This tells Claude Code to launch the MCP server inside the running Docker container. The `server` binary handles MCP communication over stdio.

### Step 2: Install hooks

The fastest way is the bundled installer:

```bash
mememory install-hooks
```

This patches `~/.claude/settings.json` with **four** hooks: `SessionStart` (loads bootstrap), `UserPromptSubmit` (reinjects pinned rules every turn), `PreToolUse` (blocks tools until the agent recalls in this session), and `PostToolUse` (clears the recall lock after recall completes). Existing settings are preserved; a timestamped backup is written before any change. Run `mememory install-hooks --uninstall` to remove cleanly.

If you prefer to edit `settings.json` by hand, the post-install structure is:

```json
{
  "hooks": {
    "SessionStart": [
      {"matcher": "", "hooks": [{"type": "command", "command": "mememory bootstrap --hook"}]}
    ],
    "UserPromptSubmit": [
      {"matcher": "", "hooks": [{"type": "command", "command": "mememory pinned --hook"}]}
    ],
    "PreToolUse": [
      {"matcher": "", "hooks": [{"type": "command", "command": "mememory recall-gate"}]}
    ],
    "PostToolUse": [
      {"matcher": "mcp__mememory__recall", "hooks": [{"type": "command", "command": "mememory recall-ack"}]}
    ]
  }
}
```

What each one does:
- `SessionStart` — runs `mememory bootstrap --hook`. Loads bootstrap memories into the system prompt and arms the recall-pending lock for this session.
- `UserPromptSubmit` — runs `mememory pinned --hook`. Renders pinned-delivery memories wrapped in `<system-reminder>` and injects them on every agent turn.
- `PreToolUse` — runs `mememory recall-gate`. Blocks any tool that isn't an `mcp__mememory__*` tool while the recall-pending lock exists.
- `PostToolUse` (matcher: `mcp__mememory__recall`) — runs `mememory recall-ack`. Removes the lock after the agent's first recall, freeing all subsequent tools.

For the full picture of what pinned and forced recall do, see [Pinned Rules & Forced Recall](/guide/pinned).

### Step 3: Verify

Start a new Claude Code session and check:

1. The bootstrap output should appear in the conversation (rules, feedback, etc.)
2. Ask the agent: "What memory tools do you have?" — it should list `remember`, `recall`, `forget`, `update`, `list`, `stats`, `help`
3. Test: "Remember that this project uses VitePress for documentation" — should succeed

### Project-specific config

For project-level MCP config, create `.mcp.json` in the project root:

```json
{
  "mcpServers": {
    "mememory": {
      "type": "stdio",
      "command": "docker",
      "args": ["exec", "-i", "mememory", "server"],
      "env": {}
    }
  }
}
```

## OpenAI Codex CLI

Codex supports the same `SessionStart` hook protocol as Claude Code, plus MCP servers over stdio. Configure both for the best experience.

### Step 1: Register the MCP server

Add to `~/.codex/config.toml`:

```toml
[features]
codex_hooks = true

[mcp_servers.mememory]
command = "docker"
args = ["exec", "-i", "mememory", "server"]
```

The `codex_hooks` feature flag is required to enable SessionStart hooks in current Codex releases.

### Step 2: Add SessionStart hook

Add to `~/.codex/hooks.json`:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "*",
        "hooks": [
          {
            "type": "command",
            "command": "mememory bootstrap --hook"
          }
        ]
      }
    ]
  }
}
```

The `--hook` flag is critical. Without it, `mememory bootstrap` prints raw Markdown to stdout, and Codex will dump the entire payload into the user's terminal instead of injecting it silently into the model context. With `--hook`, Codex parses the JSON envelope and treats `additionalContext` as invisible developer context — identical behaviour to Claude Code.

### Step 3: Verify

Start a new Codex session and check:

1. The terminal should be clean — no bootstrap payload printed.
2. Ask the agent about a fact you have stored in mememory (e.g. "what do you know about my project conventions?") — it should answer from the injected context.
3. Ask: "What MCP tools do you have for memory?" — it should list `remember`, `recall`, `forget`, `update`, `list`, `stats`.

## Claude Desktop

Claude Desktop supports MCP servers through its configuration file.

### macOS

Edit `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "mememory": {
      "command": "docker",
      "args": ["exec", "-i", "mememory", "server"]
    }
  }
}
```

### Windows

Edit `%APPDATA%\Claude\claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "mememory": {
      "command": "docker",
      "args": ["exec", "-i", "mememory", "server"]
    }
  }
}
```

### Linux

Edit `~/.config/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "mememory": {
      "command": "docker",
      "args": ["exec", "-i", "mememory", "server"]
    }
  }
}
```

::: tip
Claude Desktop does not support SessionStart hooks. The MCP server exposes bootstrap data as MCP resources (`mememory://bootstrap` and `mememory://bootstrap/{project}`) which some clients can read at connection time.
:::

## Generic MCP Clients

Any MCP-compatible client can connect to mememory. The server uses stdio transport:

**Command:** `docker exec -i mememory server`

**Capabilities advertised:**
- Tools: 7 tools (`remember`, `recall`, `forget`, `update`, `list`, `stats`, `help`)
- Resources: `mememory://bootstrap`, `mememory://bootstrap/{project}`

### Native binary (alternative)

If you built the binary locally instead of using Docker:

```json
{
  "mcpServers": {
    "mememory": {
      "command": "/path/to/mememory-server",
      "env": {
        "DATABASE_URL": "postgres://mememory:memory@localhost:5432/mememory?sslmode=disable",
        "OLLAMA_URL": "http://localhost:11434"
      }
    }
  }
}
```

When running the binary directly, you must provide `DATABASE_URL` and `OLLAMA_URL` (or `EMBEDDING_*` variables) since the binary is not inside the Docker network.

## Troubleshooting

### "docker exec" fails

Ensure the Docker stack is running:

```bash
docker compose -f docker/docker-compose.yml -p mememory ps
```

All three containers (`mememory-postgres`, `mememory-ollama`, `mememory`) should show `Up (healthy)`.

### MCP tools not appearing

Some clients cache the tool list. Restart the client after adding the MCP server configuration.

### Bootstrap produces no output

This is normal if no memories are stored yet. Store some memories first, then restart the session.

### "embed: request failed" errors

The Ollama service may not be ready yet. Check:

```bash
docker logs mememory-ollama
```

The model download may still be in progress on first run.
