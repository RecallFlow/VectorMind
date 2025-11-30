package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"vectormind/api"
	"vectormind/helpers"
	"vectormind/mcptools"
	"vectormind/store"

	"github.com/mark3labs/mcp-go/server"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func main() {
	ctx := context.Background()

	mcpHttpPort := helpers.GetEnvOrDefault("MCP_HTTP_PORT", "9090")
	apiRestPort := helpers.GetEnvOrDefault("API_REST_PORT", "8080")

	redisIndexName := helpers.GetEnvOrDefault("REDIS_INDEX_NAME", "vector_idx")
	redisAddress := helpers.GetEnvOrDefault("REDIS_ADDRESS", "localhost:6379")
	redisPassword := helpers.GetEnvOrDefault("REDIS_PASSWORD", "")

	embeddingModelId := helpers.GetEnvOrDefault("EMBEDDING_MODEL", "ai/mxbai-embed-large")
	api.SetEmbeddingModelId(embeddingModelId)
	mcptools.SetEmbeddingModelId(embeddingModelId)
	modelRunnerEndpoint := helpers.GetEnvOrDefault("MODEL_RUNNER_BASE_URL", "http://localhost:12434/engines/llama.cpp/v1")

	// Initialize OpenAI client
	openaiClient := openai.NewClient(
		option.WithBaseURL(modelRunnerEndpoint),
		option.WithAPIKey(""),
	)

	// Calculate the embedding dimension based on the model
	var embeddingDimension int
	e, err := store.CreateEmbeddingFromText(ctx, openaiClient, "Hello World", embeddingModelId)
	if err != nil {
		log.Fatalf("Failed to create test embedding to determine dimension: %v", err)
	}
	embeddingDimension = len(e)
	api.SetEmbeddingDimension(embeddingDimension)
	mcptools.SetEmbeddingDimension(embeddingDimension)
	fmt.Printf("Using embedding dimension: %d\n", embeddingDimension)

	// Create Redis client
	redisClient := store.CreateRedisClient(redisAddress, redisPassword)
	defer store.CloseRedisClient(redisClient)

	// Check if index exists, create it if not
	exists, err := store.IndexExists(ctx, redisClient, redisIndexName)
	if err != nil {
		fmt.Printf("Error checking index: %v\n", err)
		return
	}

	if !exists {
		fmt.Printf("Index '%s' does not exist, creating it...\n", redisIndexName)
		err = store.CreateEmbeddingIndex(ctx, redisClient, redisIndexName, embeddingDimension)
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

	// Register MCP tools
	mcptools.RegisterTools(mcpServer, openaiClient, redisClient, embeddingModelId, redisIndexName)

	// Create REST API mux
	apiMux := http.NewServeMux()

	// Add healthcheck endpoint
	apiMux.HandleFunc("/health", api.HealthCheckHandler)

	// Add embedding model info endpoint
	apiMux.HandleFunc("/embedding-model-info", api.GetEmbeddingModelInfoHandler)

	// Add create embedding endpoint
	apiMux.HandleFunc("/embeddings", func(w http.ResponseWriter, r *http.Request) {
		api.CreateEmbeddingHandler(w, r, ctx, &openaiClient, redisClient, embeddingModelId, redisIndexName)
	})

	// Add similarity search endpoint
	apiMux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		api.SimilaritySearchHandler(w, r, ctx, &openaiClient, redisClient, embeddingModelId, redisIndexName)
	})

	// Add similarity search with label endpoint
	apiMux.HandleFunc("/search_with_label", func(w http.ResponseWriter, r *http.Request) {
		api.SimilaritySearchWithLabelHandler(w, r, ctx, &openaiClient, redisClient, embeddingModelId, redisIndexName)
	})

	// Add chunk and store endpoint
	apiMux.HandleFunc("/chunk-and-store", func(w http.ResponseWriter, r *http.Request) {
		api.ChunkAndStoreHandler(w, r, ctx, &openaiClient, redisClient, embeddingModelId, redisIndexName)
	})

	// Add split and store markdown sections endpoint
	apiMux.HandleFunc("/split-and-store-markdown-sections", func(w http.ResponseWriter, r *http.Request) {
		api.SplitAndStoreMarkdownSectionsHandler(w, r, ctx, &openaiClient, redisClient, embeddingModelId, redisIndexName)
	})

	// Add split and store with delimiter endpoint
	apiMux.HandleFunc("/split-and-store-with-delimiter", func(w http.ResponseWriter, r *http.Request) {
		api.SplitAndStoreWithDelimiterHandler(w, r, ctx, &openaiClient, redisClient, embeddingModelId, redisIndexName)
	})

	// Add split and store markdown with hierarchy endpoint
	apiMux.HandleFunc("/split-and-store-markdown-with-hierarchy", func(w http.ResponseWriter, r *http.Request) {
		api.SplitAndStoreMarkdownWithHierarchyHandler(w, r, ctx, &openaiClient, redisClient, embeddingModelId, redisIndexName)
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
