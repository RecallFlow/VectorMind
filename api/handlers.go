package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"
	"vectormind/models"
	"vectormind/store"

	"github.com/google/uuid"
	"github.com/openai/openai-go"
	"github.com/redis/go-redis/v9"
)

var embeddingDimension int
var embeddingModelId string

func SetEmbeddingDimension(dim int) {
	embeddingDimension = dim
}

func GetEmbeddingDimension() int {
	return embeddingDimension
}

func SetEmbeddingModelId(modelId string) {
	embeddingModelId = modelId
}

func GetEmbeddingModelId() string {
	return embeddingModelId
}

// GetEmbeddingModelInfoHandler handles requests for embedding model information
func GetEmbeddingModelInfoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Only accept GET requests
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Method not allowed. Use GET",
		})
		return
	}

	response := map[string]interface{}{
		"success":   true,
		"model_id":  embeddingModelId,
		"dimension": embeddingDimension,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HealthCheckHandler handles health check requests
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status": "healthy",
		"server": "mcp-vectormind-server",
	}
	json.NewEncoder(w).Encode(response)
}

// CreateEmbeddingHandler handles embedding creation requests
func CreateEmbeddingHandler(w http.ResponseWriter, r *http.Request, ctx context.Context, openaiClient *openai.Client, redisClient *redis.Client, embeddingModelId, indexName string) {
	w.Header().Set("Content-Type", "application/json")

	// Only accept POST requests
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.CreateEmbeddingResponse{
			Success: false,
			Error:   "Method not allowed. Use POST",
		})
		return
	}

	// Parse request body
	var req models.CreateEmbeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.CreateEmbeddingResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	// Validate required fields
	if req.Content == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.CreateEmbeddingResponse{
			Success: false,
			Error:   "Content is required",
		})
		return
	}

	// Create embedding from text
	embedding, err := store.CreateEmbeddingFromText(ctx, *openaiClient, req.Content, embeddingModelId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.CreateEmbeddingResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create embedding: %v", err),
		})
		return
	}

	// Generate unique document ID
	docID := fmt.Sprintf("doc:%s", uuid.New().String())

	// Store embedding in Redis
	err = store.StoreEmbedding(ctx, redisClient, docID, req.Content, embedding, req.Label, req.Metadata)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.CreateEmbeddingResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to store embedding: %v", err),
		})
		return
	}

	// Success response
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.CreateEmbeddingResponse{
		ID:        docID,
		Content:   req.Content,
		Label:     req.Label,
		Metadata:  req.Metadata,
		CreatedAt: time.Now(),
		Success:   true,
	})
}

// SimilaritySearchHandler handles similarity search requests
func SimilaritySearchHandler(w http.ResponseWriter, r *http.Request, ctx context.Context, openaiClient *openai.Client, redisClient *redis.Client, embeddingModelId, indexName string) {
	w.Header().Set("Content-Type", "application/json")

	// Only accept POST requests
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.SimilaritySearchResponse{
			Success: false,
			Error:   "Method not allowed. Use POST",
		})
		return
	}

	// Parse request body
	var req models.SimilaritySearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.SimilaritySearchResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	// Validate required fields
	if req.Text == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.SimilaritySearchResponse{
			Success: false,
			Error:   "Text is required",
		})
		return
	}

	if req.MaxCount <= 0 {
		req.MaxCount = 5 // Default value
	}

	// Create embedding from query text
	queryEmbedding, err := store.CreateEmbeddingFromText(ctx, *openaiClient, req.Text, embeddingModelId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.SimilaritySearchResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create embedding: %v", err),
		})
		return
	}

	// Perform similarity search
	docs, err := store.SimilaritySearch(ctx, redisClient, indexName, queryEmbedding, req.MaxCount)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.SimilaritySearchResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to perform similarity search: %v", err),
		})
		return
	}

	// Convert results to response format
	results := make([]models.SimilaritySearchResult, 0, len(docs))
	for _, doc := range docs {
		str := doc.Fields["vector_distance"]
		distance, err := strconv.ParseFloat(str, 32)
		if err != nil {
			distance = 9.9
		}

		// Filter by distance threshold if specified
		if req.DistanceThreshold != nil && distance > *req.DistanceThreshold {
			continue
		}

		createdAtUnix, _ := strconv.ParseInt(doc.Fields["created_at"], 10, 64)
		createdAt := time.Unix(createdAtUnix, 0).Format(time.RFC3339)

		result := models.SimilaritySearchResult{
			ID:        doc.ID,
			Content:   doc.Fields["content"],
			Label:     doc.Fields["label"],
			Metadata:  doc.Fields["metadata"],
			Distance:  distance,
			CreatedAt: createdAt,
		}

		results = append(results, result)
	}

	// Sort results by distance in ascending order (closest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	// Success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.SimilaritySearchResponse{
		Results: results,
		Success: true,
	})
}

// SimilaritySearchWithLabelHandler handles similarity search with label filter requests
func SimilaritySearchWithLabelHandler(w http.ResponseWriter, r *http.Request, ctx context.Context, openaiClient *openai.Client, redisClient *redis.Client, embeddingModelId, indexName string) {
	w.Header().Set("Content-Type", "application/json")

	// Only accept POST requests
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.SimilaritySearchResponse{
			Success: false,
			Error:   "Method not allowed. Use POST",
		})
		return
	}

	// Parse request body
	var req models.SimilaritySearchWithLabelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.SimilaritySearchResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	// Validate required fields
	if req.Text == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.SimilaritySearchResponse{
			Success: false,
			Error:   "Text is required",
		})
		return
	}

	if req.Label == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.SimilaritySearchResponse{
			Success: false,
			Error:   "Label is required",
		})
		return
	}

	if req.MaxCount <= 0 {
		req.MaxCount = 5 // Default value
	}

	// Create embedding from query text
	queryEmbedding, err := store.CreateEmbeddingFromText(ctx, *openaiClient, req.Text, embeddingModelId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.SimilaritySearchResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create embedding: %v", err),
		})
		return
	}

	// Perform similarity search with label filter
	docs, err := store.SimilaritySearchWithLabel(ctx, redisClient, indexName, queryEmbedding, req.MaxCount, req.Label)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.SimilaritySearchResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to perform similarity search: %v", err),
		})
		return
	}

	// Convert results to response format
	results := make([]models.SimilaritySearchResult, 0, len(docs))
	for _, doc := range docs {
		str := doc.Fields["vector_distance"]
		distance, err := strconv.ParseFloat(str, 32)
		if err != nil {
			distance = 9.9
		}

		// Filter by distance threshold if specified
		if req.DistanceThreshold != nil && distance > *req.DistanceThreshold {
			continue
		}

		createdAtUnix, _ := strconv.ParseInt(doc.Fields["created_at"], 10, 64)
		createdAt := time.Unix(createdAtUnix, 0).Format(time.RFC3339)

		result := models.SimilaritySearchResult{
			ID:        doc.ID,
			Content:   doc.Fields["content"],
			Label:     doc.Fields["label"],
			Metadata:  doc.Fields["metadata"],
			Distance:  distance,
			CreatedAt: createdAt,
		}

		results = append(results, result)
	}

	// Sort results by distance in ascending order (closest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	// Success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.SimilaritySearchResponse{
		Results: results,
		Success: true,
	})
}
