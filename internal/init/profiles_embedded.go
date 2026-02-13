package init

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed profiles/*.json
var embeddedProfiles embed.FS

// GetProfileMap loads a profile JSON file by name and returns it as a raw map.
// This allows profiles to carry any JSON fields (orchestrator_action, epic_workflow, etc.)
// without requiring Go struct changes.
func GetProfileMap(name string) (map[string]interface{}, error) {
	name = strings.ToLower(name)
	if name == "" {
		return nil, fmt.Errorf("profile name cannot be empty")
	}

	filename := fmt.Sprintf("profiles/%s.json", name)
	data, err := embeddedProfiles.ReadFile(filename)
	if err != nil {
		available, listErr := ListProfiles()
		if listErr != nil {
			return nil, fmt.Errorf("profile not found: %s (and failed to list available profiles: %w)", name, listErr)
		}
		return nil, fmt.Errorf("profile not found: %s (available: %s)", name, strings.Join(available, ", "))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse profile %s: %w", name, err)
	}

	return result, nil
}
