package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

// createCmd is the parent command for creating entities
var createCmd = &cobra.Command{
	Use:     "create <type> [args]",
	Short:   "Create an epic, feature, task, bug, change, tech-debt, idea, or note",
	GroupID: "manage",
	Long: `Create a new entity. Dispatches to the appropriate create handler based on type.

Entity types:
  epic        Create a new epic
  feature     Create a feature in an epic
  task        Create a task in a feature
  bug         Create a bug report
  change      Create a change card (also: changes, change-card, change_card)
  tech-debt   Create a tech-debt item (also: td)
  idea        Create a new idea
  note        Add a note to any entity (auto-detects type from key)

Examples:
  shark create epic "Q1 2025 Roadmap"
  shark create feature E07 "User Authentication"
  shark create task E07 F01 "Implement login" --agent=backend --priority=5
  shark create bug "Login page crashes on Safari" --severity=high
  shark create change "Migrate auth to OAuth2" --justification="Security requirement"
  shark create tech-debt "Refactor auth module" --category=architecture --severity=high
  shark create note E07-F01-001 "Decided to use JWT for stateless auth" --type=decision`,
}

// createEpicCmd delegates to runEpicCreate
var createEpicCmd = &cobra.Command{
	Use:   "epic <title> [flags]",
	Short: "Create a new epic (alias for 'epic create')",
	Long: `Create a new epic. Equivalent to 'shark epic create'.

Examples:
  shark create epic "Q1 2025 Roadmap"
  shark create epic "Bug Fixes" --key=bugs
  shark create epic "Custom Epic" --file="docs/custom/epic.md"`,
	Args: cobra.ExactArgs(1),
	RunE: runEpicCreate,
}

// createFeatureCmd delegates to runFeatureCreate
var createFeatureCmd = &cobra.Command{
	Use:   "feature [EPIC] <title> [flags]",
	Short: "Create a new feature (alias for 'feature create')",
	Long: `Create a new feature. Equivalent to 'shark feature create'.

Supports positional syntax:
  - 2-arg format: shark create feature E07 "User Authentication"
  - With --epic flag: shark create feature "User Authentication" --epic=E07

Examples:
  shark create feature E07 "User Authentication"
  shark create feature "User Authentication" --epic=E07
  shark create feature E07 "Custom Feature" --file="docs/custom/feature.md"`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runFeatureCreate,
}

// createTaskCmd delegates to runTaskCreate
var createTaskCmd = &cobra.Command{
	Use:   "task [EPIC] [FEATURE] <title> [flags]",
	Short: "Create a new task (alias for 'task create')",
	Long: `Create a new task. Equivalent to 'shark task create'.

Supports positional syntax:
  - 3-arg format: shark create task E07 F01 "Implement login"
  - 2-arg format: shark create task E07-F01 "Implement login"
  - With flags: shark create task "Implement login" --epic=E07 --feature=F01

Examples:
  shark create task E07 F01 "Implement login" --agent=backend --priority=5
  shark create task E07-F01 "Implement login"
  shark create task "Implement login" --epic=E07 --feature=F01`,
	Args: cobra.RangeArgs(1, 3),
	RunE: runTaskCreate,
}

// createBugCmd delegates to runBugCreate
var createBugCmd = &cobra.Command{
	Use:   "bug <title> [flags]",
	Short: "Create a new bug report",
	Long: `Create a new bug report with auto-generated key (B###).

Examples:
  shark create bug "Login page crashes on Safari"
  shark create bug "Payment fails" --severity=critical --description="Card is declined unexpectedly"
  shark create bug "Slow query" --severity=low --linked-type=feature --linked-key=E07-F01`,
	Args: cobra.ExactArgs(1),
	RunE: runBugCreate,
}

// createChangeCmd delegates to runChangeCardCreate
var createChangeCmd = &cobra.Command{
	Use:     "change <title> [flags]",
	Aliases: []string{"changes", "change_card", "change_cards", "change-cards"},
	Short:   "Create a new change card",
	Long: `Create a new change card with auto-generated key (CC-###).

Examples:
  shark create change "Migrate auth to OAuth2"
  shark create change "Update dependencies" --justification="Security patches"
  shark create change "Refactor DB layer" --requested-by="Product Team" --epic=E07`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeCardCreate,
}

