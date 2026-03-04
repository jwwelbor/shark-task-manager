package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// --- Result types for JSON output ---

// StatusSetResult is the JSON output for the status set command.
type StatusSetResult struct {
	EntityType         string      `json:"entity_type"`
	EntityKey          string      `json:"entity_key"`
	Status             string      `json:"status"`
	Changed            bool        `json:"changed"`
	Message            string      `json:"message,omitempty"`
	PreviousStatus     string      `json:"previous_status,omitempty"`
	IsBackward         bool        `json:"is_backward,omitempty"`
	IsForced           bool        `json:"is_forced,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	ChildCount         int         `json:"child_count,omitempty"`
	OrchestratorAction interface{} `json:"orchestrator_action,omitempty"`
}

// StatusHistoryResult is the JSON output for the status history command.
type StatusHistoryResult struct {
	EntityType string               `json:"entity_type"`
	EntityKey  string               `json:"entity_key"`
	History    []StatusHistoryEntry `json:"history"`
	Total      int                  `json:"total"`
}

// StatusHistoryEntry represents a single status change in the history.
type StatusHistoryEntry struct {
	Timestamp string `json:"timestamp"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
	Agent     string `json:"agent,omitempty"`
	Notes     string `json:"notes,omitempty"`
}

// --- Adapter type for entityTransitioner interface ---

// entityTransitionerFunc adapts a function to the entityTransitioner interface.
type entityTransitionerFunc func(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error)

func (f entityTransitionerFunc) TransitionStatus(ctx context.Context, key string, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	return f(ctx, key, targetStatus, opts)
}

// --- Command definitions ---

// statusSetCmd sets an entity to a specific status.
var statusSetCmd = &cobra.Command{
	Use:   "set <key> <status>",
	Short: "Set entity to a specific status",
	Long: `Set an epic, feature, or task to a specific status. Entity type is auto-detected from the key format.

Idempotent: if the entity is already at the target status, returns exit 0 with "changed": false.

Key Formats:
  E07                Epic
  E07-F01            Feature
  E07-F01-001        Task

Examples:
  shark status set E07 active                       Set epic to active
  shark status set E07-F01 active                   Set feature to active
  shark status set E07-F01-001 in_development       Set task to in_development
  shark status set E07-F01-001 todo --reason="Needs rework" --force   Force backward transition
  shark status set E07-F01-001 in_development --json                   JSON output`,
	Args: cobra.ExactArgs(2),
	RunE: runStatusSet,
}

// statusAdvanceCmd advances an entity to its next workflow status.
var statusAdvanceCmd = &cobra.Command{
	Use:   "advance <key>",
	Short: "Advance entity to next workflow status",
	Long: `Advance an epic, feature, or task to its next workflow status.

Auto-selects the default next status. If the entity is in a terminal status
or has no valid transitions, reports that and exits.

For choosing a specific target status, use 'shark status set'.
For viewing available transitions, use 'shark status transitions'.

Examples:
  shark status advance E07-F01-001              Advance to next status
  shark status advance E07                      Advance epic
  shark status advance E07-F01-001 --json       JSON output`,
	Args: cobra.ExactArgs(1),
	RunE: runStatusAdvance,
}

// statusTransitionsCmd shows available transitions for an entity (read-only).
var statusTransitionsCmd = &cobra.Command{
	Use:   "transitions <key>",
	Short: "Show available status transitions",
	Long: `Show the available status transitions for an epic, feature, or task without making any changes.

This is a read-only command that displays what transitions are currently valid.

Key Formats:
  E07                Epic
  E07-F01            Feature
  E07-F01-001        Task

Examples:
  shark status transitions E07-F01-001          Show transitions for task
  shark status transitions E07                  Show transitions for epic
  shark status transitions E07-F01 --json       JSON output`,
	Args: cobra.ExactArgs(1),
	RunE: runStatusTransitions,
}

