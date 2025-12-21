package discovery

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// FrontmatterData represents parsed YAML frontmatter from a markdown file
type FrontmatterData struct {
	EpicKey          string
	FeatureKey       string
	Title            string
	Description      string
	CustomFolderPath *string
}

// ParseFrontmatter reads and parses YAML frontmatter from a markdown file
// Returns frontmatter data, content (everything after frontmatter), and error
func ParseFrontmatter(filePath string) (*FrontmatterData, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// First line should be "---"
	if !scanner.Scan() {
		return nil, "", fmt.Errorf("empty file or read error: %s", filePath)
	}
	firstLine := strings.TrimSpace(scanner.Text())
	if firstLine != "---" {
		// No frontmatter, return empty data
		return &FrontmatterData{}, "", nil
	}

	// Read frontmatter until closing "---"
	var frontmatterLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		frontmatterLines = append(frontmatterLines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("error reading frontmatter from %s: %w", filePath, err)
	}

	// Parse YAML frontmatter
	frontmatter := strings.Join(frontmatterLines, "\n")
	rawData := make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(frontmatter), &rawData); err != nil {
		// If YAML parsing fails, return empty data (graceful degradation)
		return &FrontmatterData{}, "", nil
	}

	// Extract fields
	data := &FrontmatterData{}

	// Epic key
	if val, ok := rawData["epic_key"]; ok {
		if str, ok := val.(string); ok {
			data.EpicKey = str
		}
	}

	// Feature key
	if val, ok := rawData["feature_key"]; ok {
		if str, ok := val.(string); ok {
			data.FeatureKey = str
		}
	}

	// Title
	if val, ok := rawData["title"]; ok {
		if str, ok := val.(string); ok {
			data.Title = str
		}
	}

	// Description
	if val, ok := rawData["description"]; ok {
		if str, ok := val.(string); ok {
			data.Description = str
		}
	}

	// Custom folder path
	if val, ok := rawData["custom_folder_path"]; ok {
		if str, ok := val.(string); ok && str != "" {
			data.CustomFolderPath = &str
		}
	}

	// Read remaining content (markdown body)
	var contentLines []string
	for scanner.Scan() {
		contentLines = append(contentLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return data, "", fmt.Errorf("error reading content from %s: %w", filePath, err)
	}

	content := strings.Join(contentLines, "\n")

	return data, content, nil
}
