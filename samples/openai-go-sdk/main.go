package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const VECTORMIND_API = "http://localhost:8080"

// EmbeddingRequest represents the request to create an embedding
type EmbeddingRequest struct {
	Content  string `json:"content"`
	Label    string `json:"label,omitempty"`
	Metadata string `json:"metadata,omitempty"`
}

// EmbeddingResponse represents the embedding creation response
type EmbeddingResponse struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Label     string `json:"label"`
	Metadata  string `json:"metadata"`
	CreatedAt string `json:"created_at"`
	Success   bool   `json:"success"`
}

// SearchRequest represents the search request
type SearchRequest struct {
	Text              string  `json:"text"`
	MaxCount          int     `json:"max_count,omitempty"`
	DistanceThreshold float64 `json:"distance_threshold,omitempty"`
}

// SearchResult represents a search result
type SearchResult struct {
	ID       string  `json:"id"`
	Content  string  `json:"content"`
	Distance float64 `json:"distance"`
}

// SearchResponse represents the search response
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Success bool           `json:"success"`
}

// CreateEmbedding creates an embedding in VectorMind
func CreateEmbedding(content, label, metadata string) (*EmbeddingResponse, error) {
	reqBody := EmbeddingRequest{
		Content:  content,
		Label:    label,
		Metadata: metadata,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling error: %w", err)
	}

	resp, err := http.Post(VECTORMIND_API+"/embeddings", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("response reading error: %w", err)
	}

	var result EmbeddingResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling error: %w", err)
	}

	return &result, nil
}

// SearchSimilar searches for similar documents
func SearchSimilar(text string, maxCount int, distanceThreshold float64) (*SearchResponse, error) {
	reqBody := SearchRequest{
		Text:              text,
		MaxCount:          maxCount,
		DistanceThreshold: distanceThreshold,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling error: %w", err)
	}

	resp, err := http.Post(VECTORMIND_API+"/search", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("response reading error: %w", err)
	}

	var result SearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshaling error: %w", err)
	}

	return &result, nil
}

func main() {

	baseURL := "http://localhost:12434/engines/llama.cpp/v1/"
	model := "hf.co/menlo/jan-nano-gguf:q4_k_m"

	client := openai.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey(""),
	)

	ctx := context.Background()

	chunks := []string{
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

	// Creation of embeddings
	fmt.Println("Creation of embeddings...")
	for _, chunk := range chunks {
		result, err := CreateEmbedding(chunk, "fantasy-creatures", "")
		if err != nil {
			fmt.Printf("Error when embedding: %v\n", err)
			continue
		}
		fmt.Printf("Embedding created: ID=%s, Success=%v\n", result.ID, result.Success)
	}

	// Search for similar documents
	fmt.Println("\n\nSearch for similar documents...")

	userInput := "Tell me something about the dragons"

	searchResult, err := SearchSimilar(userInput, 2, 0.7)
	if err != nil {
		fmt.Printf("Error search: %v\n", err)
	}

	fmt.Printf("Found: %d\n", len(searchResult.Results))

	var documents string

	for i, result := range searchResult.Results {
		fmt.Printf("  %d. Distance: %.4f\n", i+1, result.Distance)
		fmt.Printf("     ID: %s\n", result.ID)
		fmt.Printf("     Content: %s...\n", result.Content[:50])
		documents += result.Content + "\n"
	}

	fmt.Println(strings.Repeat("-", 50))
	fmt.Println("Chat Completion with retrieved documents as context:")

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("Using the following documents:"),
		openai.SystemMessage("documents:\n" + documents),
		openai.UserMessage(userInput),
	}

	param := openai.ChatCompletionNewParams{
		Messages:    messages,
		Model:       model,
		Temperature: openai.Opt(0.0),
	}

	stream := client.Chat.Completions.NewStreaming(ctx, param)

	for stream.Next() {
		chunk := stream.Current()
		// Stream each chunk as it arrives
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
	}

	if err := stream.Err(); err != nil {
		log.Fatalln("Error with the completion:", err)
	}
}
