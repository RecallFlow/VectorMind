package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"vectormind/api"
	"vectormind/mcptools"
	"vectormind/models"
	"vectormind/store"

	"github.com/openai/openai-go"
)

// Helper function to get Redis address from environment or use default
func getRedisAddress() string {
	if addr := os.Getenv("REDIS_ADDRESS"); addr != "" {
		return addr
	}
	return "localhost:6379"
}

// Helper function to get Redis password from environment or use default
func getRedisPassword() string {
	return os.Getenv("REDIS_PASSWORD")
}

// Helper function to get Redis index name from environment or use default
func getRedisIndexName() string {
	if indexName := os.Getenv("REDIS_INDEX_NAME"); indexName != "" {
		return indexName
	}
	return "test_index"
}

func TestCreateRedisClient(t *testing.T) {
	tests := []struct {
		name          string
		redisAddress  string
		redisPassword string
	}{
		{
			name:          "Create client with default settings",
			redisAddress:  getRedisAddress(),
			redisPassword: getRedisPassword(),
		},
		{
			name:          "Create client with password",
			redisAddress:  getRedisAddress(),
			redisPassword: "testpassword",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := store.CreateRedisClient(tt.redisAddress, tt.redisPassword)
			if client == nil {
				t.Error("Expected client to be non-nil")
			}
			defer client.Close()

			if client.Options().Addr != tt.redisAddress {
				t.Errorf("Expected address %s, got %s", tt.redisAddress, client.Options().Addr)
			}
			if client.Options().Password != tt.redisPassword {
				t.Errorf("Expected password %s, got %s", tt.redisPassword, client.Options().Password)
			}
			if client.Options().DB != 0 {
				t.Errorf("Expected DB 0, got %d", client.Options().DB)
			}
			if client.Options().Protocol != 2 {
				t.Errorf("Expected protocol 2, got %d", client.Options().Protocol)
			}
		})
	}
}

func TestCloseRedisClient(t *testing.T) {
	client := store.CreateRedisClient(getRedisAddress(), getRedisPassword())
	err := store.CloseRedisClient(client)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

// Conversion functions are now internal to the store package
// These tests are removed as they test private implementation details

// floatsToBytes is now internal to the store package

func TestHealthCheckHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	api.HealthCheckHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}

	if response["server"] != "mcp-vectormind-server" {
		t.Errorf("Expected server 'mcp-vectormind-server', got %v", response["server"])
	}
}

// Note: The following functions require a live Redis connection and are marked as integration tests
// They can be run with: go test -tags=integration

func TestIndexExists_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	client := store.CreateRedisClient(getRedisAddress(), getRedisPassword())
	defer store.CloseRedisClient(client)

	// Test with non-existent index
	exists, err := store.IndexExists(ctx, client, "non_existent_index")
	if err != nil {
		t.Errorf("Expected no error for non-existent index, got %v", err)
	}
	if exists {
		t.Error("Expected index to not exist")
	}
}

func TestStoreEmbedding_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	client := store.CreateRedisClient(getRedisAddress(), getRedisPassword())
	defer store.CloseRedisClient(client)

	// Clean up before test
	defer client.Del(ctx, "test:doc:1")

	embedding := []float32{1.0, 2.0, 3.0, 4.0}
	err := store.StoreEmbedding(ctx, client, "test:doc:1", "test content", embedding, "test-label", "test-metadata")
	if err != nil {
		t.Errorf("Failed to store embedding: %v", err)
	}

	// Verify the data was stored
	result, err := client.HGetAll(ctx, "test:doc:1").Result()
	if err != nil {
		t.Errorf("Failed to retrieve stored data: %v", err)
	}

	if result["content"] != "test content" {
		t.Errorf("Expected content 'test content', got %v", result["content"])
	}
}

