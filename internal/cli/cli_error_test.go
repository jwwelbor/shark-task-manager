package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"
)

func TestCLIError_JSONSerialization(t *testing.T) {
	e := CLIError{
		Error:   true,
		Code:    ErrCodeNotFound,
		Message: "task not found: E07-F01-999",
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var parsed CLIError
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if parsed.Error != true {
		t.Errorf("expected Error=true, got %v", parsed.Error)
	}
	if parsed.Code != ErrCodeNotFound {
		t.Errorf("expected Code=%s, got %s", ErrCodeNotFound, parsed.Code)
	}
	if parsed.Message != "task not found: E07-F01-999" {
		t.Errorf("expected Message='task not found: E07-F01-999', got %s", parsed.Message)
	}
}

func TestCLIError_JSONSerialization_WithOptionalFields(t *testing.T) {
	e := CLIError{
		Error:            true,
		Code:             ErrCodeInvalidTransition,
		Message:          "cannot transition from todo to completed",
		Entity:           "task",
		EntityKey:        "E07-F01-001",
		CurrentStatus:    "todo",
		ValidTransitions: []string{"in_progress"},
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var parsed CLIError
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if parsed.Entity != "task" {
		t.Errorf("expected Entity=task, got %s", parsed.Entity)
	}
	if parsed.EntityKey != "E07-F01-001" {
		t.Errorf("expected EntityKey=E07-F01-001, got %s", parsed.EntityKey)
	}
	if parsed.CurrentStatus != "todo" {
		t.Errorf("expected CurrentStatus=todo, got %s", parsed.CurrentStatus)
	}
	if len(parsed.ValidTransitions) != 1 || parsed.ValidTransitions[0] != "in_progress" {
		t.Errorf("expected ValidTransitions=[in_progress], got %v", parsed.ValidTransitions)
	}
}

func TestCLIError_OmitsEmptyOptionalFields(t *testing.T) {
	e := CLIError{
		Error:   true,
		Code:    ErrCodeCommandError,
		Message: "something went wrong",
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Check that optional fields are omitted from JSON
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	for _, field := range []string{"entity", "entity_key", "current_status", "valid_transitions"} {
		if _, exists := raw[field]; exists {
			t.Errorf("expected field %s to be omitted, but it was present", field)
		}
	}

	// Verify required fields are present
	for _, field := range []string{"error", "code", "message"} {
		if _, exists := raw[field]; !exists {
			t.Errorf("expected field %s to be present, but it was missing", field)
		}
	}
}

func TestErrorJSON_InJSONMode(t *testing.T) {
	// Save and restore global config
	origJSON := GlobalConfig.JSON
	defer func() { GlobalConfig.JSON = origJSON }()

	GlobalConfig.JSON = true

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ErrorJSON(CLIError{
		Code:    ErrCodeNotFound,
		Message: "task not found",
	})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read captured output: %v", err)
	}

	var parsed CLIError
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nRaw: %s", err, buf.String())
	}

	if !parsed.Error {
		t.Error("expected Error=true")
	}
	if parsed.Code != ErrCodeNotFound {
		t.Errorf("expected Code=%s, got %s", ErrCodeNotFound, parsed.Code)
	}
	if parsed.Message != "task not found" {
		t.Errorf("expected Message='task not found', got %s", parsed.Message)
	}
}

func TestErrorJSON_SetsErrorTrue(t *testing.T) {
	// Verify ErrorJSON always sets Error=true even if caller sets false
	origJSON := GlobalConfig.JSON
	defer func() { GlobalConfig.JSON = origJSON }()

	GlobalConfig.JSON = true

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ErrorJSON(CLIError{
		Error:   false, // Intentionally false
		Code:    ErrCodeCommandError,
		Message: "test",
	})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read captured output: %v", err)
	}

	var parsed CLIError
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if !parsed.Error {
		t.Error("ErrorJSON should always set Error=true")
	}
}

func TestErrorJSON_InHumanMode(t *testing.T) {
	// In human mode, ErrorJSON should not output JSON to stdout
	origJSON := GlobalConfig.JSON
	origNoColor := GlobalConfig.NoColor
	defer func() {
		GlobalConfig.JSON = origJSON
		GlobalConfig.NoColor = origNoColor
	}()

	GlobalConfig.JSON = false
	GlobalConfig.NoColor = true // Use simple format for testing

	// Capture stderr (Error() writes to stderr in human mode)
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ErrorJSON(CLIError{
		Code:    ErrCodeNotFound,
		Message: "task not found",
	})

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read captured output: %v", err)
	}

	output := buf.String()
	// Should contain the message in human format
	if len(output) == 0 {
		t.Error("expected output on stderr in human mode")
	}
}

func TestError_InJSONMode(t *testing.T) {
	// Error() should output JSON when in JSON mode
	origJSON := GlobalConfig.JSON
	defer func() { GlobalConfig.JSON = origJSON }()

	GlobalConfig.JSON = true

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Error("something failed")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read captured output: %v", err)
	}

	var parsed CLIError
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("Error() in JSON mode should output valid JSON: %v\nRaw: %s", err, buf.String())
	}

	if !parsed.Error {
		t.Error("expected Error=true")
	}
	if parsed.Code != ErrCodeCommandError {
		t.Errorf("expected Code=%s, got %s", ErrCodeCommandError, parsed.Code)
	}
	if parsed.Message != "something failed" {
		t.Errorf("expected Message='something failed', got %s", parsed.Message)
	}
}

func TestErrorCodeConstants(t *testing.T) {
	// Verify error code constants are distinct and non-empty
	codes := []string{
		ErrCodeNotFound,
		ErrCodeInvalidTransition,
		ErrCodeValidationError,
		ErrCodeDatabaseError,
		ErrCodeInvalidArgs,
		ErrCodeCommandError,
	}

	seen := make(map[string]bool)
	for _, code := range codes {
		if code == "" {
			t.Error("error code constant should not be empty")
		}
		if seen[code] {
			t.Errorf("duplicate error code constant: %s", code)
		}
		seen[code] = true
	}
}
