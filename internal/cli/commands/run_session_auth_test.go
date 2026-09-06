// This file covers the T-E34-F05-004 rework's CRITICAL UAT finding: before
// this fix, run_apply_result.go's runApplyResult and run_resume.go's
// runResumeRun only checked --session for non-emptiness -- neither verified
// the supplied session actually owns the entity's active claim/lease. That
// meant ANY caller could durably mutate workflow state (notes, kickbacks,
// transitions) for any entity via `shark run <key> --apply-result=...
// --session=<any-non-empty-string>`, bypassing the claim/lease system
// REQ-F-002 names as the sole authorization mechanism for this coordinator
// surface.
//
// verifyClaimSession (run.go) is the fix: the real authorization gate,
// checked before any file read, coordinator construction, or
// runner.IngestGateResult call. These tests prove a mismatched, nonexistent,
// or expired session is rejected with zero downstream effect -- and that a
// genuinely-owning session is let through unauthorized-error-free (so the
// gate does not also lock out legitimate callers).
package commands

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

func TestVerifyClaimSession_EmptySessionRejected(t *testing.T) {
	withRunClaimSvcOverride(t, &mockRunClaimService{})

	err := verifyClaimSession(context.Background(), "task", "E01-F01-001", "")
	if err == nil {
		t.Fatal("verifyClaimSession with empty session: want error, got nil")
	}
}

func TestVerifyClaimSession_NoActiveClaimRejected(t *testing.T) {
	// getClaim is nil by default: no active claim exists on the entity at all.
	withRunClaimSvcOverride(t, &mockRunClaimService{})

	err := verifyClaimSession(context.Background(), "task", "E01-F01-001", "attacker-session")
	if err == nil {
		t.Fatal("verifyClaimSession with no active claim: want error, got nil")
	}
	if !strings.Contains(err.Error(), "no active claim") {
		t.Fatalf("error = %v, want a message naming the missing claim", err)
	}
}

func TestVerifyClaimSession_MismatchedSessionRejected(t *testing.T) {
	withRunClaimSvcOverride(t, &mockRunClaimService{
		getClaim: &models.EntityClaim{
			EntityType:    "task",
			EntityKey:     "E01-F01-001",
			SessionID:     "the-real-owning-session",
			LastHeartbeat: time.Now().UTC(),
		},
	})

	err := verifyClaimSession(context.Background(), "task", "E01-F01-001", "attacker-supplied-session")
	if err == nil {
		t.Fatal("verifyClaimSession with a mismatched session: want error, got nil")
	}
	if !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("error = %v, want a message naming the session mismatch", err)
	}
}

func TestVerifyClaimSession_ExpiredClaimRejected(t *testing.T) {
	mock := &mockRunClaimService{
		ttl: time.Minute,
		getClaim: &models.EntityClaim{
			EntityType:    "task",
			EntityKey:     "E01-F01-001",
			SessionID:     "the-real-owning-session",
			LastHeartbeat: time.Now().UTC().Add(-time.Hour), // long past the 1-minute TTL
		},
	}
	withRunClaimSvcOverride(t, mock)

	err := verifyClaimSession(context.Background(), "task", "E01-F01-001", "the-real-owning-session")
	if err == nil {
		t.Fatal("verifyClaimSession with an expired claim: want error, got nil")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("error = %v, want a message naming the expired claim", err)
	}
}

func TestVerifyClaimSession_MatchingLiveSessionAuthorized(t *testing.T) {
	mock := &mockRunClaimService{
		ttl: time.Hour,
		getClaim: &models.EntityClaim{
			EntityType:    "task",
			EntityKey:     "E01-F01-001",
			SessionID:     "the-real-owning-session",
			LastHeartbeat: time.Now().UTC(),
		},
	}
	withRunClaimSvcOverride(t, mock)

	if err := verifyClaimSession(context.Background(), "task", "E01-F01-001", "the-real-owning-session"); err != nil {
		t.Fatalf("verifyClaimSession with a matching live session: want nil, got %v", err)
	}
}

