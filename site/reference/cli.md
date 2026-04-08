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

### bootstrap

Load `bootstrap`-type memories for the current session and print them as Markdown to stdout. Designed for use in `SessionStart` hooks. Only memories with `type=bootstrap` are returned — everything else must be fetched by the agent via `recall`.

```bash
mememory bootstrap [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--project <name>` | auto-detect from git | Project name for scope filtering |
| `--url <url>` | `http://localhost:4200` | Admin API URL |

**Behavior:**

1. If `--project` is not set, auto-detects from the current git repository's root directory name. Falls back to the current directory name if not inside a git repo.
2. Fetches global memories with `type=bootstrap` from the Admin API
3. If a project is set, also fetches project-scoped bootstrap memories
4. Formats all memories as Markdown with a hard-coded `## System` section followed by grouped memories (bootstrap > rules > feedback > facts > decisions > context)
5. If the combined output exceeds `MaxBootstrapBytes` (10KB), prints a warning to stderr — MCP clients may truncate the output, so the bootstrap set should be kept small
6. Prints to stdout

**Exit behavior:**

- If the Admin API is unreachable, exits silently with no output (exit code 0). This ensures bootstrap never blocks agent session start.
- If memories are found, prints formatted Markdown and exits with code 0.
- If no memories match, produces no output and exits with code 0.

**Examples:**

```bash
# Auto-detect project from git
mememory bootstrap

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
  setup        Bring up the bundled Docker stack and write .env
  uninstall    Stop the Docker stack (data preserved). Use --purge to also delete data
  bootstrap    Load memories for the current session (SessionStart hook)
  status       Check health of memory services
  version      Print version

Bootstrap flags:
  --project <name>    Override project name (default: auto-detect from git)
  --url <url>         Admin API URL (default: http://localhost:4200)

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

## SessionStart Hook Usage

The primary use case for the CLI is as a `SessionStart` hook in Claude Code:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "type": "command",
        "command": "mememory bootstrap"
      }
    ]
  }
}
```

The hook captures stdout and injects it into the agent's context. Stderr output (from `status`) is not captured.
