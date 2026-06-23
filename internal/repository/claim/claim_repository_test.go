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
	"github.com/jwwelbor/shark-task-manager/internal/test"
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

func TestClaim_Renew(t *testing.T) {
	repo, _ := setup(t)
	ctx := context.Background()
	if _, err := repo.Claim(ctx, newClaim("E01-F01-004")); err != nil {
		t.Fatalf("claim: %v", err)
	}
	prog := 0.42
	ok, err := repo.Renew(ctx, "task", "E01-F01-004", "sess-abc", &prog, "halfway")
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
	ok, _ = repo.Renew(ctx, "task", "E01-F01-004", "wrong", nil, "")
	if ok {
		t.Error("renew with wrong session should not update")
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
