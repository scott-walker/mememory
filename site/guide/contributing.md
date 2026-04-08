# Contributing

This guide covers local development setup, project structure, and how to contribute to mememory.

## Prerequisites

- **Go 1.22+**
- **Docker** and **Docker Compose**
- **Node.js 18+** and **pnpm** (for the admin web UI)
- **Make**

## Development Setup

### 1. Clone the repository

```bash
git clone https://github.com/scott-walker/mememory.git
cd mememory
```

### 2. Start infrastructure

```bash
make infra-up
```

This starts PostgreSQL (pgvector) and Ollama in Docker. On first run, Ollama downloads the `nomic-embed-text` model.

### 3. Run the MCP server (dev mode)

```bash
make dev
```

This runs `go run ./cmd/mememory-server` with the default environment variables pointing to the local Docker services.

### 4. Run the Admin API + Web UI

In a separate terminal:

```bash
make admin-dev
```

This starts the Go admin server and the React dev server (with hot reload). The admin UI is available at [http://localhost:4200](http://localhost:4200).

### 5. Build binaries

```bash
make build        # Build MCP server → bin/server
make cli          # Build mememory CLI → bin/mememory
make admin-build  # Build admin (Go + React) → bin/admin
```

## Project Structure

```
cmd/
├── mememory-server/    # MCP server → `server` binary in container
├── mememory-admin/     # Admin API → `admin` binary in container
└── mememory/         # Native CLI (bootstrap, status)

internal/
├── api/              # REST API handlers (chi router)
├── bootstrap/        # Markdown formatter for session bootstrap
├── embeddings/       # Embedding provider abstraction + implementations
├── mcp/              # MCP tool and resource registration
├── memory/           # Business logic (scoring, CRUD, contradictions)
├── postgres/         # PostgreSQL client, migrations, queries
└── types/            # Shared types and DTOs

web/                  # React admin UI
docker/               # Docker Compose, Dockerfiles
scripts/              # Setup and utility scripts
site/                 # VitePress documentation
```

## Key Interfaces

### Embedder

The embedding provider interface in `internal/embeddings/embedder.go`:

```go
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    EmbedOne(ctx context.Context, text string) ([]float32, error)
}
```

All embedding providers implement this interface. `Embed` handles batch embedding, `EmbedOne` is a convenience wrapper.

### Memory Service

The business logic layer in `internal/engine/service.go`:

```go
type Service struct { ... }

func (s *Service) Remember(ctx, input) (*RememberResult, error)
func (s *Service) Recall(ctx, input) ([]RecallResult, error)
func (s *Service) Forget(ctx, id) error
func (s *Service) Update(ctx, id, content) (*Memory, error)
func (s *Service) List(ctx, input) ([]Memory, error)
func (s *Service) Stats(ctx) (*StatsResult, error)
func (s *Service) CleanExpired(ctx) (int, error)
```

## Adding an Embedding Provider

To add a new embedding provider (e.g., Voyage AI):

### 1. Create the client

Create `internal/embeddings/voyage.go`:

```go
package embeddings

import (
    "context"
    // ...
)

type VoyageClient struct {
    url    string
    apiKey string
    model  string
    // ...
}

func NewVoyageClient(apiKey string) *VoyageClient {
    return &VoyageClient{
        url:    "https://api.voyageai.com/v1/embeddings",
        apiKey: apiKey,
        model:  "voyage-3",
    }
}

func (c *VoyageClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
    // Implement the Voyage AI embedding API call
}

func (c *VoyageClient) EmbedOne(ctx context.Context, text string) ([]float32, error) {
    vectors, err := c.Embed(ctx, []string{text})
    if err != nil {
        return nil, err
    }
    return vectors[0], nil
}
```

### 2. Register in the factory

Update `internal/embeddings/factory.go`:

```go
func New(cfg Config) (Embedder, error) {
    switch cfg.Provider {
    case "", "ollama":
        // ...existing code...
    case "openai":
        // ...existing code...
    case "voyage":
        if cfg.APIKey == "" {
            return nil, fmt.Errorf("EMBEDDING_API_KEY is required for voyage provider")
        }
        return NewVoyageClient(cfg.APIKey), nil
    default:
        return nil, fmt.Errorf("unknown embedding provider: %q (supported: ollama, openai, voyage)", cfg.Provider)
    }
}
```

### 3. Update documentation

- Add the provider to `site/guide/embedding-providers.md`
- Update the provider table and add a configuration example

## Running Tests

```bash
go test ./...
```

## Code Style

- Go standard formatting (`gofmt`)
- Error wrapping with context: `fmt.Errorf("operation: %w", err)`
- No global state — dependency injection through constructors
- Minimal interfaces — only abstract what has multiple implementations

## PR Guidelines

1. One feature or fix per PR
2. Include a clear description of what changed and why
3. Update documentation if the change affects user-facing behavior
4. Ensure `go test ./...` passes
5. Ensure `go vet ./...` passes

## Useful Commands

```bash
make infra-up       # Start Docker services
make infra-down     # Stop Docker services
make dev            # Run MCP server (dev mode)
make admin          # Run admin API (dev mode)
make admin-dev      # Run admin API + React dev server
make build          # Build MCP server binary
make cli            # Build mememory CLI binary
make admin-build    # Build admin binary with embedded web UI
make clean          # Remove binaries + Docker volumes
```
