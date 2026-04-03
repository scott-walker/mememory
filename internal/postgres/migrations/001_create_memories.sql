CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS memories (
    id          UUID PRIMARY KEY,
    content     TEXT NOT NULL,
    embedding   vector(768) NOT NULL,
    scope       TEXT NOT NULL DEFAULT 'global',
    project     TEXT,
    persona     TEXT,
    type        TEXT NOT NULL DEFAULT 'fact',
    tags        TEXT[],
    weight      DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    supersedes  UUID REFERENCES memories(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ttl         TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_memories_scope ON memories(scope);
CREATE INDEX IF NOT EXISTS idx_memories_project ON memories(project) WHERE project IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_memories_persona ON memories(persona) WHERE persona IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(type);
CREATE INDEX IF NOT EXISTS idx_memories_ttl ON memories(ttl) WHERE ttl IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_memories_embedding ON memories USING hnsw (embedding vector_cosine_ops);
