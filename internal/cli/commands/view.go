package commands

import (
	"fmt"
	"path/filepath"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/cli/scope"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/view"
	"github.com/spf13/cobra"
)

// viewCmd represents the view command
var viewCmd = &cobra.Command{
	Use:     "view <KEY>",
	Short:   "View epic, feature, or task specification in external viewer",
	GroupID: "essentials",
	Long: `View specification files in an external viewer (glow, nano, cat, etc.)

The viewer can be configured in .sharkconfig.json:
{
  "viewer": "glow"
}

Defaults to "cat" if not configured.

Positional Arguments:
  EPIC                  View epic spec (e.g., E04)
  EPIC FEATURE          View feature spec (e.g., E04 F01 or E04-F01)
  EPIC FEATURE TASKNUM  View task spec (e.g., E04 F01 001)
  FULL_TASK_KEY         View task spec (e.g., T-E04-F01-001)

Examples:
  shark view E10                    View epic E10 spec
  shark view E10 F01                View feature E10-F01 spec
  shark view E10-F01                View feature E10-F01 spec (combined)
  shark view E10 F01 001            View task T-E10-F01-001 spec
  shark view T-E10-F01-001          View task spec (full key)
  shark view e10                    View epic E10 (case insensitive)
`,
	RunE: runView,
}

func init() {
	// Register view command with root
	cli.RootCmd.AddCommand(viewCmd)
}

// runView executes the view command
func runView(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Get database
	repoDb, err := cli.GetDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	// Get project root for config loading
	projectRoot, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	// Load config for viewer setting
	configPath := filepath.Join(projectRoot, ".sharkconfig.json")
	cfgManager := config.NewManager(configPath)
	cfg, err := cfgManager.Load()
	if err != nil {
		// Non-fatal: config not found is okay, we'll use defaults
		cfg = &config.Config{}
	}

	// Parse scope from arguments using ScopeInterpreter
	interpreter := scope.NewInterpreter()
	parsedScope, err := interpreter.ParseScope(args)
	if err != nil {
		return err
	}

	// Create repositories
	epicRepo := repository.NewEpicRepository(repoDb)
	featureRepo := repository.NewFeatureRepository(repoDb)
	taskRepo := repository.NewTaskRepository(repoDb)

	// Create view service
	viewService := view.NewService(epicRepo, featureRepo, taskRepo)

	// Get file path
	filePath, err := viewService.GetFilePath(ctx, parsedScope)
	if err != nil {
		return fmt.Errorf("failed to get file path: %w", err)
	}

	// Launch viewer
	viewerCmd := cfg.GetViewer()
	if err := viewService.LaunchViewer(ctx, filePath, viewerCmd); err != nil {
		return fmt.Errorf("failed to launch viewer: %w", err)
	}

	return nil
}
