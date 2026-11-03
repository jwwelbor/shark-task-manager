package commands

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

type queuedTaskListService struct {
	results [][]*models.Task
	errors  []error
	filters []services.TaskFilters
}

func (s *queuedTaskListService) ListTasks(_ context.Context, filters services.TaskFilters) ([]*models.Task, error) {
	s.filters = append(s.filters, filters)
	index := len(s.filters) - 1
	if index < len(s.errors) && s.errors[index] != nil {
		return nil, s.errors[index]
	}
	if index >= len(s.results) {
		return nil, nil
	}
	return s.results[index], nil
}

func TestTaskListHintsWhenOnlyCompletedTasksAreHidden(t *testing.T) {
	backend := "backend"
	completedTasks := []*models.Task{
		{BaseEntity: models.BaseEntity{Key: "T-E04-F01-001"}, Status: models.TaskStatusCompleted, AgentType: &backend, Priority: 3},
		{BaseEntity: models.BaseEntity{Key: "T-E04-F01-002"}, Status: models.TaskStatusCompleted, AgentType: &backend, Priority: 4},
	}
	for _, tc := range []struct {
		name  string
		tasks []*models.Task
		want  string
	}{
		{name: "plural", tasks: completedTasks, want: "2 completed tasks hidden; run with --all to display"},
		{name: "singular", tasks: completedTasks[:1], want: "1 completed task hidden; run with --all to display"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runCompletedTaskHintCase(t, tc.tasks, tc.want)
		})
	}
}

func TestTaskListDoesNotRetryWhenVisibleTasksExist(t *testing.T) {
	configureTaskListHintOutput(t, false)

	service := &queuedTaskListService{results: [][]*models.Task{{{
		BaseEntity: models.BaseEntity{Key: "T-E04-F01-003", Title: "Visible task"},
		Status:     models.TaskStatus("todo"),
	}}}}
	cmd := newTaskListHintTestCommand(t)
	output, err := captureTaskListHintStdout(t, func() error {
		return runTaskListWithService(cmd, []string{"E04", "F01"}, service)
	})
	require.NoError(t, err, "runTaskListWithService()")
	require.Contains(t, output, "T-E04-F01-003")
	require.Contains(t, output, "Visible task")
	require.NotContains(t, output, "--all to display")
	require.Len(t, service.filters, 1, "ListTasks() calls when visible tasks exist")
}

func runCompletedTaskHintCase(t *testing.T, completedTasks []*models.Task, want string) {
	t.Helper()
	configureTaskListHintOutput(t, false)
	service := &queuedTaskListService{results: [][]*models.Task{{}, completedTasks}}
	cmd := newTaskListHintTestCommand(t)
	setTaskListHintFlags(t, cmd)

	output, err := captureTaskListHintStdout(t, func() error {
		return runTaskListWithService(cmd, []string{"E04", "F01"}, service)
	})
	require.NoError(t, err, "runTaskListWithService()")
	require.Contains(t, output, want)
	require.NotContains(t, output, "No results found", "completed-task hint should replace generic empty-result message")
	assertTaskListHintFilters(t, service.filters)
}

func setTaskListHintFlags(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	for flag, value := range map[string]string{
		"agent":        "backend",
		"priority-min": "2",
		"priority-max": "5",
	} {
		require.NoError(t, cmd.Flags().Set(flag, value), "set %s", flag)
	}
}

func assertTaskListHintFilters(t *testing.T, filters []services.TaskFilters) {
	t.Helper()
	require.Len(t, filters, 2, "ListTasks() calls")
	require.False(t, filters[0].ShowAll, "first query should preserve the default completed-task filter")
	require.True(t, filters[1].ShowAll, "retry should include completed tasks")
	first, second := filters[0], filters[1]
	first.ShowAll = false
	second.ShowAll = false
	require.Equal(t, first, second, "retry filters should match the original except ShowAll")
}

func TestTaskListKeepsGenericEmptyMessageWhenNoCompletedTasksAreHidden(t *testing.T) {
	configureTaskListHintOutput(t, false)

	for _, tc := range []struct {
		name    string
		results [][]*models.Task
	}{
		{name: "feature has no tasks", results: [][]*models.Task{{}, {}}},
		{name: "retry includes non-completed task", results: [][]*models.Task{{}, {
			{BaseEntity: models.BaseEntity{Key: "T-E04-F01-001"}, Status: models.TaskStatus("todo")},
		}}},
		{name: "retry includes mixed statuses", results: [][]*models.Task{{}, {
			{BaseEntity: models.BaseEntity{Key: "T-E04-F01-001"}, Status: models.TaskStatusCompleted},
			{BaseEntity: models.BaseEntity{Key: "T-E04-F01-002"}, Status: models.TaskStatus("todo")},
		}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &queuedTaskListService{results: tc.results}
			cmd := newTaskListHintTestCommand(t)
			output, err := captureTaskListHintStdout(t, func() error {
				return runTaskListWithService(cmd, []string{"E04", "F01"}, service)
			})
			require.NoError(t, err, "runTaskListWithService()")
			require.Contains(t, output, "No results found")
			require.NotContains(t, output, "--all to display")
		})
	}
}

