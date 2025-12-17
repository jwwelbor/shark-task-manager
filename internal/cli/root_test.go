package cli

import (
	"testing"
)

func TestRootCommand(t *testing.T) {
	// Test that the root command exists
	if RootCmd == nil {
		t.Fatal("RootCmd should not be nil")
	}

	// Test command properties
	if RootCmd.Use != "shark" {
		t.Errorf("Expected command use to be 'shark', got '%s'", RootCmd.Use)
	}

	if RootCmd.Version != "0.1.0" {
		t.Errorf("Expected version to be '0.1.0', got '%s'", RootCmd.Version)
	}
}

func TestGlobalConfig(t *testing.T) {
	// Test that GlobalConfig exists
	if GlobalConfig == nil {
		t.Fatal("GlobalConfig should not be nil")
	}

	// Test default values
	if GlobalConfig.JSON {
		t.Error("Expected JSON to be false by default")
	}

	if GlobalConfig.NoColor {
		t.Error("Expected NoColor to be false by default")
	}

	if GlobalConfig.Verbose {
		t.Error("Expected Verbose to be false by default")
	}
}

func TestGetDBPath(t *testing.T) {
	// Set a test DB path
	GlobalConfig.DBPath = "test.db"

	path, err := GetDBPath()
	if err != nil {
		t.Fatalf("GetDBPath() returned error: %v", err)
	}

	if path == "" {
		t.Error("GetDBPath() returned empty path")
	}
}
