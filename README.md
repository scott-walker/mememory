# MEMEMORY

Persistent semantic memory for AI agents. MCP server that stores, searches, and delivers knowledge across sessions. All data stays local.

## What it does

- **Stores** memories with scope, type, weight, tags, and TTL
- **Searches** by semantic similarity with hierarchical scope inheritance
- **Delivers** accumulated rules and knowledge to agents at session start
- **Detects** contradictions when new memories conflict with existing ones
- **Evolves** beliefs through supersede/weight mechanisms without losing history

## Quick Start

```bash
git clone https://github.com/scott-walker/mememory.git
cd mememory
cp .env.example .env
docker compose -f docker/docker-compose.yml up -d
```

Add to Claude Code config (`~/.claude/.claude.json` -> `mcpServers`):

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

Admin UI at `http://localhost:4200`.

## Stack

```
Agent ──stdio──> memory-server (Go)
                      │
              ┌───────┴───────┐
              ▼               ▼
         PostgreSQL       Ollama
       (pgvector)    (nomic-embed-text)
```

One `docker compose up` — no Go, Node.js, or other toolchains needed.

## MCP Tools

| Tool | Description |
|------|-------------|
| `remember` | Store a memory with scope, type, tags, optional TTL |
| `recall` | Semantic search with hierarchical scope inheritance |
| `forget` | Delete by ID |
| `update` | Update content, re-embed |
| `list` | List with filters |
| `stats` | Count breakdown |

## Key Concepts

**Scopes** — global (everywhere), project (one project), persona (one agent role within a project). Recall searches hierarchically: persona sees global + project + own.

**Types** — rule, feedback, fact, decision, context. Rules and feedback load automatically at session start.

**Scoring** — `similarity x scope_weight x memory_weight x temporal_decay`. Recent, specific, high-weight memories rank higher.

**Contradiction detection** — warns when a new memory is >75% similar to existing ones. Does not block storage.

**Session bootstrap** — all global rules and feedback are sent to the agent as MCP instructions at connection time. No config needed per project.

## Documentation

- [Architecture](docs/architecture.md) — system design, data flow, infrastructure
- [Memory Model](docs/memory-model.md) — scopes, types, scoring algorithm, belief evolution
- [MCP Tools Reference](docs/mcp-tools.md) — all tools, parameters, API endpoints
- [Setup Guide](docs/setup.md) — installation, configuration, development

## License

MIT
