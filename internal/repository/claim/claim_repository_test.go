// Package claim_test provides integration tests for the claim Repository.
// Per .claude/rules/testing/repository-tests.md these use the real test DB via
// test.NewIsolatedTestDB(t) for full isolation.
package claim_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/claim"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) (*claim.Repository, *dbconn.DB) {
	t.Helper()
	database := test.NewIsolatedTestDB(t)
	db := dbconn.NewDB(database)
	return claim.NewRepository(db), db
}

func newClaim(key string) *models.EntityClaim {
	return &models.EntityClaim{
		EntityType: "task",
		EntityKey:  key,
		ClaimedBy:  "agent-1",
		SessionID:  "sess-abc",
	}
}

func TestClaim_SingleGrab(t *testing.T) {
	repo, _ := setup(t)
	ctx := context.Background()

	got, err := repo.Claim(ctx, newClaim("E01-F01-001"))
	if err != nil {
		t.Fatalf("first claim failed: %v", err)
	}
	if got.ID == 0 || got.ClaimedBy != "agent-1" {
		t.Fatalf("unexpected claim: %+v", got)
	}

	// Second claim on the same entity must fail (single-grab).
	_, err = repo.Claim(ctx, &models.EntityClaim{
		EntityType: "task", EntityKey: "E01-F01-001", ClaimedBy: "agent-2", SessionID: "sess-xyz",
	})
	if !errors.Is(err, claim.ErrAlreadyClaimed) {
		t.Fatalf("expected ErrAlreadyClaimed, got %v", err)
	}
}

func TestClaim_GetAndRelease(t *testing.T) {
	repo, _ := setup(t)
	ctx := context.Background()

	if _, err := repo.Claim(ctx, newClaim("E01-F01-002")); err != nil {
		t.Fatalf("claim: %v", err)
	}
	got, err := repo.Get(ctx, "task", "E01-F01-002")
	if err != nil || got == nil {
		t.Fatalf("get: %v claim=%v", err, got)
	}

	released, err := repo.Release(ctx, "task", "E01-F01-002")
	if err != nil || !released {
		t.Fatalf("release: %v released=%v", err, released)
	}
	// Now unclaimed; a new claim succeeds.
	if _, err := repo.Claim(ctx, newClaim("E01-F01-002")); err != nil {
		t.Fatalf("re-claim after release: %v", err)
	}
	// Get on unclaimed entity returns nil, nil.
	if c, err := repo.Get(ctx, "task", "NOPE-001"); err != nil || c != nil {
		t.Fatalf("get unclaimed: %v claim=%v", err, c)
	}
}

func TestClaim_ReleaseSessionOnlyMatchingSession(t *testing.T) {
	repo, _ := setup(t)
	ctx := context.Background()
	if _, err := repo.Claim(ctx, newClaim("E01-F01-003")); err != nil {
		t.Fatalf("claim: %v", err)
	}
	// Wrong session does not release.
	released, err := repo.ReleaseSession(ctx, "task", "E01-F01-003", "other-session")
	if err != nil || released {
		t.Fatalf("expected no release for wrong session, got released=%v err=%v", released, err)
	}
	// Correct session releases.
	released, err = repo.ReleaseSession(ctx, "task", "E01-F01-003", "sess-abc")
	if err != nil || !released {
		t.Fatalf("expected release for correct session, got released=%v err=%v", released, err)
	}
}

func TestClaim_VerifySessionWithTx(t *testing.T) {
	repo, db := setup(t)
	ctx := context.Background()
	claimRecord := newClaim("E01-F01-003-tx")
	_, err := repo.Claim(ctx, claimRecord)
	require.NoError(t, err)

	ttl := time.Minute
	svc := services.NewClaimService(repo, &ttl)
	tx, err := db.BeginTxContext(ctx)
	require.NoError(t, err)
	require.NoError(t, svc.VerifySessionWithTx(ctx, tx, models.EntityTypeTask, claimRecord.EntityKey, claimRecord.SessionID))
	require.NoError(t, tx.Rollback())

	tx, err = db.BeginTxContext(ctx)
	require.NoError(t, err)
	err = svc.VerifySessionWithTx(ctx, tx, models.EntityTypeTask, claimRecord.EntityKey, "another-session")
	require.ErrorContains(t, err, "no longer matches or has expired")
	require.NoError(t, tx.Rollback())

	_, err = db.ExecContext(ctx, `UPDATE entity_claims SET last_heartbeat = datetime('now', '-2 minutes') WHERE entity_key = ?`, claimRecord.EntityKey)
	require.NoError(t, err)
	tx, err = db.BeginTxContext(ctx)
	require.NoError(t, err)
	err = svc.VerifySessionWithTx(ctx, tx, models.EntityTypeTask, claimRecord.EntityKey, claimRecord.SessionID)
	require.ErrorContains(t, err, "no longer matches or has expired")
	require.NoError(t, tx.Rollback())
}

