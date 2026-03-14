package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds the global CLI configuration
type Config struct {
	JSON       bool
	NoColor    bool
	Verbose    bool
	ConfigFile string
	DBPath     string
	Field      string
}

// GlobalConfig is the shared configuration instance
var GlobalConfig = &Config{}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "shark",
	Short: "Shark Task Manager - Task management CLI for AI-driven development",
	Long: `Shark is a command-line task management tool for AI Agents.

Examples:
  shark list E07                      List features in an epic
  shark get E07-F01-001               View task details
  shark create feature E07 "new feature" --description="important feature"
  shark update E07-F02 --filename="docs/plan/E07-important-epic/F02-new-feature/specs.md"
  shark status advance T-E07-F01-001  Advance to the next status in the workflow`,
	Version: "dev", // Will be set by SetVersion() from build-time injection
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize configuration
		if err := initConfig(); err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
		}

		// Configure template directory from .sharkconfig.json
		cfgPath, cfgErr := GetConfigPath()
		if cfgErr == nil {
			templates.SetConfiguredTemplateDir(config.GetTemplateDirectoryFromConfig(cfgPath))
		}

		// Disable color output if requested
		if GlobalConfig.NoColor {
			pterm.DisableColor()
		}

		// Set verbose logging if requested
		if GlobalConfig.Verbose {
			pterm.EnableDebugMessages()
		}

		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		// Close database connection if it was opened
		if err := CloseDB(); err != nil {
			// Log warning but don't fail - cleanup errors shouldn't break exit
			if GlobalConfig.Verbose {
				pterm.Warning.Printf("Failed to close database: %v\n", err)
			}
		}
		return nil
	},
}

// SetVersion sets the version string from build-time injection
func SetVersion(version string) {
	RootCmd.Version = version
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return RootCmd.Execute()
}

func init() {
	// Disable alphabetical sorting so commands appear in registration order
	cobra.EnableCommandSorting = false

	// Define command groups for better organization in help output
	RootCmd.AddGroup(
		&cobra.Group{
			ID:    "inspect",
			Title: "Inspect:",
		},
		&cobra.Group{
			ID:    "manage",
			Title: "Manage:",
		},
		&cobra.Group{
			ID:    "advanced",
			Title: "Advanced:",
		},
	)

	// Set the completion command to the advanced group
	RootCmd.SetCompletionCommandGroupID("advanced")

	// Put the help command in the advanced group and hide it from top-level help.
	// Users can still use --help on any command.
	RootCmd.SetHelpCommandGroupID("advanced")
	RootCmd.InitDefaultHelpCmd()
	for _, cmd := range RootCmd.Commands() {
		if cmd.Name() == "help" {
			cmd.Hidden = true
			break
		}
	}

	// Customize usage template to show [--help] and add note
	RootCmd.SetUsageTemplate(rootUsageTemplate)

	// Silence Cobra's default error and usage printing so we handle errors ourselves
	RootCmd.SilenceErrors = true
	RootCmd.SilenceUsage = true

	// Customize the help flag description
	RootCmd.Flags().BoolP("help", "h", false, "help for shark or shark commands")

	// Global flags available to all commands
	RootCmd.PersistentFlags().BoolVar(&GlobalConfig.JSON, "json", false, "Output in JSON format (machine-readable)")
	RootCmd.PersistentFlags().BoolVar(&GlobalConfig.NoColor, "no-color", false, "Disable colored output")
	RootCmd.PersistentFlags().BoolVarP(&GlobalConfig.Verbose, "verbose", "v", false, "Enable verbose/debug output")
	RootCmd.PersistentFlags().StringVar(&GlobalConfig.ConfigFile, "config", "", "Config file path (default: .sharkconfig.json)")
	RootCmd.PersistentFlags().StringVar(&GlobalConfig.DBPath, "db", "shark-tasks.db", "Database file path")
	RootCmd.PersistentFlags().StringVar(&GlobalConfig.Field, "field", "", "Extract a single field from JSON output (e.g., --field status)")

	// Bind flags to viper for config file support
	if err := viper.BindPFlag("json", RootCmd.PersistentFlags().Lookup("json")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("no-color", RootCmd.PersistentFlags().Lookup("no-color")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("verbose", RootCmd.PersistentFlags().Lookup("verbose")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("db", RootCmd.PersistentFlags().Lookup("db")); err != nil {
		panic(err)
	}
}

// rootUsageTemplate customizes the usage output for the root command.
// It adds [--help] to the usage line and a note about subcommands.
const rootUsageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command] [--help]
  *Note: most commands have subcommands and flags. Use --help to view syntax.{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) .IsAvailableCommand)}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") .IsAvailableCommand)}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

