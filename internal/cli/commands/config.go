package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
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

		if patternsOnly {
			return showPatternsConfig()
		}

		// Show all configuration
		settings := map[string]interface{}{
			"json":     viper.GetBool("json"),
			"no-color": viper.GetBool("no-color"),
			"verbose":  viper.GetBool("verbose"),
			"db":       viper.GetString("db"),
		}

		// Try to load patterns if available
		if configFile != "" {
			patternsConfig, err := loadPatternsFromConfig(configFile)
			if err == nil {
				settings["patterns"] = patternsConfig
			}
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(settings)
		}

		fmt.Println("\nCurrent Settings:")
		for key, value := range settings {
			if key == "patterns" {
				fmt.Printf("  %s: (use --patterns flag to view)\n", key)
			} else {
				fmt.Printf("  %s: %v\n", key, value)
			}
		}

		return nil
	},
}

// configValidateCmd validates configuration
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	Long:  `Check configuration file for errors and validate settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile := viper.ConfigFileUsed()
		if configFile == "" {
			cli.Warning("No configuration file to validate")
			return nil
		}

		// Configuration is already validated during loading
		// If we got here, it's valid
		cli.Success(fmt.Sprintf("Configuration file is valid: %s", configFile))
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
		// Load pattern configuration
		configFile := ".sharkconfig.json"
		patternsConfig, err := loadPatternsFromConfig(configFile)
		if err != nil {
			cli.Error(fmt.Sprintf("Failed to load patterns: %v", err))
			return err
		}

		// Validate all patterns
		validationErr := patterns.ValidatePatternConfig(patternsConfig)

		// Count results
		epicValid, epicErrors := countPatternResults(patternsConfig.Epic, "epic")
		featureValid, featureErrors := countPatternResults(patternsConfig.Feature, "feature")
		taskValid, taskErrors := countPatternResults(patternsConfig.Task, "task")

		// Get warnings
		epicWarnings := getPatternWarnings(patternsConfig.Epic, "epic")
		featureWarnings := getPatternWarnings(patternsConfig.Feature, "feature")
		taskWarnings := getPatternWarnings(patternsConfig.Task, "task")

		// Display results
		fmt.Println("\nPattern Validation Report:")
		fmt.Printf("  Epic patterns: %d valid, %d errors, %d warnings\n", epicValid, len(epicErrors), len(epicWarnings))
		fmt.Printf("  Feature patterns: %d valid, %d errors, %d warnings\n", featureValid, len(featureErrors), len(featureWarnings))
		fmt.Printf("  Task patterns: %d valid, %d errors, %d warnings\n", taskValid, len(taskErrors), len(taskWarnings))

		// Display errors
		hasErrors := false
		if len(epicErrors) > 0 || len(featureErrors) > 0 || len(taskErrors) > 0 {
			hasErrors = true
			fmt.Println("\nErrors:")
			for _, err := range epicErrors {
				cli.Error(fmt.Sprintf("  [ERROR] Epic: %s", err))
			}
			for _, err := range featureErrors {
				cli.Error(fmt.Sprintf("  [ERROR] Feature: %s", err))
			}
			for _, err := range taskErrors {
				cli.Error(fmt.Sprintf("  [ERROR] Task: %s", err))
			}
		}

		// Display warnings
		if len(epicWarnings) > 0 || len(featureWarnings) > 0 || len(taskWarnings) > 0 {
			fmt.Println("\nWarnings:")
			for _, warn := range epicWarnings {
				cli.Warning(fmt.Sprintf("  [WARN] Epic: %s", warn))
			}
			for _, warn := range featureWarnings {
				cli.Warning(fmt.Sprintf("  [WARN] Feature: %s", warn))
			}
			for _, warn := range taskWarnings {
				cli.Warning(fmt.Sprintf("  [WARN] Task: %s", warn))
			}
		}

		if hasErrors || validationErr != nil {
			fmt.Println("")
			os.Exit(1)
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
			entityType = "epic" // default
		}

		start := time.Now()

		// Test pattern match
		matched, groups, err := testPatternMatch(pattern, testString)

		duration := time.Since(start)

		fmt.Println("\nPattern Test Result:")
		fmt.Printf("  Pattern: %s\n", pattern)
		fmt.Printf("  Test String: %s\n", testString)
		fmt.Printf("  Match: %v\n", matched)

		if err != nil {
			cli.Error(fmt.Sprintf("  Pattern Error: %v\n", err))
			return err
		}

		if matched {
			fmt.Println("  Captured Groups:")
			for name, value := range groups {
				fmt.Printf("    - %s: %s\n", name, value)
			}
		} else {
			fmt.Println("  No match found")

			// Suggest similar patterns from config
			if configFile := ".sharkconfig.json"; fileExists(configFile) {
				patternsConfig, err := loadPatternsFromConfig(configFile)
				if err == nil {
					suggestions := findMatchingPatterns(patternsConfig, testString, entityType)
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

		// Load pattern configuration
		configFile := ".sharkconfig.json"
		patternsConfig, err := loadPatternsFromConfig(configFile)
		if err != nil {
			// Use defaults if no config file
			patternsConfig = patterns.GetDefaultPatterns()
		}

		var format string
		switch entityType {
		case "epic":
			format = patternsConfig.Epic.Generation.Format
		case "feature":
			format = patternsConfig.Feature.Generation.Format
		case "task":
			format = patternsConfig.Task.Generation.Format
		default:
			return fmt.Errorf("invalid type: %s (must be epic, feature, or task)", entityType)
		}

		// Generate example
		example := generateFormatExample(entityType, format)
		placeholders := getPlaceholdersForType(entityType)

		if jsonOutput {
			output := map[string]interface{}{
				"format":       format,
				"example":      example,
				"placeholders": placeholders,
			}
			return cli.OutputJSON(output)
		}

		fmt.Printf("\nGeneration Format for %s:\n", entityType)
		fmt.Printf("  Format: %s\n", format)
		fmt.Printf("  Example: %s\n", example)
		fmt.Printf("  Available placeholders: %s\n", strings.Join(placeholders, ", "))

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
			// Show available presets on error
			presets := patterns.ListPresets()
			cli.Error(fmt.Sprintf("Unknown preset: %s", presetName))
			fmt.Println("\nAvailable presets:")
			for _, p := range presets {
				fmt.Printf("  - %s\n", p.Name)
			}
			return err
		}

		// Get preset info for description
		info, _ := patterns.GetPresetInfo(presetName)

		if cli.GlobalConfig.JSON {
			output := map[string]interface{}{
				"name":        info.Name,
				"description": info.Description,
				"patterns":    preset,
			}
			return cli.OutputJSON(output)
		}

		// Display preset info
		fmt.Printf("\nPreset: %s\n", info.Name)
		fmt.Printf("Description: %s\n\n", info.Description)

		// Display patterns as formatted JSON
		data, err := json.MarshalIndent(preset, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format preset: %w", err)
		}

		fmt.Println("Patterns:")
		fmt.Println(string(data))
		fmt.Println()

		// Show which entity types are affected
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

		// Get the preset
		preset, err := patterns.GetPreset(presetName)
		if err != nil {
			// Show available presets on error
			presets := patterns.ListPresets()
			cli.Error(fmt.Sprintf("Unknown preset: %s", presetName))
			fmt.Println("\nAvailable presets:")
			for _, p := range presets {
				fmt.Printf("  - %s\n", p.Name)
			}
			return err
		}

		// Load current config
		configPath := viper.GetString("config")
		if configPath == "" {
			configPath = ".sharkconfig.json"
		}

		// Read existing config
		var currentConfig *patterns.PatternConfig
		if _, err := os.Stat(configPath); err == nil {
			// Config exists, read it
			data, err := os.ReadFile(configPath)
			if err != nil {
				return fmt.Errorf("failed to read config file: %w", err)
			}

			// Parse config to get patterns
			var configData struct {
				Patterns json.RawMessage `json:"patterns"`
			}
			if err := json.Unmarshal(data, &configData); err != nil {
				return fmt.Errorf("failed to parse config file: %w", err)
			}

			if len(configData.Patterns) > 0 {
				currentConfig = &patterns.PatternConfig{}
				if err := json.Unmarshal(configData.Patterns, currentConfig); err != nil {
					return fmt.Errorf("failed to parse patterns: %w", err)
				}
			}
		}

		// If no config exists, use defaults
		if currentConfig == nil {
			currentConfig = patterns.GetDefaultPatterns()
		}

		// Merge patterns
		mergedConfig, stats := patterns.MergePatternsWithStats(currentConfig, preset)

		// Validate merged patterns
		if err := patterns.ValidatePatternConfig(mergedConfig); err != nil {
			cli.Error("Merged patterns failed validation")
			return fmt.Errorf("pattern validation failed: %w", err)
		}

		// Update config file
		// Read full config
		var fullConfig map[string]interface{}
		if data, err := os.ReadFile(configPath); err == nil {
			json.Unmarshal(data, &fullConfig)
		} else {
			fullConfig = make(map[string]interface{})
		}

		// Update patterns section
		fullConfig["patterns"] = mergedConfig

		// Write back to file
		data, err := json.MarshalIndent(fullConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		// Create backup
		if _, err := os.Stat(configPath); err == nil {
			backupPath := configPath + ".bak"
			if err := os.Rename(configPath, backupPath); err != nil {
				cli.Warning(fmt.Sprintf("Failed to create backup: %v", err))
			} else {
				defer func() {
					// Remove backup on success
					os.Remove(backupPath)
				}()
			}
		}

		// Write new config
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}

		// Display results
		if cli.GlobalConfig.JSON {
			output := map[string]interface{}{
				"preset":  presetName,
				"added":   stats.Added,
				"skipped": stats.Skipped,
				"details": stats.Details,
			}
			return cli.OutputJSON(output)
		}

		cli.Success(fmt.Sprintf("Added preset '%s' to configuration", presetName))
		fmt.Printf("\nResults:\n")
		for _, detail := range stats.Details {
			fmt.Printf("  - %s\n", detail)
		}
		fmt.Println()

		return nil
	},
}

func init() {
	// Register config command with root
	cli.RootCmd.AddCommand(configCmd)

	// Add subcommands
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configValidatePatternsCmd)
	configCmd.AddCommand(configTestPatternCmd)
	configCmd.AddCommand(configGetFormatCmd)
	configCmd.AddCommand(configListPresetsCmd)
	configCmd.AddCommand(configShowPresetCmd)
	configCmd.AddCommand(configAddPatternCmd)

	// Flags for show command
	configShowCmd.Flags().Bool("patterns", false, "Show only pattern configuration")

	// Flags for test-pattern command
	configTestPatternCmd.Flags().String("pattern", "", "Regex pattern to test")
	configTestPatternCmd.Flags().String("test-string", "", "String to test pattern against")
	configTestPatternCmd.Flags().String("type", "epic", "Entity type (epic, feature, task)")

	// Flags for get-format command
	configGetFormatCmd.Flags().String("type", "", "Entity type (epic, feature, task)")
	configGetFormatCmd.Flags().Bool("json", false, "Output in JSON format")

	// Flags for add-pattern command
	configAddPatternCmd.Flags().String("preset", "", "Name of the preset to add (required)")
}

// Helper functions

// showPatternsConfig displays only the patterns configuration
func showPatternsConfig() error {
	configFile := ".sharkconfig.json"
	patternsConfig, err := loadPatternsFromConfig(configFile)
	if err != nil {
		cli.Warning("Using default patterns (no config file found)")
		patternsConfig = patterns.GetDefaultPatterns()
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

// loadPatternsFromConfig loads pattern configuration from a file
func loadPatternsFromConfig(configPath string) (*patterns.PatternConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config struct {
		Patterns *patterns.PatternConfig `json:"patterns"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if config.Patterns == nil {
		return patterns.GetDefaultPatterns(), nil
	}

	return config.Patterns, nil
}

