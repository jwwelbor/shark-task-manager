// Package commands tests for T-E34-F01-002: harness fields wired through
// `shark claim`'s CLI flags into the real ClaimService (mocked only at the
// ClaimRepository seam — mocking ClaimService itself is forbidden by
// test-plan.md's Caller-Path Contract for TC-001/TC-002/TC-012/TC-013).
package commands

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	claimrepo "github.com/jwwelbor/shark-task-manager/internal/repository/claim"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// fakeClaimRepo is a stateful, in-memory implementation of
// services.ClaimRepository. Unlike a pure function-field stub, it actually
// stores claims so a `shark claim` followed by a `shark claims --json` in the
// same test observes the round-tripped values — matching TC-001/TC-012's
// "shark claims --json reports ..." expected output.
type fakeClaimRepo struct {
	mu         sync.Mutex
	claims     map[string]*models.EntityClaim
	claimCalls int
	lastClaim  *models.EntityClaim
}

func newFakeClaimRepo() *fakeClaimRepo {
	return &fakeClaimRepo{claims: map[string]*models.EntityClaim{}}
}

func fakeClaimKey(entityType, entityKey string) string { return entityType + "/" + entityKey }

func (f *fakeClaimRepo) Claim(_ context.Context, c *models.EntityClaim) (*models.EntityClaim, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.claimCalls++
	cp := *c
	f.lastClaim = &cp
	key := fakeClaimKey(c.EntityType, c.EntityKey)
	if _, exists := f.claims[key]; exists {
		return nil, claimrepo.ErrAlreadyClaimed
	}
	cp.ID = int64(len(f.claims) + 1)
	cp.ClaimedAt = time.Now().UTC()
	cp.LastHeartbeat = cp.ClaimedAt
	f.claims[key] = &cp
	return &cp, nil
}

func (f *fakeClaimRepo) Get(_ context.Context, entityType, entityKey string) (*models.EntityClaim, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.claims[fakeClaimKey(entityType, entityKey)], nil
}

func (f *fakeClaimRepo) Release(_ context.Context, entityType, entityKey string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := fakeClaimKey(entityType, entityKey)
	if _, ok := f.claims[key]; !ok {
		return false, nil
	}
	delete(f.claims, key)
	return true, nil
}

func (f *fakeClaimRepo) ReleaseSession(_ context.Context, entityType, entityKey, sessionID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := fakeClaimKey(entityType, entityKey)
	c, ok := f.claims[key]
	if !ok || c.SessionID != sessionID {
		return false, nil
	}
	delete(f.claims, key)
	return true, nil
}

func (f *fakeClaimRepo) Renew(_ context.Context, entityType, entityKey, sessionID string, progress *float64, note string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.claims[fakeClaimKey(entityType, entityKey)]
	if !ok || c.SessionID != sessionID {
		return false, nil
	}
	c.LastHeartbeat = time.Now().UTC()
	if progress != nil {
		c.Progress = progress
	}
	if note != "" {
		c.Note = note
	}
	return true, nil
}

func (f *fakeClaimRepo) ReclaimExpired(_ context.Context, _ time.Duration) (int64, error) {
	return 0, nil
}

func (f *fakeClaimRepo) List(_ context.Context) ([]*models.EntityClaim, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*models.EntityClaim, 0, len(f.claims))
	for _, c := range f.claims {
		out = append(out, c)
	}
	return out, nil
}

// Compile-time check: fakeClaimRepo satisfies the real repository interface.
var _ services.ClaimRepository = (*fakeClaimRepo)(nil)

// withClaimSvcOverride installs a *services.ClaimService (the real, concrete
// service type — not a mock of it) built from a mocked ClaimRepository, and
// restores the previous override on cleanup. This is the only mock seam the
// Caller-Path Contract permits for TC-001/002/012/013.
func withClaimSvcOverride(t *testing.T, repo services.ClaimRepository) {
	t.Helper()
	orig := claimSvcOverride
	claimSvcOverride = services.NewClaimService(repo, durationPtrForTest(time.Hour))
	t.Cleanup(func() { claimSvcOverride = orig })
}

func durationPtrForTest(d time.Duration) *time.Duration { return &d }