// statusHistoryCmd shows task status change history.
var statusHistoryCmd = &cobra.Command{
	Use:   "history <task-key>",
	Short: "Show task status change history",
	Long: `Show the status change history for a task. Only tasks have history records;
epics and features do not track status change history.

Key Formats:
  E07-F01-001        Task (short format)
  T-E07-F01-001      Task (traditional format)

Examples:
  shark status history E07-F01-001              Show task history
  shark status history E07-F01-001 --limit=10   Show last 10 changes
  shark status history E07-F01-001 --json       JSON output`,
	Args: cobra.ExactArgs(1),
	RunE: runStatusHistory,
}

func init() {
	// statusSetCmd flags
	statusSetCmd.Flags().String("reason", "", "Reason for backward or forced transitions")
	statusSetCmd.Flags().Bool("force", false, "Bypass workflow validation")
	statusSetCmd.Flags().String("notes", "", "Transition notes")

	// statusAdvanceCmd has no flags — just advances to the default next status.

	// statusHistoryCmd flags
	statusHistoryCmd.Flags().Int("limit", 50, "Maximum number of history entries to show")

	// Register subcommands under statusCmd (defined in status.go)
	statusCmd.AddCommand(statusSetCmd)
	statusCmd.AddCommand(statusAdvanceCmd)
	statusCmd.AddCommand(statusTransitionsCmd)
	statusCmd.AddCommand(statusHistoryCmd)
}

// --- Helper functions ---

// capitalizeEntityType returns the entity type with first letter capitalized.
// Used for user-facing messages (e.g., "epic" -> "Epic").
func capitalizeEntityType(entityType string) string {
	if entityType == "" {
		return ""
	}
	return strings.ToUpper(entityType[:1]) + entityType[1:]
}