// testPatternMatch tests if a pattern matches a string and returns captured groups
func testPatternMatch(pattern, testString string) (bool, map[string]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, nil, fmt.Errorf("invalid regex syntax: %w", err)
	}

	if !re.MatchString(testString) {
		return false, nil, nil
	}

	// Extract captured groups
	match := re.FindStringSubmatch(testString)
	names := re.SubexpNames()

	groups := make(map[string]string)
	for i, name := range names {
		if i > 0 && name != "" && i < len(match) {
			groups[name] = match[i]
		}
	}

	return true, groups, nil
}

// countPatternResults counts valid patterns and errors for an entity
func countPatternResults(entity patterns.EntityPatterns, entityType string) (int, []string) {
	valid := 0
	var errors []string

	// Count folder patterns
	for i, pattern := range entity.Folder {
		if err := patterns.ValidatePattern(pattern, entityType); err != nil {
			errors = append(errors, fmt.Sprintf("folder pattern #%d: %v", i+1, err))
		} else {
			valid++
		}
	}

	// Count file patterns (syntax only, no capture group requirements)
	for i, pattern := range entity.File {
		if err := patterns.ValidatePatternSyntaxOnly(pattern); err != nil {
			errors = append(errors, fmt.Sprintf("file pattern #%d: %v", i+1, err))
		} else {
			valid++
		}
	}

	return valid, errors
}