// FindProjectRoot walks up the directory tree to find the project root.
// It looks for markers with different priorities:
// 1. .sharkconfig.json (STRONGEST - always preferred)
// 2. shark-tasks.db (STRONG - used if no .sharkconfig.json found)
// 3. .git/ directory (WEAK - used if no stronger markers found)
//
// The search goes all the way to the filesystem root and returns the BEST marker found,
// preferring .sharkconfig.json over everything else. This ensures that if .sharkconfig.json
// exists in the project root but shark-tasks.db exists in a subdirectory (like /docs),
// we correctly identify the project root.
//
// Returns the project root directory, or current directory if no markers found.
func FindProjectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Track the best marker found during search
	var foundConfig string // .sharkconfig.json (highest priority)
	var foundDB string     // shark-tasks.db (medium priority)
	var foundGit string    // .git directory (lowest priority)

	currentDir := wd

	// Search from current directory up to filesystem root
	for {
		// Check for .sharkconfig.json (highest priority)
		// Take the first (closest) one found
		if foundConfig == "" {
			if _, err := os.Stat(filepath.Join(currentDir, ".sharkconfig.json")); err == nil {
				foundConfig = currentDir
			}
		}

		// Check for shark-tasks.db (medium priority)
		if foundDB == "" {
			if _, err := os.Stat(filepath.Join(currentDir, "shark-tasks.db")); err == nil {
				foundDB = currentDir
			}
		}

		// Check for .git directory (lowest priority)
		if foundGit == "" {
			if _, err := os.Stat(filepath.Join(currentDir, ".git")); err == nil {
				foundGit = currentDir
			}
		}

		// Move up one directory
		parentDir := filepath.Dir(currentDir)

		// If we've reached the filesystem root, stop searching
		if parentDir == currentDir {
			break
		}

		currentDir = parentDir
	}

	// Return the best marker found, in priority order
	if foundConfig != "" {
		return foundConfig, nil
	}
	if foundDB != "" {
		return foundDB, nil
	}
	if foundGit != "" {
		return foundGit, nil
	}

	// No markers found, use working directory
	return wd, nil
}

// GetConfigPath returns the absolute path to .sharkconfig.json.
// It respects the --config flag if set, otherwise finds the project root
// and returns the config file path in that directory.
//
// This function ensures that config file discovery works consistently
// from any subdirectory within the project.
//
// Returns:
//   - Absolute path to .sharkconfig.json
//   - Error if project root cannot be determined
func GetConfigPath() (string, error) {
	// If explicit config path was provided via --config flag, use it
	if GlobalConfig.ConfigFile != "" {
		return GlobalConfig.ConfigFile, nil
	}

	// Find project root
	projectRoot, err := FindProjectRoot()
	if err != nil {
		return "", fmt.Errorf("failed to find project root: %w", err)
	}

	// Return config path in project root
	return filepath.Join(projectRoot, ".sharkconfig.json"), nil
}

// initConfig reads in config file and ENV variables if set
func initConfig() error {
	// Find project root (unless explicit config path was given)
	if GlobalConfig.ConfigFile == "" {
		projectRoot, err := FindProjectRoot()
		if err != nil {
			return fmt.Errorf("failed to find project root: %w", err)
		}

		if GlobalConfig.Verbose {
			pterm.Debug.Printf("Project root: %s\n", projectRoot)
		}

		// Look for config in project root
		viper.AddConfigPath(projectRoot)
		viper.SetConfigType("json")
		viper.SetConfigName(".sharkconfig")

		// If DBPath is still the default relative path, make it relative to project root
		if GlobalConfig.DBPath == "shark-tasks.db" {
			GlobalConfig.DBPath = filepath.Join(projectRoot, "shark-tasks.db")
		}
	} else {
		// Use config file from the flag
		viper.SetConfigFile(GlobalConfig.ConfigFile)
	}

	// Read environment variables with PM_ prefix
	viper.SetEnvPrefix("PM")
	viper.AutomaticEnv()

	// Try to read config file (don't error if it doesn't exist)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error occurred
			return fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; using defaults and flags
	} else if GlobalConfig.Verbose {
		pterm.Debug.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	}

	// Update GlobalConfig from viper
	GlobalConfig.JSON = viper.GetBool("json")
	GlobalConfig.NoColor = viper.GetBool("no-color")
	GlobalConfig.Verbose = viper.GetBool("verbose")

	// Check SHARK_OUTPUT environment variable for session-wide JSON mode
	applySharkOutputEnv()

	// --field implies JSON mode
	if GlobalConfig.Field != "" {
		GlobalConfig.JSON = true
	}

	// Only override DBPath from viper if it was explicitly set (not default)
	if viper.IsSet("db") {
		GlobalConfig.DBPath = viper.GetString("db")
	}

	return nil
}

