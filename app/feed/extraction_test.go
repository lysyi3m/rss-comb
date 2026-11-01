package feed

import (
	"strings"
	"testing"
)

func TestExtractResolvesRelativeURLs(t *testing.T) {
	text := strings.Repeat("Enough article text for extraction. ", 40)
	page := `<html><body><article><p>` + text + `<a href="/other">other</a></p><p>` + text + `</p><img src="images/photo.jpg"></article></body></html>`

	content, err := Extract([]byte(page), "https://example.com/blog/post")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "https://example.com/other") {
		t.Errorf("link not resolved: %s", content)
	}
	if !strings.Contains(content, "https://example.com/blog/images/photo.jpg") {
		t.Errorf("image not resolved: %s", content)
	}
}
