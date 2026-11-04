package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workercontrol"
)

// BugGateAdmission checks the exact plan bytes named by a bug gate's existing
// EvidenceRef. Rework and parking remain available without this evidence.
// This is one part of admission: approval and candidate binding also need
// independently verified evidence before the workflow can be accepted.
func BugGateAdmission(_ context.Context, req gatepersist.Request) error {
	if req.EntityType != models.EntityTypeBug || req.Role != gateresult.RoleSuccess {
		return nil
	}
	switch req.Gate {
	case "repair_plan", "plan_review", "development", "code_review", "qa":
	default:
		return nil
	}
	envelope, err := workercontrol.Decode(req.EnvelopeJSON)
	if err != nil {
		return fmt.Errorf("decode plan evidence: %w", err)
	}
	var found *workercontrol.EvidenceRef
	for i := range envelope.Evidence {
		if envelope.Evidence[i].Kind != "bug_repair_plan" {
			continue
		}
		if found != nil {
			return fmt.Errorf("exactly one bug_repair_plan evidence reference is required")
		}
		found = &envelope.Evidence[i]
	}
	if found == nil {
		return fmt.Errorf("bug_repair_plan evidence is required")
	}
	if !filepath.IsLocal(found.Pointer) || found.Pointer == "." {
		return fmt.Errorf("plan evidence path must be local to project")
	}
	const prefix = "sha256:"
	if !strings.HasPrefix(found.Summary, prefix) {
		return fmt.Errorf("plan evidence summary must contain sha256 digest")
	}
	expected, err := hex.DecodeString(strings.TrimPrefix(found.Summary, prefix))
	if err != nil || len(expected) != sha256.Size {
		return fmt.Errorf("invalid plan evidence digest")
	}
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(req.RunDir)))
	root, err := os.OpenRoot(projectRoot)
	if err != nil {
		return fmt.Errorf("open project root: %w", err)
	}
	defer func() { _ = root.Close() }()
	file, err := root.OpenFile(found.Pointer, os.O_RDONLY|bugPlanOpenFlags, 0)
	if err != nil {
		return fmt.Errorf("open plan evidence: %w", err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 2*1024*1024 {
		return fmt.Errorf("plan evidence must be a bounded regular file")
	}
	h := sha256.New()
	count, err := io.Copy(h, io.LimitReader(file, 2*1024*1024+1))
	if err != nil {
		return fmt.Errorf("read plan evidence: %w", err)
	}
	if count > 2*1024*1024 {
		return fmt.Errorf("plan evidence exceeds size limit")
	}
	if !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), hex.EncodeToString(expected)) {
		return fmt.Errorf("plan evidence digest is stale")
	}
	return nil
}
