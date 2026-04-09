package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/scott-walker/mememory/internal/bootstrap"
	"github.com/scott-walker/mememory/internal/engine"
)

func RegisterTools(srv *server.MCPServer, svc *engine.Service) {
	registerHelp(srv)
	registerRemember(srv, svc)
	registerRecall(srv, svc)
	registerForget(srv, svc)
	registerUpdate(srv, svc)
	registerList(srv, svc)
	registerStats(srv, svc)
}

func registerRemember(srv *server.MCPServer, svc *engine.Service) {
	tool := mcpsdk.NewTool("remember",
		mcpsdk.WithDescription("Store a new memory. Memories persist across sessions and are searchable by semantic similarity."),
		mcpsdk.WithString("content",
			mcpsdk.Required(),
			mcpsdk.Description("The content to remember"),
		),
		mcpsdk.WithString("scope",
			mcpsdk.Description("Memory scope: global (all projects) or project (specific project). Default: global"),
		),
		mcpsdk.WithString("project",
			mcpsdk.Description("Project name (required when scope=project)"),
		),
		mcpsdk.WithString("type",
			mcpsdk.Description("Memory type: fact, rule, decision, feedback, context. Default: fact"),
		),
		mcpsdk.WithString("tags",
			mcpsdk.Description("Comma-separated tags for additional filtering"),
		),
		mcpsdk.WithString("ttl",
			mcpsdk.Description("Time-to-live duration, e.g. '24h', '7d'. Memory auto-expires after this period"),
		),
		mcpsdk.WithNumber("weight",
			mcpsdk.Description("Confidence/priority weight from 0.1 to 1.0. Default: 1.0. Lower weight = less influence in recall results. Use to downgrade outdated beliefs without deleting them"),
		),
		mcpsdk.WithString("supersedes",
			mcpsdk.Description("ID of an existing memory that this one replaces. The old memory will be auto-downgraded and excluded from recall results. Use when your view has changed"),
		),
	)

	srv.AddTool(tool, func(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		content, _ := req.GetArguments()["content"].(string)
		scope, _ := req.GetArguments()["scope"].(string)
		project, _ := req.GetArguments()["project"].(string)
		typ, _ := req.GetArguments()["type"].(string)
		tags, _ := req.GetArguments()["tags"].(string)
		ttl, _ := req.GetArguments()["ttl"].(string)
		weightF, _ := req.GetArguments()["weight"].(float64)
		supersedes, _ := req.GetArguments()["supersedes"].(string)

		if content == "" {
			return mcpsdk.NewToolResultError("content is required"), nil
		}

		// Normalize TTL: convert "7d" to "168h" since Go only supports hour-level durations
		ttl = normalizeTTL(ttl)

		var tagList []string
		if tags != "" {
			tagList = strings.Split(tags, ",")
			for i := range tagList {
				tagList[i] = strings.TrimSpace(tagList[i])
			}
		}

		result, err := svc.Remember(ctx, engine.RememberInput{
			Content:    content,
			Scope:      engine.Scope(scope),
			Project:    project,
			Type:       engine.MemoryType(typ),
			Tags:       tagList,
			Weight:     weightF,
			Supersedes: supersedes,
			TTL:        ttl,
		})
		if err != nil {
			return mcpsdk.NewToolResultError(fmt.Sprintf("remember failed: %v", err)), nil
		}

		// Build response with contradiction warnings
		if len(result.Contradictions) > 0 {
			return jsonResultWithWarning(result)
		}

		// Check bootstrap size when adding a bootstrap memory
		if typ == "bootstrap" {
			proj := project
			allBootstrap, _ := svc.List(ctx, engine.ListInput{
				Scope: "global",
				Type:  "bootstrap",
				Limit: 100,
			})
			if proj != "" {
				projBootstrap, _ := svc.List(ctx, engine.ListInput{
					Scope:   "project",
					Project: proj,
					Type:    "bootstrap",
					Limit:   100,
				})
				allBootstrap = append(allBootstrap, projBootstrap...)
			}
			if warn := bootstrap.CheckBudget(allBootstrap); warn != "" {
				data, _ := json.MarshalIndent(result.Memory, "", "  ")
				return mcpsdk.NewToolResultText(fmt.Sprintf("WARNING: %s\n\nStored memory:\n%s", warn, data)), nil
			}
		}

		return jsonResult(result.Memory)
	})
}

