package splitter

import (
	"regexp"
	"strings"
)

// ExtractSectionHeader extracts the markdown header from a section
// Returns the header line (e.g., "## Title") or empty string if not found
func ExtractSectionHeader(section string) string {
	if section == "" {
		return ""
	}

	// Regex to match markdown headers at the beginning of the section
	headerRegex := regexp.MustCompile(`(?m)^\s*(#+\s+.*)$`)

	matches := headerRegex.FindStringSubmatch(section)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	return ""
}

// SplitMarkdownBySections splits markdown content by headers (# ## ### etc.)
// Returns a slice where each element contains a section starting with a header
func SplitMarkdownBySections(markdown string) []string {
	if markdown == "" {
		return []string{}
	}

	// Regex to match markdown headers (# ## ### etc. allowing leading whitespace)
	headerRegex := regexp.MustCompile(`(?m)^\s*#+\s+.*$`)

	// Find all header positions
	headerMatches := headerRegex.FindAllStringIndex(markdown, -1)

	if len(headerMatches) == 0 {
		// No headers found, return the entire content as one section
		return []string{strings.TrimSpace(markdown)}
	}

	var sections []string

	// Handle content before first header
	if headerMatches[0][0] > 0 {
		preHeader := strings.TrimSpace(markdown[:headerMatches[0][0]])
		if preHeader != "" {
			sections = append(sections, preHeader)
		}
	}

	// Split by headers
	for i, match := range headerMatches {
		start := match[0]
		var end int

		if i < len(headerMatches)-1 {
			// Not the last header, end at next header
			end = headerMatches[i+1][0]
		} else {
			// Last header, end at document end
			end = len(markdown)
		}

		section := strings.TrimSpace(markdown[start:end])
		if section != "" {
			sections = append(sections, section)
		}
	}

	return sections
}
