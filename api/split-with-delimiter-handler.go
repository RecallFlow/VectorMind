package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	"vectormind/models"
	"vectormind/splitter"
	"vectormind/store"

	"github.com/google/uuid"
	"github.com/openai/openai-go"
	"github.com/redis/go-redis/v9"
)

// SplitAndStoreWithDelimiterHandler handles requests to split text by delimiter and store all chunks
func SplitAndStoreWithDelimiterHandler(w http.ResponseWriter, r *http.Request, ctx context.Context, openaiClient *openai.Client, redisClient *redis.Client, embeddingModelId, indexName string) {
	w.Header().Set("Content-Type", "application/json")

	// Only accept POST requests
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.SplitAndStoreWithDelimiterResponse{
			Success: false,
			Error:   "Method not allowed. Use POST",
		})
		return
	}

	// Parse request body
	var req models.SplitAndStoreWithDelimiterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.SplitAndStoreWithDelimiterResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	// Validate required fields
	if req.Document == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.SplitAndStoreWithDelimiterResponse{
			Success: false,
			Error:   "Document is required",
		})
		return
	}

	if req.Delimiter == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.SplitAndStoreWithDelimiterResponse{
			Success: false,
			Error:   "Delimiter is required",
		})
		return
	}

	// Split text by delimiter
	chunks := splitter.SplitTextWithDelimiter(req.Document, req.Delimiter)

	if len(chunks) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.SplitAndStoreWithDelimiterResponse{
			Success: false,
			Error:   "No chunks generated from the document",
		})
		return
	}

	// Get embedding dimension for validation
	embeddingDim := GetEmbeddingDimension()

	// Store all chunks (subdividing if necessary)
	chunkIDs := make([]string, 0)
	createdAt := time.Now()

	for _, chunk := range chunks {
		// Extract first 2 non-empty lines from the chunk
		chunkHeader := splitter.ExtractFirstNonEmptyLines(chunk, 2)

		// If chunk is larger than embedding dimension, subdivide it
		var chunksToStore []string
		if len(chunk) > embeddingDim {
			// Subdivide the chunk into smaller pieces without overlap
			chunksToStore = splitter.ChunkText(chunk, embeddingDim, 0)
			log.Println("🟠 Chunk exceeded embedding dimension, subdivided into", len(chunksToStore), "chunks")

			// If we have a header and subdivided chunks, prepend the header to each sub-chunk
			// (except the first one which already contains it)
			if chunkHeader != "" && len(chunksToStore) > 1 {
				for i := 1; i < len(chunksToStore); i++ {
					chunksToStore[i] = chunkHeader + "\n\n" + chunksToStore[i]
					log.Println("🟡", i+1, "Prepended header to sub-chunk")
				}
			}

		} else {
			chunksToStore = []string{chunk}
		}

		// Store each chunk
		for _, chunkToStore := range chunksToStore {
			// Create embedding from chunk text
			embedding, err := store.CreateEmbeddingFromText(ctx, *openaiClient, chunkToStore, embeddingModelId)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(models.SplitAndStoreWithDelimiterResponse{
					Success: false,
					Error:   fmt.Sprintf("Failed to create embedding for chunk: %v", err),
				})
				return
			}

			// Generate unique document ID for this chunk
			chunkID := fmt.Sprintf("doc:%s", uuid.New().String())

			// Store embedding in Redis with the same label and metadata for all chunks
			err = store.StoreEmbedding(ctx, redisClient, chunkID, chunkToStore, embedding, req.Label, req.Metadata)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(models.SplitAndStoreWithDelimiterResponse{
					Success: false,
					Error:   fmt.Sprintf("Failed to store chunk embedding: %v", err),
				})
				return
			}

			chunkIDs = append(chunkIDs, chunkID)
		}
	}

	// Success response
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.SplitAndStoreWithDelimiterResponse{
		ChunkIDs:     chunkIDs,
		ChunksStored: len(chunkIDs),
		CreatedAt:    createdAt,
		Success:      true,
	})
}
