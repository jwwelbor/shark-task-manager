package models

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrInvalidSize is returned when a size value is not in the canonical
// Fibonacci set {1, 2, 3, 5, 8, 13} or its label form {XS, S, M, L, XL, XXL}.
var ErrInvalidSize = errors.New(
	"invalid size: must be one of 1, 2, 3, 5, 8, 13 (or XS, S, M, L, XL, XXL)")

// maxSizeInputLen is the maximum accepted input length for ParseSize.
// Inputs longer than this are rejected to prevent oversized payloads.
const maxSizeInputLen = 20

// CanonicalSizes is the allowed numeric set, in ascending order.
// Exported so CLI help-text and documentation generators can iterate it.
var CanonicalSizes = []int{1, 2, 3, 5, 8, 13}

// sizeLabels maps numeric values to their canonical t-shirt labels.
var sizeLabels = map[int]string{
	1: "XS", 2: "S", 3: "M", 5: "L", 8: "XL", 13: "XXL",
}

// labelToSize is the inverse of sizeLabels, normalized to uppercase keys.
var labelToSize = map[string]int{
	"XS": 1, "S": 2, "M": 3, "L": 5, "XL": 8, "XXL": 13,
}

// validSize is an O(1) allowlist for canonical numeric values.
var validSize = map[int]bool{1: true, 2: true, 3: true, 5: true, 8: true, 13: true}

// ParseSize accepts either a t-shirt label ("XS"–"XXL", case-insensitive)
// or a numeric string ("1", "2", "3", "5", "8", "13") and returns the
// canonical numeric value. Whitespace is trimmed. Empty input is an error.
// Inputs longer than 20 characters are rejected without ambiguity.
//
// Follows .claude/rules/go/input-sanitization.md:
//   - strings.TrimSpace on input before parsing.
//   - Case-insensitive label match via strings.ToUpper.
//   - Errors quote user input with %q to prevent log injection.
//   - Oversized inputs (>20 chars) rejected explicitly.
func ParseSize(input string) (int, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return 0, fmt.Errorf("%w: input cannot be empty", ErrInvalidSize)
	}
	if len(trimmed) > maxSizeInputLen {
		return 0, fmt.Errorf("%w: got %q", ErrInvalidSize, input)
	}
	upper := strings.ToUpper(trimmed)
	if v, ok := labelToSize[upper]; ok {
		return v, nil
	}
	n, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("%w: got %q", ErrInvalidSize, input)
	}
	if !validSize[n] {
		return 0, fmt.Errorf("%w: got %q", ErrInvalidSize, input)
	}
	return n, nil
}

// SizeLabel returns the canonical t-shirt label for a numeric size.
// Returns ("", error) if the input is not in the canonical set.
func SizeLabel(n int) (string, error) {
	if label, ok := sizeLabels[n]; ok {
		return label, nil
	}
	return "", fmt.Errorf("%w: got %d", ErrInvalidSize, n)
}

// ValidateSize returns nil if n is in the canonical Fibonacci set
// {1, 2, 3, 5, 8, 13}. Use this in entity .Validate() methods for
// structural checks.
//
// This is an allowlist-based validator per .claude/rules/go/patterns.md
// (two-level validation: structural check here, workflow-aware checks
// at the service layer).
func ValidateSize(n int) error {
	if !validSize[n] {
		return fmt.Errorf("%w: got %d", ErrInvalidSize, n)
	}
	return nil
}