func registerRecall(srv *server.MCPServer, svc *engine.Service) {
	tool := mcpsdk.NewTool("recall",
		mcpsdk.WithDescription("Search memories by semantic similarity. Returns the most relevant memories matching the query. Supports hierarchical search: project scope sees global + project memories."),
		mcpsdk.WithString("query",
			mcpsdk.Required(),
			mcpsdk.Description("Natural language query to search for"),
		),
		mcpsdk.WithString("scope",
			mcpsdk.Description("Filter by scope: global, project. If omitted with project set, uses hierarchical search"),
		),
		mcpsdk.WithString("project",
			mcpsdk.Description("Filter by project name. Enables hierarchical search (global + this project)"),
		),
		mcpsdk.WithNumber("limit",
			mcpsdk.Description("Max results to return. Default: 5"),
		),
	)

	srv.AddTool(tool, func(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		query, _ := req.GetArguments()["query"].(string)
		scope, _ := req.GetArguments()["scope"].(string)
		project, _ := req.GetArguments()["project"].(string)
		limitF, _ := req.GetArguments()["limit"].(float64)

		if query == "" {
			return mcpsdk.NewToolResultError("query is required"), nil
		}

		limit := int(limitF)
		if limit <= 0 {
			limit = 5
		}

		results, err := svc.Recall(ctx, engine.RecallInput{
			Query:   query,
			Scope:   scope,
			Project: project,
			Limit:   limit,
		})
		if err != nil {
			return mcpsdk.NewToolResultError(fmt.Sprintf("recall failed: %v", err)), nil
		}

		return jsonResult(results)
	})
}

func registerForget(srv *server.MCPServer, svc *engine.Service) {
	tool := mcpsdk.NewTool("forget",
		mcpsdk.WithDescription("Delete a memory by its ID"),
		mcpsdk.WithString("id",
			mcpsdk.Required(),
			mcpsdk.Description("Memory ID to delete"),
		),
	)

	srv.AddTool(tool, func(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		id, _ := req.GetArguments()["id"].(string)
		if id == "" {
			return mcpsdk.NewToolResultError("id is required"), nil
		}

		if err := svc.Forget(ctx, id); err != nil {
			return mcpsdk.NewToolResultError(fmt.Sprintf("forget failed: %v", err)), nil
		}

		return mcpsdk.NewToolResultText(fmt.Sprintf("Memory %s deleted", id)), nil
	})
}

func registerUpdate(srv *server.MCPServer, svc *engine.Service) {
	tool := mcpsdk.NewTool("update",
		mcpsdk.WithDescription("Update an existing memory's content. Re-embeds the content for updated semantic search."),
		mcpsdk.WithString("id",
			mcpsdk.Required(),
			mcpsdk.Description("Memory ID to update"),
		),
		mcpsdk.WithString("content",
			mcpsdk.Required(),
			mcpsdk.Description("New content for the memory"),
		),
	)

	srv.AddTool(tool, func(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		id, _ := req.GetArguments()["id"].(string)
		content, _ := req.GetArguments()["content"].(string)

		if id == "" {
			return mcpsdk.NewToolResultError("id is required"), nil
		}
		if content == "" {
			return mcpsdk.NewToolResultError("content is required"), nil
		}

		mem, err := svc.Update(ctx, id, content)
		if err != nil {
			return mcpsdk.NewToolResultError(fmt.Sprintf("update failed: %v", err)), nil
		}

		return jsonResult(mem)
	})
}

