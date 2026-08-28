// Package runner
//
// Transcript file capture tests.
//
// These tests verify the opt-in per-dispatch transcript file capture:
//
//   - When observability.capture_agent_transcripts == true, RunController
//     writes a transcript file for each agent dispatch at
//     {project_root}/.shark/runs/{run_id}/{entity_key}/{stage_n}-{status}-{provider}.log
//     with file mode 0644 and parent directory mode 0755.
//   - File contents use the EXACT format:
//     COMMAND: <cmd>
//     EXIT: <code>
//     DURATION: <ms>ms
//     ---STDOUT---
//     <stdout>
//     ---STDERR---
//     <stderr>
//   - run.stage.complete / run.stage.error events carry a `transcript_path`
//     attribute equal to the PROJECT-RELATIVE path (".shark/runs/...") when a
//     transcript was successfully written.
//   - Write failures are non-fatal: exactly ONE run.transcript.warning is
//     emitted per run, subsequent writes are suppressed, and the run continues.
//   - When capture_agent_transcripts == false, NO transcript files are written
//     and NO transcript_path attributes appear on any event.
//
// Golden rule: no real database, no real agent. The only real I/O is a
// t.TempDir()-scoped filesystem (acting as the project root).
package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// ---------------------------------------------------------------------------
// Shared helpers (transcript-specific)
// ---------------------------------------------------------------------------

// obsWithTranscripts returns an ObservabilityConfig with Enabled=true and
// CaptureAgentTranscripts set to the supplied value. truncate may be 0 for
// default (4096). This is a small wrapper over obsEnabled() that sets the
// opt-in CaptureAgentTranscripts field.
func obsWithTranscripts(truncate int, capture bool) config.ObservabilityConfig {
	cfg := obsEnabled(truncate)
	cfg.CaptureAgentTranscripts = capture
	return cfg
}

// expectedTranscript returns the exact byte content that the writer must
// produce for a dispatch with the given fields. It is the ONE source of truth
// for the format string in these tests — if the production code deviates (even
// by a newline), the content assertions will fail.
func expectedTranscript(command string, exit int, durationMS int64, stdout, stderr string) string {
	return fmt.Sprintf(
		"COMMAND: %s\nEXIT: %d\nDURATION: %dms\n---STDOUT---\n%s\n---STDERR---\n%s",
		command, exit, durationMS, stdout, stderr,
	)
}

// relPathFor returns the project-relative transcript path that the controller
// should emit on slog events and write on disk, for stage number n, status s,
// and provider p under the given run ID and entity key.
func relPathFor(runID, entityKey string, n int, status, provider string) string {
	return filepath.Join(".shark", "runs", runID, entityKey, fmt.Sprintf("%d-%s-%s.log", n, status, provider))
}

// ---------------------------------------------------------------------------
// AC: capture_agent_transcripts=true writes a file for the Claude dispatcher
// ---------------------------------------------------------------------------

