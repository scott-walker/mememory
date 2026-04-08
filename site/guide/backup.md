# Backup & Migration

**TL;DR:** Stop the stack, copy `$DATA_DIR`. Or run `pg_dump` against `DATABASE_URL`. That's it.

```bash
# Option A: file copy
mememory uninstall                              # stops containers, preserves data
cp -a "$DATA_DIR" "$DATA_DIR.backup-$(date +%F)"
mememory setup                                  # back up

# Option B: logical dump
pg_dump "$DATABASE_URL" > mememory-$(date +%F).sql
```

All mememory data lives in a single directory on your machine. This page covers backup, restore, export/import, and migration between machines.

## Data Location

By default, all persistent data is stored under an OS-standard path resolved by the `mememory` CLI:

| Platform | Default |
|----------|---------|
| Linux    | `~/.local/share/mememory` (or `$XDG_DATA_HOME/mememory`) |
| macOS    | `~/Library/Application Support/mememory` |
| Windows  | `%LOCALAPPDATA%\mememory` |

```
$DATA_DIR/
├── postgres/       # PostgreSQL data files (memories, vectors, indexes)
└── ollama/         # Ollama model files (nomic-embed-text, etc.)
```

This location is controlled by the `DATA_DIR` environment variable. If unset, the CLI auto-resolves it.

## Backup

### Full backup (recommended)

Stop the stack and copy the data directory:

```bash
# Stop services to ensure data consistency (containers only, data preserved)
mememory uninstall

# Copy the data directory
cp -a "$DATA_DIR" "$DATA_DIR.backup-$(date +%Y%m%d)"

# Restart
mememory setup
```

::: warning
Always stop PostgreSQL before copying its data directory. Copying while PostgreSQL is running may produce a corrupted backup.
:::

### Logical dump (works against any Postgres)

```bash
pg_dump "$DATABASE_URL" > mememory-$(date +%F).sql
```

Restore later with `psql "$DATABASE_URL" < mememory-2026-04-08.sql`.

### JSON export (portable across embedding providers)

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
mememory uninstall
cp -a "$DATA_DIR.backup-20250115" "$DATA_DIR"
mememory setup
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
mememory uninstall
tar czf mememory-data.tar.gz -C "$(dirname "$DATA_DIR")" "$(basename "$DATA_DIR")"

# Transfer to target machine
scp mememory-data.tar.gz user@target:~/

# On target machine
tar xzf mememory-data.tar.gz -C "$(dirname "$DATA_DIR")"
mememory setup
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

To delete all data and start fresh, use `mememory uninstall --purge`. This requires interactive confirmation (you must type the full data directory path):

```bash
mememory uninstall --purge
```

The CLI never destroys Docker volumes — all data lives in a bind-mounted directory you control, and removal happens through `os.RemoveAll` against the path you confirmed.

## Selective Deletion

Delete memories matching specific filters via the Admin API:

```bash
# Delete all memories for a specific project
curl "http://localhost:4200/api/memories?project=old-project" -X DELETE

# Delete all memories of a specific type
curl "http://localhost:4200/api/memories?type=context" -X DELETE

# Delete all memories in a scope
curl "http://localhost:4200/api/memories?scope=project&project=match" -X DELETE
```

Or delete individual memories by ID:

```bash
curl http://localhost:4200/api/memories/MEMORY-UUID-HERE -X DELETE
```
