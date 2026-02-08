// Package taskfile provides utilities for reading and writing task markdown files.
//
// Task files use YAML frontmatter for metadata and markdown for content:
//
//	---
//	task_key: T-E04-F05-001
//	status: todo
//	title: Implement file path utilities
//	---
//
//	# Task Description
//	Implementation details go here...
package taskfile

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// TaskMetadata represents the YAML frontmatter of a task file
type TaskMetadata struct {
	TaskKey       string   `yaml:"task_key"`
	Status        string   `yaml:"status"`
	Title         string   `yaml:"title"`
	Description   string   `yaml:"description,omitempty"`
	Feature       string   `yaml:"feature,omitempty"`
	CreatedAt     string   `yaml:"created,omitempty"`
	AssignedAgent string   `yaml:"assigned_agent,omitempty"`
	Dependencies  []string `yaml:"dependencies,omitempty"`
	EstimatedTime string   `yaml:"estimated_time,omitempty"`
	Priority      int      `yaml:"priority,omitempty"`
}

// TaskFile represents a complete task file with metadata and content
type TaskFile struct {
	Metadata TaskMetadata
	Content  string // Markdown content after frontmatter
}

// ParseTaskFile reads and parses a task file from the given path.
// Returns the parsed metadata and content, or an error if parsing fails.
func ParseTaskFile(filePath string) (*TaskFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open task file %s: %w", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// First line should be "---"
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty file or read error: %s", filePath)
	}
	firstLine := strings.TrimSpace(scanner.Text())
	if firstLine != "---" {
		return nil, fmt.Errorf("invalid task file format: missing frontmatter delimiter (expected '---', got '%s')", firstLine)
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
		return nil, fmt.Errorf("error reading frontmatter from %s: %w", filePath, err)
	}

	// Parse YAML frontmatter
	frontmatter := strings.Join(frontmatterLines, "\n")
	var metadata TaskMetadata
	if err := yaml.Unmarshal([]byte(frontmatter), &metadata); err != nil {
		return nil, fmt.Errorf("error parsing frontmatter in %s: %w", filePath, err)
	}

	// Read remaining content (markdown body)
	var contentLines []string
	for scanner.Scan() {
		contentLines = append(contentLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading content from %s: %w", filePath, err)
	}

	content := strings.Join(contentLines, "\n")

	return &TaskFile{
		Metadata: metadata,
		Content:  content,
	}, nil
}

// ParseTaskFileContent parses a task file from a string (useful for testing)
func ParseTaskFileContent(content string) (*TaskFile, error) {
	lines := strings.Split(content, "\n")

	if len(lines) < 3 {
		return nil, fmt.Errorf("invalid task file: too short")
	}

	// First line should be "---"
	if strings.TrimSpace(lines[0]) != "---" {
		return nil, fmt.Errorf("invalid task file format: missing frontmatter delimiter")
	}

	// Find closing "---"
	closingIndex := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			closingIndex = i
			break
		}
	}

	if closingIndex == -1 {
		return nil, fmt.Errorf("invalid task file format: missing closing frontmatter delimiter")
	}

	// Parse frontmatter
	frontmatter := strings.Join(lines[1:closingIndex], "\n")
	var metadata TaskMetadata
	if err := yaml.Unmarshal([]byte(frontmatter), &metadata); err != nil {
		return nil, fmt.Errorf("error parsing frontmatter: %w", err)
	}

	// Get remaining content
	var bodyLines []string
	if closingIndex+1 < len(lines) {
		bodyLines = lines[closingIndex+1:]
	}
	body := strings.Join(bodyLines, "\n")

	return &TaskFile{
		Metadata: metadata,
		Content:  body,
	}, nil
}

// Validate validates the task metadata.
// This performs basic structural validation (required fields, non-empty checks).
// Status is checked for non-empty only; workflow-aware status validation should
// be performed at the CLI/command layer using workflow.Service.ValidateStatus().
func (m *TaskMetadata) Validate() error {
	if m.TaskKey == "" {
		return fmt.Errorf("task_key is required")
	}
	if m.Status == "" {
		return fmt.Errorf("status is required")
	}
	if m.Title == "" {
		return fmt.Errorf("title is required")
	}

	return nil
}
