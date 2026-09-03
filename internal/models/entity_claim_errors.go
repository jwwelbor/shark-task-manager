package models

import "errors"

// Claim validation sentinel errors (E35-F03).
var (
	ErrClaimMissingEntityType = errors.New("claim: entity_type cannot be empty")
	ErrClaimMissingEntityKey  = errors.New("claim: entity_key cannot be empty")
	ErrClaimMissingClaimedBy  = errors.New("claim: claimed_by cannot be empty")
	ErrClaimMissingSession    = errors.New("claim: session_id cannot be empty")
	// ErrClaimHarnessFieldTooLong is the sentinel for a harness identity
	// field (harness, harness_version, harness_model) exceeding the
	// REQ-NF-004 length cap (E34-F01). The wrapping error names the specific
	// field and quotes the offending input.
	ErrClaimHarnessFieldTooLong = errors.New("claim: harness field exceeds maximum length of 100 characters")
)
