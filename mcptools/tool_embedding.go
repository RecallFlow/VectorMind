package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"vectormind/store"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/openai/openai-go"
	"github.com/redis/go-redis/v9"
)

// RegisterEmbeddingTools registers the create_embedding and get_embedding_model_info tools
func RegisterEmbeddingTools(mcpServer *server.MCPServer, openaiClient openai.Client, redisClient *redis.Client, embeddingModelId string) {
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
}
