package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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

// statusHistoryCmd shows entity status change history.
var statusHistoryCmd = &cobra.Command{
	Use:   "history <key>",
	Short: "Show entity status change history",
	Long: `Show the status change history for any entity type. Entity type is auto-detected
from the key format.

Key Formats:
  E07                Epic
  E07-F01            Feature
  E07-F01-001        Task (short format)
  T-E07-F01-001      Task (traditional format)
  B001               Bug
  CC-001             Change Card

Examples:
  shark status history E07                      Show epic history
  shark status history E07-F01                  Show feature history
  shark status history E07-F01-001              Show task history
  shark status history B001                     Show bug history
  shark status history CC-001                   Show change-card history
  shark status history E07-F01 --limit=10       Show last 10 changes
  shark status history E07-F01-001 --json       JSON output`,
	Args: cobra.ExactArgs(1),
	RunE: runStatusHistory,
}

func init() {
	// statusSetCmd flags
	statusSetCmd.Flags().String("reason", "", "Reason for backward or forced transitions")
	statusSetCmd.Flags().Bool("force", false, "Bypass workflow validation")
	statusSetCmd.Flags().String("notes", "", "Transition notes")

	// statusAdvanceCmd: route-based release. Without --outcome it advances to the
	// default next status (legacy behavior). With --outcome it resolves
	// step.outcomes[outcome] and routes there (E35-F02).
	statusAdvanceCmd.Flags().String("outcome", "", "Release a semantic outcome (pass|fail|blocked|…) and route via the workflow's outcomes map")
	statusAdvanceCmd.Flags().String("reason", "", "Reason recorded with the transition")

	// statusHistoryCmd flags
	statusHistoryCmd.Flags().Int("limit", 50, "Maximum number of history entries to show")

	// Register subcommands under statusCmd (defined in status.go)
	statusCmd.AddCommand(statusSetCmd)
	statusCmd.AddCommand(statusAdvanceCmd)
	statusCmd.AddCommand(statusTransitionsCmd)
	statusCmd.AddCommand(statusHistoryCmd)
}

// --- Helper functions ---

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
		return cli.GetBugService().TransitionStatus(ctx, key, targetStatus, opts)
	case "change", "change_card":
		return getChangeCardService().TransitionStatus(ctx, key, targetStatus, opts)
	case "tech_debt":
		return cli.GetTechDebtService().TransitionStatus(ctx, key, targetStatus, opts)
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
		return cli.GetBugService().GetNextStatus(ctx, key)
	case "change", "change_card":
		return getChangeCardService().GetNextStatus(ctx, key)
	case "tech_debt":
		return cli.GetTechDebtService().GetNextStatus(ctx, key)
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
		cli.Error(fmt.Sprintf("%s %s not found", displayEntityTypeName(entityType), key))
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

	// Step 2: Perform transition (idempotency is handled in the service layer)
	transResult, err := dispatchTransition(ctx, entityType, key, targetStatus, opts)
	if err != nil {
		handleStatusTransitionError(entityType, key, err)
		return fmt.Errorf("failed to set %s status: %w", entityType, err)
	}

	// Step 3: Format output — check Transitioned to decide message
	if !transResult.Transitioned {
		result := &StatusSetResult{
			EntityType: entityType,
			EntityKey:  transResult.EntityKey,
			Status:     transResult.ToStatus,
			Changed:    false,
			Message:    fmt.Sprintf("Entity already at status '%s'", transResult.ToStatus),
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}

		cli.Success(result.Message)
		return nil
	}
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

	cli.Success(fmt.Sprintf("%s %s: %s -> %s", displayEntityTypeName(entityType), transResult.EntityKey, transResult.FromStatus, transResult.ToStatus))
	if transResult.IsBackward && transResult.Reason != "" {
		cli.Info(fmt.Sprintf("Reason: %s", transResult.Reason))
	}
	if transResult.IsForced {
		cli.Warning("Workflow validation was bypassed with --force")
	}
	if transResult.ChildCount > 0 {
		cli.Warning(fmt.Sprintf("%d child entities remain in current states.", transResult.ChildCount))
	}
	cli.Info(fmt.Sprintf("Run `shark get %s --field orchestrator_action` to get your next instructions.", transResult.EntityKey))
	return nil
}

