package splitter

import (
	"strings"
	"testing"
)

func TestParseMarkdownHierarchy(t *testing.T) {
	tests := []struct {
		name           string
		markdown       string
		expectedChunks int
		validateChunks func(*testing.T, []MarkdownChunk)
	}{
		{
			name: "Simple two-level hierarchy",
			markdown: `# Main Title
Content under main title.

## Subsection
Content under subsection.`,
			expectedChunks: 2,
			validateChunks: func(t *testing.T, chunks []MarkdownChunk) {
				if chunks[0].Header != "Main Title" {
					t.Errorf("Expected first header 'Main Title', got %s", chunks[0].Header)
				}
				if chunks[0].Level != 1 {
					t.Errorf("Expected first level 1, got %d", chunks[0].Level)
				}
				if chunks[0].Prefix != "#" {
					t.Errorf("Expected first prefix '#', got %s", chunks[0].Prefix)
				}
				if !strings.Contains(chunks[0].Content, "Content under main title") {
					t.Error("Expected first chunk content to contain 'Content under main title'")
				}

				if chunks[1].Header != "Subsection" {
					t.Errorf("Expected second header 'Subsection', got %s", chunks[1].Header)
				}
				if chunks[1].Level != 2 {
					t.Errorf("Expected second level 2, got %d", chunks[1].Level)
				}
				if chunks[1].ParentHeader != "Main Title" {
					t.Errorf("Expected parent header 'Main Title', got %s", chunks[1].ParentHeader)
				}
			},
		},
		{
			name: "Three-level hierarchy",
			markdown: `# Chapter 1
Chapter content.

## Section 1.1
Section content.

### Subsection 1.1.1
Subsection content.`,
			expectedChunks: 3,
			validateChunks: func(t *testing.T, chunks []MarkdownChunk) {
				if chunks[2].Level != 3 {
					t.Errorf("Expected third level 3, got %d", chunks[2].Level)
				}
				if chunks[2].ParentHeader != "Section 1.1" {
					t.Errorf("Expected parent header 'Section 1.1', got %s", chunks[2].ParentHeader)
				}
				expectedHierarchy := "Chapter 1 > Section 1.1 > Subsection 1.1.1"
				if chunks[2].Hierarchy != expectedHierarchy {
					t.Errorf("Expected hierarchy '%s', got '%s'", expectedHierarchy, chunks[2].Hierarchy)
				}
			},
		},
		{
			name: "Skip level hierarchy",
			markdown: `# Top Level
Content.

### Deep Level
Skipped level 2.`,
			expectedChunks: 2,
			validateChunks: func(t *testing.T, chunks []MarkdownChunk) {
				if chunks[1].Level != 3 {
					t.Errorf("Expected second level 3, got %d", chunks[1].Level)
				}
				if chunks[1].ParentHeader != "Top Level" {
					t.Errorf("Expected parent 'Top Level', got '%s'", chunks[1].ParentHeader)
				}
			},
		},
		{
			name: "Multiple sections at same level",
			markdown: `# Chapter 1
Content 1.

## Section 1.1
Section 1.1 content.

## Section 1.2
Section 1.2 content.

# Chapter 2
Content 2.`,
			expectedChunks: 4,
			validateChunks: func(t *testing.T, chunks []MarkdownChunk) {
				// Both section 1.1 and 1.2 should have Chapter 1 as parent
				if chunks[1].ParentHeader != "Chapter 1" {
					t.Errorf("Expected Section 1.1 parent 'Chapter 1', got '%s'", chunks[1].ParentHeader)
				}
				if chunks[2].ParentHeader != "Chapter 1" {
					t.Errorf("Expected Section 1.2 parent 'Chapter 1', got '%s'", chunks[2].ParentHeader)
				}
				// Chapter 2 should have no parent
				if chunks[3].ParentHeader != "" {
					t.Errorf("Expected Chapter 2 to have no parent, got '%s'", chunks[3].ParentHeader)
				}
			},
		},
		{
			name:           "Empty markdown",
			markdown:       "",
			expectedChunks: 0,
			validateChunks: func(t *testing.T, chunks []MarkdownChunk) {},
		},
		{
			name: "Headers with no content",
			markdown: `# Header 1
## Header 2
### Header 3`,
			expectedChunks: 3,
			validateChunks: func(t *testing.T, chunks []MarkdownChunk) {
				for i, chunk := range chunks {
					if chunk.Content != "" {
						t.Errorf("Expected chunk %d to have empty content, got '%s'", i, chunk.Content)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := ParseMarkdownHierarchy(tt.markdown)

			if len(chunks) != tt.expectedChunks {
				t.Errorf("Expected %d chunks, got %d", tt.expectedChunks, len(chunks))
			}

			tt.validateChunks(t, chunks)
		})
	}
}

func TestBuildHierarchy(t *testing.T) {
	tests := []struct {
		name            string
		stack           []MarkdownChunk
		currentHeader   string
		expectedHierarchy string
	}{
		{
			name: "Single level",
			stack: []MarkdownChunk{},
			currentHeader: "Top",
			expectedHierarchy: "Top",
		},
		{
			name: "Two levels",
			stack: []MarkdownChunk{
				{Header: "Chapter"},
			},
			currentHeader: "Section",
			expectedHierarchy: "Chapter > Section",
		},
		{
			name: "Three levels",
			stack: []MarkdownChunk{
				{Header: "Chapter"},
				{Header: "Section"},
			},
			currentHeader: "Subsection",
			expectedHierarchy: "Chapter > Section > Subsection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hierarchy := buildHierarchy(tt.stack, tt.currentHeader)
			if hierarchy != tt.expectedHierarchy {
				t.Errorf("Expected hierarchy '%s', got '%s'", tt.expectedHierarchy, hierarchy)
			}
		})
	}
}

func TestChunkWithMarkdownHierarchy(t *testing.T) {
	tests := []struct {
		name           string
		markdown       string
		expectedChunks int
		validateChunks func(*testing.T, []string)
	}{
		{
			name: "Basic markdown with hierarchy",
			markdown: `# Chapter 1
This is the introduction.

## Section 1.1
This is a section.`,
			expectedChunks: 2,
			validateChunks: func(t *testing.T, chunks []string) {
				// First chunk should contain TITLE, HIERARCHY, and CONTENT
				if !strings.Contains(chunks[0], "TITLE: # Chapter 1") {
					t.Error("First chunk should contain 'TITLE: # Chapter 1'")
				}
				if !strings.Contains(chunks[0], "HIERARCHY: Chapter 1") {
					t.Error("First chunk should contain 'HIERARCHY: Chapter 1'")
				}
				if !strings.Contains(chunks[0], "CONTENT: This is the introduction") {
					t.Error("First chunk should contain 'CONTENT: This is the introduction'")
				}

				// Second chunk should show hierarchy
				if !strings.Contains(chunks[1], "TITLE: ## Section 1.1") {
					t.Error("Second chunk should contain 'TITLE: ## Section 1.1'")
				}
				if !strings.Contains(chunks[1], "HIERARCHY: Chapter 1 > Section 1.1") {
					t.Error("Second chunk should contain proper hierarchy")
				}
			},
		},
		{
			name: "Deep hierarchy",
			markdown: `# Book
Book intro.

## Chapter
Chapter intro.

### Section
Section content.

#### Subsection
Subsection content.`,
			expectedChunks: 4,
			validateChunks: func(t *testing.T, chunks []string) {
				// Last chunk should have full hierarchy path
				lastChunk := chunks[3]
				if !strings.Contains(lastChunk, "HIERARCHY: Book > Chapter > Section > Subsection") {
					t.Errorf("Last chunk should contain full hierarchy, got: %s", lastChunk)
				}
			},
		},
		{
			name:           "No headers",
			markdown:       "Just plain text with no headers.",
			expectedChunks: 0,
			validateChunks: func(t *testing.T, chunks []string) {},
		},
		{
			name: "Headers with special characters",
			markdown: `# Introduction: Getting Started
Content here.

## Part 1.1 - The Beginning
More content.`,
			expectedChunks: 2,
			validateChunks: func(t *testing.T, chunks []string) {
				if !strings.Contains(chunks[0], "TITLE: # Introduction: Getting Started") {
					t.Error("Should handle headers with special characters")
				}
				if !strings.Contains(chunks[1], "TITLE: ## Part 1.1 - The Beginning") {
					t.Error("Should handle headers with special characters")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := ChunkWithMarkdownHierarchy(tt.markdown)

			if len(chunks) != tt.expectedChunks {
				t.Errorf("Expected %d chunks, got %d", tt.expectedChunks, len(chunks))
			}

			tt.validateChunks(t, chunks)
		})
	}
}

func TestChunkWithMarkdownHierarchy_Format(t *testing.T) {
	markdown := `# Main
Main content.`

	chunks := ChunkWithMarkdownHierarchy(markdown)

	if len(chunks) != 1 {
		t.Fatalf("Expected 1 chunk, got %d", len(chunks))
	}

	chunk := chunks[0]

	// Check that chunk has all required parts
	parts := []string{"TITLE:", "HIERARCHY:", "CONTENT:"}
	for _, part := range parts {
		if !strings.Contains(chunk, part) {
			t.Errorf("Chunk should contain '%s', got: %s", part, chunk)
		}
	}

	// Check format structure
	lines := strings.Split(chunk, "\n")
	if len(lines) < 3 {
		t.Errorf("Expected at least 3 lines, got %d", len(lines))
	}

	if !strings.HasPrefix(lines[0], "TITLE:") {
		t.Errorf("First line should start with 'TITLE:', got: %s", lines[0])
	}
	if !strings.HasPrefix(lines[1], "HIERARCHY:") {
		t.Errorf("Second line should start with 'HIERARCHY:', got: %s", lines[1])
	}
	if !strings.HasPrefix(lines[2], "CONTENT:") {
		t.Errorf("Third line should start with 'CONTENT:', got: %s", lines[2])
	}
}

func TestMarkdownChunkStruct(t *testing.T) {
	chunk := MarkdownChunk{
		Header:       "Test Header",
		Content:      "Test Content",
		Level:        2,
		Prefix:       "##",
		ParentLevel:  1,
		ParentHeader: "Parent",
		ParentPrefix: "#",
		Hierarchy:    "Parent > Test Header",
	}

	if chunk.Header != "Test Header" {
		t.Errorf("Expected Header 'Test Header', got %s", chunk.Header)
	}
	if chunk.Level != 2 {
		t.Errorf("Expected Level 2, got %d", chunk.Level)
	}
	if chunk.ParentHeader != "Parent" {
		t.Errorf("Expected ParentHeader 'Parent', got %s", chunk.ParentHeader)
	}
	if chunk.Hierarchy != "Parent > Test Header" {
		t.Errorf("Expected Hierarchy 'Parent > Test Header', got %s", chunk.Hierarchy)
	}
}