// dispatchTransition routes a transition request to the correct service based on entity type.
func dispatchTransition(ctx context.Context, entityType, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	switch entityType {
	case "epic":
		return cli.GetEpicService().TransitionStatus(ctx, key, targetStatus, opts)
	case "feature":
		return cli.GetFeatureService().TransitionStatus(ctx, key, targetStatus, opts)
	case "task":
		return cli.GetTaskService().TransitionStatus(ctx, key, targetStatus, opts)
	case "bug":
		bug, err := cli.GetBugService().SetBugStatus(ctx, key, targetStatus, opts.Force)
		if err != nil {
			return nil, err
		}
		return &services.TransitionResult{
			EntityType:   "bug",
			EntityKey:    bug.Key,
			ToStatus:     string(bug.Status),
			Transitioned: true,
			IsForced:     opts.Force,
			Reason:       opts.Reason,
		}, nil
	case "change_card":
		card, err := cli.GetChangeCardService().SetChangeCardStatus(ctx, key, targetStatus)
		if err != nil {
			return nil, err
		}
		return &services.TransitionResult{
			EntityType:   "change_card",
			EntityKey:    card.Key,
			ToStatus:     string(card.Status),
			Transitioned: true,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported entity type: %s", entityType)
	}
}

// dispatchNextStatus routes a next-status request to the correct service based on entity type.
func dispatchNextStatus(ctx context.Context, entityType, key string) (*services.NextStatusInfo, error) {
	switch entityType {
	case "epic":
		return cli.GetEpicService().GetNextStatus(ctx, key)
	case "feature":
		return cli.GetFeatureService().GetNextStatus(ctx, key)
	case "task":
		return cli.GetTaskService().GetNextStatus(ctx, key)
	case "bug":
		bug, err := cli.GetBugService().GetBug(ctx, key)
		if err != nil {
			return nil, err
		}
		return &services.NextStatusInfo{
			EntityType:           "bug",
			EntityKey:            bug.Key,
			CurrentStatus:        string(bug.Status),
			AvailableTransitions: nil,
		}, nil
	case "change_card":
		card, err := cli.GetChangeCardService().GetChangeCard(ctx, key)
		if err != nil {
			return nil, err
		}
		return &services.NextStatusInfo{
			EntityType:           "change_card",
			EntityKey:            card.Key,
			CurrentStatus:        string(card.Status),
			AvailableTransitions: nil,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported entity type: %s", entityType)
	}
}

// handleStatusTransitionError handles common error patterns for status transition commands.
// Calls os.Exit for known error types (not found, reason required).
func handleStatusTransitionError(entityType, key string, err error) {
	if errors.Is(err, services.ErrReasonRequired) || errors.Is(err, services.ErrForceReasonRequired) {
		cli.Error(err.Error())
		os.Exit(3)
	}
	if strings.Contains(err.Error(), "not found") {
		cli.Error(fmt.Sprintf("%s %s not found", capitalizeEntityType(entityType), key))
		os.Exit(1)
	}
}

// --- Command implementations ---

// runStatusSet implements the `shark status set <key> <status>` command.
func runStatusSet(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Step 1: Parse arguments
	entityType, key, err := ParseGetArgs(args[:1])
	if err != nil {
		return fmt.Errorf("invalid key format: %w", err)
	}

	targetStatus := strings.ToLower(strings.TrimSpace(args[1]))
	if targetStatus == "" {
		return fmt.Errorf("target status cannot be empty")
	}

	// Parse flags
	reason, _ := cmd.Flags().GetString("reason")
	force, _ := cmd.Flags().GetBool("force")

	opts := services.TransitionOptions{
		Force:  force,
		Reason: reason,
	}

	// Step 2: Check idempotency - if already at target status, return success with changed=false
	info, err := dispatchNextStatus(ctx, entityType, key)
	if err != nil {
		handleStatusTransitionError(entityType, key, err)
		return fmt.Errorf("failed to get %s status: %w", entityType, err)
	}

	if strings.EqualFold(info.CurrentStatus, targetStatus) {
		result := &StatusSetResult{
			EntityType: entityType,
			EntityKey:  info.EntityKey,
			Status:     info.CurrentStatus,
			Changed:    false,
			Message:    fmt.Sprintf("Entity already at status '%s'", info.CurrentStatus),
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}

		cli.Success(result.Message)
		return nil
	}

	// Step 3: Perform transition
	transResult, err := dispatchTransition(ctx, entityType, key, targetStatus, opts)
	if err != nil {
		handleStatusTransitionError(entityType, key, err)
		return fmt.Errorf("failed to set %s status: %w", entityType, err)
	}

	// Step 4: Format output
	if cli.GlobalConfig.JSON {
		result := &StatusSetResult{
			EntityType:         entityType,
			EntityKey:          transResult.EntityKey,
			Status:             transResult.ToStatus,
			Changed:            true,
			Message:            fmt.Sprintf("%s -> %s", transResult.FromStatus, transResult.ToStatus),
			PreviousStatus:     transResult.FromStatus,
			IsBackward:         transResult.IsBackward,
			IsForced:           transResult.IsForced,
			Reason:             transResult.Reason,
			ChildCount:         transResult.ChildCount,
			OrchestratorAction: transResult.OrchestratorAction,
		}
		return cli.OutputJSON(result)
	}

	cli.Success(fmt.Sprintf("%s %s: %s -> %s", capitalizeEntityType(entityType), transResult.EntityKey, transResult.FromStatus, transResult.ToStatus))
	if transResult.IsBackward && transResult.Reason != "" {
		cli.Info(fmt.Sprintf("Reason: %s", transResult.Reason))
	}
	if transResult.IsForced {
		cli.Warning("Workflow validation was bypassed with --force")
	}
	if transResult.ChildCount > 0 {
		cli.Warning(fmt.Sprintf("%d child entities remain in current states.", transResult.ChildCount))
	}
	displayOrchestratorAction(transResult.OrchestratorAction)
	return nil
}

// dispatchAdvance routes an advance request for entity types that have native advance methods
// (bug, change_card). For epic/feature/task, returns (nil, nil) to fall through to the
// generic info-based advance path.
func dispatchAdvance(ctx context.Context, entityType, key string) (*services.TransitionResult, error) {
	switch entityType {
	case "bug":
		bug, err := cli.GetBugService().AdvanceBugStatus(ctx, key)
		if err != nil {
			return nil, err
		}
		return &services.TransitionResult{
			EntityType:   "bug",
			EntityKey:    bug.Key,
			ToStatus:     string(bug.Status),
			Transitioned: true,
		}, nil
	case "change_card":
		card, err := cli.GetChangeCardService().AdvanceChangeCardStatus(ctx, key)
		if err != nil {
			return nil, err
		}
		return &services.TransitionResult{
			EntityType:   "change_card",
			EntityKey:    card.Key,
			ToStatus:     string(card.Status),
			Transitioned: true,
		}, nil
	default:
		// Fall through to the generic info-based path for epic/feature/task.
		return nil, nil
	}
}

// runStatusAdvance implements the `shark status advance <key>` command.
func runStatusAdvance(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Step 1: Parse arguments
	entityType, key, err := ParseGetArgs(args)
	if err != nil {
		return fmt.Errorf("invalid key format: %w", err)
	}

	// Step 2: For bug/change_card, use native advance methods directly.
	if transResult, err := dispatchAdvance(ctx, entityType, key); err != nil {
		handleStatusTransitionError(entityType, key, err)
		return fmt.Errorf("failed to advance %s status: %w", entityType, err)
	} else if transResult != nil {
		if cli.GlobalConfig.JSON {
			result := &StatusSetResult{
				EntityType: entityType,
				EntityKey:  transResult.EntityKey,
				Status:     transResult.ToStatus,
				Changed:    true,
				Message:    fmt.Sprintf("-> %s", transResult.ToStatus),
			}
			return cli.OutputJSON(result)
		}
		cli.Success(fmt.Sprintf("%s %s advanced to %s", capitalizeEntityType(entityType), transResult.EntityKey, transResult.ToStatus))
		return nil
	}

	// Step 3: Generic path for epic/feature/task — get current status info
	info, err := dispatchNextStatus(ctx, entityType, key)
	if err != nil {
		handleStatusTransitionError(entityType, key, err)
		return fmt.Errorf("failed to get %s status: %w", entityType, err)
	}

	// Build result for JSON output
	result := buildNextStatusResult(entityType, info)

	// Handle terminal status
	if info.IsTerminal {
		result.Message = fmt.Sprintf("%s is in terminal status '%s' - no transitions available", capitalizeEntityType(entityType), info.CurrentStatus)

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}

		cli.Warning(result.Message)
		return nil
	}

	// Handle no transitions
	if len(info.AvailableTransitions) == 0 {
		result.Message = fmt.Sprintf("No valid transitions from status '%s'", info.CurrentStatus)

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}

		cli.Warning(result.Message)
		return nil
	}

	// Auto-select first (default) transition
	autoTarget := info.AvailableTransitions[0].TargetStatus

	// Create adapter to satisfy entityTransitioner interface
	svc := entityTransitionerFunc(func(ctx context.Context, k string, ts string, opts services.TransitionOptions) (*services.TransitionResult, error) {
		return dispatchTransition(ctx, entityType, k, ts, opts)
	})

	opts := services.TransitionOptions{}
	return performEntityTransition(ctx, svc, info.EntityKey, autoTarget, opts, result)
}

