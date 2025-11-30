package mcptools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterAboutTool registers the about_vectormind tool
func RegisterAboutTool(mcpServer *server.MCPServer) {
	aboutTool := mcp.NewTool("about_vectormind",
		mcp.WithDescription("This tool provides information about the VectorMind MCP server."),
	)
	mcpServer.AddTool(aboutTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("This MCP Server is a Text RAG System based on Redis"), nil
	})
}
