# Setup

## Requirements

- PostgreSQL >= 14 with the `pgvector` extension
- Docker and Docker Compose (only for the bundled quick-start path)
- An MCP-compatible client (Claude Code, or any MCP client)

You can either bring your own Postgres or let `mememory setup` bring up the bundled Docker stack (Postgres with pgvector + Ollama).

## Quick Start (bundled stack)

```bash
go install github.com/scott-walker/mememory/cmd/mememory@latest
mememory setup
```

`mememory setup` extracts the bundled Docker Compose file, resolves a data directory, and runs `docker compose up -d`. No source checkout required. First start pulls the embedding model (~274 MB).

When the stack is up, `mememory setup` interactively asks whether to install the four Claude Code hooks (SessionStart, UserPromptSubmit, PreToolUse, PostToolUse) into `~/.claude/settings.json`. Skip the prompt and run `mememory install-hooks` later — the result is identical.

## Bring your own Postgres

`DATABASE_URL` is **required** — there is no fallback. If unset, the server exits immediately with a clear hint.

```bash
export DATABASE_URL=postgres://user:pass@your-host:5432/mememory?sslmode=disable
```

The `pgvector` extension is verified at startup. If missing, the server fails fast with installation instructions.

### Connect your MCP client

Add to your Claude Code config (`~/.claude/.mcp.json`, inside `mcpServers`):

```json
{
  "mememory": {
    "type": "stdio",
    "command": "docker",
    "args": ["exec", "-i", "mememory", "server"],
    "env": {}
  }
}
```

Or create `.mcp.json` in your project root for per-project setup.

### Enable session bootstrap

Add a `SessionStart` hook to your Claude Code settings (`settings.json`) so that stored memories are loaded into every new session automatically:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
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

### Verify

Start a new Claude Code session. The agent should have access to `remember`, `recall`, `forget`, `update`, `list`, `stats`, and `help` tools.

Admin UI is at `http://localhost:4200`.

## Configuration

All configuration is via environment variables in `.env`:

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | _required_ | PostgreSQL connection string. No fallback — server fails fast if unset. |
| `DATA_DIR` | OS-standard path | Persistent data directory. CLI auto-resolves if unset. |
| `POSTGRES_PORT` | `5432` | PostgreSQL host port (bundled stack) |
| `OLLAMA_PORT` | `11434` | Ollama API port |
| `ADMIN_PORT` | `4200` | Admin UI port |

## Where your data lives

If `DATA_DIR` is not set, the `mememory` CLI auto-resolves to an OS-standard path:

| Platform | Default |
|----------|---------|
| Linux    | `~/.local/share/mememory` (or `$XDG_DATA_HOME/mememory`) |
| macOS    | `~/Library/Application Support/mememory` |
| Windows  | `%LOCALAPPDATA%\mememory` |

Inside that directory:

```
mememory/
  postgres/    PostgreSQL data files
  ollama/      Embedding model files (~274 MB)
```

## Backup

Two equivalent options:

- **File copy:** copy the entire `DATA_DIR` while containers are stopped.
- **Logical dump:** `pg_dump postgres://mememory:mememory@localhost:5432/mememory > backup.sql`

## Port Conflicts

If port 5432 is already taken, set `POSTGRES_PORT=5434` in `.env`. The internal container port stays 5432 — only the host mapping changes. Same for Ollama (11434) and Admin UI (4200).

## Stack Commands

```bash
# Start
mememory setup

# Stop (data preserved)
mememory uninstall

# Stop and delete data (requires interactive confirmation)
mememory uninstall --purge
```

## Development

For working on the server itself:

```bash
make infra-up     # Start PostgreSQL + Ollama
make dev          # Run MCP server locally (go run)
make build        # Build binary -> bin/server
make admin-dev    # Admin UI with hot reload
```

Requires Go 1.22+ and Node.js 22+ for development.
