package services

import (
	"context"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	claimrepo "github.com/jwwelbor/shark-task-manager/internal/repository/claim"
)

// mockClaimRepo is a function-field mock of ClaimRepository.
type mockClaimRepo struct {
	ClaimFn          func(ctx context.Context, c *models.EntityClaim) (*models.EntityClaim, error)
	GetFn            func(ctx context.Context, t, k string) (*models.EntityClaim, error)
	ReleaseFn        func(ctx context.Context, t, k string) (bool, error)
	ReleaseSessionFn func(ctx context.Context, t, k, s string) (bool, error)
	RenewFn          func(ctx context.Context, t, k, s string, p *float64, n string) (bool, error)
	ReclaimFn        func(ctx context.Context, ttl time.Duration) (int64, error)
	ListFn           func(ctx context.Context) ([]*models.EntityClaim, error)
}

func (m *mockClaimRepo) Claim(ctx context.Context, c *models.EntityClaim) (*models.EntityClaim, error) {
	return m.ClaimFn(ctx, c)
}
func (m *mockClaimRepo) Get(ctx context.Context, t, k string) (*models.EntityClaim, error) {
	return m.GetFn(ctx, t, k)
}
func (m *mockClaimRepo) Release(ctx context.Context, t, k string) (bool, error) {
	return m.ReleaseFn(ctx, t, k)
}
func (m *mockClaimRepo) ReleaseSession(ctx context.Context, t, k, s string) (bool, error) {
	return m.ReleaseSessionFn(ctx, t, k, s)
}
func (m *mockClaimRepo) Renew(ctx context.Context, t, k, s string, p *float64, n string) (bool, error) {
	return m.RenewFn(ctx, t, k, s, p, n)
}
func (m *mockClaimRepo) ReclaimExpired(ctx context.Context, ttl time.Duration) (int64, error) {
	return m.ReclaimFn(ctx, ttl)
}
func (m *mockClaimRepo) List(ctx context.Context) ([]*models.EntityClaim, error) {
	return m.ListFn(ctx)
}

func TestClaimService_Claim_ReclaimsBeforeClaiming(t *testing.T) {
	reclaimCalled := false
	m := &mockClaimRepo{
		ReclaimFn: func(ctx context.Context, ttl time.Duration) (int64, error) { reclaimCalled = true; return 0, nil },
		ClaimFn: func(ctx context.Context, c *models.EntityClaim) (*models.EntityClaim, error) {
			c.ID = 1
			return c, nil
		},
	}
	svc := NewClaimService(m, time.Minute)
	got, err := svc.Claim(context.Background(), ClaimInput{EntityType: "task", EntityKey: "E1-F1-001"})
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if !reclaimCalled {
		t.Error("expected ReclaimExpired to be called before claiming")
	}
	if got.SessionID == "" || got.ClaimedBy == "" {
		t.Errorf("claim defaults not filled: %+v", got)
	}
}

func TestClaimService_Claim_BlockedWhenLive(t *testing.T) {
	m := &mockClaimRepo{
		ReclaimFn: func(ctx context.Context, ttl time.Duration) (int64, error) { return 0, nil },
		ClaimFn: func(ctx context.Context, c *models.EntityClaim) (*models.EntityClaim, error) {
			return nil, claimrepo.ErrAlreadyClaimed
		},
		GetFn: func(ctx context.Context, t, k string) (*models.EntityClaim, error) {
			return &models.EntityClaim{EntityType: t, EntityKey: k, ClaimedBy: "other", SessionID: "s9", ClaimedAt: time.Now()}, nil
		},
	}
	svc := NewClaimService(m, time.Minute)
	_, err := svc.Claim(context.Background(), ClaimInput{EntityType: "task", EntityKey: "E1-F1-001"})
	if err == nil {
		t.Fatal("expected error claiming a live-claimed entity")
	}
}

func TestClaimService_Claim_ForceSteals(t *testing.T) {
	released := false
	claimCalls := 0
	m := &mockClaimRepo{
		ReclaimFn: func(ctx context.Context, ttl time.Duration) (int64, error) { return 0, nil },
		ClaimFn: func(ctx context.Context, c *models.EntityClaim) (*models.EntityClaim, error) {
			claimCalls++
			if claimCalls == 1 {
				return nil, claimrepo.ErrAlreadyClaimed
			}
			c.ID = 2
			return c, nil
		},
		ReleaseFn: func(ctx context.Context, t, k string) (bool, error) { released = true; return true, nil },
	}
	svc := NewClaimService(m, time.Minute)
	got, err := svc.Claim(context.Background(), ClaimInput{EntityType: "task", EntityKey: "E1-F1-001", Force: true})
	if err != nil {
		t.Fatalf("force claim: %v", err)
	}
	if !released {
		t.Error("force claim should release the existing claim first")
	}
	if got.ID != 2 {
		t.Errorf("expected re-claim after steal, got ID %d", got.ID)
	}
}

func TestClaimService_IsClaimable(t *testing.T) {
	now := time.Now().UTC()
	cases := []struct {
		name string
		get  *models.EntityClaim
		want bool
	}{
		{"unclaimed", nil, true},
		{"live claim", &models.EntityClaim{LastHeartbeat: now}, false},
		{"expired claim", &models.EntityClaim{LastHeartbeat: now.Add(-2 * time.Hour)}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &mockClaimRepo{
				GetFn: func(ctx context.Context, t, k string) (*models.EntityClaim, error) { return tc.get, nil },
			}
			svc := NewClaimService(m, time.Hour)
			got, err := svc.IsClaimable(context.Background(), "task", "E1-F1-001")
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("IsClaimable = %v, want %v", got, tc.want)
			}
		})
	}
}
