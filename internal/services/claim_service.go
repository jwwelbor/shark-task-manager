package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	claimrepo "github.com/jwwelbor/shark-task-manager/internal/repository/claim"
)

// DefaultClaimTTL is the lease time-to-live backstop. A claim whose heartbeat
// goes stale for longer than this is reclaimable by ReclaimExpired (and is
// auto-reclaimed before a fresh Claim). Expressed as one global cadence rather
// than per-step timeouts (design §6: TTL = K missed updates).
const DefaultClaimTTL = 15 * time.Minute

// ClaimRepository is the data-access surface ClaimService depends on (E35-F03).
type ClaimRepository interface {
	Claim(ctx context.Context, c *models.EntityClaim) (*models.EntityClaim, error)
	Get(ctx context.Context, entityType, entityKey string) (*models.EntityClaim, error)
	Release(ctx context.Context, entityType, entityKey string) (bool, error)
	ReleaseSession(ctx context.Context, entityType, entityKey, sessionID string) (bool, error)
	Renew(ctx context.Context, entityType, entityKey, sessionID string, progress *float64, note string) (bool, error)
	ReclaimExpired(ctx context.Context, ttl time.Duration) (int64, error)
	List(ctx context.Context) ([]*models.EntityClaim, error)
}

// ClaimService orchestrates the claim/session lease (E35-F03). Status is a pure
// phase; the claim is the in-flight lease. The service owns lease policy (TTL
// backstop, reclaim-before-claim, force-steal) on top of the repository.
type ClaimService struct {
	repo ClaimRepository
	ttl  time.Duration
}

// NewClaimService constructs a ClaimService. A non-positive ttl falls back to
// DefaultClaimTTL (or the SHARK_CLAIM_TTL_SECONDS override).
func NewClaimService(repo ClaimRepository, ttl time.Duration) *ClaimService {
	if ttl <= 0 {
		ttl = claimTTLFromEnv()
	}
	return &ClaimService{repo: repo, ttl: ttl}
}

// TTL returns the configured lease time-to-live.
func (s *ClaimService) TTL() time.Duration { return s.ttl }

// ClaimInput holds the parameters for claiming an entity.
type ClaimInput struct {
	EntityType string
	EntityKey  string
	ClaimedBy  string // agent/user identity; defaults to $SHARK_ACTOR or "cli"
	SessionID  string // lease id; generated when empty
	Force      bool   // steal an existing (even live) claim
}

// Claim leases an entity. Expired leases are reclaimed first (TTL backstop); a
// live lease blocks the claim unless Force is set, in which case it is stolen.
// Returns the resulting claim (including the session id to use for heartbeats
// and release).
func (s *ClaimService) Claim(ctx context.Context, in ClaimInput) (*models.EntityClaim, error) {
	// Always sweep expired leases first so a dead holder never wedges an entity.
	if _, err := s.repo.ReclaimExpired(ctx, s.ttl); err != nil {
		return nil, fmt.Errorf("reclaim expired before claim: %w", err)
	}

	by := in.ClaimedBy
	if by == "" {
		by = actorIdentity()
	}
	session := in.SessionID
	if session == "" {
		session = newSessionID()
	}

	c := &models.EntityClaim{
		EntityType: in.EntityType,
		EntityKey:  in.EntityKey,
		ClaimedBy:  by,
		SessionID:  session,
	}

	claimed, err := s.repo.Claim(ctx, c)
	if err == nil {
		return claimed, nil
	}
	if !errors.Is(err, claimrepo.ErrAlreadyClaimed) {
		return nil, err
	}
	// Already claimed and not expired.
	if !in.Force {
		existing, _ := s.repo.Get(ctx, in.EntityType, in.EntityKey)
		if existing != nil {
			return nil, fmt.Errorf("%s %s is already claimed by %s (session %s) since %s; use --force to steal",
				in.EntityType, in.EntityKey, existing.ClaimedBy, existing.SessionID, existing.ClaimedAt.Format(time.RFC3339))
		}
		return nil, claimrepo.ErrAlreadyClaimed
	}
	// Force: release the existing claim and re-claim.
	if _, err := s.repo.Release(ctx, in.EntityType, in.EntityKey); err != nil {
		return nil, fmt.Errorf("force-release existing claim: %w", err)
	}
	return s.repo.Claim(ctx, c)
}

// Release frees the lease on an entity. When sessionID is non-empty the release
// is session-scoped (safe sync-release that won't steal a re-issued lease);
// otherwise it is an unconditional administrative release.
func (s *ClaimService) Release(ctx context.Context, entityType, entityKey, sessionID string) (bool, error) {
	if sessionID != "" {
		return s.repo.ReleaseSession(ctx, entityType, entityKey, sessionID)
	}
	return s.repo.Release(ctx, entityType, entityKey)
}

// Heartbeat renews a lease and records optional progress/note. The triple-duty
// update (lease + progress + telemetry) returns an error if no live lease for
// the session exists.
func (s *ClaimService) Heartbeat(ctx context.Context, entityType, entityKey, sessionID string, progress *float64, note string) error {
	ok, err := s.repo.Renew(ctx, entityType, entityKey, sessionID, progress, note)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("no active claim for %s %s held by session %s", entityType, entityKey, sessionID)
	}
	return nil
}

// Get returns the current claim for an entity, or nil when unclaimed.
func (s *ClaimService) Get(ctx context.Context, entityType, entityKey string) (*models.EntityClaim, error) {
	return s.repo.Get(ctx, entityType, entityKey)
}

// IsClaimable reports whether an entity can currently be picked up: unclaimed,
// or claimed by an expired lease. Used by `shark next` to hand out only
// unclaimed entities.
func (s *ClaimService) IsClaimable(ctx context.Context, entityType, entityKey string) (bool, error) {
	c, err := s.repo.Get(ctx, entityType, entityKey)
	if err != nil {
		return false, err
	}
	if c == nil {
		return true, nil
	}
	return c.IsExpired(time.Now().UTC(), s.ttl), nil
}

// List returns all current claims, sweeping expired leases first.
func (s *ClaimService) List(ctx context.Context) ([]*models.EntityClaim, error) {
	if _, err := s.repo.ReclaimExpired(ctx, s.ttl); err != nil {
		return nil, err
	}
	return s.repo.List(ctx)
}

// ReclaimExpired frees all leases whose heartbeats are stale and returns the
// count reclaimed.
func (s *ClaimService) ReclaimExpired(ctx context.Context) (int64, error) {
	return s.repo.ReclaimExpired(ctx, s.ttl)
}

// --- helpers ---

func claimTTLFromEnv() time.Duration {
	if v := os.Getenv("SHARK_CLAIM_TTL_SECONDS"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return DefaultClaimTTL
}

func actorIdentity() string {
	if a := os.Getenv("SHARK_ACTOR"); a != "" {
		return a
	}
	return "cli"
}

// newSessionID returns a short random hex lease id. crypto/rand keeps it
// collision-free across concurrent agents without an external uuid dependency.
func newSessionID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fall back to a timestamp-derived id; uniqueness is best-effort here.
		return "sess-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 36)
	}
	return "sess-" + hex.EncodeToString(b[:])
}
