package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/scott-walker/mememory/internal/api"
	"github.com/scott-walker/mememory/internal/embeddings"
	"github.com/scott-walker/mememory/internal/memory"
	pg "github.com/scott-walker/mememory/internal/postgres"
)

const (
	defaultPort        = 4200
	defaultDatabaseURL = "postgres://memory:memory@localhost:5432/memory?sslmode=disable"
	defaultOllamaURL   = "http://localhost:11434"
	defaultStaticDir   = "web/dist"
)

func main() {
	logger := log.New(os.Stderr, "[memory-admin] ", log.LstdFlags)

	port := envIntOrDefault("ADMIN_PORT", defaultPort)
	staticDir := envOrDefault("STATIC_DIR", defaultStaticDir)
	databaseURL := envOrDefault("DATABASE_URL", defaultDatabaseURL)
	ollamaURL := envOrDefault("OLLAMA_URL", defaultOllamaURL)

	logger.Println("Connecting to PostgreSQL")
	pgClient, err := pg.NewClient(databaseURL)
	if err != nil {
		logger.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgClient.Close()

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
	if embeddingCfg.Provider == "" || embeddingCfg.Provider == "ollama" {
		if embeddingCfg.URL == "" {
			embeddingCfg.URL = ollamaURL
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

	router := api.NewRouter(svc)

	mux := http.NewServeMux()

	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		logger.Printf("Serving static files from %s", staticDir)
		staticFS := http.Dir(staticDir)
		fileServer := http.FileServer(staticFS)

		mux.Handle("/api/", router)
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if _, err := fs.Stat(os.DirFS(staticDir), r.URL.Path[1:]); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
			http.ServeFile(w, r, staticDir+"/index.html")
		})
	} else {
		logger.Println("No static files found, API-only mode")
		mux.Handle("/", router)
	}

	addr := fmt.Sprintf(":%d", port)
	logger.Printf("Starting admin server on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Fatalf("Server error: %v", err)
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOrDefault(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
