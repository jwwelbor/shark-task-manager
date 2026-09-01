package gatepersist

import (
	"errors"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TestValidateKickbacks_TargetEntityWorkflowMembership table-tests the
// target-entity workflow-membership check per feature.md's verification
// plan: valid target status, invalid/unknown target status, and (via the
// coordinator-level tests) idempotent vs. conflicting retry.
func TestValidateKickbacks_TargetEntityWorkflowMembership(t *testing.T) {
	tests := []struct {
		name       string
		kickbacks  []gateresult.Kickback
		mainEntity string
		validator  *fakeStatusValidator
		wantErr    bool
		wantKind   string // "" | "invalid_status" | "targets_main"
	}{
		{
			name:       "valid target status is accepted",
			kickbacks:  []gateresult.Kickback{{EntityKey: "T-E01-F01-001", TargetStatus: "todo", Reason: "r"}},
			mainEntity: "E01-F02",
			validator:  newFakeStatusValidator().allow(models.EntityTypeTask, "todo"),
		},
		{
			name:       "unknown target status is rejected",
			kickbacks:  []gateresult.Kickback{{EntityKey: "T-E01-F01-001", TargetStatus: "not_a_status", Reason: "r"}},
			mainEntity: "E01-F02",
			validator:  newFakeStatusValidator().allow(models.EntityTypeTask, "todo"),
			wantErr:    true,
			wantKind:   "invalid_status",
		},
		{
			name:       "kickback targeting the bound main entity is rejected regardless of status validity",
			kickbacks:  []gateresult.Kickback{{EntityKey: "E01-F02", TargetStatus: "todo", Reason: "r"}},
			mainEntity: "E01-F02",
			validator:  newFakeStatusValidator().allow(models.EntityTypeFeature, "todo"),
			wantErr:    true,
			wantKind:   "targets_main",
		},
		{
			name:       "unrecognized key shape is rejected",
			kickbacks:  []gateresult.Kickback{{EntityKey: "not-a-key-shape", TargetStatus: "todo", Reason: "r"}},
			mainEntity: "E01-F02",
			validator:  newFakeStatusValidator().allow(models.EntityTypeTask, "todo"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateKickbacks(tt.kickbacks, tt.mainEntity, tt.validator)
			if tt.wantErr && err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantKind == "invalid_status" {
				var kerr *KickbackValidationError
				if !errors.As(err, &kerr) {
					t.Fatalf("expected *KickbackValidationError, got %T: %v", err, err)
				}
			}
		})
	}
}

func TestKickbackReasonTokenRoundTrip(t *testing.T) {
	subID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"  // 64 hex chars, like a real sha256 suboperation ID
	digest := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" // 64 hex chars, like a real sha256 content digest
	reason := buildKickbackReason("please fix the caller", subID, digest)
	gotSub, gotDigest, ok := parseKickbackToken(reason)
	if !ok {
		t.Fatalf("expected token to parse from %q", reason)
	}
	if gotSub != subID {
		t.Fatalf("parseKickbackToken() subID = %q, want %q", gotSub, subID)
	}
	if gotDigest != digest {
		t.Fatalf("parseKickbackToken() digest = %q, want %q", gotDigest, digest)
	}
}

func TestKickbackReasonTokenAbsent(t *testing.T) {
	if _, _, ok := parseKickbackToken("a plain reason with no token"); ok {
		t.Fatalf("expected no token to be found")
	}
}
