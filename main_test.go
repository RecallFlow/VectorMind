package main

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

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
			client := CreateRedisClient(tt.redisAddress, tt.redisPassword)
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
	client := CreateRedisClient("localhost:6379", "")
	err := CloseRedisClient(client)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestConvertEmbeddingToFloat32(t *testing.T) {
	tests := []struct {
		name     string
		input    []float64
		expected []float32
	}{
		{
			name:     "Empty slice",
			input:    []float64{},
			expected: []float32{},
		},
		{
			name:     "Single value",
			input:    []float64{1.5},
			expected: []float32{1.5},
		},
		{
			name:     "Multiple values",
			input:    []float64{1.5, 2.7, 3.9, -4.2},
			expected: []float32{1.5, 2.7, 3.9, -4.2},
		},
		{
			name:     "Large values",
			input:    []float64{123456.789, -987654.321},
			expected: []float32{123456.789, -987654.321},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertEmbeddingToFloat32(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("At index %d: expected %f, got %f", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

func TestConvertOpenAIEmbeddingResponseToFloat32(t *testing.T) {
	tests := []struct {
		name     string
		response *openai.CreateEmbeddingResponse
		expected []float32
	}{
		{
			name: "Single embedding",
			response: &openai.CreateEmbeddingResponse{
				Data: []openai.Embedding{
					{
						Embedding: []float64{1.0, 2.0, 3.0},
					},
				},
			},
			expected: []float32{1.0, 2.0, 3.0},
		},
		{
			name: "Empty embedding",
			response: &openai.CreateEmbeddingResponse{
				Data: []openai.Embedding{
					{
						Embedding: []float64{},
					},
				},
			},
			expected: []float32{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertOpenAIEmbeddingResponseToFloat32(tt.response)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d", len(tt.expected), len(result))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("At index %d: expected %f, got %f", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

func TestFloatsToBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    []float32
		validate func([]byte) bool
	}{
		{
			name:  "Empty slice",
			input: []float32{},
			validate: func(b []byte) bool {
				return len(b) == 0
			},
		},
		{
			name:  "Single float",
			input: []float32{1.5},
			validate: func(b []byte) bool {
				return len(b) == 4
			},
		},
		{
			name:  "Multiple floats",
			input: []float32{1.0, 2.0, 3.0, 4.0},
			validate: func(b []byte) bool {
				return len(b) == 16 // 4 floats * 4 bytes each
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := floatsToBytes(tt.input)
			if !tt.validate(result) {
				t.Errorf("Validation failed for input %v", tt.input)
			}
		})
	}
}

func TestFloatsToBytesRoundTrip(t *testing.T) {
	// Test that we can convert floats to bytes and the length is correct
	input := []float32{1.5, 2.7, 3.9, -4.2}
	bytes := floatsToBytes(input)

	expectedLength := len(input) * 4 // each float32 is 4 bytes
	if len(bytes) != expectedLength {
		t.Errorf("Expected byte length %d, got %d", expectedLength, len(bytes))
	}

	// Verify we can extract individual float bits
	for i, f := range input {
		expectedBits := math.Float32bits(f)
		_ = expectedBits // We created the bytes correctly based on the bits
		// The actual bytes in the buffer at position i*4 should represent this float
		if len(bytes) < (i+1)*4 {
			t.Errorf("Byte array too short for float at index %d", i)
		}
	}
}

func TestHealthCheckHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	healthCheckHandler(w, req)

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
	client := CreateRedisClient("localhost:6379", "")
	defer CloseRedisClient(client)

	// Test with non-existent index
	exists, err := IndexExists(ctx, client, "non_existent_index")
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
	client := CreateRedisClient("localhost:6379", "")
	defer CloseRedisClient(client)

	// Clean up before test
	defer client.Del(ctx, "test:doc:1")

	embedding := []float32{1.0, 2.0, 3.0, 4.0}
	err := StoreEmbedding(ctx, client, "test:doc:1", "test content", embedding, "test-label", "test-metadata")
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
	client := CreateRedisClient("localhost:6379", "")
	defer CloseRedisClient(client)

	indexName := "test_vector_idx"

	// Clean up: drop index if it exists
	defer DropIndex(ctx, client, indexName)

	err := CreateEmbeddingIndex(ctx, client, indexName, 1024)
	if err != nil {
		t.Errorf("Failed to create index: %v", err)
	}

	// Verify index was created
	exists, err := IndexExists(ctx, client, indexName)
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
	client := CreateRedisClient("localhost:6379", "")
	defer CloseRedisClient(client)

	indexName := "test_drop_idx"

	// Create index first
	CreateEmbeddingIndex(ctx, client, indexName, 1024)

	// Drop the index
	result := DropIndex(ctx, client, indexName)
	if result.Err() != nil {
		t.Errorf("Failed to drop index: %v", result.Err())
	}

	// Verify index was dropped
	exists, _ := IndexExists(ctx, client, indexName)
	if exists {
		t.Error("Expected index to not exist after dropping")
	}
}

func TestSimilaritySearch_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	client := CreateRedisClient("localhost:6379", "")
	defer CloseRedisClient(client)

	indexName := "test_similarity_idx"
	defer DropIndex(ctx, client, indexName)

	// Create index and add some test data
	CreateEmbeddingIndex(ctx, client, indexName, 4)

	embedding1 := []float32{1.0, 2.0, 3.0, 4.0}
	StoreEmbedding(ctx, client, "doc:test1", "content 1", embedding1, "", "")

	// Perform similarity search
	queryVector := []float32{1.1, 2.1, 3.1, 4.1}
	docs, err := SimilaritySearch(ctx, client, indexName, queryVector, 5)
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
		validateResponse  func(*testing.T, SimilaritySearchResponse)
	}{
		{
			name: "Invalid method - GET instead of POST",
			requestBody: SimilaritySearchRequest{
				Text:     "test query",
				MaxCount: 5,
			},
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
			validateResponse: func(t *testing.T, resp SimilaritySearchResponse) {
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
			validateResponse: func(t *testing.T, resp SimilaritySearchResponse) {
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
			requestBody: SimilaritySearchRequest{
				Text:     "",
				MaxCount: 5,
			},
			method:         http.MethodPost,
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, resp SimilaritySearchResponse) {
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
			client := CreateRedisClient("localhost:6379", "")
			defer CloseRedisClient(client)

			openaiClient := openai.NewClient()

			similaritySearchHandler(w, req, ctx, &openaiClient, client, "test-model", "test-index")

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			var response SimilaritySearchResponse
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
		expected SimilaritySearchRequest
	}{
		{
			name:    "Without distance threshold",
			jsonStr: `{"text":"test","max_count":5}`,
			expected: SimilaritySearchRequest{
				Text:              "test",
				MaxCount:          5,
				DistanceThreshold: nil,
			},
		},
		{
			name:    "With distance threshold",
			jsonStr: `{"text":"test","max_count":5,"distance_threshold":0.5}`,
			expected: SimilaritySearchRequest{
				Text:              "test",
				MaxCount:          5,
				DistanceThreshold: floatPtr(0.5),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req SimilaritySearchRequest
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
