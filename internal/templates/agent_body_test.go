package templates

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRenderAgentBody_DashToUnderscoreAlias(t *testing.T) {
	body := "Task: <task-id> in file <file_path> on branch <branch>."
	vars := map[string]string{
		"task_id":   "T-E01-F41-003",
		"file_path": "internal/observability/recorder.go",
		"branch":    "shark2-engine",
	}
	got := RenderAgentBody(body, vars)
	want := "Task: T-E01-F41-003 in file internal/observability/recorder.go on branch shark2-engine."
	if got != want {
		t.Errorf("RenderAgentBody result mismatch\n  got:  %q\n  want: %q", got, want)
	}
}

func TestRenderAgentBody_LeavesUnknownTokens(t *testing.T) {
	body := "<task-id> doing <unknown-thing>"
	vars := map[string]string{"task_id": "T-E01-F41-003"}
	got := RenderAgentBody(body, vars)
	if got != "T-E01-F41-003 doing <unknown-thing>" {
		t.Errorf("unexpected: %q", got)
	}
}

func TestRenderAgentBody_IgnoresNonTokenAngleBrackets(t *testing.T) {
	// HTML and comparison operators must not be touched.
	body := `if x < y && y > 0 { /* <p class="x">text</p> */ }`
	got := RenderAgentBody(body, map[string]string{"x": "1"})
	if got != body {
		t.Errorf("body changed unexpectedly: %q", got)
	}
}

func TestFirstUnrenderedToken_FindsSurvivor(t *testing.T) {
	s := "rendered T-E01-F41-003 still has <unknown-thing> left"
	tok, ok := FirstUnrenderedToken(s)
	if !ok {
		t.Fatal("expected to find a survivor token")
	}
	if tok != "<unknown-thing>" {
		t.Errorf("expected <unknown-thing>, got %q", tok)
	}
}

func TestFirstUnrenderedToken_CleanString(t *testing.T) {
	s := "fully rendered T-E01-F41-003 ready to dispatch"
	if _, ok := FirstUnrenderedToken(s); ok {
		t.Error("expected no surviving tokens")
	}
}

// TestRenderAndLintAgentBody_RejectsUnfilledToken covers the loudness
// guarantee: a body with `<token>` and no matching var must surface as
// an UnrenderedTokenError naming the offending token and the agent file.
func TestRenderAndLintAgentBody_RejectsUnfilledToken(t *testing.T) {
	body := "Task: <task_id>. Bad: <unfilled>."
	vars := map[string]string{"task_id": "T-E01-F01-001"}

	_, err := RenderAndLintAgentBody(body, "developer", vars)
	if err == nil {
		t.Fatal("expected error for unfilled token, got nil")
	}
	var tokErr *UnrenderedTokenError
	if !errors.As(err, &tokErr) {
		t.Fatalf("expected *UnrenderedTokenError, got %T: %v", err, err)
	}
	if tokErr.Token != "<unfilled>" {
		t.Errorf("expected Token=<unfilled>, got %q", tokErr.Token)
	}
	if tokErr.AgentType != "developer" {
		t.Errorf("expected AgentType=developer, got %q", tokErr.AgentType)
	}
}