// buildClaimCmdForTest returns a fresh, isolated `shark claim` command with
// the same flags as the production claimCmd (internal/cli/commands/claim.go)
// bound to the real runClaim entrypoint — mirroring the isolated-command
// pattern used by TestBugCreate_* in bug_test.go so package-level claimCmd
// (wired into cli.RootCmd) is never mutated by tests.
func buildClaimCmdForTest() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "claim <key>",
		Args: cobra.ExactArgs(1),
		RunE: runClaim,
	}
	cmd.Flags().String("by", "", "")
	cmd.Flags().String("session", "", "")
	cmd.Flags().Bool("force", false, "")
	cmd.Flags().String("harness", "", "")
	cmd.Flags().String("harness-version", "", "")
	cmd.Flags().String("harness-model", "", "")
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	return cmd
}

func buildClaimsCmdForTest() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "claims",
		Args: cobra.NoArgs,
		RunE: runClaims,
	}
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	return cmd
}

func withJSONOutput(t *testing.T) {
	t.Helper()
	orig := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	t.Cleanup(func() { cli.GlobalConfig.JSON = orig })
}

// TestClaimCmd_RegistersHarnessFlags guards against the isolated test
// commands above masking a missing production registration: they declare
// their own --harness* flags, so they would stay green even if the
// claimCmd.Flags().String(...) calls in claim.go's init() were deleted.
// AC-01's Given/When is a literal `shark claim <key> --harness=...`
// invocation, so the production command object must expose the flags too.
func TestClaimCmd_RegistersHarnessFlags(t *testing.T) {
	for _, name := range []string{"harness", "harness-version", "harness-model"} {
		if claimCmd.Flags().Lookup(name) == nil {
			t.Errorf("claimCmd must register --%s", name)
		}
	}
}

// TestRunClaim_TC001_PersistsAllThreeHarnessFields drives runClaim through
// Cobra with all three --harness* flags set, then asserts the persisted
// claim (via shark claims --json) reports the three values. TC-001,
// spec.md AC-01/AC-09.
func TestRunClaim_TC001_PersistsAllThreeHarnessFields(t *testing.T) {
	repo := newFakeClaimRepo()
	withClaimSvcOverride(t, repo)
	withJSONOutput(t)

	claimCmd := buildClaimCmdForTest()
	claimCmd.SetArgs([]string{
		"E34-F01-001", "--by=agent1",
		"--harness=claude", "--harness-version=2.1.0", "--harness-model=opus",
	})
	if err := claimCmd.Execute(); err != nil {
		t.Fatalf("claim Execute() error = %v", err)
	}

	if repo.claimCalls != 1 {
		t.Fatalf("expected exactly one repository Claim call, got %d", repo.claimCalls)
	}
	if repo.lastClaim.Harness != "claude" || repo.lastClaim.HarnessVersion != "2.1.0" || repo.lastClaim.HarnessModel != "opus" {
		t.Fatalf("captured ClaimInput harness fields = %+v, want claude/2.1.0/opus", repo.lastClaim)
	}

	out := captureStdout(t, func() {
		claimsCmd := buildClaimsCmdForTest()
		if err := claimsCmd.Execute(); err != nil {
			t.Fatalf("claims Execute() error = %v", err)
		}
	})
	var claims []models.EntityClaim
	if err := json.Unmarshal([]byte(out), &claims); err != nil {
		t.Fatalf("unmarshal claims JSON: %v (output: %s)", err, out)
	}
	if len(claims) != 1 {
		t.Fatalf("expected 1 claim, got %d", len(claims))
	}
	if claims[0].Harness != "claude" || claims[0].HarnessVersion != "2.1.0" || claims[0].HarnessModel != "opus" {
		t.Errorf("shark claims --json harness fields = %+v, want claude/2.1.0/opus", claims[0])
	}
}

