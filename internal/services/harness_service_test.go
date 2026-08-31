package services

import (
	"context"
	"errors"
	"testing"

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

	reader := &fakeClaimReader{claim: &models.EntityClaim{Harness: "codex"}}
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

	reader := &fakeClaimReader{claim: &models.EntityClaim{Harness: "codex"}}
	resolver := NewHarnessResolver(reader)

	got, err := resolver.Resolve(context.Background(), "task", "E34-F01-001", HarnessIdentity{})

	require.NoError(t, err)
	assert.Equal(t, "codex", got.Type)
	assert.Equal(t, "9.9", got.Version)
	assert.Equal(t, "", got.Model)
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