// runStatusAdvance implements the `shark status advance <key>` command.
func runStatusAdvance(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start OTel span for this advance operation. The tracer returns a no-op
	// span when OTel is disabled, so no guard is needed.
	tracer := cli.GetTracer("shark.cli")
	ctx, span := tracer.Start(ctx, "shark.advance")
	defer span.End()

	// Step 1: Parse arguments
	entityType, key, err := ParseGetArgs(args)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("invalid key format: %w", err)
	}

	// Step 2: Get current status info
	info, err := dispatchNextStatus(ctx, entityType, key)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		handleStatusTransitionError(entityType, key, err)
		return fmt.Errorf("failed to get %s status: %w", entityType, err)
	}

	// Capture from_status before the transition.
	fromStatus := info.CurrentStatus
	entityKey := info.EntityKey

	// Build result for JSON output
	result := buildNextStatusResult(entityType, info)

	// Handle terminal status — no transition happens; still record span.
	if info.IsTerminal {
		span.SetAttributes(
			attribute.String("entity_key", entityKey),
			attribute.String("entity_type", entityType),
			attribute.String("from_status", fromStatus),
			attribute.String("to_status", fromStatus), // no transition
			attribute.String("actor", advanceActor()),
			attribute.Bool("forced", false),
			attribute.Bool("had_rejection_note", false),
		)
		result.Message = fmt.Sprintf("%s is in terminal status '%s' - no transitions available", displayEntityTypeName(entityType), info.CurrentStatus)

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}

		cli.Warning(result.Message)
		return nil
	}

	// Handle no transitions — still record span.
	if len(info.AvailableTransitions) == 0 {
		span.SetAttributes(
			attribute.String("entity_key", entityKey),
			attribute.String("entity_type", entityType),
			attribute.String("from_status", fromStatus),
			attribute.String("to_status", fromStatus), // no transition
			attribute.String("actor", advanceActor()),
			attribute.Bool("forced", false),
			attribute.Bool("had_rejection_note", false),
		)
		result.Message = fmt.Sprintf("No valid transitions from status '%s'", info.CurrentStatus)

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}

		cli.Warning(result.Message)
		return nil
	}

	// Determine the target. With --outcome, resolve via the route-based outcomes
	// map (release semantics, D4). Without it, auto-select the default transition.
	// cmd may be nil in unit tests that call runStatusAdvance directly.
	var outcome, reason string
	if cmd != nil {
		outcome, _ = cmd.Flags().GetString("outcome")
		reason, _ = cmd.Flags().GetString("reason")
	}

	autoTarget := info.AvailableTransitions[0].TargetStatus
	opts := services.TransitionOptions{Reason: reason}

	if strings.TrimSpace(outcome) != "" {
		outcome = strings.TrimSpace(strings.ToLower(outcome))
		target, ok := info.Outcomes[outcome]
		if !ok {
			// Case-insensitive match against defined outcomes.
			for k, v := range info.Outcomes {
				if strings.EqualFold(k, outcome) {
					target, ok = v, true
					break
				}
			}
		}
		if !ok {
			span.SetStatus(codes.Error, "unknown outcome")
			if len(info.Outcomes) == 0 {
				cli.Error(fmt.Sprintf("Status '%s' has no outcomes (route-based workflow required, or terminal/parking step). Use 'shark status set' to target a status directly.", fromStatus))
			} else {
				cli.Error(fmt.Sprintf("Unknown outcome '%s' for status '%s'. Valid outcomes: %s", outcome, fromStatus, strings.Join(sortedOutcomeKeys(info.Outcomes), ", ")))
			}
			return fmt.Errorf("unknown outcome %q", outcome)
		}
		autoTarget = target
		// The resolved route is authoritative; record the outcome as the reason
		// so backward routes (e.g. fail) pass the backward-transition guard
		// without requiring --force.
		if opts.Reason == "" {
			opts.Reason = "release outcome: " + outcome
		}
	}

	// Check whether the entity has a rejection note before transitioning.
	hadRejectionNote := checkRejectionNote(ctx, entityType, entityKey)

	// Set span attributes now that we have all values.
	span.SetAttributes(
		attribute.String("entity_key", entityKey),
		attribute.String("entity_type", entityType),
		attribute.String("from_status", fromStatus),
		attribute.String("to_status", autoTarget),
		attribute.String("actor", advanceActor()),
		attribute.Bool("forced", false), // statusAdvanceCmd has no --force flag
		attribute.Bool("had_rejection_note", hadRejectionNote),
	)
	if outcome != "" {
		span.SetAttributes(attribute.String("outcome", outcome))
	}

	// Create adapter to satisfy entityTransitioner interface
	svc := entityTransitionerFunc(func(ctx context.Context, k string, ts string, opts services.TransitionOptions) (*services.TransitionResult, error) {
		return dispatchTransition(ctx, entityType, k, ts, opts)
	})

	if err := performEntityTransition(ctx, svc, entityKey, autoTarget, opts, result); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

