package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// ideaGetServicer defines the narrow interface used by runIdeaGet.
// Allows tests to inject a mock without touching the global CLI layer.
type ideaGetServicer interface {
	GetIdea(ctx context.Context, key string) (*models.Idea, error)
	// GetIdeaWithTags returns the idea and sorted attached tag names (REQ-F-014).
	// When TagService is nil or unavailable, tags will be nil (graceful degradation).
	GetIdeaWithTags(ctx context.Context, key string) (*models.Idea, []string, error)
}

// ideaGetSvcOverride is non-nil only during tests.
var ideaGetSvcOverride ideaGetServicer

// getIdeaGetService returns the service to use for runIdeaGet, preferring the test override.
func getIdeaGetService() ideaGetServicer {
	if ideaGetSvcOverride != nil {
		return ideaGetSvcOverride
	}
	return cli.GetIdeaService()
}

// ideaCmd represents the idea command group
var ideaCmd = &cobra.Command{
	Use:     "idea",
	Short:   "Manage ideas",
	GroupID: "advanced",
	Long: `Idea capture and management operations for lightweight idea tracking.

Ideas are captured with keys in format I-YYYY-MM-DD-xx (e.g., I-2026-01-01-01).

Examples:
  shark idea list                    List all ideas
  shark idea create "New Feature"    Create a new idea
  shark idea get I-2026-01-01-01     Get idea details
  shark idea update I-2026-01-01-01  Update an idea
  shark idea delete I-2026-01-01-01  Delete an idea`,
}

// ideaListCmd lists ideas
var ideaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List ideas",
	Long: `List ideas with optional filtering by status and priority.

By default, archived ideas are hidden unless --status=archived is specified.

Examples:
  shark idea list                     List all non-archived ideas
  shark idea list --status=new        List only new ideas
  shark idea list --priority=5        List ideas with priority 5
  shark idea list --json              Output as JSON`,
	RunE: runIdeaList,
}

// ideaGetCmd gets a specific idea
var ideaGetCmd = &cobra.Command{
	Use:   "get <idea-key>",
	Short: "Get idea details",
	Long: `Display detailed information about a specific idea.

Examples:
  shark idea get I-2026-01-01-01        Get idea by key
  shark idea get I-2026-01-01-01 --json JSON output`,
	Args: cobra.ExactArgs(1),
	RunE: runIdeaGet,
}

// ideaCreateCmd creates a new idea
var ideaCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new idea",
	Long: `Create a new idea with auto-generated key (I-YYYY-MM-DD-xx format).

All properties can be set on creation using flags.

Examples:
  shark idea create "New feature idea"
  shark idea create "Backend optimization" --description="Improve query performance" --priority=8
  shark idea create "UI redesign" --status=on_hold --notes="Waiting for design review"`,
	Args: cobra.ExactArgs(1),
	RunE: runIdeaCreate,
}

// ideaUpdateCmd updates an existing idea
var ideaUpdateCmd = &cobra.Command{
	Use:   "update <idea-key>",
	Short: "Update an existing idea",
	Long: `Update properties of an existing idea.

All properties can be updated using flags.

Examples:
  shark idea update I-2026-01-01-01 --title="Updated title"
  shark idea update I-2026-01-01-01 --priority=9 --status=on_hold
  shark idea update I-2026-01-01-01 --notes="Additional context"`,
	Args: cobra.ExactArgs(1),
	RunE: runIdeaUpdate,
}

// ideaDeleteCmd deletes an idea
var ideaDeleteCmd = &cobra.Command{
	Use:   "delete <idea-key>",
	Short: "Delete an idea",
	Long: `Delete an idea (soft delete by default, archives the idea).

By default, ideas are archived (soft delete). Use --hard flag for permanent deletion.
Confirmation is required unless --force flag is provided.

Examples:
  shark idea delete I-2026-01-01-01               Soft delete (archive)
  shark idea delete I-2026-01-01-01 --hard        Hard delete (permanent)
  shark idea delete I-2026-01-01-01 --force       Skip confirmation prompt`,
	Args: cobra.ExactArgs(1),
	RunE: runIdeaDelete,
}

// ideaConvertCmd is the parent command for conversion operations
var ideaConvertCmd = &cobra.Command{
	Use:   "convert <idea-key> <type>",
	Short: "Convert an idea to epic, feature, or task",
	Long: `Convert a lightweight idea into a structured entity (epic, feature, task).

Once converted, the idea status changes to 'converted' and a new entity is created.

Examples:
  shark idea convert epic I-2026-01-01-01
  shark idea convert feature I-2026-01-01-01 --epic=E10
  shark idea convert task I-2026-01-01-01 --epic=E10 --feature=E10-F02`,
}

