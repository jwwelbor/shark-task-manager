package runner

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/stretchr/testify/require"
)

func TestBugPlanMustBeCommittedAndNotIgnored(t *testing.T) {
	root := t.TempDir()
	path := "docs/plan/plan.md"
	plan := []byte("reviewed plan")
	commitTestBugPlan(t, root, path, plan)
	hash := sha256.Sum256(plan)
	req := gatepersist.Request{RunDir: filepath.Join(root, ".shark", "runs", "test"), EnvelopeJSON: []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[{"kind":"bug_repair_plan","pointer":"%s","summary":"sha256:%x"}]}`, path, hash))}
	_, _, _, err := committedBugPlan(context.Background(), req)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, path), []byte("unapproved edit"), 0600))
	_, _, _, err = committedBugPlan(context.Background(), req)
	require.Error(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, path), plan, 0600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".git", "info", "exclude"), []byte("docs/plan/plan.md\n"), 0600))
	_, _, _, err = committedBugPlan(context.Background(), req)
	require.ErrorContains(t, err, "ignored")
	untracked := "docs/plan/untracked.md"
	require.NoError(t, os.WriteFile(filepath.Join(root, untracked), plan, 0600))
	req.EnvelopeJSON = []byte(fmt.Sprintf(`{"kind":"final","recommended_outcome":"pass","evidence":[{"kind":"bug_repair_plan","pointer":"%s","summary":"sha256:%x"}]}`, untracked, hash))
	_, _, _, err = committedBugPlan(context.Background(), req)
	require.ErrorContains(t, err, "not tracked")
}
