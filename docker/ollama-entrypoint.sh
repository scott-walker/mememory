#!/bin/sh
set -e

# Start Ollama server in background
ollama serve &

# Wait for it to be ready
until ollama list > /dev/null 2>&1; do
  sleep 1
done

# Pull the embedding model (idempotent — no-op if already present)
ollama pull nomic-embed-text

# Keep the server running in foreground
wait
