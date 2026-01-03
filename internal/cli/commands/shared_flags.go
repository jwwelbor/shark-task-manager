package commands

import (
	"github.com/spf13/cobra"
)

// FlagSet represents a group of related flags that can be added to commands
type FlagSet string

const (
	FlagSetMetadata      FlagSet = "metadata"       // --title, --description
	FlagSetPath          FlagSet = "path"           // --path, --filename, --force
	FlagSetEpicStatus    FlagSet = "epic_status"    // --status, --priority, --business-value
	FlagSetFeatureStatus FlagSet = "feature_status" // --status
	FlagSetCustomKey     FlagSet = "custom_key"     // --key
)

// flagConfig holds configuration for flag registration
type flagConfig struct {
	defaults map[string]interface{}
	required []string
}

// FlagOption allows customization of flag behavior
type FlagOption func(*flagConfig)

// WithRequired marks flags as required
func WithRequired(flagNames ...string) FlagOption {
	return func(cfg *flagConfig) {
		cfg.required = append(cfg.required, flagNames...)
	}
}

// WithDefaults sets default values for flags
func WithDefaults(defaults map[string]interface{}) FlagOption {
	return func(cfg *flagConfig) {
		cfg.defaults = defaults
	}
}

// AddFlagSet adds a predefined set of flags to a command
// This is the primary API for composing flag groups
func AddFlagSet(cmd *cobra.Command, flagSet FlagSet, opts ...FlagOption) {
	// Apply options
	cfg := &flagConfig{
		defaults: make(map[string]interface{}),
		required: []string{},
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Register flags based on flag set type
	switch flagSet {
	case FlagSetMetadata:
		addMetadataFlagsWithConfig(cmd, cfg)
	case FlagSetPath:
		addPathFlagsWithConfig(cmd, cfg)
	case FlagSetEpicStatus:
		addEpicStatusFlagsWithConfig(cmd, cfg)
	case FlagSetFeatureStatus:
		addFeatureStatusFlagsWithConfig(cmd, cfg)
	case FlagSetCustomKey:
		addCustomKeyFlagWithConfig(cmd, cfg)
	default:
		panic("invalid flag set: " + string(flagSet))
	}

	// Mark required flags
	for _, flagName := range cfg.required {
		_ = cmd.MarkFlagRequired(flagName)
	}
}

// addMetadataFlagsWithConfig adds metadata flags with configuration
func addMetadataFlagsWithConfig(cmd *cobra.Command, cfg *flagConfig) {
	titleDefault := ""
	descDefault := ""

	if val, ok := cfg.defaults["title"]; ok {
		if strVal, ok := val.(string); ok {
			titleDefault = strVal
		}
	}
	if val, ok := cfg.defaults["description"]; ok {
		if strVal, ok := val.(string); ok {
			descDefault = strVal
		}
	}

	cmd.Flags().String("title", titleDefault, "Title")
	cmd.Flags().String("description", descDefault, "Description (optional)")
}

// addPathFlagsWithConfig adds path flags with configuration
func addPathFlagsWithConfig(cmd *cobra.Command, cfg *flagConfig) {
	pathDefault := ""
	filenameDefault := ""
	forceDefault := false

	if val, ok := cfg.defaults["path"]; ok {
		if strVal, ok := val.(string); ok {
			pathDefault = strVal
		}
	}
	if val, ok := cfg.defaults["filename"]; ok {
		if strVal, ok := val.(string); ok {
			filenameDefault = strVal
		}
	}
	if val, ok := cfg.defaults["force"]; ok {
		if boolVal, ok := val.(bool); ok {
			forceDefault = boolVal
		}
	}

	cmd.Flags().String("path", pathDefault, "Custom base folder path (relative to project root)")
	cmd.Flags().String("filename", filenameDefault, "Custom filename path (relative to project root, must end in .md)")
	cmd.Flags().Bool("force", forceDefault, "Force reassignment if file already claimed")
}

// addEpicStatusFlagsWithConfig adds epic status flags with configuration
func addEpicStatusFlagsWithConfig(cmd *cobra.Command, cfg *flagConfig) {
	statusDefault := ""
	priorityDefault := ""
	bvDefault := ""

	if val, ok := cfg.defaults["status"]; ok {
		if strVal, ok := val.(string); ok {
			statusDefault = strVal
		}
	}
	if val, ok := cfg.defaults["priority"]; ok {
		if strVal, ok := val.(string); ok {
			priorityDefault = strVal
		}
	}
	if val, ok := cfg.defaults["business-value"]; ok {
		if strVal, ok := val.(string); ok {
			bvDefault = strVal
		}
	}

	cmd.Flags().String("status", statusDefault, "Status: draft, active, completed, archived")
	cmd.Flags().String("priority", priorityDefault, "Priority: low, medium, high")
	cmd.Flags().String("business-value", bvDefault, "Business value: low, medium, high (optional)")
}

// addFeatureStatusFlagsWithConfig adds feature status flags with configuration
func addFeatureStatusFlagsWithConfig(cmd *cobra.Command, cfg *flagConfig) {
	statusDefault := ""

	if val, ok := cfg.defaults["status"]; ok {
		if strVal, ok := val.(string); ok {
			statusDefault = strVal
		}
	}

	cmd.Flags().String("status", statusDefault, "Status: draft, active, completed, archived")
}

// addCustomKeyFlagWithConfig adds custom key flag with configuration
func addCustomKeyFlagWithConfig(cmd *cobra.Command, cfg *flagConfig) {
	keyDefault := ""

	if val, ok := cfg.defaults["key"]; ok {
		if strVal, ok := val.(string); ok {
			keyDefault = strVal
		}
	}

	cmd.Flags().String("key", keyDefault, "Custom key")
}

// AddMetadataFlags adds metadata flags to a command
// Individual flag registration function for granular control
func AddMetadataFlags(cmd *cobra.Command) {
	cmd.Flags().String("title", "", "Title")
	cmd.Flags().String("description", "", "Description (optional)")
}

// AddPathFlags adds path-related flags to a command
// Individual flag registration function for granular control
func AddPathFlags(cmd *cobra.Command) {
	cmd.Flags().String("path", "", "Custom base folder path (relative to project root)")
	cmd.Flags().String("filename", "", "Custom filename path (relative to project root, must end in .md)")
	cmd.Flags().Bool("force", false, "Force reassignment if file already claimed")
}

// AddEpicStatusFlags adds epic status flags to a command
// Individual flag registration function for granular control
func AddEpicStatusFlags(cmd *cobra.Command, defaults map[string]string) {
	statusDefault := ""
	priorityDefault := ""
	bvDefault := ""

	if val, ok := defaults["status"]; ok {
		statusDefault = val
	}
	if val, ok := defaults["priority"]; ok {
		priorityDefault = val
	}
	if val, ok := defaults["business-value"]; ok {
		bvDefault = val
	}

	cmd.Flags().String("status", statusDefault, "Status: draft, active, completed, archived")
	cmd.Flags().String("priority", priorityDefault, "Priority: low, medium, high")
	cmd.Flags().String("business-value", bvDefault, "Business value: low, medium, high (optional)")
}

// AddFeatureStatusFlags adds feature status flags to a command
// Individual flag registration function for granular control
func AddFeatureStatusFlags(cmd *cobra.Command, defaults map[string]string) {
	statusDefault := ""

	if val, ok := defaults["status"]; ok {
		statusDefault = val
	}

	cmd.Flags().String("status", statusDefault, "Status: draft, active, completed, archived")
}

// AddCustomKeyFlag adds custom key flag to a command
// Individual flag registration function for granular control
func AddCustomKeyFlag(cmd *cobra.Command) {
	cmd.Flags().String("key", "", "Custom key")
}
