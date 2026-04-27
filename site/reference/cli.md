# CLI Reference

The `mememory` CLI is a native Go binary that runs on the host machine. It communicates with the Admin API over HTTP and handles session bootstrap, status checks, and version info.

## Installation

```bash
go install github.com/scott-walker/mememory/cmd/mememory@latest
```

Or build from source:

```bash
make cli    # Produces bin/mememory
```

## Commands

### setup

Bring up the bundled Docker stack and write a `.env` file. Idempotent — never overwrites an existing `.env`.

```bash
mememory setup
```

**What it does:**

1. Resolves `DATA_DIR` (env override or OS-standard auto-resolve, see below).
2. Looks for `docker/docker-compose.yml` relative to the current directory or the binary location.
3. Creates a `.env` next to the compose file with `DATABASE_URL` and `DATA_DIR` if it doesn't already exist.
4. Runs `docker compose -f docker/docker-compose.yml up -d` with `DATA_DIR` exported.
5. Prints the data directory path, Admin UI URL, and a backup hint.

---

### uninstall

Stop the bundled Docker stack. Data is **preserved by default** — no Docker volumes are removed.

```bash
mememory uninstall [--purge]
```

**Without `--purge`:**

- Runs `docker compose down` (without `-v`) — containers stop, bind-mounted data stays untouched.

**With `--purge`:**

- Stops the stack the same way, then prompts for interactive confirmation: you must type the full data directory path. Any other input aborts.
- On confirmation, removes `$DATA_DIR` recursively.
- The CLI never destroys Docker volumes — all data lives in a bind-mounted directory you control.

---

### pinned

Load `pinned`-delivery memories and render them as a `<system-reminder>`-wrapped payload for the `UserPromptSubmit` hook. Pinned is reinjected on every agent turn — see [Pinned Rules & Forced Recall](/guide/pinned).

```bash
mememory pinned [flags]
```

**Flags:** identical to `bootstrap` (`--hook`, `--project`, `--url`).

**Behaviour:**
- The `--hook` envelope sets `hookEventName: "UserPromptSubmit"`.
- Output combines a system meta-rules layer (managed by mememory) with user-defined pinned memories grouped by scope.
- The opening/closing imperatives and meta-rule formulations rotate per render to defend against agent adaptation.
- Empty pinned set → empty output, exit 0 (the hook injects nothing).

---

### install-hooks

Install or remove the four mememory hooks in `~/.claude/settings.json`. Idempotent: re-running on an already-installed configuration is a no-op. Existing customisations (e.g. `--url` flags you added manually) are preserved.

```bash
mememory install-hooks [--uninstall] [--path <settings.json>]
```

**Flags:**

| Flag | Description |
|---|---|
| `--uninstall` | Remove mememory hooks instead of installing them. Foreign hooks remain untouched. |
| `--path <path>` | Override settings.json location. Default: `~/.claude/settings.json`. |

