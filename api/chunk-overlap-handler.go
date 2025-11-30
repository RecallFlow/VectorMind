package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"vectormind/models"
	"vectormind/splitter"
	"vectormind/store"

	"github.com/google/uuid"
	"github.com/openai/openai-go"
	"github.com/redis/go-redis/v9"
)

// ChunkAndStoreHandler handles requests to chunk a document and store all chunks
func ChunkAndStoreHandler(w http.ResponseWriter, r *http.Request, ctx context.Context, openaiClient *openai.Client, redisClient *redis.Client, embeddingModelId, indexName string) {
	w.Header().Set("Content-Type", "application/json")

	// Only accept POST requests
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.ChunkAndStoreResponse{
			Success: false,
			Error:   "Method not allowed. Use POST",
		})
		return
	}

	// Parse request body
	var req models.ChunkAndStoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ChunkAndStoreResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	// Validate required fields
	if req.Document == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ChunkAndStoreResponse{
			Success: false,
			Error:   "Document is required",
		})
		return
	}

	if req.ChunkSize <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ChunkAndStoreResponse{
			Success: false,
			Error:   "ChunkSize must be greater than 0",
		})
		return
	}

	if req.Overlap < 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ChunkAndStoreResponse{
			Success: false,
			Error:   "Overlap cannot be negative",
		})
		return
	}

	if req.Overlap >= req.ChunkSize {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ChunkAndStoreResponse{
			Success: false,
			Error:   "Overlap must be less than ChunkSize",
		})
		return
	}

	// Validate chunk_size <= embeddingDimension
	embeddingDim := GetEmbeddingDimension()
	if req.ChunkSize > embeddingDim {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ChunkAndStoreResponse{
			Success: false,
			Error:   fmt.Sprintf("ChunkSize (%d) must be less than or equal to embedding dimension (%d)", req.ChunkSize, embeddingDim),
		})
		return
	}

	// Chunk the document
	chunks := splitter.ChunkText(req.Document, req.ChunkSize, req.Overlap)

	if len(chunks) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ChunkAndStoreResponse{
			Success: false,
			Error:   "No chunks generated from the document",
		})
		return
	}

	// Store all chunks
	chunkIDs := make([]string, 0, len(chunks))
	createdAt := time.Now()

	for _, chunk := range chunks {
		// Create embedding from chunk text
		embedding, err := store.CreateEmbeddingFromText(ctx, *openaiClient, chunk, embeddingModelId)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ChunkAndStoreResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to create embedding for chunk: %v", err),
			})
			return
		}

		// Generate unique document ID for this chunk
		chunkID := fmt.Sprintf("doc:%s", uuid.New().String())

		// Store embedding in Redis with the same label and metadata for all chunks
		err = store.StoreEmbedding(ctx, redisClient, chunkID, chunk, embedding, req.Label, req.Metadata)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ChunkAndStoreResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to store chunk embedding: %v", err),
			})
			return
		}

		chunkIDs = append(chunkIDs, chunkID)
	}

	// Success response
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.ChunkAndStoreResponse{
		ChunkIDs:     chunkIDs,
		ChunksStored: len(chunkIDs),
		CreatedAt:    createdAt,
		Success:      true,
	})
}
