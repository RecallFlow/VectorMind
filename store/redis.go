package store

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

// CreateRedisClient creates a new Redis client
func CreateRedisClient(redisAddress, redisPassword string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: redisPassword,
		DB:       0, // use default DB
		Protocol: 2, // specify the Redis protocol version
	})

	return client
}

// CloseRedisClient closes the Redis client connection
func CloseRedisClient(client *redis.Client) error {
	return client.Close()
}

// IndexExists checks if a Redis search index exists
func IndexExists(ctx context.Context, redisClient *redis.Client, indexName string) (bool, error) {
	_, err := redisClient.FTInfo(ctx, indexName).Result()
	if err != nil {
		// Check if error indicates index doesn't exist
		errMsg := err.Error()
		if errMsg == "Unknown index name" ||
			redis.HasErrorPrefix(err, "vectormind_index: no such index") ||
			redis.HasErrorPrefix(err, indexName+": no such index") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateEmbeddingIndex creates a new Redis search index for embeddings
func CreateEmbeddingIndex(ctx context.Context, redisClient *redis.Client, indexName string, embeddingDimension int) error {
	_, err := redisClient.FTCreate(ctx,
		indexName,
		&redis.FTCreateOptions{
			OnHash: true,
			Prefix: []any{"doc:"},
		},
		&redis.FieldSchema{
			FieldName: "content",
			FieldType: redis.SearchFieldTypeText,
		},
		&redis.FieldSchema{
			FieldName: "label",
			FieldType: redis.SearchFieldTypeTag,
		},
		&redis.FieldSchema{
			FieldName: "metadata",
			FieldType: redis.SearchFieldTypeText,
		},
		&redis.FieldSchema{
			FieldName: "created_at",
			FieldType: redis.SearchFieldTypeNumeric,
		},
		&redis.FieldSchema{
			FieldName: "embedding",
			FieldType: redis.SearchFieldTypeVector,
			VectorArgs: &redis.FTVectorArgs{
				HNSWOptions: &redis.FTHNSWOptions{
					Dim:            embeddingDimension,
					DistanceMetric: "L2",
					Type:           "FLOAT32",
				},
			},
		},
	).Result()

	return err
}

// DropIndex drops a Redis search index
func DropIndex(ctx context.Context, redisClient *redis.Client, indexName string) *redis.StatusCmd {
	return redisClient.FTDropIndexWithArgs(ctx,
		indexName,
		&redis.FTDropIndexOptions{
			DeleteDocs: true,
		},
	)
}

// SimilaritySearch performs a vector similarity search
func SimilaritySearch(ctx context.Context, redisClient *redis.Client, indexName string, queryVector []float32, numberOfTopSimilarities int) ([]redis.Document, error) {
	buffer := floatsToBytes(queryVector) // embedding vector as byte array

	query := fmt.Sprintf("*=>[KNN %d @embedding $vec AS vector_distance]", numberOfTopSimilarities)

	results, err := redisClient.FTSearchWithArgs(ctx,
		indexName,
		query,
		&redis.FTSearchOptions{
			Return: []redis.FTSearchReturn{
				{FieldName: "vector_distance"},
				{FieldName: "content"},
				{FieldName: "label"},
				{FieldName: "metadata"},
				{FieldName: "created_at"},
			},
			DialectVersion: 2,
			Params: map[string]any{
				"vec": buffer,
			},
		},
	).Result()
	if err != nil {
		return nil, err
	}

	return results.Docs, nil
}

// SimilaritySearchWithLabel performs a vector similarity search filtered by label
func SimilaritySearchWithLabel(ctx context.Context, redisClient *redis.Client, indexName string, queryVector []float32, numberOfTopSimilarities int, label string) ([]redis.Document, error) {
	buffer := floatsToBytes(queryVector) // embedding vector as byte array

	query := fmt.Sprintf("@label:{%s}=>[KNN %d @embedding $vec AS vector_distance]", label, numberOfTopSimilarities)

	results, err := redisClient.FTSearchWithArgs(ctx,
		indexName,
		query,
		&redis.FTSearchOptions{
			Return: []redis.FTSearchReturn{
				{FieldName: "vector_distance"},
				{FieldName: "content"},
				{FieldName: "label"},
				{FieldName: "metadata"},
				{FieldName: "created_at"},
			},
			DialectVersion: 2,
			Params: map[string]any{
				"vec": buffer,
			},
		},
	).Result()
	if err != nil {
		return nil, err
	}

	return results.Docs, nil
}

// StoreEmbedding stores an embedding in Redis
func StoreEmbedding(ctx context.Context, redisClient *redis.Client, docID string, content string, embedding []float32, label string, metadata string) error {
	buffer := floatsToBytes(embedding) // embedding vector as byte array
	_, err := redisClient.HSet(ctx,
		docID,
		map[string]any{
			"content":    content,
			"label":      label,
			"metadata":   metadata,
			"created_at": time.Now().Unix(),
			"embedding":  buffer,
		},
	).Result()

	return err
}

// floatsToBytes converts a slice of float32 to bytes
func floatsToBytes(fs []float32) []byte {
	buf := make([]byte, len(fs)*4)

	for i, f := range fs {
		u := math.Float32bits(f)
		binary.NativeEndian.PutUint32(buf[i*4:], u)
	}

	return buf
}
