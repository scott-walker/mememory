package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	t "github.com/scott-walker/mememory/internal/types"
)

func runStatus() error {
	adminURL := envOrDefault("MEMORY_URL", defaultAdminURL)
	client := &http.Client{Timeout: 3 * time.Second}

	fmt.Fprintf(os.Stderr, "Checking %s ...\n", adminURL)

	resp, err := client.Get(adminURL + "/api/stats")
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: admin API unreachable: %v\n", err)
		fmt.Fprintf(os.Stderr, "Fix: docker compose -f docker/docker-compose.yml -p mememory up -d\n")
		return err
	}
	defer resp.Body.Close()

	var stats t.StatsResult
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return fmt.Errorf("decode stats: %w", err)
	}

	fmt.Fprintf(os.Stderr, "OK: %d memories stored\n", stats.Total)
	for scope, count := range stats.ByScope {
		fmt.Fprintf(os.Stderr, "  %s: %d\n", scope, count)
	}
	for project, count := range stats.ByProject {
		fmt.Fprintf(os.Stderr, "  project/%s: %d\n", project, count)
	}

	return nil
}
