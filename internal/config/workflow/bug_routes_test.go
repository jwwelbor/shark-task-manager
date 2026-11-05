package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBugEvidenceRoutes(t *testing.T) {
	cfg := &WorkflowConfig{Steps: map[string]*Step{
		"research": {Outcomes: map[string]string{"pass": "plan_review"}},
		"plan_review": {
			ResultContract: ResultContractGateResultV1,
			Outcomes:       map[string]string{"pass": "development", "also_pass": "development", "retry": "plan_review", "also_retry": "plan_review"},
			OutcomeRoles:   map[string]string{"pass": "success", "also_pass": "success", "retry": "route_rework", "also_retry": "kickback_rework"},
		},
		"development": {Aliases: []string{"old_development"}},
	}}
	require.NoError(t, ValidateBugEvidenceRoutes(cfg))
	cfg.Steps["plan_review"].Outcomes["retry"] = "old_development"
	require.ErrorContains(t, ValidateBugEvidenceRoutes(cfg), "same target")
	cfg.Steps["plan_review"].Outcomes["retry"] = "plan_review"
	delete(cfg.Steps["research"].Outcomes, "pass")
	require.ErrorContains(t, ValidateBugEvidenceRoutes(cfg), "no pass outcome")
}