// TestTranscript_EnabledWritesFile_ClaudeDispatcher drives a single successful
// stage with provider="anthropic" (Claude) and verifies:
//   - the file is written under .shark/runs/{run_id}/1-in_development-anthropic.log
//   - the parent directory mode is 0755
//   - the file mode is 0644
//   - the contents match the documented on-disk format EXACTLY
func TestTranscript_EnabledWritesFile_ClaudeDispatcher(t *testing.T) {
	_ = captureSlog(t) // suppress noisy handler; content of slog not asserted here

	root := t.TempDir()
	runID := "run-claude-001"

	ctrl := happyPathFixture(t, func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
		return &DispatchResult{
			ExitCode: 0,
			Stdout:   "hello from claude",
			Stderr:   "",
			Duration: mustParseMS(t, 250),
			Command:  "claude -p 'do work'",
		}, nil
	})
	opts := RunOptions{
		RunID:         runID,
		ProjectRoot:   root,
		Observability: obsWithTranscripts(0, true),
	}
	res, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.Outcome != "completed" {
		t.Fatalf("expected outcome=completed, got %s", res.Outcome)
	}

	// File should exist at the expected path (provider from happyPathFixture
	// is "anthropic" — the Claude dispatcher's registration key).
	rel := relPathFor(runID, "E07-F01-001", 1, "in_development", "anthropic")
	abs := filepath.Join(root, rel)
	info, err := os.Stat(abs)
	if err != nil {
		t.Fatalf("expected transcript file at %s, got stat err: %v", abs, err)
	}

	// Skip permission bit assertions on Windows where unix mode bits aren't meaningful.
	if runtime.GOOS != "windows" {
		if got := info.Mode().Perm(); got != 0o644 {
			t.Errorf("file perm = %o, want 0644", got)
		}
		parent, err := os.Stat(filepath.Dir(abs))
		if err != nil {
			t.Fatalf("parent dir stat err: %v", err)
		}
		if got := parent.Mode().Perm(); got != 0o755 {
			t.Errorf("parent dir perm = %o, want 0755", got)
		}
	}

	// Exact content match.
	got, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("read transcript: %v", err)
	}
	want := expectedTranscript("claude -p 'do work'", 0, 250, "hello from claude", "")
	if string(got) != want {
		t.Errorf("transcript content mismatch.\n got=%q\nwant=%q", string(got), want)
	}
}

// ---------------------------------------------------------------------------
// AC: capture_agent_transcripts=true writes a file for the Codex dispatcher
// ---------------------------------------------------------------------------

// TestTranscript_EnabledWritesFile_CodexDispatcher is the same scenario as the
// Claude test but with provider="codex", verifying dispatcher-agnosticism.
func TestTranscript_EnabledWritesFile_CodexDispatcher(t *testing.T) {
	_ = captureSlog(t)

	root := t.TempDir()
	runID := "run-codex-001"

	ctrl := codexHappyPathFixture(t, func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
		return &DispatchResult{
			ExitCode: 0,
			Stdout:   "codex stdout",
			Stderr:   "codex warning",
			Duration: mustParseMS(t, 99),
			Command:  "codex exec --skip-git-repo-check 'do work'",
		}, nil
	})
	opts := RunOptions{
		RunID:         runID,
		ProjectRoot:   root,
		Observability: obsWithTranscripts(0, true),
	}
	res, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.Outcome != "completed" {
		t.Fatalf("expected outcome=completed, got %s", res.Outcome)
	}

	rel := relPathFor(runID, "E07-F01-001", 1, "in_development", "codex")
	abs := filepath.Join(root, rel)
	got, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("read transcript (%s): %v", abs, err)
	}
	want := expectedTranscript("codex exec --skip-git-repo-check 'do work'", 0, 99, "codex stdout", "codex warning")
	if string(got) != want {
		t.Errorf("transcript content mismatch.\n got=%q\nwant=%q", string(got), want)
	}
}

// ---------------------------------------------------------------------------
// AC: capture_agent_transcripts=false writes nothing + no transcript_path attr
// ---------------------------------------------------------------------------

// TestTranscript_Disabled_NoFileWritten confirms the opt-in semantics: when
// capture_agent_transcripts is false, the .shark/runs/{run_id} directory is
// never created, no .log file is written, and no event carries a
// transcript_path attribute.
func TestTranscript_Disabled_NoFileWritten(t *testing.T) {
	buf := captureSlog(t)

	root := t.TempDir()
	runID := "run-disabled-001"

	ctrl := happyPathFixture(t, nil)
	opts := RunOptions{
		RunID:         runID,
		ProjectRoot:   root,
		Observability: obsWithTranscripts(0, false), // disabled
	}
	res, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.Outcome != "completed" {
		t.Fatalf("expected outcome=completed, got %s", res.Outcome)
	}

	// Run-id directory must not exist.
	runDir := filepath.Join(root, ".shark", "runs", runID)
	if _, err := os.Stat(runDir); !os.IsNotExist(err) {
		t.Errorf("expected run dir %s NOT to exist, stat err = %v", runDir, err)
	}

	// No event anywhere should carry transcript_path.
	events := parseEvents(t, buf)
	for i, ev := range events {
		if _, ok := ev["transcript_path"]; ok {
			t.Errorf("event %d (msg=%v) unexpectedly has transcript_path attr: %v",
				i, ev["msg"], ev)
		}
	}
}

