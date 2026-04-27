package main

import (
	"fmt"
	"os"
)

// Set by GoReleaser via ldflags; default reflects current release version.
var Version = "0.6.0"

const defaultAdminURL = "http://localhost:4200"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "bootstrap":
		args := parseBootstrapArgs(os.Args[2:])
		if err := runBootstrap(args); err != nil {
			fmt.Fprintf(os.Stderr, "bootstrap: %v\n", err)
			os.Exit(1)
		}
	case "pinned":
		args := parsePinnedArgs(os.Args[2:])
		if err := runPinned(args); err != nil {
			fmt.Fprintf(os.Stderr, "pinned: %v\n", err)
			os.Exit(1)
		}
	case "recall-gate":
		if err := runRecallGate(); err != nil {
			fmt.Fprintf(os.Stderr, "recall-gate: %v\n", err)
			os.Exit(1)
		}
	case "recall-ack":
		if err := runRecallAck(); err != nil {
			fmt.Fprintf(os.Stderr, "recall-ack: %v\n", err)
			os.Exit(1)
		}
	case "install-hooks":
		args := parseInstallHooksArgs(os.Args[2:])
		if err := runInstallHooks(args); err != nil {
			fmt.Fprintf(os.Stderr, "install-hooks: %v\n", err)
			os.Exit(1)
		}
	case "status":
		if err := runStatus(); err != nil {
			fmt.Fprintf(os.Stderr, "status: %v\n", err)
			os.Exit(1)
		}
	case "setup":
		if err := runSetup(); err != nil {
			fmt.Fprintf(os.Stderr, "setup: %v\n", err)
			os.Exit(1)
		}
	case "uninstall":
		if err := runUninstall(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "uninstall: %v\n", err)
			os.Exit(1)
		}
	case "version":
		fmt.Printf("mememory %s\n", Version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: mememory <command> [flags]

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

Bootstrap flags:
  --hook              Wrap output in a hookSpecificOutput JSON envelope. Use
                      this in SessionStart hook configs for Claude Code and
                      OpenAI Codex CLI — the runner parses it silently and
                      injects the payload into the model context without
                      printing anything to the terminal.
  --project <name>    Override project name. Without this flag, the project is
                      resolved via this priority chain:
                        1. .mememory file (walk-up from cwd)
                        2. git rev-parse --show-toplevel basename
                        3. basename(cwd)
                      See docs/config/mememory-file.md for the .mememory format.
  --url <url>         Admin API URL (default: http://localhost:4200)

Pinned flags:
  Same as bootstrap. Pinned is meant for the UserPromptSubmit hook —
  the rendered payload is wrapped in a <system-reminder> block with rotated
  framing so the rules act as a per-turn checklist. Use --hook to emit the
  hookSpecificOutput JSON envelope (hookEventName=UserPromptSubmit).

Install-hooks flags:
  --uninstall         Remove mememory hooks from settings.json instead of installing
  --path <path>       Override settings.json location (default: ~/.claude/settings.json)

Uninstall flags:
  --purge             Delete the data directory after stopping containers (requires path confirmation)`)
}
