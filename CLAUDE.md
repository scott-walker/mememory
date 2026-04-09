# mememory

Hierarchical MCP memory server for AI agents. Provides persistent semantic memory across sessions with two scope levels: global and project.

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
make build        # Build Go binary → bin/server
make run          # Build and run
make dev          # Run with go run (no build)
make clean        # Remove binary + Docker volumes

make admin        # Run admin UI (dev mode)
make admin-build  # Build admin binary with embedded web UI
```

## Architecture

```
cmd/mememory-server/main.go           # MCP server entry point → `server` binary in container
cmd/mememory-admin/main.go            # Admin API + web UI server → `admin` binary in container
cmd/mememory/main.go                # User-facing CLI (setup, bootstrap, status)
cmd/mememory/bootstrap.go           # CLI bootstrap: fetches type=bootstrap memories via admin API
internal/bootstrap/format.go        # Shared bootstrap formatter (CLI + MCP resources)
internal/mcp/tools.go               # 7 MCP tools (remember, recall, forget, update, list, stats, help)
internal/mcp/resources.go           # MCP resources (mememory://bootstrap, mememory://bootstrap/{project})
internal/engine/service.go          # Business logic, scoring, contradiction detection
internal/engine/types.go            # Type re-exports from internal/types
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
| `remember` | content, scope, project?, type, tags?, weight?, ttl?, supersedes? | Store a memory (warns if a `bootstrap`-typed entry pushes total bootstrap output past `MaxBootstrapTokens`) |
| `recall` | query, scope?, project?, limit? | Semantic search with hierarchical inheritance |
| `forget` | id | Delete by ID |
| `update` | id, content | Re-embed and update |
| `list` | scope?, project?, type?, limit? | List with filters |
| `stats` | — | Count breakdown by scope/project/type |
| `help` | topic? | Usage documentation |

Memory types: `fact`, `rule`, `decision`, `feedback`, `context`, `bootstrap`. Only `bootstrap`-typed memories are loaded automatically at session start; everything else is fetched on demand via `recall`.

## Bootstrap (CLI mode)

```bash
mememory bootstrap                            # auto-detect project (.mememory file → git → cwd)
mememory bootstrap --project myapp            # explicit project override
mememory bootstrap --url http://host:4200     # custom admin API URL
```

Designed for SessionStart hooks. Talks to the admin HTTP API (no direct DB access), filters by `type=bootstrap`, formats as Markdown, prints to stdout.

**Project resolution priority:** `--project` flag → `.mememory` file (walk-up from cwd) → `git rev-parse --show-toplevel` basename → `basename(cwd)`. The chosen source is reported in the trailing `## Bootstrap Stats` block. See `docs/config/mememory-file.md` for the `.mememory` file specification.

**Token budget:** the bootstrap payload is bounded by `bootstrap.MaxBootstrapTokens` (30K tokens, ≈15% of a 200K-token context window). Token counts are estimated from byte length using `bootstrap.BytesPerToken` (3.5 — tuned for mixed Cyrillic prose and code; per-tokenizer accuracy is out of scope). Overflow appends a `WARNING` line to the output but does not truncate. The Markdown always begins with a hard-coded `## System` section instructing the agent to treat mememory as the only memory source and to call `recall` on the user's first message, and ends with a `## Bootstrap Stats` block reporting project, source, memory counts, and budget usage.

## Scopes & Hierarchy

- **global** — visible to all projects
- **project** — visible within a specific project

Hierarchical search: `recall(project=Y)` searches global + project:Y. Scope weights in scoring: `project=1.0`, `global=0.8`.

## Environment Variables

| Var | Default | Description |
|-----|---------|-------------|
| `DATABASE_URL` | _required, no default_ | PostgreSQL connection (e.g. `postgres://mememory:mememory@localhost:5432/mememory?sslmode=disable`). Server fails fast if unset. |
| `DATA_DIR` | OS-standard path | Persistent data directory. CLI auto-resolves: `~/.local/share/mememory` (Linux), `~/Library/Application Support/mememory` (macOS), `%LOCALAPPDATA%\mememory` (Windows). |
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
