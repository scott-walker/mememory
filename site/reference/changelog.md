# Changelog

All notable changes to mememory are documented here.

## Unreleased

### Refactor: pluggable connection

- Renamed `internal/memory` package to `internal/engine` to better separate domain logic from the "memory" concept.
- Removed hardcoded `defaultDatabaseURL` constants from server binaries. `DATABASE_URL` is now required; missing value causes fail-fast with a clear hint.
- Added pgvector preflight check in `pg.NewClient` — server fails fast with installation hint if the extension is missing.
- Renamed `MEMEMORY_DATA_DIR` env var to `DATA_DIR`.
- Added OS-aware default `DATA_DIR` resolver in the `mememory` CLI: `~/.local/share/mememory` (Linux/XDG), `~/Library/Application Support/mememory` (macOS), `%LOCALAPPDATA%\mememory` (Windows).
- New `mememory setup` command: resolves DATA_DIR, writes `.env`, brings up bundled Docker stack.
- New `mememory uninstall` command: stops containers but preserves data by default. `--purge` flag requires interactive path confirmation to delete data.
- PostgreSQL password default changed from `memory` to `mememory`.

### Added

- **`bootstrap` memory type**: only memories of this type are loaded automatically at session start. All other types are fetched on demand via `recall`.
- **Bootstrap System section**: hard-coded directives at the top of every bootstrap output — "use `mememory` as the only persistent memory source" and "always call `recall` on the user's first message".
- **Bootstrap size limit**: output is capped at 10KB (`MaxBootstrapBytes`). `mememory bootstrap` prints a stderr warning when the limit is exceeded, and `remember(type="bootstrap", ...)` warns when the combined set would exceed it.

### Changed

- **Two scopes only**: removed the `persona` scope. Scopes are now `global` and `project`. The `--persona` CLI flag, `persona` column in filters, `by_persona` stats field, and `ScopePersona` constant are gone.
- **Updated scope weights** in recall scoring: `project=1.0`, `global=0.8` (previously `persona=1.0`, `project=0.8`, `global=0.6`).
- **Bootstrap loads only `type=bootstrap`**: `mememory bootstrap`, `memory-server --bootstrap`, and the `memory://bootstrap` / `memory://bootstrap/{project}` MCP resources now filter by `type=bootstrap` instead of returning all memories in scope.
- **MCP config path**: Claude Code configuration moved from `~/.claude/.claude.json` to `~/.claude/.mcp.json` (or project-level `.mcp.json`). The server name is now `mememory` (was `memory`).
- **MCP server instructions** advertise two scopes (`global`, `project`) and six types (`fact`, `rule`, `decision`, `feedback`, `context`, `bootstrap`).

### Renamed

- Renamed binaries: `memory-server` → `mememory-server`, `memory-admin` → `mememory-admin`.
- **MCP resource URI scheme** renamed from `memory://` to `mememory://`. The bootstrap resources are now `mememory://bootstrap` and `mememory://bootstrap/{project}`.
- **Environment variable** `MEMORY_DATA_DIR` renamed (later superseded by `DATA_DIR` — see "Refactor: pluggable connection" above).

### Breaking

- **PostgreSQL database and user renamed from `memory` to `mememory`.** Existing installations must reset the postgres data directory (stop the stack, move `$DATA_DIR/postgres` aside, then `mememory setup`) — the old `memory` database will not be migrated automatically.

### Features

- **MCP server** with 7 tools: `remember`, `recall`, `forget`, `update`, `list`, `stats`, `help`
- **MCP resources**: `memory://bootstrap` and `memory://bootstrap/{project}` for session initialization
- **Hierarchical scopes**: `global` and `project`, with project automatically inheriting global during recall
- **6 memory types**: fact, rule, decision, feedback, context, bootstrap
- **Semantic search** via PostgreSQL pgvector with cosine similarity and HNSW indexing
- **Composite scoring**: similarity x scope_weight x memory_weight x temporal_decay
- **Contradiction detection**: warns when new memories are >75% similar to existing ones
- **Belief evolution**: `supersedes` parameter for clean knowledge chains with auto-downgrade
- **TTL support**: auto-expiring memories with hourly cleanup
- **Pluggable embedding providers**: Ollama (local, default) and OpenAI (+ any OpenAI-compatible API)
- **Dynamic embedding dimensions**: auto-detected at startup, validated against database
- **Admin API**: full REST API for CRUD, search, bulk delete, export/import
- **Admin web UI**: React SPA for browsing and managing memories
- **mememory CLI**: native Go binary with `bootstrap`, `status`, `version` commands
- **Session bootstrap**: Markdown-formatted output of `bootstrap`-type memories for SessionStart hooks, capped at 10KB
- **Auto-detect project**: CLI detects project name from git repository root
- **Docker stack**: PostgreSQL (pgvector) + Ollama + Admin in a single `docker compose up`
- **Privacy first**: all data stays local, no external telemetry
