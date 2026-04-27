# Changelog

## [0.6.0] - 2026-04-27

This release introduces a third memory delivery type — `pinned` — for rules that must be reinjected on every agent turn rather than only at session start, plus a forced-recall mechanism that physically blocks tool calls until the agent has loaded project context via `mcp__mememory__recall`. Together they turn critical rules into a per-turn checklist instead of background that drifts up the context.

### Added

- **`delivery=pinned` memory type.** Reinjected on every agent turn through the `UserPromptSubmit` hook. Payload wraps in `<system-reminder>` with rotated framing imperatives so the rules stay weighted as a checklist instead of fading into background.
- **System meta-rules layer** hard-coded in the binary (`internal/system_rules/`). Five formulations of each meta-rule (recall mandate, code-vs-memory truth source, "rule violation = task failure") plus rotated openings/closings defend against agent adaptation to a single phrasing.
- **Forced recall via PreToolUse hook.** Lock file keyed on `session_id` is armed at SessionStart and removed by PostToolUse on `mcp__mememory__recall`. While the lock exists, `mememory recall-gate` denies any tool whose name doesn't start with `mcp__mememory__`. Stale locks (>24h) are GC'd at the next SessionStart.
- **`mememory install-hooks` command.** Idempotent `~/.claude/settings.json` patcher with timestamped backup. Preserves existing settings and foreign hooks; leaves customised mememory commands alone on re-install. `--uninstall` removes the four entries cleanly.
- **`mememory setup` interactive prompt.** At the end of setup, asks whether to install the four Claude Code hooks now.
- **CLI commands:** `pinned`, `recall-gate`, `recall-ack`, `install-hooks`. `bootstrap --hook` now also creates the recall-pending lock from the SessionStart stdin payload.
- **MCP resources:** `mememory://pinned` and `mememory://pinned/{project}`.
- **Soft budget warning** at ~5,000 tokens for the pinned payload — informational, never blocks. Pinned must stay tight to act as a checklist.
- **API endpoint** `GET /api/pinned/preview?project=...` returns rendered markdown plus stats `{global, project, tokens}`.
- **Admin UI: Pinned Preview page** at `/pinned` renders the exact payload your agent receives for any project, with token estimate and per-scope counts.
- **`pinned` option** in admin UI delivery filter (Memories page) and New Memory form.
- **New documentation page:** `site/guide/pinned.md` — what pinned is, how it differs from bootstrap, hook chain, forced recall mechanism, settings.json shape, CLI usage, MCP resources alternative.

### Changed

- **Bootstrap output** now includes a forced-recall directive in the System section reminding the agent that `recall` is the obligatory first operation in a session.
- **MCP `remember` tool** description and help texts mention `pinned` everywhere; soft-budget warning fires when the pinned set exceeds the threshold.
- **Persona scope** removed from the entire admin UI (legacy — backend already without it). Cleaned in `FilterBar`, `MemoryForm`, `MemoryList`, `Search`, `Settings`, `MemoryCard`, `MemoryDetail`, `Badge`, `StatsCards`, plus the unused `--color-scope-persona` CSS token.
- **Documentation updates:** bootstrap.md cross-link to pinned, getting-started.md install-hooks step, mcp-client-setup.md hook config (all four hooks), reference/cli.md command details, README feature bullets, docs/setup.md.

### Compatibility

- **No SQL migration.** The `delivery` column is `TEXT`, so adding `pinned` is a Go-only change. Existing `bootstrap` and `on_demand` memories are untouched.
- **OpenAI Codex CLI** install-hooks parity scheduled for 0.7.0. Codex users can still wire `SessionStart → mememory bootstrap --hook` manually.

## [0.5.0] - 2026-04-09

This release separates the loading strategy from the semantic type by introducing a new `delivery` dimension (`bootstrap` | `on_demand`). Previously, `type=bootstrap` served double duty as both a semantic category and a loading mechanism, which meant bootstrap memories lost their true type (rule, fact, feedback, etc.). Now any memory type can be marked as `delivery=bootstrap` to be loaded at session start, while retaining its semantic classification.