// sortedOutcomeKeys returns the outcome names sorted for stable error output.
func sortedOutcomeKeys(outcomes map[string]string) []string {
	keys := make([]string, 0, len(outcomes))
	for k := range outcomes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// advanceActor returns the actor identity for the shark.advance span.
// It reads the SHARK_ACTOR environment variable, defaulting to "cli".
func advanceActor() string {
	if actor := os.Getenv("SHARK_ACTOR"); actor != "" {
		return actor
	}
	return "cli"
}

// checkRejectionNote returns true when the entity has at least one note of
// type "rejection". It is fail-soft: any error (DB unavailable, entity not
// found, etc.) returns false so the span is never blocked.
func checkRejectionNote(ctx context.Context, entityType, entityKey string) bool {
	// Only epic / feature / task entity types carry EntityNote records.
	// For bugs, change-cards, tech-debt, etc., fall back to false.
	var modelType models.EntityType
	switch entityType {
	case "epic":
		modelType = models.EntityTypeEpic
	case "feature":
		modelType = models.EntityTypeFeature
	case "task":
		modelType = models.EntityTypeTask
	default:
		return false
	}

	noteSvc, err := cli.GetNoteService(ctx)
	if err != nil {
		return false
	}

	notes, err := noteSvc.ListNotes(ctx, modelType, entityKey, []string{"rejection"})
	if err != nil {
		return false
	}
	return len(notes) > 0
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
		result.Message = fmt.Sprintf("%s is in terminal status '%s' - no transitions available", displayEntityTypeName(entityType), info.CurrentStatus)
	} else if len(info.AvailableTransitions) == 0 {
		result.Message = fmt.Sprintf("No valid transitions from status '%s'", info.CurrentStatus)
	}

	// Step 4: Format output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}

	fmt.Printf("\n%s: %s\n", displayEntityTypeName(entityType), info.EntityKey)
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

	// Step 2: Normalize entity type (ADR-1: change_card -> change)
	if entityType == "change_card" {
		entityType = "change"
	}

	limit, _ := cmd.Flags().GetInt("limit")

	// Step 3: Get history via EntityHistoryService
	svc := cli.GetEntityHistoryService()
	history, err := svc.GetHistory(ctx, models.EntityType(entityType), key)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			cli.Error(fmt.Sprintf("%s %s not found", displayEntityTypeName(entityType), key))
			os.Exit(1)
		}
		return fmt.Errorf("failed to get status history: %w", err)
	}

	// Step 4: Apply limit (keep most recent entries; history is ordered DESC)
	if limit > 0 && len(history) > limit {
		history = history[:limit]
	}

	entries := make([]StatusHistoryEntry, 0, len(history))
	for _, h := range history {
		entry := StatusHistoryEntry{
			Timestamp: h.ChangedAt.Format(time.RFC3339),
			NewStatus: h.ToStatus,
		}
		if h.FromStatus != nil {
			entry.OldStatus = *h.FromStatus
		}
		if h.ChangedBy != nil {
			entry.Agent = *h.ChangedBy
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
		cli.Info(fmt.Sprintf("No status history found for %s %s", displayEntityTypeName(entityType), key))
		return nil
	}

	fmt.Printf("\nStatus History for %s %s (%d entries)\n", displayEntityTypeName(entityType), key, len(entries))
	fmt.Println(strings.Repeat("-", 80))

	headers := []string{"Timestamp", "Old Status", "New Status", "Agent", "Notes"}
	const notesColIdx = 4

	notesText := make([]string, 0, len(entries))
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		oldStatus := e.OldStatus
		if oldStatus == "" {
			oldStatus = "(initial)"
		}
		notesText = append(notesText, formatHistoryNotesForDisplay(e.Notes))
		rows = append(rows, []string{
			e.Timestamp,
			oldStatus,
			e.NewStatus,
			e.Agent,
			"",
		})
	}

	notesMax := availableTitleWidth(cli.GetConsoleWidth(), headers, rows, notesColIdx)
	for i := range rows {
		rows[i][notesColIdx] = truncateToWidth(notesText[i], notesMax)
	}

	cli.OutputTable(headers, rows)
	return nil
}

// formatHistoryNotesForDisplay returns the notes string for human-readable table output.
// If the notes start with the "auto_reopen:" prefix, a bracketed "[auto-reopen]" label
// is appended so operators can visually distinguish automated cascade-reopens from
// manually authored notes.  The raw notes value stored in JSON output is unchanged.
func formatHistoryNotesForDisplay(notes string) string {
	if strings.HasPrefix(notes, "auto_reopen:") {
		return notes + " [auto-reopen]"
	}
	return notes
}
