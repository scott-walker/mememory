# Changelog

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
- Simplified `memory/service.go` — removed Qdrant-specific logic
- Updated Docker Compose services and environment variables
- Updated README with new architecture and quick start

### Removed
- Qdrant client (`internal/qdrant/client.go`)
- Qdrant gRPC dependency

## [0.0.0] - Initial

- Initial commit: MCP memory server with Qdrant backend
