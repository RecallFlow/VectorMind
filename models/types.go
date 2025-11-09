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
