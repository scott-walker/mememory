# mememory

Hierarchical MCP memory server for AI agents. Provides persistent semantic memory across sessions with three scope levels: global, project, persona.

## Stack

- **Go** — MCP server binary (stdio transport) + admin API
- **PostgreSQL + pgvector** — vector storage and metadata (Docker)
- **Ollama** — local embeddings (Docker, nomic-embed-text 768d)
- **React + TypeScript** — admin web UI

## Commands

```bash
make infra-up     # Start PostgreSQL + Ollama (Docker)
make infra-down   # Stop Docker services
make setup        # Init DB + pull embedding model (run once)
make build        # Build Go binary → bin/memory-server
make run          # Build and run
make dev          # Run with go run (no build)
make clean        # Remove binary + Docker volumes

make admin        # Run admin UI (dev mode)
make admin-build  # Build admin binary with embedded web UI
```

## Architecture

```
cmd/memory-server/main.go           # Entry point, CLI bootstrap mode, stdio MCP transport
cmd/memory-admin/main.go            # Admin API + web UI server (:4200)
internal/bootstrap/format.go        # Shared bootstrap formatter (CLI + MCP resources)
internal/mcp/tools.go               # 7 MCP tools (remember, recall, forget, update, list, stats, help)
internal/mcp/resources.go           # MCP resources (memory://bootstrap, memory://bootstrap/{project})
internal/memory/service.go          # Business logic, scoring, contradiction detection
internal/memory/types.go            # Type re-exports from internal/types
internal/types/types.go             # Memory, Scope, MemoryType, input/output DTOs
internal/postgres/client.go         # PostgreSQL client, migrations, filters, hierarchical WHERE
internal/embeddings/ollama.go       # Ollama HTTP embedding client
internal/api/                       # REST API handlers for admin web UI
docker/docker-compose.yml           # PostgreSQL + Ollama + Admin services
docker/Dockerfile                   # Multi-stage build (Go + React → Alpine)
web/                                # React admin UI source
```

## MCP Tools

| Tool | Params | Description |
|------|--------|-------------|
| `remember` | content, scope, project?, persona?, type, tags?, weight?, ttl?, supersedes? | Store a memory |
| `recall` | query, scope?, project?, persona?, limit? | Semantic search with hierarchical inheritance |
| `forget` | id | Delete by ID |
| `update` | id, content | Re-embed and update |
| `list` | scope?, project?, persona?, type?, limit? | List with filters |
| `stats` | — | Count breakdown by scope/project/type |
| `help` | topic? | Usage documentation |

## Bootstrap (CLI mode)

```bash
memory-server --bootstrap                                  # global only
memory-server --bootstrap --project myapp                  # global + project
memory-server --bootstrap --project myapp --persona dev    # global + project + persona
```

Designed for SessionStart hooks. Connects to PostgreSQL, reads memories, prints Markdown, exits. No Ollama needed.

## Scopes & Hierarchy

- **global** — visible to all projects and personas
- **project** — visible within a specific project
- **persona** — visible to a specific agent persona within a project

Hierarchical search: `recall(persona=X, project=Y)` searches global + project:Y + persona:X.

## Environment Variables

| Var | Default | Description |
|-----|---------|-------------|
| `DATABASE_URL` | postgres://memory:memory@localhost:5432/memory?sslmode=disable | PostgreSQL connection |
| `OLLAMA_URL` | http://localhost:11434 | Ollama API URL |
| `ADMIN_PORT` | 4200 | Admin UI port |

## Design Principles

- **Cross-platform.** All user-facing functionality must work on Linux, macOS, and Windows. No shell scripts, no platform-specific glue. If logic can't be expressed through Go code or Docker commands, it doesn't belong in the product.
- **Zero friction.** `mememory setup` — one command from install to working memory. Auto-detect MCP client, auto-configure hooks, auto-detect project from working directory.
- **Docker for infra, native binary for UX.** Heavy services (DB, embeddings) in Docker. User-facing CLI runs natively for speed and OS integration.
- **Occam's Razor.** Start with the simplest solution that works. Don't add abstractions until the simple approach proves insufficient.
- **Privacy first.** All data stays on the user's machine. Embeddings computed locally. No data leaves the network.

## Product Vision

See `docs/vision-memory-cli.md` — `mememory` as the single entry point: setup, bootstrap, status, export/import, upgrade.
