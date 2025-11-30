package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"vectormind/splitter"
	"vectormind/store"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/openai/openai-go"
	"github.com/redis/go-redis/v9"
)

// RegisterChunkingTool registers the chunk_and_store tool
func RegisterChunkingTool(mcpServer *server.MCPServer, openaiClient openai.Client, redisClient *redis.Client, embeddingModelId string) {
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
}
