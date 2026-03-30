# mememory

MCP server for persistent semantic memory in Claude Code. Stores memories as vectors in Qdrant, uses Ollama for local embeddings. No data leaves your machine.

## Architecture

```
Claude Code ──stdio──▶ memory-server (Go, MCP)
                            │
                    ┌───────┴───────┐
                    ▼               ▼
               Qdrant          Ollama
            (vector DB)    (nomic-embed-text)
```

Three scope levels with hierarchical inheritance:
- **global** — visible everywhere
- **project** — visible within a specific project
- **persona** — visible to a specific agent persona within a project

`recall(persona=X, project=Y)` searches global + project:Y + persona:X.

## Setup

### 1. Start the stack

```bash
git clone git@github.com:scott-walker/mememory.git
cd mememory
cp .env.example .env    # optionally edit MEMORY_DATA_DIR, ports
docker compose -f docker/docker-compose.yml up -d
```

First start pulls the embedding model (~274 MB), subsequent starts are instant.

Data is stored in `$MEMORY_DATA_DIR` (default `~/.claude-memory/`).

### 2. Connect to Claude Code

Add to your Claude Code config (`~/.claude/.claude.json` → `mcpServers`):

```json
{
  "memory": {
    "type": "stdio",
    "command": "docker",
    "args": ["exec", "-i", "claude-memory-admin", "memory-server"],
    "env": {}
  }
}
```

Or copy `.mcp.json.example` into your project as `.mcp.json` for per-project setup.

## MCP Tools

| Tool | Description |
|------|-------------|
| `remember` | Store a memory with scope, type, tags, optional TTL |
| `recall` | Semantic search with hierarchical scope inheritance |
| `forget` | Delete a memory by ID |
| `update` | Update content and re-embed |
| `list` | List memories with filters (scope, project, persona, type) |
| `stats` | Count breakdown by scope, project, persona, type |
| `help` | Show usage instructions |

### Parameters

**remember:**
- `content` (required) — text to store
- `scope` — `global` | `project` | `persona` (default: `global`)
- `project` — project name (required for project/persona scope)
- `persona` — agent persona name (required for persona scope)
- `type` — `fact` | `rule` | `decision` | `feedback` | `context` (default: `fact`)
- `tags` — comma-separated tags
- `ttl` — auto-expire duration, e.g. `24h`, `7d`
- `weight` — priority 0.1–1.0 (default: 1.0)
- `supersedes` — ID of memory this one replaces

**recall:**
- `query` (required) — natural language search query
- `scope` — filter by scope (omit for hierarchical search)
- `project` — filter by project
- `persona` — filter by persona
- `limit` — max results (default: 5)

## Admin UI

Web interface at `http://localhost:4200` for browsing, searching, and managing memories.

## Configuration

Environment variables (set in `.env`):

| Variable | Default | Description |
|----------|---------|-------------|
| `MEMORY_DATA_DIR` | `~/.claude-memory` | Persistent storage directory |
| `QDRANT_PORT_REST` | `6333` | Qdrant REST API port |
| `QDRANT_PORT_GRPC` | `6334` | Qdrant gRPC port |
| `OLLAMA_PORT` | `11434` | Ollama API port |
| `ADMIN_PORT` | `4200` | Admin UI port |

## Development

```bash
make infra-up     # start Qdrant + Ollama containers
make dev          # run MCP server locally (go run)
make build        # build binary → bin/memory-server
make admin-dev    # run admin UI in dev mode (hot reload)
```

## Stack

- Go — MCP server + admin API
- Qdrant — vector database
- Ollama — local embeddings (nomic-embed-text, 768d)
- React + TypeScript — admin UI

## License

MIT
