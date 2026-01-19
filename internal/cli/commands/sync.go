package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/reporting"
	"github.com/jwwelbor/shark-task-manager/internal/sync"
	"github.com/spf13/cobra"
)

var (
	syncFolder            string
	syncDryRun            bool
	syncStrategy          string
	syncCreateMissing     bool
	syncCleanup           bool
	syncPatterns          []string
	syncForceFullScan     bool
	syncOutput            string
	syncQuiet             bool
	syncIndex             bool
	syncDiscoveryStrategy string
	syncValidationLevel   string
)

var syncCmd = &cobra.Command{
	Use:     "sync",
	Short:   "Synchronize task files with database",
	GroupID: "setup",
	Long: `Synchronize task markdown files with the database by scanning feature folders,
parsing frontmatter, detecting conflicts, and applying resolution strategies.

Status is managed exclusively in the database and is NOT synced from files.`,
	Example: `  # Sync all feature folders (task pattern only)
  shark sync

  # Sync PRP files only
  shark sync --pattern=prp

  # Sync both task and PRP files
  shark sync --pattern=task --pattern=prp

  # Sync specific folder
  shark sync --folder=docs/plan/E04-task-mgmt-cli-core/E04-F06-task-creation

  # Preview changes without applying (dry-run)
  shark sync --dry-run

  # Use database-wins strategy for conflicts
  shark sync --strategy=database-wins

  # Manually resolve conflicts interactively
  shark sync --strategy=manual

  # Auto-create missing epics/features
  shark sync --create-missing

  # Delete orphaned database tasks (files deleted)
  shark sync --cleanup

  # Output as JSON for scripting
  shark sync --output=json

  # Quiet mode (minimal output)
  shark sync --quiet

  # Enable discovery mode with epic-index.md
  shark sync --index

  # Use index-only discovery strategy
  shark sync --index --discovery-strategy=index-only

  # Use merge strategy with permissive validation
  shark sync --index --discovery-strategy=merge --validation-level=permissive`,
	RunE: runSync,
}

func init() {
	// DISABLED: Sync command causes catastrophic data loss - wiped all task statuses
	// DO NOT RE-ENABLE without thorough investigation and fix
	// cli.RootCmd.AddCommand(syncCmd)

	syncCmd.Flags().StringVar(&syncFolder, "folder", "",
		"Sync specific folder only (default: docs/plan)")
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false,
		"Preview changes without applying them")
	syncCmd.Flags().StringVar(&syncStrategy, "strategy", "file-wins",
		"Conflict resolution strategy: file-wins, database-wins, newer-wins, manual")
	syncCmd.Flags().BoolVar(&syncCreateMissing, "create-missing", false,
		"Auto-create missing epics/features")
	syncCmd.Flags().BoolVar(&syncCleanup, "cleanup", false,
		"Delete orphaned database tasks (files deleted)")
	syncCmd.Flags().StringSliceVar(&syncPatterns, "pattern", []string{"task"},
		"File patterns to scan: task, prp (can specify multiple)")
	syncCmd.Flags().BoolVar(&syncForceFullScan, "force-full-scan", false,
		"Force full scan, ignoring incremental filtering")
	syncCmd.Flags().StringVar(&syncOutput, "output", "text",
		"Output format: text, json")
	syncCmd.Flags().BoolVar(&syncQuiet, "quiet", false,
		"Quiet mode (only show errors, useful for scripting)")
	syncCmd.Flags().BoolVar(&syncIndex, "index", false,
		"Enable discovery mode (parse epic-index.md)")
	syncCmd.Flags().StringVar(&syncDiscoveryStrategy, "discovery-strategy", "merge",
		"Discovery strategy: index-only, folder-only, merge")
	syncCmd.Flags().StringVar(&syncValidationLevel, "validation-level", "balanced",
		"Validation level: strict, balanced, permissive")
}