// createTechDebtCmd delegates to runTdCreate. `td` is registered as an alias
// so `shark create td "..."` also works (matches the top-level `shark td`
// command name). Flags are registered via the same helper used by
// `shark td create`, ensuring identical behavior.
var createTechDebtCmd = &cobra.Command{
	Use:     "tech-debt <title> [flags]",
	Aliases: []string{"td"},
	Short:   "Create a new tech-debt item (alias for 'td create')",
	Long: `Create a new tech-debt item with auto-generated key (TD-### format).
Equivalent to 'shark td create'.

Examples:
  shark create tech-debt "Refactor auth module"
  shark create tech-debt "Update dependencies" --category=dependency --severity=high
  shark create tech-debt "Add unit tests" --effort-estimate=M --tag=cleanup
  shark create tech-debt "Refactor auth module" --file=docs/plan/td/refactor.md`,
	Args: cobra.ExactArgs(1),
	RunE: runTdCreate,
}

// createIdeaCmd delegates to runIdeaCreate
var createIdeaCmd = &cobra.Command{
	Use:   "idea <title> [flags]",
	Short: "Create a new idea",
	Long: `Create a new idea with auto-generated key (I-YYYY-MM-DD-## format).

Examples:
  shark create idea "New feature idea"
  shark create idea "Backend optimization" --description="Improve query performance" --priority=8
  shark create idea "UI redesign" --status=on_hold --notes="Waiting for design review"`,
	Args: cobra.ExactArgs(1),
	RunE: runIdeaCreate,
}

var createQuestionCmd = &cobra.Command{Use: "question <title>", Args: cobra.ExactArgs(1), RunE: runQuestionCreate}

// createChangeCardCmd is an alias for createChangeCmd (accepts "change-card")
var createChangeCardCmd = &cobra.Command{
	Use:   "change-card <title> [flags]",
	Short: "Create a new change card (alias for 'change')",
	Long: `Create a new change card with auto-generated key (CC-###).
Alias for 'shark create change'.

Examples:
  shark create change-card "Migrate auth to OAuth2"
  shark create change-card "Update dependencies" --justification="Security patches"`,
	Args:   cobra.ExactArgs(1),
	RunE:   runChangeCardCreate,
	Hidden: false,
}

// createNoteCmd adds a note to any entity, auto-detecting type from key
var createNoteCmd = &cobra.Command{
	Use:   "note <entity-key> <content>",
	Short: "Add a note to any entity (auto-detects type from key)",
	Long: `Add a typed note to an entity. The entity type is auto-detected from the key format.

Key format detection:
  E##                        Epic
  E##-F## or F##             Feature
  E##-F##-### or T-E##-F##-### Task
  B###                       Bug
  CC-###                     Change card (C###/CC### aliases accepted)
  TD-###                     Tech-debt
  I-YYYY-MM-DD-##            Idea
  S###                       Sprint

Note Types:
  comment        General observation (default)
  decision       Why we chose X over Y
  blocker        What's blocking progress
  solution       How we solved a problem
  reference      External links, documentation
  implementation What we actually built
  testing        Test results, coverage
  future         Future improvements / TODO
  question       Unanswered questions

Examples:
  shark create note E07 "Kicked off Q1 planning"
  shark create note E07-F01 "Decided to use JWT for stateless auth" --type=decision
  shark create note E07-F01-001 "Waiting for API spec" --type=blocker --created-by=alice
  shark create note B001 "Reproduced on Safari 17.2" --type=comment`,
	Args: cobra.ExactArgs(2),
	RunE: runCreateNote,
}