func registerList(srv *server.MCPServer, svc *engine.Service) {
	tool := mcpsdk.NewTool("list",
		mcpsdk.WithDescription("List memories with optional filters. No semantic search — returns all matching memories."),
		mcpsdk.WithString("scope",
			mcpsdk.Description("Filter by scope: global, project"),
		),
		mcpsdk.WithString("project",
			mcpsdk.Description("Filter by project name"),
		),
		mcpsdk.WithString("type",
			mcpsdk.Description("Filter by type: fact, rule, decision, feedback, context, bootstrap"),
		),
		mcpsdk.WithNumber("limit",
			mcpsdk.Description("Max results. Default: 20"),
		),
	)

	srv.AddTool(tool, func(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		scope, _ := req.GetArguments()["scope"].(string)
		project, _ := req.GetArguments()["project"].(string)
		typ, _ := req.GetArguments()["type"].(string)
		limitF, _ := req.GetArguments()["limit"].(float64)

		limit := int(limitF)
		if limit <= 0 {
			limit = 20
		}

		memories, err := svc.List(ctx, engine.ListInput{
			Scope:   scope,
			Project: project,
			Type:    typ,
			Limit:   limit,
		})
		if err != nil {
			return mcpsdk.NewToolResultError(fmt.Sprintf("list failed: %v", err)), nil
		}

		return jsonResult(memories)
	})
}

func registerStats(srv *server.MCPServer, svc *engine.Service) {
	tool := mcpsdk.NewTool("stats",
		mcpsdk.WithDescription("Get memory statistics: total count and breakdown by scope, project, and type"),
	)

	srv.AddTool(tool, func(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		stats, err := svc.Stats(ctx)
		if err != nil {
			return mcpsdk.NewToolResultError(fmt.Sprintf("stats failed: %v", err)), nil
		}

		return jsonResult(stats)
	})
}

func registerHelp(srv *server.MCPServer) {
	tool := mcpsdk.NewTool("help",
		mcpsdk.WithDescription("Get documentation on how to use the memory system. Call this FIRST before using any other memory tools. Returns usage guide, tool reference, scope/type taxonomy, and examples."),
		mcpsdk.WithString("topic",
			mcpsdk.Description("Optional topic: 'overview', 'tools', 'scopes', 'types', 'examples', 'best-practices'. Default: full guide"),
		),
	)

	srv.AddTool(tool, func(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		topic, _ := req.GetArguments()["topic"].(string)

		switch topic {
		case "tools":
			return mcpsdk.NewToolResultText(helpTools), nil
		case "scopes":
			return mcpsdk.NewToolResultText(helpScopes), nil
		case "types":
			return mcpsdk.NewToolResultText(helpTypes), nil
		case "examples":
			return mcpsdk.NewToolResultText(helpExamples), nil
		case "best-practices":
			return mcpsdk.NewToolResultText(helpBestPractices), nil
		default:
			return mcpsdk.NewToolResultText(helpFull), nil
		}
	})
}

