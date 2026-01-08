package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/spf13/cobra"
)

// taskNoteCmd is the parent command for note operations
var taskNoteCmd = &cobra.Command{
	Use:   "note",
	Short: "Manage task notes",
	Long:  `Add, view, and manage typed notes for tasks.`,
}

// taskNoteAddCmd adds a note to a task
var taskNoteAddCmd = &cobra.Command{
	Use:   "add <task-key> --type <type> <content>",
	Short: "Add a typed note to a task",
	Long: `Add a typed note to a task for context, decisions, and documentation.

Note Types:
  comment        - General observation
  decision       - Why we chose X over Y
  blocker        - What's blocking progress
  solution       - How we solved a problem
  reference      - External links, documentation
  implementation - What we actually built
  testing        - Test results, coverage
  future         - Future improvements / TODO
  question       - Unanswered questions

Examples:
  shark task note add T-E10-F01-001 --type decision "Used SQLite for persistence"
  shark task note add T-E10-F01-001 --type blocker "Waiting for API specification" --created-by alice
  shark task note add T-E10-F01-001 --type reference "https://example.com/docs"
  shark task note add T-E10-F01-001 --type solution "Fixed by adding null check" --json`,
	Args: cobra.ExactArgs(2),
	RunE: runTaskNoteAdd,
}

