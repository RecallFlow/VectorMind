package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"vectormind/splitter"
	"vectormind/store"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/openai/openai-go"
	"github.com/redis/go-redis/v9"
)

// RegisterMarkdownTools registers all markdown-related tools
func RegisterMarkdownTools(mcpServer *server.MCPServer, openaiClient openai.Client, redisClient *redis.Client, embeddingModelId string) {
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
