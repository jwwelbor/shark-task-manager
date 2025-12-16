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
	JSON     bool
	NoColor  bool
	Verbose  bool
	ConfigFile string
	DBPath   string
}

// GlobalConfig is the shared configuration instance
var GlobalConfig = &Config{}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "pm",
	Short: "Project Manager - Task management CLI for AI-driven development",
	Long: `PM (Project Manager) is a command-line tool for managing tasks, epics, and features
in multi-agent software development projects.

It provides a SQLite-backed database for tracking project state with commands
optimized for both human developers and AI agents.`,
	Version: "0.1.0",
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

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return RootCmd.Execute()
}

func init() {
	// Global flags available to all commands
	RootCmd.PersistentFlags().BoolVar(&GlobalConfig.JSON, "json", false, "Output in JSON format (machine-readable)")
	RootCmd.PersistentFlags().BoolVar(&GlobalConfig.NoColor, "no-color", false, "Disable colored output")
	RootCmd.PersistentFlags().BoolVarP(&GlobalConfig.Verbose, "verbose", "v", false, "Enable verbose/debug output")
	RootCmd.PersistentFlags().StringVar(&GlobalConfig.ConfigFile, "config", "", "Config file path (default: .pmconfig.json)")
	RootCmd.PersistentFlags().StringVar(&GlobalConfig.DBPath, "db", "shark-tasks.db", "Database file path")

	// Bind flags to viper for config file support
	viper.BindPFlag("json", RootCmd.PersistentFlags().Lookup("json"))
	viper.BindPFlag("no-color", RootCmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("verbose", RootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("db", RootCmd.PersistentFlags().Lookup("db"))
}

// initConfig reads in config file and ENV variables if set
func initConfig() error {
	if GlobalConfig.ConfigFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(GlobalConfig.ConfigFile)
	} else {
		// Search for config in current directory
		viper.AddConfigPath(".")
		viper.SetConfigType("json")
		viper.SetConfigName(".pmconfig")
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
	GlobalConfig.DBPath = viper.GetString("db")

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

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
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
func Info(message string) {
	if !GlobalConfig.NoColor {
		pterm.Info.Println(message)
	} else {
		fmt.Println("ℹ", message)
	}
}

// GetDBPath returns the database file path, ensuring parent directory exists
func GetDBPath() (string, error) {
	dbPath := GlobalConfig.DBPath

	// If relative path, make it absolute
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
