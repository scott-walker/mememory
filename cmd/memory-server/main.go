package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/scott-walker/mememory/internal/bootstrap"
	"github.com/scott-walker/mememory/internal/embeddings"
	mcptools "github.com/scott-walker/mememory/internal/mcp"
	"github.com/scott-walker/mememory/internal/memory"
	pg "github.com/scott-walker/mememory/internal/postgres"
)

const (
	defaultDatabaseURL = "postgres://memory:memory@localhost:5432/memory?sslmode=disable"
	defaultOllamaURL   = "http://localhost:11434"
	ttlCleanInterval   = 1 * time.Hour
)

func main() {
	// CLI mode: --bootstrap prints session context to stdout and exits
	//   memory-server --bootstrap                           → global only
	//   memory-server --bootstrap --project myapp           → global + project
	//   memory-server --bootstrap --project myapp --persona dev  → global + project + persona
	if len(os.Args) > 1 && os.Args[1] == "--bootstrap" {
		project, persona := parseBootstrapArgs(os.Args[2:])
		runBootstrap(project, persona)
		return
	}

	logger := log.New(os.Stderr, "[memory-server] ", log.LstdFlags)

	databaseURL := envOrDefault("DATABASE_URL", defaultDatabaseURL)

	logger.Println("Connecting to PostgreSQL")
	pgClient, err := pg.NewClient(databaseURL)
	if err != nil {
		logger.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer func() { _ = pgClient.Close() }()

	logger.Println("Running migrations")
	if err := pgClient.RunMigrations(context.Background()); err != nil {
		logger.Fatalf("Failed to run migrations: %v", err)
	}

	embeddingCfg := embeddings.Config{
		Provider: os.Getenv("EMBEDDING_PROVIDER"),
		URL:      os.Getenv("EMBEDDING_URL"),
		APIKey:   os.Getenv("EMBEDDING_API_KEY"),
		Model:    os.Getenv("EMBEDDING_MODEL"),
	}
	// Backward compatibility: OLLAMA_URL → EMBEDDING_URL when provider is ollama
	if embeddingCfg.Provider == "" || embeddingCfg.Provider == "ollama" {
		if embeddingCfg.URL == "" {
			embeddingCfg.URL = envOrDefault("OLLAMA_URL", defaultOllamaURL)
		}
	}

	embedder, err := embeddings.New(embeddingCfg)
	if err != nil {
		logger.Fatalf("Failed to create embedder: %v", err)
	}
	logger.Printf("Embedding provider: %s", envOrDefault("EMBEDDING_PROVIDER", "ollama"))

	dim, err := embeddings.ProbeDimension(context.Background(), embedder)
	if err != nil {
		logger.Fatalf("Failed to probe embedding dimension: %v", err)
	}
	logger.Printf("Embedding dimension: %d", dim)

	if err := pgClient.EnsureEmbeddingDimension(context.Background(), dim); err != nil {
		logger.Fatalf("Embedding dimension check failed: %v", err)
	}

	svc := memory.NewService(pgClient, embedder)

	srv := server.NewMCPServer(
		"mememory",
		"0.1.1",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
		server.WithInstructions("Persistent semantic memory for AI agents. "+
			"Tools: remember (store), recall (search), forget (delete), update (re-embed), list (browse), stats (counts), help (docs). "+
			"Supports hierarchical scopes (global > project > persona) and memory types (fact, rule, decision, feedback, context)."),
	)

	mcptools.RegisterTools(srv, svc)
	mcptools.RegisterResources(srv, svc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(ttlCleanInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleaned, err := svc.CleanExpired(ctx)
				if err != nil {
					logger.Printf("TTL cleanup error: %v", err)
				} else if cleaned > 0 {
					logger.Printf("TTL cleanup: removed %d expired memories", cleaned)
				}
			}
		}
	}()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		logger.Printf("Received %v, shutting down", sig)
		cancel()
		os.Exit(0)
	}()

	logger.Println("Starting MCP server on stdio")
	stdio := server.NewStdioServer(srv)
	if err := stdio.Listen(ctx, os.Stdin, os.Stdout); err != nil {
		logger.Fatalf("Server error: %v", err)
	}
}

// runBootstrap connects to the database, loads memories for the given scope
// hierarchy, and prints them as formatted Markdown to stdout. Designed to be
// called from a SessionStart hook so the agent receives context automatically.
//
// With no flags: loads global memories only (backward compatible).
// With --project: loads global + project-scoped memories.
// With --project + --persona: loads global + project + persona memories.
func runBootstrap(project, persona string) {
	databaseURL := envOrDefault("DATABASE_URL", defaultDatabaseURL)

	pgClient, err := pg.NewClient(databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap: failed to connect to PostgreSQL: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = pgClient.Close() }()

	ctx := context.Background()

	// Always load global memories
	memories, err := pgClient.List(ctx, pg.Filter{Scope: "global"}, 100)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap: failed to list global memories: %v\n", err)
		os.Exit(1)
	}

	// Load project-scoped memories when project is specified
	if project != "" {
		projectMems, err := pgClient.List(ctx, pg.Filter{Scope: "project", Project: project}, 100)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bootstrap: failed to list project memories: %v\n", err)
			os.Exit(1)
		}
		memories = append(memories, projectMems...)
	}

	// Load persona-scoped memories when both project and persona are specified
	if project != "" && persona != "" {
		personaMems, err := pgClient.List(ctx, pg.Filter{Scope: "persona", Project: project, Persona: persona}, 100)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bootstrap: failed to list persona memories: %v\n", err)
			os.Exit(1)
		}
		memories = append(memories, personaMems...)
	}

	if len(memories) == 0 {
		return
	}

	fmt.Print(bootstrap.Format(project, memories))
}

// parseBootstrapArgs extracts --project and --persona values from args following --bootstrap.
func parseBootstrapArgs(args []string) (project, persona string) {
	for i := 0; i < len(args)-1; i++ {
		switch args[i] {
		case "--project":
			project = args[i+1]
			i++
		case "--persona":
			persona = args[i+1]
			i++
		}
	}
	return project, persona
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
