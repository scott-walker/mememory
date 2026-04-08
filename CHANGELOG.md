# Changelog

## [0.2.1] - 2026-04-08

### Fixed
- Landing page (`site/.vitepress/theme/Landing.vue`): the `go install` command now points at the actual main package (`/cmd/mememory`), not the module root.
- README: added an explicit Install section. Previous Quick Start assumed the `mememory` binary was already on the PATH without explaining how to get it.
- `site/guide/getting-started.md`: removed dead references to a non-existent `scripts/install.sh` and an unpublished Homebrew tap. Replaced with a pointer to GitHub Releases.
- `site/guide/bootstrap.md`: removed the "legacy `mememory-server --bootstrap`" section. The flag was removed when bootstrap moved into the dedicated `mememory` CLI.
- `site/guide/architecture.md`: removed stale "supports `--bootstrap` mode" line from `mememory-server` description.
- `docs/architecture.md`, `docs/memory-model.md`: SessionStart hook example now uses `mememory bootstrap` instead of the dead `docker exec mememory-admin mememory-server --bootstrap` form.
- `docs/setup.md`: removed broken numbered subheadings inside the BYO Postgres section.
- Removed `docs/examples-bootstrap-hook.sh` — the entire example shell script was based on the removed `--bootstrap` and `--persona` flags.
- `docs/releasing.md`, `.github/ISSUE_TEMPLATE/release.md`: removed verification steps for the Homebrew tap, which is currently disabled in `.goreleaser.yml`.

## [0.2.0] - 2026-04-08

This release rebrands user-visible identifiers to `mememory` everywhere, makes the database connection fully user-configurable, introduces a new `bootstrap` memory type with size-bounded session initialization, and adds OS-aware data persistence with safe install/uninstall flows. **This is a breaking release for existing installations** — see the Breaking section below for migration details.

### Added
- New memory type `bootstrap` — only memories of this type are automatically loaded into the agent's context at session start. All other types are loaded on demand via `recall`.
- Hard-coded `## System` section in bootstrap output with two directives: use `mememory` as the only persistent memory source, and always call `recall` on the first user message.
- Bootstrap output size limit (`MaxBootstrapBytes` = 10KB). CLI prints a warning to stderr and `remember` with `type=bootstrap` returns a warning when the combined bootstrap output exceeds the limit.
- `mememory setup` CLI command: resolves `DATA_DIR`, writes `.env`, brings up the bundled Docker stack.
- `mememory uninstall` CLI command: stops containers but preserves data by default. `--purge` requires interactive path confirmation to delete data.
- OS-aware `DATA_DIR` resolver in the `mememory` CLI: `~/.local/share/mememory` (Linux/XDG), `~/Library/Application Support/mememory` (macOS), `%LOCALAPPDATA%\mememory` (Windows). Override via `DATA_DIR` env.
- pgvector preflight check in `pg.NewClient` — server fails fast with installation hint if the extension is missing.

### Changed
- `internal/memory` package renamed to `internal/engine` to separate domain logic from the "memory" concept.
- `DATABASE_URL` is now required; missing value causes fail-fast with a clear hint. The previous hardcoded `defaultDatabaseURL` constants in server binaries have been removed.
- Scope weights in recall scoring: `project=1.0`, `global=0.8` (previously `persona=1.0`, `project=0.8`, `global=0.6`).
- `mememory bootstrap` and the `mememory://bootstrap` / `mememory://bootstrap/{project}` MCP resources now load only memories with `type=bootstrap` instead of all memories in scope.
- MCP client configuration moved from `~/.claude/.claude.json` to `~/.claude/.mcp.json` (or project-level `.mcp.json`). The MCP server name is now `mememory` (was `memory`).
- MCP server instructions advertise two scopes (`global`, `project`) and six memory types (`fact`, `rule`, `decision`, `feedback`, `context`, `bootstrap`).
- Documentation repositioned: PostgreSQL + pgvector is the required backend; the bundled Docker stack is one quick-start option, not the only path. Bring-your-own-Postgres is a first-class supported configuration.

### Renamed
- Binaries: `memory-server` → `mememory-server`, `memory-admin` → `mememory-admin`.
- MCP resource URI scheme: `memory://` → `mememory://`. Bootstrap resources are now `mememory://bootstrap` and `mememory://bootstrap/{project}`.
- Environment variable `MEMEMORY_DATA_DIR` → `DATA_DIR`.
- PostgreSQL password default: `memory` → `mememory`.

### Removed
- `persona` scope and everything related: `ScopePersona` constant, `persona` column in filters, `--persona` CLI flag, `by_persona` field in `StatsResult`, persona-related parameters from MCP tools.

### Breaking
- **PostgreSQL database and user renamed from `memory` to `mememory`.** Existing installations will not connect with the new defaults. Migration path: stop the stack with `docker compose stop`, move `$DATA_DIR/postgres` aside as a backup (do **not** delete it), then run `mememory setup` to bring up a fresh stack with the new credentials. To recover the old data, restore it manually via `pg_dump` from the backup.
- **MCP URI scheme `memory://` → `mememory://`**. MCP clients hardcoding the old URI must be updated.
- **`--persona` CLI flag and `persona` MCP tool parameter removed.** Any agent or script using these will fail.
- **`MEMEMORY_DATA_DIR` env var removed.** Use `DATA_DIR` instead.
- **Hardcoded `DATABASE_URL` default removed.** Servers fail fast if it is not set. Run `mememory setup` (creates `.env`) or set the env var manually.

## [0.1.1] - 2026-04-03

### Added
- CLI binary `mememory` with `bootstrap`, `status`, `version` commands
- PostgreSQL + pgvector storage backend (`internal/postgres/`)
- Embedding provider factory with Ollama and OpenAI support
- Embedding dimension auto-detection (`probe.go`)
- Shared bootstrap formatter (`internal/bootstrap/`)
- Type definitions package (`internal/types/`)
- GoReleaser configuration for cross-platform builds
- GitHub Actions CI/CD (build, test, branding check)
- Documentation site (`site/`) and docs (`docs/`)
- Docker multi-stage build with embedded admin UI

### Changed
- Migrated from Qdrant to PostgreSQL + pgvector
- Rebranded from `claude-memory` to `mememory`
- Refactored embedding client to interface-based design (`Embedder`)
- Simplified `engine/service.go` — removed Qdrant-specific logic
- Updated Docker Compose services and environment variables
- Updated README with new architecture and quick start

### Removed
- Qdrant client (`internal/qdrant/client.go`)
- Qdrant gRPC dependency

## [0.0.0] - Initial

- Initial commit: MCP memory server with Qdrant backend