func runSync(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	// Get database path
	dbPath, err := cli.GetDBPath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	// Parse conflict strategy
	strategy, err := parseConflictStrategy(syncStrategy)
	if err != nil {
		return fmt.Errorf("invalid strategy: %w", err)
	}

	// Parse and validate patterns
	patterns, err := validatePatterns(syncPatterns)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	// Parse discovery strategy
	discoveryStrategy, err := parseDiscoveryStrategy(syncDiscoveryStrategy)
	if err != nil {
		return fmt.Errorf("invalid discovery strategy: %w", err)
	}

	// Parse validation level
	validationLevel, err := parseValidationLevel(syncValidationLevel)
	if err != nil {
		return fmt.Errorf("invalid validation level: %w", err)
	}

	// Default folder path
	folderPath := syncFolder
	if folderPath == "" {
		folderPath = "docs/plan"
	}

	// Load config to get last_sync_time
	configPath, err := cli.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}
	configManager := config.NewManager(configPath)
	cfg, err := configManager.Load()
	if err != nil {
		// Config load error is not fatal - just log warning and continue with full scan
		fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
	}

	// Create sync options
	opts := sync.SyncOptions{
		DBPath:            dbPath,
		FolderPath:        folderPath,
		DryRun:            syncDryRun,
		Strategy:          strategy,
		CreateMissing:     syncCreateMissing,
		Cleanup:           syncCleanup,
		ForceFullScan:     syncForceFullScan,
		LastSyncTime:      nil, // Will be set from config if available
		EnableDiscovery:   syncIndex,
		DiscoveryStrategy: discoveryStrategy,
		ValidationLevel:   validationLevel,
	}

	// Set last sync time from config (if not forcing full scan)
	if cfg != nil && !syncForceFullScan {
		opts.LastSyncTime = cfg.LastSyncTime
	}

	// Create sync engine with specified patterns
	engine, err := sync.NewSyncEngineWithPatterns(dbPath, patterns)
	if err != nil {
		return fmt.Errorf("failed to create sync engine: %w", err)
	}
	defer engine.Close()

	// Run sync with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	syncReport, err := engine.Sync(ctx, opts)
	if err != nil {
		if syncOutput == "json" || cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			})
		}
		return err
	}

	// Update last_sync_time in config after successful sync (non-dry-run only)
	if !syncDryRun && cfg != nil {
		syncTime := time.Now()
		if updateErr := configManager.UpdateLastSyncTime(syncTime); updateErr != nil {
			// Log warning but don't fail the sync
			fmt.Fprintf(os.Stderr, "Warning: Failed to update last_sync_time in config: %v\n", updateErr)
		}
	}

	// Convert sync report to scan report for enhanced reporting
	scanReport := convertToScanReport(syncReport, startTime, folderPath, patterns)

	// Output report
	if syncOutput == "json" || cli.GlobalConfig.JSON {
		fmt.Println(reporting.FormatJSON(scanReport))
		return nil
	}

	if !syncQuiet {
		// Use color output if terminal supports it
		useColor := isTerminal()
		fmt.Println(reporting.FormatCLI(scanReport, useColor))
	} else {
		// Quiet mode: only show errors
		if len(syncReport.Errors) > 0 {
			for _, err := range syncReport.Errors {
				fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
			}
			return fmt.Errorf("sync completed with %d error(s)", len(syncReport.Errors))
		}
	}

	return nil
}

func parseConflictStrategy(s string) (sync.ConflictStrategy, error) {
	switch s {
	case "file-wins":
		return sync.ConflictStrategyFileWins, nil
	case "database-wins":
		return sync.ConflictStrategyDatabaseWins, nil
	case "newer-wins":
		return sync.ConflictStrategyNewerWins, nil
	case "manual":
		return sync.ConflictStrategyManual, nil
	default:
		return "", fmt.Errorf("unknown strategy: %s (valid: file-wins, database-wins, newer-wins, manual)", s)
	}
}

