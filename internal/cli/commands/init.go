package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	init_pkg "github.com/jwwelbor/shark-task-manager/internal/init"
	"github.com/spf13/cobra"
)

var (
	initNonInteractive bool
	initForce          bool
	workflowName       string
	updateForce        bool
	updateDryRun       bool
)

var initCmd = &cobra.Command{
	Use:     "init",
	Short:   "Initialize Shark CLI infrastructure",
	GroupID: "setup",
	Long: `Initialize Shark CLI infrastructure by creating database schema,
folder structure, configuration file, and task templates.

This command is idempotent and safe to run multiple times.`,
	Example: `  # Initialize with default settings
  shark init

  # Initialize without prompts (for automation)
  shark init --non-interactive

  # Force overwrite existing config
  shark init --force`,
	RunE: runInit,
}

var initUpdateCmd = &cobra.Command{
	Use:   "update [flags]",
	Short: "Update Shark configuration",
	Long: `Update Shark configuration with workflow profiles or add missing fields.

Without --workflow flag, adds missing configuration fields while preserving
all existing values.

With --workflow flag, applies the specified workflow profile (basic or advanced).

Use --dry-run to preview changes before applying.`,
	Example: `  # Add missing fields only
  shark init update

  # Apply basic workflow (5 statuses)
  shark init update --workflow=basic

  # Apply advanced workflow (19 statuses)
  shark init update --workflow=advanced

  # Preview changes without applying
  shark init update --workflow=advanced --dry-run

  # Force overwrite existing status configurations
  shark init update --workflow=basic --force`,
	RunE: runInitUpdate,
}

func init() {
	cli.RootCmd.AddCommand(initCmd)
	initCmd.AddCommand(initUpdateCmd)

	// Existing init flags
	initCmd.Flags().BoolVar(&initNonInteractive, "non-interactive", false,
		"Skip all prompts (use defaults)")
	initCmd.Flags().BoolVar(&initForce, "force", false,
		"Overwrite existing config and templates")

	// Update subcommand flags
	initUpdateCmd.Flags().StringVar(&workflowName, "workflow", "",
		"Apply workflow profile (basic, advanced)")
	initUpdateCmd.Flags().BoolVar(&updateForce, "force", false,
		"Overwrite existing status configurations")
	initUpdateCmd.Flags().BoolVar(&updateDryRun, "dry-run", false,
		"Preview changes without applying")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Get database path from global config
	dbPath, err := cli.GetDBPath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	// Create backup before init (even though init shouldn't modify existing data)
	// This protects against unexpected issues
	if _, err := os.Stat(dbPath); err == nil {
		// Database exists, create backup
		backupPath, err := db.BackupDatabase(dbPath)
		if err != nil {
			cli.Warning(fmt.Sprintf("Failed to create backup: %v", err))
			// Continue anyway - init should be safe
		} else {
			if !cli.GlobalConfig.JSON {
				cli.Info(fmt.Sprintf("Database backup created: %s", backupPath))
			}
		}
	}

	// Create initializer options
	opts := init_pkg.InitOptions{
		DBPath:         dbPath,
		ConfigPath:     ".sharkconfig.json", // Default
		NonInteractive: initNonInteractive || cli.GlobalConfig.JSON,
		Force:          initForce,
	}

	// Create initializer
	initializer := init_pkg.NewInitializer()

	// Run initialization with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := initializer.Initialize(ctx, opts)
	if err != nil {
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			})
		}
		return err
	}

	// If config was created, apply basic profile by default
	if result.ConfigCreated {
		profileService := init_pkg.NewProfileService(result.ConfigPath)
		profileOpts := init_pkg.UpdateOptions{
			ConfigPath:     result.ConfigPath,
			WorkflowName:   "basic", // Apply basic profile by default
			DryRun:         false,
			Force:          false,
			NonInteractive: initNonInteractive || cli.GlobalConfig.JSON,
			Verbose:        cli.GlobalConfig.Verbose,
		}

		_, err := profileService.ApplyProfile(profileOpts)
		if err != nil {
			if cli.GlobalConfig.JSON {
				return cli.OutputJSON(map[string]interface{}{
					"status": "error",
					"error":  fmt.Sprintf("failed to apply default workflow profile: %v", err),
				})
			}
			return fmt.Errorf("failed to apply default workflow profile: %w", err)
		}
	}

	// Output results
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"status":           "success",
			"database_created": result.DatabaseCreated,
			"database_path":    result.DatabasePath,
			"folders_created":  result.FoldersCreated,
			"config_created":   result.ConfigCreated,
			"config_path":      result.ConfigPath,
			"templates_copied": result.TemplatesCopied,
		})
	}

	displayInitSuccess(result)
	return nil
}

