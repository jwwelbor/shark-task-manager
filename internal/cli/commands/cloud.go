package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/spf13/cobra"
)

// Cloud command flags
var (
	cloudURL            string
	cloudAuthToken      string
	cloudAuthFile       string
	cloudNonInteractive bool
)

// cloudCmd represents the cloud command group
var cloudCmd = &cobra.Command{
	Use:     "cloud",
	Short:   "Manage cloud database configuration",
	GroupID: "setup",
	Long: `Commands for configuring and managing Turso cloud database integration.

The cloud command group provides:
  - init: Configure Turso as your database backend
  - login: Authenticate with Turso
  - logout: Clear Turso credentials
  - status: Check cloud connection status
  - sync: Manually trigger cloud synchronization`,
}

// cloudInitCmd initializes cloud database configuration
var cloudInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Turso cloud database configuration",
	Long: `Initialize Turso cloud database by configuring connection URL and authentication.

This command updates .sharkconfig.json to use Turso as the database backend
instead of local SQLite. You can provide the database URL and authentication
token via flags or interactive prompts.`,
	Example: `  # Interactive setup with prompts
  shark cloud init

  # Non-interactive with flags
  shark cloud init --url="libsql://mydb.turso.io" --auth-token="eyJ..."

  # Use token from file
  shark cloud init --url="libsql://mydb.turso.io" --auth-file="~/.turso/token"

  # Use environment variable TURSO_AUTH_TOKEN
  shark cloud init --url="libsql://mydb.turso.io"`,
	RunE: runCloudInit,
}

// cloudStatusCmd shows current cloud database configuration status
var cloudStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show cloud database configuration status",
	Long: `Display the current cloud database configuration including backend type,
connection URL, and authentication status.

This command reads .sharkconfig.json and shows whether cloud database (Turso)
is configured and active.`,
	Example: `  # Show current cloud status
  shark cloud status

  # Show status in JSON format
  shark cloud status --json`,
	RunE: runCloudStatus,
}

func init() {
	// Register cloud command group
	cli.RootCmd.AddCommand(cloudCmd)

	// Register subcommands
	cloudCmd.AddCommand(cloudInitCmd)
	cloudCmd.AddCommand(cloudStatusCmd)

	// Cloud init flags
	cloudInitCmd.Flags().StringVar(&cloudURL, "url", "", "Turso database URL (libsql://...)")
	cloudInitCmd.Flags().StringVar(&cloudAuthToken, "auth-token", "", "Turso auth token (JWT)")
	cloudInitCmd.Flags().StringVar(&cloudAuthFile, "auth-file", "", "Path to file containing auth token")
	cloudInitCmd.Flags().BoolVar(&cloudNonInteractive, "non-interactive", false, "Skip prompts, fail if required info missing")
}

func runCloudStatus(cmd *cobra.Command, args []string) error {
	// Get project root and config path
	projectRoot, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}
	configPath := filepath.Join(projectRoot, ".sharkconfig.json")

	// Get cloud status
	status, err := getCloudStatus(configPath)
	if err != nil {
		return fmt.Errorf("failed to read cloud status: %w", err)
	}

	// Test actual database connection
	status.ConnectionTested = true
	repoDb, connErr := cli.GetDB(cmd.Context())
	if connErr != nil {
		status.ConnectionStatus = "failed"
		status.ConnectionError = connErr.Error()
	} else {
		status.ConnectionStatus = "success"
		// Quick test query to verify connection really works
		if err := repoDb.Ping(); err != nil {
			status.ConnectionStatus = "failed"
			status.ConnectionError = fmt.Sprintf("connection established but ping failed: %v", err)
		}
	}

	// Output in JSON if requested
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(status)
	}

	// Pretty output
	cli.Info("Cloud Database Status")
	cli.Info("━━━━━━━━━━━━━━━━━━━━")
	cli.Info("")

	if status.IsCloudConfigured {
		cli.Success("Cloud database is CONFIGURED")
		cli.Info("Backend: %s", status.Backend)
		cli.Info("URL: %s", status.URL)
		if status.AuthTokenFile != "" {
			cli.Info("Auth token file: %s", status.AuthTokenFile)
		}
	} else {
		if status.Backend != "" {
			cli.Warning("Using local database")
			cli.Info("Backend: %s", status.Backend)
			cli.Info("URL: %s", status.URL)
		} else {
			cli.Warning("No database configuration found")
		}
		cli.Info("")
		cli.Info("To configure cloud database, run:")
		cli.Info("  shark cloud init --url=<turso-url> --auth-token=<token>")
	}

	// Show connection test results
	cli.Info("")
	cli.Info("Connection Test:")
	if status.ConnectionStatus == "success" {
		cli.Success("✓ Database connection successful")
	} else {
		cli.Error("✗ Database connection failed")
		if status.ConnectionError != "" {
			cli.Error(fmt.Sprintf("  Error: %s", status.ConnectionError))
		}
		cli.Info("")
		cli.Info("Troubleshooting tips:")
		if status.IsCloudConfigured {
			cli.Info("  • Verify your auth token is valid: turso db tokens list")
			cli.Info("  • Check token file exists: cat %s", status.AuthTokenFile)
			cli.Info("  • Test with turso CLI: turso db shell <database-name>")
			cli.Info("  • Generate new token: turso db tokens create <database-name>")
		} else {
			cli.Info("  • Run: shark init --non-interactive")
			cli.Info("  • Check database file exists: ls -lh shark-tasks.db")
		}
	}

	return nil
}

