package contracts

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestB078RelatedDocsDeleteReturnsDatabaseInitializationErrorBeforeServiceDispatch(t *testing.T) {
	cli, _, _, _ := newB078RelatedDocsDeleteFixture(t)
	invalidDB := filepath.Join(cli.dir, "invalid-shark-tasks.db")
	require.NoError(t, os.WriteFile(invalidDB, []byte("not a SQLite database"), 0600))
	for _, route := range b078DatabaseFailureRoutes() {
		t.Run(route.name, func(t *testing.T) {
			for _, jsonMode := range []bool{true, false} {
				mode := "human"
				if jsonMode {
					mode = "json"
				}
				t.Run(mode, func(t *testing.T) { assertB078DatabaseFailure(t, cli, invalidDB, route, jsonMode) })
			}
		})
	}

	assertB078MissingSelectorPrecedesDatabaseAcquisition(t, cli, invalidDB)
}

func b078DatabaseFailureRoutes() []struct{ name, selector, key string } {
	return []struct{ name, selector, key string }{
		{name: "epic", selector: "--epic", key: "E10"},
		{name: "feature", selector: "--feature", key: "E10-F02"},
		{name: "task", selector: "--task", key: "T-E10-F02-013"},
		{name: "bug", selector: "--bug", key: "B001"},
		{name: "change-card", selector: "--change", key: "CC-001"},
		{name: "question", selector: "--question", key: "Q001"},
		{name: "tech-debt", selector: "--tech-debt", key: "TD-001"},
	}
}

func assertB078DatabaseFailure(t *testing.T, cli b078CLI, invalidDB string, route struct{ name, selector, key string }, jsonMode bool) {
	t.Helper()
	stdout, stderr, exitCode := cli.runWithDatabase(invalidDB, jsonMode, "related-docs", "delete", "missing", route.selector+"="+route.key)
	require.Equal(t, 2, exitCode)
	message := string(stderr)
	if jsonMode {
		var response struct {
			Error   bool   `json:"error"`
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		require.NoError(t, json.Unmarshal(stdout, &response))
		require.True(t, response.Error)
		require.Equal(t, "COMMAND_ERROR", response.Code)
		message = response.Message
	} else {
		require.Empty(t, stdout)
	}
	require.Contains(t, message, "failed to unlink document from "+route.name+" "+route.key)
	require.NotContains(t, strings.ToLower(message), "panic")
	require.NotContains(t, strings.ToLower(message), "goroutine")
	require.NotContains(t, message, "unlinked")
}

func assertB078MissingSelectorPrecedesDatabaseAcquisition(t *testing.T, cli b078CLI, invalidDB string) {
	t.Helper()
	stdout, stderr, exitCode := cli.runWithDatabase(invalidDB, false, "related-docs", "delete", "missing")
	require.Equal(t, 1, exitCode)
	require.Empty(t, stdout)
	require.NotContains(t, strings.ToLower(string(stderr)), "database")
	require.Contains(t, string(stderr), "exactly one")
}
