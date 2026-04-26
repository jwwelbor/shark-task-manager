package cli

import (
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/auth/maintainer"
	"github.com/jwwelbor/shark-task-manager/internal/config"
)

// GetMaintainerGate returns a maintainer.Gate instance configured for the
// current project root and .sharkconfig.json.
//
// Behaviour:
//   - Calls FindProjectRoot() and panics if the project root cannot be found
//     (consistent with other Get*Service() accessors that panic on failure).
//   - Calls GetConfig() and panics if the config cannot be loaded.
//   - If the loaded config has no Maintainer section (nil), returns a gate that
//     always returns *UnauthorizedError — does NOT panic.
//   - Returns a new *maintainer.FileGate instance on every call (no singleton).
//     This matches the "new instance per call" pattern used by GetTaskService()
//     and avoids stale-gate problems when the config changes between calls.
//
// Usage:
//
//	gate := cli.GetMaintainerGate()
//	if err := gate.Authorize(cmd.Context(), pass); err != nil {
//	    return err
//	}
//	defer func() { _ = gate.RecordSuccess(cmd.Context()) }()
//
// Spec reference: spec.md REQ-F-009, §2.6 (GetMaintainerGate code shape).
func GetMaintainerGate() maintainer.Gate {
	projectRoot, err := FindProjectRoot()
	if err != nil {
		panic(fmt.Sprintf("failed to find project root for maintainer gate: %v", err))
	}

	cfg, err := GetConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to load config for maintainer gate: %v", err))
	}

	var mc *config.MaintainerConfig
	if cfg != nil {
		mc = cfg.Maintainer
	}

	// mc may be nil — NewFileGate accepts nil and returns a gate that always fails
	// with *UnauthorizedError{Reason: "missing_config"}.
	return maintainer.NewFileGate(projectRoot, mc, mc.CacheWindow())
}
