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

// SplitAndStoreMarkdownWithHierarchyHandler handles requests to split markdown with hierarchy and store all chunks
func SplitAndStoreMarkdownWithHierarchyHandler(w http.ResponseWriter, r *http.Request, ctx context.Context, openaiClient *openai.Client, redisClient *redis.Client, embeddingModelId, indexName string) {
	w.Header().Set("Content-Type", "application/json")

	// Only accept POST requests
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.SplitAndStoreMarkdownWithHierarchyResponse{
			Success: false,
			Error:   "Method not allowed. Use POST",
		})
		return
	}

	// Parse request body
	var req models.SplitAndStoreMarkdownWithHierarchyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.SplitAndStoreMarkdownWithHierarchyResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	// Validate required fields
	if req.Document == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.SplitAndStoreMarkdownWithHierarchyResponse{
			Success: false,
			Error:   "Document is required",
		})
		return
	}

	// Split markdown with hierarchy
	chunks := splitter.ChunkWithMarkdownHierarchy(req.Document)

	if len(chunks) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.SplitAndStoreMarkdownWithHierarchyResponse{
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
		// If chunk is larger than embedding dimension, subdivide it
		var chunksToStore []string
		if len(chunk) > embeddingDim {
			// Subdivide the chunk into smaller chunks without overlap
			chunksToStore = splitter.ChunkText(chunk, embeddingDim, 0)
			log.Println("🟠 Chunk exceeded embedding dimension, subdivided into", len(chunksToStore), "chunks")
		} else {
			chunksToStore = []string{chunk}
		}

		// Store each sub-chunk
		for _, subChunk := range chunksToStore {
			// Create embedding from chunk text
			embedding, err := store.CreateEmbeddingFromText(ctx, *openaiClient, subChunk, embeddingModelId)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(models.SplitAndStoreMarkdownWithHierarchyResponse{
					Success: false,
					Error:   fmt.Sprintf("Failed to create embedding for chunk: %v", err),
				})
				return
			}

			// Generate unique document ID for this chunk
			chunkID := fmt.Sprintf("doc:%s", uuid.New().String())

			// Store embedding in Redis with the same label and metadata for all chunks
			err = store.StoreEmbedding(ctx, redisClient, chunkID, subChunk, embedding, req.Label, req.Metadata)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(models.SplitAndStoreMarkdownWithHierarchyResponse{
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
	json.NewEncoder(w).Encode(models.SplitAndStoreMarkdownWithHierarchyResponse{
		ChunkIDs:     chunkIDs,
		ChunksStored: len(chunkIDs),
		CreatedAt:    createdAt,
		Success:      true,
	})
}