// TestRunClaim_TC001_EdgeCase_OnlyHarnessFlagSet covers TC-001's edge case:
// only --harness supplied leaves version/model as empty strings, not a
// sentinel.
func TestRunClaim_TC001_EdgeCase_OnlyHarnessFlagSet(t *testing.T) {
	repo := newFakeClaimRepo()
	withClaimSvcOverride(t, repo)

	claimCmd := buildClaimCmdForTest()
	claimCmd.SetArgs([]string{"E34-F01-002", "--by=agent1", "--harness=claude"})
	if err := claimCmd.Execute(); err != nil {
		t.Fatalf("claim Execute() error = %v", err)
	}

	if repo.lastClaim.Harness != "claude" {
		t.Errorf("Harness = %q, want claude", repo.lastClaim.Harness)
	}
	if repo.lastClaim.HarnessVersion != "" || repo.lastClaim.HarnessModel != "" {
		t.Errorf("unset harness fields must be empty, got version=%q model=%q",
			repo.lastClaim.HarnessVersion, repo.lastClaim.HarnessModel)
	}
}

// TestRunClaim_TC002_NoHarnessFlags_EmptyNotSentinel is TC-002: a claim with
// no --harness* flags succeeds and the captured ClaimInput carries empty
// strings, never a sentinel like "unknown". spec.md AC-01 negative partition.
func TestRunClaim_TC002_NoHarnessFlags_EmptyNotSentinel(t *testing.T) {
	repo := newFakeClaimRepo()
	withClaimSvcOverride(t, repo)
	withJSONOutput(t)

	claimCmd := buildClaimCmdForTest()
	claimCmd.SetArgs([]string{"E34-F01-003", "--by=agent1"})
	if err := claimCmd.Execute(); err != nil {
		t.Fatalf("claim Execute() error = %v", err)
	}

	if repo.lastClaim.Harness != "" || repo.lastClaim.HarnessVersion != "" || repo.lastClaim.HarnessModel != "" {
		t.Fatalf("expected all-empty harness fields, got %+v", repo.lastClaim)
	}

	out := captureStdout(t, func() {
		claimsCmd := buildClaimsCmdForTest()
		if err := claimsCmd.Execute(); err != nil {
			t.Fatalf("claims Execute() error = %v", err)
		}
	})
	if strings.Contains(strings.ToLower(out), "unknown") || strings.Contains(strings.ToLower(out), "\"none\"") || strings.Contains(strings.ToLower(out), "unset") {
		t.Errorf("output must not contain a sentinel value for unset harness fields, got: %s", out)
	}
	var claims []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &claims); err != nil {
		t.Fatalf("unmarshal claims JSON: %v (output: %s)", err, out)
	}
	if len(claims) != 1 {
		t.Fatalf("expected 1 claim, got %d", len(claims))
	}
	// omitempty means the keys are simply absent for unset harness fields.
	for _, key := range []string{"harness", "harness_version", "harness_model"} {
		if _, present := claims[0][key]; present {
			t.Errorf("expected %q to be omitted from JSON when unset, got %v", key, claims[0][key])
		}
	}
}

// TestRunClaim_TC012_HarnessTrimmedAndLowercased is TC-012 (spec.md AC-09,
// REQ-F-001): --harness is trimmed and lowercased; --harness-version and
// --harness-model are trimmed only, never lowercased.
func TestRunClaim_TC012_HarnessTrimmedAndLowercased(t *testing.T) {
	repo := newFakeClaimRepo()
	withClaimSvcOverride(t, repo)

	claimCmd := buildClaimCmdForTest()
	claimCmd.SetArgs([]string{
		"E34-F01-004", "--by=agent1",
		"--harness=  CLAUDE  ", "--harness-model=Opus-4",
	})
	if err := claimCmd.Execute(); err != nil {
		t.Fatalf("claim Execute() error = %v", err)
	}

	if repo.lastClaim.Harness != "claude" {
		t.Errorf("Harness = %q, want %q (trimmed and lowercased)", repo.lastClaim.Harness, "claude")
	}
	if repo.lastClaim.HarnessModel != "Opus-4" {
		t.Errorf("HarnessModel = %q, want %q (trimmed only, not lowercased)", repo.lastClaim.HarnessModel, "Opus-4")
	}
}

// TestRunClaim_TC012_EdgeCase_MixedCaseNoWhitespace covers TC-012's edge
// case: mixed case with no surrounding whitespace still lowercases.
func TestRunClaim_TC012_EdgeCase_MixedCaseNoWhitespace(t *testing.T) {
	repo := newFakeClaimRepo()
	withClaimSvcOverride(t, repo)

	claimCmd := buildClaimCmdForTest()
	claimCmd.SetArgs([]string{"E34-F01-005", "--by=agent1", "--harness=Claude"})
	if err := claimCmd.Execute(); err != nil {
		t.Fatalf("claim Execute() error = %v", err)
	}
	if repo.lastClaim.Harness != "claude" {
		t.Errorf("Harness = %q, want claude", repo.lastClaim.Harness)
	}
}

