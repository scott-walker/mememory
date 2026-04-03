package embeddings

import "fmt"

// Config holds embedding provider configuration from environment variables.
type Config struct {
	Provider string // "ollama" (default), "openai"
	URL      string // Provider-specific URL
	APIKey   string // API key (required for cloud providers)
	Model    string // Model name (optional, provider has defaults)
}

// New creates an Embedder from the given configuration.
func New(cfg Config) (Embedder, error) {
	switch cfg.Provider {
	case "", "ollama":
		url := cfg.URL
		if url == "" {
			url = "http://localhost:11434"
		}
		return NewOllamaClient(url), nil

	case "openai":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("EMBEDDING_API_KEY is required for openai provider")
		}
		var opts []OpenAIOption
		if cfg.URL != "" {
			opts = append(opts, WithOpenAIURL(cfg.URL))
		}
		if cfg.Model != "" {
			opts = append(opts, WithOpenAIModel(cfg.Model))
		}
		return NewOpenAIClient(cfg.APIKey, opts...), nil

	default:
		return nil, fmt.Errorf("unknown embedding provider: %q (supported: ollama, openai)", cfg.Provider)
	}
}
