#!/usr/bin/env bash
# SessionStart hook — loads persistent memory context into the agent session.
#
# Configuration via environment variables:
#   MEMORY_PROJECT   — project name for scoped memories
#   MEMORY_PERSONA   — persona name (requires MEMORY_PROJECT)
#   MEMORY_CONTAINER — Docker container name (default: mememory-admin)
#
# Examples:
#   # Global memories only
#   ./bootstrap-hook.sh
#
#   # Project-scoped
#   MEMORY_PROJECT=myapp ./bootstrap-hook.sh
#
#   # Project + persona
#   MEMORY_PROJECT=myapp MEMORY_PERSONA=reviewer ./bootstrap-hook.sh
#
# Claude Code hook config (~/.claude/settings.json):
#   "SessionStart": [{
#     "matcher": "",
#     "hooks": [{ "type": "command", "command": "MEMORY_PROJECT=myapp ~/.local/bin/memory-bootstrap.sh" }]
#   }]

set -euo pipefail

CONTAINER="${MEMORY_CONTAINER:-mememory-admin}"

ARGS="--bootstrap"
[ -n "${MEMORY_PROJECT:-}" ] && ARGS="$ARGS --project $MEMORY_PROJECT"
[ -n "${MEMORY_PERSONA:-}" ] && ARGS="$ARGS --persona $MEMORY_PERSONA"

# shellcheck disable=SC2086
docker exec "$CONTAINER" mememory-server $ARGS