// TestRunClaim_TC013_OversizedHarnessRejected is TC-013 (spec.md AC-10,
// REQ-NF-004): a >100 char --harness value fails the command with a
// non-zero exit, an error naming the field and quoting the input, and the
// repository's Claim method must never be invoked (no partial claim row).
func TestRunClaim_TC013_OversizedHarnessRejected(t *testing.T) {
	repo := newFakeClaimRepo()
	withClaimSvcOverride(t, repo)

	oversized := strings.Repeat("x", 101)
	claimCmd := buildClaimCmdForTest()
	claimCmd.SetArgs([]string{"E34-F01-006", "--by=agent1", "--harness=" + oversized})

	err := claimCmd.Execute()
	if err == nil {
		t.Fatal("expected a non-zero-exit error for an oversized --harness value")
	}
	if !strings.Contains(err.Error(), "harness") {
		t.Errorf("error must name the field %q, got: %v", "harness", err)
	}
	if !strings.Contains(err.Error(), oversized) {
		t.Errorf("error must quote the offending input, got: %v", err)
	}
	if repo.claimCalls != 0 {
		t.Fatalf("repository Claim must never be invoked on validation failure, got %d calls", repo.claimCalls)
	}
}

// TestRunClaim_TC013_EdgeCase_Exactly100CharsAccepted covers TC-013's
// boundary-inclusive edge case: exactly 100 characters is valid.
func TestRunClaim_TC013_EdgeCase_Exactly100CharsAccepted(t *testing.T) {
	repo := newFakeClaimRepo()
	withClaimSvcOverride(t, repo)

	exactly100 := strings.Repeat("x", 100)
	claimCmd := buildClaimCmdForTest()
	claimCmd.SetArgs([]string{"E34-F01-007", "--by=agent1", "--harness=" + exactly100})
	if err := claimCmd.Execute(); err != nil {
		t.Fatalf("claim Execute() error = %v (expected 100-char harness to be accepted)", err)
	}
	if repo.claimCalls != 1 {
		t.Fatalf("expected repository Claim to be invoked once, got %d", repo.claimCalls)
	}
}

// TestRunClaim_TC013_EdgeCase_VersionAndModelIndependentlyCapped covers
// TC-013's edge case that --harness-version and --harness-model are each
// independently length-capped, and that a validation failure on one field
// leaves no partial claim (harness itself must not have been persisted).
func TestRunClaim_TC013_EdgeCase_VersionAndModelIndependentlyCapped(t *testing.T) {
	oversized := strings.Repeat("y", 101)

	t.Run("harness-version", func(t *testing.T) {
		repo := newFakeClaimRepo()
		withClaimSvcOverride(t, repo)
		claimCmd := buildClaimCmdForTest()
		claimCmd.SetArgs([]string{"E34-F01-008", "--by=agent1", "--harness=claude", "--harness-version=" + oversized})
		err := claimCmd.Execute()
		if err == nil {
			t.Fatal("expected error for oversized --harness-version")
		}
		if !strings.Contains(err.Error(), "harness_version") {
			t.Errorf("error must name harness_version, got: %v", err)
		}
		if repo.claimCalls != 0 {
			t.Fatalf("no partial claim row: repository Claim must not be called, got %d calls", repo.claimCalls)
		}
	})

	t.Run("harness-model", func(t *testing.T) {
		repo := newFakeClaimRepo()
		withClaimSvcOverride(t, repo)
		claimCmd := buildClaimCmdForTest()
		claimCmd.SetArgs([]string{"E34-F01-009", "--by=agent1", "--harness=claude", "--harness-model=" + oversized})
		err := claimCmd.Execute()
		if err == nil {
			t.Fatal("expected error for oversized --harness-model")
		}
		if !strings.Contains(err.Error(), "harness_model") {
			t.Errorf("error must name harness_model, got: %v", err)
		}
		if repo.claimCalls != 0 {
			t.Fatalf("no partial claim row: repository Claim must not be called, got %d calls", repo.claimCalls)
		}
	})
}
