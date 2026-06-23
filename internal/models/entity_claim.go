package models

import (
	"strings"
	"time"
)

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
