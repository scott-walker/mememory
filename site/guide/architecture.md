# Architecture

mememory is a Go monorepo with three binaries, a React admin UI, and a Docker-based infrastructure layer. This page covers the system design, data flow, and key implementation decisions.

## System Overview

```
                     Host Machine                          Docker Stack
                ┌─────────────────────┐          ┌──────────────────────────┐
                │                     │          │                          │
User ────────── │   mememory CLI      │──HTTP──> │   Admin API (:4200)      │
                │   (bootstrap,       │          │   mememory                 │
                │    status)          │          │      │                   │
                │                     │          │      ├── REST endpoints  │
                └─────────────────────┘          │      └── Web UI (React) │
                                                 │            │             │
Agent ──stdio── │   server              │──────────│────────────┤             │
                │   (inside Docker)   │          │            │             │
                │                     │          │            ▼             │
                └─────────────────────┘          │   ┌──────────────┐      │
                                                 │   │  PostgreSQL  │      │
                                                 │   │  + pgvector  │      │
                                                 │   │  (:5432)     │      │
                                                 │   └──────────────┘      │
                                                 │                          │
                                                 │   ┌──────────────┐      │
                                                 │   │   Ollama     │      │
                                                 │   │  (:11434)    │      │
                                                 │   └──────────────┘      │
                                                 └──────────────────────────┘
```

## Components

### server (MCP server)

The MCP server binary (`server` inside the container, built from `cmd/mememory-server/`). Communicates with agents via stdio (stdin/stdout). Registers 7 MCP tools and 2 MCP resources. Runs inside the `mememory` Docker container.

- Entry point: `cmd/mememory-server/main.go`
- Runs a background TTL cleanup goroutine (hourly)
- On startup: requires `DATABASE_URL` (fails fast if missing), connects to PostgreSQL, runs `CREATE EXTENSION IF NOT EXISTS vector`, applies migrations, probes embedding dimension, validates database column

### admin (Admin API)

The Admin API and web UI server (`admin` inside the container, built from `cmd/mememory-admin/`). Serves REST endpoints on port 4200 and the React admin UI as static files.

- Entry point: `cmd/mememory-admin/main.go`
- REST API: `internal/api/router.go`, `internal/api/handler.go`
- Static file serving: embedded React build from `web/dist/`

### mememory CLI

Native Go binary for host-side operations. Does not connect to PostgreSQL or Ollama directly — communicates with the Admin API over HTTP.

- Entry point: `cmd/mememory/main.go`
- Commands: `bootstrap`, `status`, `version`
- Auto-detects project from git repository root

### PostgreSQL + pgvector

Stores all memories with their vector embeddings. Uses the HNSW index for approximate nearest neighbor search with cosine distance.

- Docker image: `pgvector/pgvector:pg17`
- Data persisted to `$DATA_DIR/postgres/` (CLI auto-resolves `DATA_DIR` to an OS-standard path)
- Schema managed via embedded SQL migrations
- Cosine distance operator: `<=>` (lower = more similar)

### Ollama

Runs local embedding models. The default model is `nomic-embed-text` (768 dimensions). No data leaves the machine.

- Docker image: custom build that pulls the model on start
- Data persisted to `$DATA_DIR/ollama/`
- HTTP API at port 11434

### React Admin UI

Single-page application for browsing and managing memories. Built with React + TypeScript.

- Source: `web/`
- Communicates with Admin API endpoints
- Served as static files by the `admin` binary

## Directory Structure

