package gaterun

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApprovedPlanIsCreateOnceAndRejectsSymlink(t *testing.T) {
	dir := newRunDir(t)
	created, err := CreateApprovedPlan(dir, []byte("approved revision"))
	require.NoError(t, err)
	require.True(t, created)
	created, err = CreateApprovedPlan(dir, []byte("approved revision"))
	require.NoError(t, err)
	require.False(t, created)
	_, err = CreateApprovedPlan(dir, []byte("different revision"))
	require.True(t, IsConflict(err))
	got, exists, err := ReadApprovedPlan(dir)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, []byte("approved revision"), got)
	if err := os.Remove(filepath.Join(dir, approvedPlanFileName)); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "target")
	require.NoError(t, os.WriteFile(target, []byte("external"), 0600))
	if err := os.Symlink(target, filepath.Join(dir, approvedPlanFileName)); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, _, err = ReadApprovedPlan(dir)
	require.Error(t, err)
	_, err = CreateApprovedPlan(dir, []byte("external"))
	require.Error(t, err)
}
