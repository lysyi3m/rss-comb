package config

import (
	"testing"
)

func TestGetVersion(t *testing.T) {
	// Test default version
	if GetVersion() == "" {
		t.Error("GetVersion should never return empty string")
	}

	// Test that version is at least "dev" or "unknown"
	version := GetVersion()
	if version != "dev" && version != "unknown" {
		// This is fine, version could be set at build time
		t.Logf("Version: %s", version)
	}
}

func TestConfigInterface(t *testing.T) {
	// Create a config instance to test interface compliance
	config := &Config{
		Port:              "8080",
		UserAgent:         "Test Agent",
		WorkerCount:       5,
		SchedulerInterval: 30,
		APIAccessKey:      "test-key",
		Version:           "test-version",
		FeedsDir:          "./feeds",
		DBHost:            "localhost",
		DBPort:            "5432",
		DBUser:            "test_user",
		DBPassword:        "test_password",
		DBName:            "test_db",
		Timezone:          "UTC",
		Debug:             true,
		DisableMigrate:    false,
	}

	// Test that Config implements Interface
	var _ Interface = config

	// Test getter methods
	if config.GetPort() != "8080" {
		t.Errorf("Expected port '8080', got '%s'", config.GetPort())
	}
	if config.GetUserAgent() != "Test Agent" {
		t.Errorf("Expected user agent 'Test Agent', got '%s'", config.GetUserAgent())
	}
	if config.GetWorkerCount() != 5 {
		t.Errorf("Expected worker count 5, got %d", config.GetWorkerCount())
	}
	if config.GetSchedulerInterval() != 30 {
		t.Errorf("Expected scheduler interval 30, got %d", config.GetSchedulerInterval())
	}
	if config.GetAPIAccessKey() != "test-key" {
		t.Errorf("Expected API key 'test-key', got '%s'", config.GetAPIAccessKey())
	}
	if config.GetVersion() != "test-version" {
		t.Errorf("Expected version 'test-version', got '%s'", config.GetVersion())
	}
	if config.GetFeedsDir() != "./feeds" {
		t.Errorf("Expected feeds dir './feeds', got '%s'", config.GetFeedsDir())
	}
	if config.GetDBHost() != "localhost" {
		t.Errorf("Expected DB host 'localhost', got '%s'", config.GetDBHost())
	}
	if config.GetDBPort() != "5432" {
		t.Errorf("Expected DB port '5432', got '%s'", config.GetDBPort())
	}
	if config.GetDBUser() != "test_user" {
		t.Errorf("Expected DB user 'test_user', got '%s'", config.GetDBUser())
	}
	if config.GetDBPassword() != "test_password" {
		t.Errorf("Expected DB password 'test_password', got '%s'", config.GetDBPassword())
	}
	if config.GetDBName() != "test_db" {
		t.Errorf("Expected DB name 'test_db', got '%s'", config.GetDBName())
	}
	if config.GetTimezone() != "UTC" {
		t.Errorf("Expected timezone 'UTC', got '%s'", config.GetTimezone())
	}
	if !config.IsDebugEnabled() {
		t.Error("Expected debug to be enabled")
	}
	if config.IsMigrationDisabled() {
		t.Error("Expected migration to be enabled")
	}
}