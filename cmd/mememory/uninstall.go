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
// the data directory after interactive path confirmation. Never destroys
// Docker volumes; uses plain `docker compose down`.
func runUninstall(args []string) error {
	purge := false
	for _, a := range args {
		if a == "--purge" {
			purge = true
		}
	}

	composePath, err := findComposeFile()
	if err != nil {
		return err
	}
	composeDir := filepath.Dir(composePath)

	dataDir, err := ResolveDataDir()
	if err != nil {
		return err
	}

	// Always stop containers without -v (volumes preserved)
	stop := exec.Command("docker", "compose", "-f", composePath, "down")
	stop.Dir = filepath.Dir(composeDir)
	stop.Stdout = os.Stdout
	stop.Stderr = os.Stderr
	stop.Env = append(os.Environ(), "DATA_DIR="+dataDir)
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