func displayInitSuccess(result *init_pkg.InitResult) {
	cli.Success("Shark CLI initialized successfully!")
	fmt.Println()

	if result.DatabaseCreated {
		fmt.Printf("✓ Database created: %s\n", result.DatabasePath)
	} else {
		fmt.Printf("✓ Database exists: %s\n", result.DatabasePath)
	}

	if len(result.FoldersCreated) > 0 {
		for _, folder := range result.FoldersCreated {
			fmt.Printf("✓ Folder created: %s\n", folder)
		}
	} else {
		fmt.Println("✓ Folder structure exists: docs/plan/, shark-templates/")
	}

	if result.ConfigCreated {
		fmt.Printf("✓ Config file created: %s\n", result.ConfigPath)
	} else {
		fmt.Printf("✓ Config file exists: %s\n", result.ConfigPath)
	}

	if result.TemplatesCopied > 0 {
		fmt.Printf("✓ Templates copied: %d files\n", result.TemplatesCopied)
	} else {
		fmt.Println("✓ Templates exist")
	}

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("1. Edit .sharkconfig.json to set default epic and agent")
	fmt.Println("2. Create tasks with: shark task create \"Task title\" --epic=E01 --feature=F01 --agent=backend")
	fmt.Println("3. Import existing tasks with: shark sync")

	// Only show profile message if config was created
	if result.ConfigCreated {
		fmt.Println()
		fmt.Println("Workflow profile applied: basic (5 statuses: todo, in_progress, ready_for_review, completed, blocked)")
		fmt.Println("To upgrade to advanced profile: shark init update --workflow=advanced")
	}
}

func runInitUpdate(cmd *cobra.Command, args []string) error {
	// Determine config path
	configPath := ".sharkconfig.json"
	if configFlag := cmd.Flag("config"); configFlag != nil && configFlag.Value.String() != "" {
		configPath = configFlag.Value.String()
	}

	// Create profile service
	service := init_pkg.NewProfileService(configPath)

	// Build update options
	opts := init_pkg.UpdateOptions{
		ConfigPath:     configPath,
		WorkflowName:   workflowName,
		Force:          updateForce,
		DryRun:         updateDryRun,
		NonInteractive: cli.GlobalConfig.JSON,
		Verbose:        cli.GlobalConfig.Verbose,
	}

	// Apply profile (or add missing fields if no workflow specified)
	result, err := service.ApplyProfile(opts)
	if err != nil {
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
		}
		return fmt.Errorf("failed to update config: %w", err)
	}

	// Output results
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}

	displayUpdateResult(result)
	return nil
}

func displayUpdateResult(result *init_pkg.UpdateResult) {
	// Header
	if result.DryRun {
		cli.Info("DRY RUN - No changes applied")
		fmt.Println()
	}

	// Success message
	if result.ProfileName != "" {
		cli.Success(fmt.Sprintf("Applied %s workflow profile", result.ProfileName))
	} else {
		cli.Success("Updated configuration")
	}
	fmt.Println()

	// Backup info
	if result.BackupPath != "" {
		fmt.Printf("✓ Backed up config to %s\n", result.BackupPath)
		fmt.Println()
	}

	// Changes summary
	changes := result.Changes
	if len(changes.Added) > 0 {
		fmt.Printf("  Added: %s\n", strings.Join(changes.Added, ", "))
	}
	if len(changes.Overwritten) > 0 {
		fmt.Printf("  Overwritten: %s\n", strings.Join(changes.Overwritten, ", "))
	}
	if len(changes.Preserved) > 0 {
		fmt.Printf("  Preserved: %s\n", strings.Join(changes.Preserved, ", "))
	}

	// Statistics
	if changes.Stats != nil {
		fmt.Println()
		fmt.Printf("  Statuses: %d added\n", changes.Stats.StatusesAdded)
		if changes.Stats.FlowsAdded > 0 {
			fmt.Printf("  Flows: %d added\n", changes.Stats.FlowsAdded)
		}
		if changes.Stats.GroupsAdded > 0 {
			fmt.Printf("  Groups: %d added\n", changes.Stats.GroupsAdded)
		}
		fmt.Printf("  Fields: %d preserved\n", changes.Stats.FieldsPreserved)
	}

	// Final status
	if !result.DryRun {
		fmt.Println()
		fmt.Printf("✓ Config updated: %s\n", result.ConfigPath)
	} else {
		fmt.Println()
		cli.Info("Run without --dry-run to apply these changes")
	}
}
