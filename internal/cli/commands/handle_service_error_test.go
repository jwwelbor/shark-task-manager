package commands

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// TestHandleServiceError_Nil verifies that nil errors are handled gracefully (no exit called).
func TestHandleServiceError_Nil(t *testing.T) {
	// This should not call os.Exit, so we can call it directly.
	handleServiceError(nil, "task", "E07-F01-001")
	// If we reach here, no exit occurred - test passes.
}

// TestHandleServiceError_NotFound_ExitsWithCode1 verifies exit code 1 for "not found" errors.
// Uses subprocess pattern because handleServiceError calls os.Exit.
func TestHandleServiceError_NotFound_ExitsWithCode1(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		// This runs in the child process - call the function under test directly.
		handleServiceError(fmt.Errorf("epic not found: E07: %w", repository.ErrNotFound), "epic", "E07")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestHandleServiceError_NotFound_ExitsWithCode1")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		if e.ExitCode() != 1 {
			t.Errorf("expected exit code 1, got %d", e.ExitCode())
		}
		return
	}
	t.Fatal("expected process to exit with code 1")
}

// TestHandleServiceError_DoesNotExist_ExitsWithCode1 verifies "does not exist" messages also exit 1.
func TestHandleServiceError_DoesNotExist_ExitsWithCode1(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		handleServiceError(fmt.Errorf("feature does not exist: E07-F01"), "feature", "E07-F01")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestHandleServiceError_DoesNotExist_ExitsWithCode1")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		if e.ExitCode() != 1 {
			t.Errorf("expected exit code 1, got %d", e.ExitCode())
		}
		return
	}
	t.Fatal("expected process to exit with code 1")
}

// TestHandleServiceError_InvalidTransition_ExitsWithCode3 verifies exit code 3 for workflow errors.
func TestHandleServiceError_InvalidTransition_ExitsWithCode3(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		handleServiceError(fmt.Errorf("invalid transition from 'todo' to 'completed'"), "task", "E07-F01-001")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestHandleServiceError_InvalidTransition_ExitsWithCode3")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		if e.ExitCode() != 3 {
			t.Errorf("expected exit code 3, got %d", e.ExitCode())
		}
		return
	}
	t.Fatal("expected process to exit with code 3")
}

// TestHandleServiceError_AlreadyExists_ExitsWithCode3 verifies exit code 3 for conflict errors.
func TestHandleServiceError_AlreadyExists_ExitsWithCode3(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		handleServiceError(fmt.Errorf("epic already exists: E07"), "epic", "E07")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestHandleServiceError_AlreadyExists_ExitsWithCode3")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		if e.ExitCode() != 3 {
			t.Errorf("expected exit code 3, got %d", e.ExitCode())
		}
		return
	}
	t.Fatal("expected process to exit with code 3")
}

// TestHandleServiceError_ValidationFailed_ExitsWithCode3 verifies validation errors exit with code 3.
func TestHandleServiceError_ValidationFailed_ExitsWithCode3(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		handleServiceError(fmt.Errorf("validation failed: title cannot be empty"), "task", "E07-F01-001")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestHandleServiceError_ValidationFailed_ExitsWithCode3")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		if e.ExitCode() != 3 {
			t.Errorf("expected exit code 3, got %d", e.ExitCode())
		}
		return
	}
	t.Fatal("expected process to exit with code 3")
}

// TestHandleServiceError_CannotBeEmpty_ExitsWithCode3 verifies empty field errors exit with code 3.
func TestHandleServiceError_CannotBeEmpty_ExitsWithCode3(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		handleServiceError(fmt.Errorf("title cannot be empty"), "task", "E07-F01-001")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestHandleServiceError_CannotBeEmpty_ExitsWithCode3")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		if e.ExitCode() != 3 {
			t.Errorf("expected exit code 3, got %d", e.ExitCode())
		}
		return
	}
	t.Fatal("expected process to exit with code 3")
}

// TestHandleServiceError_CannotStart_ExitsWithCode3 verifies "cannot start" errors exit with code 3.
func TestHandleServiceError_CannotStart_ExitsWithCode3(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		handleServiceError(fmt.Errorf("cannot start task in status completed"), "task", "E07-F01-001")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestHandleServiceError_CannotStart_ExitsWithCode3")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		if e.ExitCode() != 3 {
			t.Errorf("expected exit code 3, got %d", e.ExitCode())
		}
		return
	}
	t.Fatal("expected process to exit with code 3")
}

// TestHandleServiceError_CannotComplete_ExitsWithCode3 verifies "cannot complete" errors exit with code 3.
func TestHandleServiceError_CannotComplete_ExitsWithCode3(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		handleServiceError(fmt.Errorf("cannot complete task in status todo"), "task", "E07-F01-001")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestHandleServiceError_CannotComplete_ExitsWithCode3")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		if e.ExitCode() != 3 {
			t.Errorf("expected exit code 3, got %d", e.ExitCode())
		}
		return
	}
	t.Fatal("expected process to exit with code 3")
}

// TestHandleServiceError_GenericError_ExitsWithCode2 verifies generic errors exit with code 2.
func TestHandleServiceError_GenericError_ExitsWithCode2(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		handleServiceError(fmt.Errorf("database connection failed"), "task", "E07-F01-001")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestHandleServiceError_GenericError_ExitsWithCode2")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		if e.ExitCode() != 2 {
			t.Errorf("expected exit code 2, got %d", e.ExitCode())
		}
		return
	}
	t.Fatal("expected process to exit with code 2")
}

// TestHandleServiceError_SystemError_ExitsWithCode2 verifies unexpected system errors exit with code 2.
func TestHandleServiceError_SystemError_ExitsWithCode2(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		handleServiceError(fmt.Errorf("unexpected internal error"), "feature", "E07-F01")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestHandleServiceError_SystemError_ExitsWithCode2")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		if e.ExitCode() != 2 {
			t.Errorf("expected exit code 2, got %d", e.ExitCode())
		}
		return
	}
	t.Fatal("expected process to exit with code 2")
}
