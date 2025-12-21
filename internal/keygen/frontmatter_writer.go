package keygen

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// FrontmatterWriter handles atomic updates to YAML frontmatter in markdown files
type FrontmatterWriter struct {
	preservePermissions bool
}

// NewFrontmatterWriter creates a new FrontmatterWriter
func NewFrontmatterWriter() *FrontmatterWriter {
	return &FrontmatterWriter{
		preservePermissions: true,
	}
}

// Frontmatter represents the YAML frontmatter structure
type Frontmatter map[string]interface{}

// WriteTaskKey updates or adds the task_key field in a file's frontmatter
// This operation is atomic: it writes to a temp file and then renames (POSIX atomic operation)
func (w *FrontmatterWriter) WriteTaskKey(filePath string, taskKey string) error {
	// Read current file contents
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Get original file permissions
	var originalPerm os.FileMode
	if w.preservePermissions {
		info, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("failed to stat file: %w", err)
		}
		originalPerm = info.Mode()
	}

	// Parse and update frontmatter
	updatedContent, err := w.updateFrontmatter(content, taskKey)
	if err != nil {
		return fmt.Errorf("failed to update frontmatter: %w", err)
	}

	// Write to temporary file
	tempFile, err := os.CreateTemp(filepath.Dir(filePath), ".shark-tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Ensure temp file is cleaned up on error
	defer func() {
		if err != nil {
			os.Remove(tempPath)
		}
	}()

	// Write updated content
	if _, err := tempFile.Write(updatedContent); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Close temp file
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set permissions on temp file to match original
	if w.preservePermissions {
		if err := os.Chmod(tempPath, originalPerm); err != nil {
			return fmt.Errorf("failed to set permissions on temp file: %w", err)
		}
	}

	// Atomic rename (POSIX guarantees atomicity)
	if err := os.Rename(tempPath, filePath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// updateFrontmatter updates the frontmatter with the task_key field
func (w *FrontmatterWriter) updateFrontmatter(content []byte, taskKey string) ([]byte, error) {
	reader := bytes.NewReader(content)
	scanner := bufio.NewScanner(reader)

	var result bytes.Buffer
	var frontmatterLines []string
	var afterFrontmatter []string
	inFrontmatter := false
	frontmatterFound := false
	frontmatterClosed := false

	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Check for frontmatter delimiter
		if line == "---" {
			if !frontmatterFound {
				// First delimiter - opening frontmatter
				frontmatterFound = true
				inFrontmatter = true
				continue // Don't add delimiter yet
			} else if inFrontmatter {
				// Second delimiter - closing frontmatter
				inFrontmatter = false
				frontmatterClosed = true
				continue // Don't add delimiter yet
			}
		}

		if inFrontmatter {
			// Collect frontmatter lines
			frontmatterLines = append(frontmatterLines, line)
		} else if frontmatterClosed {
			// Collect lines after frontmatter
			afterFrontmatter = append(afterFrontmatter, line)
		} else if !frontmatterFound {
			// No frontmatter found, collect as after content
			afterFrontmatter = append(afterFrontmatter, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan file: %w", err)
	}

	// Parse existing frontmatter or create new
	var fm Frontmatter
	if len(frontmatterLines) > 0 {
		// Parse existing YAML
		yamlContent := strings.Join(frontmatterLines, "\n")
		if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
			return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
		}
	} else {
		// Create new frontmatter
		fm = make(Frontmatter)
	}

	// Update task_key field
	fm["task_key"] = taskKey

	// Marshal back to YAML
	yamlData, err := yaml.Marshal(fm)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Build final content
	result.WriteString("---\n")
	result.Write(yamlData)
	result.WriteString("---\n")

	// Add content after frontmatter
	if len(afterFrontmatter) > 0 {
		// If frontmatter existed and first line of content is blank, preserve it
		if frontmatterFound && len(afterFrontmatter) > 0 && afterFrontmatter[0] == "" {
			result.WriteString("\n")
			if len(afterFrontmatter) > 1 {
				result.WriteString(strings.Join(afterFrontmatter[1:], "\n"))
			}
		} else {
			// If no frontmatter existed, add blank line before content
			result.WriteString("\n")
			result.WriteString(strings.Join(afterFrontmatter, "\n"))
		}
		// Ensure file ends with newline
		if len(afterFrontmatter) > 0 && afterFrontmatter[len(afterFrontmatter)-1] != "" {
			result.WriteString("\n")
		}
	} else {
		result.WriteString("\n")
	}

	return result.Bytes(), nil
}

// ReadFrontmatter reads and parses the frontmatter from a file
func (w *FrontmatterWriter) ReadFrontmatter(filePath string) (Frontmatter, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	reader := bytes.NewReader(content)
	scanner := bufio.NewScanner(reader)

	var frontmatterLines []string
	inFrontmatter := false
	frontmatterFound := false

	for scanner.Scan() {
		line := scanner.Text()

		if line == "---" {
			if !frontmatterFound {
				frontmatterFound = true
				inFrontmatter = true
				continue
			} else if inFrontmatter {
				// Closing delimiter
				break
			}
		}

		if inFrontmatter {
			frontmatterLines = append(frontmatterLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan file: %w", err)
	}

	if len(frontmatterLines) == 0 {
		return make(Frontmatter), nil // Empty frontmatter
	}

	var fm Frontmatter
	yamlContent := strings.Join(frontmatterLines, "\n")
	if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
		// Gracefully handle malformed YAML by returning empty frontmatter
		// This allows sync to continue and generate a key even if YAML is malformed
		fm = make(Frontmatter)
	}

	return fm, nil
}

// HasTaskKey checks if a file already has a task_key in its frontmatter
func (w *FrontmatterWriter) HasTaskKey(filePath string) (bool, string, error) {
	fm, err := w.ReadFrontmatter(filePath)
	if err != nil {
		return false, "", err
	}

	if taskKey, ok := fm["task_key"]; ok {
		if keyStr, ok := taskKey.(string); ok && keyStr != "" {
			return true, keyStr, nil
		}
	}

	return false, "", nil
}

// ValidateFileWritable checks if a file can be written to
func (w *FrontmatterWriter) ValidateFileWritable(filePath string) error {
	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("file does not exist or cannot be accessed: %w", err)
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", filePath)
	}

	// Check if we can write to the directory (needed for atomic rename)
	dir := filepath.Dir(filePath)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("directory not accessible: %w", err)
	}

	// Try to open file for writing (without actually writing)
	file, err := os.OpenFile(filePath, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("file not writable: %w", err)
	}
	file.Close()

	return nil
}

// ExtractContentReader returns a reader for the markdown content (without frontmatter)
func (w *FrontmatterWriter) ExtractContentReader(filePath string) (io.Reader, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	reader := bytes.NewReader(content)
	scanner := bufio.NewScanner(reader)

	var contentLines []string
	inFrontmatter := false
	frontmatterFound := false
	frontmatterClosed := false

	for scanner.Scan() {
		line := scanner.Text()

		if line == "---" {
			if !frontmatterFound {
				frontmatterFound = true
				inFrontmatter = true
				continue
			} else if inFrontmatter {
				inFrontmatter = false
				frontmatterClosed = true
				continue
			}
		}

		if !inFrontmatter && (frontmatterClosed || !frontmatterFound) {
			contentLines = append(contentLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan file: %w", err)
	}

	content_str := strings.Join(contentLines, "\n")
	return strings.NewReader(content_str), nil
}
