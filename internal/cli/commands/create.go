package commands

import (
	"github.com/spf13/cobra"
)

// createCmd is the parent command for creating entities
var createCmd = &cobra.Command{
	Use:     "create <type> [args]",
	Short:   "Create an epic, feature, task, bug, or change-card",
	GroupID: "manage",
	Long: `Create a new entity. Dispatches to the appropriate create handler based on type.

Examples:
  shark create epic "Q1 2025 Roadmap"
  shark create feature E07 "User Authentication"
  shark create task E07 F01 "Implement login" --agent=backend --priority=5
  shark create bug "Login page crashes on Safari" --severity=high
  shark create change "Migrate auth to OAuth2" --justification="Security requirement"`,
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

// createChangeCmd delegates to runChangeCardCreate (accepts "change" or "change-card")
var createChangeCmd = &cobra.Command{
	Use:   "change <title> [flags]",
	Short: "Create a new change-card",
	Long: `Create a new change-card with auto-generated key (CC-###).

Examples:
  shark create change "Migrate auth to OAuth2"
  shark create change "Update dependencies" --justification="Security patches"
  shark create change "Refactor DB layer" --requested-by="Product Team" --epic=E07`,
	Args: cobra.ExactArgs(1),
	RunE: runChangeCardCreate,
}

func init() {
	// Register subcommands
	createCmd.AddCommand(createEpicCmd)
	createCmd.AddCommand(createFeatureCmd)
	createCmd.AddCommand(createTaskCmd)
	createCmd.AddCommand(createBugCmd)
	createCmd.AddCommand(createChangeCmd)

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

	// File path flags: --file is primary, --filename and --path are hidden aliases
	createFeatureCmd.Flags().String("file", "", "Full file path (e.g., docs/custom/feature.md)")
	createFeatureCmd.Flags().String("filename", "", "Alias for --file")
	createFeatureCmd.Flags().String("path", "", "Alias for --file")
	_ = createFeatureCmd.Flags().MarkHidden("filename")
	_ = createFeatureCmd.Flags().MarkHidden("path")

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

	// File path flags: --file is primary, --filename and --path are hidden aliases
	createTaskCmd.Flags().String("file", "", "Full file path (e.g., docs/custom/task.md)")
	createTaskCmd.Flags().String("filename", "", "Alias for --file")
	createTaskCmd.Flags().String("path", "", "Alias for --file")
	_ = createTaskCmd.Flags().MarkHidden("filename")
	_ = createTaskCmd.Flags().MarkHidden("path")

	// ======================================================================
	// Bug Create Flags
	// ======================================================================
	createBugCmd.Flags().String("severity", "medium", "Severity: critical, high, medium, low (default: medium)")
	createBugCmd.Flags().String("description", "", "Bug description (optional)")
	createBugCmd.Flags().String("linked-type", "", "Linked entity type: epic, feature, or task (optional)")
	createBugCmd.Flags().String("linked-key", "", "Linked entity key (e.g., E07-F01-001) - requires --linked-type")

	// ======================================================================
	// Change-Card Create Flags
	// ======================================================================
	createChangeCmd.Flags().String("description", "", "Change-card description (optional)")
	createChangeCmd.Flags().String("justification", "", "Business justification for the change (optional)")
	createChangeCmd.Flags().String("requested-by", "", "Name or team requesting the change (optional)")
	createChangeCmd.Flags().String("epic", "", "Link to epic key (e.g., E07) (optional)")
	createChangeCmd.Flags().String("feature", "", "Link to feature key (e.g., E07-F01) (optional)")
}
