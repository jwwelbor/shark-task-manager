package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReviewStatusPreservesTrackedAndOtherProjectChanges(t *testing.T) {
	prefix, err := reviewRuntimePrefix("nested")
	require.NoError(t, err)
	status := []byte("?? nested/.shark/runs/run/result.json\x00 M nested/.shark/runs/tracked\x00?? .shark/runs/other-project\x00R  renamed\x00?? nested/.shark/runs/old-name\x00")
	want := []byte(" M nested/.shark/runs/tracked\x00?? .shark/runs/other-project\x00R  renamed\x00?? nested/.shark/runs/old-name\x00")
	got, err := filterReviewStatus(status, prefix)
	require.NoError(t, err)
	require.Equal(t, want, got)
	_, err = filterReviewStatus([]byte("?? truncated"), prefix)
	require.Error(t, err)
	for _, selected := range []string{"nested", "nested/.shark", "nested/.shark/runs", "nested/.shark/runs/run/file"} {
		require.True(t, selectionOverlapsRuntime(selected, prefix), selected)
	}
	require.False(t, selectionOverlapsRuntime("nested/app.go", prefix))
	require.False(t, selectionOverlapsRuntime("other/.shark/runs/run/file", prefix))
	for _, invalid := range []string{"../elsewhere", "nested/../other", "nested//child", `nested\other`, ""} {
		_, err := reviewRuntimePrefix(invalid)
		require.Error(t, err, invalid)
	}
}
