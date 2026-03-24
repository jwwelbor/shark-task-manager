package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/patterns"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// configCmd represents the config command group
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",

	Long: `View, validate, and test pattern configuration settings.

Examples:
  shark config show                           Show current configuration
  shark config show --patterns                Show only pattern configuration
  shark config validate-patterns              Validate all patterns in config
  shark config test-pattern                   Test a pattern against a string
  shark config get-format --type=task         Get generation format for entity type`,
}

// configShowCmd shows current configuration
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current configuration including file location and all settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		patternsOnly, _ := cmd.Flags().GetBool("patterns")

		configFile := viper.ConfigFileUsed()
		if configFile == "" {
			cli.Info("No configuration file loaded (using defaults)")
		} else {
			cli.Info(fmt.Sprintf("Configuration file: %s", configFile))
		}

		svc := cli.GetConfigService()

		if patternsOnly {
			return runShowPatternsConfig(svc, configFile)
		}

		result, err := svc.BuildShowConfig(configFile,
			viper.GetBool("json"),
			viper.GetBool("no-color"),
			viper.GetBool("verbose"),
		)
		if err != nil {
			return err
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}

		fmt.Println("\nCurrent Settings:")
		settings := map[string]interface{}{
			"json":     result.JSON,
			"no-color": result.NoColor,
			"verbose":  result.Verbose,
		}
		if result.Database != nil {
			settings["database"] = result.Database
		}
		if result.WorkflowSources != nil {
			settings["workflow_sources"] = result.WorkflowSources
		}

		for key, value := range settings {
			if key == "database" {
				dbMap, ok := value.(map[string]interface{})
				if !ok {
					fmt.Printf("  %s: %v\n", key, value)
					continue
				}
				fmt.Printf("  %s:\n", key)
				for k, v := range dbMap {
					fmt.Printf("    %s: %v\n", k, v)
				}
			} else if key == "workflow_sources" {
				sourcesMap, ok := value.(map[string]string)
				if !ok {
					fmt.Printf("  %s: %v\n", key, value)
					continue
				}
				fmt.Printf("  %s:\n", key)
				for k, v := range sourcesMap {
					fmt.Printf("    %s_workflow: %s\n", k, v)
				}
			} else {
				fmt.Printf("  %s: %v\n", key, value)
			}
		}

		// Always note patterns are available via --patterns
		fmt.Printf("  %s: (use --patterns flag to view)\n", "patterns")

		return nil
	},
}

// runShowPatternsConfig displays only the patterns configuration.
func runShowPatternsConfig(svc interface {
	PatternsOnlyConfig(string) (*patterns.PatternConfig, error)
}, configFile string) error {
	patternsConfig, err := svc.PatternsOnlyConfig(configFile)
	if err != nil {
		return err
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]interface{}{
			"patterns": patternsConfig,
		})
	}

	fmt.Println("\nPattern Configuration:")

	// Epic patterns
	fmt.Println("\nEpic:")
	fmt.Println("  Folder patterns:")
	for i, p := range patternsConfig.Epic.Folder {
		fmt.Printf("    %d. %s\n", i+1, p)
	}
	fmt.Println("  File patterns:")
	for i, p := range patternsConfig.Epic.File {
		fmt.Printf("    %d. %s\n", i+1, p)
	}
	fmt.Printf("  Generation format: %s\n", patternsConfig.Epic.Generation.Format)

	// Feature patterns
	fmt.Println("\nFeature:")
	fmt.Println("  Folder patterns:")
	for i, p := range patternsConfig.Feature.Folder {
		fmt.Printf("    %d. %s\n", i+1, p)
	}
	fmt.Println("  File patterns:")
	for i, p := range patternsConfig.Feature.File {
		fmt.Printf("    %d. %s\n", i+1, p)
	}
	fmt.Printf("  Generation format: %s\n", patternsConfig.Feature.Generation.Format)

	// Task patterns
	fmt.Println("\nTask:")
	if len(patternsConfig.Task.Folder) > 0 {
		fmt.Println("  Folder patterns:")
		for i, p := range patternsConfig.Task.Folder {
			fmt.Printf("    %d. %s\n", i+1, p)
		}
	}
	fmt.Println("  File patterns:")
	for i, p := range patternsConfig.Task.File {
		fmt.Printf("    %d. %s\n", i+1, p)
	}
	fmt.Printf("  Generation format: %s\n", patternsConfig.Task.Generation.Format)

	return nil
}

