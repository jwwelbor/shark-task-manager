package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
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

// WorkSessionLog is the optional session-journal surface ClaimService writes
// to: a session opens when an entity is claimed and closes when the lease is
// released (or superseded by the next claim). It provides the
// active-vs-idle wall-clock split that entity_history cannot.
type WorkSessionLog interface {
	Open(ctx context.Context, session *models.WorkSession) error
	CloseOpenForEntity(ctx context.Context, entityType, entityKey, outcome string, endedAt time.Time) (int64, error)
}

// TaskKeyResolver resolves a task key to its numeric task row so claim-created
// task sessions remain visible to legacy task_id-based analytics/resume flows.
type TaskKeyResolver interface {
	GetByKey(ctx context.Context, key string) (*models.Task, error)
}

// ClaimService orchestrates the claim/session lease (E35-F03). Status is a pure
// phase; the claim is the in-flight lease. The service owns lease policy (TTL
// backstop, reclaim-before-claim, force-steal) on top of the repository.
type ClaimService struct {
	repo         ClaimRepository
	ttl          time.Duration
	sessions     WorkSessionLog // optional; nil degrades gracefully
	taskResolver TaskKeyResolver
}

// SetSessionLog attaches the optional work-session journal. Session writes
// are best-effort telemetry: failures are logged, never propagated, so a
// broken journal can never wedge the lease machinery.
func (s *ClaimService) SetSessionLog(log WorkSessionLog) {
	s.sessions = log
}

// SetTaskResolver attaches the optional task-key resolver used to backfill
// task_id when journaling task claims. Non-task entities ignore it.
func (s *ClaimService) SetTaskResolver(resolver TaskKeyResolver) {
	s.taskResolver = resolver
}

// NewClaimService constructs a ClaimService. A nil ttl falls back to
// DefaultClaimTTL (or the SHARK_CLAIM_TTL_SECONDS override). A non-nil zero
// ttl explicitly disables claim expiry.
func NewClaimService(repo ClaimRepository, ttl *time.Duration) *ClaimService {
	return &ClaimService{repo: repo, ttl: resolveClaimTTL(ttl)}
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

	// Harness identity (E34-F01, spec.md REQ-F-001). Optional and
	// independently settable; supplying none is valid. Callers are expected
	// to have already normalized Harness (trimmed, lowercased) and
	// HarnessVersion/HarnessModel (trimmed only) per REQ-F-001 — this input
	// carries the values through unchanged.
	Harness        string
	HarnessVersion string
	HarnessModel   string
}

// Claim leases an entity. Expired leases are reclaimed first (TTL backstop); a
// live lease blocks the claim unless Force is set, in which case it is stolen.
// Returns the resulting claim (including the session id to use for heartbeats
// and release).
func (s *ClaimService) Claim(ctx context.Context, in ClaimInput) (*models.EntityClaim, error) {
	// Always sweep expired leases first so a dead holder never wedges an entity.
	if s.ttl > 0 {
		if _, err := s.repo.ReclaimExpired(ctx, s.ttl); err != nil {
			return nil, fmt.Errorf("reclaim expired before claim: %w", err)
		}
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
		EntityType:     in.EntityType,
		EntityKey:      in.EntityKey,
		ClaimedBy:      by,
		SessionID:      session,
		Harness:        in.Harness,
		HarnessVersion: in.HarnessVersion,
		HarnessModel:   in.HarnessModel,
	}
	// Validate before touching the repository (REQ-NF-004, AC-10 / TC-013):
	// an oversized harness field must reject the claim with no partial row
	// ever written, so the repository's Claim must not be reached at all.
	if err := c.Validate(); err != nil {
		return nil, err
	}

	claimed, err := s.repo.Claim(ctx, c)
	if err == nil {
		s.openSession(ctx, claimed)
		return claimed, nil
	}
	if !errors.Is(err, claimrepo.ErrAlreadyClaimed) {
		return nil, err
	}
	// Already claimed and not expired.
	if !in.Force {
		// A Get error here only costs us the richer "claimed by …" detail; fall
		// back to the bare ErrAlreadyClaimed sentinel rather than masking the
		// original conflict with a lookup error.
		existing, _ := s.repo.Get(ctx, in.EntityType, in.EntityKey)
		if existing != nil {
			return nil, fmt.Errorf("%s %s is already claimed by %s (session %s) since %s; use --force to steal: %w",
				in.EntityType, in.EntityKey, existing.ClaimedBy, existing.SessionID, existing.ClaimedAt.Format(time.RFC3339), claimrepo.ErrAlreadyClaimed)
		}
		return nil, claimrepo.ErrAlreadyClaimed
	}
	// Force: release the existing claim and re-claim. The release+claim pair is
	// not atomic, so a concurrent claimant can grab the lease in the gap; wrap
	// that loss with race context so the caller gets an actionable message
	// instead of a bare ErrAlreadyClaimed.
	if _, err := s.repo.Release(ctx, in.EntityType, in.EntityKey); err != nil {
		return nil, fmt.Errorf("force-release existing claim: %w", err)
	}
	claimed, err = s.repo.Claim(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("lost steal race after releasing existing claim on %s %s: %w",
			in.EntityType, in.EntityKey, err)
	}
	s.openSession(ctx, claimed)
	return claimed, nil
}

