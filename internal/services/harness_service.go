package services

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// HarnessIdentity captures the harness type, version, and model resolved
// for prompt rendering (spec.md §3.2, REQ-F-002/REQ-F-003).
type HarnessIdentity struct {
	Type    string
	Version string
	Model   string
}

// IsZero reports whether every field is unset.
func (i HarnessIdentity) IsZero() bool {
	return i.Type == "" && i.Version == "" && i.Model == ""
}

// Vars returns all three harness placeholder keys — "harness",
// "harness_version", "harness_model" — unconditionally, mapping any unset
// field to the empty string. Keys are never omitted.
//
// This is load-bearing, not stylistic (spec.md D-F01-07): Go's
// text/template fails a typed-helper render (e.g. `{{if isClaude .harness}}`)
// with "invalid value; expected string" when a map key is entirely absent,
// even though the same template renders fine when the key is present but
// empty. A future refactor that conditionally inserts keys only when
// non-empty (`if v != "" { m[k] = v }`) would silently reintroduce that
// failure — see TestHarnessIdentity_Vars_ZeroValue_AlwaysThreeKeys.
func (i HarnessIdentity) Vars() map[string]string {
	return map[string]string{
		"harness":         i.Type,
		"harness_version": i.Version,
		"harness_model":   i.Model,
	}
}

// ClaimReader is the one-method consumer-side interface HarnessResolver
// needs from the claim store, per the accept-interfaces rule in
// .claude/rules/go/patterns.md. Satisfied as-is by
// internal/repository/claim.Repository.Get — do not redefine that method
// elsewhere with a different shape.
type ClaimReader interface {
	Get(ctx context.Context, entityType, entityKey string) (*models.EntityClaim, error)
}

// HarnessResolver resolves harness identity per spec.md REQ-F-002's
// precedence, evaluated per field (D-F01-04): explicit override (flag) >
// active claim > environment > zero (unset).
//
// "Active" is load-bearing (REQ-F-002): a claim row whose lease has expired
// (stale heartbeat, not yet swept) is not an active claim, and its harness
// identity must never render into live prompts. Resolve enforces this with
// the same claim.IsExpired(now, ttl) check ClaimService already applies via
// IsClaimable/FilterActiveReadOnly (claim_service.go) — this was the
// T-E34-F01-003 UAT rejection's defect: Resolve originally used a claim's
// harness fields with no expiry check at all.
type HarnessResolver struct {
	claims ClaimReader
	ttl    time.Duration
}

// NewHarnessResolver constructs a HarnessResolver backed by claims. claims
// may be nil, in which case claim lookups are skipped and resolution falls
// through to environment / zero for every field. The default lease TTL is
// resolved the same way ClaimService's default resolves it (claimTTLFromEnv:
// SHARK_CLAIM_TTL_SECONDS, else DefaultClaimTTL) — call SetTTL to align this
// resolver with a ClaimService constructed from `.sharkconfig.json`'s
// claim_ttl_seconds, exactly as GetHarnessResolver does.
func NewHarnessResolver(claims ClaimReader) *HarnessResolver {
	return &HarnessResolver{claims: claims, ttl: claimTTLFromEnv()}
}

// SetTTL overrides the lease TTL used to judge claim expiry. Without this,
// Resolve's expiry check silently disagrees with a ClaimService configured
// via `.sharkconfig.json`'s claim_ttl_seconds (including claim_ttl_seconds:
// 0, which disables expiry entirely per docs/guides/route-based-workflow.md
// §4) — the same "claim liveness must defer to the configured authority"
// principle the T-E34-F01-003 rework fix is about, just one level up.
func (r *HarnessResolver) SetTTL(ttl time.Duration) { r.ttl = ttl }

// TTL returns the configured lease TTL, mirroring ClaimService.TTL(). Used to
// verify GetHarnessResolver's config wiring without duplicating Resolve's
// expiry-check behavior in an integration test.
func (r *HarnessResolver) TTL() time.Duration { return r.ttl }

// Resolve returns the harness identity for (entityType, entityKey), given an
// explicit override (e.g. CLI flags). Precedence is evaluated independently
// per field, so a claim may supply the type while an override supplies only
// the model (spec.md REQ-F-002/D-F01-04).
//
// A claim-read error is logged and swallowed to the zero value for every
// field the claim would have supplied — it never fails the render
// (REQ-NF-002, D-F01-05). Resolve therefore always returns a nil error; the
// error return exists to satisfy the documented contract shape in spec.md
// §3.2 and to leave room for a future failure mode without a signature
// change.
func (r *HarnessResolver) Resolve(ctx context.Context, entityType, entityKey string, override HarnessIdentity) (HarnessIdentity, error) {
	var claimed HarnessIdentity
	if r.claims != nil {
		claim, err := r.claims.Get(ctx, entityType, entityKey)
		if err != nil {
			slog.Warn("harness resolution: failed to read claim; degrading to unresolved harness identity",
				"entity_type", entityType, "entity_key", entityKey, "error", err)
		} else if claim != nil && claim.IsExpired(time.Now().UTC(), r.ttl) {
			// Stale-lease-as-live-input defect (T-E34-F01-003 rework): an
			// expired-but-unswept claim row is not an active claim per
			// REQ-F-002 — treat it exactly like no claim at all rather than
			// rendering its harness identity into live prompts.
			slog.Warn("harness resolution: claim lease expired; degrading to unresolved harness identity",
				"entity_type", entityType, "entity_key", entityKey, "session_id", claim.SessionID)
		} else if claim != nil {
			claimed = HarnessIdentity{Type: claim.Harness, Version: claim.HarnessVersion, Model: claim.HarnessModel}
		}
	}

	env := HarnessIdentity{
		Type:    os.Getenv("SHARK_HARNESS"),
		Version: os.Getenv("SHARK_HARNESS_VERSION"),
		Model:   os.Getenv("SHARK_HARNESS_MODEL"),
	}

	return HarnessIdentity{
		Type:    resolveHarnessField(override.Type, claimed.Type, env.Type),
		Version: resolveHarnessField(override.Version, claimed.Version, env.Version),
		Model:   resolveHarnessField(override.Model, claimed.Model, env.Model),
	}, nil
}

// resolveHarnessField applies the flag > claim > env precedence to a single
// field, returning the empty string when none of the three sources set it.
func resolveHarnessField(flag, claim, env string) string {
	if flag != "" {
		return flag
	}
	if claim != "" {
		return claim
	}
	return env
}