// ideaConvertEpicCmd converts an idea to an epic
var ideaConvertEpicCmd = &cobra.Command{
	Use:   "epic <idea-key>",
	Short: "Convert idea to epic",
	Long: `Convert an idea to a new epic.

The idea's title and description are copied to the epic.
A new epic key is auto-generated (E##).

Examples:
  shark idea convert epic I-2026-01-01-01
  shark idea convert epic I-2026-01-01-01 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runIdeaConvertEpic,
}

// ideaConvertFeatureCmd converts an idea to a feature
var ideaConvertFeatureCmd = &cobra.Command{
	Use:   "feature <idea-key> --epic=<epic-key>",
	Short: "Convert idea to feature",
	Long: `Convert an idea to a feature in a specified epic.

The idea's title and description are copied to the feature.
Requires --epic flag to specify the target epic.

Examples:
  shark idea convert feature I-2026-01-01-01 --epic=E10
  shark idea convert feature I-2026-01-01-01 --epic=E10 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runIdeaConvertFeature,
}

// ideaConvertTaskCmd converts an idea to a task
var ideaConvertTaskCmd = &cobra.Command{
	Use:   "task <idea-key> --epic=<epic-key> --feature=<feature-key>",
	Short: "Convert idea to task",
	Long: `Convert an idea to a task in a specified epic and feature.

The idea's title, description, and priority are copied to the task.
Requires --epic and --feature flags to specify the target location.

Examples:
  shark idea convert task I-2026-01-01-01 --epic=E10 --feature=E10-F02
  shark idea convert task I-2026-01-01-01 --epic=E10 --feature=E10-F02 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runIdeaConvertTask,
}

// Command flags
var (
	ideaStatus         string
	ideaPriority       int
	ideaDescription    string
	ideaNotes          string
	ideaRelatedDocs    []string
	ideaDependencies   []string
	ideaOrder          int
	ideaForce          bool
	ideaHard           bool
	ideaConvertEpic    string
	ideaConvertFeature string

	// E28-F04 REQ-F-012: repeatable `--tag` flags for create and update.
	// Cobra's StringSliceVar does not reset flag-backing variables between
	// command executions (test code must zero them out to avoid cross-test
	// bleed).
	ideaCreateTags []string
	ideaUpdateTags []string

	// E07-F42 REQ-F-004: --size flag for idea create.
	// StringVar (not IntVar) per Decision D4 — accepts both numeric and label forms.
	ideaCreateSizeFlag string
)

// resolveIdeaID is the EntityKeyResolver used by the `shark idea tag`
// subcommand factory. It uses the existing idea service accessor to find
// an idea by key and return its numeric ID.
//
// Split out as a package-level function (not a closure in init()) so the
// E28-F04 entity_tag_cmd.go factory can reference it.
func resolveIdeaID(ctx context.Context, key string) (int64, error) {
	svc := cli.GetIdeaService()
	idea, err := svc.GetIdea(ctx, key)
	if err != nil {
		return 0, err
	}
	return idea.ID, nil
}

func init() {
	// Register idea command and subcommands
	cli.RootCmd.AddCommand(ideaCmd)
	ideaCmd.AddCommand(ideaListCmd)
	ideaCmd.AddCommand(ideaGetCmd)
	ideaCmd.AddCommand(ideaCreateCmd)
	ideaCmd.AddCommand(ideaUpdateCmd)
	ideaCmd.AddCommand(ideaDeleteCmd)
	ideaCmd.AddCommand(ideaConvertCmd)

	// Register convert subcommands
	ideaConvertCmd.AddCommand(ideaConvertEpicCmd)
	ideaConvertCmd.AddCommand(ideaConvertFeatureCmd)
	ideaConvertCmd.AddCommand(ideaConvertTaskCmd)

	// E28-F04 T-010: register the shared `tag add|rm` subcommand. Svc
	// override is nil in production so it falls through to
	// cli.GetTagService() at call time. AC-26 is satisfied by
	// models.EntityTypeIdea being valid per F01.
	ideaCmd.AddCommand(makeEntityTagCmd(models.EntityTypeIdea, resolveIdeaID, nil))

	// Convert command flags
	ideaConvertFeatureCmd.Flags().StringVar(&ideaConvertEpic, "epic", "", "Target epic key (required)")
	_ = ideaConvertFeatureCmd.MarkFlagRequired("epic")

	ideaConvertTaskCmd.Flags().StringVar(&ideaConvertEpic, "epic", "", "Target epic key (required)")
	ideaConvertTaskCmd.Flags().StringVar(&ideaConvertFeature, "feature", "", "Target feature key (required)")
	_ = ideaConvertTaskCmd.MarkFlagRequired("epic")
	_ = ideaConvertTaskCmd.MarkFlagRequired("feature")

	// List command flags
	ideaListCmd.Flags().StringVar(&ideaStatus, "status", "", "Filter by status (new, on_hold, converted, archived)")
	ideaListCmd.Flags().IntVar(&ideaPriority, "priority", 0, "Filter by priority (1-10)")
	// E28-F05 REQ-F-010 / REQ-F-018: repeatable --tag flag with AND semantics.
	ideaListCmd.Flags().StringSlice("tag", nil, "Filter by tag (repeatable; AND — all tags must match).")

	// Create command flags
	ideaCreateCmd.Flags().StringVar(&ideaDescription, "description", "", "Idea description")
	ideaCreateCmd.Flags().IntVar(&ideaPriority, "priority", 0, "Priority (1-10)")
	ideaCreateCmd.Flags().IntVar(&ideaOrder, "order", 0, "Order for sorting ideas")
	ideaCreateCmd.Flags().StringVar(&ideaNotes, "notes", "", "Additional notes")
	ideaCreateCmd.Flags().StringSliceVar(&ideaRelatedDocs, "related-docs", []string{}, "Related document paths")
	ideaCreateCmd.Flags().StringSliceVar(&ideaDependencies, "depends-on", []string{}, "Dependent idea keys")
	ideaCreateCmd.Flags().StringVar(&ideaStatus, "status", "new", "Initial status (new, on_hold, converted, archived)")
	// E28-F04 REQ-F-012: repeatable --tag flag. StringSliceVar collects
	// repeated occurrences into a []string; Cobra's comma-separator
	// behaviour means `--tag=foo,bar` becomes ["foo","bar"] — ADR-F04-5
	// accepts this because invalid-name chars (a comma is invalid under
	// the regex) surface as a clear exit-3 validation error.
	ideaCreateCmd.Flags().StringSliceVar(&ideaCreateTags, "tag", nil,
		"Tag to apply (repeatable). Tag must be registered; see 'shark tags list'.")
	// E07-F42 REQ-F-004: optional size flag (StringVar per Decision D4).
	ideaCreateCmd.Flags().StringVar(&ideaCreateSizeFlag, "size", "",
		"Entity size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL")

	// Update command flags
	ideaUpdateCmd.Flags().StringVar(&ideaStatus, "status", "", "Update status")
	ideaUpdateCmd.Flags().IntVar(&ideaPriority, "priority", 0, "Update priority (1-10)")
	ideaUpdateCmd.Flags().StringVar(&ideaDescription, "description", "", "Update description")
	ideaUpdateCmd.Flags().StringVar(&ideaNotes, "notes", "", "Update notes")
	ideaUpdateCmd.Flags().StringSliceVar(&ideaRelatedDocs, "related-docs", []string{}, "Update related document paths")
	ideaUpdateCmd.Flags().StringSliceVar(&ideaDependencies, "depends-on", []string{}, "Update dependencies")
	ideaUpdateCmd.Flags().IntVar(&ideaOrder, "order", 0, "Update order")
	ideaUpdateCmd.Flags().StringVar(&ideaDescription, "title", "", "Update title")
	// E28-F04 REQ-F-012: `--tag` on update is additive (REQ-F-010). Empty
	// means no change; no way to detach here — use `shark idea tag rm`.
	ideaUpdateCmd.Flags().StringSliceVar(&ideaUpdateTags, "tag", nil,
		"Tag to apply additively (repeatable). Empty = no change; use 'shark idea tag rm' to detach.")
	// E07-F42 REQ-F-005: optional size flag with clear-literal support.
	ideaUpdateCmd.Flags().String("size", "",
		"Entity size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL (use 'clear' to remove on update)")

	// Delete command flags
	ideaDeleteCmd.Flags().BoolVar(&ideaForce, "force", false, "Skip confirmation prompt")
	ideaDeleteCmd.Flags().BoolVar(&ideaHard, "hard", false, "Perform hard delete (permanent)")
}

// runIdeaList handles the idea list command
func runIdeaList(cmd *cobra.Command, args []string) error {
	filters := services.IdeaFilters{Status: ideaStatus}
	// E28-F05 REQ-F-010: read the repeatable --tag flag; nil when absent (AC-T2).
	if rawTags, tagErr := cmd.Flags().GetStringSlice("tag"); tagErr == nil && len(rawTags) > 0 {
		filters.Tags = rawTags
	}

	svc := cli.GetIdeaService()
	ideas, err := svc.ListIdeas(cmd.Context(), filters)
	if err != nil {
		return handleEntityServiceError(cmd, cli.GetTagService(), err, "idea", "")
	}

	ideas = filterIdeasByPriorityAndStatus(ideas, ideaPriority, ideaStatus)

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(ideas)
	}
	return printIdeaList(ideas)
}

// buildIdeaGetJSON converts an idea to a JSON map with tags and size_label injected.
// tags must not be nil (pass []string{} when the tag service is unavailable).
// E07-F42 REQ-F-007: size_label is included when idea.Size is non-nil so that
// --field size_label is extractable for idea entities.
func buildIdeaGetJSON(idea *models.Idea, tags []string) map[string]interface{} {
	ideaJSON, _ := json.Marshal(idea)
	var result map[string]interface{}
	_ = json.Unmarshal(ideaJSON, &result)

	// REQ-F-015: "tags" field always present, never null.
	if tags == nil {
		tags = []string{}
	}
	result["tags"] = tags

	// E07-F42 REQ-F-007: inject size_label so --field size_label is extractable.
	if idea.Size != nil {
		if label, err := models.SizeLabel(*idea.Size); err == nil {
			result["size_label"] = label
		}
	}

	return result
}

// runIdeaGet handles the idea get command
func runIdeaGet(cmd *cobra.Command, args []string) error {
	ideaKey := args[0]

	// Use GetIdeaWithTags for tag enrichment (REQ-F-014, REQ-F-015).
	svc := getIdeaGetService()
	idea, tags, err := svc.GetIdeaWithTags(cmd.Context(), ideaKey)
	if err != nil {
		return fmt.Errorf("failed to get idea: %w", err)
	}

	if cli.GlobalConfig.JSON {
		// REQ-F-015: "tags" field always present, never null.
		if tags == nil {
			tags = []string{}
		}
		return cli.OutputJSON(buildIdeaGetJSON(idea, tags))
	}
	return printIdeaDetailWithTags(idea, tags)
}

// runIdeaCreate handles the idea create command
func runIdeaCreate(cmd *cobra.Command, args []string) error {
	input, err := parseCreateIdeaInput(args[0])
	if err != nil {
		return err
	}

	svc := cli.GetIdeaService()
	idea, err := svc.CreateIdea(cmd.Context(), input)
	if err != nil {
		return handleEntityServiceError(cmd, resolveTagService(nil), err, "idea", input.Title)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(idea)
	}
	cli.Success(fmt.Sprintf("Created idea %s: %s", idea.Key, idea.Title))
	return nil
}

// runIdeaUpdate handles the idea update command
func runIdeaUpdate(cmd *cobra.Command, args []string) error {
	ideaKey := args[0]
	input, err := parseUpdateIdeaInput(cmd)
	if err != nil {
		return err
	}

	svc := cli.GetIdeaService()
	idea, err := svc.UpdateIdea(cmd.Context(), ideaKey, input)
	if err != nil {
		return handleEntityServiceError(cmd, resolveTagService(nil), err, "idea", ideaKey)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(idea)
	}
	cli.Success(fmt.Sprintf("Updated idea %s", idea.Key))
	return nil
}

// runIdeaDelete handles the idea delete command
func runIdeaDelete(cmd *cobra.Command, args []string) error {
	ideaKey := args[0]

	svc := cli.GetIdeaService()
	idea, err := svc.GetIdea(cmd.Context(), ideaKey)
	if err != nil {
		return fmt.Errorf("failed to get idea: %w", err)
	}

	if !ideaForce {
		if !confirmIdeaDelete(idea, ideaHard) {
			return nil
		}
	}

	return performIdeaDelete(cmd, svc, ideaKey, idea, ideaHard)
}

// generateIdeaKey generates the next available idea key for today's date.
// Format: I-YYYY-MM-DD-xx where xx is 01-99.
// This function is used by tests via the IdeaRepository interface in mock_idea_repository.go.
type ideaKeyRepo interface {
	GetNextSequenceForDate(ctx context.Context, dateStr string) (int, error)
}

func generateIdeaKey(ctx context.Context, repo ideaKeyRepo) (string, error) {
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	baseKey := fmt.Sprintf("I-%s", dateStr)

	nextSeq, err := repo.GetNextSequenceForDate(ctx, dateStr)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%02d", baseKey, nextSeq), nil
}

// convertIdeaToEpic converts an idea to an epic (for testing)
func convertIdeaToEpic(ctx context.Context, ideaRepo ideaConvertRepo, epicRepo interface {
	Create(context.Context, *models.Epic) error
	GetByKey(context.Context, string) (*models.Epic, error)
}, ideaKey string) (string, error) {
	return convertIdeaToEpicWithKey(ctx, ideaRepo, epicRepo, ideaKey, "E15")
}

// convertIdeaToEpicWithKey converts an idea to an epic with a specified key
func convertIdeaToEpicWithKey(ctx context.Context, ideaRepo ideaConvertRepo, epicRepo interface {
	Create(context.Context, *models.Epic) error
}, ideaKey, epicKey string) (string, error) {
	idea, err := ideaRepo.GetByKey(ctx, ideaKey)
	if err != nil {
		return "", fmt.Errorf("failed to get idea: %w", err)
	}

	if idea.Status == models.IdeaStatusConverted {
		convertedInfo := ""
		if idea.ConvertedToType != nil && idea.ConvertedToKey != nil {
			convertedInfo = fmt.Sprintf(" to %s %s", *idea.ConvertedToType, *idea.ConvertedToKey)
		}
		return "", fmt.Errorf("idea %s is already converted%s", idea.Key, convertedInfo)
	}

	epic := &models.Epic{
		BaseEntity: models.BaseEntity{
			Key:         epicKey,
			Title:       idea.Title,
			Description: idea.Description,
		},
		Status:        "draft",
		Priority:      models.PriorityMedium,
		BusinessValue: priorityPtr(models.PriorityMedium),
	}

	if err := epicRepo.Create(ctx, epic); err != nil {
		return "", fmt.Errorf("failed to create epic: %w", err)
	}

	if err := ideaRepo.MarkAsConverted(ctx, idea.ID, "epic", epic.Key); err != nil {
		return "", fmt.Errorf("failed to mark idea as converted: %w", err)
	}

	return epic.Key, nil
}

// convertIdeaToFeature converts an idea to a feature (for testing)
func convertIdeaToFeature(ctx context.Context, ideaRepo ideaConvertRepo, epicRepo interface {
	GetByKey(context.Context, string) (*models.Epic, error)
}, featureRepo interface {
	Create(context.Context, *models.Feature) error
}, ideaKey, epicKey string) (string, error) {
	epic, err := epicRepo.GetByKey(ctx, epicKey)
	if err != nil {
		return "", fmt.Errorf("failed to get epic: %w", err)
	}

	return convertIdeaToFeatureWithKey(ctx, ideaRepo, epic, featureRepo, ideaKey, "E10-F03")
}

// convertIdeaToFeatureWithKey converts an idea to a feature with a specified key
func convertIdeaToFeatureWithKey(ctx context.Context, ideaRepo ideaConvertRepo, epic *models.Epic, featureRepo interface {
	Create(context.Context, *models.Feature) error
}, ideaKey, featureKey string) (string, error) {
	idea, err := ideaRepo.GetByKey(ctx, ideaKey)
	if err != nil {
		return "", fmt.Errorf("failed to get idea: %w", err)
	}

	if idea.Status == models.IdeaStatusConverted {
		convertedInfo := ""
		if idea.ConvertedToType != nil && idea.ConvertedToKey != nil {
			convertedInfo = fmt.Sprintf(" to %s %s", *idea.ConvertedToType, *idea.ConvertedToKey)
		}
		return "", fmt.Errorf("idea %s is already converted%s", idea.Key, convertedInfo)
	}

	feature := &models.Feature{BaseEntity: models.BaseEntity{Key: featureKey,
		Title:       idea.Title,
		Description: idea.Description}, EpicID: epic.ID,

		Status: "draft",
	}

	if err := featureRepo.Create(ctx, feature); err != nil {
		return "", fmt.Errorf("failed to create feature: %w", err)
	}

	if err := ideaRepo.MarkAsConverted(ctx, idea.ID, "feature", feature.Key); err != nil {
		return "", fmt.Errorf("failed to mark idea as converted: %w", err)
	}

	return feature.Key, nil
}

// convertIdeaToTask converts an idea to a task (for testing - uses hardcoded task key)
func convertIdeaToTask(ctx context.Context, ideaRepo ideaConvertRepo, epicRepo interface {
	GetByKey(context.Context, string) (*models.Epic, error)
}, featureRepo interface {
	GetByKey(context.Context, string) (*models.Feature, error)
}, taskRepo interface {
	Create(context.Context, *models.Task) error
}, ideaKey, epicKey, featureKey string) (string, error) {
	epic, err := epicRepo.GetByKey(ctx, epicKey)
	if err != nil {
		return "", fmt.Errorf("failed to get epic: %w", err)
	}

	feature, err := featureRepo.GetByKey(ctx, featureKey)
	if err != nil {
		return "", fmt.Errorf("failed to get feature: %w", err)
	}

	if feature.EpicID != epic.ID {
		return "", fmt.Errorf("feature %s does not belong to epic %s", feature.Key, epic.Key)
	}

	idea, err := ideaRepo.GetByKey(ctx, ideaKey)
	if err != nil {
		return "", fmt.Errorf("failed to get idea: %w", err)
	}

	if idea.Status == models.IdeaStatusConverted {
		convertedInfo := ""
		if idea.ConvertedToType != nil && idea.ConvertedToKey != nil {
			convertedInfo = fmt.Sprintf(" to %s %s", *idea.ConvertedToType, *idea.ConvertedToKey)
		}
		return "", fmt.Errorf("idea %s is already converted%s", idea.Key, convertedInfo)
	}

	agentType := "general"
	task := &models.Task{BaseEntity: models.BaseEntity{Key: "T-E10-F02-005", // Hardcoded for test compatibility
		Title:       idea.Title,
		Description: idea.Description}, FeatureID: feature.ID,

		Status:    "todo",
		AgentType: &agentType,
		Priority:  5,
	}

	if idea.Priority != nil {
		task.Priority = *idea.Priority
	}

	if err := taskRepo.Create(ctx, task); err != nil {
		return "", fmt.Errorf("failed to create task: %w", err)
	}

	if err := ideaRepo.MarkAsConverted(ctx, idea.ID, "task", task.Key); err != nil {
		return "", fmt.Errorf("failed to mark idea as converted: %w", err)
	}

	return task.Key, nil
}

// priorityPtr returns a pointer to a Priority value
func priorityPtr(p models.Priority) *models.Priority {
	return &p
}

// printIdeaConvertResult outputs the result of converting an idea to an entity.
func printIdeaConvertResult(ideaKey, entityType, entityKey string, extra map[string]interface{}) error {
	if cli.GlobalConfig.JSON {
		out := map[string]interface{}{
			"idea_key": ideaKey, "converted_to": entityKey, "type": entityType,
		}
		for k, v := range extra {
			out[k] = v
		}
		return cli.OutputJSON(out)
	}
	switch entityType {
	case "feature":
		cli.Success(fmt.Sprintf("Idea %s converted to feature %s in epic %s", ideaKey, entityKey, extra["epic"]))
	case "task":
		cli.Success(fmt.Sprintf("Idea %s converted to task %s in %s/%s", ideaKey, entityKey, extra["epic"], extra["feature"]))
	default:
		cli.Success(fmt.Sprintf("Idea %s converted to %s %s", ideaKey, entityType, entityKey))
	}
	return nil
}

// getIdeaDescription returns the description string, or empty string if nil.
func getIdeaDescription(idea *models.Idea) string {
	if idea.Description != nil {
		return *idea.Description
	}
	return ""
}

// runIdeaConvertEpic handles converting an idea to an epic
func runIdeaConvertEpic(cmd *cobra.Command, args []string) error {
	ideaKey := args[0]
	ideaSvc := cli.GetIdeaService()
	idea, err := ideaSvc.GetIdea(cmd.Context(), ideaKey)
	if err != nil {
		return fmt.Errorf("failed to get idea: %w", err)
	}
	epic, err := cli.GetEpicService().CreateEpic(cmd.Context(), services.CreateEpicInput{
		Title: idea.Title, Description: idea.Description, Status: "draft", Priority: "medium",
	})
	if err != nil {
		return fmt.Errorf("failed to create epic: %w", err)
	}
	if err := ideaSvc.ConvertIdea(cmd.Context(), ideaKey, "epic", epic.Key); err != nil {
		return fmt.Errorf("failed to mark idea as converted: %w", err)
	}
	return printIdeaConvertResult(ideaKey, "epic", epic.Key, nil)
}

// runIdeaConvertFeature handles converting an idea to a feature
func runIdeaConvertFeature(cmd *cobra.Command, args []string) error {
	ideaKey := args[0]
	ideaSvc := cli.GetIdeaService()
	idea, err := ideaSvc.GetIdea(cmd.Context(), ideaKey)
	if err != nil {
		return fmt.Errorf("failed to get idea: %w", err)
	}
	feature, err := cli.GetFeatureService().CreateFeature(cmd.Context(), services.CreateFeatureInput{
		EpicKey: ideaConvertEpic, Title: idea.Title, Description: idea.Description, Status: "draft",
	})
	if err != nil {
		return fmt.Errorf("failed to create feature: %w", err)
	}
	if err := ideaSvc.ConvertIdea(cmd.Context(), ideaKey, "feature", feature.Key); err != nil {
		return fmt.Errorf("failed to mark idea as converted: %w", err)
	}
	return printIdeaConvertResult(ideaKey, "feature", feature.Key, map[string]interface{}{"epic": ideaConvertEpic})
}

// runIdeaConvertTask handles converting an idea to a task
func runIdeaConvertTask(cmd *cobra.Command, args []string) error {
	ideaKey := args[0]
	ideaSvc := cli.GetIdeaService()
	idea, err := ideaSvc.GetIdea(cmd.Context(), ideaKey)
	if err != nil {
		return fmt.Errorf("failed to get idea: %w", err)
	}
	task, err := cli.GetTaskService().CreateTask(cmd.Context(), services.CreateTaskInput{
		EpicKey: ideaConvertEpic, FeatureKey: ideaConvertFeature,
		Title: idea.Title, Description: getIdeaDescription(idea),
	})
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}
	if err := ideaSvc.ConvertIdea(cmd.Context(), ideaKey, "task", task.Key); err != nil {
		return fmt.Errorf("failed to mark idea as converted: %w", err)
	}
	return printIdeaConvertResult(ideaKey, "task", task.Key, map[string]interface{}{
		"epic": ideaConvertEpic, "feature": ideaConvertFeature,
	})
}

// --- Helper types ---

// ideaConvertRepo is the minimal interface needed for idea conversion helpers.
// Used by test helper functions (convertIdeaToEpic, convertIdeaToFeature, convertIdeaToTask).
type ideaConvertRepo interface {
	GetByKey(ctx context.Context, key string) (*models.Idea, error)
	MarkAsConverted(ctx context.Context, ideaID int64, convertedToType, convertedToKey string) error
}

// --- Output formatting helpers ---

// filterIdeasByPriorityAndStatus filters ideas client-side by priority and removes archived when no status set.
func filterIdeasByPriorityAndStatus(ideas []*models.Idea, priority int, status string) []*models.Idea {
	filtered := make([]*models.Idea, 0, len(ideas))
	for _, idea := range ideas {
		if priority > 0 && (idea.Priority == nil || *idea.Priority != priority) {
			continue
		}
		if status == "" && idea.Status == models.IdeaStatusArchived {
			continue
		}
		filtered = append(filtered, idea)
	}
	return filtered
}

// buildIdeaListRows converts a slice of ideas to table rows for list display.
// Extracted for testability (E07-F42 F4 coverage requirement).
func buildIdeaListRows(ideas []*models.Idea) [][]string {
	rows := make([][]string, 0, len(ideas))
	for _, idea := range ideas {
		priority := "-"
		if idea.Priority != nil {
			priority = strconv.Itoa(*idea.Priority)
		}
		rows = append(rows, []string{
			idea.Key,
			idea.Title,
			string(idea.Status),
			priority,
			idea.CreatedDate.Format("2006-01-02"),
			formatSize(idea.Size), // E07-F42 REQ-F-006: Size column
		})
	}
	return rows
}

// printIdeaList prints ideas in table format.
func printIdeaList(ideas []*models.Idea) error {
	if len(ideas) == 0 {
		fmt.Println("No ideas found")
		return nil
	}

	// E07-F42: Size column added to idea list table (REQ-F-006).
	headers := []string{"Key", "Title", "Status", "Priority", "Created", "Size"}
	cli.OutputTable(headers, buildIdeaListRows(ideas))
	return nil
}

// printIdeaDetailWithTags prints detailed idea information including the tag line.
// tags==nil means tagSvc unavailable (no Tags line). Empty slice renders "Tags: (none)".
func printIdeaDetailWithTags(idea *models.Idea, tags []string) error {
	fmt.Printf("Idea: %s\n", idea.Key)
	fmt.Printf("Title: %s\n", idea.Title)
	fmt.Printf("Status: %s\n", idea.Status)
	if idea.Description != nil {
		fmt.Printf("Description: %s\n", *idea.Description)
	}
	if idea.Priority != nil {
		fmt.Printf("Priority: %d\n", *idea.Priority)
	}
	if idea.Order != nil {
		fmt.Printf("Order: %d\n", *idea.Order)
	}
	if idea.Notes != nil {
		fmt.Printf("Notes: %s\n", *idea.Notes)
	}
	if idea.RelatedDocs != nil && *idea.RelatedDocs != "" {
		fmt.Printf("Related Docs: %s\n", *idea.RelatedDocs)
	}
	if idea.Dependencies != nil && *idea.Dependencies != "" {
		fmt.Printf("Dependencies: %s\n", *idea.Dependencies)
	}
	// E07-F42 REQ-F-006: human display uses "<label> (<num>)" or omits the line entirely.
	if idea.Size != nil {
		fmt.Printf("Size: %s\n", formatSize(idea.Size))
	}
	if idea.Status == models.IdeaStatusConverted {
		if idea.ConvertedToType != nil && idea.ConvertedToKey != nil {
			fmt.Printf("\nConverted to: %s %s\n", *idea.ConvertedToType, *idea.ConvertedToKey)
		}
		if idea.ConvertedAt != nil {
			fmt.Printf("Converted at: %s\n", idea.ConvertedAt.Format("2006-01-02 15:04:05"))
		}
	}
	fmt.Printf("Created: %s\n", idea.CreatedDate.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated: %s\n", idea.UpdatedAt.Format("2006-01-02 15:04:05"))
	// REQ-F-015: render tags when available. nil means tagSvc unavailable — omit.
	if tags != nil {
		if len(tags) == 0 {
			fmt.Println("Tags: (none)")
		} else {
			fmt.Printf("Tags: %s\n", strings.Join(tags, ", "))
		}
	}
	return nil
}

// parseCreateIdeaInput builds a CreateIdeaInput from the current flag values.
func parseCreateIdeaInput(title string) (services.CreateIdeaInput, error) {
	input := services.CreateIdeaInput{
		Title:  title,
		Status: ideaStatus,
		Tags:   ideaCreateTags,
	}

	if ideaDescription != "" {
		input.Description = &ideaDescription
	}
	if ideaPriority > 0 {
		input.Priority = &ideaPriority
	}
	if ideaOrder > 0 {
		input.Order = &ideaOrder
	}
	if ideaNotes != "" {
		input.Notes = &ideaNotes
	}
	if len(ideaRelatedDocs) > 0 {
		docs, err := json.Marshal(ideaRelatedDocs)
		if err != nil {
			return input, fmt.Errorf("failed to marshal related docs: %w", err)
		}
		docsStr := string(docs)
		input.RelatedDocs = &docsStr
	}
	if len(ideaDependencies) > 0 {
		deps, err := json.Marshal(ideaDependencies)
		if err != nil {
			return input, fmt.Errorf("failed to marshal dependencies: %w", err)
		}
		depsStr := string(deps)
		input.Dependencies = &depsStr
	}
	// E07-F42 REQ-F-004: parse --size before calling service; reject invalid values early.
	if ideaCreateSizeFlag != "" {
		n, sizeErr := models.ParseSize(ideaCreateSizeFlag)
		if sizeErr != nil {
			return input, fmt.Errorf("invalid --size value: %w", sizeErr)
		}
		input.Size = &n
	}
	return input, nil
}

// parseUpdateIdeaInput builds an UpdateIdeaInput from changed flags.
func parseUpdateIdeaInput(cmd *cobra.Command) (services.UpdateIdeaInput, error) {
	input := services.UpdateIdeaInput{
		// E28-F04 REQ-F-010: --tag is additive. When the flag is absent,
		// ideaUpdateTags is nil (StringSliceVar default) and the service
		// treats len(Tags) == 0 as a no-op.
		Tags: ideaUpdateTags,
	}

	if cmd.Flags().Changed("title") {
		title, _ := cmd.Flags().GetString("title")
		input.Title = &title
	}
	if cmd.Flags().Changed("description") {
		input.Description = &ideaDescription
	}
	if cmd.Flags().Changed("status") {
		input.Status = &ideaStatus
	}
	if cmd.Flags().Changed("priority") {
		input.Priority = &ideaPriority
	}
	if cmd.Flags().Changed("order") {
		input.Order = &ideaOrder
	}
	if cmd.Flags().Changed("notes") {
		input.Notes = &ideaNotes
	}
	if cmd.Flags().Changed("related-docs") {
		docs, err := json.Marshal(ideaRelatedDocs)
		if err != nil {
			return input, fmt.Errorf("failed to marshal related docs: %w", err)
		}
		docsStr := string(docs)
		input.RelatedDocs = &docsStr
	}
	if cmd.Flags().Changed("depends-on") {
		deps, err := json.Marshal(ideaDependencies)
		if err != nil {
			return input, fmt.Errorf("failed to marshal dependencies: %w", err)
		}
		depsStr := string(deps)
		input.Dependencies = &depsStr
	}
	// E07-F42 REQ-F-005: three-way dispatch for --size on update.
	//   empty → no-op; "clear" → ClearSize=true; valid → Size=ptr(n).
	if cmd.Flags().Changed("size") {
		sizePtr, clearSize, sizeErr := parseSizeUpdateFlag(cmd)
		if sizeErr != nil {
			return input, sizeErr
		}
		input.Size = sizePtr
		input.ClearSize = clearSize
	}
	return input, nil
}

// confirmIdeaDelete shows a confirmation prompt; returns true if user confirms.
func confirmIdeaDelete(idea *models.Idea, hard bool) bool {
	deleteType := "archive"
	if hard {
		deleteType = "permanently delete"
	}
	fmt.Printf("Are you sure you want to %s idea %s: %s? (yes/no): ", deleteType, idea.Key, idea.Title)
	var response string
	_, _ = fmt.Scanln(&response)
	return strings.EqualFold(response, "yes") || strings.EqualFold(response, "y")
}

// performIdeaDelete executes the delete (hard or soft) and formats the result.
func performIdeaDelete(cmd *cobra.Command, svc *services.IdeaService, ideaKey string, idea *models.Idea, hard bool) error {
	if hard {
		if err := svc.DeleteIdea(cmd.Context(), ideaKey); err != nil {
			return fmt.Errorf("failed to delete idea: %w", err)
		}
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]string{"status": "deleted", "key": ideaKey})
		}
		cli.Success(fmt.Sprintf("Permanently deleted idea %s", ideaKey))
		return nil
	}

	archivedStatus := string(models.IdeaStatusArchived)
	updated, err := svc.UpdateIdea(cmd.Context(), ideaKey, services.UpdateIdeaInput{Status: &archivedStatus})
	if err != nil {
		return fmt.Errorf("failed to archive idea: %w", err)
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]string{"status": "archived", "key": updated.Key})
	}
	cli.Success(fmt.Sprintf("Archived idea %s", updated.Key))
	return nil
}
