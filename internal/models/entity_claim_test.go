package models

import (
	"errors"
	"fmt"
	"strings"
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
		// TC-013 (test-plan.md, model layer): AC-10 — Validate() rejects a
		// harness field over 100 characters; exactly 100 is the accepted
		// boundary (REQ-NF-004). Independently exercised for all three
		// harness fields per the test-plan's "repeat for version/model" edge
		// case.
		{"harness at 100 chars is valid", func(c *EntityClaim) { c.Harness = strings.Repeat("a", 100) }, nil},
		{"harness over 100 chars is rejected", func(c *EntityClaim) { c.Harness = strings.Repeat("a", 101) }, ErrClaimHarnessFieldTooLong},
		{"harness_version at 100 chars is valid", func(c *EntityClaim) { c.HarnessVersion = strings.Repeat("b", 100) }, nil},
		{"harness_version over 100 chars is rejected", func(c *EntityClaim) { c.HarnessVersion = strings.Repeat("b", 101) }, ErrClaimHarnessFieldTooLong},
		{"harness_model at 100 chars is valid", func(c *EntityClaim) { c.HarnessModel = strings.Repeat("c", 100) }, nil},
		{"harness_model over 100 chars is rejected", func(c *EntityClaim) { c.HarnessModel = strings.Repeat("c", 101) }, ErrClaimHarnessFieldTooLong},
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

// TestEntityClaim_Validate_HarnessFieldTooLong_NamesFieldAndQuotesInput is
// TC-013's model-layer slice (test-plan.md; task-scoped per T-E34-F01-001's
// task spec — the CLI-boundary half of TC-013, including "no partial claim
// row is written," is covered by T-E34-F01-002 at the runClaim seam). It
// pins down AC-10's error-shape requirement: the error must name the
// offending field and quote the offending input with %q, per
// .claude/rules/go/input-sanitization.md (REQ-NF-004). errors.Is() alone
// (covered by the table above) cannot distinguish "rejected the right
// field" from "rejected the wrong field with the same sentinel", so this
// test inspects the message text directly.
func TestEntityClaim_Validate_HarnessFieldTooLong_NamesFieldAndQuotesInput(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*EntityClaim)
		wantField string
		wantValue string
	}{
		{"harness", func(c *EntityClaim) { c.Harness = strings.Repeat("x", 101) }, "harness", strings.Repeat("x", 101)},
		{"harness_version", func(c *EntityClaim) { c.HarnessVersion = strings.Repeat("y", 101) }, "harness_version", strings.Repeat("y", 101)},
		{"harness_model", func(c *EntityClaim) { c.HarnessModel = strings.Repeat("z", 101) }, "harness_model", strings.Repeat("z", 101)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validClaim()
			tt.mutate(&c)
			err := c.Validate()
			if err == nil {
				t.Fatal("Validate() = nil, want a length-cap error")
			}
			if !errors.Is(err, ErrClaimHarnessFieldTooLong) {
				t.Errorf("Validate() = %v, want wrapped %v", err, ErrClaimHarnessFieldTooLong)
			}
			if !strings.Contains(err.Error(), tt.wantField) {
				t.Errorf("Validate() error %q does not name field %q", err.Error(), tt.wantField)
			}
			quoted := fmt.Sprintf("%q", tt.wantValue)
			if !strings.Contains(err.Error(), quoted) {
				t.Errorf("Validate() error %q does not quote input %s", err.Error(), quoted)
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
