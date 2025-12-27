package status

// StatusError represents an error that occurred in the status service
type StatusError struct {
	Message string
	Code    int // Exit code
}

// Error implements the error interface
func (e *StatusError) Error() string {
	return e.Message
}

// NewStatusError creates a new StatusError with the given message and default exit code
func NewStatusError(message string) error {
	return &StatusError{
		Message: message,
		Code:    1,
	}
}

// NewStatusErrorWithCode creates a new StatusError with a specific exit code
func NewStatusErrorWithCode(message string, code int) error {
	return &StatusError{
		Message: message,
		Code:    code,
	}
}
