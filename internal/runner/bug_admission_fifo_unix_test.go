//go:build !windows

package runner

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/require"
)

func TestBugPlanAdmissionRejectsFIFOWithoutBlocking(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "docs", "plan"), 0o700))
	require.NoError(t, syscall.Mkfifo(filepath.Join(root, "docs", "plan", "plan.md"), 0o600))
	req := gatepersist.Request{EntityType: models.EntityTypeBug, Gate: "plan_review", Role: gateresult.RoleSuccess,
		RunDir:       filepath.Join(root, ".shark", "runs", "r"),
		EnvelopeJSON: []byte(`{"kind":"final","recommended_outcome":"pass","evidence":[{"kind":"bug_repair_plan","pointer":"docs/plan/plan.md","summary":"sha256:0000000000000000000000000000000000000000000000000000000000000000"}],"gate_result":{"schema_version":1,"summary":"ok"}}`)}
	done := make(chan error, 1)
	go func() { done <- BugGateAdmission(context.Background(), req) }()
	select {
	case err := <-done:
		require.ErrorContains(t, err, "evidence must be a bounded regular file")
	case <-time.After(2 * time.Second):
		t.Error("FIFO blocked admission")
	}
}
