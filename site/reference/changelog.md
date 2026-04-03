# Changelog

All notable changes to mememory are documented here.

## Unreleased

### Features

- **MCP server** with 7 tools: `remember`, `recall`, `forget`, `update`, `list`, `stats`, `help`
- **MCP resources**: `memory://bootstrap` and `memory://bootstrap/{project}` for session initialization
- **Hierarchical scopes**: global, project, persona with automatic inheritance during recall
- **5 memory types**: fact, rule, decision, feedback, context
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
- **Session bootstrap**: Markdown-formatted memory output for SessionStart hooks, grouped by type (rules > feedback > facts > decisions > context)
- **Auto-detect project**: CLI detects project name from git repository root
- **Docker stack**: PostgreSQL (pgvector) + Ollama + Admin in a single `docker compose up`
- **Privacy first**: all data stays local, no external telemetry
