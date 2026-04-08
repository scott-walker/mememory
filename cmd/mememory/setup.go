package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// runSetup resolves DATA_DIR, ensures a .env file exists next to the bundled
// docker-compose.yml, and brings the Docker stack up.
func runSetup() error {
	dataDir, err := ResolveDataDir()
	if err != nil {
		return err
	}

	composePath, err := findComposeFile()
	if err != nil {
		return err
	}
	composeDir := filepath.Dir(composePath)
	envPath := filepath.Join(filepath.Dir(composeDir), ".env")

	// Write .env if it doesn't exist
	if _, err := os.Stat(envPath); err == nil {
		fmt.Printf("Found existing .env at %s\n", envPath)
	} else if os.IsNotExist(err) {
		content := fmt.Sprintf("DATABASE_URL=postgres://mememory:mememory@localhost:5432/mememory?sslmode=disable\nDATA_DIR=%s\n", dataDir)
		if err := os.WriteFile(envPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write .env: %w", err)
		}
		fmt.Printf("Wrote %s\n", envPath)
	} else {
		return fmt.Errorf("stat .env: %w", err)
	}

	fmt.Println("Starting Docker stack...")
	cmd := exec.Command("docker", "compose", "-f", composePath, "up", "-d")
	cmd.Dir = filepath.Dir(composeDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "DATA_DIR="+dataDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose up: %w", err)
	}

	fmt.Println()
	fmt.Println("\u2713 mememory is running")
	fmt.Printf("  Data directory: %s\n", dataDir)
	fmt.Println("  Admin UI: http://localhost:4200")
	fmt.Printf("  To back up: copy %s\n", dataDir)
	return nil
}

// findComposeFile looks for docker/docker-compose.yml relative to cwd or the
// binary location.
func findComposeFile() (string, error) {
	candidates := []string{}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "docker", "docker-compose.yml"),
			filepath.Join(cwd, "docker-compose.yml"),
		)
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "docker", "docker-compose.yml"),
			filepath.Join(filepath.Dir(exeDir), "docker", "docker-compose.yml"),
		)
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("docker-compose.yml not found.\nrun `mememory setup` from the repository root, or place docker/docker-compose.yml beside the binary")
}
