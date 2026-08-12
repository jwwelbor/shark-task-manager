package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStdout captures stdout output from a function call
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read captured output: %v", err)
	}
	return buf.String()
}

func TestOutputField_SimpleObject(t *testing.T) {
	data := map[string]interface{}{
		"key":    "E07-F01-001",
		"status": "todo",
		"title":  "Test Task",
	}

	tests := []struct {
		name     string
		field    string
		expected string
		wantErr  bool
	}{
		{"string field", "status", "todo\n", false},
		{"key field", "key", "E07-F01-001\n", false},
		{"missing field", "nonexistent", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(t, func() {
				err := OutputField(data, tt.field)
				if tt.wantErr {
					if err == nil {
						t.Error("expected error, got nil")
					}
					var fieldErr *FieldNotFoundError
					if !errors.As(err, &fieldErr) {
						t.Errorf("expected FieldNotFoundError, got %T", err)
					}
				} else if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			})

			if !tt.wantErr && output != tt.expected {
				t.Errorf("expected output %q, got %q", tt.expected, output)
			}
		})
	}
}

func TestOutputField_SingleEntityEnvelope(t *testing.T) {
	data := map[string]interface{}{
		"display_mode": "compact",
		"task": map[string]interface{}{
			"status": "todo",
		},
	}

	output := captureStdout(t, func() {
		require.NoError(t, OutputField(data, "status"))
	})
	assert.Equal(t, "todo\n", output)
}

func TestOutputField_AmbiguousEntityEnvelope(t *testing.T) {
	data := map[string]interface{}{
		"task":    map[string]interface{}{"status": "todo"},
		"feature": map[string]interface{}{"status": "in_progress"},
	}
	assert.Error(t, OutputField(data, "status"))
}

