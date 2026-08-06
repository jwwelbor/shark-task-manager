// Package contracts — I-03 / X-08 contract tests for E40-F04's run-liveness
// recorder (spec.md D7, shape source architecture.md#run-liveness-contract).
//
// Both fixed-name tests below are CI-gated and environment-free per D7: no
// spawned agent, no scratch project, no database.
//
// TestTC001_I03LivenessContract drives runner.RunController.Run in process
// with the stub transitioner/action-service/dispatcher fixtures, matching
// D7's exact mechanism. D7/test-plan.md ask this file to reuse
// controller_test.go's MockTransitioner/MockActionService/MockDispatcher/
// MockCascadeChildrenService "verbatim". That instruction turns out to be
// unsatisfiable as written: those types are declared in
// internal/runner/controller_test.go, a _test.go file, which Go does not
// compile into the importable "runner" package — package contracts (a
// different package entirely) cannot see them regardless of exported name.
// This file instead:
//   - reuses config.MockActionService (= action.MockActionService), which
//     genuinely is exported from internal/config/action/mock_service.go and
//     is the same shape controller_test.go's copy hand-mirrors, and
//   - declares minimal local stub types for runner.EntityTransitioner and
//     runner.AgentDispatcher, the two seams with no exported mock anywhere
//     in the module, implementing only what TC-001's single-stage flow
//     exercises.
//
// TestTC002_X08StdoutPurity covers the source-guard half only (go/parser
// over run.go's AST) plus the AC-03 confirmation. The runtime half — which
// test-plan.md's Caller-Path Contract names as calling the unexported
// outputRunResult(result) directly — cannot be written in this package for
// the same Go-visibility reason and instead lives in
// internal/cli/commands/run_stdout_purity_test.go as
// TestTC002_X08StdoutPurityRuntimeHalf. This split is intentional: spec.md's
// own rationale for putting TC-002 in tests/contracts/ is specifically the
// source guard's environment-free, cross-epic-inheritable nature
// (spec.md:429-435) — the runtime half was never the part that needed to
// live here.
//
// Naming/placement otherwise follows e40_i01_corpus_contract_test.go:
// package contracts, TestTC00N_... names, real files, no DB.
package contracts

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// ---------------------------------------------------------------------------
// TC-001: I-03 producer shape + per-event durability
// ---------------------------------------------------------------------------

// e40I03StubTransitioner is a minimal local runner.EntityTransitioner stub.
// TC-001's flow never calls TransitionStatus (the spawn_agent stage's
// post-dispatch GetNextStatus reports zero AvailableTransitions, so the
// controller records an implicit transition itself — see controller.go's
// handleSpawnAgent), so that method is intentionally unimplemented.
type e40I03StubTransitioner struct {
	getNextStatusFunc func(ctx context.Context, key string) (*services.NextStatusInfo, error)
}

func (s *e40I03StubTransitioner) GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error) {
	return s.getNextStatusFunc(ctx, key)
}

func (s *e40I03StubTransitioner) TransitionStatus(ctx context.Context, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	return nil, fmt.Errorf("e40I03StubTransitioner.TransitionStatus: not implemented (TC-001's flow never calls it)")
}

var _ runner.EntityTransitioner = (*e40I03StubTransitioner)(nil)

// e40I03StubDispatcher is a minimal local runner.AgentDispatcher stub whose
// Dispatch delegates to a test-supplied func — used to block the controller
// mid-stage (test-plan.md TC-001 precondition: "configured for one entity
// that opens a dispatch stage and does not return before the test reads
// run.log").
type e40I03StubDispatcher struct {
	dispatchFunc func(ctx context.Context, input runner.DispatchInput) (*runner.DispatchResult, error)
}

func (d *e40I03StubDispatcher) Dispatch(ctx context.Context, input runner.DispatchInput) (*runner.DispatchResult, error) {
	return d.dispatchFunc(ctx, input)
}

func (d *e40I03StubDispatcher) Name() string { return "e40-i03-stub" }

func (d *e40I03StubDispatcher) BuildCommand(input runner.DispatchInput) (string, error) {
	return "e40-i03-stub-cmd", nil
}

