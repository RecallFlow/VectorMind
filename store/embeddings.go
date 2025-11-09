package store

import (
	"context"

	"github.com/openai/openai-go"
)

// CreateEmbeddingFromText creates an embedding vector from text using OpenAI API
func CreateEmbeddingFromText(ctx context.Context, openaiClient openai.Client, text, embeddingModelId string) ([]float32, error) {
	embeddingsResponse, err := openaiClient.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(text),
		},
		Model: embeddingModelId,
	})

	// convert the embedding to a []float32
	embedding := make([]float32, len(embeddingsResponse.Data[0].Embedding))
	for i, f := range embeddingsResponse.Data[0].Embedding {
		embedding[i] = float32(f)
	}

	return embedding, err
}
