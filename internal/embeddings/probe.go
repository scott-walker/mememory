package embeddings

import "context"

// ProbeDimension sends a test string to the embedder and returns the vector dimension.
func ProbeDimension(ctx context.Context, e Embedder) (int, error) {
	vec, err := e.EmbedOne(ctx, "dimension probe")
	if err != nil {
		return 0, err
	}
	return len(vec), nil
}