// ---------------------------------------------------------------------------
// AC: run.stage.complete carries transcript_path with relative path
// ---------------------------------------------------------------------------

// TestTranscript_Enabled_StageCompleteHasPathAttr asserts that the successful
// stage emits exactly one run.stage.complete event whose transcript_path
// attribute equals the project-relative file path.
func TestTranscript_Enabled_StageCompleteHasPathAttr(t *testing.T) {
	buf := captureSlog(t)

	root := t.TempDir()
	runID := "run-complete-attr"

	ctrl := happyPathFixture(t, func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
		return &DispatchResult{
			ExitCode: 0,
			Stdout:   "ok",
			Duration: mustParseMS(t, 10),
			Command:  "claude -p x",
		}, nil
	})
	opts := RunOptions{
		RunID:         runID,
		ProjectRoot:   root,
		Observability: obsWithTranscripts(0, true),
	}
	if _, err := ctrl.Run(context.Background(), "E07-F01-001", opts); err != nil {
		t.Fatalf("Run: %v", err)
	}

	completes := eventsByMsg(parseEvents(t, buf), "run.stage.complete")
	if len(completes) != 1 {
		t.Fatalf("expected exactly 1 run.stage.complete, got %d", len(completes))
	}
	wantRel := relPathFor(runID, "E07-F01-001", 1, "in_development", "anthropic")
	got, ok := completes[0]["transcript_path"].(string)
	if !ok {
		t.Fatalf("run.stage.complete missing transcript_path (attrs: %v)", completes[0])
	}
	if got != wantRel {
		t.Errorf("transcript_path = %q, want %q", got, wantRel)
	}
}

// ---------------------------------------------------------------------------
// AC: run.stage.error carries transcript_path with relative path
// ---------------------------------------------------------------------------

// TestTranscript_Enabled_StageErrorHasPathAttr triggers a dispatch failure
// (non-zero exit code with no Go error — the mock-style path) and asserts that
// the run.stage.error event carries the transcript_path. The transcript file
// must still be written on disk because it captures the failed run's output
// for operator debugging.
func TestTranscript_Enabled_StageErrorHasPathAttr(t *testing.T) {
	buf := captureSlog(t)

	root := t.TempDir()
	runID := "run-error-attr"

	// Trigger the non-zero exit code path (no Go error).
	ctrl := happyPathFixture(t, func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
		return &DispatchResult{
			ExitCode: 7,
			Stdout:   "partial output",
			Stderr:   "boom",
			Duration: mustParseMS(t, 42),
			Command:  "claude -p x",
		}, nil
	})
	opts := RunOptions{
		RunID:         runID,
		ProjectRoot:   root,
		Observability: obsWithTranscripts(0, true),
	}
	res, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.Outcome != "failed" {
		t.Fatalf("expected outcome=failed, got %s", res.Outcome)
	}

	errors_ := eventsByMsg(parseEvents(t, buf), "run.stage.error")
	if len(errors_) != 1 {
		t.Fatalf("expected exactly 1 run.stage.error, got %d", len(errors_))
	}
	wantRel := relPathFor(runID, "E07-F01-001", 1, "in_development", "anthropic")
	got, ok := errors_[0]["transcript_path"].(string)
	if !ok {
		t.Fatalf("run.stage.error missing transcript_path (attrs: %v)", errors_[0])
	}
	if got != wantRel {
		t.Errorf("transcript_path = %q, want %q", got, wantRel)
	}

	// File must be on disk too — failed runs still get a transcript.
	abs := filepath.Join(root, wantRel)
	content, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("expected transcript at %s, got err: %v", abs, err)
	}
	want := expectedTranscript("claude -p x", 7, 42, "partial output", "boom")
	if string(content) != want {
		t.Errorf("transcript content mismatch.\n got=%q\nwant=%q", string(content), want)
	}
}

