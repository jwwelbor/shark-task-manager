package gateresult

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// credentialLabelPatterns and forbiddenSubstrings mirror
// internal/models/question.go's ValidateQuestionBoundedText marker set
// exactly (REQ-NF-001: "Reuse the Question model's bounded-text and
// forbidden-marker approach"). Reimplemented locally rather than imported so
// this package can classify the failure into its own ErrorClass taxonomy
// instead of parsing another package's error string.
var credentialLabelPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)password\s*=`),
	regexp.MustCompile(`(?i)authorization\s*:`),
	regexp.MustCompile(`(?i)bearer(?:\s|:)`),
}

var forbiddenSubstrings = []string{"api_key", "system prompt", "user prompt", "assistant:"}

func containsForbiddenContent(value string) bool {
	for _, pattern := range credentialLabelPatterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	lower := strings.ToLower(value)
	for _, marker := range forbiddenSubstrings {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// boundedText enforces the shared bounded-text contract for every GateResult
// free-text field: trimmed, valid UTF-8, within [minBytes, maxBytes], and
// free of forbidden credential/rendered-prompt/transcript markers.
func boundedText(field, value string, minBytes, maxBytes int) error {
	trimmed := strings.TrimSpace(value)
	if trimmed != value {
		return newValidationError(field, ErrorClassBounds, "must be trimmed")
	}
	if !utf8.ValidString(value) {
		return newValidationError(field, ErrorClassBounds, "must be valid UTF-8")
	}
	if n := len(value); n < minBytes || n > maxBytes {
		return newValidationError(field, ErrorClassBounds, "must contain "+strconv.Itoa(minBytes)+" through "+strconv.Itoa(maxBytes)+" UTF-8 bytes")
	}
	if containsForbiddenContent(value) {
		return newValidationError(field, ErrorClassForbiddenContent, "must not contain credential, rendered prompt, or transcript material")
	}
	return nil
}
