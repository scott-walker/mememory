# What is mememory?

mememory is a persistent semantic memory server for AI agents. It implements the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) and gives agents the ability to store, search, and recall knowledge across sessions.

## The Problem

AI agents start every session with a blank slate. They forget your preferences, project context, architectural decisions, and lessons learned. You end up repeating yourself.

## The Solution

mememory stores knowledge as vector embeddings and delivers it back to the agent automatically at session start. The agent remembers your rules, your project context, and your preferences — without you asking.

## How it works

```
Session starts
    ↓
mememory bootstrap loads your rules and context
    ↓
Agent has full memory from the first message
    ↓
During the session, agent stores new knowledge via MCP tools
    ↓
Next session — everything is remembered
```

## Key Features

- **Semantic search** — recall by meaning, not keywords
- **Hierarchical scopes** — global rules, project-specific context, persona-level behavior
- **Contradiction detection** — warns when new memories conflict with existing ones
- **Belief evolution** — supersede old knowledge without losing history
- **Auto-expiry** — TTL for temporary context (sprint goals, deadlines)
- **Session bootstrap** — rules loaded automatically at session start
- **Pluggable embeddings** — Ollama (local), OpenAI, or any compatible provider
- **Privacy first** — all data stays on your machine

## Architecture

```
mememory CLI (host)           Docker stack
───────────────               ────────────
bootstrap ──────HTTP────────> Admin API (:4200) ──> PostgreSQL + pgvector
                                   │
                                Ollama (embeddings)
```

- **mememory** — native Go binary on the host. Handles setup, bootstrap, status.
- **Docker stack** — PostgreSQL with pgvector for storage, Ollama for local embeddings.
- **MCP server** — runs via stdio, connects agent to the memory store.
