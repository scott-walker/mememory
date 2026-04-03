package embeddings

import "context"

// Embedder converts text into vector embeddings.
// Implementations: Ollama (local), OpenAI, Voyage AI, etc.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	EmbedOne(ctx context.Context, text string) ([]float32, error)
}