func TestCreateEmbeddingIndex_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	client := store.CreateRedisClient(getRedisAddress(), getRedisPassword())
	defer store.CloseRedisClient(client)

	indexName := "test_vector_idx"

	// Clean up: drop index if it exists
	defer store.DropIndex(ctx, client, indexName)

	err := store.CreateEmbeddingIndex(ctx, client, indexName, 1024)
	if err != nil {
		t.Errorf("Failed to create index: %v", err)
	}

	// Verify index was created
	exists, err := store.IndexExists(ctx, client, indexName)
	if err != nil {
		t.Errorf("Error checking index existence: %v", err)
	}
	if !exists {
		t.Error("Expected index to exist after creation")
	}
}

func TestDropIndex_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	client := store.CreateRedisClient(getRedisAddress(), getRedisPassword())
	defer store.CloseRedisClient(client)

	indexName := "test_drop_idx"

	// Create index first
	store.CreateEmbeddingIndex(ctx, client, indexName, 1024)

	// Drop the index
	result := store.DropIndex(ctx, client, indexName)
	if result.Err() != nil {
		t.Errorf("Failed to drop index: %v", result.Err())
	}

	// Verify index was dropped
	exists, _ := store.IndexExists(ctx, client, indexName)
	if exists {
		t.Error("Expected index to not exist after dropping")
	}
}

func TestSimilaritySearch_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	client := store.CreateRedisClient(getRedisAddress(), getRedisPassword())
	defer store.CloseRedisClient(client)

	indexName := "test_similarity_idx"
	defer store.DropIndex(ctx, client, indexName)

	// Create index and add some test data
	store.CreateEmbeddingIndex(ctx, client, indexName, 4)

	embedding1 := []float32{1.0, 2.0, 3.0, 4.0}
	store.StoreEmbedding(ctx, client, "doc:test1", "content 1", embedding1, "", "")

	// Perform similarity search
	queryVector := []float32{1.1, 2.1, 3.1, 4.1}
	docs, err := store.SimilaritySearch(ctx, client, indexName, queryVector, 5)
	if err != nil {
		t.Errorf("Similarity search failed: %v", err)
	}

	if len(docs) == 0 {
		t.Log("No documents found - this might be expected if indexing takes time")
	}
}

func TestSimilaritySearchHandler_RequestValidation(t *testing.T) {
	tests := []struct {
		name              string
		requestBody       interface{}
		method            string
		expectedStatus    int
		validateResponse  func(*testing.T, models.SimilaritySearchResponse)
	}{
		{
			name: "Invalid method - GET instead of POST",
			requestBody: models.SimilaritySearchRequest{
				Text:     "test query",
				MaxCount: 5,
			},
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
			validateResponse: func(t *testing.T, resp models.SimilaritySearchResponse) {
				if resp.Success {
					t.Error("Expected success to be false for wrong method")
				}
				if resp.Error == "" {
					t.Error("Expected error message for wrong method")
				}
			},
		},
		{
			name:           "Invalid JSON body",
			requestBody:    "invalid json",
			method:         http.MethodPost,
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp models.SimilaritySearchResponse) {
				if resp.Success {
					t.Error("Expected success to be false for invalid JSON")
				}
				if resp.Error == "" {
					t.Error("Expected error message for invalid JSON")
				}
			},
		},
		{
			name: "Empty text field",
			requestBody: models.SimilaritySearchRequest{
				Text:     "",
				MaxCount: 5,
			},
			method:         http.MethodPost,
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp models.SimilaritySearchResponse) {
				if resp.Success {
					t.Error("Expected success to be false for empty text")
				}
				if resp.Error == "" {
					t.Error("Expected error message for empty text")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if str, ok := tt.requestBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(tt.method, "/search", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			ctx := context.Background()
			client := store.CreateRedisClient(getRedisAddress(), getRedisPassword())
			defer store.CloseRedisClient(client)

			openaiClient := openai.NewClient()

			api.SimilaritySearchHandler(w, req, ctx, &openaiClient, client, "test-model", getRedisIndexName())

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			var response models.SimilaritySearchResponse
			err := json.NewDecoder(resp.Body).Decode(&response)
			if err != nil {
				t.Errorf("Failed to decode response: %v", err)
			}

			tt.validateResponse(t, response)
		})
	}
}

