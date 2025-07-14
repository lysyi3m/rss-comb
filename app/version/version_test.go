package version

import (
	"testing"
)

func TestGetVersion(t *testing.T) {
	// Test with default version
	if GetVersion() != "dev" {
		t.Errorf("Expected 'dev', got '%s'", GetVersion())
	}

	// Test with empty version
	original := Version
	Version = ""
	if GetVersion() != "unknown" {
		t.Errorf("Expected 'unknown', got '%s'", GetVersion())
	}
	Version = original

	// Test with custom version
	Version = "1.2.3"
	if GetVersion() != "1.2.3" {
		t.Errorf("Expected '1.2.3', got '%s'", GetVersion())
	}
	Version = original
}
