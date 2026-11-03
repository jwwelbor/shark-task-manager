package commands

import (
	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

func maybeShowCompletedTaskHint(cmd *cobra.Command, svc taskListServicer, filters services.TaskFilters, tasks []*models.Task) (bool, error) {
	if filters.ShowAll || len(tasks) > 0 {
		return false, nil
	}

	allFilters := filters
	allFilters.ShowAll = true
	allTasks, err := svc.ListTasks(cmd.Context(), allFilters)
	if err != nil {
		return false, handleEntityServiceError(cmd, cli.GetTagService(), err, models.EntityTypeTask, "")
	}
	hiddenCount, ok := hiddenCompletedTaskCount(allTasks)
	if !ok {
		return false, nil
	}

	unit := "tasks"
	if hiddenCount == 1 {
		unit = "task"
	}
	cli.Info("%d completed %s hidden; run with --all to display", hiddenCount, unit)
	return true, nil
}

func hiddenCompletedTaskCount(tasks []*models.Task) (int, bool) {
	if len(tasks) == 0 {
		return 0, false
	}
	for _, task := range tasks {
		if task == nil || task.Status != models.TaskStatusCompleted {
			return 0, false
		}
	}
	return len(tasks), true
}
