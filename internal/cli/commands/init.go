package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
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
docs/plan/ and shark-templates/ folder structure, a default .sharkconfig.json,
and copy the embedded shark-templates/ tree.

The shark-templates/ tree is re-synced from the embedded version on every run
so template/workflow updates shipped with a new shark binary flow through
automatically. Files you have modified locally are NOT overwritten — they are
reported as differing from the shipped version, and you can re-run with
--force to accept the upstream version.

This command is idempotent and safe to run multiple times.`,
	Example: `  # Initialize with default settings
  shark admin init

  # Initialize without prompts (for automation)
  shark admin init --non-interactive

  # Force overwrite existing config and locally-modified templates
  shark admin init --force`,
	RunE: runInit,
}

func init() {
	adminCmd.AddCommand(initCmd)

	initCmd.Flags().BoolVar(&initNonInteractive, "non-interactive", false,
		"Skip all prompts (use defaults)")
	initCmd.Flags().BoolVar(&initForce, "force", false,
		"Overwrite existing config and locally-modified templates")
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

	// Get template directory from existing config (if any)
	configPath, _ := cli.GetConfigPath()
	templateDir := config.GetTemplateDirectoryFromConfig(configPath)

	// Create initializer options
	opts := init_pkg.InitOptions{
		DBPath:         dbPath,
		ConfigPath:     ".sharkconfig.json", // Default
		NonInteractive: initNonInteractive || cli.GlobalConfig.JSON,
		Force:          initForce,
		TemplateDir:    templateDir,
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
			"status":              "success",
			"database_created":    result.DatabaseCreated,
			"database_path":       result.DatabasePath,
			"folders_created":     result.FoldersCreated,
			"config_created":      result.ConfigCreated,
			"config_path":         result.ConfigPath,
			"templates_copied":    result.TemplatesCopied,
			"templates_refreshed": result.TemplatesRefreshed,
			"templates_differed":  result.TemplatesDiffered,
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

	switch {
	case result.TemplatesCopied > 0 && result.TemplatesRefreshed > 0:
		fmt.Printf("✓ Templates: %d copied, %d refreshed\n", result.TemplatesCopied, result.TemplatesRefreshed)
	case result.TemplatesCopied > 0:
		fmt.Printf("✓ Templates copied: %d files\n", result.TemplatesCopied)
	case result.TemplatesRefreshed > 0:
		fmt.Printf("✓ Templates refreshed: %d files\n", result.TemplatesRefreshed)
	default:
		fmt.Println("✓ Templates up to date")
	}

	// Surface locally-modified templates that diverge from the shipped version
	// so the user knows an upgrade is available behind --force.
	if len(result.TemplatesDiffered) > 0 {
		fmt.Println()
		cli.Warning(fmt.Sprintf("%d template file(s) differ from the shipped version and were left unchanged:", len(result.TemplatesDiffered)))
		for _, p := range result.TemplatesDiffered {
			fmt.Printf("  - %s\n", p)
		}
		fmt.Println("  Re-run with --force to overwrite local changes with the shipped version.")
	}

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("1. Edit .sharkconfig.json to point at the workflow file you want (default: shark-templates/.sharkworkflow-short.json)")
	fmt.Println("2. Create your first epic with: shark epic create \"Epic Title\"")
	fmt.Println("3. Create your first task with: shark task create E01 F01 \"Task title\"")
}
