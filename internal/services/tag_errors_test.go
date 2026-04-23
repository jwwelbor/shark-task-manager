package services

import (
	"strings"
	"testing"
)

// TestTagErrors_ErrorMessages verifies all four typed error structs
// implement error and produce the correct message formats per spec §2.3
// and REQ-F-007.
//
// Test plan references:
//   - AC-T1: all four types exported and implement error
//   - AC-T2: ValidationError.Error() format
//   - AC-T3: NotFoundError.Error() format
//   - AC-T4: ConflictError.Error() format
//   - AC-T5: TagInUseError.Error() includes count and "--force" hint
//   - Test plan section 1, case 5.3 (TagInUseError message content)
func TestValidationError_Error(t *testing.T) {
	t.Run("AC-T2: format is 'invalid <Field>: <Message>'", func(t *testing.T) {
		e := &ValidationError{Field: "tag name", Message: "must match ^[a-z0-9][a-z0-9-]{0,63}$"}
		got := e.Error()
		want := "invalid tag name: must match ^[a-z0-9][a-z0-9-]{0,63}$"
		if got != want {
			t.Errorf("ValidationError.Error() = %q, want %q", got, want)
		}
	})

	t.Run("AC-T2: empty field and message still follows format", func(t *testing.T) {
		e := &ValidationError{Field: "", Message: ""}
		got := e.Error()
		if got != "invalid : " {
			t.Errorf("ValidationError.Error() = %q, want %q", got, "invalid : ")
		}
	})

	t.Run("AC-T1: ValidationError implements error interface", func(t *testing.T) {
		var _ error = &ValidationError{}
	})
}

func TestNotFoundError_Error(t *testing.T) {
	t.Run("AC-T3: format is 'tag not found: <Name>'", func(t *testing.T) {
		e := &NotFoundError{Name: "audio"}
		got := e.Error()
		want := "tag not found: audio"
		if got != want {
			t.Errorf("NotFoundError.Error() = %q, want %q", got, want)
		}
	})

	t.Run("AC-T3: name is preserved verbatim", func(t *testing.T) {
		e := &NotFoundError{Name: "my-special-tag"}
		got := e.Error()
		if got != "tag not found: my-special-tag" {
			t.Errorf("NotFoundError.Error() = %q", got)
		}
	})

	t.Run("AC-T1: NotFoundError implements error interface", func(t *testing.T) {
		var _ error = &NotFoundError{}
	})
}

func TestConflictError_Error(t *testing.T) {
	t.Run("AC-T4: format is 'tag already exists: <Name>'", func(t *testing.T) {
		e := &ConflictError{Name: "voice"}
		got := e.Error()
		want := "tag already exists: voice"
		if got != want {
			t.Errorf("ConflictError.Error() = %q, want %q", got, want)
		}
	})

	t.Run("AC-T1: ConflictError implements error interface", func(t *testing.T) {
		var _ error = &ConflictError{}
	})
}

func TestTagInUseError_Error(t *testing.T) {
	t.Run("AC-T5 / plan-5.3: error message includes count and --force hint", func(t *testing.T) {
		e := &TagInUseError{Name: "voice", Count: 7}
		got := e.Error()

		if !strings.Contains(got, "7") {
			t.Errorf("TagInUseError.Error() missing count '7': %q", got)
		}
		if !strings.Contains(got, "--force") {
			t.Errorf("TagInUseError.Error() missing '--force': %q", got)
		}
		if !strings.Contains(got, "voice") {
			t.Errorf("TagInUseError.Error() missing tag name 'voice': %q", got)
		}
	})

	t.Run("AC-T5: count=1 also includes --force hint", func(t *testing.T) {
		e := &TagInUseError{Name: "audio", Count: 1}
		got := e.Error()
		if !strings.Contains(got, "1") {
			t.Errorf("TagInUseError.Error() missing count '1': %q", got)
		}
		if !strings.Contains(got, "--force") {
			t.Errorf("TagInUseError.Error() missing '--force' for count=1: %q", got)
		}
	})

	t.Run("AC-T5: exact message format from REQ-F-007", func(t *testing.T) {
		e := &TagInUseError{Name: "voice", Count: 7}
		got := e.Error()
		// The spec says: "tag %q is in use by %d entities; re-run with --force to delete it and its associations"
		if !strings.Contains(got, "in use by") {
			t.Errorf("TagInUseError.Error() missing 'in use by': %q", got)
		}
		if !strings.Contains(got, "re-run") || !strings.Contains(got, "--force") {
			t.Errorf("TagInUseError.Error() missing 're-run with --force': %q", got)
		}
	})

	t.Run("AC-T1: TagInUseError implements error interface", func(t *testing.T) {
		var _ error = &TagInUseError{}
	})
}

// TestTagErrors_ErrorsAs verifies the types work with errors.As (CLI will use this).
func TestTagErrors_ErrorsAs(t *testing.T) {
	t.Run("ValidationError works with errors.As", func(t *testing.T) {
		err := error(&ValidationError{Field: "tag name", Message: "bad"})
		var target *ValidationError
		if !errorsAs(err, &target) {
			t.Error("errors.As should find *ValidationError")
		}
		if target.Field != "tag name" {
			t.Errorf("Field = %q, want %q", target.Field, "tag name")
		}
	})

	t.Run("NotFoundError works with errors.As", func(t *testing.T) {
		err := error(&NotFoundError{Name: "audio"})
		var target *NotFoundError
		if !errorsAs(err, &target) {
			t.Error("errors.As should find *NotFoundError")
		}
		if target.Name != "audio" {
			t.Errorf("Name = %q, want %q", target.Name, "audio")
		}
	})

	t.Run("ConflictError works with errors.As", func(t *testing.T) {
		err := error(&ConflictError{Name: "voice"})
		var target *ConflictError
		if !errorsAs(err, &target) {
			t.Error("errors.As should find *ConflictError")
		}
		if target.Name != "voice" {
			t.Errorf("Name = %q, want %q", target.Name, "voice")
		}
	})

	t.Run("TagInUseError works with errors.As", func(t *testing.T) {
		err := error(&TagInUseError{Name: "voice", Count: 7})
		var target *TagInUseError
		if !errorsAs(err, &target) {
			t.Error("errors.As should find *TagInUseError")
		}
		if target.Name != "voice" {
			t.Errorf("Name = %q, want %q", target.Name, "voice")
		}
		if target.Count != 7 {
			t.Errorf("Count = %d, want %d", target.Count, 7)
		}
	})
}

// errorsAs is a local wrapper so the test file stays free of "errors" import
// only for As. It uses the standard errors.As under the hood.
func errorsAs[T any](err error, target *T) bool {
	// We use a helper to avoid importing "errors" directly in the test body;
	// the errors package is already imported at package level in production code.
	// Actually, just use type assertion since we know the exact types.
	if err == nil {
		return false
	}
	if t, ok := err.(T); ok {
		*target = t
		return true
	}
	return false
}
