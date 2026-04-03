# Backup & Migration

All mememory data lives in a single directory on your machine. This page covers backup, restore, export/import, and migration between machines.

## Data Location

By default, all persistent data is stored in `~/.mememory/`:

```
~/.mememory/
├── postgres/       # PostgreSQL data files (memories, vectors, indexes)
└── ollama/         # Ollama model files (nomic-embed-text, etc.)
```

This location is controlled by the `MEMORY_DATA_DIR` environment variable in `docker-compose.yml`.

## Backup

### Full backup (recommended)

Stop the stack and copy the data directory:

```bash
# Stop services to ensure data consistency
docker compose -f docker/docker-compose.yml -p mememory down

# Copy the data directory
cp -r ~/.mememory ~/.mememory-backup-$(date +%Y%m%d)

# Restart
docker compose -f docker/docker-compose.yml -p mememory up -d
```

::: warning
Always stop PostgreSQL before copying its data directory. Copying while PostgreSQL is running may produce a corrupted backup.
:::

### JSON export (portable)

Export memories as JSON via the Admin API. This exports content and metadata but not vector embeddings — vectors are re-computed on import.

```bash
curl http://localhost:4200/api/memories/export \
  -X POST \
  -o memories-backup.json
```

The exported JSON is an array of memory objects:

```json
[
  {
    "id": "a1b2c3d4-...",
    "content": "Never commit .env files",
    "scope": "global",
    "type": "rule",
    "tags": ["security"],
    "weight": 1.0,
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-15T10:30:00Z"
  }
]
```

## Restore

### From full backup

```bash
# Stop services
docker compose -f docker/docker-compose.yml -p mememory down

# Replace data directory
rm -rf ~/.mememory
cp -r ~/.mememory-backup-20250115 ~/.mememory

# Restart
docker compose -f docker/docker-compose.yml -p mememory up -d
```

### From JSON export

```bash
curl http://localhost:4200/api/memories/import \
  -X POST \
  -H "Content-Type: application/json" \
  -d @memories-backup.json
```

The import process:
1. Each memory is re-embedded through the current embedding provider
2. New UUIDs are generated (old IDs are not preserved)
3. `supersedes` references are not preserved (old IDs no longer exist)
4. The response reports how many memories were imported

```json
{"imported": 42}
```

::: tip
JSON export/import is the recommended method when migrating between different embedding providers. The re-embedding ensures vectors are compatible with the new provider.
:::

## Migration Between Machines

### Method 1: Full data copy

If both machines use the same embedding provider and model:

```bash
# On source machine
docker compose -f docker/docker-compose.yml -p mememory down
tar czf mememory-data.tar.gz -C ~ .mememory

# Transfer to target machine
scp mememory-data.tar.gz user@target:~/

# On target machine
tar xzf mememory-data.tar.gz -C ~
docker compose -f docker/docker-compose.yml -p mememory up -d
```

### Method 2: JSON export/import

Works across different embedding providers:

```bash
# On source machine
curl http://localhost:4200/api/memories/export -X POST -o memories.json

# Transfer
scp memories.json user@target:~/

# On target machine (stack must be running)
curl http://localhost:4200/api/memories/import \
  -X POST \
  -H "Content-Type: application/json" \
  -d @memories.json
```

## Resetting

To delete all data and start fresh:

```bash
# Stop everything
docker compose -f docker/docker-compose.yml -p mememory down

# Remove data and Docker volumes
rm -rf ~/.mememory
docker compose -f docker/docker-compose.yml -p mememory down -v

# Start clean
docker compose -f docker/docker-compose.yml -p mememory up -d
```

Or use the Makefile:

```bash
make clean
```

## Selective Deletion

Delete memories matching specific filters via the Admin API:

```bash
# Delete all memories for a specific project
curl "http://localhost:4200/api/memories?project=old-project" -X DELETE

# Delete all memories of a specific type
curl "http://localhost:4200/api/memories?type=context" -X DELETE

# Delete all memories in a scope
curl "http://localhost:4200/api/memories?scope=persona&project=match&persona=reviewer" -X DELETE
```

Or delete individual memories by ID:

```bash
curl http://localhost:4200/api/memories/MEMORY-UUID-HERE -X DELETE
```
