package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"
	"vectormind/helpers"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()

	mcpHttpPort := helpers.GetEnvOrDefault("MCP_HTTP_PORT", "9090")
	apiRestPort := helpers.GetEnvOrDefault("API_REST_PORT", "8080")

	redisIndexName := helpers.GetEnvOrDefault("REDIS_INDEX_NAME", "vector_idx")
	redisAddress := helpers.GetEnvOrDefault("REDIS_ADDRESS", "localhost:6379")
	redisPassword := helpers.GetEnvOrDefault("REDIS_PASSWORD", "")
	embeddingDimension := helpers.StringToInt(helpers.GetEnvOrDefault("EMBEDDING_DIMENSION", "1024"))

	embeddingModelId := helpers.GetEnvOrDefault("EMBEDDING_MODEL", "ai/mxbai-embed-large")
	modelRunnerEndpoint := helpers.GetEnvOrDefault("MODEL_RUNNER_BASE_URL", "http://localhost:12434/engines/llama.cpp/v1")

	// Initialize OpenAI openaiClient
	openaiClient := openai.NewClient(
		option.WithBaseURL(modelRunnerEndpoint),
		option.WithAPIKey(""),
	)

	// Create Redis client
	redisClient := CreateRedisClient(redisAddress, redisPassword)
	defer CloseRedisClient(redisClient)

	// Check if index exists, create it if not
	exists, err := IndexExists(ctx, redisClient, redisIndexName)
	if err != nil {
		fmt.Printf("Error checking index: %v\n", err)
		return
	}

	if !exists {
		fmt.Printf("Index '%s' does not exist, creating it...\n", redisIndexName)
		err = CreateEmbeddingIndex(ctx, redisClient, redisIndexName, embeddingDimension)
		if err != nil {
			fmt.Printf("Error creating index: %v\n", err)
			return
		}
		fmt.Printf("Index '%s' created successfully\n", redisIndexName)
	} else {
		fmt.Printf("Index '%s' already exists\n", redisIndexName)
	}

	// Create MCP server
	mcpServer := server.NewMCPServer(
		"mcp-vectormind",
		"0.0.0",
	)

	aboutTool := mcp.NewTool("about_vectormind",
		mcp.WithDescription("This tool provides information about the VectorMind MCP server."),
	)
	mcpServer.AddTool(aboutTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("This MCP Server is a Text RAG System based on Redis"), nil
	})

	// Add create_embedding MCP tool
	createEmbeddingTool := mcp.NewTool("create_embedding",
		mcp.WithDescription("Create and store an embedding from text content with optional label and metadata."),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("The text content to create an embedding from"),
		),
		mcp.WithString("label",
			mcp.Description("Optional label/tag for the document"),
		),
		mcp.WithString("metadata",
			mcp.Description("Optional metadata for the document"),
		),
	)
	mcpServer.AddTool(createEmbeddingTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		args := request.GetArguments()

		content, ok := args["content"].(string)
		if !ok || content == "" {
			return mcp.NewToolResultError("content parameter is required"), nil
		}

		label, _ := args["label"].(string)
		metadata, _ := args["metadata"].(string)

		// Create embedding from text
		embedding, err := CreateEmbeddingFromText(ctx, openaiClient, content, embeddingModelId)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create embedding: %v", err)), nil
		}

		// Generate unique document ID
		docID := fmt.Sprintf("doc:%s", uuid.New().String())

		// Store embedding in Redis
		err = StoreEmbedding(ctx, redisClient, docID, content, embedding, label, metadata)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to store embedding: %v", err)), nil
		}

		// Return success response
		result := map[string]interface{}{
			"success":    true,
			"id":         docID,
			"content":    content,
			"label":      label,
			"metadata":   metadata,
			"created_at": time.Now().Format(time.RFC3339),
		}

		resultJSON, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultJSON)), nil
	})

	// Add similarity_search MCP tool
	similaritySearchTool := mcp.NewTool("similarity_search",
		mcp.WithDescription("Search for similar documents based on text query. Returns documents ordered by similarity (closest first). Optionally filter by distance threshold."),
		mcp.WithString("text",
			mcp.Required(),
			mcp.Description("The text query to search for similar documents"),
		),
		mcp.WithNumber("max_count",
			mcp.Description("Maximum number of results to return (default: 1)"),
		),
		mcp.WithNumber("distance_threshold",
			mcp.Description("Optional distance threshold. Only returns documents with distance <= threshold"),
		),
	)
	mcpServer.AddTool(similaritySearchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		args := request.GetArguments()

		text, ok := args["text"].(string)
		if !ok || text == "" {
			return mcp.NewToolResultError("text parameter is required"), nil
		}

		// Get max_count with default value of 1
		maxCount := 1
		if mc, ok := args["max_count"].(float64); ok {
			maxCount = int(mc)
		}
		if maxCount <= 0 {
			maxCount = 1
		}

		// Get optional distance_threshold
		var distanceThreshold *float64
		if dt, ok := args["distance_threshold"].(float64); ok {
			distanceThreshold = &dt
		}

		// Create embedding from query text
		queryEmbedding, err := CreateEmbeddingFromText(ctx, openaiClient, text, embeddingModelId)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create embedding: %v", err)), nil
		}

		// Perform similarity search
		docs, err := SimilaritySearch(ctx, redisClient, redisIndexName, queryEmbedding, maxCount)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to perform similarity search: %v", err)), nil
		}

		// Convert results to response format
		results := make([]SimilaritySearchResult, 0, len(docs))
		for _, doc := range docs {
			str := doc.Fields["vector_distance"]
			distance, err := strconv.ParseFloat(str, 32)
			if err != nil {
				distance = 0.0
			}

			// Filter by distance threshold if specified
			if distanceThreshold != nil && distance > *distanceThreshold {
				continue
			}

			result := SimilaritySearchResult{
				ID:       doc.ID,
				Content:  doc.Fields["content"],
				Distance: distance,
			}

			results = append(results, result)
		}

		// Sort results by distance in ascending order (closest first)
		sort.Slice(results, func(i, j int) bool {
			return results[i].Distance < results[j].Distance
		})

		// Return success response
		response := map[string]interface{}{
			"success": true,
			"results": results,
		}

		resultJSON, _ := json.Marshal(response)
		return mcp.NewToolResultText(string(resultJSON)), nil
	})

	// Create REST API mux
	apiMux := http.NewServeMux()

	// Add healthcheck endpoint
	apiMux.HandleFunc("/health", healthCheckHandler)

	// Add create embedding endpoint
	apiMux.HandleFunc("/embeddings", func(w http.ResponseWriter, r *http.Request) {
		createEmbeddingHandler(w, r, ctx, &openaiClient, redisClient, embeddingModelId, redisIndexName)
	})

	// Add similarity search endpoint
	apiMux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		similaritySearchHandler(w, r, ctx, &openaiClient, redisClient, embeddingModelId, redisIndexName)
	})

	// Create MCP mux
	mcpMux := http.NewServeMux()

	// Add MCP endpoint
	httpServer := server.NewStreamableHTTPServer(mcpServer,
		server.WithEndpointPath("/mcp"),
	)
	mcpMux.Handle("/mcp", httpServer)

	// Start REST API server in a goroutine
	go func() {
		log.Println("REST API Server is running on port", apiRestPort)
		if err := http.ListenAndServe(":"+apiRestPort, apiMux); err != nil {
			log.Fatal("REST API Server error:", err)
		}
	}()

	// Start MCP server on main thread
	log.Println("MCP Server is running on port", mcpHttpPort)
	log.Fatal(http.ListenAndServe(":"+mcpHttpPort, mcpMux))

}

