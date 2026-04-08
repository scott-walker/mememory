package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/scott-walker/mememory/internal/embeddings"
	mcptools "github.com/scott-walker/mememory/internal/mcp"
	"github.com/scott-walker/mememory/internal/engine"
	pg "github.com/scott-walker/mememory/internal/postgres"
)

const (
	defaultOllamaURL = "http://localhost:11434"
	ttlCleanInterval = 1 * time.Hour
)

func main() {
	logger := log.New(os.Stderr, "[mememory-server] ", log.LstdFlags)

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Fatal("DATABASE_URL is required. Set it in your .env or run `mememory setup` to bootstrap with the bundled Docker stack.")
	}

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

	svc := engine.NewService(pgClient, embedder)

	srv := server.NewMCPServer(
		"mememory",
		"0.3.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
		server.WithInstructions("Persistent semantic memory for AI agents. "+
			"Tools: remember (store), recall (search), forget (delete), update (re-embed), list (browse), stats (counts), help (docs). "+
			"Supports two scopes (global, project) and memory types (fact, rule, decision, feedback, context, bootstrap)."),
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

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
