# Embedding Providers

mememory converts text into vector embeddings for semantic search. The embedding provider determines which model generates these vectors. You can use a local model via Ollama (default, private) or a cloud API like OpenAI.

## Supported Providers

| Provider | Privacy | Cost | Quality | Default Model | Dimensions |
|----------|---------|------|---------|---------------|------------|
| Ollama (local) | Full — no data leaves your machine | Free | Good | nomic-embed-text | 768 |
| OpenAI | Data sent to OpenAI API | Paid per token | Excellent | text-embedding-3-small | 1536 |
| OpenAI-compatible | Varies | Varies | Varies | Configurable | Varies |

## Ollama (Default)

Ollama runs locally inside Docker. No API keys, no data leaves your machine.

### Configuration

```bash
# These are the defaults — no env vars needed for standard setup
EMBEDDING_PROVIDER=ollama
OLLAMA_URL=http://localhost:11434
```

The `nomic-embed-text` model is automatically pulled when the Docker stack starts. It produces 768-dimensional vectors.

### Using a different Ollama model

```bash
# Pull a different model first
docker exec mememory-ollama ollama pull mxbai-embed-large

# Then configure
EMBEDDING_PROVIDER=ollama
EMBEDDING_MODEL=mxbai-embed-large
```

::: warning
Changing the embedding model changes the vector dimensions. You cannot mix vectors of different dimensions in the same database. See [Switching Providers](#switching-providers) below.
:::

## OpenAI

Uses the OpenAI embeddings API. Requires an API key and internet connectivity.

### Configuration

```bash
EMBEDDING_PROVIDER=openai
EMBEDDING_API_KEY=sk-your-api-key-here
```

The default model is `text-embedding-3-small` (1536 dimensions). To use a different model:

```bash
EMBEDDING_PROVIDER=openai
EMBEDDING_API_KEY=sk-your-api-key-here
EMBEDDING_MODEL=text-embedding-3-large
```

| Model | Dimensions | Cost (per 1M tokens) |
|-------|------------|---------------------|
| text-embedding-3-small | 1536 | $0.02 |
| text-embedding-3-large | 3072 | $0.13 |
| text-embedding-ada-002 | 1536 | $0.10 |

## OpenAI-Compatible APIs

Any API that implements the OpenAI embeddings endpoint format works with the `openai` provider. This includes Azure OpenAI, Mistral, Together AI, and others.

### Azure OpenAI

```bash
EMBEDDING_PROVIDER=openai
EMBEDDING_URL=https://YOUR-RESOURCE.openai.azure.com/openai/deployments/YOUR-DEPLOYMENT/embeddings?api-version=2024-02-01
EMBEDDING_API_KEY=your-azure-api-key
EMBEDDING_MODEL=text-embedding-3-small
```

### Mistral

```bash
EMBEDDING_PROVIDER=openai
EMBEDDING_URL=https://api.mistral.ai/v1/embeddings
EMBEDDING_API_KEY=your-mistral-api-key
EMBEDDING_MODEL=mistral-embed
```

### Together AI

```bash
EMBEDDING_PROVIDER=openai
EMBEDDING_URL=https://api.together.xyz/v1/embeddings
EMBEDDING_API_KEY=your-together-api-key
EMBEDDING_MODEL=togethercomputer/m2-bert-80M-8k-retrieval
```

### Local LLM servers (LM Studio, vLLM, etc.)

```bash
EMBEDDING_PROVIDER=openai
EMBEDDING_URL=http://localhost:1234/v1/embeddings
EMBEDDING_API_KEY=not-needed
EMBEDDING_MODEL=nomic-embed-text
```

## Dimension Auto-Detection

mememory automatically detects the vector dimension of the configured embedding provider at startup. It sends a test string ("dimension probe") and measures the returned vector length.

On first run, the database is created with the detected dimension. On subsequent runs, the detected dimension is compared against the existing database column:

- **Match** — normal startup
- **Mismatch** — startup fails with an error explaining the conflict

This prevents silent data corruption from mixing embeddings of different sizes.

## Switching Providers

Because different models produce vectors of different dimensions, you cannot simply change the provider and keep existing data. The vectors would be incompatible.

### Migration process

1. **Export** your memories:

```bash
curl http://localhost:4200/api/memories/export -X POST -o backup.json
```

2. **Stop** the Docker stack:

```bash
docker compose -f docker/docker-compose.yml -p mememory down
```

3. **Reset** the database (deletes all data including vectors):

```bash
# Remove PostgreSQL data
rm -rf ~/.mememory/postgres
```

4. **Update** environment variables with the new provider configuration.

5. **Start** the stack:

```bash
docker compose -f docker/docker-compose.yml -p mememory up -d
```

6. **Import** the backup. This re-embeds all memories with the new provider:

```bash
curl http://localhost:4200/api/memories/import \
  -X POST \
  -H "Content-Type: application/json" \
  -d @backup.json
```

::: danger
The import re-embeds every memory through the new provider. For OpenAI, this costs tokens. For Ollama, this takes time proportional to the number of memories.
:::

## Implementing a Custom Provider

mememory uses a simple `Embedder` interface:

```go
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    EmbedOne(ctx context.Context, text string) ([]float32, error)
}
```

To add a new provider, implement this interface and register it in the factory function. See the [Contributing guide](/guide/contributing#adding-an-embedding-provider) for details.