// configValidateCmd validates configuration
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	Long: `Check configuration file for errors and validate settings.
Validates both .sharkconfig.json and .sharkworkflow.json (if present).
Reports workflow sources, duplicate definitions, and structural issues.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile, err := cli.GetConfigPath()
		if err != nil || configFile == "" {
			cli.Warning("No configuration file to validate")
			return nil
		}

		results := config.ValidateWorkflowFiles(configFile)

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(results)
		}

		hasErrors := false
		hasWarnings := false

		fmt.Println("\nWorkflow Sources:")
		for _, r := range results {
			if r.Level == "info" {
				fmt.Printf("  %s\n", r.Message)
			}
		}

		for _, r := range results {
			if r.Level == "warning" {
				hasWarnings = true
				cli.Warning(r.Message)
			}
		}

		for _, r := range results {
			if r.Level == "error" {
				hasErrors = true
				cli.Error(r.Message)
			}
		}

		if hasErrors {
			fmt.Println()
			return fmt.Errorf("configuration validation failed")
		}

		if hasWarnings {
			cli.Info(fmt.Sprintf("\nConfiguration valid with warnings: %s", configFile))
		} else {
			cli.Success(fmt.Sprintf("\nConfiguration file is valid: %s", configFile))
		}
		return nil
	},
}

// configValidatePatternsCmd validates all patterns in configuration
var configValidatePatternsCmd = &cobra.Command{
	Use:   "validate-patterns",
	Short: "Validate all patterns in configuration",
	Long: `Validate all regex patterns in .sharkconfig.json.
Reports validation results grouped by entity type (epic, feature, task).
Exits with non-zero status if any errors found (for CI integration).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile, err := cli.GetConfigPath()
		if err != nil {
			return fmt.Errorf("failed to get config path: %w", err)
		}

		svc := cli.GetConfigService()

		patternsConfig, err := svc.LoadPatternsFromConfig(configFile)
		if err != nil {
			cli.Error(fmt.Sprintf("Failed to load patterns: %v", err))
			return err
		}

		report := svc.ValidateAllPatterns(patternsConfig)

		// Also run full validation for error detection
		validationErr := patterns.ValidatePatternConfig(patternsConfig)

		fmt.Println("\nPattern Validation Report:")
		fmt.Printf("  Epic patterns: %d valid, %d errors, %d warnings\n",
			report.EpicValid, len(report.EpicErrors), len(report.EpicWarnings))
		fmt.Printf("  Feature patterns: %d valid, %d errors, %d warnings\n",
			report.FeatureValid, len(report.FeatureErrors), len(report.FeatureWarnings))
		fmt.Printf("  Task patterns: %d valid, %d errors, %d warnings\n",
			report.TaskValid, len(report.TaskErrors), len(report.TaskWarnings))

		if report.HasErrors() {
			fmt.Println("\nErrors:")
			for _, e := range report.EpicErrors {
				cli.Error(fmt.Sprintf("  [ERROR] Epic: %s", e))
			}
			for _, e := range report.FeatureErrors {
				cli.Error(fmt.Sprintf("  [ERROR] Feature: %s", e))
			}
			for _, e := range report.TaskErrors {
				cli.Error(fmt.Sprintf("  [ERROR] Task: %s", e))
			}
		}

		if len(report.EpicWarnings) > 0 || len(report.FeatureWarnings) > 0 || len(report.TaskWarnings) > 0 {
			fmt.Println("\nWarnings:")
			for _, w := range report.EpicWarnings {
				cli.Warning(fmt.Sprintf("  [WARN] Epic: %s", w))
			}
			for _, w := range report.FeatureWarnings {
				cli.Warning(fmt.Sprintf("  [WARN] Feature: %s", w))
			}
			for _, w := range report.TaskWarnings {
				cli.Warning(fmt.Sprintf("  [WARN] Task: %s", w))
			}
		}

		if report.HasErrors() || validationErr != nil {
			fmt.Println("")
			return fmt.Errorf("pattern validation failed")
		}

		cli.Success("\nAll patterns validated successfully")
		return nil
	},
}

