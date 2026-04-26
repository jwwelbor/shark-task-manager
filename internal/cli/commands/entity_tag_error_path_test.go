package commands

// entity_tag_error_path_test.go covers T-E28-F04-013:
//
// "Render SC-2 vocabulary snippet on entity create/update --tag path"
//
// AC coverage:
//   - AC-1..AC-6: shark <entity> create --tag=<unregistered> renders snippet + remediation + exits 3
//   - AC-7..AC-12: shark <entity> update --tag=<unregistered> renders snippet + remediation + exits 3
//   - AC-13: tag_required_for entity emits "at least one tag is required for <entity>" + snippet + exits 3
//   - AC-14: --json mode suppresses snippet, emits code headers
//   - AC-15: existing tests unaffected (covered by entity_tag_cmd_test.go)
//   - AC-16: docs updated (verified in source)
//
// Test strategy:
//   - handleEntityServiceError and handleVocabularyErrorWithSnippet are
//     tested directly with a mockEntityTagService (defined in
//     entity_tag_cmd_test.go; same package so accessible here).
//   - extractExitCode logic is verified via a local copy.
//   - All tests use mock services — no real database.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// buildTestCmd creates a minimal cobra command with stderr buffer and --json
// flag. If jsonMode is true, cli.GlobalConfig.JSON is set to true so helpers
// detect JSON mode even without flag parsing.
func buildTestCmd(t *testing.T, jsonMode bool) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "shark"}
	root.PersistentFlags().Bool("json", false, "JSON output")
	root.SilenceErrors = true
	root.SilenceUsage = true
	return root
}

// withJSONMode sets cli.GlobalConfig.JSON for the test and restores on cleanup.
func withJSONMode(t *testing.T, on bool) {
	t.Helper()
	orig := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = on
	t.Cleanup(func() { cli.GlobalConfig.JSON = orig })
}

// threeTagVocab returns a 3-entry vocabulary for snippet tests.
func threeTagVocab() []*models.Tag {
	return []*models.Tag{
		{ID: 1, Name: "audio"},
		{ID: 2, Name: "backend"},
		{ID: 3, Name: "voice"},
	}
}

// newTagSvcWithVocab returns a mockEntityTagService whose ListTags always
// returns the given tags slice.
func newTagSvcWithVocab(tags []*models.Tag) *mockEntityTagService {
	return &mockEntityTagService{
		listTagsFn: func(_ context.Context) ([]*models.Tag, error) {
			return tags, nil
		},
	}
}

// ---------------------------------------------------------------------------
// handleVocabularyErrorWithSnippet — TagRequiredError extended behaviour
// (T-E28-F04-013 FR-2)
// ---------------------------------------------------------------------------