var _ runner.AgentDispatcher = (*e40I03StubDispatcher)(nil)

// e40I03NDJSONRequiredKeys are D3's "always present" NDJSON fields (D3's
// table: ts, run_id, event, entity_key, iteration, status, action,
// stage_elapsed_ms, total_elapsed_ms are always present; agent_type/provider
// are omitted, not empty-valued, when unset — not asserted as "required"
// here for that reason).
var e40I03NDJSONRequiredKeys = []string{
	"ts", "run_id", "event", "entity_key", "iteration", "status", "action",
	"stage_elapsed_ms", "total_elapsed_ms",
}

// TestTC001_I03LivenessContract is I-03's shared-contract evidence, reused
// verbatim by E40-F02 per this task's spec (T-E40-F04-006.md). It drives
// RunController.Run in process (D7's exact mechanism) with the recorder
// wired via RunOptions.Progress exactly as run.go wires it, captures stderr
// through an os.Pipe swap, and — critically — reads run.log from disk while
// the dispatch stage is still open and BEFORE Finish() is ever called, which
// is what proves REQ-F-010's per-event durability rather than
// flush-at-end.
//
// Design decision on the heartbeat clause: spec.md D7's narrative describes
// TC-001 as asserting "the corresponding stage lines" without naming
// heartbeat explicitly, and the sibling unit test
// (internal/runner/liveness_test.go's TestLiveness_TC019_ReadBeforeCloseDurability)
// asserts durability using only stage_start. But this task's own AC-T1 and
// spec.md's AC-07 both explicitly say "stage_start and latest heartbeat", so
// this test follows that more specific, task-proximate wording (Rule 7):
// it lets the recorder's real fixed-10s ticker (Start()) run and blocks the
// stub dispatcher until a real heartbeat line lands in run.log. That costs
// ~10-15s of wall clock — a deliberate, documented trade against a faster
// but AC-incomplete test.
func TestTC001_I03LivenessContract(t *testing.T) {
	root := t.TempDir()
	runID := uuid.New().String()
	entityKey := "T-E40-F04-006"
	start := time.Now()

	// Capture stderr through an os.Pipe swap (D7's mechanism) via a
	// background reader so we can inspect partial output while the run is
	// still mid-flight (needed for the heartbeat poll below).
	origStderr := os.Stderr
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	var stderrMu sync.Mutex
	var stderrBuf strings.Builder
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		buf := make([]byte, 4096)
		for {
			n, rerr := pr.Read(buf)
			if n > 0 {
				stderrMu.Lock()
				stderrBuf.Write(buf[:n])
				stderrMu.Unlock()
			}
			if rerr != nil {
				return
			}
		}
	}()
	os.Stderr = pw

	restoreStderr := func() {
		os.Stderr = origStderr
		_ = pw.Close()
		<-readerDone
		_ = pr.Close()
	}
	// Belt-and-braces: if a t.Fatalf below exits the goroutine early, still
	// put stderr back for any later test in the same package binary.
	defer func() {
		if os.Stderr == pw {
			restoreStderr()
		}
	}()

	rec := runner.NewLivenessRecorder(root, runID, entityKey, true /* jsonMode */, start)
	rec.Start()
	// Stop() is idempotent (stopOnce-guarded) so this is safe alongside the
	// explicit rec.Stop() call below on the success path — it exists so a
	// t.Fatal between Start() and that call doesn't leak the ticker
	// goroutine into the rest of the package's test binary.
	t.Cleanup(rec.Stop)

	dispatchEntered := make(chan struct{})
	proceed := make(chan struct{})
	dispatcher := &e40I03StubDispatcher{
		dispatchFunc: func(ctx context.Context, input runner.DispatchInput) (*runner.DispatchResult, error) {
			close(dispatchEntered)
			<-proceed
			return &runner.DispatchResult{ExitCode: 0, Duration: time.Millisecond}, nil
		},
	}

	calls := 0
	transitioner := &e40I03StubTransitioner{
		getNextStatusFunc: func(ctx context.Context, key string) (*services.NextStatusInfo, error) {
			calls++
			if calls == 1 {
				// Pre-loop status read: not terminal, so the loop proceeds
				// to the spawn_agent action.
				return &services.NextStatusInfo{CurrentStatus: "in_development"}, nil
			}
			// Post-dispatch status read (handleSpawnAgent): zero available
			// transitions with a changed status ends the run as "completed"
			// without a second iteration/second dispatch — keeping this a
			// deliberate single-stage flow.
			return &services.NextStatusInfo{CurrentStatus: "completed"}, nil
		},
	}

	actionSvc := &config.MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*config.PopulatedAction, error) {
			return &config.PopulatedAction{
				Action:      config.ActionSpawnAgent,
				AgentType:   "developer",
				Provider:    "anthropic",
				Instruction: "do the work",
			}, nil
		},
	}

	controller, err := runner.NewRunController(runner.RunControllerDeps{
		Transitioner: transitioner,
		ActionSvc:    actionSvc,
		WorkflowSvc:  workflow.NewService(""),
		Dispatchers:  map[string]runner.AgentDispatcher{"anthropic": dispatcher},
	})
	if err != nil {
		t.Fatalf("NewRunController: %v", err)
	}

	type runOutcome struct {
		result *runner.RunResult
		err    error
	}
	runDone := make(chan runOutcome, 1)
	go func() {
		result, runErr := controller.Run(context.Background(), entityKey, runner.RunOptions{
			Progress:    rec.Observe,
			ProjectRoot: root,
			RunID:       runID,
		})
		runDone <- runOutcome{result, runErr}
	}()

	select {
	case <-dispatchEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the controller to open the dispatch stage")
	}

	// AC-T1/AC-07: run.log is durable while the stage is still open and
	// Finish() has not been called anywhere yet. Line-scoped (not
	// whole-file Contains) so this actually pins the fields to the
	// stage_start line itself, rather than passing vacuously because some
	// other line in the file happens to contain the same tokens.
	logPath := rec.LogPath()
	stageStartLine := e40I03LastLineContaining(e40I03MustReadFile(t, logPath), "  stage_start  ")
	if stageStartLine == "" {
		t.Fatalf("run.log (pre-close) has no stage_start line; got:\n%s", e40I03MustReadFile(t, logPath))
	}
	for _, want := range []string{entityKey, "status=in_development", "agent=developer", "provider=anthropic"} {
		if !strings.Contains(stageStartLine, want) {
			t.Errorf("run.log stage_start line missing %q: %q", want, stageStartLine)
		}
	}

	// Poll (not sleep-then-read, to avoid flake) for a real heartbeat line —
	// see the heartbeat design-decision note in the function doc comment.
	deadline := time.Now().Add(30 * time.Second)
	var heartbeatLine string
	for time.Now().Before(deadline) {
		heartbeatLine = e40I03LastLineContaining(e40I03MustReadFile(t, logPath), "  heartbeat  ")
		if heartbeatLine != "" {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if heartbeatLine == "" {
		t.Fatal("run.log did not contain a heartbeat line for the open stage within 30s")
	}
	// Same line-scoping rationale as the stage_start check above: this must
	// pin the fields to the heartbeat line, since the earlier stage_start
	// line already carries the same entity/status/agent/provider tokens and
	// would make a whole-file Contains check pass regardless of what the
	// heartbeat renderer actually emits.
	for _, want := range []string{entityKey, "status=in_development", "agent=developer", "provider=anthropic"} {
		if !strings.Contains(heartbeatLine, want) {
			t.Errorf("run.log heartbeat line missing %q: %q", want, heartbeatLine)
		}
	}

	// Unblock the dispatch and let the run finish.
	close(proceed)

	var outcome runOutcome
	select {
	case outcome = <-runDone:
	case <-time.After(10 * time.Second):
		t.Fatal("controller.Run did not return after the dispatch was unblocked")
	}
	if outcome.err != nil {
		t.Fatalf("controller.Run() error = %v", outcome.err)
	}

	// Mirror run.go's teardown order (D6 edit 2): Stop() before Finish().
	rec.Stop()
	rec.Finish(outcome.result)

	restoreStderr()

	finalLog := e40I03MustReadFile(t, logPath)
	if !strings.Contains(finalLog, "stage_end") {
		t.Errorf("run.log missing stage_end after Finish(); got:\n%s", finalLog)
	}

	stderrMu.Lock()
	stderrOut := stderrBuf.String()
	stderrMu.Unlock()

	sawEvents := map[string]bool{}
	for _, line := range strings.Split(strings.TrimRight(stderrOut, "\n"), "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "run.log: ") {
			// D3's consumer rule: Start()'s one-time path announcement is
			// plain text, not NDJSON, and a consumer must skip it.
			continue
		}
		var fields map[string]json.RawMessage
		if jerr := json.Unmarshal([]byte(line), &fields); jerr != nil {
			t.Fatalf("stderr line failed to parse as JSON (I-03 consumer-rule violation for a non-plain-text line): %q: %v", line, jerr)
		}
		for _, key := range e40I03NDJSONRequiredKeys {
			if _, ok := fields[key]; !ok {
				t.Errorf("NDJSON line missing required key %q: %s", key, line)
			}
		}
		if _, ok := fields["phase"]; ok {
			t.Errorf("NDJSON line unexpectedly carries a phase key (D3: phase is deliberately absent): %s", line)
		}
		var event string
		if raw, ok := fields["event"]; ok {
			_ = json.Unmarshal(raw, &event)
		}
		switch event {
		case "stage_start", "heartbeat", "stage_end":
			sawEvents[event] = true
		default:
			t.Errorf("NDJSON line has event %q, want one of stage_start/heartbeat/stage_end: %s", event, line)
		}
	}
	for _, want := range []string{"stage_start", "heartbeat", "stage_end"} {
		if !sawEvents[want] {
			t.Errorf("stderr never emitted a %q event across the whole run", want)
		}
	}
}