// taskNotesCmd lists notes for a task
var taskNotesCmd = &cobra.Command{
	Use:   "notes <task-key>",
	Short: "List notes for a task",
	Long: `List all notes for a task, optionally filtered by type.

Examples:
  shark task notes T-E10-F01-001                    List all notes
  shark task notes T-E10-F01-001 --type decision    List decision notes only
  shark task notes T-E10-F01-001 --type decision,solution  List multiple types
  shark task notes T-E10-F01-001 --json             Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskNotes,
}

// taskTimelineCmd shows task timeline
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

// TimelineEvent represents a unified timeline event (status change or note)
type TimelineEvent struct {
	Timestamp time.Time `json:"timestamp"`
	EventType string    `json:"event_type"` // "status" or note type
	Content   string    `json:"content"`
	Actor     string    `json:"actor,omitempty"`
}

// runTaskNoteAdd handles the task note add command
func runTaskNoteAdd(cmd *cobra.Command, args []string) error {
	taskKey := args[0]
	content := args[1]

	noteTypeStr, _ := cmd.Flags().GetString("type")
	createdBy, _ := cmd.Flags().GetString("created-by")

	if noteTypeStr == "" {
		return fmt.Errorf("--type flag is required")
	}

	// Validate note type
	if err := models.ValidateNoteType(noteTypeStr); err != nil {
		return err
	}

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	ctx := context.Background()
	dbWrapper := repoDb
	taskRepo := repository.NewTaskRepository(dbWrapper)
	noteRepo := repository.NewTaskNoteRepository(dbWrapper)

	// Get task by key
	task, err := taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		return fmt.Errorf("task %s not found", taskKey)
	}

	// Create note
	var createdByPtr *string
	if createdBy != "" {
		createdByPtr = &createdBy
	}

	note := &models.TaskNote{
		TaskID:    task.ID,
		NoteType:  models.NoteType(noteTypeStr),
		Content:   content,
		CreatedBy: createdByPtr,
	}

	err = noteRepo.Create(ctx, note)
	if err != nil {
		return fmt.Errorf("failed to create note: %w", err)
	}

	// Retrieve the note to get the timestamp
	note, err = noteRepo.GetByID(ctx, note.ID)
	if err != nil {
		return fmt.Errorf("failed to retrieve created note: %w", err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(note)
	}

	// Human-readable output
	creator := "unknown"
	if note.CreatedBy != nil {
		creator = *note.CreatedBy
	}

	fmt.Printf("Note added to %s\n\n", taskKey)
	fmt.Printf("[%s] %s (%s)\n", strings.ToUpper(noteTypeStr), note.CreatedAt.Format("2006-01-02 15:04"), creator)
	fmt.Println(content)

	return nil
}

// runTaskNotes handles the task notes command
func runTaskNotes(cmd *cobra.Command, args []string) error {
	taskKey := args[0]
	noteTypesStr, _ := cmd.Flags().GetString("type")

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	ctx := context.Background()
	dbWrapper := repoDb
	taskRepo := repository.NewTaskRepository(dbWrapper)
	noteRepo := repository.NewTaskNoteRepository(dbWrapper)

	// Get task by key
	task, err := taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		return fmt.Errorf("task %s not found", taskKey)
	}

	// Parse note types filter
	var noteTypes []string
	if noteTypesStr != "" {
		noteTypes = strings.Split(noteTypesStr, ",")
		// Validate each note type
		for _, nt := range noteTypes {
			if err := models.ValidateNoteType(strings.TrimSpace(nt)); err != nil {
				return err
			}
		}
	}

	// Get notes
	var notes []*models.TaskNote
	if len(noteTypes) > 0 {
		notes, err = noteRepo.GetByTaskIDAndType(ctx, task.ID, noteTypes)
	} else {
		notes, err = noteRepo.GetByTaskID(ctx, task.ID)
	}
	if err != nil {
		return fmt.Errorf("failed to get notes: %w", err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(notes)
	}

	// Human-readable output
	if len(notes) == 0 {
		fmt.Printf("No notes found for task %s\n", taskKey)
		return nil
	}

	fmt.Printf("Task %s: %s (%d notes)\n\n", taskKey, task.Title, len(notes))

	for _, note := range notes {
		creator := "unknown"
		if note.CreatedBy != nil {
			creator = *note.CreatedBy
		}

		fmt.Printf("[%s] %s (%s)\n", strings.ToUpper(string(note.NoteType)), note.CreatedAt.Format("2006-01-02 15:04"), creator)
		fmt.Println(note.Content)
		fmt.Println()
	}

	return nil
}

// runTaskTimeline handles the task timeline command
func runTaskTimeline(cmd *cobra.Command, args []string) error {
	taskKey := args[0]

	// Get database connection
	repoDb, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}
	// Note: Database will be closed automatically by PersistentPostRunE hook

	ctx := context.Background()
	taskRepo := repository.NewTaskRepository(repoDb)
	noteRepo := repository.NewTaskNoteRepository(repoDb)

	// Get task by key
	task, err := taskRepo.GetByKey(ctx, taskKey)
	if err != nil {
		return fmt.Errorf("task %s not found", taskKey)
	}

	// Get status changes from task_history
	rows, err := repoDb.DB.QueryContext(ctx, `
		SELECT old_status, new_status, agent, timestamp
		FROM task_history
		WHERE task_id = ?
		ORDER BY timestamp ASC
	`, task.ID)
	if err != nil {
		return fmt.Errorf("failed to get task history: %w", err)
	}
	defer rows.Close()

	var timeline []TimelineEvent

	// Add task creation event
	timeline = append(timeline, TimelineEvent{
		Timestamp: task.CreatedAt,
		EventType: "status",
		Content:   "Created",
		Actor:     "",
	})

	// Add status changes
	for rows.Next() {
		var oldStatus, newStatus string
		var oldStatusPtr, agentPtr *string
		var ts time.Time
		if err := rows.Scan(&oldStatusPtr, &newStatus, &agentPtr, &ts); err != nil {
			return fmt.Errorf("failed to scan task history: %w", err)
		}

		agent := ""
		if agentPtr != nil {
			agent = *agentPtr
		}

		if oldStatusPtr != nil {
			oldStatus = *oldStatusPtr
		} else {
			oldStatus = ""
		}

		var content string
		if oldStatus == "" {
			content = fmt.Sprintf("Status: → %s", newStatus)
		} else {
			content = fmt.Sprintf("Status: %s → %s", oldStatus, newStatus)
		}

		timeline = append(timeline, TimelineEvent{
			Timestamp: ts,
			EventType: "status",
			Content:   content,
			Actor:     agent,
		})
	}

	// Get notes
	notes, err := noteRepo.GetByTaskID(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("failed to get notes: %w", err)
	}

	// Add notes to timeline
	for _, note := range notes {
		actor := ""
		if note.CreatedBy != nil {
			actor = *note.CreatedBy
		}

		// Truncate long content for timeline view
		content := note.Content
		if len(content) > 80 {
			content = content[:77] + "..."
		}

		timeline = append(timeline, TimelineEvent{
			Timestamp: note.CreatedAt,
			EventType: string(note.NoteType),
			Content:   content,
			Actor:     actor,
		})
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

	// Human-readable output
	fmt.Printf("Task %s: %s\n\n", taskKey, task.Title)
	fmt.Println("Timeline:")

	for _, event := range timeline {
		actor := ""
		if event.Actor != "" {
			actor = fmt.Sprintf(" (%s)", event.Actor)
		}

		if event.EventType == "status" {
			fmt.Printf("  %s  %s%s\n", event.Timestamp.Format("2006-01-02 15:04"), event.Content, actor)
		} else {
			fmt.Printf("  %s  [%s] %s%s\n", event.Timestamp.Format("2006-01-02 15:04"), strings.ToUpper(event.EventType), event.Content, actor)
		}
	}

	return nil
}

func init() {
	// Add note subcommand to task command
	taskCmd.AddCommand(taskNoteCmd)
	taskCmd.AddCommand(taskNotesCmd)
	taskCmd.AddCommand(taskTimelineCmd)

	// Add subcommands to note command
	taskNoteCmd.AddCommand(taskNoteAddCmd)

	// Flags for note add
	taskNoteAddCmd.Flags().StringP("type", "t", "", "Note type (required): comment, decision, blocker, solution, reference, implementation, testing, future, question")
	taskNoteAddCmd.Flags().StringP("created-by", "c", "", "Creator name (optional)")
	_ = taskNoteAddCmd.MarkFlagRequired("type")

	// Flags for notes list
	taskNotesCmd.Flags().StringP("type", "t", "", "Filter by note type (comma-separated for multiple)")
}