// ---------------------------------------------------------------------------
// AC: write failures emit exactly one run.transcript.warning per run
// ---------------------------------------------------------------------------

// TestTranscript_WriteFailure_EmitsWarningOnce_RunContinues verifies:
//   - When the first transcript write fails, one run.transcript.warning event
//     is emitted at WARN level carrying run_id, path, and error.
//   - The failing stage's run.stage.complete event does NOT include a
//     transcript_path attribute.
//   - Subsequent stages do NOT retry: no second warning is ever emitted (the
//     run-scoped disable flag suppresses all further attempts).
//   - The run itself still completes successfully — write failures are
//     non-fatal.
//
// Strategy: make ".shark" a *file* (not a directory) under the project root,
// which guarantees a non-recoverable MkdirAll failure for every subsequent
// write attempt regardless of platform.
func TestTranscript_WriteFailure_EmitsWarningOnce_RunContinues(t *testing.T) {
	buf := captureSlog(t)

	root := t.TempDir()
	// Force transcript dir creation to fail: create a regular file at ".shark"
	// so MkdirAll cannot create .shark/runs beneath it.
	block := filepath.Join(root, ".shark")
	if err := os.WriteFile(block, []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to set up failure sentinel: %v", err)
	}

	runID := "run-write-fail"

	// Drive two successful stages in one run so we can assert "at most one"
	// warning across the run.
	ctrl := twoStageFixture(t, func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
		return &DispatchResult{
			ExitCode: 0,
			Stdout:   "ok",
			Duration: mustParseMS(t, 1),
			Command:  "claude -p x",
		}, nil
	})
	opts := RunOptions{
		RunID:         runID,
		ProjectRoot:   root,
		Observability: obsWithTranscripts(0, true),
	}
	res, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.Outcome != "completed" {
		t.Fatalf("expected outcome=completed despite write failure, got %s (error=%s)", res.Outcome, res.Error)
	}
	// Expect two stages actually executed.
	if len(res.Stages) < 2 {
		t.Fatalf("expected 2 stages executed, got %d", len(res.Stages))
	}

	events := parseEvents(t, buf)

	warnings := eventsByMsg(events, "run.transcript.warning")
	if len(warnings) != 1 {
		t.Fatalf("expected exactly 1 run.transcript.warning, got %d: %v", len(warnings), warnings)
	}
	w := warnings[0]
	if got, _ := w["run_id"].(string); got != runID {
		t.Errorf("warning.run_id = %q, want %q", got, runID)
	}
	if _, ok := w["path"]; !ok {
		t.Errorf("warning missing `path` attribute: %v", w)
	}
	if _, ok := w["error"]; !ok {
		t.Errorf("warning missing `error` attribute: %v", w)
	}
	// Must be at WARN level.
	if lvl, _ := w["level"].(string); !strings.EqualFold(lvl, "WARN") {
		t.Errorf("warning level = %q, want WARN", lvl)
	}

	// Neither run.stage.complete event should carry transcript_path — writes
	// failed for both, and the run-scoped disable flag blocked the second
	// attempt outright (so the second complete wouldn't have a path either way).
	for i, c := range eventsByMsg(events, "run.stage.complete") {
		if _, ok := c["transcript_path"]; ok {
			t.Errorf("run.stage.complete[%d] unexpectedly has transcript_path: %v", i, c)
		}
	}
}

// ---------------------------------------------------------------------------
// AC: stage counter increments per dispatch (filename numbering)
// ---------------------------------------------------------------------------

