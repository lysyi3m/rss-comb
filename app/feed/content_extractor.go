package feed

import (
	"fmt"
	"strings"

	"github.com/go-shiori/go-readability"
)

type ContentExtractor struct{}

func NewContentExtractor() *ContentExtractor {
	return &ContentExtractor{}
}

func (e *ContentExtractor) Run(data []byte) (string, error) {
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
