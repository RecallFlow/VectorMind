package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"
	"vectormind/models"
	"vectormind/store"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/openai/openai-go"
	"github.com/redis/go-redis/v9"
)

// RegisterSearchTools registers the similarity_search and similarity_search_with_label tools
func RegisterSearchTools(mcpServer *server.MCPServer, openaiClient openai.Client, redisClient *redis.Client, embeddingModelId, redisIndexName string) {
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
}
