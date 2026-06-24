package models

import (
	"errors"
	"testing"
	"time"
)

func validClaim() EntityClaim {
	return EntityClaim{
		EntityType: "task",
		EntityKey:  "E01-F01-001",
		ClaimedBy:  "dev-agent",
		SessionID:  "sess-123",
	}
}

func TestEntityClaim_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*EntityClaim)
		wantErr error
	}{
		{"valid", func(_ *EntityClaim) {}, nil},
		{"missing entity_type", func(c *EntityClaim) { c.EntityType = "" }, ErrClaimMissingEntityType},
		{"whitespace entity_type", func(c *EntityClaim) { c.EntityType = "   " }, ErrClaimMissingEntityType},
		{"missing entity_key", func(c *EntityClaim) { c.EntityKey = "" }, ErrClaimMissingEntityKey},
		{"whitespace entity_key", func(c *EntityClaim) { c.EntityKey = "\t" }, ErrClaimMissingEntityKey},
		{"missing claimed_by", func(c *EntityClaim) { c.ClaimedBy = "" }, ErrClaimMissingClaimedBy},
		{"whitespace claimed_by", func(c *EntityClaim) { c.ClaimedBy = " " }, ErrClaimMissingClaimedBy},
		{"missing session_id", func(c *EntityClaim) { c.SessionID = "" }, ErrClaimMissingSession},
		{"whitespace session_id", func(c *EntityClaim) { c.SessionID = "  " }, ErrClaimMissingSession},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validClaim()
			tt.mutate(&c)
			err := c.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestEntityClaim_IsExpired(t *testing.T) {
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	ttl := 15 * time.Minute

	tests := []struct {
		name      string
		heartbeat time.Time
		ttl       time.Duration
		want      bool
	}{
		{"non-positive ttl never expires", now.Add(-time.Hour), 0, false},
		{"negative ttl never expires", now.Add(-time.Hour), -time.Minute, false},
		{"well within ttl", now.Add(-time.Minute), ttl, false},
		{"exactly at ttl is not expired", now.Add(-ttl), ttl, false},
		{"just past ttl is expired", now.Add(-ttl - time.Nanosecond), ttl, true},
		{"long past ttl is expired", now.Add(-time.Hour), ttl, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validClaim()
			c.LastHeartbeat = tt.heartbeat
			if got := c.IsExpired(now, tt.ttl); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}
