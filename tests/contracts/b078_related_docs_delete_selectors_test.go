package contracts

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestB078RelatedDocsDeleteRequiresExactlyOneSelector(t *testing.T) {
	cli, sqlDB, targetTask, otherTask := newB078RelatedDocsDeleteFixture(t)
	const title = "S010 current continuation"
	seedB078DuplicateTitleTaskLinks(t, sqlDB, targetTask, otherTask, title)
	for _, test := range []struct {
		name  string
		flags []string
	}{
		{name: "missing selector"},
		{name: "multiple selectors", flags: []string{"--epic=E10", "--task=" + targetTask.Key}},
	} {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{"related-docs", "delete", title}, test.flags...)
			message := cli.rejection(t, args...)
			require.Contains(t, message, "exactly one")
			require.Equal(t, []string{"docs/continuation/target.md"}, cli.reportPaths(t, targetTask.Key))
			require.Equal(t, []string{"docs/continuation/other.md"}, cli.reportPaths(t, otherTask.Key))
		})
	}
}
