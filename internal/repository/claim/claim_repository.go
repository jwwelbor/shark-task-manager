// Package claim provides data access for entity claims (the in-flight lease
// introduced in E35-F03). It owns SQL only — no business logic, no workflow
// knowledge.
package claim

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
)

// ErrAlreadyClaimed is returned by Claim when the entity already has a claim.
var ErrAlreadyClaimed = errors.New("entity is already claimed")

// Repository handles read/write operations on the entity_claims table.
type Repository struct {
	db *dbconn.DB
}

// NewRepository creates a claim Repository backed by db.
func NewRepository(db *dbconn.DB) *Repository {
	return &Repository{db: db}
}

// Claim atomically records a claim on (entityType, entityKey). Because the
// table has UNIQUE(entity_type, entity_key), a second claim on the same entity
// fails the constraint and is reported as ErrAlreadyClaimed — this is the
// single-grab guarantee. Returns the created claim on success.
func (r *Repository) Claim(ctx context.Context, c *models.EntityClaim) (*models.EntityClaim, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	const q = `
		INSERT INTO entity_claims (entity_type, entity_key, claimed_by, session_id, progress, note, harness, harness_version, harness_model)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := r.db.ExecContext(ctx, q, c.EntityType, c.EntityKey, c.ClaimedBy, c.SessionID, c.Progress, nullString(c.Note),
		nullString(c.Harness), nullString(c.HarnessVersion), nullString(c.HarnessModel))
	if err != nil {
		if repoerr.IsSQLiteUniqueViolation(err) {
			return nil, ErrAlreadyClaimed
		}
		return nil, fmt.Errorf("claim %s/%s: %w", c.EntityType, c.EntityKey, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("claim %s/%s: last insert id: %w", c.EntityType, c.EntityKey, err)
	}
	return r.getByID(ctx, id)
}

// Get returns the claim for an entity, or (nil, nil) when unclaimed.
func (r *Repository) Get(ctx context.Context, entityType, entityKey string) (*models.EntityClaim, error) {
	const q = `
		SELECT id, entity_type, entity_key, claimed_by, session_id, claimed_at, last_heartbeat, progress, note, harness, harness_version, harness_model
		FROM entity_claims WHERE entity_type = ? AND entity_key = ?
	`
	return r.scanOne(r.db.QueryRowContext(ctx, q, entityType, entityKey))
}

// VerifySessionWithTx checks that the expected session still owns a live
// claim and acquires SQLite's writer lock through a no-op update. A caller
// that proceeds to a child mutation in the same transaction cannot race a
// release, reissue, or expiry-reclaim between this check and commit.
func (r *Repository) VerifySessionWithTx(ctx context.Context, tx *sql.Tx, entityType, entityKey, sessionID string, ttl time.Duration) (bool, error) {
	if tx == nil {
		return false, fmt.Errorf("verify claim %s/%s: transaction is required", entityType, entityKey)
	}
	const q = `
		UPDATE entity_claims SET session_id = session_id
		WHERE entity_type = ? AND entity_key = ? AND session_id = ?
		  AND (? <= 0 OR unixepoch(CURRENT_TIMESTAMP) - unixepoch(last_heartbeat) <= ?)
	`
	res, err := tx.ExecContext(ctx, q, entityType, entityKey, sessionID, ttl.Seconds(), ttl.Seconds())
	if err != nil {
		return false, fmt.Errorf("verify claim %s/%s: %w", entityType, entityKey, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("verify claim %s/%s: rows affected: %w", entityType, entityKey, err)
	}
	return n == 1, nil
}

// Release deletes the claim for an entity regardless of session. Returns true
// when a row was removed. Used by administrative/force release.
func (r *Repository) Release(ctx context.Context, entityType, entityKey string) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM entity_claims WHERE entity_type = ? AND entity_key = ?`, entityType, entityKey)
	if err != nil {
		return false, fmt.Errorf("release %s/%s: %w", entityType, entityKey, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("release %s/%s: rows affected: %w", entityType, entityKey, err)
	}
	return n > 0, nil
}

