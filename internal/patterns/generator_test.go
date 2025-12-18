package patterns

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePlaceholder(t *testing.T) {
	tests := []struct {
		name           string
		placeholder    string
		wantField      string
		wantFormat     string
		wantFormatType string
	}{
		{
			name:           "simple number placeholder",
			placeholder:    "{number}",
			wantField:      "number",
			wantFormat:     "",
			wantFormatType: "",
		},
		{
			name:           "formatted number placeholder",
			placeholder:    "{number:02d}",
			wantField:      "number",
			wantFormat:     "02d",
			wantFormatType: "d",
		},
		{
			name:           "slug placeholder",
			placeholder:    "{slug}",
			wantField:      "slug",
			wantFormat:     "",
			wantFormatType: "",
		},
		{
			name:           "epic placeholder",
			placeholder:    "{epic}",
			wantField:      "epic",
			wantFormat:     "",
			wantFormatType: "",
		},
		{
			name:           "formatted epic placeholder",
			placeholder:    "{epic:02d}",
			wantField:      "epic",
			wantFormat:     "02d",
			wantFormatType: "d",
		},
		{
			name:           "feature placeholder",
			placeholder:    "{feature}",
			wantField:      "feature",
			wantFormat:     "",
			wantFormatType: "",
		},
		{
			name:           "formatted feature placeholder",
			placeholder:    "{feature:02d}",
			wantField:      "feature",
			wantFormat:     "02d",
			wantFormatType: "d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, format, formatType := ParsePlaceholder(tt.placeholder)
			assert.Equal(t, tt.wantField, field, "Field should match")
			assert.Equal(t, tt.wantFormat, format, "Format should match")
			assert.Equal(t, tt.wantFormatType, formatType, "Format type should match")
		})
	}
}

func TestApplyGenerationFormat_Epic(t *testing.T) {
	format := "E{number:02d}-{slug}"
	values := map[string]interface{}{
		"number": 5,
		"slug":   "identity-platform",
	}

	result, err := ApplyGenerationFormat(format, values)
	require.NoError(t, err)
	assert.Equal(t, "E05-identity-platform", result)
}

func TestApplyGenerationFormat_Feature(t *testing.T) {
	format := "E{epic:02d}-F{number:02d}-{slug}"
	values := map[string]interface{}{
		"epic":   4,
		"number": 8,
		"slug":   "oauth-integration",
	}

	result, err := ApplyGenerationFormat(format, values)
	require.NoError(t, err)
	assert.Equal(t, "E04-F08-oauth-integration", result)
}

func TestApplyGenerationFormat_Task(t *testing.T) {
	format := "T-E{epic:02d}-F{feature:02d}-{number:03d}.md"
	values := map[string]interface{}{
		"epic":    4,
		"feature": 7,
		"number":  7,
	}

	result, err := ApplyGenerationFormat(format, values)
	require.NoError(t, err)
	assert.Equal(t, "T-E04-F07-007.md", result)
}

func TestSanitizeSlug_Valid(t *testing.T) {
	tests := []struct {
		name string
		slug string
		want string
	}{
		{
			name: "lowercase letters and hyphens",
			slug: "oauth-integration",
			want: "oauth-integration",
		},
		{
			name: "lowercase letters and numbers",
			slug: "identity-platform-v2",
			want: "identity-platform-v2",
		},
		{
			name: "single word",
			slug: "authentication",
			want: "authentication",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizeSlug(tt.slug)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestSanitizeSlug_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		slug      string
		wantError string
	}{
		{
			name:      "contains forward slash",
			slug:      "../malicious",
			wantError: "forbidden characters",
		},
		{
			name:      "contains backslash",
			slug:      "path\\injection",
			wantError: "forbidden characters",
		},
		{
			name:      "contains colon",
			slug:      "drive:path",
			wantError: "forbidden characters",
		},
		{
			name:      "contains double dot",
			slug:      "..hidden",
			wantError: "forbidden characters",
		},
		{
			name:      "path traversal attempt",
			slug:      "../../etc/passwd",
			wantError: "forbidden characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SanitizeSlug(tt.slug)
			require.Error(t, err, "Should reject slug with forbidden characters")
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestValidatePath_WithinBoundaries(t *testing.T) {
	projectRoot := "/home/user/project"
	validPaths := []string{
		"/home/user/project/docs/plan/E05-feature",
		"/home/user/project/docs/tasks/T-E04-F07-001.md",
		"/home/user/project/README.md",
	}

	for _, path := range validPaths {
		t.Run(path, func(t *testing.T) {
			err := ValidatePathWithinProject(path, projectRoot)
			assert.NoError(t, err, "Valid path should pass validation")
		})
	}
}

func TestValidatePath_OutsideBoundaries(t *testing.T) {
	projectRoot := "/home/user/project"
	invalidPaths := []string{
		"/home/user/other-project/file.md",
		"/etc/passwd",
		"/home/user/../../../etc/shadow",
	}

	for _, path := range invalidPaths {
		t.Run(path, func(t *testing.T) {
			err := ValidatePathWithinProject(path, projectRoot)
			assert.Error(t, err, "Path outside project should fail validation")
			assert.Contains(t, err.Error(), "outside project boundaries")
		})
	}
}

func TestExtractPlaceholders(t *testing.T) {
	tests := []struct {
		name             string
		format           string
		wantPlaceholders []string
	}{
		{
			name:             "epic format",
			format:           "E{number:02d}-{slug}",
			wantPlaceholders: []string{"{number:02d}", "{slug}"},
		},
		{
			name:             "feature format",
			format:           "E{epic:02d}-F{number:02d}-{slug}",
			wantPlaceholders: []string{"{epic:02d}", "{number:02d}", "{slug}"},
		},
		{
			name:             "task format",
			format:           "T-E{epic:02d}-F{feature:02d}-{number:03d}.md",
			wantPlaceholders: []string{"{epic:02d}", "{feature:02d}", "{number:03d}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			placeholders := ExtractPlaceholders(tt.format)
			assert.Equal(t, tt.wantPlaceholders, placeholders)
		})
	}
}

func TestApplyGenerationFormat_MissingValue(t *testing.T) {
	format := "E{number:02d}-{slug}"
	values := map[string]interface{}{
		"number": 5,
		// missing slug
	}

	_, err := ApplyGenerationFormat(format, values)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing value")
	assert.Contains(t, err.Error(), "slug")
}

func TestApplyGenerationFormat_InvalidFormat(t *testing.T) {
	// Test with unclosed placeholder
	format := "E{number:02d}-{slug"
	values := map[string]interface{}{
		"number": 5,
		"slug":   "test",
	}

	_, err := ApplyGenerationFormat(format, values)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid format")
}
