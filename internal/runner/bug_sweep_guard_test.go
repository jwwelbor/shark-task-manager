package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/require"
)

// B03: a completed sweep is admitted only when its guard's retained executions
// show the restored defect flagged at the swept site and the corrected
// candidate passing. Each trial varies one fact of the otherwise valid pair.
func TestBugSweepGuardPairIsResolvedBeforeWrite(t *testing.T) {
	root := t.TempDir()
	runDir := filepath.Join(root, ".shark", "runs", "review")
	require.NoError(t, os.MkdirAll(runDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "guard_test.go"), []byte("package x"), 0o600))
	valid := func() map[string]any {
		return map[string]any{
			"guard_pointer": "guard_test.go", "class_key": "K", "site": "internal/foo.go:10", "candidate": "cand", "enabled": true,
			"restored":  map[string]any{"exit_code": 1, "flagged_site": "internal/foo.go:10", "diagnostic": "guard: internal/foo.go:10 restored defect"},
			"corrected": map[string]any{"exit_code": 0},
		}
	}
	request := func(record any, status string) gatepersist.Request {
		require.NoError(t, os.WriteFile(filepath.Join(root, "run.json"), mustJSON(t, record), 0o600))
		return gatepersist.Request{RunDir: runDir, EntityType: models.EntityTypeBug, Result: &gateresult.GateResult{RemediationSweeps: []gateresult.DefectClassSweep{{
			ClassKey: "K", Status: status, MatchingCount: 1,
			Instances: []gateresult.SweepInstance{{Fingerprint: "f", SitePointer: "internal/foo.go:10", Disposition: "fixed"}},
			Guard:     gateresult.SweepGuard{Kind: "test", ImplementationPointer: "guard_test.go", CounterfactualPointer: "run.json", Status: "verified"},
		}}}}
	}
	require.NoError(t, verifyBugSweepGuards(request(valid(), "complete")))
	for name, mutate := range map[string]func(map[string]any){
		"disabled guard":        func(m map[string]any) { m["enabled"] = false },
		"passes on both":        func(m map[string]any) { m["restored"].(map[string]any)["exit_code"] = 0 },
		"flags both":            func(m map[string]any) { m["corrected"].(map[string]any)["exit_code"] = 1 },
		"unrelated failure":     func(m map[string]any) { m["restored"].(map[string]any)["diagnostic"] = "assertion failed earlier" },
		"different site":        func(m map[string]any) { m["restored"].(map[string]any)["flagged_site"] = "internal/other.go:3" },
		"different guard":       func(m map[string]any) { m["guard_pointer"] = "other_test.go" },
		"different class":       func(m map[string]any) { m["class_key"] = "OTHER" },
		"missing restored":      func(m map[string]any) { delete(m, "restored") },
		"missing corrected":     func(m map[string]any) { delete(m, "corrected") },
		"unswept site":          func(m map[string]any) { m["site"] = "internal/else.go:1" },
		"unnamed candidate":     func(m map[string]any) { m["candidate"] = "" },
		"unknown field smuggle": func(m map[string]any) { m["verified"] = true },
	} {
		t.Run(name, func(t *testing.T) {
			record := valid()
			mutate(record)
			require.ErrorContains(t, verifyBugSweepGuards(request(record, "complete")), "unsupported closure")
		})
	}
	t.Run("absent execution record", func(t *testing.T) {
		req := request(valid(), "complete")
		req.Result.RemediationSweeps[0].Guard.CounterfactualPointer = "missing.json"
		require.ErrorContains(t, verifyBugSweepGuards(req), "unreadable")
	})
	t.Run("open sweep needs no executed guard", func(t *testing.T) {
		require.NoError(t, verifyBugSweepGuards(request(map[string]any{}, "open")))
	})
	t.Run("other entity types are untouched", func(t *testing.T) {
		req := request(map[string]any{}, "complete")
		req.EntityType = models.EntityTypeTask
		require.NoError(t, verifyBugSweepGuards(req))
	})
}