```
mememory/
├── cmd/
│   ├── mememory-server/       # MCP server → `server` binary in container
│   │   └── main.go
│   ├── mememory-admin/        # Admin API → `admin` binary in container
│   │   └── main.go
│   └── mememory/            # Native CLI (bootstrap, status, version)
│       ├── main.go
│       ├── bootstrap.go
│       └── status.go
├── internal/
│   ├── api/                 # REST API handlers (chi router)
│   │   ├── router.go
│   │   └── handler.go
│   ├── bootstrap/           # Shared bootstrap formatter (Markdown output)
│   │   └── format.go
│   ├── embeddings/          # Embedding provider abstraction
│   │   ├── embedder.go      # Embedder interface
│   │   ├── factory.go       # Provider factory (Config → Embedder)
│   │   ├── ollama.go        # Ollama HTTP client
│   │   ├── openai.go        # OpenAI-compatible HTTP client
│   │   └── probe.go         # Dimension auto-detection
│   ├── mcp/                 # MCP server registration
│   │   ├── tools.go         # 7 MCP tools + help text
│   │   └── resources.go     # MCP resources (bootstrap)
│   ├── engine/              # Business logic layer
│   │   ├── service.go       # Scoring, contradiction detection, CRUD
│   │   └── types.go         # Type re-exports
│   ├── postgres/            # PostgreSQL client
│   │   ├── client.go        # Queries, migrations, hierarchical WHERE
│   │   └── migrations/      # Embedded SQL migrations
│   └── types/               # Shared types (Memory, Scope, MemoryType, DTOs)
│       └── types.go
├── docker/
│   ├── docker-compose.yml   # Stack definition (postgres, ollama, admin)
│   ├── Dockerfile           # Multi-stage build (Go + React → Alpine)
│   └── ollama.Dockerfile    # Ollama with auto model pull
├── web/                     # React admin UI source
├── site/                    # VitePress documentation
├── scripts/
│   └── setup.sh             # First-time infrastructure setup
├── Makefile                 # Build and dev commands
└── go.mod
```

## Data Flow

### remember (store a memory)

```
Agent calls remember(content="...", scope="project", project="match", type="rule")
    ↓
MCP tool handler validates input, sets defaults
    ↓
Embedder.EmbedOne(content) → 768-dim float32 vector
    ↓
Contradiction check: SearchWithWhere(vector, scope hierarchy, limit=5)
    → Any existing memory with similarity > 0.75 → warning
    ↓
PostgreSQL INSERT (id, content, embedding, metadata)
    ↓
If supersedes is set → UPDATE old memory weight to 0.1
    ↓
Return Memory object + contradiction warnings (if any)
```

### recall (search memories)

```
Agent calls recall(query="database architecture", project="match")
    ↓
Embedder.EmbedOne(query) → query vector
    ↓
HierarchicalWhere("", "match") → WHERE (scope='global' OR (scope='project' AND project='match'))
    ↓
PostgreSQL: SELECT *, 1-(embedding <=> query_vector) AS score ... ORDER BY distance LIMIT 15
    ↓
Filter expired (TTL check)
    ↓
Filter superseded (collect supersedes IDs, remove targets)
    ↓
Score each: similarity × scope_weight × memory_weight × temporal_decay
    ↓
Sort by final score, trim to requested limit
    ↓
Return [{memory, score}, ...]
```

### bootstrap (session initialization)

```
mememory bootstrap --project match
    ↓
Auto-detect project from git root (if --project not set)
    ↓
HTTP GET /api/memories?scope=global&type=bootstrap&limit=100 → global bootstrap memories
HTTP GET /api/memories?scope=project&project=match&type=bootstrap&limit=100 → project bootstrap memories
    ↓
Merge results
    ↓
Format as Markdown with a hard-coded System section, then Bootstrap/Rules/Feedback/Facts/Decisions/Context groups
    ↓
Append a `## Bootstrap Stats` block with project source, memory counts, token estimate, budget percent
    ↓
If estimated tokens exceed MaxBootstrapTokens (30_000) → append `WARNING: bootstrap exceeds budget` to Stats block (no truncation)
    ↓
Print to stdout → captured by SessionStart hook → injected into agent context
```

## Why PostgreSQL Over a Dedicated Vector DB

mememory uses PostgreSQL with pgvector instead of a dedicated vector database (Qdrant, Pinecone, Weaviate). Reasons:

1. **Single dependency.** One database for both metadata and vectors. No need to synchronize between a relational DB and a vector DB.

2. **pgvector is sufficient.** For the scale of personal agent memory (hundreds to low thousands of memories), pgvector's HNSW index provides sub-millisecond search. Dedicated vector DBs optimize for millions+ vectors.

3. **SQL for everything.** Hierarchical scope filters, TTL cleanup, metadata queries, export/import — all standard SQL. No need to learn a vector DB query language.

4. **Mature ecosystem.** PostgreSQL has battle-tested backup, replication, monitoring, and tooling.

5. **Operational simplicity.** One container instead of two. Less memory, less disk, fewer failure modes.

The tradeoff: if the memory count grows to millions, pgvector may become a bottleneck. At that point, migrating to a dedicated vector DB is straightforward since the `Embedder` interface is already abstracted.
