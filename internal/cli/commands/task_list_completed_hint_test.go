package commands

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
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
	if err != nil {
		t.Fatalf("runTaskListWithService() error = %v", err)
	}
	if !strings.Contains(output, "T-E04-F01-003") || !strings.Contains(output, "Visible task") {
		t.Fatalf("output = %q, want visible task row", output)
	}
	if strings.Contains(output, "--all to display") {
		t.Fatalf("output = %q, unexpected completed-task hint", output)
	}
	if len(service.filters) != 1 {
		t.Fatalf("ListTasks() calls = %d, want 1 when visible tasks exist", len(service.filters))
	}
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
	if err != nil {
		t.Fatalf("runTaskListWithService() error = %v", err)
	}
	if !strings.Contains(output, want) {
		t.Fatalf("output = %q, want completed-task hint %q", output, want)
	}
	if strings.Contains(output, "No results found") {
		t.Fatalf("output = %q, generic empty-result message should be replaced by the hint", output)
	}
	assertTaskListHintFilters(t, service.filters)
}

func setTaskListHintFlags(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	for flag, value := range map[string]string{
		"agent":        "backend",
		"priority-min": "2",
		"priority-max": "5",
	} {
		if err := cmd.Flags().Set(flag, value); err != nil {
			t.Fatalf("set %s: %v", flag, err)
		}
	}
}

func assertTaskListHintFilters(t *testing.T, filters []services.TaskFilters) {
	t.Helper()
	if len(filters) != 2 {
		t.Fatalf("ListTasks() calls = %d, want 2", len(filters))
	}
	if filters[0].ShowAll || !filters[1].ShowAll {
		t.Fatalf("ShowAll filters = [%v, %v], want [false, true]", filters[0].ShowAll, filters[1].ShowAll)
	}
	first, second := filters[0], filters[1]
	first.ShowAll = false
	second.ShowAll = false
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("retry filters = %#v, want original filters except ShowAll", filters[1])
	}
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
			if err != nil {
				t.Fatalf("runTaskListWithService() error = %v", err)
			}
			if !strings.Contains(output, "No results found") {
				t.Fatalf("output = %q, want generic empty-result message", output)
			}
			if strings.Contains(output, "--all to display") {
				t.Fatalf("output = %q, unexpected --all hint", output)
			}
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
	if err != nil {
		t.Fatalf("runTaskListWithService() error = %v", err)
	}
	if strings.TrimSpace(output) != "[]" {
		t.Fatalf("JSON output = %q, want []", output)
	}
	if len(service.filters) != 1 {
		t.Fatalf("ListTasks() calls = %d, want 1 for JSON output", len(service.filters))
	}
}

func TestTaskListWithAllDoesNotRunCompletedTaskHintQuery(t *testing.T) {
	configureTaskListHintOutput(t, false)

	service := &queuedTaskListService{results: [][]*models.Task{{}}}
	cmd := newTaskListHintTestCommand(t)
	if err := cmd.Flags().Set("all", "true"); err != nil {
		t.Fatal(err)
	}
	output, err := captureTaskListHintStdout(t, func() error {
		return runTaskListWithService(cmd, []string{"E04", "F01"}, service)
	})
	if err != nil {
		t.Fatalf("runTaskListWithService() error = %v", err)
	}
	if !strings.Contains(output, "No results found") || strings.Contains(output, "--all to display") {
		t.Fatalf("output = %q, want ordinary empty-result message without hint", output)
	}
	if len(service.filters) != 1 {
		t.Fatalf("ListTasks() calls = %d, want 1 for --all", len(service.filters))
	}
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
	if err == nil {
		t.Fatal("runTaskListWithService() error = nil, want count-query error")
	}
	if len(service.filters) != 2 {
		t.Fatalf("ListTasks() calls = %d, want 2", len(service.filters))
	}
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
