package patterns

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Placeholder represents a parsed placeholder from a generation format
type Placeholder struct {
	Full       string // Full placeholder text, e.g., "{number:02d}"
	Field      string // Field name, e.g., "number"
	Format     string // Format specification, e.g., "02d"
	FormatType string // Format type, e.g., "d" for decimal
}

// ParsePlaceholder parses a placeholder string into its components
// Returns: field name, format spec, format type
func ParsePlaceholder(placeholder string) (string, string, string) {
	// Remove braces
	inner := strings.Trim(placeholder, "{}")

	// Split on colon to separate field from format
	parts := strings.SplitN(inner, ":", 2)
	field := parts[0]

	if len(parts) == 1 {
		// No format specified
		return field, "", ""
	}

	format := parts[1]
	formatType := ""

	// Extract format type (last character: d, s, etc.)
	if len(format) > 0 {
		lastChar := format[len(format)-1:]
		if lastChar == "d" || lastChar == "s" || lastChar == "x" {
			formatType = lastChar
		}
	}

	return field, format, formatType
}

// ExtractPlaceholders extracts all placeholders from a format string
func ExtractPlaceholders(format string) []string {
	placeholderRegex := regexp.MustCompile(`\{[^}]+\}`)
	return placeholderRegex.FindAllString(format, -1)
}

// ApplyGenerationFormat applies a generation format template with provided values
func ApplyGenerationFormat(format string, values map[string]interface{}) (string, error) {
	// Extract all placeholders
	placeholders := ExtractPlaceholders(format)

	// Validate format has balanced braces
	openCount := strings.Count(format, "{")
	closeCount := strings.Count(format, "}")
	if openCount != closeCount {
		return "", fmt.Errorf("invalid format template: unbalanced braces")
	}

	result := format

	// Replace each placeholder
	for _, ph := range placeholders {
		field, formatSpec, formatType := ParsePlaceholder(ph)

		// Get value for this field
		value, exists := values[field]
		if !exists {
			return "", fmt.Errorf("missing value for placeholder '%s'", field)
		}

		// Format the value
		formatted, err := formatValue(value, formatSpec, formatType)
		if err != nil {
			return "", fmt.Errorf("error formatting placeholder '%s': %w", ph, err)
		}

		// Replace placeholder with formatted value
		result = strings.Replace(result, ph, formatted, 1)
	}

	return result, nil
}

// formatValue formats a value according to the format specification
func formatValue(value interface{}, formatSpec, formatType string) (string, error) {
	if formatSpec == "" {
		// No format spec, use default string conversion
		return fmt.Sprintf("%v", value), nil
	}

	switch formatType {
	case "d":
		// Decimal integer format
		var intVal int
		switch v := value.(type) {
		case int:
			intVal = v
		case int64:
			intVal = int(v)
		case string:
			parsed, err := strconv.Atoi(v)
			if err != nil {
				return "", fmt.Errorf("cannot convert '%v' to integer", value)
			}
			intVal = parsed
		default:
			return "", fmt.Errorf("cannot format type %T as integer", value)
		}

		// Use Go's fmt with the format spec
		return fmt.Sprintf("%0"+formatSpec[0:len(formatSpec)-1]+"d", intVal), nil

	case "s":
		// String format
		return fmt.Sprintf("%v", value), nil

	default:
		// Unknown format type, use default
		return fmt.Sprintf("%v", value), nil
	}
}

// SanitizeSlug validates and sanitizes a slug value to prevent path traversal
func SanitizeSlug(slug string) (string, error) {
	// Define forbidden characters that could be used for path traversal or injection
	forbiddenChars := []string{
		"/",  // Forward slash
		"\\", // Backslash
		"..", // Parent directory
		":",  // Drive separator (Windows)
		"*",  // Wildcard
		"?",  // Wildcard
		"<",  // Redirect
		">",  // Redirect
		"|",  // Pipe
		"\"", // Quote
		"\n", // Newline
		"\r", // Carriage return
		"\t", // Tab
	}

	// Check for forbidden characters
	for _, forbidden := range forbiddenChars {
		if strings.Contains(slug, forbidden) {
			return "", fmt.Errorf("invalid slug: contains forbidden characters (%s). Forbidden characters: %s",
				forbidden, strings.Join(forbiddenChars, ", "))
		}
	}

	// Additional check: slug should not start with a dot (hidden files)
	if strings.HasPrefix(slug, ".") {
		return "", fmt.Errorf("invalid slug: cannot start with '.' (hidden files not allowed)")
	}

	// Slug should not be empty
	if strings.TrimSpace(slug) == "" {
		return "", fmt.Errorf("invalid slug: cannot be empty")
	}

	return slug, nil
}

// ValidatePathWithinProject ensures a path remains within project boundaries
func ValidatePathWithinProject(path, projectRoot string) error {
	// Clean and resolve both paths
	cleanPath := filepath.Clean(path)
	cleanRoot := filepath.Clean(projectRoot)

	// Convert to absolute paths
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	absRoot, err := filepath.Abs(cleanRoot)
	if err != nil {
		return fmt.Errorf("failed to resolve project root: %w", err)
	}

	// Check if path is within project root
	relPath, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return fmt.Errorf("failed to calculate relative path: %w", err)
	}

	// If relative path starts with "..", it's outside the project
	if strings.HasPrefix(relPath, "..") {
		return fmt.Errorf("path '%s' is outside project boundaries (project root: '%s')", path, projectRoot)
	}

	return nil
}

// GenerateEntityName generates a name for an entity using the generation format
func GenerateEntityName(formatTemplate string, entityType string, number int, slug string, parentContext map[string]int) (string, error) {
	// Sanitize slug if provided
	if slug != "" {
		sanitized, err := SanitizeSlug(slug)
		if err != nil {
			return "", err
		}
		slug = sanitized
	}

	// Build values map
	values := map[string]interface{}{
		"number": number,
		"slug":   slug,
	}

	// Add parent context (epic, feature numbers)
	for k, v := range parentContext {
		values[k] = v
	}

	// Apply generation format
	return ApplyGenerationFormat(formatTemplate, values)
}
