//go:build !windows

package runner

import (
	"context"
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
	require.NoError(t, syscall.Mkfifo(filepath.Join(root, "plan"), 0o600))
	req := gatepersist.Request{EntityType: models.EntityTypeBug, Gate: "plan_review", Role: gateresult.RoleSuccess,
		RunDir:       filepath.Join(root, ".shark", "runs", "r"),
		EnvelopeJSON: []byte(`{"kind":"final","recommended_outcome":"pass","evidence":[{"kind":"bug_repair_plan","pointer":"plan","summary":"sha256:0000000000000000000000000000000000000000000000000000000000000000"}],"gate_result":{"schema_version":1,"summary":"ok"}}`)}
	done := make(chan error, 1)
	go func() { done <- BugGateAdmission(context.Background(), req) }()
	select {
	case err := <-done:
		require.Error(t, err)
	case <-time.After(2 * time.Second):
		t.Error("FIFO blocked admission")
	}
}
