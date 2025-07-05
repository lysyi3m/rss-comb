package database

import (
	"testing"
)

func TestNewConnection(t *testing.T) {
	// Test with invalid connection parameters
	_, err := NewConnection("invalid", "invalid", "invalid", "invalid", "invalid")
	if err == nil {
		t.Error("Expected error for invalid connection parameters")
	}

	// Note: We don't test valid connection here as it requires running database
	// Integration tests should be run separately with proper test database
}