package runner

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// BugGateAdmission checks the plan bytes and the declared review baseline.
// Rework and parking remain available without this evidence.
func BugGateAdmission(ctx context.Context, req gatepersist.Request) error {
	if req.EntityType != models.EntityTypeBug || req.Role != gateresult.RoleSuccess {
		return nil
	}
	switch req.Gate {
	case "repair_plan", "plan_review", "development", "code_review", "qa":
	default:
		return nil
	}
	if err := verifyBugEvidenceRoot(ctx, req); err != nil {
		return err
	}
	if _, _, err := readBugEvidence(req, "bug_repair_plan"); err != nil {
		return fmt.Errorf("plan evidence: %w", err)
	}
	_, err := readBugBaseline(req)
	return err
}

func readBugBaseline(req gatepersist.Request) (string, error) {
	ref, err := findBugEvidence(req, "bug_review_baseline")
	if err != nil {
		return "", err
	}
	baseline := ref.Pointer
	if len(baseline) != 40 && len(baseline) != 64 {
		return "", fmt.Errorf("bug_review_baseline must be a full commit hash")
	}
	if _, err := hex.DecodeString(baseline); err != nil {
		return "", fmt.Errorf("invalid bug_review_baseline commit hash")
	}
	return strings.ToLower(baseline), nil
}
