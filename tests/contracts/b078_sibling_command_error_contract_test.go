package contracts

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/repository/techdebt"
	"github.com/stretchr/testify/require"
)

type b078SiblingCommand struct {
	name, key string
	args      []string
}

func TestB078SiblingCommandsPreserveMissingAndOperationalErrors(t *testing.T) {
	cli, sqlDB, repoDB := newB078RelatedDocsFixture(t)
	createB078ChangeCard(t, sqlDB, "CC-901", "sibling-change")
	createB078TechDebt(t, techdebt.NewTechDebtRepository(repoDB), "TD-901", "sibling-debt")
	commands := []b078SiblingCommand{
		{name: "change get", key: "CC-901", args: []string{"change", "get", "CC-901"}},
		{name: "change update", key: "CC-901", args: []string{"change", "update", "CC-901", "--title=changed"}},
		{name: "change delete", key: "CC-901", args: []string{"change", "delete", "CC-901"}},
		{name: "td get", key: "TD-901", args: []string{"td", "get", "TD-901"}},
		{name: "td update", key: "TD-901", args: []string{"td", "update", "TD-901", "--title=changed"}},
		{name: "td delete", key: "TD-901", args: []string{"td", "delete", "TD-901"}},
		{name: "td triage", key: "TD-901", args: []string{"td", "triage", "TD-901", "--severity=high"}},
	}
	for _, command := range commands {
		t.Run("missing "+command.name, func(t *testing.T) {
			args := append([]string(nil), command.args...)
			missingKey := command.key[:len(command.key)-3] + "999"
			args[2] = missingKey
			for _, jsonMode := range []bool{true, false} {
				stdout, stderr, code := cli.runWithDatabase(cli.dbPath, jsonMode, args...)
				require.Equal(t, 1, code, "%s", append(stdout, stderr...))
				message := b078SiblingMessage(t, stdout, stderr, jsonMode)
				require.Contains(t, message, missingKey)
				require.Contains(t, strings.ToLower(message), "not found")
				label := "tech-debt"
				if strings.HasPrefix(missingKey, "CC-") {
					label = "change-card"
				}
				require.Contains(t, strings.ToLower(message), label)
			}
		})
	}

	_, err := sqlDB.Exec(`UPDATE change_cards SET created_at = 'not-a-timestamp' WHERE key = 'CC-901'`)
	require.NoError(t, err)
	_, err = sqlDB.Exec(`UPDATE tech_debts SET created_at = 'not-a-timestamp' WHERE key = 'TD-901'`)
	require.NoError(t, err)
	for _, command := range commands {
		t.Run("operational "+command.name, func(t *testing.T) {
			for _, jsonMode := range []bool{true, false} {
				stdout, stderr, code := cli.runWithDatabase(cli.dbPath, jsonMode, command.args...)
				require.Equal(t, 2, code, "%s", append(stdout, stderr...))
				message := b078SiblingMessage(t, stdout, stderr, jsonMode)
				require.Contains(t, message, command.key)
				require.NotContains(t, strings.ToLower(message), "not found")
				require.NotContains(t, strings.ToLower(message), "success")
				require.NotContains(t, strings.ToLower(message), "confirm")
			}
		})
	}

	_, err = sqlDB.Exec("UPDATE change_cards SET created_at = datetime('now') WHERE key = 'CC-901'")
	require.NoError(t, err)
	_, err = sqlDB.Exec("UPDATE tech_debts SET created_at = datetime('now') WHERE key = 'TD-901'")
	require.NoError(t, err)
	for _, trigger := range []string{
		`CREATE TRIGGER b078_reject_change_update BEFORE UPDATE ON change_cards BEGIN SELECT RAISE(ABORT, 'fixture write refused'); END`,
		`CREATE TRIGGER b078_reject_change_delete BEFORE DELETE ON change_cards BEGIN SELECT RAISE(ABORT, 'fixture write refused'); END`,
		`CREATE TRIGGER b078_reject_td_update BEFORE UPDATE ON tech_debts BEGIN SELECT RAISE(ABORT, 'fixture write refused'); END`,
		`CREATE TRIGGER b078_reject_td_delete BEFORE DELETE ON tech_debts BEGIN SELECT RAISE(ABORT, 'fixture write refused'); END`,
	} {
		_, err = sqlDB.Exec(trigger)
		require.NoError(t, err)
	}
	writeCommands := []b078SiblingCommand{
		{name: "change update write", key: "CC-901", args: []string{"change", "update", "CC-901", "--title=changed"}},
		{name: "change delete write", key: "CC-901", args: []string{"change", "delete", "CC-901", "--force"}},
		{name: "td update write", key: "TD-901", args: []string{"td", "update", "TD-901", "--title=changed"}},
		{name: "td delete write", key: "TD-901", args: []string{"td", "delete", "TD-901", "--force"}},
		{name: "td triage write", key: "TD-901", args: []string{"td", "triage", "TD-901", "--severity=high"}},
	}
	for _, command := range writeCommands {
		t.Run(command.name, func(t *testing.T) {
			for _, jsonMode := range []bool{true, false} {
				stdout, stderr, code := cli.runWithDatabase(cli.dbPath, jsonMode, command.args...)
				require.Equal(t, 2, code, "%s", append(stdout, stderr...))
				message := b078SiblingMessage(t, stdout, stderr, jsonMode)
				require.Contains(t, message, "fixture write refused")
				require.Contains(t, strings.ToLower(message), command.args[1])
				require.NotContains(t, strings.ToLower(message), "not found")
				require.NotContains(t, strings.ToLower(message), "success")
			}
		})
	}
	var changeCount, debtCount int
	require.NoError(t, sqlDB.QueryRow("SELECT COUNT(*) FROM change_cards WHERE key='CC-901'").Scan(&changeCount))
	require.NoError(t, sqlDB.QueryRow("SELECT COUNT(*) FROM tech_debts WHERE key='TD-901'").Scan(&debtCount))
	require.Equal(t, 1, changeCount)
	require.Equal(t, 1, debtCount)
	var changeTitle, debtTitle, debtSeverity string
	require.NoError(t, sqlDB.QueryRow("SELECT title FROM change_cards WHERE key='CC-901'").Scan(&changeTitle))
	require.NoError(t, sqlDB.QueryRow("SELECT title, severity FROM tech_debts WHERE key='TD-901'").Scan(&debtTitle, &debtSeverity))
	require.Equal(t, "B078 CC-901", changeTitle)
	require.Equal(t, "B078 TD-901", debtTitle)
	require.Equal(t, "medium", debtSeverity)
}

func b078SiblingMessage(t *testing.T, stdout, stderr []byte, jsonMode bool) string {
	t.Helper()
	if !jsonMode {
		return string(append(append([]byte(nil), stdout...), stderr...))
	}
	var failure struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(stdout, &failure), "%s", stderr)
	require.True(t, failure.Error)
	return failure.Message
}
