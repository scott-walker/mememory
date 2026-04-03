<div align="center">

# MEMEMORY

**Persistent semantic memory for AI agents**

Store, search, and deliver knowledge across sessions. MCP server with PostgreSQL + pgvector. All data stays local.

[![CI](https://github.com/scott-walker/mememory/actions/workflows/ci.yml/badge.svg)](https://github.com/scott-walker/mememory/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/scott-walker/mememory)](https://github.com/scott-walker/mememory/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/scott-walker/mememory)](https://goreportcard.com/report/github.com/scott-walker/mememory)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[Documentation](https://scott-walker.github.io/mememory/) · [Quick Start](#quick-start) · [MCP Tools](#mcp-tools) · [Releases](https://github.com/scott-walker/mememory/releases)

</div>

---

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

Add to Claude Code config (`~/.claude/.claude.json` → `mcpServers`):

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

## Architecture

```
Agent ──stdio──▶ memory-server (Go, MCP)
                      │
              ┌───────┴───────┐
              ▼               ▼
         PostgreSQL       Ollama / OpenAI
        (pgvector)       (embeddings)
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
| `stats` | Count breakdown by scope/project/type |
| `help` | Usage documentation |

## Key Concepts

**Scopes** — global (everywhere), project (one project), persona (one agent role within a project). Recall searches hierarchically: persona sees global + project + own.

**Types** — rule, feedback, fact, decision, context. Rules and feedback load automatically at session start.

**Scoring** — `similarity × scope_weight × memory_weight × temporal_decay`. Recent, specific, high-weight memories rank higher.

**Contradiction detection** — warns when a new memory is >75% similar to existing ones. Does not block storage.

**Session bootstrap** — all global rules and feedback are sent to the agent as MCP instructions at connection time. No config needed per project.

## Embedding Providers

| Provider | Model | Dimension | Setup |
|----------|-------|-----------|-------|
| **Ollama** (default) | nomic-embed-text | 768 | Included in Docker stack |
| **OpenAI** | text-embedding-3-small | 1536 | Set `EMBEDDING_PROVIDER=openai` + API key |

## Documentation

- [Full Documentation](https://scott-walker.github.io/mememory/) — guides, reference, API
- [Architecture](docs/architecture.md) — system design, data flow
- [Memory Model](docs/memory-model.md) — scopes, types, scoring, belief evolution
- [MCP Tools Reference](docs/mcp-tools.md) — all tools and parameters
- [Setup Guide](docs/setup.md) — installation, configuration, development

## License

MIT
