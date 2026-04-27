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
	"github.com/scott-walker/mememory/internal/hooks"
	"github.com/scott-walker/mememory/internal/projectconfig"
	t "github.com/scott-walker/mememory/internal/types"
)

// staleLockMaxAge is how long a recall-pending lock file can sit on disk
// before bootstrap considers it abandoned (Claude Code crash, machine
// reboot) and removes it. 24h is generous — sessions normally live minutes
// to hours, never days.
const staleLockMaxAge = 24 * time.Hour

type bootstrapArgs struct {
	project string
	url     string
	hook    bool
}

func parseBootstrapArgs(args []string) bootstrapArgs {
	ba := bootstrapArgs{url: envOrDefault("MEMORY_URL", defaultAdminURL)}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--hook":
			ba.hook = true
		case "--project":
			if i+1 < len(args) {
				ba.project = args[i+1]
				i++
			}
		case "--url":
			if i+1 < len(args) {
				ba.url = args[i+1]
				i++
			}
		}
	}

	return ba
}

func runBootstrap(args bootstrapArgs) error {
	// Resolve the project name through the priority chain. Detection failures
	// are non-fatal: an unknown project just means we cannot fetch the
	// project-scoped slice — globals still load and the report will say so.
	project := detectProject(args.project)

	// Hook mode: read session info from stdin (Claude Code SessionStart payload)
	// and arm the forced-recall lock. Garbage-collect stale locks at the same
	// time so crashed sessions don't accumulate. Manual runs (TTY stdin) skip
	// this entirely — ReadHookInputFromStdin returns an empty input.
	if args.hook {
		if input, err := hooks.ReadHookInputFromStdin(); err == nil && input.SessionID != "" {
			_ = hooks.CreateLock(input.SessionID)
		}
		_, _ = hooks.CleanStaleLocks(staleLockMaxAge)
	}

	client := &http.Client{Timeout: 5 * time.Second}

	// Fetch global bootstrap memories. Admin API outages are silently swallowed
	// so the agent still starts, just without persisted memory context.
	globalMems, err := fetchMemories(client, args.url, "global", "", "bootstrap", 100)
	if err != nil {
		return nil
	}

	var projectMems []t.Memory
	if project.Name != "" {
		projectMems, _ = fetchMemories(client, args.url, "project", project.Name, "bootstrap", 100)
	}


	if len(globalMems) == 0 && len(projectMems) == 0 {
		return nil
	}

	bctx := bootstrap.Context{
		Project:     project,
		GlobalMems:  globalMems,
		ProjectMems: projectMems,
	}

	if args.hook {
		payload, err := bootstrap.FormatHookJSON(bctx)
		if err != nil {
			return err
		}
		if payload != "" {
			fmt.Println(payload)
		}
		return nil
	}

	fmt.Print(bootstrap.Format(bctx))
	return nil
}

func fetchMemories(client *http.Client, baseURL, scope, project, delivery string, limit int) ([]t.Memory, error) {
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
	if delivery != "" {
		q.Set("delivery", delivery)
	}
	u.RawQuery = q.Encode()

	resp, err := client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

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

// detectProject resolves the canonical project name through a priority chain.
//
// Sources, in order:
//  1. Explicit --project flag passed to bootstrap.
//  2. .mememory file discovered via walk-up from cwd.
//  3. git rev-parse --show-toplevel basename, if inside a repo.
//  4. basename(cwd) as a last resort.
//
// The first source that yields a non-empty name wins. The returned source
// label is reported back to the user in the bootstrap stats block so they can
// see exactly where the resolved name came from.
func detectProject(flag string) bootstrap.ProjectInfo {
	if flag != "" {
		return bootstrap.ProjectInfo{Name: flag, Source: "flag"}
	}

	wd, err := os.Getwd()
	if err == nil {
		if found, ferr := projectconfig.FindWalkUp(wd); ferr == nil && found != nil {
			return bootstrap.ProjectInfo{
				Name:   found.File.Project,
				Source: ".mememory file (" + found.Path + ")",
			}
		}
	}

	if name := projectFromGit(); name != "" {
		return bootstrap.ProjectInfo{Name: name, Source: "git"}
	}

	if wd != "" {
		return bootstrap.ProjectInfo{Name: filepath.Base(wd), Source: "cwd basename"}
	}

	return bootstrap.ProjectInfo{}
}

func projectFromGit() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	root := filepath.Base(trimNewline(string(out)))
	if root == "" || root == "." {
		return ""
	}
	return root
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
