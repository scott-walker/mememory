#!/usr/bin/env bash
set -euo pipefail

QDRANT_URL="${QDRANT_URL:-http://localhost:6333}"
OLLAMA_URL="${OLLAMA_URL:-http://localhost:11434}"
COLLECTION="memory"
VECTOR_SIZE=768

echo "Waiting for Qdrant..."
for i in $(seq 1 30); do
  if curl -sf "${QDRANT_URL}/healthz" > /dev/null 2>&1; then
    echo "Qdrant is ready."
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo "ERROR: Qdrant not ready after 30s"
    exit 1
  fi
  sleep 1
done

echo "Waiting for Ollama..."
for i in $(seq 1 30); do
  if curl -sf "${OLLAMA_URL}/api/tags" > /dev/null 2>&1; then
    echo "Ollama is ready."
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo "ERROR: Ollama not ready after 30s"
    exit 1
  fi
  sleep 1
done

# Create collection if not exists
EXISTS=$(curl -sf "${QDRANT_URL}/collections/${COLLECTION}" | grep -c '"status":"ok"' || true)
if [ "$EXISTS" -eq 0 ]; then
  echo "Creating collection '${COLLECTION}' (${VECTOR_SIZE}d, cosine)..."
  curl -sf -X PUT "${QDRANT_URL}/collections/${COLLECTION}" \
    -H "Content-Type: application/json" \
    -d "{
      \"vectors\": {
        \"size\": ${VECTOR_SIZE},
        \"distance\": \"Cosine\"
      }
    }" | head -c 200
  echo
  echo "Collection created."
else
  echo "Collection '${COLLECTION}' already exists."
fi

# Create payload indexes for filtering
echo "Creating payload indexes..."
for FIELD in scope project persona type; do
  curl -sf -X PUT "${QDRANT_URL}/collections/${COLLECTION}/index" \
    -H "Content-Type: application/json" \
    -d "{
      \"field_name\": \"${FIELD}\",
      \"field_schema\": \"keyword\"
    }" > /dev/null 2>&1 || true
done

curl -sf -X PUT "${QDRANT_URL}/collections/${COLLECTION}/index" \
  -H "Content-Type: application/json" \
  -d '{
    "field_name": "ttl",
    "field_schema": "datetime"
  }' > /dev/null 2>&1 || true

echo "Payload indexes created."

# Pull embedding model
echo "Pulling nomic-embed-text model..."
curl -sf -X POST "${OLLAMA_URL}/api/pull" \
  -H "Content-Type: application/json" \
  -d '{"name": "nomic-embed-text"}' | while IFS= read -r line; do
  STATUS=$(echo "$line" | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4)
  if [ -n "$STATUS" ]; then
    printf "\r  %s" "$STATUS"
  fi
done
echo
echo "Model ready."

echo "Setup complete."