// getPatternWarnings gets all warnings for an entity's patterns
func getPatternWarnings(entity patterns.EntityPatterns, entityType string) []string {
	var warnings []string

	// Check folder patterns
	for i, pattern := range entity.Folder {
		warns := patterns.GetPatternWarnings(pattern, entityType)
		for _, w := range warns {
			warnings = append(warnings, fmt.Sprintf("folder pattern #%d: %s", i+1, w))
		}
	}

	return warnings
}

// findMatchingPatterns finds patterns from config that match the test string
func findMatchingPatterns(config *patterns.PatternConfig, testString, entityType string) []string {
	var matching []string

	var patternsToCheck []string
	switch entityType {
	case "epic":
		patternsToCheck = config.Epic.Folder
	case "feature":
		patternsToCheck = config.Feature.Folder
	case "task":
		patternsToCheck = append(config.Task.Folder, config.Task.File...)
	}

	for _, pattern := range patternsToCheck {
		if matched, _, _ := testPatternMatch(pattern, testString); matched {
			matching = append(matching, pattern)
		}
	}

	return matching
}

// generateFormatExample generates an example using a format template
func generateFormatExample(entityType, format string) string {
	values := map[string]interface{}{
		"number":  4,
		"epic":    4,
		"feature": 7,
		"slug":    "example-" + entityType,
	}

	result, err := patterns.ApplyGenerationFormat(format, values)
	if err != nil {
		return ""
	}
	return result
}

// getPlaceholdersForType returns available placeholders for an entity type
func getPlaceholdersForType(entityType string) []string {
	switch entityType {
	case "epic":
		return []string{"number", "slug"}
	case "feature":
		return []string{"epic", "number", "slug"}
	case "task":
		return []string{"epic", "feature", "number", "slug"}
	default:
		return []string{}
	}
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