// Release frees the lease on an entity. When sessionID is non-empty the release
// is session-scoped (safe sync-release that won't steal a re-issued lease).
// Unscoped administrative release requires force=true. A non-empty outcome
// (typically the semantic outcome the worker released — pass/fail/blocked) is
// stamped on the work session being closed; empty defaults to "released".
func (s *ClaimService) Release(ctx context.Context, entityType, entityKey, sessionID, outcome string, force bool) (bool, error) {
	var released bool
	var err error
	if sessionID != "" {
		released, err = s.repo.ReleaseSession(ctx, entityType, entityKey, sessionID)
	} else {
		if !force {
			return false, fmt.Errorf("unscoped release requires --force or a matching --session")
		}
		released, err = s.repo.Release(ctx, entityType, entityKey)
	}
	if err == nil && released {
		if outcome == "" {
			outcome = "released"
		}
		s.closeSessions(ctx, entityType, entityKey, outcome)
	}
	return released, err
}

// openSession journals the start of a work session for a fresh claim. Any
// still-open session for the entity (dead holder, force-steal) is closed as
// "superseded" first so at most one session per entity is ever open.
func (s *ClaimService) openSession(ctx context.Context, c *models.EntityClaim) {
	if s.sessions == nil || c == nil {
		return
	}
	s.closeSessions(ctx, c.EntityType, c.EntityKey, "superseded")
	ws := &models.WorkSession{
		EntityType: c.EntityType,
		EntityKey:  c.EntityKey,
		AgentID:    &c.ClaimedBy,
		SessionID:  &c.SessionID,
		StartedAt:  time.Now().UTC(),
	}
	if c.EntityType == string(models.EntityTypeTask) && s.taskResolver != nil {
		task, err := s.taskResolver.GetByKey(ctx, c.EntityKey)
		if err != nil {
			slog.Warn("failed to resolve task for work session", "task", c.EntityKey, "error", err)
		} else if task != nil {
			ws.TaskID = task.ID
		}
	}
	if err := s.sessions.Open(ctx, ws); err != nil {
		slog.Warn("failed to open work session", "entity", c.EntityKey, "error", err)
	}
}

// closeSessions ends any open work session for the entity with the given
// outcome. Best-effort: errors are logged, never propagated.
func (s *ClaimService) closeSessions(ctx context.Context, entityType, entityKey, outcome string) {
	if s.sessions == nil {
		return
	}
	if _, err := s.sessions.CloseOpenForEntity(ctx, entityType, entityKey, outcome, time.Now().UTC()); err != nil {
		slog.Warn("failed to close work session", "entity", entityKey, "outcome", outcome, "error", err)
	}
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
	if s.ttl > 0 {
		if _, err := s.repo.ReclaimExpired(ctx, s.ttl); err != nil {
			return nil, err
		}
	}
	return s.repo.List(ctx)
}

// ListActiveReadOnly returns claims that are active at evaluatedAt without
// reclaiming expired rows or otherwise mutating claim state.
func (s *ClaimService) ListActiveReadOnly(ctx context.Context, evaluatedAt time.Time) ([]*models.EntityClaim, error) {
	claims, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active claims: %w", err)
	}
	return s.FilterActiveReadOnly(claims, evaluatedAt), nil
}

// FilterActiveReadOnly applies configured lease expiry to already-loaded claim
// rows. It performs no database access.
func (s *ClaimService) FilterActiveReadOnly(
	claims []*models.EntityClaim,
	evaluatedAt time.Time,
) []*models.EntityClaim {
	active := make([]*models.EntityClaim, 0, len(claims))
	for _, claim := range claims {
		if claim != nil && !claim.IsExpired(evaluatedAt, s.ttl) {
			active = append(active, claim)
		}
	}
	return active
}

// ReclaimExpired frees all leases whose heartbeats are stale and returns the
// count reclaimed.
func (s *ClaimService) ReclaimExpired(ctx context.Context) (int64, error) {
	if s.ttl <= 0 {
		return 0, nil
	}
	return s.repo.ReclaimExpired(ctx, s.ttl)
}

// --- helpers ---

func resolveClaimTTL(ttl *time.Duration) time.Duration {
	if ttl != nil {
		return *ttl
	}
	return claimTTLFromEnv()
}

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
