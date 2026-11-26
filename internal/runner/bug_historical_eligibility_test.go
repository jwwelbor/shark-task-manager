package runner

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gaterun"
	"github.com/stretchr/testify/require"
)

// An unversioned review packet is historical only when the parent already
// applied that operation under the pre-v2 provenance contract. Any other
// state must not let old bytes earn current review credit.
func TestUnversionedReviewEligibilityRequiresAppliedPreV2Operation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		version  int
		state    gaterun.PersistenceState
		missing  bool
		eligible bool
	}{
		{name: "applied pre-v2 operation", version: gatepersist.BoundAdmissionVersion, state: gaterun.PersistenceStateTransitioned, eligible: true},
		{name: "applied current operation", version: gatepersist.CurrentAdmissionVersion, state: gaterun.PersistenceStateTransitioned},
		{name: "pending pre-v2 journal", version: gatepersist.BoundAdmissionVersion, state: gaterun.PersistenceStatePending},
		{name: "unbound applied operation", version: 0, state: gaterun.PersistenceStateTransitioned},
		{name: "fresh run without journal", missing: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, err := gaterun.RunDir(t.TempDir(), "run")
			require.NoError(t, err)
			if !tc.missing {
				state := gaterun.NewOperationState("run", "B001", "bug", "code_review", "code_review", "digest")
				state.AdmissionVersion, state.PersistenceState = tc.version, tc.state
				require.NoError(t, state.Save(dir))
			}
			err = requireHistoricalReview(gatepersist.Request{RunDir: dir})
			if tc.eligible {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "review_contract_version 2")
			}
		})
	}
}