// e40I03MustReadFile reads path and fails the test on error. Helper name is
// prefixed e40I03 (distinct from e40_i01_corpus_contract_test.go's e40*
// helpers) since both files share package contracts.
func e40I03MustReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// e40I03LastLineContaining returns the last line of content containing
// substr, or "" if none does. "Last" matters for the heartbeat check: run.log
// can accumulate more than one heartbeat line while the test polls, and
// AC-T1/AC-07 ask for the *latest* heartbeat. Line-scoping (vs. a whole-file
// strings.Contains) is what makes these field checks non-vacuous: without
// it, an earlier stage_start line's entity/status/agent/provider tokens
// would satisfy a heartbeat-field assertion even if the heartbeat renderer
// dropped those fields entirely.
func e40I03LastLineContaining(content, substr string) string {
	var last string
	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(line, substr) {
			last = line
		}
	}
	return last
}

// ---------------------------------------------------------------------------
// TC-002: X-08 stdout purity — source-guard half (+ AC-03 confirmation)
// ---------------------------------------------------------------------------

// e40I03SyntheticGoodSource is a well-formed fixture (every stdout-writer
// call site inside outputRunResult) used to prove the guard below doesn't
// flag a clean file.
const e40I03SyntheticGoodSource = `package commands

import (
	"fmt"
	"os"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
)

func runRun() error {
	fmt.Fprintf(os.Stderr, "warning: something failed\n")
	return outputRunResult(nil)
}

func outputRunResult(result interface{}) error {
	fmt.Printf("Run complete\n")
	fmt.Println("done")
	cli.Success("ok")
	os.Stdout.Write([]byte("bytes\n"))
	return nil
}
`

