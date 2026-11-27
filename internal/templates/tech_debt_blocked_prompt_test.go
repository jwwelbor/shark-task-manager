package templates

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The tech-debt pause prompt is the change-card pause prompt with the entity
// noun swapped, with and without a blocked reason.
func TestE31F01_TC020f_TechDebtBlockedPromptMirrorsChange(t *testing.T) {
	renderer := newCanonicalOrchestratorRenderer(t)
	cases := []struct {
		name   string
		reason string
	}{
		{name: "with blocked reason", reason: "waiting for an architect decision"},
		{name: "without blocked reason"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vars := map[string]string{"id": "TD-001", "blocked_reason": tc.reason}

			got, err := renderer.Render("tech_debt/blocked.md", vars)
			require.NoError(t, err)
			change, err := renderer.Render("change/blocked.md", vars)
			require.NoError(t, err)

			assert.Equal(t, strings.Replace(change, "Change TD-001", "Tech debt TD-001", 1), got)
			assert.Contains(t, got, "Tech debt TD-001 is blocked.")
			assert.Equal(t, tc.reason != "", strings.Contains(got, "Blocked reason: "+tc.reason))
		})
	}
}
