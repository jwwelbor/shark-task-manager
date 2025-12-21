package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	init_pkg "github.com/jwwelbor/shark-task-manager/internal/init"
	"github.com/spf13/cobra"
)

var (
	initNonInteractive bool
	initForce          bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Shark CLI infrastructure",
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

func init() {
	cli.RootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolVar(&initNonInteractive, "non-interactive", false,
		"Skip all prompts (use defaults)")
	initCmd.Flags().BoolVar(&initForce, "force", false,
		"Overwrite existing config and templates")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Get database path from global config
	dbPath, err := cli.GetDBPath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
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
}