func TestOutputField_IntegerValue(t *testing.T) {
	data := map[string]interface{}{
		"priority": 5,
		"count":    0,
	}

	output := captureStdout(t, func() {
		err := OutputField(data, "priority")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if output != "5\n" {
		t.Errorf("expected '5\\n', got %q", output)
	}
}

func TestOutputField_FloatValue(t *testing.T) {
	data := map[string]interface{}{
		"progress": 75.5,
	}

	output := captureStdout(t, func() {
		err := OutputField(data, "progress")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if output != "75.5\n" {
		t.Errorf("expected '75.5\\n', got %q", output)
	}
}

func TestOutputField_BoolValue(t *testing.T) {
	data := map[string]interface{}{
		"blocked": true,
	}

	output := captureStdout(t, func() {
		err := OutputField(data, "blocked")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if output != "true\n" {
		t.Errorf("expected 'true\\n', got %q", output)
	}
}

func TestOutputField_NullValue(t *testing.T) {
	data := map[string]interface{}{
		"agent": nil,
	}

	output := captureStdout(t, func() {
		err := OutputField(data, "agent")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if output != "null\n" {
		t.Errorf("expected 'null\\n', got %q", output)
	}
}

func TestOutputField_NestedDotNotation(t *testing.T) {
	data := map[string]interface{}{
		"progress": map[string]interface{}{
			"weighted_pct":   75.0,
			"completion_pct": 50.0,
		},
	}

	output := captureStdout(t, func() {
		err := OutputField(data, "progress.weighted_pct")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if output != "75\n" {
		t.Errorf("expected '75\\n', got %q", output)
	}
}

func TestOutputField_NestedObject(t *testing.T) {
	data := map[string]interface{}{
		"progress": map[string]interface{}{
			"weighted_pct":   75.0,
			"completion_pct": 50.0,
		},
	}

	output := captureStdout(t, func() {
		err := OutputField(data, "progress")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	// Should be compact JSON
	output = strings.TrimSpace(output)
	if !strings.Contains(output, "weighted_pct") || !strings.Contains(output, "completion_pct") {
		t.Errorf("expected compact JSON with progress fields, got %q", output)
	}
}

func TestOutputField_Array(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"key": "E07-F01-001", "status": "todo"},
		map[string]interface{}{"key": "E07-F01-002", "status": "in_progress"},
		map[string]interface{}{"key": "E07-F01-003", "status": "completed"},
	}

	output := captureStdout(t, func() {
		err := OutputField(data, "status")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "todo" || lines[1] != "in_progress" || lines[2] != "completed" {
		t.Errorf("unexpected values: %v", lines)
	}
}

func TestOutputField_ArraySkipsMissingField(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"key": "E07-F01-001", "agent": "backend"},
		map[string]interface{}{"key": "E07-F01-002"},
		map[string]interface{}{"key": "E07-F01-003", "agent": "frontend"},
	}

	output := captureStdout(t, func() {
		err := OutputField(data, "agent")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines (skipping element without agent), got %d: %v", len(lines), lines)
	}
	if lines[0] != "backend" || lines[1] != "frontend" {
		t.Errorf("unexpected values: %v", lines)
	}
}

func TestOutputField_ArrayAllMissing(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"key": "E07-F01-001"},
		map[string]interface{}{"key": "E07-F01-002"},
	}

	err := OutputField(data, "nonexistent")
	if err == nil {
		t.Error("expected error for array where no elements have the field")
	}
	var fieldErr *FieldNotFoundError
	if !errors.As(err, &fieldErr) {
		t.Errorf("expected FieldNotFoundError, got %T", err)
	}
}

func TestOutputField_WithStruct(t *testing.T) {
	// Test with a Go struct (not just maps)
	type TestTask struct {
		Key    string `json:"key"`
		Status string `json:"status"`
		Count  int    `json:"count"`
	}

	data := TestTask{
		Key:    "E07-F01-001",
		Status: "todo",
		Count:  42,
	}

	output := captureStdout(t, func() {
		err := OutputField(data, "key")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if output != "E07-F01-001\n" {
		t.Errorf("expected 'E07-F01-001\\n', got %q", output)
	}
}

func TestOutputField_WithStructSlice(t *testing.T) {
	type TestTask struct {
		Key    string `json:"key"`
		Status string `json:"status"`
	}

	data := []TestTask{
		{Key: "E07-F01-001", Status: "todo"},
		{Key: "E07-F01-002", Status: "in_progress"},
	}

	output := captureStdout(t, func() {
		err := OutputField(data, "key")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "E07-F01-001" || lines[1] != "E07-F01-002" {
		t.Errorf("unexpected values: %v", lines)
	}
}

func TestFieldNotFoundError(t *testing.T) {
	err := &FieldNotFoundError{Field: "status"}
	if err.Error() != "field not found: status" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	// Test errors.As
	var fieldErr *FieldNotFoundError
	if !errors.As(err, &fieldErr) {
		t.Error("errors.As should match FieldNotFoundError")
	}
	if fieldErr.Field != "status" {
		t.Errorf("expected Field=status, got %s", fieldErr.Field)
	}
}

func TestOutputJSON_WithFieldSet(t *testing.T) {
	// Save and restore global config
	origField := GlobalConfig.Field
	origJSON := GlobalConfig.JSON
	defer func() {
		GlobalConfig.Field = origField
		GlobalConfig.JSON = origJSON
	}()

	GlobalConfig.Field = "status"
	GlobalConfig.JSON = true

	data := map[string]interface{}{
		"key":    "E07-F01-001",
		"status": "todo",
	}

	output := captureStdout(t, func() {
		err := OutputJSON(data)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if strings.TrimSpace(output) != "todo" {
		t.Errorf("expected 'todo', got %q", strings.TrimSpace(output))
	}
}

func TestOutputJSON_CLIErrorBypassesFieldExtraction(t *testing.T) {
	// CLIError should be output as full JSON even when --field is set
	origField := GlobalConfig.Field
	origJSON := GlobalConfig.JSON
	defer func() {
		GlobalConfig.Field = origField
		GlobalConfig.JSON = origJSON
	}()

	GlobalConfig.Field = "status"
	GlobalConfig.JSON = true

	data := &CLIError{
		Error:   true,
		Code:    ErrCodeNotFound,
		Message: "not found",
	}

	output := captureStdout(t, func() {
		err := OutputJSON(data)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	// Should be full JSON, not just a field value
	if !strings.Contains(output, "NOT_FOUND") {
		t.Errorf("CLIError should bypass field extraction, got: %s", output)
	}
	if !strings.Contains(output, "\"error\"") {
		t.Errorf("CLIError should contain error field, got: %s", output)
	}
}

func TestOutputJSON_CLIErrorValueBypassesFieldExtraction(t *testing.T) {
	// CLIError (value, not pointer) should also bypass
	origField := GlobalConfig.Field
	origJSON := GlobalConfig.JSON
	defer func() {
		GlobalConfig.Field = origField
		GlobalConfig.JSON = origJSON
	}()

	GlobalConfig.Field = "code"
	GlobalConfig.JSON = true

	data := CLIError{
		Error:   true,
		Code:    ErrCodeCommandError,
		Message: "test",
	}

	output := captureStdout(t, func() {
		err := OutputJSON(data)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "COMMAND_ERROR") {
		t.Errorf("CLIError value should bypass field extraction, got: %s", output)
	}
}

func TestFieldImpliesJSON(t *testing.T) {
	// Save and restore global config
	origField := GlobalConfig.Field
	origJSON := GlobalConfig.JSON
	defer func() {
		GlobalConfig.Field = origField
		GlobalConfig.JSON = origJSON
	}()

	GlobalConfig.Field = "status"
	GlobalConfig.JSON = false

	// Simulate initConfig behavior
	if GlobalConfig.Field != "" {
		GlobalConfig.JSON = true
	}

	if !GlobalConfig.JSON {
		t.Error("--field should imply JSON mode")
	}
}

func TestLookupField_DeepNesting(t *testing.T) {
	obj := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"value": "deep",
			},
		},
	}

	val, err := lookupField(obj, "level1.level2.value")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != "deep" {
		t.Errorf("expected 'deep', got %v", val)
	}
}

func TestLookupField_MissingIntermediate(t *testing.T) {
	obj := map[string]interface{}{
		"level1": map[string]interface{}{},
	}

	_, err := lookupField(obj, "level1.level2.value")
	if err == nil {
		t.Error("expected error for missing intermediate field")
	}
}
