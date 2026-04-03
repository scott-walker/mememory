# MCP Client Setup

mememory uses the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) to communicate with AI agents. The MCP server runs via stdio transport — the client launches the server process and communicates over stdin/stdout.

## Claude Code

Claude Code is the recommended client. It supports both MCP tools and SessionStart hooks for automatic bootstrap.

### Step 1: Add MCP server

Add to `~/.claude/.claude.json` (global) or `.claude.json` (project-level):

```json
{
  "mcpServers": {
    "memory": {
      "type": "stdio",
      "command": "docker",
      "args": ["exec", "-i", "mememory-admin", "memory-server"],
      "env": {}
    }
  }
}
```

This tells Claude Code to launch the MCP server inside the running Docker container. The `memory-server` binary handles MCP communication over stdio.

### Step 2: Add SessionStart hook

Add to `~/.claude/settings.json`:

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

This runs `mememory bootstrap` at the start of every session, injecting accumulated rules and context into the agent's system prompt. The project is auto-detected from the working directory.

### Step 3: Verify

Start a new Claude Code session and check:

1. The bootstrap output should appear in the conversation (rules, feedback, etc.)
2. Ask the agent: "What memory tools do you have?" — it should list `remember`, `recall`, `forget`, `update`, `list`, `stats`, `help`
3. Test: "Remember that this project uses VitePress for documentation" — should succeed

### Project-specific config

For project-level MCP config, create `.claude.json` in the project root:

```json
{
  "mcpServers": {
    "memory": {
      "type": "stdio",
      "command": "docker",
      "args": ["exec", "-i", "mememory-admin", "memory-server"],
      "env": {}
    }
  }
}
```

For project-specific bootstrap with a persona:

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

## Claude Desktop

Claude Desktop supports MCP servers through its configuration file.

### macOS

Edit `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "memory": {
      "command": "docker",
      "args": ["exec", "-i", "mememory-admin", "memory-server"]
    }
  }
}
```

### Windows

Edit `%APPDATA%\Claude\claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "memory": {
      "command": "docker",
      "args": ["exec", "-i", "mememory-admin", "memory-server"]
    }
  }
}
```

### Linux

Edit `~/.config/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "memory": {
      "command": "docker",
      "args": ["exec", "-i", "mememory-admin", "memory-server"]
    }
  }
}
```

::: tip
Claude Desktop does not support SessionStart hooks. The MCP server exposes bootstrap data as MCP resources (`memory://bootstrap` and `memory://bootstrap/{project}`) which some clients can read at connection time.
:::

## Generic MCP Clients

Any MCP-compatible client can connect to mememory. The server uses stdio transport:

**Command:** `docker exec -i mememory-admin memory-server`

**Capabilities advertised:**
- Tools: 7 tools (`remember`, `recall`, `forget`, `update`, `list`, `stats`, `help`)
- Resources: `memory://bootstrap`, `memory://bootstrap/{project}`

### Native binary (alternative)

If you built the binary locally instead of using Docker:

```json
{
  "mcpServers": {
    "memory": {
      "command": "/path/to/memory-server",
      "env": {
        "DATABASE_URL": "postgres://memory:memory@localhost:5432/memory?sslmode=disable",
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

All three containers (`mememory-postgres`, `mememory-ollama`, `mememory-admin`) should show `Up (healthy)`.

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