// e40I03SyntheticBadSource adds a cli.Warning(...) call inside runRun,
// outside outputRunResult — test-plan.md TC-002's stated negative case
// ("A cli.Warning(...) call added anywhere in run.go outside
// outputRunResult fails the guard").
const e40I03SyntheticBadSource = `package commands

import (
	"fmt"
	"os"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
)

func runRun() error {
	fmt.Fprintf(os.Stderr, "warning: something failed\n")
	cli.Warning("leaked warning")
	return outputRunResult(nil)
}

func outputRunResult(result interface{}) error {
	fmt.Printf("Run complete\n")
	return nil
}
`

// TestTC002_X08StdoutPurity is X-08's durable, cross-epic-inheritable
// artifact (spec.md:421-435): the go/parser source guard over
// internal/cli/commands/run.go, widened per test-plan.md beyond the literal
// D7 wording to also flag fmt.Fprint* targeting os.Stdout, os.Stdout.Write,
// and cli.Success/Warning/Info/OutputTable/Title — not just bare
// fmt.Print/Printf/Println. The runtime half of TC-002 (AC-02's first
// clause) lives in internal/cli/commands/run_stdout_purity_test.go as
// TestTC002_X08StdoutPurityRuntimeHalf; see this file's package doc comment
// for why.
func TestTC002_X08StdoutPurity(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	runGoPath := filepath.Join(repoRoot, "internal", "cli", "commands", "run.go")

	t.Run("every_stdout_write_site_is_inside_outputRunResult", func(t *testing.T) {
		violations := e40I03FindStdoutWriteViolations(t, runGoPath, "outputRunResult")
		if len(violations) != 0 {
			t.Fatalf("run.go has stdout-writer call sites outside outputRunResult (X-08 violation):\n%s", strings.Join(violations, "\n"))
		}
	})

	// AC-T4: the source guard also confirms AC-03 — the --worktree cleanup
	// defer's warning specifically targets os.Stderr via fmt.Fprintf. This
	// is a positive confirmation, not merely "not flagged by the guard
	// above" (a missing warning call entirely would also pass that
	// vacuously).
	t.Run("worktree_cleanup_warning_targets_stderr", func(t *testing.T) {
		e40I03AssertWorktreeCleanupWarningTargetsStderr(t, runGoPath)
	})

	t.Run("well_formed_fixture_has_no_violations", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "synthetic_run.go")
		if werr := os.WriteFile(path, []byte(e40I03SyntheticGoodSource), 0o644); werr != nil {
			t.Fatalf("write synthetic fixture: %v", werr)
		}
		violations := e40I03FindStdoutWriteViolations(t, path, "outputRunResult")
		if len(violations) != 0 {
			t.Fatalf("expected no violations for a well-formed fixture, got: %v", violations)
		}
	})

	// Negative case (test-plan.md): a cli.Warning(...) call added outside
	// outputRunResult must be flagged — proving the guard actually bites
	// rather than passing vacuously.
	t.Run("negative_case_cli_Warning_outside_outputRunResult_is_flagged", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "synthetic_run.go")
		if werr := os.WriteFile(path, []byte(e40I03SyntheticBadSource), 0o644); werr != nil {
			t.Fatalf("write synthetic fixture: %v", werr)
		}
		violations := e40I03FindStdoutWriteViolations(t, path, "outputRunResult")
		if !e40I03ContainsSubstring(violations, "cli.Warning") {
			t.Fatalf("expected the guard to flag cli.Warning outside outputRunResult, got: %v", violations)
		}
	})
}