func runCloudInit(cmd *cobra.Command, args []string) error {
	// Validate URL if provided
	if cloudURL != "" {
		if err := validateTursoURL(cloudURL); err != nil {
			return fmt.Errorf("invalid URL: %w", err)
		}
	} else if cloudNonInteractive {
		return fmt.Errorf("--url is required in non-interactive mode")
	} else {
		return fmt.Errorf("interactive mode not yet implemented; use --url flag")
	}

	// Resolve auth token
	authToken, err := resolveAuthToken(cloudAuthToken)
	if err != nil {
		return fmt.Errorf("failed to resolve auth token: %w", err)
	}

	// If no token resolved, check auth file
	if authToken == "" && cloudAuthFile != "" {
		token, err := db.LoadAuthToken(cloudAuthFile)
		if err != nil {
			return fmt.Errorf("failed to load auth token from file: %w", err)
		}
		authToken = token
	}

	// Validate we have an auth token (required for Turso)
	if authToken == "" && cloudNonInteractive {
		return fmt.Errorf("auth token required: provide --auth-token, --auth-file, or set TURSO_AUTH_TOKEN environment variable")
	}

	// Get project root and config path
	projectRoot, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}
	configPath := filepath.Join(projectRoot, ".sharkconfig.json")

	// Update .sharkconfig.json with turso backend configuration
	if err := updateCloudConfig(configPath, cloudURL, authToken, cloudAuthFile); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	cli.Success("Cloud database configured successfully")
	cli.Info("Backend: turso")
	cli.Info("URL: %s", cloudURL)
	if cloudAuthFile != "" {
		cli.Info("Auth token file: %s", cloudAuthFile)
	} else {
		cli.Info("Auth: %s", maskToken(authToken))
	}
	cli.Info("")
	cli.Info("Configuration saved to: %s", configPath)

	return nil
}

// updateCloudConfig updates the config file with Turso cloud database settings
func updateCloudConfig(configPath, url, authToken, authFile string) error {
	// Load existing config or create new one
	mgr := config.NewManager(configPath)
	cfg, err := mgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Ensure RawData exists
	if cfg.RawData == nil {
		cfg.RawData = make(map[string]interface{})
	}

	// Create database config map
	dbConfig := map[string]interface{}{
		"backend": "turso",
		"url":     url,
	}

	// Add auth token or auth file
	if authFile != "" {
		dbConfig["auth_token_file"] = authFile
	} else if authToken != "" {
		// Store token directly in config (user's choice)
		// Note: This is less secure than using a separate file
		dbConfig["auth_token"] = authToken
	}

	// Update database section
	cfg.RawData["database"] = dbConfig

	// Write config atomically
	return writeConfig(configPath, cfg.RawData)
}

