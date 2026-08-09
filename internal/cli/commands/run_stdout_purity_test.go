// Package commands — TC-002 (X-08 stdout purity) runtime half.
//
// TC-002's fixed Go name (TestTC002_X08StdoutPurity) and its source-guard
// half live in tests/contracts/e40_i03_liveness_contract_test.go, per
// spec.md D7/AC-02 and test-plan.md's Caller-Path Contract row for TC-002.
// That row names outputRunResult(result) as the runtime entrypoint, but
// outputRunResult is unexported (package commands) while tests/contracts is
// package contracts — Go visibility makes calling it from there impossible
// without exporting a wrapper, and T-E40-F04-006's scope forbids editing
// internal/cli/commands/run.go. This file is the necessary split: it
// exercises the real outputRunResult call at its real package boundary,
// while tests/contracts keeps the half that is genuinely environment-free
// and cross-package inheritable — the go/parser source guard (spec.md's
// "It lives in tests/contracts/ ... precisely so a cross-epic consumer can
// find and depend on it", spec.md:429-435).
package commands

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
)

// TestTC002_X08StdoutPurityRuntimeHalf is TC-002's runtime half (test-plan.md
// AC-T2 / spec.md AC-02's first clause): with --json set, outputRunResult's
// complete stdout byte stream decodes as exactly one runner.RunResult, and a
// second Decode returns io.EOF — no trailing bytes, no interleaved text.
func TestTC002_X08StdoutPurityRuntimeHalf(t *testing.T) {
	origJSON := cli.GlobalConfig.JSON
	origField := cli.GlobalConfig.Field
	cli.GlobalConfig.JSON = true
	cli.GlobalConfig.Field = ""
	defer func() {
		cli.GlobalConfig.JSON = origJSON
		cli.GlobalConfig.Field = origField
	}()

	result := &runner.RunResult{
		EntityKey:       "T-E40-F04-006",
		FinalStatus:     "completed",
		StagesCompleted: 1,
		Outcome:         "completed",
		Stages: []runner.StageLog{{
			Status:    "in_development",
			Action:    "spawn_agent",
			AgentType: "developer",
			Provider:  "anthropic",
		}},
	}

	var outErr error
	captured := capturingOutput(func() {
		outErr = outputRunResult(result)
	})
	if outErr != nil {
		t.Fatalf("outputRunResult() error = %v", outErr)
	}

	dec := json.NewDecoder(strings.NewReader(captured))
	var got runner.RunResult
	if err := dec.Decode(&got); err != nil {
		t.Fatalf("decode captured stdout as RunResult: %v\ncaptured: %q", err, captured)
	}
	if got.EntityKey != result.EntityKey || got.Outcome != result.Outcome {
		t.Errorf("decoded RunResult = %+v, want EntityKey=%q Outcome=%q", got, result.EntityKey, result.Outcome)
	}

	var extra json.RawMessage
	if err := dec.Decode(&extra); err != io.EOF {
		t.Fatalf("second Decode() = (%v, err=%v), want io.EOF — stdout must carry exactly one document", extra, err)
	}
}
