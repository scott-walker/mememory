package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

// EnsureEmbeddingDimension verifies that the embedding column matches the expected
// vector dimension. On first run (no column), it creates the column. If the table
// has data with a different dimension, it returns an error.
func (c *Client) EnsureEmbeddingDimension(ctx context.Context, dim int) error {
	currentDim, exists, err := c.getEmbeddingDimension(ctx)
	if err != nil {
		return fmt.Errorf("check embedding dimension: %w", err)
	}

	if !exists {
		// Table exists but no embedding column — add it
		return c.addEmbeddingColumn(ctx, dim)
	}

	if currentDim == dim {
		return nil
	}

	// Dimension mismatch — check if table has data
	var count int
	if err := c.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM memories").Scan(&count); err != nil {
		return fmt.Errorf("count memories: %w", err)
	}

	if count == 0 {
		// Empty table — safe to recreate column
		if _, err := c.db.ExecContext(ctx, "DROP INDEX IF EXISTS idx_memories_embedding"); err != nil {
			return fmt.Errorf("drop embedding index: %w", err)
		}
		if _, err := c.db.ExecContext(ctx, "ALTER TABLE memories DROP COLUMN embedding"); err != nil {
			return fmt.Errorf("drop embedding column: %w", err)
		}
		return c.addEmbeddingColumn(ctx, dim)
	}

	return fmt.Errorf(
		"embedding dimension mismatch: database has %d, embedder produces %d. "+
			"To switch models, export memories (mememory export), reset the database, "+
			"and re-import (mememory import) with the new model",
		currentDim, dim,
	)
}

func (c *Client) getEmbeddingDimension(ctx context.Context) (dim int, exists bool, err error) {
	// pg_attribute.atttypmod for vector(N) stores N
	err = c.db.QueryRowContext(ctx, `
		SELECT atttypmod
		FROM pg_attribute
		WHERE attrelid = 'memories'::regclass
		  AND attname = 'embedding'
		  AND NOT attisdropped`,
	).Scan(&dim)

	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return dim, true, nil
}

func (c *Client) addEmbeddingColumn(ctx context.Context, dim int) error {
	_, err := c.db.ExecContext(ctx, fmt.Sprintf(
		"ALTER TABLE memories ADD COLUMN embedding vector(%d) NOT NULL DEFAULT '[%s]'",
		dim, zeroVector(dim),
	))
	if err != nil {
		return fmt.Errorf("add embedding column: %w", err)
	}

	// Remove the default after adding (it was only needed for NOT NULL on existing rows)
	if _, err := c.db.ExecContext(ctx, "ALTER TABLE memories ALTER COLUMN embedding DROP DEFAULT"); err != nil {
		return fmt.Errorf("drop embedding default: %w", err)
	}

	_, err = c.db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_memories_embedding ON memories USING hnsw (embedding vector_cosine_ops)")
	if err != nil {
		return fmt.Errorf("create embedding index: %w", err)
	}

	return nil
}

// zeroVector returns "0,0,0,...,0" for N dimensions.
func zeroVector(dim int) string {
	if dim == 0 {
		return ""
	}
	b := make([]byte, 0, dim*2)
	for i := 0; i < dim; i++ {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, '0')
	}
	return string(b)
}
