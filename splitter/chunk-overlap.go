package splitter

// ChunkText takes a text string and divides it into chunks of a specified size with a given overlap.
// It returns a slice of strings, where each string represents a chunk of the original text.
//
// Parameters:
//   - text: The input text to be chunked.
//   - chunkSize: The size of each chunk.
//   - overlap: The amount of overlap between consecutive chunks.
//
// Returns:
//   - []string: A slice of strings representing the chunks of the original text.
func ChunkText(text string, chunkSize, overlap int) []string {
	chunks := []string{}
	for start := 0; start < len(text); start += chunkSize - overlap {
		end := min(start + chunkSize, len(text))
		chunks = append(chunks, text[start:end])
	}
	return chunks
}