// e40I03FindStdoutWriteViolations parses the Go source file at path and
// returns one description per call site matching TC-002's widened
// stdout-writer class (test-plan.md AC-02: fmt.Print/Printf/Println;
// fmt.Fprint/Fprintf/Fprintln targeting os.Stdout; os.Stdout.Write;
// cli.Success/Warning/Info/OutputTable/Title) that lies OUTSIDE the named
// safe function's body. An empty slice means the file is X-08-clean.
func e40I03FindStdoutWriteViolations(t *testing.T, path, safeFuncName string) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	var safeFunc *ast.FuncDecl
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == safeFuncName {
			safeFunc = fn
			break
		}
	}
	if safeFunc == nil {
		t.Fatalf("%s does not declare a %s function", path, safeFuncName)
	}
	within := func(pos token.Pos) bool {
		return pos >= safeFunc.Pos() && pos <= safeFunc.End()
	}

	var violations []string
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		kind, matched := e40I03ClassifyStdoutCall(call)
		if !matched || within(call.Pos()) {
			return true
		}
		violations = append(violations, fmt.Sprintf("%s at %s", kind, fset.Position(call.Pos())))
		return true
	})
	return violations
}

// e40I03ClassifyStdoutCall reports whether call matches TC-002's widened
// stdout-writer class, and a short label for diagnostics.
func e40I03ClassifyStdoutCall(call *ast.CallExpr) (string, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}

	// os.Stdout.Write(...)
	if sel.Sel.Name == "Write" {
		if inner, ok := sel.X.(*ast.SelectorExpr); ok {
			if pkg, ok := inner.X.(*ast.Ident); ok && pkg.Name == "os" && inner.Sel.Name == "Stdout" {
				return "os.Stdout.Write", true
			}
		}
		return "", false
	}

	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return "", false
	}

	switch pkgIdent.Name {
	case "fmt":
		switch sel.Sel.Name {
		case "Print", "Printf", "Println":
			return "fmt." + sel.Sel.Name, true
		case "Fprint", "Fprintf", "Fprintln":
			if len(call.Args) == 0 {
				return "", false
			}
			target, ok := call.Args[0].(*ast.SelectorExpr)
			if !ok {
				return "", false
			}
			if targetPkg, ok := target.X.(*ast.Ident); ok && targetPkg.Name == "os" && target.Sel.Name == "Stdout" {
				return "fmt." + sel.Sel.Name + "(os.Stdout, ...)", true
			}
			return "", false
		}
	case "cli":
		switch sel.Sel.Name {
		case "Success", "Warning", "Info", "OutputTable", "Title":
			return "cli." + sel.Sel.Name, true
		}
	}
	return "", false
}

