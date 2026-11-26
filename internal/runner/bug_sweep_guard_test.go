package runner

import (
	"context"
	"fmt"
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
	ctx := context.Background()
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
	require.NoError(t, verifyBugSweepGuards(ctx, request(valid(), "complete"), nil))
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
			require.ErrorContains(t, verifyBugSweepGuards(ctx, request(record, "complete"), nil), "unsupported closure")
		})
	}
	t.Run("absent execution record", func(t *testing.T) {
		req := request(valid(), "complete")
		req.Result.RemediationSweeps[0].Guard.CounterfactualPointer = "missing.json"
		require.ErrorContains(t, verifyBugSweepGuards(ctx, req, nil), "unreadable")
	})
	t.Run("open sweep link is resolved against the work item", func(t *testing.T) {
		link := func(mutate func(map[string]any)) gatepersist.Request {
			doc := map[string]any{"kind": "open_sweep_link", "class_key": "K", "work_item_key": "TD-001", "reason": "no feasible executable guard"}
			mutate(doc)
			return request(doc, "open")
		}
		items := func(known map[string]string) BugWorkItemResolver {
			return func(_ context.Context, key string) (models.Entity, error) {
				title, ok := known[key]
				if !ok {
					return nil, fmt.Errorf("not found")
				}
				return &models.TechDebt{BaseEntity: models.BaseEntity{Key: key, Title: title}}, nil
			}
		}
		covering := items(map[string]string{"TD-001": "Guard for class K", "TD-002": "Unrelated work"})
		require.NoError(t, verifyBugSweepGuards(ctx, link(func(map[string]any) {}), covering))
		for name, c := range map[string]struct {
			mutate   func(map[string]any)
			resolver BugWorkItemResolver
			want     string
		}{
			"link removed":      {func(m map[string]any) { delete(m, "work_item_key") }, covering, "must name its class"},
			"unresolvable item": {func(m map[string]any) { m["work_item_key"] = "TD-404" }, covering, "does not resolve"},
			"unrelated item":    {func(m map[string]any) { m["work_item_key"] = "TD-002" }, covering, "does not cover class"},
			"reason removed":    {func(m map[string]any) { m["reason"] = " " }, covering, "must name its class"},
			"other class":       {func(m map[string]any) { m["class_key"] = "OTHER" }, covering, "must name its class"},
			"no resolver":       {func(map[string]any) {}, nil, "no resolver"},
		} {
			t.Run(name, func(t *testing.T) {
				require.ErrorContains(t, verifyBugSweepGuards(ctx, link(c.mutate), c.resolver), c.want)
			})
		}
	})
	t.Run("rework with open instances needs no link", func(t *testing.T) {
		req := request(map[string]any{}, "open")
		req.Result.RemediationSweeps[0].OpenCount = 1
		req.Result.RemediationSweeps[0].Guard.CounterfactualPointer = "absent.json"
		require.NoError(t, verifyBugSweepGuards(ctx, req, nil))
	})
	t.Run("open sweep with no open instances and no link is rejected", func(t *testing.T) {
		req := request(map[string]any{}, "open")
		req.Result.RemediationSweeps[0].OpenCount = 0
		req.Result.RemediationSweeps[0].Guard.CounterfactualPointer = "absent.json"
		require.ErrorContains(t, verifyBugSweepGuards(ctx, req, nil), "unsupported closure")
	})
	t.Run("other entity types are untouched", func(t *testing.T) {
		req := request(map[string]any{}, "complete")
		req.EntityType = models.EntityTypeTask
		require.NoError(t, verifyBugSweepGuards(ctx, req, nil))
	})
}