// TestRenderAndLintAgentBody_AcceptsFullyRendered covers the happy path:
// every `<token>` in the body has a matching var, so rendering succeeds
// and the result has no surviving placeholders.
func TestRenderAndLintAgentBody_AcceptsFullyRendered(t *testing.T) {
	body := "Task: <task_id> on <branch>."
	vars := map[string]string{
		"task_id": "T-E01-F01-001",
		"branch":  "shark2-engine",
	}

	got, err := RenderAndLintAgentBody(body, "developer", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "Task: T-E01-F01-001 on shark2-engine."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestRenderAndLintAgentBody_IgnoresProseInCodeFences guarantees the lint
// inherits FirstUnrenderedToken's fence-stripping behavior: an `<example>`
// token sitting inside a triple-backtick code block is documentation, not
// a missed substitution, and must not trip the lint.
func TestRenderAndLintAgentBody_IgnoresProseInCodeFences(t *testing.T) {
	body := "Use this:\n```\nshark next <example>\n```\nDone <task_id>."
	vars := map[string]string{"task_id": "T-E01-F01-001"}

	got, err := RenderAndLintAgentBody(body, "developer", vars)
	if err != nil {
		t.Fatalf("unexpected error (fence-stripping regression?): %v", err)
	}
	if !strings.Contains(got, "<example>") {
		t.Errorf("fenced <example> should pass through untouched, got %q", got)
	}
}

func TestAugmentPlaceholderAliases_TaskKeys(t *testing.T) {
	vars := map[string]string{
		"task_key":    "T-E01-F41-003",
		"epic_key":    "E01",
		"feature_key": "E01-F41",
	}
	AugmentPlaceholderAliases(vars)

	checks := map[string]string{
		"task":    "T-E01-F41-003",
		"task_id": "T-E01-F41-003",
		"epic":    "E01",
		"epic_id": "E01",
		"feature": "E01-F41",
	}
	for k, want := range checks {
		if got := vars[k]; got != want {
			t.Errorf("vars[%q]=%q want %q", k, got, want)
		}
	}
}

// TestRenderAgentBody_DeveloperBody_NoDoublePrefixedBranchRef is the
// regression test for B020. The shipped developer agent body must not
// compose `<epic>-<feature>` (or `<epic-key>-<feature-key>`) anywhere
// outside fenced code blocks of pure documentation: `feature_key` already
// contains the epic prefix, so `E01` + `-` + `E01-F41` rendered as
// `E01-E01-F41`, breaking the suggested `git checkout -b` snippet and PR
// header line for any worker that copies them literally.
func TestRenderAgentBody_DeveloperBody_NoDoublePrefixedBranchRef(t *testing.T) {
	// Resolve both shipped copies of developer.md relative to this test
	// file so the assertion catches drift in either location.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed; cannot locate developer.md")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))

	paths := []string{
		filepath.Join(repoRoot, "shark-data", "agents", "developer.md"),
		filepath.Join(repoRoot, "internal", "sharkdata", "default_data", "agents", "developer.md"),
	}

	vars := map[string]string{
		"epic_key":    "E01",
		"feature_key": "E01-F41",
		"task_key":    "T-E01-F41-003",
		"entity_type": "task",
		"key":         "T-E01-F41-003",
	}
	AugmentPlaceholderAliases(vars)

	for _, p := range paths {
		body, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		got := RenderAgentBody(string(body), vars)
		if strings.Contains(got, "E01-E01-F41") {
			// Report the offending line for easy diagnosis.
			for i, line := range strings.Split(got, "\n") {
				if strings.Contains(line, "E01-E01-F41") {
					t.Errorf("%s line %d contains double-prefixed branch ref: %q", p, i+1, line)
				}
			}
		}
	}
}

// TestRenderAgentBody_DeveloperBody_FeaturePathsAreSlugSuffixed is the
// regression test for B021. Before the fix, the developer agent body composed
// `docs/plan/<epic>/<feature>/feature.md`, with `<epic>` and `<feature>`
// resolving to bare entity keys (`E01`, `E01-F41`). The rendered prompt then
// referenced `docs/plan/E01/E01-F41/feature.md`, which doesn't exist on disk
// in projects that use slug-suffixed feature directories
// (`docs/plan/E01-content-ingestion/E01-F41-lexicon-observation/feature.md`).
//
// The fix introduces `<epic-dir>` / `<feature-dir>` placeholder tokens that
// resolve to the actual on-disk parent directories (derived from
// `file_path`) and updates the agent files to use them instead of
// reconstructing the path from keys.
//
// This test asserts that, with `epic_dir` / `feature_dir` populated to a
// representative slug-suffixed shape, the rendered body:
//  1. Contains the correct slug-suffixed path, AND
//  2. Does NOT contain the buggy bare-key shape `docs/plan/E01/E01-F41/`.
//
// (1) prevents the path tokens from being silently dropped on drift (e.g.,
// someone reverting only one occurrence). (2) is the behavior the bug
// report calls out — the body must stop sprinkling cosmetic key-only paths.
func TestRenderAgentBody_DeveloperBody_FeaturePathsAreSlugSuffixed(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed; cannot locate developer.md")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))

	paths := []string{
		filepath.Join(repoRoot, "shark-data", "agents", "developer.md"),
		filepath.Join(repoRoot, "internal", "sharkdata", "default_data", "agents", "developer.md"),
	}

	// Populate the same placeholder map shape `shark next` uses for a task
	// living under a slug-suffixed feature. epic_dir / feature_dir are the
	// new placeholders introduced by the B021 fix.
	vars := map[string]string{
		"epic_key":    "E01",
		"feature_key": "E01-F41",
		"task_key":    "T-E01-F41-003",
		"entity_type": "task",
		"key":         "T-E01-F41-003",
		"epic_dir":    "docs/plan/E01-content-ingestion",
		"feature_dir": "docs/plan/E01-content-ingestion/E01-F41-lexicon-observation",
	}
	AugmentPlaceholderAliases(vars)

	const buggyPath = "docs/plan/E01/E01-F41/"
	const wantPath = "docs/plan/E01-content-ingestion/E01-F41-lexicon-observation/"

	for _, p := range paths {
		body, err := os.ReadFile(p)
		if err != nil {
			// shark-data/ is untracked in some checkouts during the F4
			// migration; skip its absence rather than failing.
			if os.IsNotExist(err) {
				t.Logf("skipping %s (not present in this checkout)", p)
				continue
			}
			t.Fatalf("read %s: %v", p, err)
		}
		got := RenderAgentBody(string(body), vars)

		// 1. The rendered body must NOT carry the bare-key path shape
		// anywhere outside fenced code blocks. We scan the rendered body
		// directly because any escaped/literal occurrences (e.g., the bug
		// report's example pasted as-is in prose) would be a regression.
		if strings.Contains(got, buggyPath) {
			for i, line := range strings.Split(got, "\n") {
				if strings.Contains(line, buggyPath) {
					t.Errorf("%s line %d still contains bare-key path %q (B021 regression): %q",
						p, i+1, buggyPath, strings.TrimSpace(line))
				}
			}
		}

		// 2. The rendered body must contain the slug-suffixed path —
		// proves the new <feature-dir> token is actually wired through
		// at the locations that used to use <epic>/<feature>. Without
		// this check, a regression could simply delete the path
		// references and the test in step 1 would still pass.
		if !strings.Contains(got, wantPath) {
			t.Errorf("%s rendered body missing slug-suffixed path %q — <feature-dir> token not wired through",
				p, wantPath)
		}
	}
}

func TestAugmentPlaceholderAliases_DoesNotClobberExisting(t *testing.T) {
	vars := map[string]string{
		"task_key": "T-E01-F41-003",
		"task_id":  "pre-existing",
	}
	AugmentPlaceholderAliases(vars)
	if vars["task_id"] != "pre-existing" {
		t.Errorf("task_id should not be overwritten; got %q", vars["task_id"])
	}
}