// resolveEntityFromKey returns the models.EntityType and human-readable
// display label for a key, or an error if the format isn't recognized.
//
// The tech-debt type needs a display translation: EntityTypeTechDebt is
// "tech_debt" (underscore) but displays as "tech-debt" (hyphen).
func resolveEntityFromKey(key string) (models.EntityType, string, error) {
	detected := DetectEntityType(key)
	if detected == "change_card" {
		detected = string(models.EntityTypeChange)
	}
	et := models.EntityType(detected)
	if !models.ValidEntityTypes[et] {
		return "", "", unknownEntityKeyError(key)
	}
	if et == models.EntityTypeTechDebt {
		return et, "tech-debt", nil
	}
	return et, string(et), nil
}

// unknownEntityKeyError is the canonical error when a key can't be
// resolved. Centralized so the long format list stays consistent across
// `create note`, `notes add`, and any future dispatch surface.
func unknownEntityKeyError(key string) error {
	return fmt.Errorf("cannot determine entity type from key: %s\nExpected format: E## (epic), E##-F## (feature), E##-F##-### (task), B### (bug), CC-### (change card, C###/CC### aliases accepted), TD-### (tech-debt), I-YYYY-MM-DD-## (idea), or S### (sprint)", key)
}

func runCreateNote(cmd *cobra.Command, args []string) error {
	entityKey := args[0]
	content := args[1]

	noteTypeStr, _ := cmd.Flags().GetString("type")
	createdBy, _ := cmd.Flags().GetString("created-by")
	metadata, _ := cmd.Flags().GetString("metadata")

	// Auto-detect entity type from key
	entityType, _, err := resolveEntityFromKey(entityKey)
	if err != nil {
		return err
	}

	noteSvc, err := cli.GetNoteService(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get note service: %w", err)
	}

	note, err := noteSvc.AddNoteWithMetadata(cmd.Context(), entityType, entityKey, noteTypeStr, content, createdBy, metadata)
	if err != nil {
		return fmt.Errorf("failed to add note: %w", err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(note)
	}

	creator := "unknown"
	if note.CreatedBy != nil {
		creator = *note.CreatedBy
	}
	ts := note.CreatedAt
	if ts.IsZero() {
		ts = time.Now()
	}

	fmt.Printf("Note added to %s\n\n", entityKey)
	fmt.Printf("[%s] %s (%s)\n", strings.ToUpper(noteTypeStr), ts.Format("2006-01-02 15:04"), creator)
	fmt.Println(content)

	return nil
}

func init() {
	// Register subcommands
	createCmd.AddCommand(createEpicCmd)
	createCmd.AddCommand(createFeatureCmd)
	createCmd.AddCommand(createTaskCmd)
	createCmd.AddCommand(createBugCmd)
	createCmd.AddCommand(createChangeCmd)
	createCmd.AddCommand(createChangeCardCmd)
	createCmd.AddCommand(createTechDebtCmd)
	createCmd.AddCommand(createIdeaCmd)
	createCmd.AddCommand(createQuestionCmd)
	createCmd.AddCommand(createNoteCmd)

	// Tech-debt alias under `shark create` needs the same flags as
	// `shark td create` since cobra subcommands don't inherit flags from
	// peer commands. registerTdCreateFlags is the shared helper.
	registerTdCreateFlags(createTechDebtCmd)
	createQuestionCmd.Flags().String("summary", "", "Question summary")
	createQuestionCmd.Flags().String("requester", "", "Question requester")
	createQuestionCmd.Flags().String("description", "", "Question description")
	createQuestionCmd.Flags().Bool("blocking", false, "Question blocks progress")

	// ======================================================================
	// Note Create Flags
	// ======================================================================
	createNoteCmd.Flags().String("type", "comment", "Note type: comment, decision, blocker, solution, reference, implementation, testing, future, question")
	createNoteCmd.Flags().String("created-by", "", "Author name (optional)")
	createNoteCmd.Flags().String("metadata", "", "Structured JSON object stored with the note (e.g. review-finding fields: gate, round, severity, defect_class, fingerprint)")

	// ======================================================================
	// Epic Create Flags
	// ======================================================================
	// Note: epicCreateDescription and epicCreateKey are package-level variables
	// declared in epic.go and must be bound using StringVar
	createEpicCmd.Flags().StringVar(&epicCreateDescription, "description", "", "Epic description (optional)")
	createEpicCmd.Flags().StringVar(&epicCreateKey, "key", "", "Custom key for the epic (e.g., E00, bugs). If not provided, auto-generates next E## number")

	// File path flags: --file is primary, --filename and --path are hidden aliases
	createEpicCmd.Flags().String("file", "", "Full file path (e.g., docs/custom/epic.md)")
	createEpicCmd.Flags().String("filename", "", "Alias for --file")
	createEpicCmd.Flags().String("path", "", "Alias for --file")
	_ = createEpicCmd.Flags().MarkHidden("filename")
	_ = createEpicCmd.Flags().MarkHidden("path")

	createEpicCmd.Flags().Bool("force", false, "Force reassignment if file already claimed by another epic or feature")
	createEpicCmd.Flags().String("priority", "medium", "Priority: low, medium, high (default: medium)")
	createEpicCmd.Flags().String("business-value", "", "Business value: low, medium, high (optional)")
	createEpicCmd.Flags().String("status", "draft", "Status: draft, active, completed, archived (default: draft)")
	createEpicCmd.Flags().StringSlice("tag", nil,
		"Tag to apply (repeatable). Tag must be registered; see 'shark tags list'.")
	// --size must be registered on each unified subcommand; Cobra flags are per-command, not shared via RunE.
	createEpicCmd.Flags().String("size", "", "Entity size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL")

	// ======================================================================
	// Feature Create Flags
	// ======================================================================
	// Note: featureCreateEpic, featureCreateDescription, featureCreateExecutionOrder,
	// featureCreateKey, featureCreateForce are package-level variables declared in feature.go
	createFeatureCmd.Flags().StringVar(&featureCreateEpic, "epic", "", "Epic key (e.g., E01) - can also be specified as first positional argument")
	createFeatureCmd.Flags().StringVar(&featureCreateDescription, "description", "", "Feature description (optional)")
	createFeatureCmd.Flags().IntVar(&featureCreateExecutionOrder, "execution-order", 0, "Execution order (optional, 0 = not set)")
	_ = createFeatureCmd.Flags().MarkDeprecated("execution-order", "use --order instead")
	createFeatureCmd.Flags().IntVar(&featureCreateExecutionOrder, "order", 0, "Execution order (lower runs first)")
	createFeatureCmd.Flags().StringVar(&featureCreateKey, "key", "", "Custom key for the feature (e.g., auth, F00). If not provided, auto-generates next F## number")
	createFeatureCmd.Flags().BoolVar(&featureCreateForce, "force", false, "Force reassignment if file already claimed by another feature or epic")
	createFeatureCmd.Flags().String("status", "draft", "Status: draft, active, completed, archived (default: draft)")
	createFeatureCmd.Flags().StringSlice("tag", nil,
		"Tag to apply (repeatable). Tag must be registered; see 'shark tags list'.")

	// File path flags: --file is primary, --filename and --path are hidden aliases
	createFeatureCmd.Flags().String("file", "", "Full file path (e.g., docs/custom/feature.md)")
	createFeatureCmd.Flags().String("filename", "", "Alias for --file")
	createFeatureCmd.Flags().String("path", "", "Alias for --file")
	_ = createFeatureCmd.Flags().MarkHidden("filename")
	_ = createFeatureCmd.Flags().MarkHidden("path")
	createFeatureCmd.Flags().String("size", "", "Entity size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL")

	// ======================================================================
	// Task Create Flags
	// ======================================================================
	// Note: task create reads flags from cmd.Flags() directly, not package-level variables
	createTaskCmd.Flags().StringP("epic", "e", "", "Epic key (e.g., E07) - can also be specified as first positional argument")
	createTaskCmd.Flags().StringP("feature", "f", "", "Feature key (e.g., F01) - can also be specified as second positional argument")
	createTaskCmd.Flags().StringP("agent", "a", "", "Agent type (e.g., backend, frontend, qa)")
	createTaskCmd.Flags().StringP("description", "d", "", "Detailed description")
	createTaskCmd.Flags().IntP("priority", "p", 5, "Priority (1=highest, 10=lowest)")
	createTaskCmd.Flags().String("depends-on", "", "Comma-separated dependency task keys")
	createTaskCmd.Flags().Int("execution-order", 0, "Execution order (optional, 0 = not set)")
	_ = createTaskCmd.Flags().MarkDeprecated("execution-order", "use --order instead")
	createTaskCmd.Flags().Int("order", 0, "Execution order (lower runs first)")
	createTaskCmd.Flags().String("key", "", "Custom key for the task")
	createTaskCmd.Flags().Bool("force", false, "Force reassignment if file already claimed by another task")
	createTaskCmd.Flags().Bool("create", false, "Create file if doesn't exist")
	createTaskCmd.Flags().StringSlice("tag", nil,
		"Tag to apply (repeatable). Tag must be registered; see 'shark tags list'.")

	// File path flags: --file is primary, --filename and --path are hidden aliases
	createTaskCmd.Flags().String("file", "", "Full file path (e.g., docs/custom/task.md)")
	createTaskCmd.Flags().String("filename", "", "Alias for --file")
	createTaskCmd.Flags().String("path", "", "Alias for --file")
	_ = createTaskCmd.Flags().MarkHidden("filename")
	_ = createTaskCmd.Flags().MarkHidden("path")
	createTaskCmd.Flags().String("size", "", "Entity size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL")

	// ======================================================================
	// Bug Create Flags
	// ======================================================================
	createBugCmd.Flags().String("severity", "medium", "Severity: critical, high, medium, low (default: medium)")
	createBugCmd.Flags().String("description", "", "Bug description (optional)")
	createBugCmd.Flags().String("linked-type", "", "Linked entity type: epic, feature, or task (optional)")
	createBugCmd.Flags().String("linked-key", "", "Linked entity key (e.g., E07-F01-001) - requires --linked-type")
	createBugCmd.Flags().StringSlice("tag", nil,
		"Tag to apply (repeatable). Tag must be registered; see 'shark tags list'.")
	createBugCmd.Flags().String("size", "", "Entity size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL")

	// ======================================================================
	// Idea Create Flags
	// ======================================================================
	createIdeaCmd.Flags().StringVar(&ideaDescription, "description", "", "Idea description (optional)")
	createIdeaCmd.Flags().IntVar(&ideaPriority, "priority", 0, "Priority (1-10, optional)")
	createIdeaCmd.Flags().IntVar(&ideaOrder, "order", 0, "Order for sorting (optional)")
	createIdeaCmd.Flags().StringVar(&ideaNotes, "notes", "", "Additional notes (optional)")
	createIdeaCmd.Flags().StringSliceVar(&ideaRelatedDocs, "related-docs", []string{}, "Related document paths (optional)")
	createIdeaCmd.Flags().StringSliceVar(&ideaDependencies, "depends-on", []string{}, "Dependent idea keys (optional)")
	createIdeaCmd.Flags().StringVar(&ideaStatus, "status", "new", "Initial status (new, on_hold, converted, archived)")
	createIdeaCmd.Flags().StringSlice("tag", nil,
		"Tag to apply (repeatable). Tag must be registered; see 'shark tags list'.")

	// ======================================================================
	// Change-Card Create Flags (applied to both 'change' and 'change-card')
	// ======================================================================
	for _, cmd := range []*cobra.Command{createChangeCmd, createChangeCardCmd} {
		cmd.Flags().String("description", "", "Change card description (optional)")
		cmd.Flags().String("justification", "", "Business justification for the change (optional)")
		cmd.Flags().String("requested-by", "", "Name or team requesting the change (optional)")
		cmd.Flags().String("epic", "", "Link to epic key (e.g., E07) (optional)")
		cmd.Flags().String("feature", "", "Link to feature key (e.g., E07-F01) (optional)")
		cmd.Flags().StringSlice("tag", nil,
			"Tag to apply (repeatable). Tag must be registered; see 'shark tags list'.")
		cmd.Flags().String("size", "", "Entity size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL")
	}
}