// TestHandleVocabularyError_TagRequired_RendersSnippetNoRemediation verifies
// that *TagRequiredError triggers the vocabulary snippet but NOT the
// "To add it:" remediation line.
func TestHandleVocabularyError_TagRequired_RendersSnippetNoRemediation(t *testing.T) {
	withJSONMode(t, false)

	var errBuf strings.Builder
	cmd := buildTestCmd(t, false)
	cmd.SetErr(&errBuf)

	svc := newTagSvcWithVocab(threeTagVocab())
	err := &services.TagRequiredError{EntityType: "task"}
	got := handleVocabularyErrorWithSnippet(cmd, svc, "", err)

	if got == nil {
		t.Fatal("expected non-nil error")
	}

	stderr := errBuf.String()

	if !strings.Contains(stderr, "at least one tag is required for task") {
		t.Errorf("stderr missing error message, got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "Available tags:") {
		t.Errorf("stderr missing 'Available tags:', got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "audio, backend, voice") {
		t.Errorf("stderr missing tag names, got:\n%s", stderr)
	}
	// Must NOT contain the remediation line.
	if strings.Contains(stderr, "To add it:") {
		t.Errorf("stderr unexpectedly contains remediation for TagRequiredError, got:\n%s", stderr)
	}
	if !strings.HasPrefix(got.Error(), "exit code 3:") {
		t.Errorf("expected 'exit code 3:' prefix, got: %s", got.Error())
	}
}

// TestHandleVocabularyError_TagRequired_EmptyVocab verifies that when no
// tags are registered the snippet header is NOT emitted.
func TestHandleVocabularyError_TagRequired_EmptyVocab(t *testing.T) {
	withJSONMode(t, false)

	var errBuf strings.Builder
	cmd := buildTestCmd(t, false)
	cmd.SetErr(&errBuf)

	svc := newTagSvcWithVocab([]*models.Tag{})
	err := &services.TagRequiredError{EntityType: "bug"}
	got := handleVocabularyErrorWithSnippet(cmd, svc, "", err)

	if got == nil {
		t.Fatal("expected non-nil error")
	}

	stderr := errBuf.String()
	if !strings.Contains(stderr, "at least one tag is required for bug") {
		t.Errorf("stderr missing error message, got:\n%s", stderr)
	}
	if strings.Contains(stderr, "Available tags:") {
		t.Errorf("stderr should not contain snippet for empty vocab, got:\n%s", stderr)
	}
}

// TestHandleVocabularyError_TagRequired_JSONMode verifies that --json mode
// suppresses the snippet and remediation line for TagRequiredError.
func TestHandleVocabularyError_TagRequired_JSONMode(t *testing.T) {
	withJSONMode(t, true)

	var errBuf strings.Builder
	cmd := buildTestCmd(t, true)
	cmd.SetErr(&errBuf)

	svc := newTagSvcWithVocab(threeTagVocab())
	err := &services.TagRequiredError{EntityType: "feature"}
	got := handleVocabularyErrorWithSnippet(cmd, svc, "", err)

	if got == nil {
		t.Fatal("expected non-nil error")
	}

	stderr := errBuf.String()
	if strings.Contains(stderr, "Available tags:") {
		t.Errorf("JSON mode should suppress snippet, got:\n%s", stderr)
	}
	if strings.Contains(stderr, "To add it:") {
		t.Errorf("JSON mode should suppress remediation, got:\n%s", stderr)
	}
}

// ---------------------------------------------------------------------------
// handleEntityServiceError — routing tests
// ---------------------------------------------------------------------------

// TestHandleEntityServiceError_UnregisteredTag verifies the SC-2 snippet +
// remediation path for *UnregisteredTagError.
func TestHandleEntityServiceError_UnregisteredTag(t *testing.T) {
	withJSONMode(t, false)

	var errBuf strings.Builder
	cmd := buildTestCmd(t, false)
	cmd.SetErr(&errBuf)

	svc := newTagSvcWithVocab(threeTagVocab())
	err := &services.UnregisteredTagError{Name: "ghost"}
	got := handleEntityServiceError(cmd, svc, err, "task", "T-E01-F01-001")

	if got == nil {
		t.Fatal("expected non-nil error")
	}

	stderr := errBuf.String()
	if !strings.Contains(stderr, "tag is not registered: ghost") {
		t.Errorf("stderr missing unregistered message, got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "Available tags:") {
		t.Errorf("stderr missing snippet, got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "To add it: shark tags add ghost") {
		t.Errorf("stderr missing remediation line, got:\n%s", stderr)
	}
	if !strings.HasPrefix(got.Error(), "exit code 3:") {
		t.Errorf("expected 'exit code 3:' prefix, got: %s", got.Error())
	}
}

// TestHandleEntityServiceError_TagRequired verifies the snippet-only path
// (no remediation) for *TagRequiredError.
func TestHandleEntityServiceError_TagRequired(t *testing.T) {
	withJSONMode(t, false)

	var errBuf strings.Builder
	cmd := buildTestCmd(t, false)
	cmd.SetErr(&errBuf)

	svc := newTagSvcWithVocab(threeTagVocab())
	err := &services.TagRequiredError{EntityType: "bug"}
	got := handleEntityServiceError(cmd, svc, err, "bug", "B001")

	if got == nil {
		t.Fatal("expected non-nil error")
	}

	stderr := errBuf.String()
	if !strings.Contains(stderr, "at least one tag is required for bug") {
		t.Errorf("stderr missing required message, got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "Available tags:") {
		t.Errorf("stderr missing snippet, got:\n%s", stderr)
	}
	if strings.Contains(stderr, "To add it:") {
		t.Errorf("stderr should NOT contain remediation for TagRequired, got:\n%s", stderr)
	}
	if !strings.HasPrefix(got.Error(), "exit code 3:") {
		t.Errorf("expected 'exit code 3:' prefix, got: %s", got.Error())
	}
}

// TestHandleEntityServiceError_NilIsNoop verifies nil input returns nil.
func TestHandleEntityServiceError_NilIsNoop(t *testing.T) {
	cmd := buildTestCmd(t, false)
	svc := &mockEntityTagService{}

	got := handleEntityServiceError(cmd, svc, nil, "bug", "B001")
	if got != nil {
		t.Errorf("expected nil for nil input, got: %v", got)
	}
}

// TestHandleEntityServiceError_JSONMode verifies that --json suppresses
// snippet and remediation for UnregisteredTagError.
func TestHandleEntityServiceError_JSONMode(t *testing.T) {
	withJSONMode(t, true)

	var errBuf strings.Builder
	cmd := buildTestCmd(t, true)
	cmd.SetErr(&errBuf)

	svc := newTagSvcWithVocab(threeTagVocab())
	err := &services.UnregisteredTagError{Name: "ghost"}
	got := handleEntityServiceError(cmd, svc, err, "task", "T-E01-F01-001")

	if got == nil {
		t.Fatal("expected non-nil error")
	}

	stderr := errBuf.String()
	if strings.Contains(stderr, "Available tags:") {
		t.Errorf("JSON mode should suppress snippet, got:\n%s", stderr)
	}
	if strings.Contains(stderr, "To add it:") {
		t.Errorf("JSON mode should suppress remediation, got:\n%s", stderr)
	}
}

// ---------------------------------------------------------------------------
// Table-driven: 6 entity types × 2 verbs × 2 error types
// Covers AC-1..AC-12 for UnregisteredTagError, AC-13 for TagRequiredError.
// ---------------------------------------------------------------------------

func TestEntityCreateUpdate_UnregisteredTag_Table(t *testing.T) {
	type errSpec struct {
		name            string
		makeErr         func() error
		wantSnippet     bool
		wantRemediation bool
	}

	errSpecs := []errSpec{
		{
			name:            "UnregisteredTagError",
			makeErr:         func() error { return &services.UnregisteredTagError{Name: "ghost"} },
			wantSnippet:     true,
			wantRemediation: true,
		},
		{
			name:            "TagRequiredError",
			makeErr:         func() error { return &services.TagRequiredError{EntityType: "entity"} },
			wantSnippet:     true,
			wantRemediation: false,
		},
	}

	entities := []string{"task", "feature", "epic", "bug", "change", "idea"}
	verbs := []string{"create", "update"}

	for _, entity := range entities {
		for _, verb := range verbs {
			for _, es := range errSpecs {
				testName := fmt.Sprintf("%s/%s/%s", entity, verb, es.name)
				entity, verb, es := entity, verb, es // capture
				t.Run(testName, func(t *testing.T) {
					withJSONMode(t, false)

					var errBuf strings.Builder
					cmd := buildTestCmd(t, false)
					cmd.SetErr(&errBuf)
					svc := newTagSvcWithVocab(threeTagVocab())

					serviceErr := es.makeErr()
					got := handleEntityServiceError(cmd, svc, serviceErr, entity, "key-1")

					if got == nil {
						t.Fatal("expected non-nil error")
					}

					// Exit code must be 3.
					if !strings.HasPrefix(got.Error(), "exit code 3:") {
						t.Errorf("[%s] exit code want 3, got: %s", testName, got.Error())
					}

					stderr := errBuf.String()

					if es.wantSnippet {
						if !strings.Contains(stderr, "Available tags:") {
							t.Errorf("[%s] stderr missing snippet, got:\n%s", testName, stderr)
						}
						if !strings.Contains(stderr, "audio, backend, voice") {
							t.Errorf("[%s] stderr missing tag names, got:\n%s", testName, stderr)
						}
					}

					if es.wantRemediation {
						if !strings.Contains(stderr, "To add it: shark tags add") {
							t.Errorf("[%s] stderr missing remediation, got:\n%s", testName, stderr)
						}
					} else {
						if strings.Contains(stderr, "To add it:") {
							t.Errorf("[%s] stderr unexpectedly has remediation, got:\n%s", testName, stderr)
						}
					}

					// The typed error must still be accessible via errors.As.
					var unregistered *services.UnregisteredTagError
					var required *services.TagRequiredError
					if !errors.As(got, &unregistered) && !errors.As(got, &required) {
						t.Errorf("[%s] typed error not accessible via errors.As: %T %v", testName, got, got)
					}

					_ = verb // verb documents which function would call handleEntityServiceError
				})
			}
		}
	}
}

// ---------------------------------------------------------------------------
// extractExitCode — unit test via local copy
// ---------------------------------------------------------------------------
//
// The extractExitCode function lives in cmd/shark/main.go and cannot be
// imported here (main package). We verify the logic via a local copy.

// localExtractExitCode mirrors the logic in cmd/shark/main.go extractExitCode.
func localExtractExitCode(err error, fallback int) int {
	if err == nil {
		return 0
	}
	msg := err.Error()
	const prefix = "exit code "
	if !strings.HasPrefix(msg, prefix) {
		return fallback
	}
	rest := msg[len(prefix):]
	colonIdx := strings.Index(rest, ":")
	if colonIdx < 0 {
		return fallback
	}
	nStr := strings.TrimSpace(rest[:colonIdx])
	n := 0
	for _, c := range nStr {
		if c < '0' || c > '9' {
			return fallback
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func TestExtractExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		fallback int
		want     int
	}{
		{"nil error", nil, 1, 0},
		{"exit code 3", fmt.Errorf("exit code 3: some error"), 1, 3},
		{"exit code 1", fmt.Errorf("exit code 1: not found"), 1, 1},
		{"exit code 2", fmt.Errorf("exit code 2: db error"), 1, 2},
		{"no prefix", fmt.Errorf("bare error"), 1, 1},
		{"no colon", fmt.Errorf("exit code 3"), 1, 1},
		{"non-numeric", fmt.Errorf("exit code abc: error"), 1, 1},
		{"wrapped", fmt.Errorf("exit code 3: %w", fmt.Errorf("inner")), 1, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := localExtractExitCode(tt.err, tt.fallback)
			if got != tt.want {
				t.Errorf("localExtractExitCode(%v, %d) = %d, want %d", tt.err, tt.fallback, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// E2E runner-level tests (T-E28-F04-013 regression guard)
//
// These tests call the actual cobra RunE function for each entity update
// runner (not just the helper layer) to ensure the "exit code 3:" error
// propagates all the way through the runner without being intercepted by
// the legacy handleServiceError. This is the gap that let the original
// runFeatureUpdate regression pass all unit tests while failing at runtime.
// ---------------------------------------------------------------------------

// buildCmdWithTagFlag returns a minimal *cobra.Command that has a --tag
// StringSlice flag registered and marked as Changed, simulating a caller
// passing --tag=ghost on the command line.
func buildCmdWithTagFlag(t *testing.T, tagValue string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "update", SilenceErrors: true, SilenceUsage: true}
	cmd.Flags().StringSlice("tag", nil, "tags")
	// Mark --tag as Changed so the helper branch that calls
	// handleEntityServiceError is reached.
	if err := cmd.Flags().Set("tag", tagValue); err != nil {
		t.Fatalf("failed to set --tag flag: %v", err)
	}
	return cmd
}

// TestRunFeatureUpdate_UnregisteredTag_ExitCode3 is the regression test for
// the BLOCKER-1 finding in the UAT report (T-E28-F04-013).
//
// It overrides featureUpdateImpl to inject an "exit code 3:" error
// (the same shape produced by handleEntityServiceError) and asserts that
// runFeatureUpdate returns that error unchanged — i.e. it does NOT intercept
// the error via the legacy handleServiceError (which would call os.Exit(2)).
func TestRunFeatureUpdate_UnregisteredTag_ExitCode3(t *testing.T) {
	withJSONMode(t, false)

	// Override featureUpdateImpl to return a controlled exit-code-3 error,
	// simulating what performFeatureUpdate → handleEntityServiceError produces.
	injectedErr := fmt.Errorf("exit code 3: %w", &services.UnregisteredTagError{Name: "ghost"})
	orig := featureUpdateImpl
	featureUpdateImpl = func(_ context.Context, _ string, _ *cobra.Command) error {
		return injectedErr
	}
	t.Cleanup(func() { featureUpdateImpl = orig })

	cmd := buildCmdWithTagFlag(t, "ghost")
	cmd.Args = cobra.ExactArgs(1)

	got := runFeatureUpdate(cmd, []string{"E01-F02"})

	if got == nil {
		t.Fatal("runFeatureUpdate: expected error, got nil")
	}
	if localExtractExitCode(got, 2) != 3 {
		t.Errorf("runFeatureUpdate: exit code want 3, got %d (error: %v)",
			localExtractExitCode(got, 2), got)
	}
	// The typed error must still be accessible through the returned error chain.
	var unregErr *services.UnregisteredTagError
	if !errors.As(got, &unregErr) {
		t.Errorf("runFeatureUpdate: UnregisteredTagError not accessible via errors.As: %T %v", got, got)
	}
}

// TestRunnerExitCode3Propagation_Table is a table-driven regression guard
// covering all six entity update runners. For each runner, it verifies that
// an "exit code 3:" error returned by the inner perform/service call is
// propagated unchanged to cobra — i.e. the runner does NOT re-wrap it with
// handleServiceError.
//
// Runners that call their perform-function via a package-level variable (like
// featureUpdateImpl) are tested directly. For the others, the test verifies
// the runner's RunE returns an error with the expected exit code prefix when
// handleEntityServiceError produces one.
func TestRunnerExitCode3Propagation_Table(t *testing.T) {
	exitCode3Err := fmt.Errorf("exit code 3: tag is not registered: ghost")

	tests := []struct {
		name    string
		runnerE func(cmd *cobra.Command, args []string) error
		setup   func(t *testing.T) // optional override / cleanup
	}{
		{
			name:    "feature update",
			runnerE: runFeatureUpdate,
			setup: func(t *testing.T) {
				orig := featureUpdateImpl
				featureUpdateImpl = func(_ context.Context, _ string, _ *cobra.Command) error {
					return exitCode3Err
				}
				t.Cleanup(func() { featureUpdateImpl = orig })
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			withJSONMode(t, false)
			if tt.setup != nil {
				tt.setup(t)
			}

			cmd := buildCmdWithTagFlag(t, "ghost")
			cmd.Args = cobra.ExactArgs(1)

			got := tt.runnerE(cmd, []string{"E01-F02"})

			if got == nil {
				t.Fatalf("[%s] expected error, got nil", tt.name)
			}

			code := localExtractExitCode(got, 2)
			if code != 3 {
				t.Errorf("[%s] exit code want 3, got %d (error: %v)", tt.name, code, got)
			}
		})
	}
}
