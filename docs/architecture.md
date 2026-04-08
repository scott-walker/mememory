# Architecture

## Overview

MEMEMORY is an MCP (Model Context Protocol) server that gives AI agents persistent semantic memory. Agents connect via stdio, store and retrieve memories through vector similarity search, and receive accumulated knowledge at the start of each session.

```
Agent (Claude Code) ──stdio──> mememory-server (Go)
                                    │
                            ┌───────┴───────┐
                            ▼               ▼
                       PostgreSQL       Ollama
                    (pgvector, data)  (embeddings)
```

The server runs as a Docker stack: PostgreSQL with pgvector for storage and search, Ollama for local embedding generation, and an admin container that hosts both the MCP server binary and a web UI.

## Design Principles

**Occam's Razor.** Start with the simplest solution that works. Don't add abstractions, services, or mechanisms until the simple approach proves insufficient. 20% effort, 80% result.

**Privacy first.** All data stays on the user's machine. Embeddings are computed locally via Ollama. No data leaves the network.

**One command to run.** `docker compose up` starts the entire stack — database, embedding model, admin UI. No manual setup steps, no external dependencies.

**Portable data.** All persistent state lives in a single directory (auto-resolved by the CLI per OS, override with `DATA_DIR`) as bind mounts. Backup is copying a folder. Migration is moving a folder. No Docker named volumes are used for user data.

## Stack

| Component | Image | Purpose |
|-----------|-------|---------|
| PostgreSQL | `pgvector/pgvector:pg17` | Vector storage, metadata, SQL queries |
| Ollama | Custom (ollama/ollama + entrypoint) | Local embeddings (nomic-embed-text, 768d) |
| Admin | Custom (Go + React) | MCP server binary + web admin UI |

Why PostgreSQL instead of a dedicated vector DB: one database handles both vectors and structured data (sessions, users, projects in the future). pgvector provides HNSW indexing — performance is identical for datasets under millions of records. Simpler infrastructure, single backup mechanism, standard SQL for complex queries.

## Data Flow

### Storing a memory (remember)

```
1. Agent calls remember(content, scope, type, ...)
2. Server sends content to Ollama → gets 768d embedding vector
3. Server checks for contradictions (cosine similarity > 0.75 against existing memories)
4. Server inserts into PostgreSQL: content + embedding + metadata
5. If supersedes is set, the old memory's weight drops to 0.1
6. Returns memory object + contradiction warnings (if any)
```

### Retrieving memories (recall)

```
1. Agent calls recall(query, project?)
2. Server sends query to Ollama → gets 768d embedding vector
3. PostgreSQL performs cosine similarity search with hierarchical scope filter
4. Server post-processes: filters expired/superseded, applies scoring formula
5. Returns top N results sorted by final score
```

### Session bootstrap (SessionStart hook)

```
1. Claude Code starts a new session
2. SessionStart hook runs: `docker exec mememory-admin mememory-server --bootstrap --project myapp`
3. mememory-server connects to PostgreSQL, reads global + project-scoped memories with `type=bootstrap`
4. Formats them as Markdown (with a hard-coded System section followed by Bootstrap > Rules > Feedback > Facts > Decisions > Context)
5. Prints to stdout — hook injects output into the agent's context
6. Agent receives essential directives from the first message; everything else is loaded on demand via `recall`
```

The `--bootstrap` flag is a CLI mode of the same binary. It connects directly to PostgreSQL (no Ollama needed), reads memories with `type=bootstrap`, prints Markdown, and exits. Output is capped at 10KB (`MaxBootstrapBytes`); above that, a warning is printed to stderr because MCP clients may truncate hook output.

Flags:
- `--project <name>` — include project-scoped bootstrap memories (hierarchical: global + project)

## Directory Structure

```
cmd/
  mememory-server/    MCP server entry point (stdio transport)
  mememory-admin/     Admin API + web UI entry point (HTTP :4200)
internal/
  mcp/              MCP tool and resource definitions
  engine/           Business logic, scoring, type re-exports
  postgres/         PostgreSQL client, migrations, filters
  embeddings/       Ollama HTTP client
  api/              REST API for admin UI
  types/            Shared data types (Memory, Scope, MemoryType, etc.)
docker/
  docker-compose.yml
  Dockerfile          Multi-stage build (Go + React → Alpine)
  ollama.Dockerfile   Ollama with auto model pull
  ollama-entrypoint.sh
web/                React + TypeScript admin UI
```
