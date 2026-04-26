package maintainer

import "fmt"

// UnauthorizedError is the only user-visible error returned by Authorize for
// authorization failures. Callers can inspect the Reason field for programmatic
// handling or display the Error() string to users.
//
// Spec reference: spec.md §2.2, REQ-F-002, REQ-F-007, AC-T6.
type UnauthorizedError struct {
	// Reason is a stable string for programmatic handling.
	// One of: "missing_config", "wrong_password", "expired_cache",
	// "hash_mismatch_after_rotation".
	Reason string
}

// Error returns a user-friendly message. For the "missing_config" reason, the
// message includes the literal substring "shark admin maintainer set-password"
// as required by AC-T6 and AC-3 in spec.md.
func (e *UnauthorizedError) Error() string {
	switch e.Reason {
	case "missing_config":
		return "maintainer password is not configured; run `shark admin maintainer set-password` to set one in .sharkconfig.json"
	case "wrong_password":
		return "incorrect maintainer password"
	case "expired_cache":
		return "maintainer authorization has expired; please re-enter the password"
	case "hash_mismatch_after_rotation":
		return "cached authorization is no longer valid (password may have changed); please re-enter the password"
	default:
		return fmt.Sprintf("unauthorized: %s", e.Reason)
	}
}

// UserHint returns additional guidance for the user, suitable for display in
// CLI error messages. For missing_config this includes the set-password command.
func (e *UnauthorizedError) UserHint() string {
	if e.Reason == "missing_config" {
		return "run `shark admin maintainer set-password` to configure the maintainer password in .sharkconfig.json"
	}
	return ""
}
