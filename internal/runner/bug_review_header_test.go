package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionTwoReviewHeaderRequiresModelMetadata(t *testing.T) {
	base := "**Runner metadata:** `runner_mode=dispatched-six-angle` `specialists_completed=6` `consolidator_completed=true`\n" +
		"**Review contract:** `2`\n**Requested model:** `` · **Verified effective model:** `observed-model`\n**Verdict:** PASS"
	require.NoError(t, validateUniqueBugReviewHeader(base))
	missingRequested := "**Runner metadata:** `runner_mode=dispatched-six-angle` `specialists_completed=6` `consolidator_completed=true`\n" +
		"**Review contract:** `2`\n**Verified effective model:** `observed-model`\n**Verdict:** PASS"
	require.ErrorContains(t, validateUniqueBugReviewHeader(missingRequested), "Requested model")
	historical := "**Runner metadata:** `runner_mode=dispatched-six-angle` `specialists_completed=6` `consolidator_completed=true`\n**Verdict:** PASS"
	require.NoError(t, validateUniqueBugReviewHeader(historical))
}
