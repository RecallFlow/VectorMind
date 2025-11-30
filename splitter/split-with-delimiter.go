package splitter

import "strings"

// SplitTextWithDelimiter splits the given text using the specified delimiter and returns a slice of strings.
//
// Parameters:
//   - text: The text to be split.
//   - delimiter: The delimiter used to split the text.
//
// Returns:
//   - []string: A slice of strings containing the split parts of the text.
func SplitTextWithDelimiter(text string, delimiter string) []string {
	return strings.Split(text, delimiter)
}

// ExtractFirstNonEmptyLines extracts the first N non-empty lines from a text chunk
// Returns a string with the extracted lines joined by newlines
func ExtractFirstNonEmptyLines(text string, n int) string {
	if text == "" || n <= 0 {
		return ""
	}

	lines := strings.Split(text, "\n")
	nonEmptyLines := make([]string, 0, n)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			nonEmptyLines = append(nonEmptyLines, trimmed)
			if len(nonEmptyLines) >= n {
				break
			}
		}
	}

	return strings.Join(nonEmptyLines, "\n")
}
