package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/patterns"
)

func TestIntegrationWithRealTaskFiles(t *testing.T) {
	// Find project root by looking for go.mod
	root, err := findProjectRoot()
	if err != nil {
		t.Skip("Skipping integration test: cannot find project root")
	}

	tests := []struct {
		name         string
		filePath     string
		wantTitle    string
		wantTaskKey  string
	}{
		{
			name:        "T-E06-F03-002 task file",
			filePath:    "docs/plan/E06-intelligent-scanning/E06-F03-task-recognition-import/tasks/T-E06-F03-002.md",
			wantTitle:   "Multi-Source Metadata Extraction System",
			wantTaskKey: "T-E06-F03-002",
		},
		{
			name:        "T-E04-F07-002 task file",
			filePath:    "docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/tasks/T-E04-F07-002.md",
			wantTitle:   "Initialization Command Implementation",
			wantTaskKey: "T-E04-F07-002",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Read the file
			fullPath := filepath.Join(root, tt.filePath)
			content, err := os.ReadFile(fullPath)
			if err != nil {
				t.Skipf("Cannot read file %s: %v", fullPath, err)
				return
			}

			// Create a pattern match for standard task format
			patternMatch := &patterns.MatchResult{
				Matched: true,
				CaptureGroups: map[string]string{
					"task_key": tt.wantTaskKey,
				},
			}

			// Extract metadata
			metadata, warnings := ExtractMetadata(string(content), filepath.Base(tt.filePath), patternMatch)

			// Verify task key was extracted from frontmatter
			if metadata.TaskKey != tt.wantTaskKey {
				t.Errorf("TaskKey = %q, want %q", metadata.TaskKey, tt.wantTaskKey)
			}

			// Verify title was extracted from frontmatter
			if metadata.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", metadata.Title, tt.wantTitle)
			}

			// Should not have warnings for well-formed task files
			if len(warnings) > 0 {
				t.Logf("Warnings: %v", warnings)
			}

			// Verify frontmatter parsing worked
			fm, err := ParseFrontmatter(string(content))
			if err != nil {
				t.Errorf("Failed to parse frontmatter: %v", err)
			}

			if !fm.HasFrontmatter {
				t.Error("Expected frontmatter but found none")
			}
		})
	}
}

// findProjectRoot finds the project root by looking for go.mod
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func TestExtractMetadataFromVariousFormats(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		content      string
		patternMatch *patterns.MatchResult
		wantTitle    string
		wantDesc     string
	}{
		{
			name:     "Standard task with full frontmatter",
			filename: "T-E04-F02-001-implement-auth.md",
			content: `---
task_key: T-E04-F02-001
title: Implement Authentication
description: Add JWT-based authentication to the API
status: todo
---

# Implementation

This task involves adding authentication.`,
			patternMatch: &patterns.MatchResult{
				Matched: true,
				CaptureGroups: map[string]string{
					"task_key": "T-E04-F02-001",
					"slug":     "implement-auth",
				},
			},
			wantTitle: "Implement Authentication",
			wantDesc:  "Add JWT-based authentication to the API",
		},
		{
			name:     "Numbered task without frontmatter",
			filename: "01-research-authentication.md",
			content: `# Task: Research Authentication Options

Research different authentication approaches for the API.

## Options

1. JWT
2. OAuth
3. Session-based`,
			patternMatch: &patterns.MatchResult{
				Matched: true,
				CaptureGroups: map[string]string{
					"number": "01",
				},
			},
			wantTitle: "Research Authentication",
			wantDesc:  "Research different authentication approaches for the API.",
		},
		{
			name:     "PRP file with minimal frontmatter",
			filename: "implement-caching.prp.md",
			content: `---
task_key: T-E04-F02-003
---

# PRP: Implement Caching Layer

Add Redis-based caching to improve API performance.

## Requirements

- Cache GET requests
- Invalidate on updates`,
			patternMatch: &patterns.MatchResult{
				Matched: true,
				CaptureGroups: map[string]string{
					"slug": "implement-caching",
				},
			},
			wantTitle: "Implement Caching", // Extracted from filename slug (priority 2)
			wantDesc:  "Add Redis-based caching to improve API performance.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, warnings := ExtractMetadata(tt.content, tt.filename, tt.patternMatch)

			if metadata.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", metadata.Title, tt.wantTitle)
			}

			if metadata.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", metadata.Description, tt.wantDesc)
			}

			// Log warnings for debugging
			if len(warnings) > 0 {
				t.Logf("Warnings: %v", warnings)
			}
		})
	}
}
