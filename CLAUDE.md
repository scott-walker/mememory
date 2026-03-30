# claude-memory

Hierarchical MCP memory server for Claude Code. Provides persistent semantic memory across sessions with three scope levels: global, project, persona.

## Stack

- **Go** — MCP server binary (stdio transport)
- **Qdrant** — vector database (Docker, gRPC :6334, REST :6333)
- **Ollama** — local embeddings (Docker, nomic-embed-text 768d, HTTP :11434)

## Commands

```bash
make infra-up     # Start Qdrant + Ollama (Docker)
make infra-down   # Stop Docker services
make setup        # Create collection + pull embedding model (run once)
make build        # Build Go binary → bin/memory-server
make run          # Build and run
make dev          # Run with go run (no build)
make clean        # Remove binary + Docker volumes
```

## Architecture

```
cmd/memory-server/main.go        # Entry point, DI wiring, stdio transport
internal/mcp/tools.go            # 6 MCP tool definitions (remember, recall, forget, update, list, stats)
internal/memory/service.go       # Business logic, hierarchical filter construction
internal/memory/types.go         # Memory, Scope, MemoryType, Filter types
internal/qdrant/client.go        # Qdrant gRPC wrapper
internal/embeddings/ollama.go    # Ollama HTTP embedding client
docker/docker-compose.yml        # Qdrant + Ollama services
docker/Dockerfile                # Multi-stage Go build
scripts/setup.sh                 # Init collection + pull model
```

## MCP Tools

| Tool | Params | Description |
|------|--------|-------------|
| `remember` | content, scope, project?, persona?, type, tags?, ttl? | Store a memory |
| `recall` | query, scope?, project?, persona?, limit? | Semantic search with hierarchical inheritance |
| `forget` | id | Delete by ID |
| `update` | id, content | Re-embed and update |
| `list` | scope?, project?, persona?, type?, limit? | List with filters |
| `stats` | — | Count breakdown |

## Scopes & Hierarchy

- **global** — visible to all projects and personas
- **project** — visible within a specific project
- **persona** — visible to a specific agent persona within a project

Hierarchical search: `recall(persona=X, project=Y)` searches global + project:Y + persona:X.

## Environment Variables

| Var | Default | Description |
|-----|---------|-------------|
| `QDRANT_HOST` | localhost | Qdrant gRPC host |
| `QDRANT_PORT` | 6334 | Qdrant gRPC port |
| `OLLAMA_URL` | http://localhost:11434 | Ollama API URL |