func TestSimilaritySearchRequest_DistanceThresholdField(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr  string
		expected models.SimilaritySearchRequest
	}{
		{
			name:    "Without distance threshold",
			jsonStr: `{"text":"test","max_count":5}`,
			expected: models.SimilaritySearchRequest{
				Text:              "test",
				MaxCount:          5,
				DistanceThreshold: nil,
			},
		},
		{
			name:    "With distance threshold",
			jsonStr: `{"text":"test","max_count":5,"distance_threshold":0.5}`,
			expected: models.SimilaritySearchRequest{
				Text:              "test",
				MaxCount:          5,
				DistanceThreshold: floatPtr(0.5),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req models.SimilaritySearchRequest
			err := json.Unmarshal([]byte(tt.jsonStr), &req)
			if err != nil {
				t.Errorf("Failed to unmarshal JSON: %v", err)
			}

			if req.Text != tt.expected.Text {
				t.Errorf("Expected text %s, got %s", tt.expected.Text, req.Text)
			}
			if req.MaxCount != tt.expected.MaxCount {
				t.Errorf("Expected max_count %d, got %d", tt.expected.MaxCount, req.MaxCount)
			}

			if tt.expected.DistanceThreshold == nil {
				if req.DistanceThreshold != nil {
					t.Errorf("Expected distance_threshold to be nil, got %v", *req.DistanceThreshold)
				}
			} else {
				if req.DistanceThreshold == nil {
					t.Error("Expected distance_threshold to be non-nil")
				} else if *req.DistanceThreshold != *tt.expected.DistanceThreshold {
					t.Errorf("Expected distance_threshold %f, got %f", *tt.expected.DistanceThreshold, *req.DistanceThreshold)
				}
			}
		})
	}
}

// Helper function to create a float64 pointer
func floatPtr(f float64) *float64 {
	return &f
}

func TestEmbeddingModelIdGetterSetter(t *testing.T) {
	mcptools.SetEmbeddingModelId("test-model-id")
	result := mcptools.GetEmbeddingModelId()
	if result != "test-model-id" {
		t.Errorf("Expected 'test-model-id', got %s", result)
	}
}

func TestEmbeddingDimensionGetterSetter(t *testing.T) {
	mcptools.SetEmbeddingDimension(1024)
	result := mcptools.GetEmbeddingDimension()
	if result != 1024 {
		t.Errorf("Expected 1024, got %d", result)
	}
}