// TestTranscript_MultipleStages_IncrementCounter runs two stages in one run and
// asserts both transcript files exist with monotonically increasing stage
// numbers (1-in_development-anthropic.log and 2-code_review-anthropic.log).
func TestTranscript_MultipleStages_IncrementCounter(t *testing.T) {
	_ = captureSlog(t)

	root := t.TempDir()
	runID := "run-multi"

	ctrl := twoStageFixture(t, func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
		return &DispatchResult{
			ExitCode: 0,
			Stdout:   "stage stdout",
			Duration: mustParseMS(t, 5),
			Command:  "claude -p stage",
		}, nil
	})
	opts := RunOptions{
		RunID:         runID,
		ProjectRoot:   root,
		Observability: obsWithTranscripts(0, true),
	}
	res, err := ctrl.Run(context.Background(), "E07-F01-001", opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Outcome != "completed" {
		t.Fatalf("expected completed, got %s (err=%s)", res.Outcome, res.Error)
	}

	// Both files should be present, numbered 1 and 2.
	f1 := filepath.Join(root, relPathFor(runID, "E07-F01-001", 1, "in_development", "anthropic"))
	f2 := filepath.Join(root, relPathFor(runID, "E07-F01-001", 2, "code_review", "anthropic"))
	if _, err := os.Stat(f1); err != nil {
		t.Errorf("missing stage 1 transcript at %s: %v", f1, err)
	}
	if _, err := os.Stat(f2); err != nil {
		t.Errorf("missing stage 2 transcript at %s: %v", f2, err)
	}
}

// ---------------------------------------------------------------------------
// AC (B052): sibling cascade children never collide on transcript filename
// ---------------------------------------------------------------------------

