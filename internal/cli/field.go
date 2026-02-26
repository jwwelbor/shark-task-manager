package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// FieldNotFoundError indicates that the requested field was not found in the data.
type FieldNotFoundError struct {
	Field string
}

func (e *FieldNotFoundError) Error() string {
	return fmt.Sprintf("field not found: %s", e.Field)
}

// OutputField extracts a single field from data and prints its value.
// For arrays, it prints one value per line, skipping elements that don't have the field.
// Returns FieldNotFoundError if the field is not found in any element.
func OutputField(data interface{}, field string) error {
	// Marshal to JSON and back to get a generic map/slice
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	var generic interface{}
	if err := json.Unmarshal(jsonBytes, &generic); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return extractAndPrintField(generic, field)
}

// extractAndPrintField handles both single objects and arrays.
func extractAndPrintField(data interface{}, field string) error {
	switch v := data.(type) {
	case []interface{}:
		found := false
		for _, item := range v {
			obj, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			val, err := lookupField(obj, field)
			if err != nil {
				continue // skip elements missing the field
			}
			printFieldValue(val)
			found = true
		}
		if !found {
			return &FieldNotFoundError{Field: field}
		}
		return nil

	case map[string]interface{}:
		val, err := lookupField(v, field)
		if err != nil {
			return &FieldNotFoundError{Field: field}
		}
		printFieldValue(val)
		return nil

	default:
		return &FieldNotFoundError{Field: field}
	}
}

// lookupField supports dot-notation for nested field access (e.g., "progress.weighted_pct").
func lookupField(obj map[string]interface{}, field string) (interface{}, error) {
	parts := strings.Split(field, ".")
	var current interface{} = obj

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("not an object at %s", part)
		}
		val, exists := m[part]
		if !exists {
			return nil, fmt.Errorf("field %s not found", part)
		}
		current = val
	}

	return current, nil
}

// printFieldValue prints a field value in a human-friendly format:
// - strings: bare (no quotes)
// - integers (float64 with no fractional part): no decimal
// - floats: as-is
// - booleans: true/false
// - null: "null"
// - nested objects/arrays: compact JSON
func printFieldValue(val interface{}) {
	switch v := val.(type) {
	case string:
		fmt.Fprintln(os.Stdout, v)
	case float64:
		if v == float64(int64(v)) {
			fmt.Fprintln(os.Stdout, int64(v))
		} else {
			fmt.Fprintln(os.Stdout, v)
		}
	case bool:
		fmt.Fprintln(os.Stdout, v)
	case nil:
		fmt.Fprintln(os.Stdout, "null")
	default:
		// Nested object or array: compact JSON
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			fmt.Fprintln(os.Stdout, v)
		} else {
			fmt.Fprintln(os.Stdout, string(jsonBytes))
		}
	}
}