// applySharkOutputEnv checks the SHARK_OUTPUT environment variable.
// If SHARK_OUTPUT=json (case-sensitive, lowercase only), it enables JSON output
// unless the --json flag was already set.
func applySharkOutputEnv() {
	if !GlobalConfig.JSON {
		if os.Getenv("SHARK_OUTPUT") == "json" {
			GlobalConfig.JSON = true
		}
	}
}

// OutputJSON outputs data in JSON format. If --field is set, it extracts
// and prints only the specified field value. CLIError values bypass field
// extraction and are always output as full JSON.
func OutputJSON(data interface{}) error {
	// CLIError bypass: always output full struct
	if _, ok := data.(*CLIError); ok {
		return outputJSONRaw(data)
	}
	if _, ok := data.(CLIError); ok {
		return outputJSONRaw(data)
	}

	// If --field is set, extract and print the field
	if GlobalConfig.Field != "" {
		return OutputField(data, GlobalConfig.Field)
	}

	return outputJSONRaw(data)
}

// outputJSONRaw outputs data as indented JSON to stdout.
func outputJSONRaw(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// OutputTable outputs data as a formatted table (for humans)
// This will be expanded in Phase 2 with Rich table formatting
func OutputTable(headers []string, rows [][]string) {
	if len(rows) == 0 {
		pterm.Info.Println("No results found")
		return
	}

	// Convert to pterm table format
	tableData := pterm.TableData{headers}
	for _, row := range rows {
		tableData = append(tableData, row)
	}

	if err := pterm.DefaultTable.WithHasHeader().WithData(tableData).Render(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to render table: %v\n", err)
	}
}

// Success prints a success message
func Success(message string) {
	if !GlobalConfig.NoColor {
		pterm.Success.Println(message)
	} else {
		fmt.Println("✓", message)
	}
}

// Error prints an error message. In JSON mode, it outputs a structured
// CLIError JSON object to stdout. In human mode, it prints to stderr.
func Error(message string) {
	if GlobalConfig.JSON {
		ErrorJSON(CLIError{
			Code:    ErrCodeCommandError,
			Message: message,
		})
		return
	}
	if !GlobalConfig.NoColor {
		pterm.Error.Println(message)
	} else {
		fmt.Fprintln(os.Stderr, "✗", message)
	}
}

// Warning prints a warning message
func Warning(message string) {
	if !GlobalConfig.NoColor {
		pterm.Warning.Println(message)
	} else {
		fmt.Println("⚠", message)
	}
}

// Info prints an info message
func Info(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	if !GlobalConfig.NoColor {
		pterm.Info.Println(message)
	} else {
		fmt.Println("ℹ", message)
	}
}

// Title prints a section title
func Title(message string) {
	if !GlobalConfig.NoColor {
		pterm.DefaultHeader.WithFullWidth().Println(message)
	} else {
		fmt.Println("===", message, "===")
	}
}

// Color constants for manual formatting
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
)

// GetDBPath returns the database file path, ensuring parent directory exists
// The path is already resolved to the project root by initConfig()
//
// Deprecated for general use: Commands should use GetDB() instead for database access.
// This function is maintained only for backup utilities that need the physical file path
// to copy the database file. New code should use GetDB() for all database operations.
func GetDBPath() (string, error) {
	dbPath := GlobalConfig.DBPath

	// If still relative (user explicitly set a relative path), make it absolute from cwd
	if !filepath.IsAbs(dbPath) {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
		dbPath = filepath.Join(wd, dbPath)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create database directory: %w", err)
	}

	return dbPath, nil
}
