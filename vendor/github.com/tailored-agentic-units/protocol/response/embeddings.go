package response

// EmbeddingsResponse holds the result of an embeddings request.
type EmbeddingsResponse struct {
	Embeddings [][]float64
	Model      string
	Usage      *TokenUsage
}
