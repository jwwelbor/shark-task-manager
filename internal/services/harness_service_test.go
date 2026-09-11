package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeClaimReader is a minimal test double for the one-method ClaimReader
// interface (spec.md §3.2). HarnessResolver is a new type with no existing
// mock, so this is the "New test helpers needed" fixture named by
// test-plan.md's Test Infrastructure section.
type fakeClaimReader struct {
	claim *models.EntityClaim
	err   error
}

func (f *fakeClaimReader) Get(_ context.Context, _, _ string) (*models.EntityClaim, error) {
	return f.claim, f.err
}

// TC-015: HarnessIdentity{}.Vars() always returns exactly three keys.
//
// Asserts both len(m) == 3 and each key's presence via `_, ok := m[key]`,
// not just value equality — per spec.md D-F01-07, key *presence* is
// load-bearing, independent of value. A future "tidy up empty values"
// refactor using conditional insertion (e.g. `if v != "" { m[k] = v }`)
// must fail this test even though every individual value would look
// correct.
func TestHarnessIdentity_Vars_ZeroValue_AlwaysThreeKeys(t *testing.T) {
	var id HarnessIdentity

	m := id.Vars()

	require.Len(t, m, 3)
	for _, key := range []string{"harness", "harness_version", "harness_model"} {
		v, ok := m[key]
		assert.Truef(t, ok, "key %q must be present, not just correctly valued", key)
		assert.Equal(t, "", v)
	}
}

// TC-015 edge case: a non-zero HarnessIdentity still yields all three keys,
// including the unset ones as present-but-empty.
func TestHarnessIdentity_Vars_PartiallySet_StillThreeKeys(t *testing.T) {
	id := HarnessIdentity{Type: "claude"}

	m := id.Vars()

	require.Len(t, m, 3)
	assert.Equal(t, "claude", m["harness"])
	versionVal, versionOK := m["harness_version"]
	assert.True(t, versionOK, "harness_version key must be present even when unset")
	assert.Equal(t, "", versionVal)
	modelVal, modelOK := m["harness_model"]
	assert.True(t, modelOK, "harness_model key must be present even when unset")
	assert.Equal(t, "", modelVal)
}

func TestHarnessIdentity_IsZero(t *testing.T) {
	assert.True(t, HarnessIdentity{}.IsZero())
	assert.False(t, HarnessIdentity{Type: "claude"}.IsZero())
	assert.False(t, HarnessIdentity{Version: "2.1.0"}.IsZero())
	assert.False(t, HarnessIdentity{Model: "opus"}.IsZero())
}

// AC-T3 / isHarness precedent: unit tests for the resolver's per-field
// precedence (flag > claim > env > zero), the general form of the
// resolver logic that TC-005/006/007 (T-E34-F01-004) later drive end to
// end through runNext. Direct-call tests here match the belt-and-braces
// pattern test-plan.md's Test Infrastructure section describes for the
// FuncMap helpers: correctness in isolation, then wiring correctness
// through the CLI later.
func TestHarnessResolver_Resolve_FlagBeatsClaimBeatsEnv(t *testing.T) {
	t.Setenv("SHARK_HARNESS", "claude")
	t.Setenv("SHARK_HARNESS_VERSION", "")
	t.Setenv("SHARK_HARNESS_MODEL", "")

	reader := &fakeClaimReader{claim: &models.EntityClaim{Harness: "codex", LastHeartbeat: time.Now().UTC()}}
	resolver := NewHarnessResolver(reader)

	got, err := resolver.Resolve(context.Background(), "task", "E34-F01-001", HarnessIdentity{Type: "claude"})

	require.NoError(t, err)
	assert.Equal(t, "claude", got.Type)
}

func TestHarnessResolver_Resolve_PerFieldPrecedence(t *testing.T) {
	// D-F01-04: precedence is evaluated per field, not per source. A claim
	// supplies the type (codex) while env supplies the version (9.9) — a
	// resolver that resolved per-source would wrongly take the claim's
	// (empty) version instead of falling through to env.
	t.Setenv("SHARK_HARNESS", "")
	t.Setenv("SHARK_HARNESS_VERSION", "9.9")
	t.Setenv("SHARK_HARNESS_MODEL", "")

	reader := &fakeClaimReader{claim: &models.EntityClaim{Harness: "codex", LastHeartbeat: time.Now().UTC()}}
	resolver := NewHarnessResolver(reader)

	got, err := resolver.Resolve(context.Background(), "task", "E34-F01-001", HarnessIdentity{})

	require.NoError(t, err)
	assert.Equal(t, "codex", got.Type)
	assert.Equal(t, "9.9", got.Version)
	assert.Equal(t, "", got.Model)
}