const helpFull = `# mememory — Persistent Semantic Memory for AI Agents

## What is this?

A vector-based memory system that persists across sessions. You store facts, rules, decisions, and feedback — then retrieve them later by meaning (semantic search), not by exact keywords.

All agents sharing this MCP server share the same memory. Scopes control visibility.

## Quick Start

1. Call 'help' (this tool) to understand the system
2. Call 'recall' with a query to check if relevant memories exist
3. Call 'remember' to store new knowledge
4. Call 'stats' to see what's in the database

## Tools (7 total)

| Tool | Purpose |
|------|---------|
| help | This documentation. Call with topic= for specific sections |
| remember | Store a new memory (embeds content into vector DB) |
| recall | Semantic search — find memories by meaning |
| forget | Delete a memory by ID |
| update | Change content of existing memory (re-embeds) |
| list | Browse memories with metadata filters (no semantic search) |
| stats | Get counts by scope, project, type |

## Scopes (hierarchical visibility)

| Scope | Visible to | Use when |
|-------|-----------|----------|
| global | All projects | User preferences, cross-project rules |
| project | Only within named project | Project-specific architecture, decisions |

Hierarchy: recall(project=Y) searches global + project:Y.

## Types (content classification)

| Type | What to store |
|------|--------------|
| fact | Objective information: "DB is SQLite", "user's name is Scott" |
| rule | Imperatives: "never use native select elements" |
| decision | Choices with reasoning: "chose Zustand because..." |
| feedback | User corrections: "don't refactor without asking" |
| context | Temporal situation: "preparing for demo on April 5" |
| bootstrap | Essential rules loaded at session start automatically |

## Key Parameters

remember: content (required), scope, project, type, tags (comma-separated), ttl (e.g. "24h", "7d"), weight (0.1-1.0), supersedes (old memory ID)
recall: query (required), scope, project, limit (default 5)
list: scope, project, type, limit (default 20)

## Smart Features

### Contradiction Detection
When you call 'remember', the system automatically checks for semantically similar existing memories.
If a potential conflict is found (similarity > 75%), you get a warning with options:
keep both, update old, supersede, or delete old.
IMPORTANT: When you see a contradiction warning, ALWAYS ask the user to clarify before deciding.

### Belief Revision (weight + supersedes)
- weight (0.1-1.0): How confident/current this memory is. Default 1.0. Lower = less influence in recall.
- supersedes: ID of old memory this one replaces. Old memory is auto-downgraded and hidden from recall.
Use case: "I used to believe X, now I believe Y" → remember(content=Y, supersedes=<X_id>)

### Smart Scoring in Recall
Results are ranked by: similarity × scope_weight × weight × temporal_decay
- Scope weight: project (1.0) > global (0.8) — more specific = higher priority
- Temporal decay: newer memories score slightly higher (gentle exponential decay)
- Weight: explicit confidence factor

This means project-level exceptions automatically override global rules in search results.

Call 'help' with topic='examples' or topic='best-practices' for more detail.`

const helpTools = `# Memory Tools Reference

## remember
Store a new memory. Content is embedded into a vector and stored in Qdrant.

Parameters:
- content (string, REQUIRED): The text to remember. Be specific and self-contained.
- scope (string): "global" | "project". Default: "global"
- project (string): Project name. Required when scope=project.
- type (string): "fact" | "rule" | "decision" | "feedback" | "context" | "bootstrap". Default: "fact"
- tags (string): Comma-separated tags for filtering. E.g. "frontend, performance"
- ttl (string): Auto-expire after duration. E.g. "24h", "7d", "30d". Omit for permanent.
- weight (number): Confidence weight 0.1-1.0. Default: 1.0. Use lower values for uncertain or partially outdated beliefs.
- supersedes (string): ID of memory this one replaces. The old memory is auto-downgraded (weight → 0.1) and excluded from recall results.

Returns: Memory object. If contradictions detected — returns warning with similar existing memories and resolution options. ALWAYS present contradiction warnings to the user.

## recall
Semantic search. Finds memories by meaning, not exact match.
"state management" will find "using Zustand for stores" even without shared words.

Parameters:
- query (string, REQUIRED): Natural language query.
- scope (string): Filter to specific scope. Omit for hierarchical search.
- project (string): Filter/enable hierarchical search for this project.
- limit (number): Max results. Default: 5.

Returns: Array of {memory, score} sorted by relevance (score 0-1).

## forget
Delete a memory permanently.

Parameters:
- id (string, REQUIRED): Memory UUID to delete.

## update
Replace content of an existing memory. Re-embeds for updated search.

Parameters:
- id (string, REQUIRED): Memory UUID to update.
- content (string, REQUIRED): New content.

## list
Browse memories with exact filters. No semantic search — returns all matching.

Parameters:
- scope, project, type: Exact match filters.
- limit (number): Max results. Default: 20.

## stats
No parameters. Returns: {total, by_scope, by_project, by_type}.

## help
This tool. Optional: topic= "overview" | "tools" | "scopes" | "types" | "examples" | "best-practices".`

