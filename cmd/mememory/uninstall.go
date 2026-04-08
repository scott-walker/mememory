package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runUninstall stops the Docker stack. With --purge it additionally deletes
// the data directory after interactive path confirmation.
func runUninstall(args []string) error {
	purge := false
	for _, a := range args {
		if a == "--purge" {
			purge = true
		}
	}

	dataDir, err := ResolveDataDir()
	if err != nil {
		return err
	}

	composeFile := composePath(dataDir)
	if _, err := os.Stat(composeFile); err != nil {
		return fmt.Errorf("compose file not found at %s — was mememory set up?", composeFile)
	}

	stop := exec.Command("docker", "compose", "-f", composeFile, "-p", "mememory", "down")
	stop.Dir = filepath.Dir(composeFile)
	stop.Stdout = os.Stdout
	stop.Stderr = os.Stderr
	if err := stop.Run(); err != nil {
		return fmt.Errorf("docker compose down: %w", err)
	}

	if !purge {
		fmt.Println()
		fmt.Println("\u2713 Containers stopped.")
		fmt.Printf("Data preserved at: %s\n", dataDir)
		fmt.Println("To completely remove data: mememory uninstall --purge")
		return nil
	}

	fmt.Printf("\nWARNING: This will permanently delete all data at:\n  %s\n\n", dataDir)
	fmt.Print("Type the full path to confirm: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != dataDir {
		return fmt.Errorf("path mismatch, aborting")
	}

	if err := os.RemoveAll(dataDir); err != nil {
		return fmt.Errorf("remove data dir: %w", err)
	}
	fmt.Printf("\u2713 Data directory removed: %s\n", dataDir)
	return nil
}
