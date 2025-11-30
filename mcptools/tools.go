package mcptools

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/openai/openai-go"
	"github.com/redis/go-redis/v9"
)

// RegisterTools registers all MCP tools with the server
func RegisterTools(mcpServer *server.MCPServer, openaiClient openai.Client, redisClient *redis.Client, embeddingModelId, redisIndexName string) {
	// Register all tools organized by category
	RegisterAboutTool(mcpServer)
	RegisterEmbeddingTools(mcpServer, openaiClient, redisClient, embeddingModelId)
	RegisterSearchTools(mcpServer, openaiClient, redisClient, embeddingModelId, redisIndexName)
	RegisterChunkingTool(mcpServer, openaiClient, redisClient, embeddingModelId)
	RegisterMarkdownTools(mcpServer, openaiClient, redisClient, embeddingModelId)
}