const helpScopes = `# Scopes — Hierarchical Visibility

Memory has two scope levels forming a hierarchy:

## global
- Visible to ALL projects
- Use for: user identity, universal preferences, cross-project rules
- Examples:
  - "User's name is Scott"
  - "User communicates in Russian, respond in Russian"
  - "Never commit .env files"

## project
- Visible only within the named project
- Use for: project-specific architecture, tech stack, decisions
- Requires: project= parameter
- Examples:
  - scope=project, project="match": "Uses SQLite with better-sqlite3, no ORM"
  - scope=project, project="convervox": "Go service with PostgreSQL"

## Hierarchical Search

When you call recall(project="match"):
Searches global + project="match" memories.

When you call recall() with no scope filters:
Searches global only.`

const helpTypes = `# Types — Content Classification

## fact
Objective, verifiable information.
- "The database has 9 tables"
- "User's name is Scott"
- "Frontend uses React 19 + Vite + Tailwind"

## rule
Imperatives that must be followed. Include WHY when possible.
- "Never use native <select> elements — only custom dropdowns"
- "All grays must be metallic (R < G < B) — brand requirement"
- "pnpm only, no npm/yarn"

## decision
A choice that was made, with reasoning. Future agents need to know WHY.
- "Chose sequential data collection (OU first, then BU) because OFI knows OU but not BU"
- "Using Zustand over Redux: simpler API, sufficient for this app size"

## feedback
User corrections to agent behavior. The most important type — prevents repeating mistakes.
- "Don't refactor without explicit permission"
- "Develop backend first, fix frontend separately"
- "Stop summarizing what you just did — user can read the diff"

## context
Temporal/situational information. Often benefits from TTL.
- "Preparing for investor demo on April 5" (ttl="7d")
- "Currently refactoring auth flow, don't touch auth-store.ts"
- "Sprint focus: Evidence Bundle implementation"

## bootstrap
Essential rules and directives loaded automatically at session start.
Only bootstrap-type memories are included in the SessionStart hook output.
All other types are loaded on demand via recall.
- "Always respond in Russian"
- "Use mememory MCP server as the only memory source"`

const helpExamples = `# Usage Examples

## Store a user preference (global fact)
remember(
  content="User communicates only in Russian. Always respond in Russian.",
  scope="global", type="rule", tags="language, communication"
)

## Store a project architecture decision
remember(
  content="Match project uses SQLite with better-sqlite3. No ORM — raw SQL. Schema in server/db.ts with inline migrations.",
  scope="project", project="match", type="fact", tags="database, architecture"
)

## Store user feedback
remember(
  content="Don't refactor code without explicit permission. Minimal diffs only.",
  scope="global", type="feedback", tags="workflow"
)

## Store a temporary context
remember(
  content="Preparing for investor demo on April 5. All UIs must be production-ready.",
  scope="project", project="match", type="context", tags="deadline", ttl="7d"
)

## Search for relevant memories
recall(query="how does the session state machine work", project="match")
recall(query="user preferences for communication", limit=3)
recall(query="database architecture", project="match")

## Browse all feedback
list(type="feedback")

## Browse project-specific rules
list(scope="project", project="match", type="rule")

## Check what's stored
stats()

## Supersede an old belief
# First, find the old memory
recall(query="state management approach")
# Returns: id="abc123", content="Redux is the best state manager"

# Now store updated belief, replacing the old one
remember(
  content="Zustand is better than Redux for small-medium apps — simpler API, less boilerplate",
  scope="global", type="decision",
  supersedes="abc123"
)
# Old memory auto-downgraded to weight 0.1, excluded from recall

## Store with reduced confidence
remember(
  content="GraphQL might be better than REST for this project — not sure yet",
  scope="project", project="match", type="decision",
  weight=0.5, tags="tentative"
)

## Cascade exception: project overrides global rule
# Global rule exists: "Never use ORM"
# In one project, you decided differently:
remember(
  content="In convervox we use Drizzle ORM — complex schema justifies it",
  scope="project", project="convervox", type="decision",
  supersedes="<id of global no-ORM rule>"
)`

