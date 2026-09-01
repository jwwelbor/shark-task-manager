package models

import (
	"fmt"
	"strings"
	"time"
)

// maxHarnessFieldLength is the REQ-NF-004 length cap applied to each harness
// identity field (harness, harness_version, harness_model). 100 characters is
// inclusive of the boundary — exactly 100 is valid, 101 is rejected.
const maxHarnessFieldLength = 100

// EntityClaim is the in-flight lease on an entity (E35-F03, decision D3).
//
// In the route-based model, status is a pure phase and the claim is the lease:
// an agent claims an entity before working it, `shark next` hands out only
// unclaimed entities, and heartbeats renew the lease. A TTL backstop reclaims
// leases whose heartbeats have gone stale (the universal crash-recovery
// mechanism). There is at most one claim per entity (UNIQUE(entity_type,
// entity_key) in the schema).
type EntityClaim struct {
	ID            int64     `json:"id"`
	EntityType    string    `json:"entity_type"`
	EntityKey     string    `json:"entity_key"`
	ClaimedBy     string    `json:"claimed_by"`
	SessionID     string    `json:"session_id"`
	ClaimedAt     time.Time `json:"claimed_at"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	Progress      *float64  `json:"progress,omitempty"`
	Note          string    `json:"note,omitempty"`

	// Harness identity (E34-F01, spec.md §3.1). All three are open,
	// normalized strings — no DB enum, no CHECK constraint (D-F01-03) — so
	// Shark can carry harness types it does not yet know about. NULL in the
	// database maps to "" here; empty is valid (unknown harness).
	Harness        string `json:"harness,omitempty"`
	HarnessVersion string `json:"harness_version,omitempty"`
	HarnessModel   string `json:"harness_model,omitempty"`
}

// Validate performs structural validation on a claim (no workflow knowledge).
func (c *EntityClaim) Validate() error {
	if strings.TrimSpace(c.EntityType) == "" {
		return ErrClaimMissingEntityType
	}
	if strings.TrimSpace(c.EntityKey) == "" {
		return ErrClaimMissingEntityKey
	}
	if strings.TrimSpace(c.ClaimedBy) == "" {
		return ErrClaimMissingClaimedBy
	}
	if strings.TrimSpace(c.SessionID) == "" {
		return ErrClaimMissingSession
	}
	if err := validateHarnessFieldLength("harness", c.Harness); err != nil {
		return err
	}
	if err := validateHarnessFieldLength("harness_version", c.HarnessVersion); err != nil {
		return err
	}
	if err := validateHarnessFieldLength("harness_model", c.HarnessModel); err != nil {
		return err
	}
	return nil
}

// validateHarnessFieldLength enforces REQ-NF-004's 100-character cap on a
// single harness identity field. Empty is always valid — there is no enum
// allowlist (D-F01-03), only a length bound. The error names the field and
// quotes the offending input with %q, per
// .claude/rules/go/input-sanitization.md.
func validateHarnessFieldLength(field, value string) error {
	if len(value) > maxHarnessFieldLength {
		return fmt.Errorf("%w: %s %q", ErrClaimHarnessFieldTooLong, field, value)
	}
	return nil
}

// IsExpired reports whether the claim's last heartbeat is older than ttl
// relative to now. A non-positive ttl means "never expires".
func (c *EntityClaim) IsExpired(now time.Time, ttl time.Duration) bool {
	if ttl <= 0 {
		return false
	}
	return now.Sub(c.LastHeartbeat) > ttl
}