// configTestPatternCmd tests a pattern against a test string
var configTestPatternCmd = &cobra.Command{
	Use:   "test-pattern",
	Short: "Test a regex pattern against a test string",
	Long: `Test a regex pattern to see if it matches a test string.
Displays captured groups and validates pattern for specified entity type.

Examples:
  shark config test-pattern --pattern="E(?P<number>\d{2})-(?P<slug>[a-z-]+)" --test-string="E04-task-mgmt" --type=epic
  shark config test-pattern --pattern="T-E(?P<epic_num>\d{2})-F(?P<feature_num>\d{2})-(?P<number>\d{3})\.md" --test-string="T-E04-F07-003.md" --type=task`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pattern, _ := cmd.Flags().GetString("pattern")
		testString, _ := cmd.Flags().GetString("test-string")
		entityType, _ := cmd.Flags().GetString("type")

		if pattern == "" || testString == "" {
			return fmt.Errorf("--pattern and --test-string are required")
		}

		if entityType == "" {
			entityType = "epic"
		}

		svc := cli.GetConfigService()
		start := time.Now()

		result, err := svc.TestPattern(pattern, testString)
		duration := time.Since(start)

		fmt.Println("\nPattern Test Result:")
		fmt.Printf("  Pattern: %s\n", pattern)
		fmt.Printf("  Test String: %s\n", testString)

		if err != nil {
			fmt.Printf("  Match: false\n")
			cli.Error(fmt.Sprintf("  Pattern Error: %v\n", err))
			return err
		}

		fmt.Printf("  Match: %v\n", result.Matched)

		if result.Matched {
			if len(result.Groups) > 0 {
				fmt.Println("  Captured Groups:")
				for name, value := range result.Groups {
					fmt.Printf("    - %s: %s\n", name, value)
				}
			}
		} else {
			fmt.Println("  No match found")

			// Suggest similar patterns from config
			configFile, _ := cli.GetConfigPath()
			if configFile != "" && svc.FileExists(configFile) {
				patternsConfig, loadErr := svc.LoadPatternsFromConfig(configFile)
				if loadErr == nil {
					suggestions := svc.FindMatchingPatterns(patternsConfig, testString, entityType)
					if len(suggestions) > 0 {
						fmt.Println("\n  Similar patterns from config that match:")
						for _, s := range suggestions {
							fmt.Printf("    - %s\n", s)
						}
					}
				}
			}
		}

		// Validate pattern for entity type
		fmt.Println("\nValidation:")
		validationErr := patterns.ValidatePattern(pattern, entityType)
		if validationErr != nil {
			cli.Error(fmt.Sprintf("  Pattern invalid for %s type: %v", entityType, validationErr))
		} else {
			cli.Success(fmt.Sprintf("  Pattern valid for %s type", entityType))
		}

		// Check for warnings
		warnings := patterns.GetPatternWarnings(pattern, entityType)
		if len(warnings) > 0 {
			fmt.Println("\n  Warnings:")
			for _, warn := range warnings {
				cli.Warning(fmt.Sprintf("    - %s", warn))
			}
		}

		fmt.Printf("\nCompleted in %v\n", duration)

		return nil
	},
}