### Added
- **`delivery` field** on all memories: `bootstrap` (loaded at session start) or `on_demand` (fetched via recall/list, default). Independent of `type`.
- **`delivery` parameter** in MCP tools `remember` and `list` for creating and filtering by delivery strategy.
- **`delivery` filter** in admin HTTP API (`?delivery=bootstrap` on list and bulk-delete endpoints).
- **`by_delivery` breakdown** in `stats` tool output.
- **DB migration `003_add_delivery.sql`**: adds `delivery` column with default `on_demand`, auto-migrates existing `type=bootstrap` records to `delivery=bootstrap`, adds index.
- **`DeliveryBadge` component** in web UI — shows an amber "bootstrap" badge on bootstrap memories.
- **`delivery` filter and selector** in web UI (FilterBar, MemoryForm).
- **`.mememory` file reference page** added to VitePress site.

### Changed
- **Bootstrap loading** now filters by `delivery=bootstrap` instead of `type=bootstrap` — across MCP resources, CLI `mememory bootstrap`, and the `remember` tool's budget check.
- **Bootstrap output grouping** renders memories by their semantic type (Rules, Facts, Feedback, etc.) instead of a separate "Bootstrap" section.
- **Help texts** updated: `type` enum no longer includes `bootstrap`; new `delivery` parameter documented in all relevant sections.

### Breaking
- **`type=bootstrap` no longer controls session-start loading.** After upgrading, existing `type=bootstrap` memories will have `delivery=bootstrap` set by the migration, but their `type` remains `bootstrap` — which is no longer a recognized type in the rendering pipeline. **Action required:** update these memories to a correct semantic type (`rule`, `fact`, `feedback`, etc.) via the admin UI or MCP `update` tool. Until fixed, they will not appear in bootstrap output sections.
- **`TypeBootstrap` constant removed** from Go API (`internal/types`). External code referencing `types.TypeBootstrap` will fail to compile.

## [0.4.0] - 2026-04-09

This release introduces the `.mememory` project config file, replaces the byte-based bootstrap limit with a token-based budget, and adds a `## Bootstrap Stats` reporting block to every bootstrap payload. The hook becomes self-sufficient: a single global SessionStart hook (`mememory bootstrap` with no flags) now resolves the canonical project name from a `.mememory` file walked up from `cwd`, eliminating the need for project-local hook configuration.

### Added
- **`.mememory` project config file (schema v1).** A JSON file at the project root that pins the canonical project name for any directory inside the tree. Discovered via walk-up search from `cwd`, mirroring how `git` locates its repository root. Reserved field paths for future versions (`bootstrap.budget_tokens`, `recall.auto_query`, `agent.profile`, etc.) are documented but not yet active. See `docs/config/mememory-file.md` for the full schema spec, walk-up rules, versioning policy, and examples.
- **New `internal/projectconfig` package.** Parses and validates `.mememory` files with forward-compatible unknown-field handling. Walk-up discovery (`FindWalkUp`), explicit validation (`Validate`), and future-version detection (`IsFutureVersion`).
- **`## Bootstrap Stats` section in bootstrap output.** Every bootstrap payload now ends with a stats block reporting: project name + source label (`.mememory` file / git / cwd basename / `--project` flag), loaded memory counts split by scope (global vs project), token estimate, budget percent, raw byte count, and an overflow warning if the budget is exceeded. Visible in every session at start.
- **Token-based bootstrap budget.** New constants `bootstrap.MaxBootstrapTokens` (30_000) and `bootstrap.BytesPerToken` (3.5). The 30K-token ceiling corresponds to ~15% of a 200K-token context window — bootstrap stays small enough that it never crowds out the conversation regardless of model.
- **`bootstrap.EstimateTokens(bytes)`** helper for converting byte counts to estimated token counts.
- **`bootstrap.CheckBudget(memories)`** helper that returns a non-empty warning string if a memory set would exceed the bootstrap token budget. Used by the `remember` MCP tool to flag overflow when adding bootstrap-typed memories.
- **Project resolution priority chain in `mememory bootstrap`.** Order: `--project` flag → `.mememory` file (walk-up) → `git rev-parse --show-toplevel` basename → `basename(cwd)`. The chosen source is reported in the Stats block.
- **Unit tests** for `internal/projectconfig` (10 tests covering load, validation, walk-up, future versions, error paths) and `internal/bootstrap` (10 tests covering Format, Stats, EstimateTokens, formatThousands, CheckBudget). The project previously had no tests.
- **`docs/config/mememory-file.md`** — full specification of the `.mememory` file format.
- **`BACKLOG.md`** — backlog tracking for tags, bootstrap MCP tool, content compression, and auto-recall (item 4 added in this release; items 2/3 actualized).

