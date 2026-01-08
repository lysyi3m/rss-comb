package feed

import (
	"fmt"
	"regexp"
	"strings"

	"codeberg.org/readeck/go-readability"
)

var (
	// Remove SVG elements that cause visual noise (icons, logos)
	svgRegex = regexp.MustCompile(`<svg[^>]*>[\s\S]*?</svg>`)
)

func Extract(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("HTML data is empty")
	}

	// Use custom parser with stricter settings to reduce noise
	parser := readability.NewParser()
	parser.CharThresholds = 600    // Increased from default 500 to filter small elements
	parser.KeepClasses = false     // Strip CSS classes to reduce noise
	parser.NTopCandidates = 3      // Reduced from 5 for stricter content selection

	article, err := parser.Parse(strings.NewReader(string(data)), nil)
	if err != nil {
		return "", fmt.Errorf("failed to extract content: %w", err)
	}

	if article.Content == "" {
		return "", fmt.Errorf("no content extracted from HTML data")
	}

	// Post-process to remove SVG noise
	content := svgRegex.ReplaceAllString(article.Content, "")

	return content, nil
}
