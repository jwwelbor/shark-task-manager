package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
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

	Long: `Initialize Shark CLI infrastructure: create the database, the
docs/plan/ folder, and a default .sharkconfig.json configured to use
shark-data/ for workflow and content.

Content (workflows, prompts, skills, agents) is served from the embedded
bundle by default — no shark-data/ directory required. Run
'shark admin install-shark-data' to extract the bundle to disk for
local customization.

This command is idempotent and safe to run multiple times.`,
	Example: `  # Initialize with default settings
  shark admin init

  # Initialize without prompts (for automation)
  shark admin init --non-interactive

  # Force overwrite existing config
  shark admin init --force`,
	RunE: runInit,
}

func init() {
	adminCmd.AddCommand(initCmd)

	initCmd.Flags().BoolVar(&initNonInteractive, "non-interactive", false,
		"Skip all prompts (use defaults)")
	initCmd.Flags().BoolVar(&initForce, "force", false,
		"Overwrite existing config")
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

	// Output results
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"status":           "success",
			"database_created": result.DatabaseCreated,
			"database_path":    result.DatabasePath,
			"folders_created":  result.FoldersCreated,
			"config_created":   result.ConfigCreated,
			"config_path":      result.ConfigPath,
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
		fmt.Println("✓ Folder structure exists: docs/plan/")
	}

	if result.ConfigCreated {
		fmt.Printf("✓ Config file created: %s\n", result.ConfigPath)
	} else {
		fmt.Printf("✓ Config file exists: %s\n", result.ConfigPath)
	}

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("1. Create your first epic with: shark epic create \"Epic Title\"")
	fmt.Println("2. Create your first task with: shark task create E01 F01 \"Task title\"")
	fmt.Println("3. Optionally extract workflow/prompt/skill defaults: shark admin install-shark-data")
}
