package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/scott-walker/mememory/internal/pinned"
	t "github.com/scott-walker/mememory/internal/types"
)

type pinnedArgs struct {
	project string
	url     string
	hook    bool
}

func parsePinnedArgs(args []string) pinnedArgs {
	pa := pinnedArgs{url: envOrDefault("MEMORY_URL", defaultAdminURL)}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--hook":
			pa.hook = true
		case "--project":
			if i+1 < len(args) {
				pa.project = args[i+1]
				i++
			}
		case "--url":
			if i+1 < len(args) {
				pa.url = args[i+1]
				i++
			}
		}
	}

	return pa
}

func runPinned(args pinnedArgs) error {
	// Reuse the bootstrap project resolver: same priority chain (--project flag
	// → .mememory file → git toplevel → cwd basename).
	project := detectProject(args.project)

	client := &http.Client{Timeout: 5 * time.Second}

	globalMems, err := fetchMemories(client, args.url, "global", "", "pinned", 100)
	if err != nil {
		return nil
	}

	var projectMems []t.Memory
	if project.Name != "" {
		projectMems, _ = fetchMemories(client, args.url, "project", project.Name, "pinned", 100)
	}

	if len(globalMems) == 0 && len(projectMems) == 0 {
		return nil
	}

	pctx := pinned.Context{
		Project:     project,
		GlobalMems:  globalMems,
		ProjectMems: projectMems,
	}

	if args.hook {
		payload, err := pinned.FormatHookJSON(pctx)
		if err != nil {
			return err
		}
		if payload != "" {
			fmt.Println(payload)
		}
		return nil
	}

	fmt.Print(pinned.Format(pctx))
	return nil
}
