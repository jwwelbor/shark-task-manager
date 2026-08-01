package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// CLIError represents a structured error response for JSON output mode.
// When --json is active, all errors are output as structured JSON on stdout
// instead of human-readable messages on stderr.
type CLIError struct {
	Error            bool                    `json:"error"`
	Code             string                  `json:"code"`
	Message          string                  `json:"message"`
	Entity           string                  `json:"entity,omitempty"`
	EntityKey        string                  `json:"entity_key,omitempty"`
	CurrentStatus    string                  `json:"current_status,omitempty"`
	ValidTransitions []string                `json:"valid_transitions,omitempty"`
	QuestionBlock    *services.QuestionBlock `json:"question_block,omitempty"`
}

// Error code constants for structured error responses
const (
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeInvalidTransition = "INVALID_TRANSITION"
	ErrCodeValidationError   = "VALIDATION_ERROR"
	ErrCodeDatabaseError     = "DATABASE_ERROR"
	ErrCodeInvalidArgs       = "INVALID_ARGS"
	ErrCodeCommandError      = "COMMAND_ERROR"
	ErrCodeQuestionBlocked   = "QUESTION_BLOCKED"
)

// ErrorJSON outputs a structured CLIError. In JSON mode, it outputs the full
// CLIError struct as JSON to stdout. In human mode, it falls back to the
// standard Error() function with e.Message.
func ErrorJSON(e CLIError) {
	e.Error = true
	if GlobalConfig.JSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(e); err != nil {
			// Fallback: write raw message to stderr if JSON encoding fails
			fmt.Fprintln(os.Stderr, "Error:", e.Message)
		}
	} else {
		Error(e.Message)
	}
}
