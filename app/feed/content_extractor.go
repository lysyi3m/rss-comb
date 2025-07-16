package feed

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mackee/go-readability"
)

// ContentExtractor handles extracting full content from article URLs
type ContentExtractor struct {
	client    *http.Client
	userAgent string
}

// NewContentExtractor creates a new content extractor with the given timeout and user agent
func NewContentExtractor(timeout time.Duration, userAgent string) *ContentExtractor {
	return &ContentExtractor{
		client: &http.Client{
			Timeout: timeout,
		},
		userAgent: userAgent,
	}
}

// ExtractContent extracts readable content from the given URL
func (e *ContentExtractor) ExtractContent(ctx context.Context, url string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("URL is empty")
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent to identify ourselves
	req.Header.Set("User-Agent", e.userAgent)

	// Make the request
	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response is successful
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Check content type - we only want HTML
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "text/html") {
		return "", fmt.Errorf("content type is not HTML: %s", contentType)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Extract content using readability
	options := readability.DefaultOptions()
	options.CharThreshold = 250 // Minimum 250 characters for article
	options.NbTopCandidates = 5 // Consider top 5 candidates
	
	article, err := readability.Extract(string(body), options)
	if err != nil {
		return "", fmt.Errorf("failed to extract content: %w", err)
	}

	// Get the HTML content
	content := readability.ToHTML(article.Root)
	if content == "" {
		return "", fmt.Errorf("no content extracted from URL")
	}

	// Log successful extraction
	slog.Debug("Content extracted successfully", 
		"url", url, 
		"title", article.Title,
		"content_length", len(content))

	return content, nil
}

