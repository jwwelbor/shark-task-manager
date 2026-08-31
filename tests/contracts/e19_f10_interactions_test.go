// Package contracts checks that E19-F10 keeps roadmap admission at the shared
// sprint-consumer boundary used by both planning and role-aware pull.
package contracts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTC001_E19F10SharedAdmissionConsumersRemainCentralized(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	servicePath := filepath.Join(root, "..", "..", "internal", "services", "sprint_service.go")
	content, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("read sprint service: %v", err)
	}
	source := string(content)
	for _, want := range []string{
		"func (s *SprintService) AddEntityToSprint", "s.admissionSvc.Evaluate(ctx, input.EntityKey)",
		"func (s *SprintService) BulkAddToSprint", "func (s *SprintService) SelectSprint",
		"filterSprintSelectionAdmission", "func (s *SprintService) GetSprintReadiness",
		"applyReadinessAdmission", "func (s *SprintService) PlanSprint",
		"filterSprintPlanAdmission", "func (s *SprintService) CloseSprintWithCarryover",
		"GetLatestGoalReviewTx",
	} {
		if !strings.Contains(source, want) {
			t.Errorf("E19-F10 shared contract missing %q", want)
		}
	}
}