// writeConfig writes the config to disk atomically
func writeConfig(configPath string, rawData map[string]interface{}) error {
	// Get current file permissions if file exists
	var filePerms os.FileMode = 0644
	if info, err := os.Stat(configPath); err == nil {
		filePerms = info.Mode().Perm()
	}

	// Marshal to JSON with pretty formatting
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(rawData); err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	data := buf.Bytes()

	// Write to temp file
	tmpPath := configPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, filePerms); err != nil {
		return fmt.Errorf("failed to write temp config: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, configPath); err != nil {
		os.Remove(tmpPath) // Cleanup temp file on failure
		return fmt.Errorf("failed to rename config: %w", err)
	}

	return nil
}

// validateTursoURL validates the Turso database URL format
func validateTursoURL(url string) error {
	if url == "" {
		return fmt.Errorf("database URL is required")
	}

	// Check for valid scheme (libsql:// or https://, NOT http://)
	if !strings.HasPrefix(url, "libsql://") && !strings.HasPrefix(url, "https://") {
		if strings.Contains(url, "://") {
			return fmt.Errorf("URL must use libsql:// scheme for Turso")
		}
		return fmt.Errorf("invalid URL format")
	}

	// Basic validation passed
	return nil
}

// resolveAuthToken resolves the auth token from various sources
// Priority: direct token > file path > environment variable > empty
// Supports: JWT token directly, file path, or reads TURSO_AUTH_TOKEN env var
func resolveAuthToken(input string) (string, error) {
	// If input provided, determine if it's a token or file path
	if input != "" {
		// Try as direct token first (starts with eyJ)
		if strings.HasPrefix(input, "eyJ") {
			if err := db.ValidateAuthToken(input); err != nil {
				return "", fmt.Errorf("invalid auth token format: %w", err)
			}
			return input, nil
		}

		// Try as file path using db.LoadAuthToken
		// LoadAuthToken will check if it's a file, and if not, return it as-is
		token, err := db.LoadAuthToken(input)
		if err != nil {
			return "", fmt.Errorf("failed to load auth token: %w", err)
		}
		if token != "" {
			return token, nil
		}
	}

	// Check environment variable
	envToken := os.Getenv("TURSO_AUTH_TOKEN")
	if envToken != "" {
		envToken = strings.TrimSpace(envToken)
		if err := db.ValidateAuthToken(envToken); err != nil {
			return "", fmt.Errorf("invalid TURSO_AUTH_TOKEN environment variable: %w", err)
		}
		return envToken, nil
	}

	// No token found (not an error - might be loaded separately)
	return "", nil
}

// maskToken masks an auth token for display (show first/last 4 chars)
func maskToken(token string) string {
	if token == "" {
		return "(none)"
	}
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// CloudStatus represents the status of cloud database configuration
type CloudStatus struct {
	Backend           string `json:"backend"`
	URL               string `json:"url"`
	AuthTokenFile     string `json:"auth_token_file,omitempty"`
	IsCloudConfigured bool   `json:"is_cloud_configured"`
	ConnectionTested  bool   `json:"connection_tested"`
	ConnectionStatus  string `json:"connection_status,omitempty"` // "success" or "failed"
	ConnectionError   string `json:"connection_error,omitempty"`
}

// getCloudStatus reads config and returns cloud status
func getCloudStatus(configPath string) (*CloudStatus, error) {
	mgr := config.NewManager(configPath)
	cfg, err := mgr.Load()
	if err != nil {
		return nil, err
	}

	status := &CloudStatus{}

	// Check if database config exists
	if cfg.RawData == nil {
		return status, nil
	}

	dbConfigRaw, ok := cfg.RawData["database"]
	if !ok {
		return status, nil
	}

	dbConfig, ok := dbConfigRaw.(map[string]interface{})
	if !ok {
		return status, nil
	}

	// Extract backend
	if backend, ok := dbConfig["backend"].(string); ok {
		status.Backend = backend
		status.IsCloudConfigured = (backend == "turso")
	}

	// Extract URL
	if url, ok := dbConfig["url"].(string); ok {
		status.URL = url
	}

	// Extract auth token file
	if authFile, ok := dbConfig["auth_token_file"].(string); ok {
		status.AuthTokenFile = authFile
	}

	return status, nil
}
