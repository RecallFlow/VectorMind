package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

type SearchResult struct {
	ID       string  `json:"id"`
	Content  string  `json:"content"`
	Distance float64 `json:"distance"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Success bool           `json:"success"`
}

var chunks = []string{
	`# Orcs
		Orcs are savage, brutish humanoids with dark green skin and prominent tusks.
		These fierce warriors inhabit dense forests where they hunt in packs,
		using crude but effective weapons forged from scavenged metal and bone.
		Their tribal society revolves around strength and combat prowess,
		making them formidable opponents for any adventurer brave enough to enter their woodland domain.`,

	`# Dragons
		Dragons are magnificent and ancient creatures of immense power, soaring through the skies on massive wings.
		These intelligent beings possess scales that shimmer like precious metals and breathe devastating elemental attacks.
		Known for their vast hoards of treasure and centuries of accumulated knowledge,
		dragons command both fear and respect throughout the realm.
		Their aerial dominance makes them nearly untouchable in their celestial domain.`,

	`# Goblins
		Goblins are small, cunning creatures with mottled green skin and sharp, pointed ears.
		Despite their diminutive size, they are surprisingly agile swimmers who have adapted to life around ponds and marshlands.
		These mischievous beings are known for their quick wit and tendency to play pranks on unwary travelers.
		They build elaborate underwater lairs connected by hidden tunnels beneath the murky pond waters.`,

	`# Krakens
		Krakens are colossal sea monsters with massive tentacles that can crush entire ships with ease.
		These legendary creatures dwell in the deepest ocean trenches, surfacing only to hunt or when disturbed.
		Their intelligence rivals that of the wisest sages, and their tentacles can stretch for hundreds of feet.
		Sailors speak in hushed tones of these maritime titans, whose very presence can create devastating whirlpools
		and tidal waves that reshape entire coastlines.`,
}

func main() {

	ctx := context.Background()

	// MCP client initialization
	fmt.Println("🚀 Initializing MCP StreamableHTTP client...")
	// Create HTTP transport
	httpURL := "http://localhost:9090/mcp"
	httpTransport, err := transport.NewStreamableHTTP(httpURL)
	if err != nil {
		log.Fatalf("Failed to create HTTP transport: %v", err)
	}
	// Create client with the transport
	mcpClient := client.NewClient(httpTransport)
	// Start the client
	if err := mcpClient.Start(ctx); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "MCP-Go Simple Client Example",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	_, err = mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	// Tools listing
	toolsRequest := mcp.ListToolsRequest{}
	// Get the list of tools
	toolsResult, err := mcpClient.ListTools(ctx, toolsRequest)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}
	fmt.Println("🛠️  Available tools:")
	for _, tool := range toolsResult.Tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}

	// Create Embeddings with `create_embedding` MCP tool
	fmt.Println("\n\nCreation of embeddings...")
	for _, chunk := range chunks {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "create_embedding",
				Arguments: map[string]any{
					"content":  chunk,
					"label":    "fantasy-creatures",
					"metadata": "",
				},
			},
		}
		toolResponse, err := mcpClient.CallTool(ctx, request)
		if err != nil {
			fmt.Printf("Error when embedding: %v\n", err)
			continue
		}
		if toolResponse == nil || len(toolResponse.Content) == 0 {
			fmt.Printf("No response from embedding tool\n")
			continue
		}
		fmt.Println("🛠️  Tool response:", toolResponse.Content[0].(mcp.TextContent).Text)

	}

	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("Search for similar documents...")

	userInput := "Tell me something about the dragons"

	searchRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "similarity_search",
			Arguments: map[string]any{
				"text":               userInput,
				"max_count":          2,
				"distance_threshold": 0.7,
			},
		},
	}
	searchResponse, err := mcpClient.CallTool(ctx, searchRequest)
	if err != nil {
		log.Fatalf("Error search: %v", err)
	}
	if searchResponse == nil || len(searchResponse.Content) == 0 {
		log.Fatalf("No response from search tool")
	}

	searchResult := searchResponse.Content[0].(mcp.TextContent).Text

	// Parse the JSON response
	var response SearchResponse
	err = json.Unmarshal([]byte(searchResult), &response)
	if err != nil {
		log.Fatalf("Error parsing search result: %v", err)
	}

	// Loop through results
	fmt.Println("\n📋 Search Results:")
	for _, result := range response.Results {
		fmt.Printf("\nID: %s\n", result.ID)
		fmt.Printf("Distance: %f\n", result.Distance)
		fmt.Printf("Content: %s\n", result.Content)
		fmt.Println(strings.Repeat("-", 50))
	}
}

