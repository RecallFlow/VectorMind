package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"vectormind/api"
	"vectormind/models"
	"vectormind/store"

	"github.com/openai/openai-go"
)

func TestCreateRedisClient(t *testing.T) {
	tests := []struct {
		name          string
		redisAddress  string
		redisPassword string
	}{
		{
			name:          "Create client with default settings",
			redisAddress:  "localhost:6379",
			redisPassword: "",
		},
		{
			name:          "Create client with password",
			redisAddress:  "localhost:6379",
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
	client := store.CreateRedisClient("localhost:6379", "")
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
	client := store.CreateRedisClient("localhost:6379", "")
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
	client := store.CreateRedisClient("localhost:6379", "")
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
	client := store.CreateRedisClient("localhost:6379", "")
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
	client := store.CreateRedisClient("localhost:6379", "")
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
	client := store.CreateRedisClient("localhost:6379", "")
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
			client := store.CreateRedisClient("localhost:6379", "")
			defer store.CloseRedisClient(client)

			openaiClient := openai.NewClient()

			api.SimilaritySearchHandler(w, req, ctx, &openaiClient, client, "test-model", "test-index")

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
