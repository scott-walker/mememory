# Configuration

mememory is configured through environment variables. There are three configuration contexts: Docker stack, server processes, and the CLI.

## Docker Stack Variables

These variables are used in `docker/docker-compose.yml` and control the Docker infrastructure:

| Variable | Default | Description |
|----------|---------|-------------|
| `DATA_DIR` | OS-standard path (auto-resolved by CLI) | Host directory for persistent data (PostgreSQL data, Ollama models). Bind-mounted into containers. |
| `POSTGRES_PORT` | `5432` | Host port for PostgreSQL |
| `POSTGRES_PASSWORD` | `mememory` | PostgreSQL password for the `mememory` user |
| `OLLAMA_PORT` | `11434` | Host port for the Ollama API |
| `ADMIN_PORT` | `4200` | Host port for the Admin API and web UI |

### Example: Custom ports

```bash
# .env file in the project root
DATA_DIR=/data/mememory
POSTGRES_PORT=15432
POSTGRES_PASSWORD=s3cret
OLLAMA_PORT=21434
ADMIN_PORT=9200
```

### Data directory structure

```
$DATA_DIR/
├── postgres/       # PostgreSQL data files
└── ollama/         # Ollama model files (nomic-embed-text, etc.)
```

If `DATA_DIR` is unset, the `mememory` CLI auto-resolves it:

| Platform | Default |
|----------|---------|
| Linux    | `~/.local/share/mememory` (or `$XDG_DATA_HOME/mememory`) |
| macOS    | `~/Library/Application Support/mememory` |
| Windows  | `%LOCALAPPDATA%\mememory` |

::: warning
Changing `DATA_DIR` after initial setup means the new directory starts empty. Move the existing data or use [backup/restore](/guide/backup) to migrate.
:::

## Server Variables

These variables configure the `server` (MCP server) and `admin` (Admin API) processes inside the container:

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | _required, no default_ | PostgreSQL connection string. The server fails fast on missing value. Example: `postgres://mememory:mememory@localhost:5432/mememory?sslmode=disable`. Requires PostgreSQL >= 14 with the `pgvector` extension. |
| `OLLAMA_URL` | `http://localhost:11434` | Ollama API URL (used when `EMBEDDING_PROVIDER` is `ollama` or unset) |
| `ADMIN_PORT` | `4200` | Port for the Admin API server |
| `EMBEDDING_PROVIDER` | `ollama` | Embedding provider: `ollama` or `openai` |
| `EMBEDDING_URL` | (provider-specific) | Custom embedding API URL |
| `EMBEDDING_API_KEY` | — | API key for cloud embedding providers |
| `EMBEDDING_MODEL` | (provider-specific) | Override the default embedding model |

### DATABASE_URL

Standard PostgreSQL connection string. The database must have the `vector` extension enabled (included in the `pgvector/pgvector:pg17` Docker image).

```bash
# Bundled Docker stack
DATABASE_URL=postgres://mememory:mememory@localhost:5432/mememory?sslmode=disable

# Remote server (BYO Postgres)
DATABASE_URL=postgres://user:pass@db.example.com:5432/mememory?sslmode=require
```

The connecting user needs `CREATE` privilege so the server can run `CREATE EXTENSION IF NOT EXISTS vector` on first start. Otherwise have your DBA install pgvector beforehand.

### Embedding Provider Configuration

See [Embedding Providers](/guide/embedding-providers) for detailed configuration of each provider.

**Ollama (default, local):**

```bash
EMBEDDING_PROVIDER=ollama
OLLAMA_URL=http://localhost:11434
# EMBEDDING_MODEL defaults to nomic-embed-text
```

**OpenAI:**

```bash
EMBEDDING_PROVIDER=openai
EMBEDDING_API_KEY=sk-...
# EMBEDDING_MODEL defaults to text-embedding-3-small
```

**OpenAI-compatible API (e.g., Azure, Mistral):**

```bash
EMBEDDING_PROVIDER=openai
EMBEDDING_URL=https://my-endpoint.openai.azure.com/openai/deployments/my-embedding/embeddings?api-version=2024-02-01
EMBEDDING_API_KEY=your-api-key
EMBEDDING_MODEL=text-embedding-3-small
```

## CLI Variables

The `mememory` CLI binary uses a single variable:

| Variable | Default | Description |
|----------|---------|-------------|
| `MEMORY_URL` | `http://localhost:4200` | Admin API URL for bootstrap and status commands |

### Example

```bash
# Default — connects to local Docker stack
mememory bootstrap

# Custom URL
MEMORY_URL=http://my-server:4200 mememory bootstrap

# Or use the --url flag
mememory bootstrap --url http://my-server:4200
```

## Complete Example

A full `.env` file for a custom setup:

```bash
# Data storage
DATA_DIR=/opt/mememory/data

# Ports
POSTGRES_PORT=5432
OLLAMA_PORT=11434
ADMIN_PORT=4200

# Security
POSTGRES_PASSWORD=strong-password-here

# Embedding provider (cloud)
EMBEDDING_PROVIDER=openai
EMBEDDING_API_KEY=sk-your-key-here
EMBEDDING_MODEL=text-embedding-3-small
```

## Docker Compose Override

For advanced customization, create a `docker-compose.override.yml`:

```yaml
services:
  postgres:
    ports:
      - "15432:5432"
    environment:
      POSTGRES_PASSWORD: my-secure-password

  admin:
    environment:
      EMBEDDING_PROVIDER: openai
      EMBEDDING_API_KEY: sk-your-key
```

This merges with the base `docker-compose.yml` when running `docker compose up`.

## Precedence

1. CLI flags (`--url`, `--project`, etc.) override everything
2. Environment variables override defaults
3. Default values are used when nothing else is set

Inside Docker, the `docker-compose.yml` sets internal service URLs (e.g., `DATABASE_URL=postgres://mememory:mememory@postgres:5432/mememory`) for container-to-container communication.