func TestClaim_Renew(t *testing.T) {
	repo, _ := setup(t)
	ctx := context.Background()
	if _, err := repo.Claim(ctx, newClaim("E01-F01-004")); err != nil {
		t.Fatalf("claim: %v", err)
	}
	prog := 0.42
	ok, err := repo.Renew(ctx, "task", "E01-F01-004", "sess-abc", &prog, "halfway", time.Minute)
	if err != nil || !ok {
		t.Fatalf("renew: ok=%v err=%v", ok, err)
	}
	got, _ := repo.Get(ctx, "task", "E01-F01-004")
	if got.Progress == nil || *got.Progress != 0.42 {
		t.Errorf("progress = %v, want 0.42", got.Progress)
	}
	if got.Note != "halfway" {
		t.Errorf("note = %q, want halfway", got.Note)
	}
	// Renew with wrong session is a no-op.
	ok, _ = repo.Renew(ctx, "task", "E01-F01-004", "wrong", nil, "", time.Minute)
	if ok {
		t.Error("renew with wrong session should not update")
	}
}

// C30 / R24-01: a heartbeat is the first operation after the holder goes
// quiet. No competitor or sweep removes its row before the renewal decision.
func TestHeartbeatOnUnsweptLease(t *testing.T) {
	for _, tc := range []struct {
		name    string
		ttl     time.Duration
		age     int
		session string
		renew   bool
	}{
		{"within TTL", time.Minute, 50, "sess-abc", true},
		{"at TTL", time.Minute, 60, "sess-abc", true},
		{"past TTL", time.Minute, 61, "sess-abc", false},
		{"expiry disabled", 0, 3600, "sess-abc", true},
		{"wrong holder with expiry disabled", 0, 3600, "other", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, db := setup(t)
			ctx := context.Background()
			c := newClaim("E01-F01-001")
			_, err := repo.Claim(ctx, c)
			require.NoError(t, err)
			svc := services.NewClaimService(repo, &tc.ttl)
			progress := 0.75
			// SQLite writes lease timestamps at whole-second precision. When
			// the clock rolls into the next second during setup, reseed before
			// asserting an exact age.
			for attempt := 0; attempt < 3; attempt++ {
				var started, ended int64
				require.NoError(t, db.QueryRowContext(ctx, "SELECT unixepoch()").Scan(&started))
				_, err := db.ExecContext(ctx, `UPDATE entity_claims SET last_heartbeat = datetime(?, 'unixepoch'), progress = .25, note = 'original' WHERE entity_key = ?`, started-int64(tc.age), c.EntityKey)
				require.NoError(t, err)
				before, err := repo.Get(ctx, c.EntityType, c.EntityKey)
				require.NoError(t, err)
				renewErr := svc.Heartbeat(ctx, c.EntityType, c.EntityKey, tc.session, &progress, "updated")
				after, err := repo.Get(ctx, c.EntityType, c.EntityKey)
				require.NoError(t, err)
				require.NoError(t, db.QueryRowContext(ctx, "SELECT unixepoch()").Scan(&ended))
				if started != ended {
					continue
				}
				if tc.renew {
					require.NoError(t, renewErr)
					require.Equal(t, started, after.LastHeartbeat.Unix())
					require.Equal(t, &progress, after.Progress)
					require.Equal(t, "updated", after.Note)
				} else {
					require.ErrorContains(t, renewErr, "no active claim")
					require.Equal(t, before, after)
				}
				return
			}
			t.Fatal("could not pin both observations to the same database clock second")
		})
	}
}

