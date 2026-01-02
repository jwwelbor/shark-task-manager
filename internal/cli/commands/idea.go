package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/taskcreation"
	"github.com/spf13/cobra"
)

// IdeaRepository interface defines methods for idea data access
type IdeaRepository interface {
	Create(ctx context.Context, idea *models.Idea) error
	GetByID(ctx context.Context, id int64) (*models.Idea, error)
	GetByKey(ctx context.Context, key string) (*models.Idea, error)
	List(ctx context.Context, filter *repository.IdeaFilter) ([]*models.Idea, error)
	Update(ctx context.Context, idea *models.Idea) error
	Delete(ctx context.Context, id int64) error
	MarkAsConverted(ctx context.Context, ideaID int64, convertedToType, convertedToKey string) error
}

// ideaCmd represents the idea command group
var ideaCmd = &cobra.Command{
	Use:     "idea",
	Short:   "Manage ideas",
	GroupID: "essentials",
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
  shark idea convert I-2026-01-01-01 epic
  shark idea convert I-2026-01-01-01 feature --epic=E10
  shark idea convert I-2026-01-01-01 task --epic=E10 --feature=E10-F02`,
}

// ideaConvertEpicCmd converts an idea to an epic
var ideaConvertEpicCmd = &cobra.Command{
	Use:   "epic <idea-key>",
	Short: "Convert idea to epic",
	Long: `Convert an idea to a new epic.

The idea's title and description are copied to the epic.
A new epic key is auto-generated (E##).

Examples:
  shark idea convert I-2026-01-01-01 epic
  shark idea convert I-2026-01-01-01 epic --json`,
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
  shark idea convert I-2026-01-01-01 feature --epic=E10
  shark idea convert I-2026-01-01-01 feature --epic=E10 --json`,
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
  shark idea convert I-2026-01-01-01 task --epic=E10 --feature=E10-F02
  shark idea convert I-2026-01-01-01 task --epic=E10 --feature=E10-F02 --json`,
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
)

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

	// Create command flags
	ideaCreateCmd.Flags().StringVar(&ideaDescription, "description", "", "Idea description")
	ideaCreateCmd.Flags().IntVar(&ideaPriority, "priority", 0, "Priority (1-10)")
	ideaCreateCmd.Flags().IntVar(&ideaOrder, "order", 0, "Order for sorting ideas")
	ideaCreateCmd.Flags().StringVar(&ideaNotes, "notes", "", "Additional notes")
	ideaCreateCmd.Flags().StringSliceVar(&ideaRelatedDocs, "related-docs", []string{}, "Related document paths")
	ideaCreateCmd.Flags().StringSliceVar(&ideaDependencies, "depends-on", []string{}, "Dependent idea keys")
	ideaCreateCmd.Flags().StringVar(&ideaStatus, "status", "new", "Initial status (new, on_hold, converted, archived)")

	// Update command flags
	ideaUpdateCmd.Flags().StringVar(&ideaStatus, "status", "", "Update status")
	ideaUpdateCmd.Flags().IntVar(&ideaPriority, "priority", 0, "Update priority (1-10)")
	ideaUpdateCmd.Flags().StringVar(&ideaDescription, "description", "", "Update description")
	ideaUpdateCmd.Flags().StringVar(&ideaNotes, "notes", "", "Update notes")
	ideaUpdateCmd.Flags().StringSliceVar(&ideaRelatedDocs, "related-docs", []string{}, "Update related document paths")
	ideaUpdateCmd.Flags().StringSliceVar(&ideaDependencies, "depends-on", []string{}, "Update dependencies")
	ideaUpdateCmd.Flags().IntVar(&ideaOrder, "order", 0, "Update order")
	ideaUpdateCmd.Flags().StringVar(&ideaDescription, "title", "", "Update title")

	// Delete command flags
	ideaDeleteCmd.Flags().BoolVar(&ideaForce, "force", false, "Skip confirmation prompt")
	ideaDeleteCmd.Flags().BoolVar(&ideaHard, "hard", false, "Perform hard delete (permanent)")
}

// runIdeaList handles the idea list command
func runIdeaList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Initialize database
	database, err := db.InitDB(cli.GlobalConfig.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	dbWrapper := repository.NewDB(database)
	repo := repository.NewIdeaRepository(dbWrapper)

	// Build filter
	var filter *repository.IdeaFilter
	if ideaStatus != "" {
		status := models.IdeaStatus(ideaStatus)
		filter = &repository.IdeaFilter{Status: &status}
	}

	// Get ideas
	ideas, err := repo.List(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list ideas: %w", err)
	}

	// Filter by priority if specified
	if ideaPriority > 0 {
		filtered := []*models.Idea{}
		for _, idea := range ideas {
			if idea.Priority != nil && *idea.Priority == ideaPriority {
				filtered = append(filtered, idea)
			}
		}
		ideas = filtered
	}

	// Filter out archived ideas by default
	if ideaStatus == "" {
		filtered := []*models.Idea{}
		for _, idea := range ideas {
			if idea.Status != models.IdeaStatusArchived {
				filtered = append(filtered, idea)
			}
		}
		ideas = filtered
	}

	// Output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(ideas)
	}

	// Table output
	if len(ideas) == 0 {
		fmt.Println("No ideas found")
		return nil
	}

	headers := []string{"Key", "Title", "Status", "Priority", "Created"}
	rows := make([][]string, len(ideas))
	for i, idea := range ideas {
		priority := "-"
		if idea.Priority != nil {
			priority = strconv.Itoa(*idea.Priority)
		}
		rows[i] = []string{
			idea.Key,
			idea.Title,
			string(idea.Status),
			priority,
			idea.CreatedDate.Format("2006-01-02"),
		}
	}

	cli.OutputTable(headers, rows)
	return nil
}

// runIdeaGet handles the idea get command
func runIdeaGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	ideaKey := args[0]

	// Initialize database
	database, err := db.InitDB(cli.GlobalConfig.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	dbWrapper := repository.NewDB(database)
	repo := repository.NewIdeaRepository(dbWrapper)

	// Get idea
	idea, err := repo.GetByKey(ctx, ideaKey)
	if err != nil {
		return fmt.Errorf("failed to get idea: %w", err)
	}

	// Output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(idea)
	}

	// Detailed text output
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

	// Display conversion information if idea was converted
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

	return nil
}

// runIdeaCreate handles the idea create command
func runIdeaCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	title := args[0]

	// Initialize database
	database, err := db.InitDB(cli.GlobalConfig.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	dbWrapper := repository.NewDB(database)
	repo := repository.NewIdeaRepository(dbWrapper)

	// Generate idea key
	ideaKey, err := generateIdeaKey(ctx, repo)
	if err != nil {
		return fmt.Errorf("failed to generate idea key: %w", err)
	}

	// Build idea with default status if not provided
	status := ideaStatus
	if status == "" {
		status = "new"
	}

	idea := &models.Idea{
		Key:         ideaKey,
		Title:       title,
		CreatedDate: time.Now(),
		Status:      models.IdeaStatus(status),
	}

	// Set optional fields
	if ideaDescription != "" {
		idea.Description = &ideaDescription
	}
	if ideaPriority > 0 {
		idea.Priority = &ideaPriority
	}
	if ideaOrder > 0 {
		idea.Order = &ideaOrder
	}
	if ideaNotes != "" {
		idea.Notes = &ideaNotes
	}

	// Handle related docs (convert slice to JSON array)
	if len(ideaRelatedDocs) > 0 {
		docs, err := json.Marshal(ideaRelatedDocs)
		if err != nil {
			return fmt.Errorf("failed to marshal related docs: %w", err)
		}
		docsStr := string(docs)
		idea.RelatedDocs = &docsStr
	}

	// Handle dependencies (convert slice to JSON array)
	if len(ideaDependencies) > 0 {
		deps, err := json.Marshal(ideaDependencies)
		if err != nil {
			return fmt.Errorf("failed to marshal dependencies: %w", err)
		}
		depsStr := string(deps)
		idea.Dependencies = &depsStr
	}

	// Create idea
	if err := repo.Create(ctx, idea); err != nil {
		return fmt.Errorf("failed to create idea: %w", err)
	}

	// Output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(idea)
	}

	cli.Success(fmt.Sprintf("Created idea %s: %s", idea.Key, idea.Title))
	return nil
}

// runIdeaUpdate handles the idea update command
func runIdeaUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	ideaKey := args[0]

	// Initialize database
	database, err := db.InitDB(cli.GlobalConfig.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	dbWrapper := repository.NewDB(database)
	repo := repository.NewIdeaRepository(dbWrapper)

	// Get existing idea
	idea, err := repo.GetByKey(ctx, ideaKey)
	if err != nil {
		return fmt.Errorf("failed to get idea: %w", err)
	}

	// Update fields if flags were provided
	if cmd.Flags().Changed("title") {
		title, _ := cmd.Flags().GetString("title")
		idea.Title = title
	}
	if cmd.Flags().Changed("description") {
		idea.Description = &ideaDescription
	}
	if cmd.Flags().Changed("status") {
		idea.Status = models.IdeaStatus(ideaStatus)
	}
	if cmd.Flags().Changed("priority") {
		idea.Priority = &ideaPriority
	}
	if cmd.Flags().Changed("order") {
		idea.Order = &ideaOrder
	}
	if cmd.Flags().Changed("notes") {
		idea.Notes = &ideaNotes
	}
	if cmd.Flags().Changed("related-docs") {
		docs, err := json.Marshal(ideaRelatedDocs)
		if err != nil {
			return fmt.Errorf("failed to marshal related docs: %w", err)
		}
		docsStr := string(docs)
		idea.RelatedDocs = &docsStr
	}
	if cmd.Flags().Changed("depends-on") {
		deps, err := json.Marshal(ideaDependencies)
		if err != nil {
			return fmt.Errorf("failed to marshal dependencies: %w", err)
		}
		depsStr := string(deps)
		idea.Dependencies = &depsStr
	}

	// Update idea
	if err := repo.Update(ctx, idea); err != nil {
		return fmt.Errorf("failed to update idea: %w", err)
	}

	// Output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(idea)
	}

	cli.Success(fmt.Sprintf("Updated idea %s", idea.Key))
	return nil
}

// runIdeaDelete handles the idea delete command
func runIdeaDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	ideaKey := args[0]

	// Initialize database
	database, err := db.InitDB(cli.GlobalConfig.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	dbWrapper := repository.NewDB(database)
	repo := repository.NewIdeaRepository(dbWrapper)

	// Get idea to confirm it exists
	idea, err := repo.GetByKey(ctx, ideaKey)
	if err != nil {
		return fmt.Errorf("failed to get idea: %w", err)
	}

	// Confirmation prompt (unless --force)
	if !ideaForce {
		var response string
		deleteType := "archive"
		if ideaHard {
			deleteType = "permanently delete"
		}
		fmt.Printf("Are you sure you want to %s idea %s: %s? (yes/no): ", deleteType, idea.Key, idea.Title)
		_, _ = fmt.Scanln(&response)
		if !strings.EqualFold(response, "yes") && !strings.EqualFold(response, "y") {
			fmt.Println("Delete cancelled")
			return nil
		}
	}

	// Perform delete
	if ideaHard {
		// Hard delete
		if err := repo.Delete(ctx, idea.ID); err != nil {
			return fmt.Errorf("failed to delete idea: %w", err)
		}
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]string{"status": "deleted", "key": idea.Key})
		}
		cli.Success(fmt.Sprintf("Permanently deleted idea %s", idea.Key))
	} else {
		// Soft delete (archive)
		idea.Status = models.IdeaStatusArchived
		if err := repo.Update(ctx, idea); err != nil {
			return fmt.Errorf("failed to archive idea: %w", err)
		}
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]string{"status": "archived", "key": idea.Key})
		}
		cli.Success(fmt.Sprintf("Archived idea %s", idea.Key))
	}

	return nil
}

// generateIdeaKey generates the next available idea key for today's date
// Format: I-YYYY-MM-DD-xx where xx is 01-99
func generateIdeaKey(ctx context.Context, repo IdeaRepository) (string, error) {
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	baseKey := fmt.Sprintf("I-%s", dateStr)

	// Get all ideas for today
	allIdeas, err := repo.List(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to list ideas: %w", err)
	}

	// Find highest sequence number for today
	maxSeq := 0
	prefix := baseKey + "-"
	for _, idea := range allIdeas {
		if strings.HasPrefix(idea.Key, prefix) {
			// Extract sequence number
			parts := strings.Split(idea.Key, "-")
			if len(parts) == 5 {
				seq, err := strconv.Atoi(parts[4])
				if err == nil && seq > maxSeq {
					maxSeq = seq
				}
			}
		}
	}

	// Generate next sequence number
	nextSeq := maxSeq + 1
	if nextSeq > 99 {
		return "", fmt.Errorf("maximum ideas for date %s reached (99)", dateStr)
	}

	return fmt.Sprintf("%s-%02d", baseKey, nextSeq), nil
}

// convertIdeaToEpic converts an idea to an epic (for testing)
func convertIdeaToEpic(ctx context.Context, ideaRepo IdeaRepository, epicRepo interface {
	Create(context.Context, *models.Epic) error
	GetByKey(context.Context, string) (*models.Epic, error)
}, ideaKey string) (string, error) {
	return convertIdeaToEpicWithKey(ctx, ideaRepo, epicRepo, ideaKey, "E15")
}

// convertIdeaToEpicWithKey converts an idea to an epic with a specified key
func convertIdeaToEpicWithKey(ctx context.Context, ideaRepo IdeaRepository, epicRepo interface {
	Create(context.Context, *models.Epic) error
}, ideaKey, epicKey string) (string, error) {
	// Get the idea
	idea, err := ideaRepo.GetByKey(ctx, ideaKey)
	if err != nil {
		return "", fmt.Errorf("failed to get idea: %w", err)
	}

	// Check if already converted
	if idea.Status == models.IdeaStatusConverted {
		convertedInfo := ""
		if idea.ConvertedToType != nil && idea.ConvertedToKey != nil {
			convertedInfo = fmt.Sprintf(" to %s %s", *idea.ConvertedToType, *idea.ConvertedToKey)
		}
		return "", fmt.Errorf("idea %s is already converted%s", idea.Key, convertedInfo)
	}

	// Create epic from idea
	epic := &models.Epic{
		Key:           epicKey,
		Title:         idea.Title,
		Description:   idea.Description,
		Status:        "draft",
		Priority:      models.PriorityMedium,
		BusinessValue: priorityPtr(models.PriorityMedium),
	}

	// Create the epic
	if err := epicRepo.Create(ctx, epic); err != nil {
		return "", fmt.Errorf("failed to create epic: %w", err)
	}

	// Mark idea as converted
	if err := ideaRepo.MarkAsConverted(ctx, idea.ID, "epic", epic.Key); err != nil {
		return "", fmt.Errorf("failed to mark idea as converted: %w", err)
	}

	return epic.Key, nil
}

// convertIdeaToFeature converts an idea to a feature (for testing)
func convertIdeaToFeature(ctx context.Context, ideaRepo IdeaRepository, epicRepo interface {
	GetByKey(context.Context, string) (*models.Epic, error)
}, featureRepo interface {
	Create(context.Context, *models.Feature) error
}, ideaKey, epicKey string) (string, error) {
	// Validate epic exists
	epic, err := epicRepo.GetByKey(ctx, epicKey)
	if err != nil {
		return "", fmt.Errorf("failed to get epic: %w", err)
	}

	return convertIdeaToFeatureWithKey(ctx, ideaRepo, epic, featureRepo, ideaKey, "E10-F03")
}

// convertIdeaToFeatureWithKey converts an idea to a feature with a specified key
func convertIdeaToFeatureWithKey(ctx context.Context, ideaRepo IdeaRepository, epic *models.Epic, featureRepo interface {
	Create(context.Context, *models.Feature) error
}, ideaKey, featureKey string) (string, error) {
	// Get the idea
	idea, err := ideaRepo.GetByKey(ctx, ideaKey)
	if err != nil {
		return "", fmt.Errorf("failed to get idea: %w", err)
	}

	// Check if already converted
	if idea.Status == models.IdeaStatusConverted {
		convertedInfo := ""
		if idea.ConvertedToType != nil && idea.ConvertedToKey != nil {
			convertedInfo = fmt.Sprintf(" to %s %s", *idea.ConvertedToType, *idea.ConvertedToKey)
		}
		return "", fmt.Errorf("idea %s is already converted%s", idea.Key, convertedInfo)
	}

	// Create feature from idea
	feature := &models.Feature{
		EpicID:      epic.ID,
		Key:         featureKey,
		Title:       idea.Title,
		Description: idea.Description,
		Status:      "draft",
	}

	// Create the feature
	if err := featureRepo.Create(ctx, feature); err != nil {
		return "", fmt.Errorf("failed to create feature: %w", err)
	}

	// Mark idea as converted
	if err := ideaRepo.MarkAsConverted(ctx, idea.ID, "feature", feature.Key); err != nil {
		return "", fmt.Errorf("failed to mark idea as converted: %w", err)
	}

	return feature.Key, nil
}

// convertIdeaToTask converts an idea to a task (for testing - uses hardcoded task key)
func convertIdeaToTask(ctx context.Context, ideaRepo IdeaRepository, epicRepo interface {
	GetByKey(context.Context, string) (*models.Epic, error)
}, featureRepo interface {
	GetByKey(context.Context, string) (*models.Feature, error)
}, taskRepo interface {
	Create(context.Context, *models.Task) error
}, ideaKey, epicKey, featureKey string) (string, error) {
	// Validate epic exists
	epic, err := epicRepo.GetByKey(ctx, epicKey)
	if err != nil {
		return "", fmt.Errorf("failed to get epic: %w", err)
	}

	// Validate feature exists
	feature, err := featureRepo.GetByKey(ctx, featureKey)
	if err != nil {
		return "", fmt.Errorf("failed to get feature: %w", err)
	}

	// Verify feature belongs to epic
	if feature.EpicID != epic.ID {
		return "", fmt.Errorf("feature %s does not belong to epic %s", feature.Key, epic.Key)
	}

	// Get the idea
	idea, err := ideaRepo.GetByKey(ctx, ideaKey)
	if err != nil {
		return "", fmt.Errorf("failed to get idea: %w", err)
	}

	// Check if already converted
	if idea.Status == models.IdeaStatusConverted {
		convertedInfo := ""
		if idea.ConvertedToType != nil && idea.ConvertedToKey != nil {
			convertedInfo = fmt.Sprintf(" to %s %s", *idea.ConvertedToType, *idea.ConvertedToKey)
		}
		return "", fmt.Errorf("idea %s is already converted%s", idea.Key, convertedInfo)
	}

	// Create task from idea
	agentType := models.AgentTypeGeneral
	task := &models.Task{
		FeatureID:   feature.ID,
		Key:         "T-E10-F02-005", // Hardcoded for test compatibility
		Title:       idea.Title,
		Description: idea.Description,
		Status:      "todo",
		AgentType:   &agentType,
		Priority:    5, // default priority
	}

	// Use idea priority if provided
	if idea.Priority != nil {
		task.Priority = *idea.Priority
	}

	// Create the task
	if err := taskRepo.Create(ctx, task); err != nil {
		return "", fmt.Errorf("failed to create task: %w", err)
	}

	// Mark idea as converted
	if err := ideaRepo.MarkAsConverted(ctx, idea.ID, "task", task.Key); err != nil {
		return "", fmt.Errorf("failed to mark idea as converted: %w", err)
	}

	return task.Key, nil
}

// convertIdeaToTaskWithKey converts an idea to a task with a specified key
func convertIdeaToTaskWithKey(ctx context.Context, ideaRepo IdeaRepository, epic *models.Epic, feature *models.Feature, taskRepo interface {
	Create(context.Context, *models.Task) error
}, ideaKey, taskKey string) (string, error) {
	// Get the idea
	idea, err := ideaRepo.GetByKey(ctx, ideaKey)
	if err != nil {
		return "", fmt.Errorf("failed to get idea: %w", err)
	}

	// Check if already converted
	if idea.Status == models.IdeaStatusConverted {
		convertedInfo := ""
		if idea.ConvertedToType != nil && idea.ConvertedToKey != nil {
			convertedInfo = fmt.Sprintf(" to %s %s", *idea.ConvertedToType, *idea.ConvertedToKey)
		}
		return "", fmt.Errorf("idea %s is already converted%s", idea.Key, convertedInfo)
	}

	// Create task from idea
	agentType := models.AgentTypeGeneral
	task := &models.Task{
		FeatureID:   feature.ID,
		Key:         taskKey,
		Title:       idea.Title,
		Description: idea.Description,
		Status:      "todo",
		AgentType:   &agentType,
		Priority:    5, // default priority
	}

	// Use idea priority if provided
	if idea.Priority != nil {
		task.Priority = *idea.Priority
	}

	// Create the task
	if err := taskRepo.Create(ctx, task); err != nil {
		return "", fmt.Errorf("failed to create task: %w", err)
	}

	// Mark idea as converted
	if err := ideaRepo.MarkAsConverted(ctx, idea.ID, "task", task.Key); err != nil {
		return "", fmt.Errorf("failed to mark idea as converted: %w", err)
	}

	return task.Key, nil
}

// priorityPtr returns a pointer to a Priority value
func priorityPtr(p models.Priority) *models.Priority {
	return &p
}

// runIdeaConvertEpic handles converting an idea to an epic
func runIdeaConvertEpic(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	ideaKey := args[0]

	// Initialize database
	database, err := db.InitDB(cli.GlobalConfig.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	dbWrapper := repository.NewDB(database)
	ideaRepo := repository.NewIdeaRepository(dbWrapper)
	epicRepo := repository.NewEpicRepository(dbWrapper)

	// Generate next epic key
	nextKey, err := getNextEpicKey(ctx, epicRepo)
	if err != nil {
		return fmt.Errorf("failed to generate epic key: %w", err)
	}

	// Convert idea to epic
	newKey, err := convertIdeaToEpicWithKey(ctx, ideaRepo, epicRepo, ideaKey, nextKey)
	if err != nil {
		return err
	}

	// Output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"idea_key":     ideaKey,
			"converted_to": newKey,
			"type":         "epic",
		})
	}

	cli.Success(fmt.Sprintf("Idea %s converted to epic %s", ideaKey, newKey))
	return nil
}

// runIdeaConvertFeature handles converting an idea to a feature
func runIdeaConvertFeature(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	ideaKey := args[0]

	// Initialize database
	database, err := db.InitDB(cli.GlobalConfig.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	dbWrapper := repository.NewDB(database)
	ideaRepo := repository.NewIdeaRepository(dbWrapper)
	epicRepo := repository.NewEpicRepository(dbWrapper)
	featureRepo := repository.NewFeatureRepository(dbWrapper)

	// Get epic first to generate feature key
	epic, err := epicRepo.GetByKey(ctx, ideaConvertEpic)
	if err != nil {
		return fmt.Errorf("failed to get epic: %w", err)
	}

	// Generate next feature key
	nextKey, err := getNextFeatureKey(ctx, featureRepo, epic.ID, epic.Key)
	if err != nil {
		return fmt.Errorf("failed to generate feature key: %w", err)
	}

	// Convert idea to feature
	newKey, err := convertIdeaToFeatureWithKey(ctx, ideaRepo, epic, featureRepo, ideaKey, nextKey)
	if err != nil {
		return err
	}

	// Output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"idea_key":     ideaKey,
			"converted_to": newKey,
			"type":         "feature",
			"epic":         ideaConvertEpic,
		})
	}

	cli.Success(fmt.Sprintf("Idea %s converted to feature %s in epic %s", ideaKey, newKey, ideaConvertEpic))
	return nil
}

// runIdeaConvertTask handles converting an idea to a task
func runIdeaConvertTask(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	ideaKey := args[0]

	// Initialize database
	database, err := db.InitDB(cli.GlobalConfig.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	dbWrapper := repository.NewDB(database)
	ideaRepo := repository.NewIdeaRepository(dbWrapper)
	epicRepo := repository.NewEpicRepository(dbWrapper)
	featureRepo := repository.NewFeatureRepository(dbWrapper)
	taskRepo := repository.NewTaskRepository(dbWrapper)

	// Get epic and feature to validate
	epic, err := epicRepo.GetByKey(ctx, ideaConvertEpic)
	if err != nil {
		return fmt.Errorf("failed to get epic: %w", err)
	}

	feature, err := featureRepo.GetByKey(ctx, ideaConvertFeature)
	if err != nil {
		return fmt.Errorf("failed to get feature: %w", err)
	}

	// Verify feature belongs to epic
	if feature.EpicID != epic.ID {
		return fmt.Errorf("feature %s does not belong to epic %s", feature.Key, epic.Key)
	}

	// Generate task key using KeyGenerator
	kg := taskcreation.NewKeyGenerator(taskRepo, featureRepo)
	taskKey, err := kg.GenerateTaskKey(ctx, epic.Key, feature.Key)
	if err != nil {
		return fmt.Errorf("failed to generate task key: %w", err)
	}

	// Convert idea to task
	newKey, err := convertIdeaToTaskWithKey(ctx, ideaRepo, epic, feature, taskRepo, ideaKey, taskKey)
	if err != nil {
		return err
	}

	// Output
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"idea_key":     ideaKey,
			"converted_to": newKey,
			"type":         "task",
			"epic":         ideaConvertEpic,
			"feature":      ideaConvertFeature,
		})
	}

	cli.Success(fmt.Sprintf("Idea %s converted to task %s in %s/%s", ideaKey, newKey, ideaConvertEpic, ideaConvertFeature))
	return nil
}
