package main

import (
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
	"github.com/stretchr/testify/require"
)

func TestExtractExitCodeUsesDatabaseFailureMarker(t *testing.T) {
	err := fmt.Errorf("lookup failed: %w", repoerr.ErrDatabaseFailure)
	require.Equal(t, 2, extractExitCode(err, 1), "repository database failures use exit code 2")
}
