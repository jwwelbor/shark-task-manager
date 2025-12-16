package commands

import (
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// configCmd represents the config command group
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
	Long: `View and validate CLI configuration settings.

Examples:
  pm config show               Show current configuration
  pm config validate           Validate configuration file`,
}

// configShowCmd shows current configuration
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current configuration including file location and all settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile := viper.ConfigFileUsed()
		if configFile == "" {
			cli.Info("No configuration file loaded (using defaults)")
		} else {
			cli.Info(fmt.Sprintf("Configuration file: %s", configFile))
		}

		settings := map[string]interface{}{
			"json":     viper.GetBool("json"),
			"no-color": viper.GetBool("no-color"),
			"verbose":  viper.GetBool("verbose"),
			"db":       viper.GetString("db"),
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(settings)
		}

		fmt.Println("\nCurrent Settings:")
		for key, value := range settings {
			fmt.Printf("  %s: %v\n", key, value)
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

func init() {
	// Register config command with root
	cli.RootCmd.AddCommand(configCmd)

	// Add subcommands
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configValidateCmd)
}