func CreateRedisClient(redisAddress, redisPassword string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: redisPassword,
		DB:       0, // use default DB
		Protocol: 2, // specify the Redis protocol version QUESTION: what does this do?
	})

	return client
}

func CloseRedisClient(client *redis.Client) error {
	return client.Close()
}

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

func CreateEmbeddingIndex(ctx context.Context, redisClient *redis.Client, indexName string, embeddingDimension int) error {

	/*
		Create the index.
		The schema in the example below specifies hash objects for storage and includes three fields:
		 - the text content to index,
		 - a tag field to represent the "genre" of the text,
		 - and the embedding vector generated from the original text content.
		The embedding field specifies HNSW indexing, the L2 vector distance metric, Float32 values to represent the vector's components,
		and 384 dimensions, as required by the all-MiniLM-L6-v2 embedding model.

		ai/mxbai-embed-large: This model produces 1024-dimensional embeddings.
	*/
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

func DropIndex(ctx context.Context, redisClient *redis.Client, indexName string) *redis.StatusCmd {
	return redisClient.FTDropIndexWithArgs(ctx,
		indexName,
		&redis.FTDropIndexOptions{
			DeleteDocs: true,
		},
	)
}

func ConvertEmbeddingToFloat32(embedding []float64) []float32 {
	float32Embedding := make([]float32, len(embedding))
	for i, v := range embedding {
		float32Embedding[i] = float32(v)
	}
	return float32Embedding
}

func ConvertOpenAIEmbeddingResponseToFloat32(embeddingResponse *openai.CreateEmbeddingResponse) []float32 {
	return ConvertEmbeddingToFloat32(embeddingResponse.Data[0].Embedding)
}

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

func StoreEmbedding(ctx context.Context, redisClient *redis.Client, docID string, content string, embedding []float32, label string, metadata string) error {
	buffer := floatsToBytes(embedding) // embedding vector as byte array
	_, err := redisClient.HSet(ctx,
		docID,
		map[string]any{
			"content":   content,
			"label":     label,
			"metadata":  metadata,
			"embedding": buffer,
		},
	).Result()

	return err
}

func floatsToBytes(fs []float32) []byte {
	buf := make([]byte, len(fs)*4)

	for i, f := range fs {
		u := math.Float32bits(f)
		binary.NativeEndian.PutUint32(buf[i*4:], u)
	}

	return buf
}

/* TODO:
- creation of embeddings and adding documents to the index
- remove and update documents in the index
- API + MCP
*/

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

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status": "healthy",
		"server": "mcp-vectormind-server",
	}
	json.NewEncoder(w).Encode(response)
}

