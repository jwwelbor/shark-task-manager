package init

import "fmt"

// InitError represents an initialization error
type InitError struct {
	Step    string // Which step failed
	Message string // Human-readable message
	Err     error  // Underlying error
}

func (e *InitError) Error() string {
	return fmt.Sprintf("initialization failed at step '%s': %s: %v", e.Step, e.Message, e.Err)
}

func (e *InitError) Unwrap() error {
	return e.Err
}
