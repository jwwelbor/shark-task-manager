package reporting

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGetJSONSchema(t *testing.T) {
	// Act
	schema := GetJSONSchema()

	// Assert
	if schema == "" {
		t.Fatal("Expected non-empty schema")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(schema), &parsed)
	if err != nil {
		t.Fatalf("Expected valid JSON schema, got error: %v", err)
	}

	// Verify schema metadata
	if parsed["$schema"] != "http://json-schema.org/draft-07/schema#" {
		t.Errorf("Expected JSON Schema draft-07, got %v", parsed["$schema"])
	}
	if parsed["title"] != "Shark Scan Report" {
		t.Errorf("Expected title 'Shark Scan Report', got %v", parsed["title"])
	}
	if parsed["version"] != "1.0" {
		t.Errorf("Expected version '1.0', got %v", parsed["version"])
	}
}

func TestJSONSchema_RequiredFields(t *testing.T) {
	// Arrange
	schema := GetJSONSchema()
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(schema), &parsed)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	// Act
	required := parsed["required"].([]interface{})

	// Assert
	expectedRequired := []string{
		"schema_version",
		"status",
		"dry_run",
		"metadata",
		"counts",
		"entities",
	}

	if len(required) != len(expectedRequired) {
		t.Errorf("Expected %d required fields, got %d", len(expectedRequired), len(required))
	}

	for _, field := range expectedRequired {
		found := false
		for _, r := range required {
			if r.(string) == field {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected required field '%s' not found", field)
		}
	}
}

func TestJSONSchema_Properties(t *testing.T) {
	// Arrange
	schema := GetJSONSchema()
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(schema), &parsed)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	// Act
	properties := parsed["properties"].(map[string]interface{})

	// Assert - Check key properties exist
	expectedProperties := []string{
		"schema_version",
		"status",
		"dry_run",
		"metadata",
		"counts",
		"entities",
		"errors",
		"warnings",
		"skipped_files",
	}

	for _, prop := range expectedProperties {
		if _, exists := properties[prop]; !exists {
			t.Errorf("Expected property '%s' not found in schema", prop)
		}
	}
}

func TestJSONSchema_MetadataStructure(t *testing.T) {
	// Arrange
	schema := GetJSONSchema()
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(schema), &parsed)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	// Act
	properties := parsed["properties"].(map[string]interface{})
	metadata := properties["metadata"].(map[string]interface{})
	metadataProps := metadata["properties"].(map[string]interface{})

	// Assert
	expectedMetadataProps := []string{
		"timestamp",
		"duration_seconds",
		"validation_level",
		"documentation_root",
		"patterns",
	}

	for _, prop := range expectedMetadataProps {
		if _, exists := metadataProps[prop]; !exists {
			t.Errorf("Expected metadata property '%s' not found", prop)
		}
	}
}

func TestJSONSchema_SkippedFileEntryStructure(t *testing.T) {
	// Arrange
	schema := GetJSONSchema()
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(schema), &parsed)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	// Act
	definitions := parsed["definitions"].(map[string]interface{})
	skippedEntry := definitions["SkippedFileEntry"].(map[string]interface{})
	entryProps := skippedEntry["properties"].(map[string]interface{})

	// Assert
	expectedProps := []string{
		"file_path",
		"reason",
		"suggested_fix",
		"error_type",
		"line_number",
	}

	for _, prop := range expectedProps {
		if _, exists := entryProps[prop]; !exists {
			t.Errorf("Expected SkippedFileEntry property '%s' not found", prop)
		}
	}

	// Verify required fields for SkippedFileEntry
	required := skippedEntry["required"].([]interface{})
	expectedRequired := []string{"file_path", "reason", "suggested_fix", "error_type"}

	for _, field := range expectedRequired {
		found := false
		for _, r := range required {
			if r.(string) == field {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected required SkippedFileEntry field '%s' not found", field)
		}
	}
}

func TestJSONSchema_StatusEnum(t *testing.T) {
	// Arrange
	schema := GetJSONSchema()
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(schema), &parsed)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	// Act
	properties := parsed["properties"].(map[string]interface{})
	status := properties["status"].(map[string]interface{})
	enum := status["enum"].([]interface{})

	// Assert
	if len(enum) != 2 {
		t.Errorf("Expected 2 status values, got %d", len(enum))
	}

	expectedValues := []string{"success", "failure"}
	for _, expected := range expectedValues {
		found := false
		for _, e := range enum {
			if e.(string) == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected status value '%s' not found in enum", expected)
		}
	}
}

func TestJSONSchema_ErrorTypeEnum(t *testing.T) {
	// Arrange
	schema := GetJSONSchema()
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(schema), &parsed)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	// Act
	definitions := parsed["definitions"].(map[string]interface{})
	skippedEntry := definitions["SkippedFileEntry"].(map[string]interface{})
	entryProps := skippedEntry["properties"].(map[string]interface{})
	errorType := entryProps["error_type"].(map[string]interface{})
	enum := errorType["enum"].([]interface{})

	// Assert - Verify expected error types
	expectedErrorTypes := []string{
		"parse_error",
		"validation_failure",
		"validation_warning",
		"pattern_mismatch",
		"file_access_error",
	}

	for _, expected := range expectedErrorTypes {
		found := false
		for _, e := range enum {
			if e.(string) == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected error type '%s' not found in enum", expected)
		}
	}
}

func TestJSONSchema_Description(t *testing.T) {
	// Arrange
	schema := GetJSONSchema()

	// Assert
	if !strings.Contains(schema, "description") {
		t.Error("Expected schema to contain field descriptions")
	}
	if !strings.Contains(schema, "Shark Scan Report") {
		t.Error("Expected schema to contain title")
	}
}

func TestJSONSchema_Version(t *testing.T) {
	// Arrange
	schema := GetJSONSchema()
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(schema), &parsed)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	// Assert
	version := parsed["version"]
	if version != "1.0" {
		t.Errorf("Expected schema version '1.0', got '%v'", version)
	}
}
