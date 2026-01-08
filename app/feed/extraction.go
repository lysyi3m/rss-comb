package feed

import (
	"fmt"
	"strings"

	"codeberg.org/readeck/go-readability"
)

func Extract(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("HTML data is empty")
	}

	article, err := readability.FromReader(strings.NewReader(string(data)), nil)
	if err != nil {
		return "", fmt.Errorf("failed to extract content: %w", err)
	}

	if article.Content == "" {
		return "", fmt.Errorf("no content extracted from HTML data")
	}

	return article.Content, nil
}
