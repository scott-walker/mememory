package mcp

import (
	"context"
	"fmt"
	"strings"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/scott-walker/mememory/internal/bootstrap"
	"github.com/scott-walker/mememory/internal/engine"
	"github.com/scott-walker/mememory/internal/pinned"
)

func RegisterResources(srv *server.MCPServer, svc *engine.Service) {
	srv.AddResource(
		mcpsdk.NewResource(
			"mememory://bootstrap",
			"Session Bootstrap",
			mcpsdk.WithResourceDescription("Essential memories for session initialization. Read this at the start of every session to load user identity, global rules, and feedback."),
			mcpsdk.WithMIMEType("text/plain"),
		),
		bootstrapHandler(svc),
	)

	srv.AddResourceTemplate(
		mcpsdk.NewResourceTemplate(
			"mememory://bootstrap/{project}",
			"Project Bootstrap",
			mcpsdk.WithTemplateDescription("Essential memories for a specific project session. Returns global + project-scoped memories."),
			mcpsdk.WithTemplateMIMEType("text/plain"),
		),
		projectBootstrapHandler(svc),
	)

	srv.AddResource(
		mcpsdk.NewResource(
			"mememory://pinned",
			"Active Pinned Rules",
			mcpsdk.WithResourceDescription("Pinned-delivery rules for per-turn reinjection. Wrapped in <system-reminder> with rotated framing — meant for the UserPromptSubmit hook."),
			mcpsdk.WithMIMEType("text/plain"),
		),
		pinnedHandler(svc),
	)

	srv.AddResourceTemplate(
		mcpsdk.NewResourceTemplate(
			"mememory://pinned/{project}",
			"Project Pinned Rules",
			mcpsdk.WithTemplateDescription("Pinned-delivery rules for a specific project. Returns global + project-scoped pinned memories wrapped for UserPromptSubmit reinjection."),
			mcpsdk.WithTemplateMIMEType("text/plain"),
		),
		projectPinnedHandler(svc),
	)
}

func bootstrapHandler(svc *engine.Service) server.ResourceHandlerFunc {
	return func(ctx context.Context, req mcpsdk.ReadResourceRequest) ([]mcpsdk.ResourceContents, error) {
		// Load all global bootstrap memories — they form the base context for any session
		memories, err := svc.List(ctx, engine.ListInput{
			Scope:    "global",
			Delivery: "bootstrap",
			Limit:    50,
		})
		if err != nil {
			return nil, fmt.Errorf("bootstrap: %w", err)
		}

		text := bootstrap.Format(bootstrap.Context{
			Project:    bootstrap.ProjectInfo{Source: "MCP resource"},
			GlobalMems: memories,
		})

		return []mcpsdk.ResourceContents{
			mcpsdk.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "text/plain",
				Text:     text,
			},
		}, nil
	}
}

func projectBootstrapHandler(svc *engine.Service) server.ResourceTemplateHandlerFunc {
	return func(ctx context.Context, req mcpsdk.ReadResourceRequest) ([]mcpsdk.ResourceContents, error) {
		// Extract project from URI: mememory://bootstrap/{project}
		project := extractProject(req.Params.URI)
		if project == "" {
			return nil, fmt.Errorf("project name required in URI")
		}

		// Load global bootstrap memories
		global, err := svc.List(ctx, engine.ListInput{
			Scope:    "global",
			Delivery: "bootstrap",
			Limit:    50,
		})
		if err != nil {
			return nil, fmt.Errorf("bootstrap global: %w", err)
		}

		// Load project-scoped bootstrap memories
		projectMems, err := svc.List(ctx, engine.ListInput{
			Scope:    "project",
			Project:  project,
			Delivery: "bootstrap",
			Limit:    50,
		})
		if err != nil {
			return nil, fmt.Errorf("bootstrap project: %w", err)
		}

		text := bootstrap.Format(bootstrap.Context{
			Project: bootstrap.ProjectInfo{
				Name:   project,
				Source: "MCP resource",
			},
			GlobalMems:  global,
			ProjectMems: projectMems,
		})

		return []mcpsdk.ResourceContents{
			mcpsdk.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "text/plain",
				Text:     text,
			},
		}, nil
	}
}

func extractProject(uri string) string {
	// mememory://bootstrap/match → match
	const prefix = "mememory://bootstrap/"
	if strings.HasPrefix(uri, prefix) {
		return strings.TrimPrefix(uri, prefix)
	}
	return ""
}

func extractPinnedProject(uri string) string {
	const prefix = "mememory://pinned/"
	if strings.HasPrefix(uri, prefix) {
		return strings.TrimPrefix(uri, prefix)
	}
	return ""
}

func pinnedHandler(svc *engine.Service) server.ResourceHandlerFunc {
	return func(ctx context.Context, req mcpsdk.ReadResourceRequest) ([]mcpsdk.ResourceContents, error) {
		memories, err := svc.List(ctx, engine.ListInput{
			Scope:    "global",
			Delivery: "pinned",
			Limit:    100,
		})
		if err != nil {
			return nil, fmt.Errorf("pinned: %w", err)
		}

		text := pinned.Format(pinned.Context{
			Project:    bootstrap.ProjectInfo{Source: "MCP resource"},
			GlobalMems: memories,
		})

		return []mcpsdk.ResourceContents{
			mcpsdk.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "text/plain",
				Text:     text,
			},
		}, nil
	}
}

func projectPinnedHandler(svc *engine.Service) server.ResourceTemplateHandlerFunc {
	return func(ctx context.Context, req mcpsdk.ReadResourceRequest) ([]mcpsdk.ResourceContents, error) {
		project := extractPinnedProject(req.Params.URI)
		if project == "" {
			return nil, fmt.Errorf("project name required in URI")
		}

		global, err := svc.List(ctx, engine.ListInput{
			Scope:    "global",
			Delivery: "pinned",
			Limit:    100,
		})
		if err != nil {
			return nil, fmt.Errorf("pinned global: %w", err)
		}

		projectMems, err := svc.List(ctx, engine.ListInput{
			Scope:    "project",
			Project:  project,
			Delivery: "pinned",
			Limit:    100,
		})
		if err != nil {
			return nil, fmt.Errorf("pinned project: %w", err)
		}

		text := pinned.Format(pinned.Context{
			Project: bootstrap.ProjectInfo{
				Name:   project,
				Source: "MCP resource",
			},
			GlobalMems:  global,
			ProjectMems: projectMems,
		})

		return []mcpsdk.ResourceContents{
			mcpsdk.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "text/plain",
				Text:     text,
			},
		}, nil
	}
}

