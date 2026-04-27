package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/scott-walker/mememory/internal/hooks"
)

type installHooksArgs struct {
	uninstall bool
	path      string
}

func parseInstallHooksArgs(args []string) installHooksArgs {
	out := installHooksArgs{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--uninstall":
			out.uninstall = true
		case "--path":
			if i+1 < len(args) {
				out.path = args[i+1]
				i++
			}
		}
	}
	return out
}

func runInstallHooks(args installHooksArgs) error {
	path := args.path
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve home dir: %w", err)
		}
		path = filepath.Join(home, ".claude", "settings.json")
	}

	if err := hooks.PatchClaudeSettings(path, !args.uninstall); err != nil {
		return err
	}

	if args.uninstall {
		fmt.Printf("Removed mememory hooks from %s\n", path)
	} else {
		fmt.Printf("Installed mememory hooks into %s\n", path)
		fmt.Println()
		fmt.Println("The following hooks are now active:")
		fmt.Println("  SessionStart       → mememory bootstrap --hook")
		fmt.Println("  UserPromptSubmit   → mememory pinned --hook")
		fmt.Println("  PreToolUse         → mememory recall-gate")
		fmt.Println("  PostToolUse        → mememory recall-ack (matcher: mcp__mememory__recall)")
		fmt.Println()
		fmt.Println("Run with --uninstall to remove.")
	}
	return nil
}