func validatePatterns(patternStrings []string) ([]sync.PatternType, error) {
	validPatterns := map[string]sync.PatternType{
		"task": sync.PatternTypeTask,
		"prp":  sync.PatternTypePRP,
	}

	result := make([]sync.PatternType, 0, len(patternStrings))
	for _, ps := range patternStrings {
		pt, ok := validPatterns[ps]
		if !ok {
			return nil, fmt.Errorf("unknown pattern: %s (valid: task, prp)", ps)
		}
		result = append(result, pt)
	}

	return result, nil
}

func parseDiscoveryStrategy(s string) (sync.DiscoveryStrategy, error) {
	switch s {
	case "index-only":
		return sync.DiscoveryStrategyIndexOnly, nil
	case "folder-only":
		return sync.DiscoveryStrategyFolderOnly, nil
	case "merge":
		return sync.DiscoveryStrategyMerge, nil
	default:
		return "", fmt.Errorf("unknown discovery strategy: %s (valid: index-only, folder-only, merge)", s)
	}
}

func parseValidationLevel(s string) (sync.ValidationLevel, error) {
	switch s {
	case "strict":
		return sync.ValidationLevelStrict, nil
	case "balanced":
		return sync.ValidationLevelBalanced, nil
	case "permissive":
		return sync.ValidationLevelPermissive, nil
	default:
		return "", fmt.Errorf("unknown validation level: %s (valid: strict, balanced, permissive)", s)
	}
}

// convertToScanReport converts a sync.SyncReport to reporting.ScanReport
func convertToScanReport(syncReport *sync.SyncReport, startTime time.Time, folderPath string, patterns []sync.PatternType) *reporting.ScanReport {
	scanReport := reporting.NewScanReport()

	// Set metadata
	scanReport.Metadata = reporting.ScanMetadata{
		Timestamp:         startTime,
		DurationSeconds:   time.Since(startTime).Seconds(),
		ValidationLevel:   "basic",
		DocumentationRoot: folderPath,
		Patterns:          make(map[string]string),
	}

	// Add patterns to metadata
	for _, p := range patterns {
		scanReport.Metadata.Patterns[string(p)] = "enabled"
	}

	// Set dry run flag
	scanReport.SetDryRun(syncReport.DryRun)

	// Set counts
	scanReport.Counts.Scanned = syncReport.FilesScanned
	scanReport.Counts.Matched = syncReport.TasksImported + syncReport.TasksUpdated
	scanReport.Counts.Skipped = len(syncReport.Errors) + len(syncReport.Warnings)

	// Set entity counts (tasks only for now)
	scanReport.Entities.Tasks.Matched = syncReport.TasksImported + syncReport.TasksUpdated
	scanReport.Entities.Tasks.Skipped = len(syncReport.Errors)

	// Add discovery information if enabled
	if syncReport.DiscoveryReport != nil {
		dr := syncReport.DiscoveryReport
		scanReport.Entities.Epics.Matched = dr.EpicsImported
		scanReport.Entities.Features.Matched = dr.FeaturesImported
		// Note: Total discovered counts are in dr.EpicsDiscovered and dr.FeaturesDiscovered
		// but the reporting structure doesn't have a field for "discovered" vs "imported"
	}

	// Convert warnings
	for _, warning := range syncReport.Warnings {
		scanReport.AddWarning(reporting.SkippedFileEntry{
			FilePath:     folderPath,
			Reason:       warning,
			ErrorType:    "validation_warning",
			SuggestedFix: "Review warning and take appropriate action",
		})
	}

	// Convert errors
	for _, errMsg := range syncReport.Errors {
		scanReport.AddError(reporting.SkippedFileEntry{
			FilePath:     folderPath,
			Reason:       errMsg,
			ErrorType:    "parse_error",
			SuggestedFix: "Fix the error and re-run sync",
		})
	}

	// Update status based on errors
	scanReport.Status = scanReport.GetStatus()

	return scanReport
}

// isTerminal checks if stdout is a terminal (for color output)
func isTerminal() bool {
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
		return true
	}
	return false
}