### Changed
- **`bootstrap.Format` signature is now struct-based:** `Format(Context) string` where `Context` bundles `ProjectInfo`, `GlobalMems`, and `ProjectMems`. The previous `Format(project string, memories []Memory)` form is gone. This gives the formatter enough information to render the Stats block accurately (project source, scope split) without juggling positional parameters.
- **`bootstrap.CheckSize(project, memories)` renamed to `bootstrap.CheckBudget(memories)`** with simplified signature. Internal Go API only.
- **`MaxBootstrapBytes` (10 KB) replaced by `MaxBootstrapTokens` (30_000).** Empirical testing of Claude Code SessionStart hook output showed no truncation up to 1 MB (the previous 10 KB ceiling was a self-imposed safety margin based on an outdated assumption about MCP client truncation). The new limit is denominated in tokens — a unit that scales with model context windows rather than file system bytes.
- **`mememory bootstrap` help text** rewritten to describe the new project resolution chain and the `.mememory` file.
- **`internal/mcp/resources.go`** migrated to the new `Format(Context)` API. Both the global bootstrap handler and the project bootstrap template handler now construct a `Context` struct with explicit `Source: "MCP resource"`.
- **`internal/mcp/tools.go`** `remember` handler now calls `CheckBudget(allBootstrap)` instead of the removed `CheckSize(proj, allBootstrap)`.
- **`CLAUDE.md`** Bootstrap section rewritten: documents the new resolution chain, `.mememory` file, token budget, Stats block. Removes the outdated reference to `MaxBootstrapBytes`.

### Removed
- `bootstrap.MaxBootstrapBytes` constant (superseded by `MaxBootstrapTokens`).
- `bootstrap.CheckSize` function (superseded by `CheckBudget`).
- The 10 KB hook output ceiling and its associated `WARNING: bootstrap output is XKB` stderr message. Replaced by the in-payload `WARNING: bootstrap exceeds budget by X%` line in the Stats block (printed to stdout, not stderr — visible to the agent).

### Fixed
- **Bootstrap project auto-detection no longer falls back to `cwd` basename for projects whose directory name does not match the canonical mememory project.** Previously, running `mememory bootstrap` from a directory like `/home/scott/dev/remide/projects` would resolve the project as `projects`, missing the canonical `plexo` scope and skipping all project-tagged memories. The `.mememory` file fixes this without requiring CLI flags or per-project hook configuration.
- `cmd/mememory-server`: fix `ln.Close` error return check (lint, from `efef2fb`).

### Added
- **Self-contained CLI install.** `mememory setup` now works from a standalone binary — no git clone or source checkout required. The production `docker-compose.yml` is embedded in the binary via `go:embed` and extracted to `$DATA_DIR/.infra/` on first run.
- **Automatic Postgres port detection.** If port 5432 is already in use, `mememory setup` falls back to 5434 automatically. Override with `POSTGRES_PORT` env var.
- Pre-creates `postgres/` and `ollama/` data subdirectories before `docker compose up` to avoid bind-mount failures on Docker Desktop.

### Changed
- **Docker container renamed:** `mememory-admin` → `mememory`. MCP config args change from `["exec", "-i", "mememory-admin", "mememory-server"]` to `["exec", "-i", "mememory", "server"]`.
- **Binaries inside container renamed:** `mememory-server` → `server`, `mememory-admin` → `admin`. Go source packages (`cmd/mememory-server/`, `cmd/mememory-admin/`) unchanged.
- Production compose uses published Docker images (`ghcr.io/scott-walker/mememory`, `ollama/ollama:latest`) instead of local `build:` directives.
- Ollama container switched from custom Dockerfile (with baked-in model pull) to vanilla `ollama/ollama:latest`. Model is pulled by CLI after container health check.
- `mememory uninstall` now resolves compose file from `$DATA_DIR/.infra/` instead of searching the filesystem.

### Removed
- `docker/ollama.Dockerfile` and `docker/ollama-entrypoint.sh` — replaced by vanilla Ollama image + CLI-driven model pull.
- `scripts/setup.sh` — legacy Qdrant-era setup script, replaced by Go-native setup logic.

### Breaking
- **MCP client config must be updated.** Container name and binary name changed. See Changed section above.
- **Existing Docker containers must be recreated.** Run `mememory uninstall && mememory setup` after upgrading the CLI binary.

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