func e40I03ContainsSubstring(items []string, substr string) bool {
	for _, item := range items {
		if strings.Contains(item, substr) {
			return true
		}
	}
	return false
}

// e40I03AssertWorktreeCleanupWarningTargetsStderr locates the --worktree
// cleanup defer's if-block inside runRun and asserts its warning call is
// fmt.Fprintf(os.Stderr, ...). This mirrors
// internal/cli/commands/run_worktree_test.go's
// TestRunWorktreeCleanupWarningTargetsStderr (TC-013) — duplicated rather
// than imported because that test's helpers are unexported and package
// commands is not importable from package contracts. AC-T4 asks this
// file's guard to confirm AC-03 directly, so the duplication is deliberate,
// not an oversight.
func e40I03AssertWorktreeCleanupWarningTargetsStderr(t *testing.T, runGoPath string) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, runGoPath, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", runGoPath, err)
	}

	var runRunDecl *ast.FuncDecl
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "runRun" {
			runRunDecl = fn
			break
		}
	}
	if runRunDecl == nil {
		t.Fatalf("%s does not declare a runRun function", runGoPath)
	}

	var cleanupIf *ast.IfStmt
	ast.Inspect(runRunDecl, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok || ifStmt.Init == nil {
			return true
		}
		assign, ok := ifStmt.Init.(*ast.AssignStmt)
		if !ok || len(assign.Rhs) != 1 {
			return true
		}
		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if ok && sel.Sel.Name == "RemoveWorktree" {
			cleanupIf = ifStmt
		}
		return true
	})
	if cleanupIf == nil {
		t.Fatal("could not find the --worktree cleanup if-block inside runRun")
	}

	var warnCall *ast.CallExpr
	var warnSel *ast.SelectorExpr
	ast.Inspect(cleanupIf.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "fmt" {
			return true
		}
		if sel.Sel.Name == "Printf" || sel.Sel.Name == "Fprintf" {
			warnCall, warnSel = call, sel
		}
		return true
	})
	if warnCall == nil {
		t.Fatal("worktree cleanup if-block contains no fmt.Printf/Fprintf call")
	}
	if warnSel.Sel.Name != "Fprintf" {
		t.Fatalf("worktree cleanup warning call is fmt.%s(...), want fmt.Fprintf(os.Stderr, ...)", warnSel.Sel.Name)
	}
	if len(warnCall.Args) == 0 {
		t.Fatal("fmt.Fprintf call has no arguments")
	}
	firstArg, ok := warnCall.Args[0].(*ast.SelectorExpr)
	if !ok {
		t.Fatalf("fmt.Fprintf's first argument is %T, want os.Stderr", warnCall.Args[0])
	}
	pkgIdent, ok := firstArg.X.(*ast.Ident)
	if !ok || pkgIdent.Name != "os" || firstArg.Sel.Name != "Stderr" {
		t.Fatal("fmt.Fprintf's first argument is not os.Stderr")
	}
}
