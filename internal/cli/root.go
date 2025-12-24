package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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
}

// GlobalConfig is the shared configuration instance
var GlobalConfig = &Config{}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "shark",
	Short: "Shark Task Manager - Task management CLI for AI-driven development",
	Long: `Shark is a command-line tool for managing tasks, epics, and features
in multi-agent software development projects.

It provides a SQLite-backed database for tracking project state with commands
optimized for both human developers and AI agents.`,
	Version: "dev", // Will be set by SetVersion() from build-time injection
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize configuration
		if err := initConfig(); err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
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
	// Global flags available to all commands
	RootCmd.PersistentFlags().BoolVar(&GlobalConfig.JSON, "json", false, "Output in JSON format (machine-readable)")
	RootCmd.PersistentFlags().BoolVar(&GlobalConfig.NoColor, "no-color", false, "Disable colored output")
	RootCmd.PersistentFlags().BoolVarP(&GlobalConfig.Verbose, "verbose", "v", false, "Enable verbose/debug output")
	RootCmd.PersistentFlags().StringVar(&GlobalConfig.ConfigFile, "config", "", "Config file path (default: .sharkconfig.json)")
	RootCmd.PersistentFlags().StringVar(&GlobalConfig.DBPath, "db", "shark-tasks.db", "Database file path")

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

// FindProjectRoot walks up the directory tree to find the project root.
// It looks for markers in this order:
// 1. .sharkconfig.json (primary marker)
// 2. shark-tasks.db (secondary marker)
// 3. .git/ directory (fallback for git projects)
// Returns the project root directory, or current directory if no markers found.
func FindProjectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	currentDir := wd
	for {
		// Check for .sharkconfig.json (strongest signal)
		if _, err := os.Stat(filepath.Join(currentDir, ".sharkconfig.json")); err == nil {
			return currentDir, nil
		}

		// Check for shark-tasks.db
		if _, err := os.Stat(filepath.Join(currentDir, "shark-tasks.db")); err == nil {
			return currentDir, nil
		}

		// Check for .git directory (fallback)
		if _, err := os.Stat(filepath.Join(currentDir, ".git")); err == nil {
			return currentDir, nil
		}

		// Move up one directory
		parentDir := filepath.Dir(currentDir)

		// If we've reached the root (parent == current), stop
		if parentDir == currentDir {
			// No project root found, use original working directory
			return wd, nil
		}

		currentDir = parentDir
	}
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

	// Only override DBPath from viper if it was explicitly set (not default)
	if viper.IsSet("db") {
		GlobalConfig.DBPath = viper.GetString("db")
	}

	return nil
}

// OutputJSON outputs data in JSON format
func OutputJSON(data interface{}) error {
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

// Error prints an error message
func Error(message string) {
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
