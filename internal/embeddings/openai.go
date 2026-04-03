package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	defaultOpenAIURL   = "https://api.openai.com/v1/embeddings"
	defaultOpenAIModel = "text-embedding-3-small"
)

// OpenAIClient implements Embedder using the OpenAI embeddings API.
// Compatible with any OpenAI-compatible endpoint (OpenAI, Azure, Mistral, etc).
type OpenAIClient struct {
	url        string
	apiKey     string
	model      string
	httpClient *http.Client
}

type OpenAIOption func(*OpenAIClient)

func WithOpenAIURL(url string) OpenAIOption {
	return func(c *OpenAIClient) { c.url = url }
}

func WithOpenAIModel(model string) OpenAIOption {
	return func(c *OpenAIClient) { c.model = model }
}

func NewOpenAIClient(apiKey string, opts ...OpenAIOption) *OpenAIClient {
	c := &OpenAIClient{
		url:        defaultOpenAIURL,
		apiKey:     apiKey,
		model:      defaultOpenAIModel,
		httpClient: &http.Client{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type openaiRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type openaiResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *OpenAIClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	body, err := json.Marshal(openaiRequest{
		Model: c.model,
		Input: texts,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("API error: %s", result.Error.Message)
	}

	if len(result.Data) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(result.Data))
	}

	vectors := make([][]float32, len(texts))
	for _, d := range result.Data {
		vectors[d.Index] = d.Embedding
	}

	return vectors, nil
}

func (c *OpenAIClient) EmbedOne(ctx context.Context, text string) ([]float32, error) {
	vectors, err := c.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return vectors[0], nil
}
