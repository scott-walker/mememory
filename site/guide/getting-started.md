# Getting Started

This guide walks you through installing mememory, starting the infrastructure, and connecting your first MCP client.

## Prerequisites

- **Docker** and **Docker Compose** (for PostgreSQL + Ollama)
- A supported MCP client (Claude Code, Claude Desktop, or any MCP-compatible agent)

## Installation

### Docker Only (recommended)

No local installation needed. The Docker stack includes everything: PostgreSQL with pgvector, Ollama for embeddings, the admin API, and the MCP server.

```bash
git clone https://github.com/scott-walker/mememory.git
cd mememory
cp .env.example .env
docker compose -f docker/docker-compose.yml -p mememory up -d
```

The MCP server runs inside the `mememory-admin` container. Your MCP client connects to it via `docker exec`.

### Go Install (native binary)

If you want the `mememory` CLI on your host for bootstrap and status commands:

```bash
go install github.com/scott-walker/mememory/cmd/mememory@latest
```

This installs the `mememory` binary to your `$GOPATH/bin`. You still need Docker for PostgreSQL and Ollama.

### Homebrew (macOS/Linux)

```bash
brew install scott-walker/tap/mememory
```

### Curl Script

```bash
curl -fsSL https://raw.githubusercontent.com/scott-walker/mememory/main/scripts/install.sh | bash
```

## First-Time Setup

### 1. Start infrastructure

```bash
docker compose -f docker/docker-compose.yml -p mememory up -d
```

This starts three containers:

| Container | Port | Purpose |
|-----------|------|---------|
| `mememory-postgres` | 5432 | PostgreSQL with pgvector extension |
| `mememory-ollama` | 11434 | Ollama embedding server (nomic-embed-text) |
| `mememory-admin` | 4200 | Admin API + web UI + MCP server |

::: tip
On first start, Ollama downloads the `nomic-embed-text` model (~275 MB). This happens automatically inside the Docker container.
:::

### 2. Verify services are running

```bash
# Check all containers are healthy
docker compose -f docker/docker-compose.yml -p mememory ps

# Or use the CLI
mememory status
```

Expected output:

```
Checking http://localhost:4200 ...
OK: 0 memories stored
```

### 3. Open the Admin UI

Navigate to [http://localhost:4200](http://localhost:4200) in your browser. The web UI lets you browse, search, and manage memories.

## Connect an MCP Client

### Claude Code

Add the MCP server to your config file (`~/.claude/.claude.json`):

```json
{
  "mcpServers": {
    "memory": {
      "type": "stdio",
      "command": "docker",
      "args": ["exec", "-i", "mememory-admin", "memory-server"],
      "env": {}
    }
  }
}
```

For automatic session bootstrap (loads rules and context at session start), add a hook to your [Claude Code settings](/guide/mcp-client-setup):

```json
{
  "hooks": {
    "SessionStart": [
      {
        "type": "command",
        "command": "mememory bootstrap"
      }
    ]
  }
}
```

### Claude Desktop

See the full [MCP Client Setup](/guide/mcp-client-setup) guide for Claude Desktop and other clients.

## Verify It Works

Once connected, ask your agent to use the memory tools:

```
Store a test memory: "This is a test memory from getting started"
```

The agent should call the `remember` tool and confirm the memory was stored. Then verify:

```
Recall memories about "test memory"
```

You should see the memory you just stored, with a relevance score.

Check the Admin UI at [http://localhost:4200](http://localhost:4200) to see the memory in the database.

## What's Next

- [Memory Model](/guide/memory-model) — understand memory types, fields, and when to use each
- [Scopes & Hierarchy](/guide/scopes) — learn about global, project, and persona scopes
- [Session Bootstrap](/guide/bootstrap) — configure automatic memory loading at session start
- [MCP Client Setup](/guide/mcp-client-setup) — detailed setup for all supported clients
