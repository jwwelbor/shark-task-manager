package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/trackerattempts"
	"github.com/stretchr/testify/require"
)

// B088 R4: a tracker write rejected before it mutates anything must still be on
// record. The test runs the real entry point as a child process, so removing the
// recording call from main leaves the ledger empty and fails here.
func TestEntryPointRecordsRejectedTrackerAttempt(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "shark")
	build := exec.Command("go", "build", "-o", binary, ".")
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))
	run := func(ledger string, args ...string) error {
		cmd := exec.Command(binary, args...)
		cmd.Env = append(os.Environ(), "HOME="+t.TempDir(),
			trackerattempts.EnvLog+"="+ledger, trackerattempts.EnvRun+"=run-x")
		cmd.Dir = t.TempDir()
		return cmd.Run()
	}
	ledgerPath := filepath.Join(t.TempDir(), trackerattempts.LedgerName)
	// Missing required arguments: cobra rejects both before any command body runs.
	require.Error(t, run(ledgerPath, "create", "note"))
	require.Error(t, run(ledgerPath, "no-such-command"))
	_ = run(ledgerPath, "get", "E01")

	data, err := os.ReadFile(ledgerPath)
	require.NoError(t, err)
	ledger, err := trackerattempts.Parse(data, "run-x")
	require.NoError(t, err)
	require.Len(t, ledger.Entries, 3)
	require.True(t, ledger.Entries[0].Write, "rejected create is an attempted write")
	require.True(t, ledger.Entries[1].Write, "an unresolvable command fails closed")
	require.False(t, ledger.Entries[2].Write, "a read is not a write attempt")

	// Outside a dispatched run nothing is recorded.
	unrecorded := filepath.Join(t.TempDir(), trackerattempts.LedgerName)
	cmd := exec.Command(binary, "create", "note")
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir(), trackerattempts.EnvLog+"="+unrecorded)
	cmd.Dir = t.TempDir()
	_ = cmd.Run()
	_, err = os.Stat(unrecorded)
	require.True(t, os.IsNotExist(err))
}
