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

## Requirements

- PostgreSQL >= 14 with the [pgvector](https://github.com/pgvector/pgvector) extension
- (Optional) Docker, if you want the bundled quick-start stack

## Quick Start (bundled Docker stack)

```bash
git clone https://github.com/scott-walker/mememory.git
cd mememory
mememory setup
```

`mememory setup` resolves an OS-standard data directory, writes a `.env`, and brings up the Docker stack (Postgres with pgvector + Ollama + Admin UI).

## Quick Start (BYO Postgres)

If you already have a PostgreSQL >= 14 server with pgvector, just point `DATABASE_URL` at it. There is no fallback — the server fails fast if `DATABASE_URL` is unset.

```bash
export DATABASE_URL=postgres://user:pass@your-host:5432/mememory?sslmode=disable
mememory-server
```

The server runs `CREATE EXTENSION IF NOT EXISTS vector` at startup. If your DB user lacks `CREATE` privilege, ask your DBA to install pgvector beforehand.

Add to Claude Code config (`~/.claude/.mcp.json` → `mcpServers`):

```json
{
  "mememory": {
    "type": "stdio",
    "command": "docker",
    "args": ["exec", "-i", "mememory-admin", "mememory-server"],
    "env": {}
  }
}
```

Admin UI at `http://localhost:4200`.

## Architecture

```
Agent ──stdio──▶ mememory-server (Go, MCP)
                      │
              ┌───────┴───────┐
              ▼               ▼
         PostgreSQL       Ollama / OpenAI
        (pgvector)       (embeddings)
```

Bring your own Postgres, or run `mememory setup` for the bundled Docker quick-start.

## Where your data lives

If `DATA_DIR` is not set, the `mememory` CLI auto-resolves it to an OS-standard path:

| Platform | Default location |
|----------|------------------|
| Linux    | `~/.local/share/mememory` (or `$XDG_DATA_HOME/mememory`) |
| macOS    | `~/Library/Application Support/mememory` |
| Windows  | `%LOCALAPPDATA%\mememory` |

Inside that directory you get `postgres/` (database files) and `ollama/` (embedding model). Override with `DATA_DIR=/custom/path` if you want.

## Backup

Two equivalent options:

```bash
# Stop the stack and copy the data dir
mememory uninstall
cp -a "$DATA_DIR" "$DATA_DIR.backup-$(date +%F)"
mememory setup

# Or take a logical dump against any Postgres
pg_dump "$DATABASE_URL" > mememory-$(date +%F).sql
```

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

**Scopes** — global (everywhere) and project (one project). Recall searches hierarchically: project scope sees global + its own project.

**Types** — fact, rule, decision, feedback, context, bootstrap. Only `bootstrap` memories load automatically at session start; all other types are loaded on demand via `recall`.

**Scoring** — `similarity × scope_weight × memory_weight × temporal_decay`. Recent, specific, high-weight memories rank higher. Project weight = 1.0, global weight = 0.8.

**Contradiction detection** — warns when a new memory is >75% similar to existing ones. Does not block storage.

**Session bootstrap** — `bootstrap`-type memories are delivered to the agent at session start via the `SessionStart` hook or the `mememory://bootstrap` MCP resource. Output is capped at 10KB to avoid MCP client truncation.

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
