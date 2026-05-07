package cli

import "github.com/jwwelbor/shark-task-manager/internal/services"

// getSizeEnforcement returns the SizeEnforcementConfig for the current project.
// When the config cannot be loaded, it returns EmptySizeEnforcementConfig{} so
// size enforcement is disabled gracefully (matches the tag enforcement
// degrade-on-error pattern). Wired into each entity service accessor in
// services_global.go via SetSizeEnforcement.
func getSizeEnforcement() services.SizeEnforcementConfig {
	if cfg, err := GetConfig(); err == nil && cfg != nil {
		return cfg
	}
	return services.EmptySizeEnforcementConfig{}
}