// ReleaseSession deletes the claim only if it is held by the given session.
// This is the safe sync-release used on agent exit: it will not steal a claim
// that has since been reclaimed and re-issued to a different session.
func (r *Repository) ReleaseSession(ctx context.Context, entityType, entityKey, sessionID string) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM entity_claims WHERE entity_type = ? AND entity_key = ? AND session_id = ?`,
		entityType, entityKey, sessionID)
	if err != nil {
		return false, fmt.Errorf("release session %s/%s: %w", entityType, entityKey, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("release session %s/%s: rows affected: %w", entityType, entityKey, err)
	}
	return n > 0, nil
}

// Renew updates the heartbeat (and optional progress/note) for a claim held by
// the given session. Returns true when a row was updated. A heartbeat does
// triple duty: lease renewal, progress reporting, and telemetry source.
func (r *Repository) Renew(ctx context.Context, entityType, entityKey, sessionID string, progress *float64, note string, ttl time.Duration) (bool, error) {
	const q = `
		UPDATE entity_claims
		SET last_heartbeat = CURRENT_TIMESTAMP,
		    progress = COALESCE(?, progress),
		    note = COALESCE(?, note)
		WHERE entity_type = ? AND entity_key = ? AND session_id = ?
		  AND (? <= 0 OR unixepoch(CURRENT_TIMESTAMP) - unixepoch(last_heartbeat) <= ?)
	`
	// Use the database clock in the write itself so waiting for a writer
	// cannot make a previously calculated cutoff authorize a late renewal.
	res, err := r.db.ExecContext(ctx, q, progress, nullString(note), entityType, entityKey, sessionID, ttl.Seconds(), ttl.Seconds())
	if err != nil {
		return false, fmt.Errorf("renew %s/%s: %w", entityType, entityKey, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("renew %s/%s: rows affected: %w", entityType, entityKey, err)
	}
	return n > 0, nil
}

// ReclaimExpired deletes claims whose last heartbeat is older than ttl. Returns
// the number reclaimed. This is the universal crash-recovery backstop: a lease
// whose holder died (no sync-release, no heartbeats) is freed for redispatch.
// A non-positive ttl is a no-op (never expire).
func (r *Repository) ReclaimExpired(ctx context.Context, ttl time.Duration) (int64, error) {
	if ttl <= 0 {
		return 0, nil
	}
	cutoff := time.Now().UTC().Add(-ttl)
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM entity_claims WHERE last_heartbeat < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("reclaim expired: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("reclaim expired: rows affected: %w", err)
	}
	return n, nil
}

// List returns all current claims ordered by claim time.
func (r *Repository) List(ctx context.Context) ([]*models.EntityClaim, error) {
	const q = `
		SELECT id, entity_type, entity_key, claimed_by, session_id, claimed_at, last_heartbeat, progress, note, harness, harness_version, harness_model
		FROM entity_claims ORDER BY claimed_at
	`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list claims: %w", err)
	}
	defer rows.Close()
	var out []*models.EntityClaim
	for rows.Next() {
		c, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) getByID(ctx context.Context, id int64) (*models.EntityClaim, error) {
	const q = `
		SELECT id, entity_type, entity_key, claimed_by, session_id, claimed_at, last_heartbeat, progress, note, harness, harness_version, harness_model
		FROM entity_claims WHERE id = ?
	`
	return r.scanOne(r.db.QueryRowContext(ctx, q, id))
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func (r *Repository) scanOne(row rowScanner) (*models.EntityClaim, error) {
	c, err := scanRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func scanRow(row rowScanner) (*models.EntityClaim, error) {
	var c models.EntityClaim
	var progress sql.NullFloat64
	var note sql.NullString
	var harness, harnessVersion, harnessModel sql.NullString
	if err := row.Scan(&c.ID, &c.EntityType, &c.EntityKey, &c.ClaimedBy, &c.SessionID,
		&c.ClaimedAt, &c.LastHeartbeat, &progress, &note, &harness, &harnessVersion, &harnessModel); err != nil {
		return nil, err
	}
	if progress.Valid {
		c.Progress = &progress.Float64
	}
	if note.Valid {
		c.Note = note.String
	}
	// NULL maps to "" (unknown harness) — spec.md §3.1, D-F01-03.
	c.Harness = harness.String
	c.HarnessVersion = harnessVersion.String
	c.HarnessModel = harnessModel.String
	return &c, nil
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
