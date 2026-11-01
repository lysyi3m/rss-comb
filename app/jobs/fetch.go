package jobs

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxResponseBytes = 64 << 20

// fetchURL returns the response body and the URL it came from after redirects.
func fetchURL(ctx context.Context, url string, timeout int, httpClient *http.Client, userAgent string, requireHTML bool) ([]byte, string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, "GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	if requireHTML {
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(strings.ToLower(contentType), "text/html") {
			return nil, "", fmt.Errorf("content type is not HTML: %s", contentType)
		}
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}
	if len(data) > maxResponseBytes {
		return nil, "", fmt.Errorf("response body exceeds %d bytes", maxResponseBytes)
	}

	return data, resp.Request.URL.String(), nil
}