// TestHarnessResolver_Resolve_ExpiredClaim_DegradesToZero is the rework
// regression test for the stale-lease-as-live-input defect found in UAT
// review of T-E34-F01-003: Resolve originally read the raw claim row via
// ClaimReader.Get with no expiry check at all, so an expired-but-unswept
// claim's harness identity still rendered into live prompts — violating
// REQ-F-002's "active claim" language. This claim's LastHeartbeat is far
// older than claim_service.go's DefaultClaimTTL (15m), so it must be treated
// exactly like no claim: degrading to the next precedence tier (env, then
// zero), the same check ClaimService.IsClaimable/FilterActiveReadOnly
// already apply via models.EntityClaim.IsExpired.
func TestHarnessResolver_Resolve_ExpiredClaim_DegradesToZero(t *testing.T) {
	t.Setenv("SHARK_HARNESS", "")
	t.Setenv("SHARK_HARNESS_VERSION", "")
	t.Setenv("SHARK_HARNESS_MODEL", "")
	t.Setenv("SHARK_CLAIM_TTL_SECONDS", "")

	reader := &fakeClaimReader{claim: &models.EntityClaim{
		Harness:       "codex",
		LastHeartbeat: time.Now().UTC().Add(-1 * time.Hour), // well past DefaultClaimTTL (15m)
	}}
	resolver := NewHarnessResolver(reader)

	got, err := resolver.Resolve(context.Background(), "task", "E34-F01-001", HarnessIdentity{})

	require.NoError(t, err)
	assert.True(t, got.IsZero(), "an expired claim must not supply harness identity; got %+v", got)
}

// TestHarnessResolver_Resolve_ExpiredClaim_FallsThroughToEnv pins the
// "degrades to the next precedence tier" half of the fix: an expired claim
// must not merely blank out to zero, it must fall through to env exactly as
// if there were no claim row at all (same per-field precedence chain as
// TestHarnessResolver_Resolve_NoClaimNoEnv_ZeroIdentity's claim:nil case).
func TestHarnessResolver_Resolve_ExpiredClaim_FallsThroughToEnv(t *testing.T) {
	t.Setenv("SHARK_HARNESS", "claude")
	t.Setenv("SHARK_HARNESS_VERSION", "")
	t.Setenv("SHARK_HARNESS_MODEL", "")

	reader := &fakeClaimReader{claim: &models.EntityClaim{
		Harness:       "codex",
		LastHeartbeat: time.Now().UTC().Add(-1 * time.Hour),
	}}
	resolver := NewHarnessResolver(reader)

	got, err := resolver.Resolve(context.Background(), "task", "E34-F01-001", HarnessIdentity{})

	require.NoError(t, err)
	assert.Equal(t, "claude", got.Type, "expired claim must degrade to env, not leak its own value")
}

// TestHarnessResolver_SetTTL_ZeroDisablesExpiry pins the TTL-authority-
// alignment half of the rework fix (advisor-flagged gap): GetHarnessResolver
// mirrors GetClaimService by calling SetTTL from .sharkconfig.json's
// claim_ttl_seconds, and claim_ttl_seconds: 0 means "never expires"
// (docs/guides/route-based-workflow.md §4, models.EntityClaim.IsExpired's
// ttl<=0 branch). Without SetTTL, a resolver stuck on the 15m default would
// contradict a ClaimService explicitly configured never to expire a lease.
func TestHarnessResolver_SetTTL_ZeroDisablesExpiry(t *testing.T) {
	t.Setenv("SHARK_HARNESS", "")
	t.Setenv("SHARK_HARNESS_VERSION", "")
	t.Setenv("SHARK_HARNESS_MODEL", "")

	reader := &fakeClaimReader{claim: &models.EntityClaim{
		Harness:       "codex",
		LastHeartbeat: time.Now().UTC().Add(-24 * time.Hour), // ancient by any positive TTL
	}}
	resolver := NewHarnessResolver(reader)
	resolver.SetTTL(0)

	got, err := resolver.Resolve(context.Background(), "task", "E34-F01-001", HarnessIdentity{})

	require.NoError(t, err)
	assert.Equal(t, "codex", got.Type, "TTL 0 must disable expiry, matching ClaimService/IsExpired's ttl<=0 contract")
}

func TestHarnessResolver_Resolve_NoClaimNoEnv_ZeroIdentity(t *testing.T) {
	t.Setenv("SHARK_HARNESS", "")
	t.Setenv("SHARK_HARNESS_VERSION", "")
	t.Setenv("SHARK_HARNESS_MODEL", "")

	reader := &fakeClaimReader{claim: nil} // unclaimed: (nil, nil) per claim.Repository.Get
	resolver := NewHarnessResolver(reader)

	got, err := resolver.Resolve(context.Background(), "task", "E34-F01-001", HarnessIdentity{})

	require.NoError(t, err)
	assert.True(t, got.IsZero())
}

// TC-018 (resolver-level half; the exit-0/warning-log half through runNext
// belongs to T-E34-F01-004): a claim-read error degrades to the zero
// identity and never propagates as an error from Resolve, per D-F01-05.
func TestHarnessResolver_Resolve_ClaimReadError_DegradesToZero(t *testing.T) {
	reader := &fakeClaimReader{err: errors.New("boom")}
	resolver := NewHarnessResolver(reader)

	got, err := resolver.Resolve(context.Background(), "task", "E34-F01-001", HarnessIdentity{})

	require.NoError(t, err, "a claim-read error must never fail Resolve (REQ-NF-002/D-F01-05)")
	assert.True(t, got.IsZero())
}

