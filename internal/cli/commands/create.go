package commands

import (
	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
)

// createCmd is the parent command for creating entities
var createCmd = &cobra.Command{
	Use:     "create <type> [args]",
	Short:   "Create an epic, feature, or task",
	GroupID: "essentials",
	Long: `Create a new entity. Dispatches to the appropriate create handler based on type.

Examples:
  shark create epic "Q1 2025 Roadmap"
  shark create feature E07 "User Authentication"
  shark create task E07 F01 "Implement login" --agent=backend --priority=5`,
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

func init() {
	// Register parent command
	cli.RootCmd.AddCommand(createCmd)

	// Register subcommands
	createCmd.AddCommand(createEpicCmd)
	createCmd.AddCommand(createFeatureCmd)
	createCmd.AddCommand(createTaskCmd)

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
	createTaskCmd.Flags().Int("order", 0, "Execution order (alias for --execution-order)")
	createTaskCmd.Flags().String("key", "", "Custom key for the task")
	createTaskCmd.Flags().Bool("force", false, "Force reassignment if file already claimed by another task")
	createTaskCmd.Flags().Bool("create", false, "Create file if doesn't exist")

	// File path flags: --file is primary, --filename and --path are hidden aliases
	createTaskCmd.Flags().String("file", "", "Full file path (e.g., docs/custom/task.md)")
	createTaskCmd.Flags().String("filename", "", "Alias for --file")
	createTaskCmd.Flags().String("path", "", "Alias for --file")
	_ = createTaskCmd.Flags().MarkHidden("filename")
	_ = createTaskCmd.Flags().MarkHidden("path")
}