func TestClaim_ReclaimExpired(t *testing.T) {
	repo, db := setup(t)
	ctx := context.Background()
	if _, err := repo.Claim(ctx, newClaim("E01-F01-005")); err != nil {
		t.Fatalf("claim: %v", err)
	}
	// Backdate the heartbeat well into the past.
	if _, err := db.ExecContext(ctx,
		`UPDATE entity_claims SET last_heartbeat = ? WHERE entity_key = ?`,
		time.Now().UTC().Add(-2*time.Hour), "E01-F01-005"); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	// TTL of 1h reclaims the stale lease.
	n, err := repo.ReclaimExpired(ctx, time.Hour)
	if err != nil || n != 1 {
		t.Fatalf("reclaim: n=%d err=%v", n, err)
	}
	if c, _ := repo.Get(ctx, "task", "E01-F01-005"); c != nil {
		t.Error("expired claim should have been reclaimed")
	}

	// ttl<=0 is a no-op.
	if _, err := repo.Claim(ctx, newClaim("E01-F01-006")); err != nil {
		t.Fatalf("claim: %v", err)
	}
	if n, _ := repo.ReclaimExpired(ctx, 0); n != 0 {
		t.Errorf("ttl=0 should reclaim nothing, got %d", n)
	}
}

func TestClaim_List(t *testing.T) {
	repo, _ := setup(t)
	ctx := context.Background()
	for _, k := range []string{"E01-F01-007", "E01-F01-008"} {
		if _, err := repo.Claim(ctx, newClaim(k)); err != nil {
			t.Fatalf("claim %s: %v", k, err)
		}
	}
	claims, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(claims) != 2 {
		t.Errorf("len(claims) = %d, want 2", len(claims))
	}
}

// TestClaim_HarnessFieldsRoundTrip is T-E34-F01-002's AC-T1: the three
// harness columns must flow through the real Claim insert, and back out of
// Get/List/getByID, via parameterized placeholders (spec.md §3.3). This is
// the repository-level counterpart to the mocked CLI-boundary TC-001 in
// test-plan.md, which never touches real SQL.
func TestClaim_HarnessFieldsRoundTrip(t *testing.T) {
	repo, _ := setup(t)
	ctx := context.Background()

	c := newClaim("E01-F01-009")
	c.Harness = "claude"
	c.HarnessVersion = "2.1.0"
	c.HarnessModel = "opus"

	created, err := repo.Claim(ctx, c)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if created.Harness != "claude" || created.HarnessVersion != "2.1.0" || created.HarnessModel != "opus" {
		t.Fatalf("Claim() return value harness fields = %+v", created)
	}

	// getByID path (exercised internally by Claim's return, but assert Get too).
	got, err := repo.Get(ctx, "task", "E01-F01-009")
	if err != nil || got == nil {
		t.Fatalf("get: %v claim=%v", err, got)
	}
	if got.Harness != "claude" || got.HarnessVersion != "2.1.0" || got.HarnessModel != "opus" {
		t.Errorf("Get() harness fields = %+v, want claude/2.1.0/opus", got)
	}

	claims, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, lc := range claims {
		if lc.EntityKey != "E01-F01-009" {
			continue
		}
		found = true
		if lc.Harness != "claude" || lc.HarnessVersion != "2.1.0" || lc.HarnessModel != "opus" {
			t.Errorf("List() harness fields = %+v, want claude/2.1.0/opus", lc)
		}
	}
	if !found {
		t.Fatal("List() did not return the claimed entity")
	}
}

// TestClaim_HarnessFieldsUnsetMapToEmptyString covers AC-11/D-F01-03's "NULL
// maps to empty string" contract: a claim with no harness fields supplied
// must round-trip as "" (not a sentinel), matching REQ-F-001 bullet 1.
func TestClaim_HarnessFieldsUnsetMapToEmptyString(t *testing.T) {
	repo, _ := setup(t)
	ctx := context.Background()

	if _, err := repo.Claim(ctx, newClaim("E01-F01-010")); err != nil {
		t.Fatalf("claim: %v", err)
	}
	got, err := repo.Get(ctx, "task", "E01-F01-010")
	if err != nil || got == nil {
		t.Fatalf("get: %v claim=%v", err, got)
	}
	if got.Harness != "" || got.HarnessVersion != "" || got.HarnessModel != "" {
		t.Errorf("harness fields = %+v, want all empty strings", got)
	}
}
