package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

// taskTimelineCmd shows task timeline — task-specific, not consolidated.
var taskTimelineCmd = &cobra.Command{
	Use:   "timeline <task-key>",
	Short: "Show task timeline with status changes and notes",
	Long: `Show a unified chronological timeline of status changes and notes for a task.

This command interleaves task status changes from task_history with notes from task_notes
to provide a complete history of what happened on the task.

Examples:
  shark task timeline T-E10-F01-001       Show timeline
  shark task timeline T-E10-F01-001 --json  Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskTimeline,
}

// TimelineEvent represents a unified timeline event (status change, note, or rejection)
type TimelineEvent struct {
	Timestamp      time.Time `json:"timestamp"`
	EventType      string    `json:"event_type"` // "status", "rejection", or note type
	Content        string    `json:"content"`
	Actor          string    `json:"actor,omitempty"`
	Reason         string    `json:"reason,omitempty"`          // For rejection events
	ReasonDocument *string   `json:"reason_document,omitempty"` // Document path for rejection
}

func runTaskTimeline(cmd *cobra.Command, args []string) error {
	taskKey := args[0]

	taskSvc := cli.GetTaskServiceWithDeps()

	task, err := taskSvc.GetTask(cmd.Context(), taskKey)
	if err != nil {
		return fmt.Errorf("task %s not found", taskKey)
	}

	histories, err := taskSvc.GetTaskHistory(cmd.Context(), taskKey)
	if err != nil {
		return fmt.Errorf("failed to get task history: %w", err)
	}

	var timeline []TimelineEvent

	timeline = append(timeline, TimelineEvent{
		Timestamp: task.CreatedAt,
		EventType: "status",
		Content:   "Created",
	})

	for _, history := range histories {
		oldStatus := ""
		if history.OldStatus != nil {
			oldStatus = *history.OldStatus
		}

		agent := ""
		if history.Agent != nil {
			agent = *history.Agent
		}

		if history.RejectionReason != nil {
			reason := truncateRunes(*history.RejectionReason, 77)

			var content string
			if oldStatus != "" && history.NewStatus != "" {
				content = fmt.Sprintf("⚠️ Rejected by %s: %s → %s", agent, oldStatus, history.NewStatus)
			} else if agent != "" {
				content = fmt.Sprintf("⚠️ Rejected by %s", agent)
			} else {
				content = "⚠️ Rejected"
			}

			timeline = append(timeline, TimelineEvent{
				Timestamp: history.Timestamp,
				EventType: "rejection",
				Content:   content,
				Actor:     agent,
				Reason:    reason,
			})
		} else {
			var content string
			if oldStatus == "" {
				content = fmt.Sprintf("Status: → %s", history.NewStatus)
			} else {
				content = fmt.Sprintf("Status: %s → %s", oldStatus, history.NewStatus)
			}

			timeline = append(timeline, TimelineEvent{
				Timestamp: history.Timestamp,
				EventType: "status",
				Content:   content,
				Actor:     agent,
			})
		}
	}

	noteSvc, err := cli.GetNoteService(cmd.Context())
	if err != nil {
		cli.Warning(fmt.Sprintf("Failed to get note service: %v", err))
	} else {
		notes, notesErr := noteSvc.ListNotes(cmd.Context(), models.EntityTypeTask, taskKey, nil)
		if notesErr != nil {
			cli.Warning(fmt.Sprintf("Failed to get notes: %v", notesErr))
		} else {
			for _, note := range notes {
				actor := ""
				if note.CreatedBy != nil {
					actor = *note.CreatedBy
				}

				content := truncateRunes(note.Content, 77)

				timeline = append(timeline, TimelineEvent{
					Timestamp: note.CreatedAt,
					EventType: string(note.NoteType),
					Content:   content,
					Actor:     actor,
				})
			}
		}
	}

	// Sort timeline by timestamp
	for i := 0; i < len(timeline); i++ {
		for j := i + 1; j < len(timeline); j++ {
			if timeline[j].Timestamp.Before(timeline[i].Timestamp) {
				timeline[i], timeline[j] = timeline[j], timeline[i]
			}
		}
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(timeline)
	}

	fmt.Printf("Task %s: %s\n\n", taskKey, task.Title)
	fmt.Println("Timeline:")

	for _, event := range timeline {
		actor := ""
		if event.Actor != "" {
			actor = fmt.Sprintf(" (%s)", event.Actor)
		}

		if event.EventType == "status" {
			fmt.Printf("  %s  %s%s\n", event.Timestamp.Format("2006-01-02 15:04"), event.Content, actor)
		} else if event.EventType == "rejection" {
			fmt.Printf("  %s  %s%s\n", event.Timestamp.Format("2006-01-02 15:04"), event.Content, actor)
			if event.Reason != "" {
				fmt.Printf("        Reason: %s\n", event.Reason)
			}
			if event.ReasonDocument != nil && *event.ReasonDocument != "" {
				fmt.Printf("        📄 %s\n", *event.ReasonDocument)
			}
		} else {
			fmt.Printf("  %s  [%s] %s%s\n", event.Timestamp.Format("2006-01-02 15:04"), strings.ToUpper(event.EventType), event.Content, actor)
		}
	}

	return nil
}

func init() {
	// Use generic note commands for add and list
	taskCmd.AddCommand(makeNoteCmd("task"))
	taskCmd.AddCommand(makeNotesCmd("task"))

	// Task-specific timeline command
	taskCmd.AddCommand(taskTimelineCmd)
}