// configGetFormatCmd returns the generation format for an entity type
var configGetFormatCmd = &cobra.Command{
	Use:   "get-format",
	Short: "Get generation format for entity type",
	Long: `Query the configured generation format template for a specific entity type.
Supports JSON output for programmatic access (AI agents).

Examples:
  shark config get-format --type=task
  shark config get-format --type=epic --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		entityType, _ := cmd.Flags().GetString("type")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		if entityType == "" {
			return fmt.Errorf("--type is required (epic, feature, or task)")
		}

		svc := cli.GetConfigService()

		configFile, err := cli.GetConfigPath()
		if err != nil {
			return fmt.Errorf("failed to get config path: %w", err)
		}

		patternsConfig, err := svc.LoadPatternsFromConfig(configFile)
		if err != nil {
			patternsConfig = patterns.GetDefaultPatterns()
		}

		output, err := svc.GetFormat(patternsConfig, entityType)
		if err != nil {
			return err
		}

		if jsonOutput {
			return cli.OutputJSON(output)
		}

		fmt.Printf("\nGeneration Format for %s:\n", entityType)
		fmt.Printf("  Format: %s\n", output.Format)
		fmt.Printf("  Example: %s\n", output.Example)
		fmt.Printf("  Available placeholders: %s\n", strings.Join(output.Placeholders, ", "))

		return nil
	},
}

// configListPresetsCmd lists available pattern presets
var configListPresetsCmd = &cobra.Command{
	Use:   "list-presets",
	Short: "List available pattern presets",
	Long: `Display all available pattern presets with their descriptions.

Pattern presets provide pre-built pattern collections that can be added to your
configuration without writing regex from scratch.

Examples:
  shark config list-presets               List all available presets`,
	RunE: func(cmd *cobra.Command, args []string) error {
		presets := patterns.ListPresets()

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(presets)
		}

		fmt.Println("\nAvailable Pattern Presets:")
		for _, preset := range presets {
			fmt.Printf("  %-20s - %s\n", preset.Name, preset.Description)
		}
		fmt.Println()

		return nil
	},
}

// configShowPresetCmd shows details of a specific preset
var configShowPresetCmd = &cobra.Command{
	Use:   "show-preset <name>",
	Short: "Show details of a pattern preset",
	Long: `Display the full pattern structure for a specific preset.

The output shows patterns in JSON format ready for manual copying if needed.

Examples:
  shark config show-preset standard           Show standard preset
  shark config show-preset special-epics      Show special epics preset`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		presetName := args[0]

		preset, err := patterns.GetPreset(presetName)
		if err != nil {
			presets := patterns.ListPresets()
			cli.Error(fmt.Sprintf("Unknown preset: %s", presetName))
			fmt.Println("\nAvailable presets:")
			for _, p := range presets {
				fmt.Printf("  - %s\n", p.Name)
			}
			return err
		}

		info, _ := patterns.GetPresetInfo(presetName)

		if cli.GlobalConfig.JSON {
			output := map[string]interface{}{
				"name":        info.Name,
				"description": info.Description,
				"patterns":    preset,
			}
			return cli.OutputJSON(output)
		}

		fmt.Printf("\nPreset: %s\n", info.Name)
		fmt.Printf("Description: %s\n\n", info.Description)

		data, err := json.MarshalIndent(preset, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format preset: %w", err)
		}

		fmt.Println("Patterns:")
		fmt.Println(string(data))
		fmt.Println()

		fmt.Println("Affects:")
		if len(preset.Epic.Folder) > 0 || len(preset.Epic.File) > 0 {
			fmt.Printf("  - Epic patterns (%d folder, %d file)\n", len(preset.Epic.Folder), len(preset.Epic.File))
		}
		if len(preset.Feature.Folder) > 0 || len(preset.Feature.File) > 0 {
			fmt.Printf("  - Feature patterns (%d folder, %d file)\n", len(preset.Feature.Folder), len(preset.Feature.File))
		}
		if len(preset.Task.Folder) > 0 || len(preset.Task.File) > 0 {
			fmt.Printf("  - Task patterns (%d folder, %d file)\n", len(preset.Task.Folder), len(preset.Task.File))
		}
		fmt.Println()

		return nil
	},
}

// configAddPatternCmd adds a pattern preset to the configuration
var configAddPatternCmd = &cobra.Command{
	Use:   "add-pattern",
	Short: "Add pattern preset to configuration",
	Long: `Add a pattern preset to your .sharkconfig.json file.

Patterns from the preset are appended to existing configuration. Duplicate
patterns are automatically skipped. The configuration is validated after
the patterns are added.

Examples:
  shark config add-pattern --preset=special-epics    Add special epic patterns
  shark config add-pattern --preset=numeric-only     Add numeric-only patterns`,
	RunE: func(cmd *cobra.Command, args []string) error {
		presetName, _ := cmd.Flags().GetString("preset")

		if presetName == "" {
			cli.Error("Preset name is required. Use --preset=<name>")
			presets := patterns.ListPresets()
			fmt.Println("\nAvailable presets:")
			for _, p := range presets {
				fmt.Printf("  - %s\n", p.Name)
			}
			return fmt.Errorf("missing --preset flag")
		}

		configPath, err := cli.GetConfigPath()
		if err != nil {
			return fmt.Errorf("failed to get config path: %w", err)
		}

		svc := cli.GetConfigService()
		result, err := svc.AddPreset(configPath, presetName)
		if err != nil {
			// Show available presets if preset was unknown
			presets := patterns.ListPresets()
			cli.Error(fmt.Sprintf("Failed to add preset: %v", err))
			fmt.Println("\nAvailable presets:")
			for _, p := range presets {
				fmt.Printf("  - %s\n", p.Name)
			}
			return err
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(result)
		}

		cli.Success(fmt.Sprintf("Added preset '%s' to configuration", presetName))
		fmt.Printf("\nResults:\n")
		for _, detail := range result.Details {
			fmt.Printf("  - %s\n", detail)
		}
		fmt.Println()

		return nil
	},
}

// configGetStatusActionCmd returns the orchestrator action for a status
var configGetStatusActionCmd = &cobra.Command{
	Use:   "get-status-action <status>",
	Short: "Get orchestrator action for a status",
	Long: `Get the orchestrator action definition for a specific status from workflow configuration.

This command is useful for debugging and testing workflow configuration without actually
transitioning a task. Optionally provide a task key to populate template variables.

Examples:
  shark config get-status-action ready_for_development
  shark config get-status-action ready_for_development --task=T-E01-F03-002
  shark config get-status-action blocked --json`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigGetStatusAction,
}

func runConfigGetStatusAction(cmd *cobra.Command, args []string) error {
	status := args[0]
	taskKeyFlag, _ := cmd.Flags().GetString("task")

	configPath, err := cli.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	svc := cli.GetConfigService()

	// Resolve task vars if a task key is provided
	var taskVars map[string]string
	if taskKeyFlag != "" {
		taskKey, keyErr := NormalizeTaskKey(taskKeyFlag)
		if keyErr != nil {
			return fmt.Errorf("invalid task key format: %s", taskKeyFlag)
		}

		task, taskErr := cli.GetTaskService().GetTask(cmd.Context(), taskKey)
		if taskErr == nil && task != nil {
			taskVars = config.TaskPlaceholders(task)
		} else {
			// Fallback: parse parent keys from task key format
			epicKey := config.ParseEpicKeyFromEntityKey(taskKey)
			featureKey := config.ParseFeatureKeyFromTaskKey(taskKey)
			if epicKey == "" {
				epicKey = taskKey
			}
			if featureKey == "" {
				featureKey = taskKey
			}
			taskVars = map[string]string{
				"id":          taskKey,
				"key":         taskKey,
				"task_key":    taskKey,
				"epic_key":    epicKey,
				"feature_key": featureKey,
			}
		}
	}

	result, err := svc.GetStatusAction(configPath, status, taskVars)
	if err != nil {
		return err
	}

	if cli.GlobalConfig.JSON {
		var actionField interface{}
		if result.Action != "" || len(result.Skills) > 0 || result.AgentType != "" || result.Instruction != "" {
			actionField = map[string]interface{}{
				"action":      result.Action,
				"agent_type":  result.AgentType,
				"skills":      result.Skills,
				"instruction": result.Instruction,
			}
		}
		response := map[string]interface{}{
			"status": status,
			"action": actionField,
		}
		return cli.OutputJSON(response)
	}

	// Human-readable output
	if result.Action == "" && result.Instruction == "" {
		fmt.Printf("No orchestrator action defined for status '%s'\n", status)
		return nil
	}

	fmt.Printf("Status: %s\n", status)
	fmt.Printf("Action: %s\n", result.Action)
	if result.AgentType != "" {
		fmt.Printf("Agent Type: %s\n", result.AgentType)
	}
	if len(result.Skills) > 0 {
		fmt.Printf("Skills: %s\n", strings.Join(result.Skills, ", "))
	}
	fmt.Printf("Instruction:\n  %s\n", result.Instruction)
	if taskKeyFlag == "" {
		fmt.Println("\nNote: Template variables (e.g., {task_id}) not populated. Use --task flag to populate.")
	}
	return nil
}

func init() {
	adminCmd.AddCommand(configCmd)

	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configValidatePatternsCmd)
	configCmd.AddCommand(configTestPatternCmd)
	configCmd.AddCommand(configGetFormatCmd)
	configCmd.AddCommand(configListPresetsCmd)
	configCmd.AddCommand(configShowPresetCmd)
	configCmd.AddCommand(configAddPatternCmd)
	configCmd.AddCommand(configGetStatusActionCmd)

	configShowCmd.Flags().Bool("patterns", false, "Show only pattern configuration")

	configTestPatternCmd.Flags().String("pattern", "", "Regex pattern to test")
	configTestPatternCmd.Flags().String("test-string", "", "String to test pattern against")
	configTestPatternCmd.Flags().String("type", "epic", "Entity type (epic, feature, task)")

	configGetFormatCmd.Flags().String("type", "", "Entity type (epic, feature, task)")
	configGetFormatCmd.Flags().Bool("json", false, "Output in JSON format")

	configAddPatternCmd.Flags().String("preset", "", "Name of the preset to add (required)")

	configGetStatusActionCmd.Flags().String("task", "", "Task key to populate template variables (optional)")
}