func TestTaskListJSONEmptyResultDoesNotRunCompletedTaskHintQuery(t *testing.T) {
	configureTaskListHintOutput(t, true)

	service := &queuedTaskListService{results: [][]*models.Task{{}}}
	cmd := newTaskListHintTestCommand(t)
	output, err := captureTaskListHintStdout(t, func() error {
		return runTaskListWithService(cmd, []string{"E04", "F01"}, service)
	})
	require.NoError(t, err, "runTaskListWithService()")
	require.Equal(t, "[]", strings.TrimSpace(output), "JSON output")
	require.Len(t, service.filters, 1, "ListTasks() calls for JSON output")
}

func TestTaskListWithAllDoesNotRunCompletedTaskHintQuery(t *testing.T) {
	configureTaskListHintOutput(t, false)

	service := &queuedTaskListService{results: [][]*models.Task{{}}}
	cmd := newTaskListHintTestCommand(t)
	require.NoError(t, cmd.Flags().Set("all", "true"))
	output, err := captureTaskListHintStdout(t, func() error {
		return runTaskListWithService(cmd, []string{"E04", "F01"}, service)
	})
	require.NoError(t, err, "runTaskListWithService()")
	require.Contains(t, output, "No results found")
	require.NotContains(t, output, "--all to display")
	require.Len(t, service.filters, 1, "ListTasks() calls for --all")
}

func TestTaskListReturnsErrorWhenCompletedTaskCountQueryFails(t *testing.T) {
	configureTaskListHintOutput(t, false)

	service := &queuedTaskListService{
		results: [][]*models.Task{{}},
		errors:  []error{nil, errors.New("query failed")},
	}
	cmd := newTaskListHintTestCommand(t)
	_, err := captureTaskListHintStdout(t, func() error {
		return runTaskListWithService(cmd, []string{"E04", "F01"}, service)
	})
	require.Error(t, err, "runTaskListWithService() should return the count-query error")
	require.Len(t, service.filters, 2, "ListTasks() calls")
}

func newTaskListHintTestCommand(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "task-list-test"}
	cmd.SetContext(context.Background())
	cmd.Flags().String("epic", "", "epic")
	cmd.Flags().String("feature", "", "feature")
	cmd.Flags().String("status", "", "status")
	cmd.Flags().String("agent", "", "agent")
	cmd.Flags().Bool("show-all", false, "show all")
	cmd.Flags().Bool("all", false, "all")
	cmd.Flags().Bool("blocked", false, "blocked")
	cmd.Flags().Int("priority-min", 0, "minimum priority")
	cmd.Flags().Int("priority-max", 0, "maximum priority")
	cmd.Flags().Bool("has-rejections", false, "has rejections")
	cmd.Flags().StringSlice("tag", nil, "tags")
	return cmd
}

func configureTaskListHintOutput(t *testing.T, jsonMode bool) {
	t.Helper()
	previousJSON := cli.GlobalConfig.JSON
	previousNoColor := cli.GlobalConfig.NoColor
	previousField := cli.GlobalConfig.Field
	t.Cleanup(func() {
		cli.GlobalConfig.JSON = previousJSON
		cli.GlobalConfig.NoColor = previousNoColor
		cli.GlobalConfig.Field = previousField
	})
	cli.GlobalConfig.JSON = jsonMode
	cli.GlobalConfig.NoColor = true
	cli.GlobalConfig.Field = ""
}

func captureTaskListHintStdout(t *testing.T, run func() error) (string, error) {
	t.Helper()
	previousPtermWriter := pterm.Info.Writer
	previousTable := pterm.DefaultTable
	return captureStdoutForTest(t, func() error {
		pterm.Info.Writer = os.Stdout
		pterm.DefaultTable = *pterm.DefaultTable.WithWriter(os.Stdout)
		defer func() {
			pterm.Info.Writer = previousPtermWriter
			pterm.DefaultTable = previousTable
		}()
		return run()
	})
}