const helpBestPractices = `# Best Practices

## 1. Call 'help' first
If you're a new agent connecting to this memory system, call help() before anything else.

## 2. Recall before you act
Before starting work, call recall() with a relevant query. Someone may have already stored context, decisions, or rules that affect your task.

## 3. Be specific in content
BAD: "Uses React"
GOOD: "Frontend uses React 19 + TypeScript + Vite 8 + Tailwind CSS 4. State management via Zustand stores in src/shared/store/."

## 4. Include WHY in decisions
BAD: "We use sequential data collection"
GOOD: "Sequential data collection (OU first, then BU) because OFI knows OU but BU contact comes from OU. Prevents routing errors."

## 5. Use TTL for temporal context
Deadlines, sprint goals, temporary workarounds — set ttl="7d" or similar so they auto-expire.

## 6. Don't duplicate
Before calling remember(), call recall() to check if similar knowledge already exists. Use update() to refine existing memories instead of creating duplicates.

## 7. Use tags for cross-cutting concerns
Tags like "security", "performance", "deadline" help filter across scopes and types.

## 8. Scope correctly
- Would this apply in ANY project? → global
- Only in THIS project? → project

## 9. Feedback is sacred
When the user corrects your behavior, ALWAYS store it as type="feedback". This is the most valuable memory type — it prevents the same mistake across all future sessions.

## 10. Keep content self-contained
Each memory should make sense on its own, without needing to read other memories. Include enough context in the content itself.

## 11. Handle contradictions immediately
When remember() returns a contradiction warning, ALWAYS ask the user:
"I found a similar memory that might conflict: [old content]. Which is correct now?"
Never silently keep contradicting memories — the user expects you to flag this.

## 12. Use supersedes for belief evolution
When the user's opinion changes, don't just add a new memory — supersede the old one.
This creates a clean chain of reasoning and prevents future agents from seeing outdated beliefs.

## 13. Use weight for uncertainty
Not sure about something? Store it with weight=0.5. As confidence grows, update to 1.0.
Partially outdated? Lower to 0.3 instead of deleting — keeps the history.`

// --- Helpers ---

func jsonResult(v interface{}) (*mcpsdk.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcpsdk.NewToolResultError(fmt.Sprintf("json marshal: %v", err)), nil
	}
	return mcpsdk.NewToolResultText(string(data)), nil
}

func jsonResultWithWarning(result *engine.RememberResult) (*mcpsdk.CallToolResult, error) {
	var b strings.Builder

	b.WriteString("⚠ CONTRADICTION DETECTED\n\n")
	b.WriteString("The memory was stored, but similar existing memories were found that may conflict.\n")
	b.WriteString("Ask the user to clarify before proceeding.\n\n")

	b.WriteString("New memory:\n")
	fmt.Fprintf(&b, "  [%s] %s\n\n", result.Memory.ID[:8], result.Memory.Content)

	b.WriteString("Potentially conflicting memories:\n")
	for _, c := range result.Contradictions {
		fmt.Fprintf(&b, "  [%s] (similarity: %.0f%%) %s\n", c.Memory.ID[:8], c.Similarity*100, c.Memory.Content)
	}

	b.WriteString("\nOptions:\n")
	b.WriteString("  1. Keep both — if they are complementary, not contradictory\n")
	b.WriteString("  2. Update old — call update(id=<old_id>, content=<new_content>) to fix the old memory\n")
	b.WriteString("  3. Supersede — call remember(content=..., supersedes=<old_id>) to explicitly replace\n")
	b.WriteString("  4. Delete old — call forget(id=<old_id>) if it's completely obsolete\n")

	b.WriteString("\nStored memory details:\n")
	data, err := json.MarshalIndent(result.Memory, "", "  ")
	if err != nil {
		return mcpsdk.NewToolResultError(fmt.Sprintf("json marshal: %v", err)), nil
	}
	b.Write(data)

	return mcpsdk.NewToolResultText(b.String()), nil
}

func normalizeTTL(ttl string) string {
	if ttl == "" {
		return ""
	}
	// Convert "Nd" to hours
	if strings.HasSuffix(ttl, "d") {
		numStr := strings.TrimSuffix(ttl, "d")
		var days int
		if _, err := fmt.Sscanf(numStr, "%d", &days); err == nil {
			return fmt.Sprintf("%dh", days*24)
		}
	}
	return ttl
}