// TestTranscript_CascadeChildrenProduceDistinctFiles reproduces B052: cascade
// children inherit the parent's RunID unchanged, and each child's own Run()
// independently restarts its stage counter at 1. Before B052's fix, two
// sibling children dispatching in the same status/provider ("in_development"/
// "anthropic") both resolved to the SAME transcript path
// (.shark/runs/{runID}/1-in_development-anthropic.log), so the second child's
// os.WriteFile silently truncated the first child's transcript.
//
// This test drives a real cascade — a parent RunController whose RunChild
// constructs a REAL child *RunController* per child (mirroring production
// wiring in internal/cli/commands/run.go), both children sharing the same
// RunID and dispatching their first stage as "in_development"/"anthropic" —
// and asserts BOTH children's transcript files exist on disk, each under its
// own entity-key subdirectory, with distinct content attributable to the
// correct child.
func TestTranscript_CascadeChildrenProduceDistinctFiles(t *testing.T) {
	_ = captureSlog(t)

	root := t.TempDir()
	runID := "run-cascade-collision"
	childKeys := []string{"E07-F01-T01", "E07-F01-T02"}

	dispatcher := &MockDispatcher{
		DispatchFunc: func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
			return &DispatchResult{
				ExitCode: 0,
				Stdout:   "output for " + input.EntityKey,
				Duration: mustParseMS(t, 10),
				Command:  "claude -p " + input.EntityKey,
			}, nil
		},
	}
	dispatchers := map[string]AgentDispatcher{"anthropic": dispatcher}

	childTransitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{
				CurrentStatus: "in_development",
				IsTerminal:    false,
				AvailableTransitions: []services.TransitionInfoWithAction{
					{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
				},
			}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	childActionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "anthropic",
				Instruction: "do work",
			}, nil
		},
	}

	// runChild mirrors production wiring: a real child RunController is built
	// per child and inherits the parent's RunOptions (same RunID, ProjectRoot,
	// Observability) unchanged, exactly as internal/cli/commands/run.go's
	// runChild closure and controller.go's handleCascade `childOpts := opts` do.
	runChild := func(ctx context.Context, childType, key string, childOpts RunOptions) (*RunResult, error) {
		childCtrl, err := NewRunController(RunControllerDeps{
			Transitioner: childTransitioner,
			Placeholders: &MockPlaceholderGen{},
			ActionSvc:    childActionSvc,
			WorkflowSvc:  defaultWorkflowSvc(),
			Dispatchers:  dispatchers,
		})
		if err != nil {
			t.Fatalf("build child controller for %s: %v", key, err)
		}
		childOpts.EntityType = childType
		return childCtrl.Run(ctx, key, childOpts)
	}

	cascadeSvc := &MockCascadeChildrenService{
		DescribeDispatchableChildrenFunc: func(ctx context.Context, entityType, key string) (services.CascadeChildrenState, error) {
			return services.CascadeChildrenState{
				Children: []services.CascadeChild{
					{Key: childKeys[0], EntityType: "task"},
					{Key: childKeys[1], EntityType: "task"},
				},
				TotalChildren:       2,
				NonTerminalChildren: 2,
			}, nil
		},
	}

	parentTransitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			return &services.NextStatusInfo{CurrentStatus: "active", IsTerminal: false}, nil
		},
	}
	parentActionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{Action: config.ActionCascade}, nil
		},
	}

	ctrl, err := NewRunController(RunControllerDeps{
		Transitioner: parentTransitioner,
		Placeholders: &MockPlaceholderGen{},
		ActionSvc:    parentActionSvc,
		WorkflowSvc:  defaultWorkflowSvc(),
		Dispatchers:  dispatchers,
		ChildrenSvc:  cascadeSvc,
		RunChild:     runChild,
	})
	if err != nil {
		t.Fatalf("NewRunController: %v", err)
	}

	opts := RunOptions{
		RunID:         runID,
		ProjectRoot:   root,
		EntityType:    "feature",
		Observability: obsWithTranscripts(0, true),
	}
	if _, err := ctrl.Run(context.Background(), "E07-F01", opts); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Both children's first-stage transcripts must exist, each under its own
	// entity-key subdirectory, with content attributable to the correct child.
	for _, key := range childKeys {
		rel := relPathFor(runID, key, 1, "in_development", "anthropic")
		abs := filepath.Join(root, rel)
		content, err := os.ReadFile(abs)
		if err != nil {
			t.Fatalf("expected distinct transcript for cascade child %s at %s, got err: %v", key, abs, err)
		}
		wantSub := "output for " + key
		if !strings.Contains(string(content), wantSub) {
			t.Errorf("transcript for %s missing expected content %q; got %q", key, wantSub, string(content))
		}
		// Guard against the pre-fix collision: neither child's transcript may
		// contain the OTHER child's output (which is what silent os.WriteFile
		// truncation would produce if both children wrote the same path).
		for _, other := range childKeys {
			if other == key {
				continue
			}
			if strings.Contains(string(content), "output for "+other) {
				t.Errorf("transcript for %s unexpectedly contains sibling %s's output — collision not prevented", key, other)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// AC: .shark/runs/ is ignored in .gitignore
// ---------------------------------------------------------------------------

// TestGitignore_ContainsShaRunsEntry ensures that transcript artefacts stay out
// of the project repository — `.shark/runs/` must be in .gitignore.
func TestGitignore_ContainsShaRunsEntry(t *testing.T) {
	// Walk up from this test's runtime directory to find .gitignore. Since
	// go test runs with the test's package directory as CWD, we climb to the
	// module root.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// internal/runner -> module root
	repoRoot := filepath.Join(cwd, "..", "..")
	gitignore := filepath.Join(repoRoot, ".gitignore")

	data, err := os.ReadFile(gitignore)
	if err != nil {
		t.Fatalf("read %s: %v", gitignore, err)
	}
	entry := ".shark/runs/"
	if !bytes.Contains(data, []byte(entry)) {
		t.Errorf(".gitignore does not contain %q", entry)
	}
}

// ---------------------------------------------------------------------------
// Small helpers used above
// ---------------------------------------------------------------------------

// mustParseMS converts an integer millisecond value into time.Duration.
// Rewritten as a helper so we keep test setup terse.
func mustParseMS(t *testing.T, ms int64) time.Duration {
	t.Helper()
	return time.Duration(ms) * time.Millisecond
}

// codexHappyPathFixture is a variant of happyPathFixture whose
// PopulatedAction uses provider="codex" (to exercise the Codex-dispatcher file
// path). Only provider differs from happyPathFixture's defaults.
func codexHappyPathFixture(t *testing.T, dispatchFunc func(ctx context.Context, input DispatchInput) (*DispatchResult, error)) *RunController {
	t.Helper()

	calls := 0
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			calls++
			if calls == 1 {
				return &services.NextStatusInfo{
					CurrentStatus: "in_development",
					IsTerminal:    false,
					AvailableTransitions: []services.TransitionInfoWithAction{
						{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
					},
				}, nil
			}
			return &services.NextStatusInfo{CurrentStatus: "completed", IsTerminal: true}, nil
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "codex",
				Instruction: "do work",
			}, nil
		},
	}
	if dispatchFunc == nil {
		dispatchFunc = func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
			return &DispatchResult{ExitCode: 0, Stdout: "ok", Command: "codex exec x"}, nil
		}
	}
	dispatchers := map[string]AgentDispatcher{
		"codex": &MockDispatcher{DispatchFunc: dispatchFunc},
	}
	return makeController(t, transitioner, actionSvc, dispatchers)
}

