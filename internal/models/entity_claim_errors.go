package models

import "errors"

// Claim validation sentinel errors (E35-F03).
var (
	ErrClaimMissingEntityType = errors.New("claim: entity_type cannot be empty")
	ErrClaimMissingEntityKey  = errors.New("claim: entity_key cannot be empty")
	ErrClaimMissingClaimedBy  = errors.New("claim: claimed_by cannot be empty")
	ErrClaimMissingSession    = errors.New("claim: session_id cannot be empty")
)
