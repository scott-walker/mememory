package mcp

import (
	"context"
	"fmt"
	"strings"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/scott-walker/mememory/internal/bootstrap"
	"github.com/scott-walker/mememory/internal/engine"
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
}

func bootstrapHandler(svc *engine.Service) server.ResourceHandlerFunc {
	return func(ctx context.Context, req mcpsdk.ReadResourceRequest) ([]mcpsdk.ResourceContents, error) {
		// Load all global memories — they form the base context for any session
		memories, err := svc.List(ctx, engine.ListInput{
			Scope: "global",
			Type:  "bootstrap",
			Limit: 50,
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

		// Load global memories
		global, err := svc.List(ctx, engine.ListInput{
			Scope: "global",
			Type:  "bootstrap",
			Limit: 50,
		})
		if err != nil {
			return nil, fmt.Errorf("bootstrap global: %w", err)
		}

		// Load project-scoped memories
		projectMems, err := svc.List(ctx, engine.ListInput{
			Scope:   "project",
			Project: project,
			Type:    "bootstrap",
			Limit:   50,
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