// twoStageFixture drives two successful spawn_agent stages in a single run:
// in_development -> code_review -> completed (terminal). Both stages use
// provider="anthropic".
//
// GetNextStatus call pattern (controller consults it 3 times for a 2-stage run):
//   - call 1 (pre-loop, Run() top)              : in_development, [code_review], non-terminal
//   - call 2 (post stage-1 dispatch)             : in_development, [code_review], non-terminal
//     (handleSpawnAgent picks target=code_review; workflowSvc.IsTerminalStatus("code_review") is false
//     so the loop continues with currentStatus=code_review)
//   - call 3 (post stage-2 dispatch)             : code_review, [completed], non-terminal
//     (handleSpawnAgent picks target=completed; workflowSvc.IsTerminalStatus("completed") is true
//     so the loop terminates with outcome=completed)
func twoStageFixture(t *testing.T, dispatchFunc func(ctx context.Context, input DispatchInput) (*DispatchResult, error)) *RunController {
	t.Helper()

	calls := 0
	transitioner := &MockTransitioner{
		GetNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			calls++
			switch calls {
			case 1, 2:
				return &services.NextStatusInfo{
					CurrentStatus: "in_development",
					IsTerminal:    false,
					AvailableTransitions: []services.TransitionInfoWithAction{
						{TransitionInfo: workflow.TransitionInfo{TargetStatus: "code_review"}},
					},
				}, nil
			case 3:
				return &services.NextStatusInfo{
					CurrentStatus: "code_review",
					IsTerminal:    false,
					AvailableTransitions: []services.TransitionInfoWithAction{
						{TransitionInfo: workflow.TransitionInfo{TargetStatus: "completed"}},
					},
				}, nil
			default:
				return &services.NextStatusInfo{CurrentStatus: "completed", IsTerminal: true}, nil
			}
		},
		TransitionStatusFunc: func(ctx context.Context, key, target string, opts services.TransitionOptions) (*services.TransitionResult, error) {
			return &services.TransitionResult{ToStatus: target}, nil
		},
	}
	actionSvc := &MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "anthropic",
				Instruction: "do work",
			}, nil
		},
	}
	if dispatchFunc == nil {
		dispatchFunc = func(ctx context.Context, input DispatchInput) (*DispatchResult, error) {
			return &DispatchResult{ExitCode: 0, Command: "claude -p x"}, nil
		}
	}
	dispatchers := map[string]AgentDispatcher{
		"anthropic": &MockDispatcher{DispatchFunc: dispatchFunc},
	}
	return makeController(t, transitioner, actionSvc, dispatchers)
}

// Compile-time sentinel that guards against silent changes to the
// DispatchResult.Duration field type. If the field is renamed or its type
// changes away from time.Duration, this assignment fails to compile.
var _ time.Duration = DispatchResult{}.Duration
