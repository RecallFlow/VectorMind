package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"
	"vectormind/models"
	"vectormind/splitter"
	"vectormind/store"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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

// RegisterTools registers all MCP tools with the server
func RegisterTools(mcpServer *server.MCPServer, openaiClient openai.Client, redisClient *redis.Client, embeddingModelId, redisIndexName string) {
	// About tool
	aboutTool := mcp.NewTool("about_vectormind",
		mcp.WithDescription("This tool provides information about the VectorMind MCP server."),
	)
	mcpServer.AddTool(aboutTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("This MCP Server is a Text RAG System based on Redis"), nil
	})

	// Create embedding tool
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
		args := request.GetArguments()

		content, ok := args["content"].(string)
		if !ok || content == "" {
			return mcp.NewToolResultError("content parameter is required"), nil
		}

		label, _ := args["label"].(string)
		metadata, _ := args["metadata"].(string)

		// Create embedding from text
		embedding, err := store.CreateEmbeddingFromText(ctx, openaiClient, content, embeddingModelId)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create embedding: %v", err)), nil
		}

		// Generate unique document ID
		docID := fmt.Sprintf("doc:%s", uuid.New().String())

		// Store embedding in Redis
		err = store.StoreEmbedding(ctx, redisClient, docID, content, embedding, label, metadata)
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

	// Similarity search tool
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
		args := request.GetArguments()

		text, ok := args["text"].(string)
		if !ok || text == "" {
			return mcp.NewToolResultError("text parameter is required"), nil
		}

		maxCount := 1
		if mc, ok := args["max_count"].(float64); ok {
			maxCount = int(mc)
		}
		if maxCount <= 0 {
			maxCount = 1
		}

		var distanceThreshold *float64
		if dt, ok := args["distance_threshold"].(float64); ok {
			distanceThreshold = &dt
		}

		// Create embedding from query text
		queryEmbedding, err := store.CreateEmbeddingFromText(ctx, openaiClient, text, embeddingModelId)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create embedding: %v", err)), nil
		}

		// Perform similarity search
		docs, err := store.SimilaritySearch(ctx, redisClient, redisIndexName, queryEmbedding, maxCount)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to perform similarity search: %v", err)), nil
		}

		// Convert results to response format
		results := make([]models.SimilaritySearchResult, 0, len(docs))
		for _, doc := range docs {
			str := doc.Fields["vector_distance"]
			distance, err := strconv.ParseFloat(str, 32)
			if err != nil {
				distance = 0.0
			}

			if distanceThreshold != nil && distance > *distanceThreshold {
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

		sort.Slice(results, func(i, j int) bool {
			return results[i].Distance < results[j].Distance
		})

		response := map[string]interface{}{
			"success": true,
			"results": results,
		}

		resultJSON, _ := json.Marshal(response)
		return mcp.NewToolResultText(string(resultJSON)), nil
	})

	// Get embedding model info tool
	getEmbeddingModelInfoTool := mcp.NewTool("get_embedding_model_info",
		mcp.WithDescription("Get information about the embedding model being used, including the model ID and dimension."),
	)
	mcpServer.AddTool(getEmbeddingModelInfoTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result := map[string]interface{}{
			"model_id":  GetEmbeddingModelId(),
			"dimension": GetEmbeddingDimension(),
		}

		resultJSON, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultJSON)), nil
	})

	// Similarity search with label tool
	similaritySearchWithLabelTool := mcp.NewTool("similarity_search_with_label",
		mcp.WithDescription("Search for similar documents based on text query, filtered by label. Returns documents ordered by similarity (closest first). Optionally filter by distance threshold."),
		mcp.WithString("text",
			mcp.Required(),
			mcp.Description("The text query to search for similar documents"),
		),
		mcp.WithString("label",
			mcp.Required(),
			mcp.Description("The label to filter documents by"),
		),
		mcp.WithNumber("max_count",
			mcp.Description("Maximum number of results to return (default: 1)"),
		),
		mcp.WithNumber("distance_threshold",
			mcp.Description("Optional distance threshold. Only returns documents with distance <= threshold"),
		),
	)
	mcpServer.AddTool(similaritySearchWithLabelTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()

		text, ok := args["text"].(string)
		if !ok || text == "" {
			return mcp.NewToolResultError("text parameter is required"), nil
		}

		label, ok := args["label"].(string)
		if !ok || label == "" {
			return mcp.NewToolResultError("label parameter is required"), nil
		}

		maxCount := 1
		if mc, ok := args["max_count"].(float64); ok {
			maxCount = int(mc)
		}
		if maxCount <= 0 {
			maxCount = 1
		}

		var distanceThreshold *float64
		if dt, ok := args["distance_threshold"].(float64); ok {
			distanceThreshold = &dt
		}

		// Create embedding from query text
		queryEmbedding, err := store.CreateEmbeddingFromText(ctx, openaiClient, text, embeddingModelId)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create embedding: %v", err)), nil
		}

		// Perform similarity search with label filter
		docs, err := store.SimilaritySearchWithLabel(ctx, redisClient, redisIndexName, queryEmbedding, maxCount, label)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to perform similarity search: %v", err)), nil
		}

		// Convert results to response format
		results := make([]models.SimilaritySearchResult, 0, len(docs))
		for _, doc := range docs {
			str := doc.Fields["vector_distance"]
			distance, err := strconv.ParseFloat(str, 32)
			if err != nil {
				distance = 0.0
			}

			if distanceThreshold != nil && distance > *distanceThreshold {
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

		sort.Slice(results, func(i, j int) bool {
			return results[i].Distance < results[j].Distance
		})

		response := map[string]interface{}{
			"success": true,
			"results": results,
		}

		resultJSON, _ := json.Marshal(response)
		return mcp.NewToolResultText(string(resultJSON)), nil
	})

	// Chunk and store tool
	chunkAndStoreTool := mcp.NewTool("chunk_and_store",
		mcp.WithDescription("Chunk a document into smaller pieces with overlap and store all chunks with embeddings. All chunks will share the same label and metadata."),
		mcp.WithString("document",
			mcp.Required(),
			mcp.Description("The document content to chunk and store"),
		),
		mcp.WithString("label",
			mcp.Description("Optional label to apply to all chunks"),
		),
		mcp.WithString("metadata",
			mcp.Description("Optional metadata to apply to all chunks"),
		),
		mcp.WithNumber("chunk_size",
			mcp.Required(),
			mcp.Description("Size of each chunk in characters (must be <= embedding dimension)"),
		),
		mcp.WithNumber("overlap",
			mcp.Required(),
			mcp.Description("Number of characters to overlap between consecutive chunks (must be < chunk_size)"),
		),
	)
	mcpServer.AddTool(chunkAndStoreTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()

		document, ok := args["document"].(string)
		if !ok || document == "" {
			return mcp.NewToolResultError("document parameter is required"), nil
		}

		label, _ := args["label"].(string)
		metadata, _ := args["metadata"].(string)

		chunkSize, ok := args["chunk_size"].(float64)
		if !ok || chunkSize <= 0 {
			return mcp.NewToolResultError("chunk_size must be a positive number"), nil
		}

		overlap, ok := args["overlap"].(float64)
		if !ok || overlap < 0 {
			return mcp.NewToolResultError("overlap must be a non-negative number"), nil
		}

		chunkSizeInt := int(chunkSize)
		overlapInt := int(overlap)

		// Validate overlap < chunk_size
		if overlapInt >= chunkSizeInt {
			return mcp.NewToolResultError("overlap must be less than chunk_size"), nil
		}

		// Validate chunk_size <= embeddingDimension
		if chunkSizeInt > GetEmbeddingDimension() {
			return mcp.NewToolResultError(fmt.Sprintf("chunk_size (%d) must be less than or equal to embedding dimension (%d)", chunkSizeInt, GetEmbeddingDimension())), nil
		}

		// Chunk the document
		chunks := splitter.ChunkText(document, chunkSizeInt, overlapInt)

		if len(chunks) == 0 {
			return mcp.NewToolResultError("No chunks generated from the document"), nil
		}

		// Store all chunks
		chunkIDs := make([]string, 0, len(chunks))
		createdAt := time.Now()

		for _, chunk := range chunks {
			// Create embedding from chunk text
			embedding, err := store.CreateEmbeddingFromText(ctx, openaiClient, chunk, embeddingModelId)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to create embedding for chunk: %v", err)), nil
			}

			// Generate unique document ID for this chunk
			chunkID := fmt.Sprintf("doc:%s", uuid.New().String())

			// Store embedding in Redis with the same label and metadata for all chunks
			err = store.StoreEmbedding(ctx, redisClient, chunkID, chunk, embedding, label, metadata)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to store chunk embedding: %v", err)), nil
			}

			chunkIDs = append(chunkIDs, chunkID)
		}

		// Success response
		result := map[string]interface{}{
			"success":       true,
			"chunk_ids":     chunkIDs,
			"chunks_stored": len(chunkIDs),
			"created_at":    createdAt.Format(time.RFC3339),
		}

		resultJSON, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultJSON)), nil
	})

	// Split and store markdown sections tool
	splitAndStoreMarkdownSectionsTool := mcp.NewTool("split_and_store_markdown_sections",
		mcp.WithDescription("Split a markdown document by sections (headers like #, ##, ###) and store all sections with embeddings. Sections larger than embedding dimension are automatically subdivided. All chunks will share the same label and metadata."),
		mcp.WithString("document",
			mcp.Required(),
			mcp.Description("The markdown document content to split and store"),
		),
		mcp.WithString("label",
			mcp.Description("Optional label to apply to all sections/chunks"),
		),
		mcp.WithString("metadata",
			mcp.Description("Optional metadata to apply to all sections/chunks"),
		),
	)
	mcpServer.AddTool(splitAndStoreMarkdownSectionsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()

		document, ok := args["document"].(string)
		if !ok || document == "" {
			return mcp.NewToolResultError("document parameter is required"), nil
		}

		label, _ := args["label"].(string)
		metadata, _ := args["metadata"].(string)

		// Split markdown by sections
		sections := splitter.SplitMarkdownBySections(document)

		if len(sections) == 0 {
			return mcp.NewToolResultError("No sections generated from the document"), nil
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
				embedding, err := store.CreateEmbeddingFromText(ctx, openaiClient, chunk, embeddingModelId)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to create embedding for chunk: %v", err)), nil
				}

				// Generate unique document ID for this chunk
				chunkID := fmt.Sprintf("doc:%s", uuid.New().String())

				// Store embedding in Redis with the same label and metadata for all chunks
				err = store.StoreEmbedding(ctx, redisClient, chunkID, chunk, embedding, label, metadata)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to store chunk embedding: %v", err)), nil
				}

				chunkIDs = append(chunkIDs, chunkID)
			}
		}

		// Success response
		result := map[string]interface{}{
			"success":       true,
			"chunk_ids":     chunkIDs,
			"chunks_stored": len(chunkIDs),
			"created_at":    createdAt.Format(time.RFC3339),
		}

		resultJSON, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultJSON)), nil
	})

	// Split and store with delimiter tool
	splitAndStoreWithDelimiterTool := mcp.NewTool("split_and_store_with_delimiter",
		mcp.WithDescription("Split a document by a custom delimiter and store all chunks with embeddings. Chunks larger than embedding dimension are automatically subdivided with the first 2 non-empty lines prepended to preserve context. All chunks will share the same label and metadata."),
		mcp.WithString("document",
			mcp.Required(),
			mcp.Description("The document content to split and store"),
		),
		mcp.WithString("delimiter",
			mcp.Required(),
			mcp.Description("The delimiter used to split the document (e.g., '-----', '###', etc.)"),
		),
		mcp.WithString("label",
			mcp.Description("Optional label to apply to all chunks"),
		),
		mcp.WithString("metadata",
			mcp.Description("Optional metadata to apply to all chunks"),
		),
	)
	mcpServer.AddTool(splitAndStoreWithDelimiterTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()

		document, ok := args["document"].(string)
		if !ok || document == "" {
			return mcp.NewToolResultError("document parameter is required"), nil
		}

		delimiter, ok := args["delimiter"].(string)
		if !ok || delimiter == "" {
			return mcp.NewToolResultError("delimiter parameter is required"), nil
		}

		label, _ := args["label"].(string)
		metadata, _ := args["metadata"].(string)

		// Split text by delimiter
		chunks := splitter.SplitTextWithDelimiter(document, delimiter)

		if len(chunks) == 0 {
			return mcp.NewToolResultError("No chunks generated from the document"), nil
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
				embedding, err := store.CreateEmbeddingFromText(ctx, openaiClient, chunkToStore, embeddingModelId)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to create embedding for chunk: %v", err)), nil
				}

				// Generate unique document ID for this chunk
				chunkID := fmt.Sprintf("doc:%s", uuid.New().String())

				// Store embedding in Redis with the same label and metadata for all chunks
				err = store.StoreEmbedding(ctx, redisClient, chunkID, chunkToStore, embedding, label, metadata)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to store chunk embedding: %v", err)), nil
				}

				chunkIDs = append(chunkIDs, chunkID)
			}
		}

		// Success response
		result := map[string]interface{}{
			"success":       true,
			"chunk_ids":     chunkIDs,
			"chunks_stored": len(chunkIDs),
			"created_at":    createdAt.Format(time.RFC3339),
		}

		resultJSON, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultJSON)), nil
	})

	// Split and store markdown with hierarchy tool (EXPERIMENTAL)
	splitAndStoreMarkdownWithHierarchyTool := mcp.NewTool("split_and_store_markdown_with_hierarchy",
		mcp.WithDescription("EXPERIMENTAL: Split a markdown document by headers, preserving hierarchical context (parent headers) in each chunk. Each chunk includes TITLE, HIERARCHY, and CONTENT metadata. Chunks larger than embedding dimension are automatically subdivided. All chunks share the same label and metadata."),
		mcp.WithString("document",
			mcp.Required(),
			mcp.Description("The markdown document content to split and store"),
		),
		mcp.WithString("label",
			mcp.Description("Optional label to apply to all chunks"),
		),
		mcp.WithString("metadata",
			mcp.Description("Optional metadata to apply to all chunks"),
		),
	)
	mcpServer.AddTool(splitAndStoreMarkdownWithHierarchyTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()

		document, ok := args["document"].(string)
		if !ok || document == "" {
			return mcp.NewToolResultError("document parameter is required"), nil
		}

		label, _ := args["label"].(string)
		metadata, _ := args["metadata"].(string)

		// Split markdown with hierarchy
		chunks := splitter.ChunkWithMarkdownHierarchy(document)

		if len(chunks) == 0 {
			return mcp.NewToolResultError("No chunks generated from the document"), nil
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
				embedding, err := store.CreateEmbeddingFromText(ctx, openaiClient, subChunk, embeddingModelId)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to create embedding for chunk: %v", err)), nil
				}

				// Generate unique document ID for this chunk
				chunkID := fmt.Sprintf("doc:%s", uuid.New().String())

				// Store embedding in Redis with the same label and metadata for all chunks
				err = store.StoreEmbedding(ctx, redisClient, chunkID, subChunk, embedding, label, metadata)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Failed to store chunk embedding: %v", err)), nil
				}

				chunkIDs = append(chunkIDs, chunkID)
			}
		}

		// Success response
		result := map[string]interface{}{
			"success":       true,
			"chunk_ids":     chunkIDs,
			"chunks_stored": len(chunkIDs),
			"created_at":    createdAt.Format(time.RFC3339),
		}

		resultJSON, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultJSON)), nil
	})
}
