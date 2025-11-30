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

// SplitAndStoreMarkdownSectionsHandler handles requests to split markdown by sections and store all chunks
func SplitAndStoreMarkdownSectionsHandler(w http.ResponseWriter, r *http.Request, ctx context.Context, openaiClient *openai.Client, redisClient *redis.Client, embeddingModelId, indexName string) {
	w.Header().Set("Content-Type", "application/json")

	// Only accept POST requests
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.SplitAndStoreMarkdownSectionsResponse{
			Success: false,
			Error:   "Method not allowed. Use POST",
		})
		return
	}

	// Parse request body
	var req models.SplitAndStoreMarkdownSectionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.SplitAndStoreMarkdownSectionsResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	// Validate required fields
	if req.Document == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.SplitAndStoreMarkdownSectionsResponse{
			Success: false,
			Error:   "Document is required",
		})
		return
	}

	// Split markdown by sections
	sections := splitter.SplitMarkdownBySections(req.Document)

	if len(sections) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.SplitAndStoreMarkdownSectionsResponse{
			Success: false,
			Error:   "No sections generated from the document",
		})
		return
	}

	// Get embedding dimension for validation
	embeddingDim := GetEmbeddingDimension()

	// Store all sections (subdividing if necessary)
	chunkIDs := make([]string, 0)
	createdAt := time.Now()

	for _, section := range sections {
		// Extract section header (if any)
		sectionHeader := splitter.ExtractSectionHeader(section)

		// If section is larger than embedding dimension, subdivide it
		var chunksToStore []string
		if len(section) > embeddingDim {
			// Subdivide the section into smaller chunks without overlap
			chunksToStore = splitter.ChunkText(section, embeddingDim, 0)
			log.Println("🟠 Section exceeded embedding dimension, subdivided into", len(chunksToStore), "chunks")

			// If we have a header and subdivided chunks, prepend the header to each sub-chunk
			// (except the first one which already contains it)
			if sectionHeader != "" && len(chunksToStore) > 1 {
				for i := 1; i < len(chunksToStore); i++ {
					chunksToStore[i] = sectionHeader + "\n\n" + chunksToStore[i]
					log.Println("🟡", i+1, chunksToStore[i])
				}
			}

		} else {
			chunksToStore = []string{section}
		}

		// Store each chunk
		for _, chunk := range chunksToStore {
			// Create embedding from chunk text
			embedding, err := store.CreateEmbeddingFromText(ctx, *openaiClient, chunk, embeddingModelId)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(models.SplitAndStoreMarkdownSectionsResponse{
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
				json.NewEncoder(w).Encode(models.SplitAndStoreMarkdownSectionsResponse{
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
	json.NewEncoder(w).Encode(models.SplitAndStoreMarkdownSectionsResponse{
		ChunkIDs:     chunkIDs,
		ChunksStored: len(chunkIDs),
		CreatedAt:    createdAt,
		Success:      true,
	})
}
