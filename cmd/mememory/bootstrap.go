package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/scott-walker/mememory/internal/bootstrap"
	t "github.com/scott-walker/mememory/internal/types"
)

type bootstrapArgs struct {
	project string
	persona string
	url     string
}

func parseBootstrapArgs(args []string) bootstrapArgs {
	ba := bootstrapArgs{url: envOrDefault("MEMORY_URL", defaultAdminURL)}

	for i := 0; i < len(args)-1; i++ {
		switch args[i] {
		case "--project":
			ba.project = args[i+1]
			i++
		case "--persona":
			ba.persona = args[i+1]
			i++
		case "--url":
			ba.url = args[i+1]
			i++
		}
	}

	return ba
}

func runBootstrap(args bootstrapArgs) error {
	// Auto-detect project from git repo if not explicitly set
	if args.project == "" {
		args.project = detectProject()
	}

	client := &http.Client{Timeout: 5 * time.Second}

	// Fetch global memories
	memories, err := fetchMemories(client, args.url, "global", "", "", 100)
	if err != nil {
		// Silent exit if admin API is unreachable — agent starts without memory
		return nil
	}

	// Fetch project-scoped memories
	if args.project != "" {
		projectMems, err := fetchMemories(client, args.url, "project", args.project, "", 100)
		if err == nil {
			memories = append(memories, projectMems...)
		}
	}

	// Fetch persona-scoped memories
	if args.project != "" && args.persona != "" {
		personaMems, err := fetchMemories(client, args.url, "persona", args.project, args.persona, 100)
		if err == nil {
			memories = append(memories, personaMems...)
		}
	}

	if len(memories) == 0 {
		return nil
	}

	fmt.Print(bootstrap.Format(args.project, memories))
	return nil
}

func fetchMemories(client *http.Client, baseURL, scope, project, persona string, limit int) ([]t.Memory, error) {
	u, err := url.Parse(baseURL + "/api/memories/")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("scope", scope)
	q.Set("limit", strconv.Itoa(limit))
	if project != "" {
		q.Set("project", project)
	}
	if persona != "" {
		q.Set("persona", persona)
	}
	u.RawQuery = q.Encode()

	resp, err := client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, body)
	}

	var memories []t.Memory
	if err := json.NewDecoder(resp.Body).Decode(&memories); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return memories, nil
}

// detectProject determines the project name from the current working directory.
// Uses git repository root directory name if inside a git repo, otherwise the
// current directory name. Returns empty string if detection fails.
func detectProject() string {
	// Try git first — most reliable for monorepos and nested directories
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		root := filepath.Base(trimNewline(string(out)))
		if root != "" && root != "." {
			return root
		}
	}

	// Fallback to current working directory name
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Base(wd)
}

func trimNewline(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