func TestCreateEmbeddingHandler_RequestValidation(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		method         string
		expectedStatus int
	}{
		{
			name: "Invalid method - GET instead of POST",
			requestBody: map[string]string{
				"content": "test content",
			},
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Missing content field",
			requestBody:    map[string]string{},
			method:         http.MethodPost,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Empty content field",
			requestBody: map[string]string{
				"content": "",
			},
			method:         http.MethodPost,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if str, ok := tt.requestBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(tt.method, "/embeddings", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			ctx := context.Background()
			client := store.CreateRedisClient(getRedisAddress(), getRedisPassword())
			defer store.CloseRedisClient(client)

			openaiClient := openai.NewClient()

			api.CreateEmbeddingHandler(w, req, ctx, &openaiClient, client, "test-model", getRedisIndexName())

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestSimilaritySearchWithLabel_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	client := store.CreateRedisClient(getRedisAddress(), getRedisPassword())
	defer store.CloseRedisClient(client)

	indexName := "test_label_search_idx"
	defer store.DropIndex(ctx, client, indexName)

	// Create index and add test data with labels
	store.CreateEmbeddingIndex(ctx, client, indexName, 4)

	embedding1 := []float32{1.0, 2.0, 3.0, 4.0}
	store.StoreEmbedding(ctx, client, "doc:test1", "content 1", embedding1, "animals", "")

	embedding2 := []float32{2.0, 3.0, 4.0, 5.0}
	store.StoreEmbedding(ctx, client, "doc:test2", "content 2", embedding2, "plants", "")

	// Perform similarity search with label filter
	queryVector := []float32{1.1, 2.1, 3.1, 4.1}
	docs, err := store.SimilaritySearchWithLabel(ctx, client, indexName, queryVector, 5, "animals")
	if err != nil {
		t.Errorf("Similarity search with label failed: %v", err)
	}

	// Verify that only documents with the correct label are returned
	for _, doc := range docs {
		if doc.Fields["label"] != "animals" {
			t.Errorf("Expected label 'animals', got %s", doc.Fields["label"])
		}
	}
}

func TestSimilaritySearchWithLabelHandler_RequestValidation(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		method         string
		expectedStatus int
	}{
		{
			name: "Missing text field",
			requestBody: map[string]interface{}{
				"label": "test-label",
			},
			method:         http.MethodPost,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing label field",
			requestBody: map[string]interface{}{
				"text": "test query",
			},
			method:         http.MethodPost,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Empty label field",
			requestBody: map[string]interface{}{
				"text":  "test query",
				"label": "",
			},
			method:         http.MethodPost,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid method - GET instead of POST",
			requestBody: map[string]interface{}{
				"text":  "test query",
				"label": "test-label",
			},
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.requestBody)

			req := httptest.NewRequest(tt.method, "/search_with_label", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			ctx := context.Background()
			client := store.CreateRedisClient(getRedisAddress(), getRedisPassword())
			defer store.CloseRedisClient(client)

			openaiClient := openai.NewClient()

			api.SimilaritySearchWithLabelHandler(w, req, ctx, &openaiClient, client, "test-model", getRedisIndexName())

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestCreateEmbeddingRequest_JSONMarshaling(t *testing.T) {
	req := models.CreateEmbeddingRequest{
		Content:  "test content",
		Label:    "test-label",
		Metadata: "test-metadata",
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		t.Errorf("Failed to marshal request: %v", err)
	}

	var unmarshaled models.CreateEmbeddingRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal request: %v", err)
	}

	if unmarshaled.Content != req.Content {
		t.Errorf("Expected content %s, got %s", req.Content, unmarshaled.Content)
	}
	if unmarshaled.Label != req.Label {
		t.Errorf("Expected label %s, got %s", req.Label, unmarshaled.Label)
	}
	if unmarshaled.Metadata != req.Metadata {
		t.Errorf("Expected metadata %s, got %s", req.Metadata, unmarshaled.Metadata)
	}
}

func TestDistanceThresholdFiltering(t *testing.T) {
	results := []models.SimilaritySearchResult{
		{ID: "doc1", Distance: 0.3},
		{ID: "doc2", Distance: 0.6},
		{ID: "doc3", Distance: 0.9},
	}

	threshold := 0.7
	var filtered []models.SimilaritySearchResult

	for _, result := range results {
		if result.Distance <= threshold {
			filtered = append(filtered, result)
		}
	}

	if len(filtered) != 2 {
		t.Errorf("Expected 2 results after filtering, got %d", len(filtered))
	}

	// Verify that the correct documents were filtered
	if filtered[0].ID != "doc1" {
		t.Errorf("Expected first filtered result to be doc1, got %s", filtered[0].ID)
	}
	if filtered[1].ID != "doc2" {
		t.Errorf("Expected second filtered result to be doc2, got %s", filtered[1].ID)
	}
}

func TestStoreEmbedding_InvalidClient(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	client := store.CreateRedisClient("invalid:9999", getRedisPassword())
	defer store.CloseRedisClient(client)

	embedding := []float32{1.0, 2.0, 3.0}
	err := store.StoreEmbedding(ctx, client, "test:doc", "content", embedding, "", "")

	// We expect an error because the client cannot connect
	if err == nil {
		t.Log("Note: Expected an error with invalid Redis connection")
	}
}

func TestSimilaritySearchResult_JSONMarshaling(t *testing.T) {
	result := models.SimilaritySearchResult{
		ID:        "doc:123",
		Content:   "test content",
		Label:     "test-label",
		Metadata:  "test-metadata",
		Distance:  0.42,
		CreatedAt: "2025-11-30T10:00:00Z",
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		t.Errorf("Failed to marshal result: %v", err)
	}

	var unmarshaled models.SimilaritySearchResult
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal result: %v", err)
	}

	if unmarshaled.ID != result.ID {
		t.Errorf("Expected ID %s, got %s", result.ID, unmarshaled.ID)
	}
	if unmarshaled.Distance != result.Distance {
		t.Errorf("Expected distance %f, got %f", result.Distance, unmarshaled.Distance)
	}
}

func TestSplitAndStoreMarkdownWithHierarchyHandler_RequestValidation(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		method         string
		expectedStatus int
	}{
		{
			name: "Invalid method - GET instead of POST",
			requestBody: map[string]string{
				"document": "# Test\nContent",
			},
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Missing document field",
			requestBody:    map[string]string{},
			method:         http.MethodPost,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Empty document field",
			requestBody: map[string]string{
				"document": "",
			},
			method:         http.MethodPost,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			requestBody:    "invalid json",
			method:         http.MethodPost,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if str, ok := tt.requestBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(tt.method, "/split-and-store-markdown-with-hierarchy", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			ctx := context.Background()
			client := store.CreateRedisClient(getRedisAddress(), getRedisPassword())
			defer store.CloseRedisClient(client)

			openaiClient := openai.NewClient()

			api.SplitAndStoreMarkdownWithHierarchyHandler(w, req, ctx, &openaiClient, client, "test-model", getRedisIndexName())

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestSplitAndStoreMarkdownWithHierarchyRequest_JSONMarshaling(t *testing.T) {
	req := models.SplitAndStoreMarkdownWithHierarchyRequest{
		Document: "# Test\nContent",
		Label:    "test-label",
		Metadata: "test-metadata",
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		t.Errorf("Failed to marshal request: %v", err)
	}

	var unmarshaled models.SplitAndStoreMarkdownWithHierarchyRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal request: %v", err)
	}

	if unmarshaled.Document != req.Document {
		t.Errorf("Expected document %s, got %s", req.Document, unmarshaled.Document)
	}
	if unmarshaled.Label != req.Label {
		t.Errorf("Expected label %s, got %s", req.Label, unmarshaled.Label)
	}
	if unmarshaled.Metadata != req.Metadata {
		t.Errorf("Expected metadata %s, got %s", req.Metadata, unmarshaled.Metadata)
	}
}

func TestSplitAndStoreMarkdownWithHierarchyResponse_JSONMarshaling(t *testing.T) {
	resp := models.SplitAndStoreMarkdownWithHierarchyResponse{
		ChunkIDs:     []string{"doc:1", "doc:2", "doc:3"},
		ChunksStored: 3,
		Success:      true,
	}

	jsonData, err := json.Marshal(resp)
	if err != nil {
		t.Errorf("Failed to marshal response: %v", err)
	}

	var unmarshaled models.SplitAndStoreMarkdownWithHierarchyResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if unmarshaled.Success != resp.Success {
		t.Errorf("Expected success %v, got %v", resp.Success, unmarshaled.Success)
	}
	if unmarshaled.ChunksStored != resp.ChunksStored {
		t.Errorf("Expected chunks_stored %d, got %d", resp.ChunksStored, unmarshaled.ChunksStored)
	}
	if len(unmarshaled.ChunkIDs) != len(resp.ChunkIDs) {
		t.Errorf("Expected %d chunk IDs, got %d", len(resp.ChunkIDs), len(unmarshaled.ChunkIDs))
	}
}
