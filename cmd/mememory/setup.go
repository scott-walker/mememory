package main

import (
	_ "embed"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

//go:embed docker-compose.prod.yml
var embeddedCompose []byte

// infraDir returns $DATA_DIR/.infra and ensures it exists.
func infraDir(dataDir string) (string, error) {
	dir := filepath.Join(dataDir, ".infra")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create infra dir: %w", err)
	}
	return dir, nil
}

// writeCompose writes the embedded docker-compose.yml and .env into the infra dir.
// Returns the path to docker-compose.yml.
func writeCompose(dataDir string) (string, error) {
	dir, err := infraDir(dataDir)
	if err != nil {
		return "", err
	}

	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composePath, embeddedCompose, 0o644); err != nil {
		return "", fmt.Errorf("write compose file: %w", err)
	}

	pgPort := resolvePostgresPort()

	envPath := filepath.Join(dir, ".env")
	envContent := fmt.Sprintf(
		"DATA_DIR=%s\nMEMEMORY_VERSION=%s\nPOSTGRES_PORT=%s\nDATABASE_URL=postgres://mememory:mememory@localhost:%s/mememory?sslmode=disable\n",
		dataDir, Version, pgPort, pgPort,
	)
	if err := os.WriteFile(envPath, []byte(envContent), 0o644); err != nil {
		return "", fmt.Errorf("write .env: %w", err)
	}

	return composePath, nil
}

// resolvePostgresPort returns the host port for Postgres.
// Uses POSTGRES_PORT env if set, otherwise tries 5432, falls back to 5434.
func resolvePostgresPort() string {
	if p := os.Getenv("POSTGRES_PORT"); p != "" {
		return p
	}
	if portFree("5432") {
		return "5432"
	}
	fmt.Println("Port 5432 is in use, using 5434 for Postgres")
	return "5434"
}

func portFree(port string) bool {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

// composePath returns the path to the compose file under $DATA_DIR/.infra.
// Does NOT create or write anything.
func composePath(dataDir string) string {
	return filepath.Join(dataDir, ".infra", "docker-compose.yml")
}

// runSetup resolves DATA_DIR, writes the embedded compose stack, brings it up,
// and pulls the embedding model.
func runSetup() error {
	dataDir, err := ResolveDataDir()
	if err != nil {
		return err
	}

	composeFile, err := writeCompose(dataDir)
	if err != nil {
		return err
	}
	composeDir := filepath.Dir(composeFile)

	// Pre-create data subdirectories so Docker bind-mounts work.
	for _, sub := range []string{"postgres", "ollama"} {
		if err := os.MkdirAll(filepath.Join(dataDir, sub), 0o755); err != nil {
			return fmt.Errorf("create %s dir: %w", sub, err)
		}
	}

	fmt.Println("Starting Docker stack...")
	cmd := exec.Command("docker", "compose", "-f", composeFile, "-p", "mememory", "up", "-d")
	cmd.Dir = composeDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose up: %w", err)
	}

	fmt.Println("Pulling embedding model (nomic-embed-text)...")
	if err := pullOllamaModel(); err != nil {
		return fmt.Errorf("pull embedding model: %w", err)
	}

	fmt.Println()
	fmt.Println("\u2713 mememory is running")
	fmt.Printf("  Data directory: %s\n", dataDir)
	fmt.Printf("  Compose file:   %s\n", composeFile)
	fmt.Println("  Admin UI:       http://localhost:4200")
	return nil
}

// pullOllamaModel waits for the ollama container to be healthy, then pulls
// nomic-embed-text via docker exec.
func pullOllamaModel() error {
	// Wait for container to be running (up to 60s)
	for i := 0; i < 30; i++ {
		out, err := exec.Command("docker", "inspect", "--format", "{{.State.Health.Status}}", "mememory-ollama").Output()
		if err == nil {
			status := string(out)
			if len(status) > 0 && status[:len(status)-1] == "healthy" {
				break
			}
		}
		if i == 29 {
			return fmt.Errorf("ollama container did not become healthy within 60s")
		}
		time.Sleep(2 * time.Second)
	}

	cmd := exec.Command("docker", "exec", "mememory-ollama", "ollama", "pull", "nomic-embed-text")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
