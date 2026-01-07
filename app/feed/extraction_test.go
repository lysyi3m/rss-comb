package feed

import (
	"testing"
)

func TestExtract_EmptyData(t *testing.T) {
	result, err := Extract([]byte{})

	if err == nil {
		t.Errorf("Expected error for empty data")
	}

	if result != "" {
		t.Errorf("Expected empty result for empty data")
	}

	expectedError := "HTML data is empty"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestExtract_NilData(t *testing.T) {
	result, err := Extract(nil)

	if err == nil {
		t.Errorf("Expected error for nil data")
	}

	if result != "" {
		t.Errorf("Expected empty result for nil data")
	}

	expectedError := "HTML data is empty"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}
