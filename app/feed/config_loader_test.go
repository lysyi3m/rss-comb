package feed

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_TitleOverride(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, "test-feed.yml", `
url: "https://example.com/feed.xml"
title: "My Custom Title"
enabled: true
`)

	config, _, err := LoadConfig(dir, "test-feed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.Title != "My Custom Title" {
		t.Errorf("expected config.Title = 'My Custom Title', got %q", config.Title)
	}
}

func TestLoadConfig_TitleOmitted(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, "test-feed.yml", `
url: "https://example.com/feed.xml"
enabled: true
`)

	config, _, err := LoadConfig(dir, "test-feed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.Title != "" {
		t.Errorf("expected empty title when omitted, got %q", config.Title)
	}

}

func TestLoadConfig_Defaults(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, "test-feed.yml", `
url: "https://example.com/feed.xml"
enabled: true
`)

	config, _, err := LoadConfig(dir, "test-feed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.Settings.RefreshInterval != 1800 {
		t.Errorf("expected default refresh_interval 1800, got %d", config.Settings.RefreshInterval)
	}
	if config.Settings.MaxItems != 50 {
		t.Errorf("expected default max_items 50, got %d", config.Settings.MaxItems)
	}
	if config.Settings.Timeout != 30 {
		t.Errorf("expected default timeout 30, got %d", config.Settings.Timeout)
	}
}

func TestLoadConfig_NameFromFilename(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, "my-feed.yml", `
url: "https://example.com/feed.xml"
enabled: true
`)

	config, _, err := LoadConfig(dir, "my-feed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.Name != "my-feed" {
		t.Errorf("expected name 'my-feed', got %q", config.Name)
	}
}

func TestLoadConfig_ValidTypes(t *testing.T) {
	for _, typ := range []string{"", "podcast", "youtube"} {
		dir := t.TempDir()
		content := `url: "https://example.com/feed.xml"
enabled: true
`
		if typ != "" {
			content += "type: " + typ + "\n"
		}
		writeTestConfig(t, dir, "test-feed.yml", content)

		_, _, err := LoadConfig(dir, "test-feed")
		if err != nil {
			t.Errorf("expected no error for type %q, got: %v", typ, err)
		}
	}
}

func TestLoadConfig_InvalidType(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, "test-feed.yml", `
url: "https://example.com/feed.xml"
type: invalid
enabled: true
`)

	_, _, err := LoadConfig(dir, "test-feed")
	if err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestLoadConfig_ExtractContentOnlyForBasicType(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, "test-feed.yml", `
url: "https://example.com/feed.xml"
type: podcast
enabled: true
settings:
  extract_content: true
`)

	_, _, err := LoadConfig(dir, "test-feed")
	if err == nil {
		t.Error("expected error for extract_content on non-basic type")
	}
}

func TestLoadConfig_MissingURL(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, "test-feed.yml", `
enabled: true
`)

	_, _, err := LoadConfig(dir, "test-feed")
	if err == nil {
		t.Error("expected error for missing URL")
	}
}

func TestLoadConfig_ConfigHash(t *testing.T) {
	dir := t.TempDir()
	content := `
url: "https://example.com/feed.xml"
enabled: true
`
	writeTestConfig(t, dir, "test-feed.yml", content)

	_, hash1, err := LoadConfig(dir, "test-feed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, hash2, err := LoadConfig(dir, "test-feed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hash1 != hash2 {
		t.Error("expected same hash for identical config")
	}

	writeTestConfig(t, dir, "test-feed.yml", content+"\n# changed")

	_, hash3, err := LoadConfig(dir, "test-feed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hash1 == hash3 {
		t.Error("expected different hash for changed config")
	}
}

func writeTestConfig(t *testing.T, dir, filename, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
}