// runStatusTransitions implements the `shark status transitions <key>` command.
func runStatusTransitions(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Step 1: Parse arguments
	entityType, key, err := ParseGetArgs(args)
	if err != nil {
		return fmt.Errorf("invalid key format: %w", err)
	}

	// Step 2: Get current status info
	info, err := dispatchNextStatus(ctx, entityType, key)
	if err != nil {
		handleStatusTransitionError(entityType, key, err)
		return fmt.Errorf("failed to get %s status: %w", entityType, err)
	}

	// Step 3: Build result (read-only, never transitioned)
	result := buildNextStatusResult(entityType, info)
	result.Transitioned = false

	if info.IsTerminal {
		result.Message = fmt.Sprintf("%s is in terminal status '%s' - no transitions available", capitalizeEntityType(entityType), info.CurrentStatus)
	} else if len(info.AvailableTransitions) == 0 {
		result.Message = fmt.Sprintf("No valid transitions from status '%s'", info.CurrentStatus)
	}

	// Step 4: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}

	fmt.Printf("\n%s: %s\n", capitalizeEntityType(entityType), info.EntityKey)
	fmt.Printf("Current status: %s", info.CurrentStatus)
	if info.CurrentPhase != "" {
		fmt.Printf(" (phase: %s)", info.CurrentPhase)
	}
	fmt.Println()

	if info.IsTerminal {
		fmt.Println()
		cli.Warning("Terminal status - no transitions available")
		return nil
	}

	if len(info.AvailableTransitions) == 0 {
		fmt.Println()
		cli.Warning("No valid transitions from current status")
		return nil
	}

	fmt.Println()
	fmt.Println("Available transitions:")
	printEntityTransitions(result.AvailableTransitions)
	fmt.Println()
	fmt.Printf("Use 'shark status set %s <status>' or 'shark status advance %s' to transition\n", info.EntityKey, info.EntityKey)
	return nil
}