**Behaviour:**
1. Reads the target settings.json (creates an empty `{}` if missing).
2. Writes a backup at `<path>.mememory-backup-<timestamp>` before any modification (skipped only when the source file didn't exist).
3. For install: ensures four entries exist — SessionStart (bootstrap), UserPromptSubmit (pinned), PreToolUse (recall-gate), PostToolUse with matcher `mcp__mememory__recall` (recall-ack). Entries already present (matched by `mememory <command-word>`) are left as-is.
4. For uninstall: removes all entries whose command matches `mememory <command-word>` for any of the four managed words. Empty hook arrays and an empty top-level `hooks` key are cleaned up.

---

### recall-gate

The `PreToolUse` hook command. Reads the hook payload from stdin, checks the recall-pending lock for the session, and emits a deny decision when an unrelated tool tries to run before the agent has called `mcp__mememory__recall` at least once.

```bash
mememory recall-gate    # stdin: Claude Code PreToolUse JSON payload
```

Not intended for direct invocation — wired into `~/.claude/settings.json` by `install-hooks`. Returns exit 0 always; deny is communicated via stdout JSON `{"permissionDecision": "deny", "permissionDecisionReason": "..."}`. Tolerant fallback: if stdin is empty or unparseable, allows the tool through (the pinned-payload reinjection still carries the recall directive in plain text).

---

### recall-ack

The `PostToolUse` hook command (gated by Claude Code's matcher to fire only after `mcp__mememory__recall`). Reads the hook payload from stdin and removes the recall-pending lock for the session, unblocking all tools.

```bash
mememory recall-ack    # stdin: Claude Code PostToolUse JSON payload
```

Not intended for direct invocation. Errors are intentionally swallowed: a missing lock is normal (second recall in the same session), and a stale filesystem error shouldn't break the agent's flow.

---

### bootstrap

Load `bootstrap`-type memories for the current session and print them to stdout. Designed for use in `SessionStart` hooks. Only memories with `type=bootstrap` are returned — everything else must be fetched by the agent via `recall`.

When run with `--hook` and stdin contains a Claude Code SessionStart JSON payload, bootstrap **also** creates the recall-pending lock for forced-recall enforcement and garbage-collects stale locks (older than 24 hours). Manual TTY runs without `--hook` skip the lock step entirely.

```bash
mememory bootstrap [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--hook` | off | Wrap output in a `hookSpecificOutput` JSON envelope (see below). Use this in SessionStart hooks for Claude Code and OpenAI Codex CLI — the runner parses it silently and injects `additionalContext` into the model context without printing anything to the terminal. Without `--hook`, the CLI prints raw Markdown, which is handy for manual inspection but noisy inside hook runners. |
| `--project <name>` | auto-detected via priority chain | Project name override for scope filtering |
| `--url <url>` | `http://localhost:4200` | Admin API URL |

**`--hook` output shape:**

```json
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "<markdown bootstrap payload, same bytes as the non-hook output>"
  }
}
```

This envelope format is the de-facto standard shared by Claude Code and OpenAI Codex CLI SessionStart hooks. Hook runners that recognise the schema parse it silently; runners that do not will print the JSON as plain text (noisy but not destructive).

**Project resolution priority:**

When `--project` is not set, the canonical project name is resolved through this chain. The first source that yields a non-empty name wins, and the chosen source is reported in the `## Bootstrap Stats` block at the end of every payload.

1. `.mememory` file discovered via walk-up from `cwd` (see [`.mememory` File Specification](mememory-file))
2. `git rev-parse --show-toplevel` basename
3. `basename(cwd)` as last-resort fallback

**Behavior:**

1. Resolves the project name (see priority chain above)
2. Fetches global memories with `type=bootstrap` from the Admin API
3. If a project is set, also fetches project-scoped bootstrap memories
4. Formats all memories as Markdown with a hard-coded `## System` section, followed by grouped memories (bootstrap > rules > feedback > facts > decisions > context), followed by a `## Bootstrap Stats` block
5. If the estimated token count exceeds `MaxBootstrapTokens` (30_000 tokens, ≈15% of a 200K-token context window), appends a `WARNING: bootstrap exceeds budget` line to the Stats block — but does not truncate the output
6. Prints to stdout

**Exit behavior:**

- If the Admin API is unreachable, exits silently with no output (exit code 0). This ensures bootstrap never blocks agent session start.
- If memories are found, prints formatted Markdown and exits with code 0.
- If no memories match, produces no output and exits with code 0.

**Examples:**

```bash
# Raw Markdown to stdout (manual inspection)
mememory bootstrap

# JSON envelope for hook runners (Claude Code, Codex CLI)
mememory bootstrap --hook

# Explicit project
mememory bootstrap --project match

# Custom API URL
mememory bootstrap --url http://my-server:9200
```

---

### status

Check the health of the memory services and display basic statistics.

```bash
mememory status
```

**Output (stderr):**

```
Checking http://localhost:4200 ...
OK: 42 memories stored
  global: 20
  project: 22
  project/match: 13
  project/mememory: 9
```

**Exit codes:**

| Code | Meaning |
|------|---------|
| 0 | Admin API is reachable and responding |
| 1 | Admin API is unreachable or returned an error |

On failure, a hint is printed:

```
FAIL: admin API unreachable: connection refused
Fix: mememory setup
```

---

### version

Print the mememory version.

```bash
mememory version
```

**Output:**

```
mememory v0.1.0
```

The version is set by GoReleaser via ldflags at build time. Development builds show `mememory dev`.

---

### help

Print usage information.

```bash
mememory help
mememory --help
mememory -h
```

**Output:**

```
Usage: mememory <command> [flags]

Commands:
  setup           Bring up the bundled Docker stack and write .env
  uninstall       Stop the Docker stack (data preserved). Use --purge to also delete data
  bootstrap       Load memories for the current session (SessionStart hook)
  pinned          Load pinned-delivery rules for reinjection (UserPromptSubmit hook)
  recall-gate     PreToolUse hook — blocks tools until first recall in session
  recall-ack      PostToolUse hook on recall — clears the recall-pending lock
  install-hooks   Install/uninstall Claude Code hooks in ~/.claude/settings.json
  status          Check health of memory services
  version         Print version

Bootstrap / pinned flags:
  --hook              Wrap output in hookSpecificOutput JSON envelope
  --project <name>    Override project name (default: auto-detect)
  --url <url>         Admin API URL (default: http://localhost:4200)

Install-hooks flags:
  --uninstall         Remove mememory hooks instead of installing
  --path <path>       Override settings.json location (default: ~/.claude/settings.json)

Uninstall flags:
  --purge             Delete the data directory after stopping containers (requires path confirmation)
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MEMORY_URL` | `http://localhost:4200` | Admin API URL. Overridden by `--url` flag. |
| `DATA_DIR` | OS-standard path | Persistent data directory. Auto-resolved if unset: `~/.local/share/mememory` (Linux/XDG), `~/Library/Application Support/mememory` (macOS), `%LOCALAPPDATA%\mememory` (Windows). |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success (or silent failure for bootstrap) |
| 1 | Error: unknown command, API failure (status), or other errors |

## Hook Usage

The CLI is designed to back four Claude Code hooks. Install them all in one shot:

```bash
mememory install-hooks
```

This patches `~/.claude/settings.json` with `SessionStart` (bootstrap), `UserPromptSubmit` (pinned), `PreToolUse` (recall-gate), and `PostToolUse` matched on `mcp__mememory__recall` (recall-ack).

For OpenAI Codex CLI, the SessionStart-only manual configuration in `~/.codex/hooks.json` still works — see [Bootstrap — Hook Configuration](/guide/bootstrap#hook-configuration). Full Codex parity (UserPromptSubmit / PreToolUse / PostToolUse) is scheduled for the next release.

The hooks capture stdout and inject it into the agent's context. Stderr output (from `status`) is not captured.
