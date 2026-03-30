package mcp

import (
	"context"
	"fmt"
	"strings"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/scott/claude-memory/internal/memory"
)

func RegisterResources(srv *server.MCPServer, svc *memory.Service) {
	srv.AddResource(
		mcpsdk.NewResource(
			"memory://bootstrap",
			"Session Bootstrap",
			mcpsdk.WithResourceDescription("Essential memories for session initialization. Read this at the start of every session to load user identity, global rules, and feedback."),
			mcpsdk.WithMIMEType("text/plain"),
		),
		bootstrapHandler(svc),
	)

	srv.AddResourceTemplate(
		mcpsdk.NewResourceTemplate(
			"memory://bootstrap/{project}",
			"Project Bootstrap",
			mcpsdk.WithTemplateDescription("Essential memories for a specific project session. Returns global + project-scoped memories."),
			mcpsdk.WithTemplateMIMEType("text/plain"),
		),
		projectBootstrapHandler(svc),
	)
}

func bootstrapHandler(svc *memory.Service) server.ResourceHandlerFunc {
	return func(ctx context.Context, req mcpsdk.ReadResourceRequest) ([]mcpsdk.ResourceContents, error) {
		// Load all global memories — they form the base context for any session
		memories, err := svc.List(ctx, memory.ListInput{
			Scope: "global",
			Limit: 50,
		})
		if err != nil {
			return nil, fmt.Errorf("bootstrap: %w", err)
		}

		text := formatBootstrap("", memories)

		return []mcpsdk.ResourceContents{
			mcpsdk.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "text/plain",
				Text:     text,
			},
		}, nil
	}
}

func projectBootstrapHandler(svc *memory.Service) server.ResourceTemplateHandlerFunc {
	return func(ctx context.Context, req mcpsdk.ReadResourceRequest) ([]mcpsdk.ResourceContents, error) {
		// Extract project from URI: memory://bootstrap/{project}
		project := extractProject(req.Params.URI)
		if project == "" {
			return nil, fmt.Errorf("project name required in URI")
		}

		// Load global memories
		global, err := svc.List(ctx, memory.ListInput{
			Scope: "global",
			Limit: 50,
		})
		if err != nil {
			return nil, fmt.Errorf("bootstrap global: %w", err)
		}

		// Load project-scoped memories
		projectMems, err := svc.List(ctx, memory.ListInput{
			Scope:   "project",
			Project: project,
			Limit:   50,
		})
		if err != nil {
			return nil, fmt.Errorf("bootstrap project: %w", err)
		}

		all := append(global, projectMems...)
		text := formatBootstrap(project, all)

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
	// memory://bootstrap/match → match
	const prefix = "memory://bootstrap/"
	if strings.HasPrefix(uri, prefix) {
		return strings.TrimPrefix(uri, prefix)
	}
	return ""
}

func formatBootstrap(project string, memories []memory.Memory) string {
	if len(memories) == 0 {
		return "No memories stored yet. Use the 'remember' tool to start building context."
	}

	var b strings.Builder

	b.WriteString("# Session Bootstrap\n\n")
	if project != "" {
		b.WriteString(fmt.Sprintf("Project: %s\n\n", project))
	}

	// Group by type, prioritized: rule > feedback > fact > decision > context
	groups := map[memory.MemoryType][]memory.Memory{}
	for _, m := range memories {
		groups[m.Type] = append(groups[m.Type], m)
	}

	typeOrder := []struct {
		typ   memory.MemoryType
		label string
	}{
		{memory.TypeRule, "Rules"},
		{memory.TypeFeedback, "Feedback"},
		{memory.TypeFact, "Facts"},
		{memory.TypeDecision, "Decisions"},
		{memory.TypeContext, "Context"},
	}

	for _, to := range typeOrder {
		mems := groups[to.typ]
		if len(mems) == 0 {
			continue
		}

		b.WriteString(fmt.Sprintf("## %s\n\n", to.label))
		for _, m := range mems {
			scope := string(m.Scope)
			if m.Project != "" {
				scope += "/" + m.Project
			}
			b.WriteString(fmt.Sprintf("- [%s] %s\n", scope, m.Content))
		}
		b.WriteString("\n")
	}

	return b.String()
}
