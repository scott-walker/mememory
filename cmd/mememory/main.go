package main

import (
	"fmt"
	"os"
)

// Set by GoReleaser via ldflags
var Version = "dev"

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
	case "status":
		if err := runStatus(); err != nil {
			fmt.Fprintf(os.Stderr, "status: %v\n", err)
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
  bootstrap    Load memories for the current session (SessionStart hook)
  status       Check health of memory services
  version      Print version

Bootstrap flags:
  --project <name>    Override project name (default: auto-detect from git)
  --persona <name>    Include persona-scoped memories
  --url <url>         Admin API URL (default: http://localhost:4200)`)
}