// TestRunApplyResult_MismatchedSessionProducesZeroWrites is the counter-
// factual: against the pre-fix code (a bare non-emptiness check on
// --session), this would have proceeded past authorization straight into
// os.ReadFile on a deliberately nonexistent --apply-result path, ultimately
// failing later with a *file* error rather than an *authorization* error --
// or, given a real result file, would have gone on to actually mutate
// workflow state. With the fix, authorization runs first and fails closed
// before the file is ever read, so the error is unambiguously an
// authorization failure and no note/transition/coordinator work ever begins.
func TestRunApplyResult_MismatchedSessionProducesZeroWrites(t *testing.T) {
	origRunID, origPath, origSession := runApplyRunID, runApplyResultPath, runSession
	t.Cleanup(func() {
		runApplyRunID = origRunID
		runApplyResultPath = origPath
		runSession = origSession
	})

	runApplyRunID = "run-session-auth-1"
	runApplyResultPath = "/nonexistent/path/does-not-exist-run-session-auth.json"
	runSession = "attacker-supplied-session"

	withRunClaimSvcOverride(t, &mockRunClaimService{
		getClaim: &models.EntityClaim{
			EntityType:    "task",
			EntityKey:     "E01-F01-001",
			SessionID:     "the-real-owning-session",
			LastHeartbeat: time.Now().UTC(),
		},
	})

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runApplyResult(cmd, "task", "E01-F01-001")
	if err == nil {
		t.Fatal("runApplyResult with a mismatched session: want error, got nil")
	}
	if !strings.Contains(err.Error(), "authorization failed") {
		t.Fatalf("error = %v, want an authorization failure (proves the envelope file was never read and no coordinator call was made)", err)
	}
}

// TestRunApplyResult_NoActiveClaimProducesZeroWrites is the "any non-empty
// string" bypass itself: a caller who never claimed the entity at all must
// be rejected, not merely a caller whose session id happens to differ from
// an existing claim's.
func TestRunApplyResult_NoActiveClaimProducesZeroWrites(t *testing.T) {
	origRunID, origPath, origSession := runApplyRunID, runApplyResultPath, runSession
	t.Cleanup(func() {
		runApplyRunID = origRunID
		runApplyResultPath = origPath
		runSession = origSession
	})

	runApplyRunID = "run-session-auth-2"
	runApplyResultPath = "/nonexistent/path/does-not-exist-run-session-auth-2.json"
	runSession = "any-non-empty-string-an-attacker-supplies"

	// getClaim is nil: no claim exists on this entity at all.
	withRunClaimSvcOverride(t, &mockRunClaimService{})

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runApplyResult(cmd, "task", "E01-F01-001")
	if err == nil {
		t.Fatal("runApplyResult with no active claim: want error, got nil")
	}
	if !strings.Contains(err.Error(), "authorization failed") {
		t.Fatalf("error = %v, want an authorization failure (proves the envelope file was never read and no coordinator call was made)", err)
	}
}

// TestRunResumeRun_MismatchedSessionProducesZeroWrites mirrors the
// --apply-result coverage above for --resume-run: authorization must run
// (and fail closed) before resolveResumeStatusAndDecision, and long before
// either resumeGateIngestIfConfigured or
// resumeGateIngestForUninitializedState ever construct a
// gatepersist.Coordinator.
func TestRunResumeRun_MismatchedSessionProducesZeroWrites(t *testing.T) {
	origResumeID, origSession := runResumeID, runSession
	t.Cleanup(func() {
		runResumeID = origResumeID
		runSession = origSession
	})

	runResumeID = "run-session-auth-3"
	runSession = "attacker-supplied-session"

	withRunClaimSvcOverride(t, &mockRunClaimService{
		getClaim: &models.EntityClaim{
			EntityType:    "task",
			EntityKey:     "E01-F01-001",
			SessionID:     "the-real-owning-session",
			LastHeartbeat: time.Now().UTC(),
		},
	})

	err := runResumeRun(context.Background(), "task", "E01-F01-001")
	if err == nil {
		t.Fatal("runResumeRun with a mismatched session: want error, got nil")
	}
	if !strings.Contains(err.Error(), "authorization failed") {
		t.Fatalf("error = %v, want an authorization failure (proves the run directory was never read and no coordinator call was made)", err)
	}
}
