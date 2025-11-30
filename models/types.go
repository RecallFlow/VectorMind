package models

import "time"

// CreateEmbeddingRequest represents the request to create an embedding
type CreateEmbeddingRequest struct {
	Content  string `json:"content"`
	Label    string `json:"label"`
	Metadata string `json:"metadata"`
}

// CreateEmbeddingResponse represents the response after creating an embedding
type CreateEmbeddingResponse struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Label     string    `json:"label"`
	Metadata  string    `json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

// SimilaritySearchRequest represents the request for similarity search
type SimilaritySearchRequest struct {
	Text              string   `json:"text"`
	MaxCount          int      `json:"max_count"`
	DistanceThreshold *float64 `json:"distance_threshold,omitempty"`
}

// SimilaritySearchWithLabelRequest represents the request for similarity search with label filter
type SimilaritySearchWithLabelRequest struct {
	Text              string   `json:"text"`
	Label             string   `json:"label"`
	MaxCount          int      `json:"max_count"`
	DistanceThreshold *float64 `json:"distance_threshold,omitempty"`
}

// SimilaritySearchResult represents a single search result
type SimilaritySearchResult struct {
	ID        string  `json:"id"`
	Content   string  `json:"content"`
	Label     string  `json:"label"`
	Metadata  string  `json:"metadata"`
	Distance  float64 `json:"distance"`
	CreatedAt string  `json:"created_at"`
}

// SimilaritySearchResponse represents the response for similarity search
type SimilaritySearchResponse struct {
	Results []SimilaritySearchResult `json:"results"`
	Success bool                     `json:"success"`
	Error   string                   `json:"error,omitempty"`
}

// ChunkAndStoreRequest represents the request to chunk and store a document
type ChunkAndStoreRequest struct {
	Document  string `json:"document"`
	Label     string `json:"label"`
	Metadata  string `json:"metadata"`
	ChunkSize int    `json:"chunk_size"`
	Overlap   int    `json:"overlap"`
}

// ChunkAndStoreResponse represents the response after chunking and storing a document
type ChunkAndStoreResponse struct {
	ChunkIDs     []string  `json:"chunk_ids"`
	ChunksStored int       `json:"chunks_stored"`
	CreatedAt    time.Time `json:"created_at"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
}

// SplitAndStoreMarkdownSectionsRequest represents the request to split markdown by sections and store
type SplitAndStoreMarkdownSectionsRequest struct {
	Document string `json:"document"`
	Label    string `json:"label"`
	Metadata string `json:"metadata"`
}

// SplitAndStoreMarkdownSectionsResponse represents the response after splitting and storing markdown sections
type SplitAndStoreMarkdownSectionsResponse struct {
	ChunkIDs     []string  `json:"chunk_ids"`
	ChunksStored int       `json:"chunks_stored"`
	CreatedAt    time.Time `json:"created_at"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
}

// SplitAndStoreWithDelimiterRequest represents the request to split text with a delimiter and store
type SplitAndStoreWithDelimiterRequest struct {
	Document  string `json:"document"`
	Delimiter string `json:"delimiter"`
	Label     string `json:"label"`
	Metadata  string `json:"metadata"`
}

// SplitAndStoreWithDelimiterResponse represents the response after splitting and storing with delimiter
type SplitAndStoreWithDelimiterResponse struct {
	ChunkIDs     []string  `json:"chunk_ids"`
	ChunksStored int       `json:"chunks_stored"`
	CreatedAt    time.Time `json:"created_at"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
}

// SplitAndStoreMarkdownWithHierarchyRequest represents the request to split markdown with hierarchy and store
type SplitAndStoreMarkdownWithHierarchyRequest struct {
	Document string `json:"document"`
	Label    string `json:"label"`
	Metadata string `json:"metadata"`
}

// SplitAndStoreMarkdownWithHierarchyResponse represents the response after splitting and storing markdown with hierarchy
type SplitAndStoreMarkdownWithHierarchyResponse struct {
	ChunkIDs     []string  `json:"chunk_ids"`
	ChunksStored int       `json:"chunks_stored"`
	CreatedAt    time.Time `json:"created_at"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
}