// runStatusHistory implements the `shark status history <key>` command.
func runStatusHistory(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Step 1: Parse arguments
	entityType, key, err := ParseGetArgs(args)
	if err != nil {
		return fmt.Errorf("invalid key format: %w", err)
	}

	// Step 2: Validate entity type - only tasks have history
	if entityType != "task" {
		cli.Error(fmt.Sprintf("Status history is only available for tasks, not %ss", entityType))
		cli.Info("Use 'shark status transitions <key>' to see available transitions for any entity type")
		os.Exit(3)
	}

	limit, _ := cmd.Flags().GetInt("limit")

	// Step 3: Get task history
	taskSvc := cli.GetTaskServiceWithHistory()
	history, err := taskSvc.GetTaskHistory(ctx, key)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			cli.Error(fmt.Sprintf("Task %s not found", key))
			os.Exit(1)
		}
		return fmt.Errorf("failed to get task history: %w", err)
	}

	// Apply limit
	if limit > 0 && len(history) > limit {
		history = history[len(history)-limit:]
	}

	// Step 4: Build result
	entries := make([]StatusHistoryEntry, 0, len(history))
	for _, h := range history {
		entry := StatusHistoryEntry{
			Timestamp: h.Timestamp.Format(time.RFC3339),
			NewStatus: h.NewStatus,
		}
		if h.OldStatus != nil {
			entry.OldStatus = *h.OldStatus
		}
		if h.Agent != nil {
			entry.Agent = *h.Agent
		}
		if h.Notes != nil {
			entry.Notes = *h.Notes
		}
		entries = append(entries, entry)
	}

	result := &StatusHistoryResult{
		EntityType: entityType,
		EntityKey:  key,
		History:    entries,
		Total:      len(entries),
	}

	// Step 5: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}

	if len(entries) == 0 {
		cli.Info(fmt.Sprintf("No status history found for task %s", key))
		return nil
	}

	fmt.Printf("\nStatus History for %s (%d entries)\n", key, len(entries))
	fmt.Println(strings.Repeat("-", 80))

	headers := []string{"Timestamp", "Old Status", "New Status", "Agent", "Notes"}
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		oldStatus := e.OldStatus
		if oldStatus == "" {
			oldStatus = "(initial)"
		}
		rows = append(rows, []string{
			e.Timestamp,
			oldStatus,
			e.NewStatus,
			e.Agent,
			truncateString(e.Notes, 40),
		})
	}

	cli.OutputTable(headers, rows)
	return nil
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
