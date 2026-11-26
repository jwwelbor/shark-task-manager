package trackerattempts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The ledger is the only record of a write rejected before it mutated, so each
// property below is one the parent relies on when it admits a withdrawal.
func TestLedgerBindsRunAndAttributesAttemptsToFindings(t *testing.T) {
	path := filepath.Join(t.TempDir(), LedgerName)
	require.NoError(t, Append(path, "run-a", "create note", []string{"create", "note", "B001", "F-1 is style only"}, true))
	require.NoError(t, Append(path, "run-a", "create note", []string{"create", "note", "B001", "F-10 real defect"}, true))
	require.NoError(t, Append(path, "run-a", "get", []string{"get", "B001"}, false))
	require.NoError(t, Append(path, "run-a", "create tech-debt", []string{"create", "tech-debt", "untitled"}, true))
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	ledger, err := Parse(data, "run-a")
	require.NoError(t, err)
	require.Len(t, ledger.Entries, 4)
	// F-10 must not be mistaken for F-1, and a read is never an attempt.
	require.Equal(t, 1, ledger.WriteAttempts("F-1", []string{"F-1", "F-10"}, true))
	// An unattributed write is charged to a finding only when no other finding is expected to file.
	require.Equal(t, 2, ledger.WriteAttempts("F-1", []string{"F-1", "F-10"}, false))

	_, err = Parse(data, "run-b")
	require.ErrorContains(t, err, `belongs to run "run-a", not "run-b"`)
	_, err = Parse(append(data, []byte("not json\n")...), "run-a")
	require.ErrorContains(t, err, "invalid")
}