func TestHarnessResolver_Resolve_NilClaimReader_ZeroIdentity(t *testing.T) {
	t.Setenv("SHARK_HARNESS", "")
	t.Setenv("SHARK_HARNESS_VERSION", "")
	t.Setenv("SHARK_HARNESS_MODEL", "")

	resolver := NewHarnessResolver(nil)

	got, err := resolver.Resolve(context.Background(), "task", "E34-F01-001", HarnessIdentity{})

	require.NoError(t, err)
	assert.True(t, got.IsZero())
}

// TestHarnessIdentity_Normalized pins the exact normalization rule (review
// finding F1): Type is trimmed and lowercased (a small bounded vocabulary
// compared case-insensitively by isClaude/isCodex/isHarness), while
// Version/Model are trimmed only — their case is opaque free text.
func TestHarnessIdentity_Normalized(t *testing.T) {
	got := HarnessIdentity{Type: " Claude ", Version: " 1.2.3 ", Model: " Opus "}.Normalized()

	assert.Equal(t, "claude", got.Type)
	assert.Equal(t, "1.2.3", got.Version)
	assert.Equal(t, "Opus", got.Model, "Model case must be preserved, only trimmed")
}

// TestHarnessResolver_Resolve_OverrideNormalizesWhitespaceAndCase is the
// regression test for review finding F1: an untrimmed/mixed-case --harness
// override used to reach the rendered identity verbatim, silently defeating
// the isClaude/isCodex EqualFold check downstream (EqualFold is
// case-insensitive but not whitespace-tolerant). Resolve must normalize the
// override before applying precedence.
func TestHarnessResolver_Resolve_OverrideNormalizesWhitespaceAndCase(t *testing.T) {
	t.Setenv("SHARK_HARNESS", "")
	t.Setenv("SHARK_HARNESS_VERSION", "")
	t.Setenv("SHARK_HARNESS_MODEL", "")

	resolver := NewHarnessResolver(nil)

	got, err := resolver.Resolve(context.Background(), "task", "E34-F01-001", HarnessIdentity{Type: " Claude "})

	require.NoError(t, err)
	assert.Equal(t, "claude", got.Type, "override must be trimmed+lowercased before precedence is applied")
}

// TestHarnessResolver_Resolve_EnvNormalizesWhitespaceAndCase is F1's env-tier
// counterpart: SHARK_HARNESS is operator-set shell state and just as
// susceptible to stray whitespace/case as a CLI flag.
func TestHarnessResolver_Resolve_EnvNormalizesWhitespaceAndCase(t *testing.T) {
	t.Setenv("SHARK_HARNESS", " Codex ")
	t.Setenv("SHARK_HARNESS_VERSION", "")
	t.Setenv("SHARK_HARNESS_MODEL", "")

	resolver := NewHarnessResolver(nil)

	got, err := resolver.Resolve(context.Background(), "task", "E34-F01-001", HarnessIdentity{})

	require.NoError(t, err)
	assert.Equal(t, "codex", got.Type, "env value must be trimmed+lowercased before precedence is applied")
}

// TestMergeResolvedHarness_NilResolver_MergesZeroIdentity pins the "no
// resolver" branch shared by next.go's resolveEntity and controller.go's Run
// (review finding F2): vars must still carry all three harness keys, present
// but empty, exactly like the zero-identity fallback each call site used to
// duplicate inline.
func TestMergeResolvedHarness_NilResolver_MergesZeroIdentity(t *testing.T) {
	vars := map[string]string{"existing": "kept"}

	got, err := MergeResolvedHarness(context.Background(), nil, "task", "E34-F01-001", HarnessIdentity{}, vars)

	require.NoError(t, err)
	assert.True(t, got.IsZero())
	assert.Equal(t, "kept", vars["existing"], "unrelated vars must be untouched")
	for _, key := range []string{"harness", "harness_version", "harness_model"} {
		v, ok := vars[key]
		assert.Truef(t, ok, "key %q must be present even with a nil resolver", key)
		assert.Equal(t, "", v)
	}
}

// TestMergeResolvedHarness_ResolverSet_MergesResolvedIdentity pins the
// resolver-present branch: the resolved identity's Vars() are merged into
// vars and also returned, so a caller like next.go can mirror it onto a
// response object without resolving twice.
func TestMergeResolvedHarness_ResolverSet_MergesResolvedIdentity(t *testing.T) {
	t.Setenv("SHARK_HARNESS", "")
	t.Setenv("SHARK_HARNESS_VERSION", "")
	t.Setenv("SHARK_HARNESS_MODEL", "")

	resolver := NewHarnessResolver(nil)
	vars := map[string]string{}

	got, err := MergeResolvedHarness(context.Background(), resolver, "task", "E34-F01-001", HarnessIdentity{Type: "claude"}, vars)

	require.NoError(t, err)
	assert.Equal(t, "claude", got.Type)
	assert.Equal(t, "claude", vars["harness"])
}
