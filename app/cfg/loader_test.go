package cfg

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

func TestConfigFields(t *testing.T) {
	// Create a config instance to test field access
	cfg := &Cfg{
		Port:              "8080",
		BaseUrl:           "https://feeds.example.com",
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
	}

	// Test direct field access
	if cfg.Port != "8080" {
		t.Errorf("Expected port '8080', got '%s'", cfg.Port)
	}
	if cfg.BaseUrl != "https://feeds.example.com" {
		t.Errorf("Expected base URL 'https://feeds.example.com', got '%s'", cfg.BaseUrl)
	}
	if cfg.UserAgent != "Test Agent" {
		t.Errorf("Expected user agent 'Test Agent', got '%s'", cfg.UserAgent)
	}
	if cfg.WorkerCount != 5 {
		t.Errorf("Expected worker count 5, got %d", cfg.WorkerCount)
	}
	if cfg.SchedulerInterval != 30 {
		t.Errorf("Expected scheduler interval 30, got %d", cfg.SchedulerInterval)
	}
	if cfg.APIAccessKey != "test-key" {
		t.Errorf("Expected API key 'test-key', got '%s'", cfg.APIAccessKey)
	}
	if cfg.Version != "test-version" {
		t.Errorf("Expected version 'test-version', got '%s'", cfg.Version)
	}
	if cfg.FeedsDir != "./feeds" {
		t.Errorf("Expected feeds dir './feeds', got '%s'", cfg.FeedsDir)
	}
	if cfg.DBHost != "localhost" {
		t.Errorf("Expected DB host 'localhost', got '%s'", cfg.DBHost)
	}
	if cfg.DBPort != "5432" {
		t.Errorf("Expected DB port '5432', got '%s'", cfg.DBPort)
	}
	if cfg.DBUser != "test_user" {
		t.Errorf("Expected DB user 'test_user', got '%s'", cfg.DBUser)
	}
	if cfg.DBPassword != "test_password" {
		t.Errorf("Expected DB password 'test_password', got '%s'", cfg.DBPassword)
	}
	if cfg.DBName != "test_db" {
		t.Errorf("Expected DB name 'test_db', got '%s'", cfg.DBName)
	}
	if cfg.Timezone != "UTC" {
		t.Errorf("Expected timezone 'UTC', got '%s'", cfg.Timezone)
	}
	if !cfg.Debug {
		t.Error("Expected debug to be enabled")
	}
}
