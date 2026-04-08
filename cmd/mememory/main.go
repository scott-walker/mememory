package main

import (
	"fmt"
	"os"
)

// Set by GoReleaser via ldflags; default reflects current release version.
var Version = "0.2.0"

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
  setup        Bring up the bundled Docker stack and write .env
  uninstall    Stop the Docker stack (data preserved). Use --purge to also delete data
  bootstrap    Load memories for the current session (SessionStart hook)
  status       Check health of memory services
  version      Print version

Bootstrap flags:
  --project <name>    Override project name (default: auto-detect from git)
  --url <url>         Admin API URL (default: http://localhost:4200)

Uninstall flags:
  --purge             Delete the data directory after stopping containers (requires path confirmation)`)
}