type CreateEmbeddingRequest struct {
	Content  string `json:"content"`
	Label    string `json:"label"`
	Metadata string `json:"metadata"`
}

type CreateEmbeddingResponse struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Label     string    `json:"label"`
	Metadata  string    `json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

type SimilaritySearchRequest struct {
	Text              string   `json:"text"`
	MaxCount          int      `json:"max_count"`
	DistanceThreshold *float64 `json:"distance_threshold,omitempty"`
}

type SimilaritySearchResult struct {
	ID       string  `json:"id"`
	Content  string  `json:"content"`
	Distance float64 `json:"distance"`
}

type SimilaritySearchResponse struct {
	Results []SimilaritySearchResult `json:"results"`
	Success bool                     `json:"success"`
	Error   string                   `json:"error,omitempty"`
}

func createEmbeddingHandler(w http.ResponseWriter, r *http.Request, ctx context.Context, openaiClient *openai.Client, redisClient *redis.Client, embeddingModelId, indexName string) {
	w.Header().Set("Content-Type", "application/json")

	// Only accept POST requests
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(CreateEmbeddingResponse{
			Success: false,
			Error:   "Method not allowed. Use POST",
		})
		return
	}

	// Parse request body
	var req CreateEmbeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(CreateEmbeddingResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	// Validate required fields
	if req.Content == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(CreateEmbeddingResponse{
			Success: false,
			Error:   "Content is required",
		})
		return
	}

	// Create embedding from text
	embedding, err := CreateEmbeddingFromText(ctx, *openaiClient, req.Content, embeddingModelId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(CreateEmbeddingResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create embedding: %v", err),
		})
		return
	}

	// Generate unique document ID
	docID := fmt.Sprintf("doc:%s", uuid.New().String())

	// Store embedding in Redis
	err = StoreEmbedding(ctx, redisClient, docID, req.Content, embedding, req.Label, req.Metadata)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(CreateEmbeddingResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to store embedding: %v", err),
		})
		return
	}

	// Success response
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateEmbeddingResponse{
		ID:        docID,
		Content:   req.Content,
		Label:     req.Label,
		Metadata:  req.Metadata,
		CreatedAt: time.Now(),
		Success:   true,
	})
}

func similaritySearchHandler(w http.ResponseWriter, r *http.Request, ctx context.Context, openaiClient *openai.Client, redisClient *redis.Client, embeddingModelId, indexName string) {
	w.Header().Set("Content-Type", "application/json")

	// Only accept POST requests
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(SimilaritySearchResponse{
			Success: false,
			Error:   "Method not allowed. Use POST",
		})
		return
	}

	// Parse request body
	var req SimilaritySearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SimilaritySearchResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	// Validate required fields
	if req.Text == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SimilaritySearchResponse{
			Success: false,
			Error:   "Text is required",
		})
		return
	}

	if req.MaxCount <= 0 {
		req.MaxCount = 5 // Default value
	}

	// Create embedding from query text
	queryEmbedding, err := CreateEmbeddingFromText(ctx, *openaiClient, req.Text, embeddingModelId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SimilaritySearchResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create embedding: %v", err),
		})
		return
	}

	// Perform similarity search
	docs, err := SimilaritySearch(ctx, redisClient, indexName, queryEmbedding, req.MaxCount)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SimilaritySearchResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to perform similarity search: %v", err),
		})
		return
	}
	/*
		The results are ordered according to the value of the vector_distance field,
		with the lowest distance indicating the greatest similarity to the query.
	*/


	// Convert results to response format
	results := make([]SimilaritySearchResult, 0, len(docs))
	for _, doc := range docs {

		str := doc.Fields["vector_distance"]
		distance, err := strconv.ParseFloat(str, 32)
		if err != nil {
			// TODO:
			distance = 9.9
		}

		// Filter by distance threshold if specified
		if req.DistanceThreshold != nil && distance > *req.DistanceThreshold {
			continue
		}

		result := SimilaritySearchResult{
			ID:       doc.ID,
			Content:  doc.Fields["content"],
			Distance: distance,
		}

		results = append(results, result)
	}

	// Sort results by distance in ascending order (closest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	// Success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SimilaritySearchResponse{
		Results: results,
		Success: true,
	})
}
