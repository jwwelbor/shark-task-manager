package init

import (
	"encoding/json"
	"fmt"
	"strings"
)

// GetProfile retrieves a workflow profile by name (case-insensitive).
// This is a backward-compatible shim that loads from embedded JSON and
// converts to a WorkflowProfile struct. For new code, prefer GetProfileMap()
// which returns the full JSON map including fields not in the struct
// (orchestrator_action, epic_workflow, feature_workflow, etc.).
func GetProfile(name string) (*WorkflowProfile, error) {
	profileMap, err := GetProfileMap(name)
	if err != nil {
		return nil, err
	}

	// Marshal back to JSON and unmarshal into struct
	data, err := json.Marshal(profileMap)
	if err != nil {
		return nil, fmt.Errorf("failed to convert profile %s: %w", name, err)
	}

	var profile WorkflowProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse profile %s into struct: %w", name, err)
	}

	return &profile, nil
}

// ListProfiles returns a list of available profile names.
// An error indicates the embedded filesystem is corrupted or the binary was
// built incorrectly.
func ListProfiles() ([]string, error) {
	entries, err := embeddedProfiles.ReadDir("profiles")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded profiles: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".json") {
			names = append(names, strings.TrimSuffix(name, ".json"))
		}
	}
	return names, nil
}
