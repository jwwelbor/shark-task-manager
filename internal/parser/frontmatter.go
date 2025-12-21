package parser

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Frontmatter represents parsed YAML frontmatter from a task file
type Frontmatter struct {
	HasFrontmatter bool
	TaskKey        string   `yaml:"task_key"`
	Title          string   `yaml:"title"`
	Description    string   `yaml:"description"`
	Status         string   `yaml:"status"`
	Feature        string   `yaml:"feature"`
	AssignedAgent  string   `yaml:"assigned_agent"`
	Priority       int      `yaml:"priority"`
	BlockedReason  string   `yaml:"blocked_reason"`
	AgentType      string   `yaml:"agent_type"`
	Created        string   `yaml:"created"`
	Dependencies   []string `yaml:"dependencies"`
}

// ParseFrontmatter extracts and parses YAML frontmatter from markdown content
// Returns Frontmatter struct with HasFrontmatter=false if no frontmatter present
// Returns error if frontmatter is present but has invalid YAML syntax
func ParseFrontmatter(content string) (*Frontmatter, error) {
	fm := &Frontmatter{
		HasFrontmatter: false,
	}

	// Check if content starts with frontmatter delimiter
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return fm, nil
	}

	// Find closing delimiter
	lines := strings.Split(content, "\n")
	closingIndex := -1
	for i := 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "---" {
			closingIndex = i
			break
		}
	}

	if closingIndex == -1 {
		return nil, fmt.Errorf("frontmatter missing closing delimiter '---'")
	}

	// Extract frontmatter content (between delimiters)
	frontmatterLines := lines[1:closingIndex]
	frontmatterContent := strings.Join(frontmatterLines, "\n")

	// Parse YAML
	if err := yaml.Unmarshal([]byte(frontmatterContent), fm); err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	fm.HasFrontmatter = true
	return fm, nil
}

// GetContentAfterFrontmatter returns the markdown content after frontmatter
// If no frontmatter exists, returns the entire content
func GetContentAfterFrontmatter(content string, fm *Frontmatter) string {
	if !fm.HasFrontmatter {
		return content
	}

	// Find closing delimiter
	lines := strings.Split(content, "\n")
	for i := 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "---" {
			// Return everything after the closing delimiter
			if i+1 < len(lines) {
				return strings.Join(lines[i+1:], "\n")
			}
			return ""
		}
	}

	return content
}

// UpdateFrontmatterField updates or adds a field in the frontmatter
// Creates frontmatter if it doesn't exist
// Preserves existing fields and markdown content
func UpdateFrontmatterField(content, fieldName, value string) (string, error) {
	fm, err := ParseFrontmatter(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse existing frontmatter: %w", err)
	}

	var updatedContent string

	if !fm.HasFrontmatter {
		// Create new frontmatter
		updatedContent = fmt.Sprintf("---\n%s: %s\n---\n\n%s", fieldName, value, content)
	} else {
		// Parse existing frontmatter into map for manipulation
		lines := strings.Split(content, "\n")
		closingIndex := -1
		for i := 1; i < len(lines); i++ {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == "---" {
				closingIndex = i
				break
			}
		}

		if closingIndex == -1 {
			return "", fmt.Errorf("frontmatter missing closing delimiter")
		}

		// Extract frontmatter and body
		frontmatterLines := lines[1:closingIndex]
		bodyLines := lines[closingIndex+1:]

		// Parse frontmatter into map
		var frontmatterMap map[string]interface{}
		frontmatterContent := strings.Join(frontmatterLines, "\n")
		if err := yaml.Unmarshal([]byte(frontmatterContent), &frontmatterMap); err != nil {
			return "", fmt.Errorf("failed to parse frontmatter for update: %w", err)
		}

		if frontmatterMap == nil {
			frontmatterMap = make(map[string]interface{})
		}

		// Update or add field
		frontmatterMap[fieldName] = value

		// Convert back to YAML
		updatedYAML, err := yaml.Marshal(frontmatterMap)
		if err != nil {
			return "", fmt.Errorf("failed to marshal updated frontmatter: %w", err)
		}

		// Reconstruct content
		updatedContent = fmt.Sprintf("---\n%s---\n%s",
			string(updatedYAML),
			strings.Join(bodyLines, "\n"))
	}

	return updatedContent, nil
}
