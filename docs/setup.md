# Setup

## Requirements

- Docker and Docker Compose
- An MCP-compatible client (Claude Code, or any MCP client)

No Go, Node.js, or other toolchains needed. Everything runs in containers.

## Quick Start

### 1. Clone and start

```bash
git clone https://github.com/scott-walker/mememory.git
cd mememory
cp .env.example .env    # edit if needed
docker compose -f docker/docker-compose.yml up -d
```

First start pulls the embedding model (~274 MB). Subsequent starts are instant.

### 2. Connect your MCP client

Add to your Claude Code config (`~/.claude/.claude.json`, inside `mcpServers`):

```json
{
  "memory": {
    "type": "stdio",
    "command": "docker",
    "args": ["exec", "-i", "mememory-admin", "memory-server"],
    "env": {}
  }
}
```

Or create `.mcp.json` in your project root for per-project setup.

### 3. Enable session bootstrap

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
            "command": "docker exec mememory-admin memory-server --bootstrap"
          }
        ]
      }
    ]
  }
}
```

### 4. Verify

Start a new Claude Code session. The agent should have access to `remember`, `recall`, `forget`, `update`, `list`, `stats`, and `help` tools.

Admin UI is at `http://localhost:4200`.

## Configuration

All configuration is via environment variables in `.env`:

| Variable | Default | Description |
|----------|---------|-------------|
| `MEMORY_DATA_DIR` | `~/.mememory` | Where data is stored on disk |
| `POSTGRES_PORT` | `5432` | PostgreSQL port (change if 5432 is taken) |
| `POSTGRES_PASSWORD` | `memory` | PostgreSQL password |
| `OLLAMA_PORT` | `11434` | Ollama API port |
| `ADMIN_PORT` | `4200` | Admin UI port |

## Data Storage

All persistent data lives in `$MEMORY_DATA_DIR`:

```
~/.mememory/
  postgres/    PostgreSQL data files
  ollama/      Embedding model files (~274 MB)
```

Backup: copy the directory. Restore: put it back.

## Port Conflicts

If port 5432 is already taken (e.g., another PostgreSQL instance), set `POSTGRES_PORT=5434` in `.env`. The internal container port stays 5432 — only the host mapping changes.

Same for Ollama (11434) and Admin UI (4200).

## Stack Commands

```bash
# Start
docker compose -f docker/docker-compose.yml --env-file .env up -d

# Stop (preserves data)
docker compose -f docker/docker-compose.yml down

# Rebuild after code changes
docker compose -f docker/docker-compose.yml --env-file .env up -d --build

# Full reset (destroys data)
docker compose -f docker/docker-compose.yml down -v
```

## Development

For working on the server itself:

```bash
make infra-up     # Start PostgreSQL + Ollama
make dev          # Run MCP server locally (go run)
make build        # Build binary -> bin/memory-server
make admin-dev    # Admin UI with hot reload
```

Requires Go 1.22+ and Node.js 22+ for development.
