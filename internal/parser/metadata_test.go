package parser

import (
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/patterns"
)

func TestExtractTitleFromFilename(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		patternMatch *patterns.MatchResult
		wantTitle    string
	}{
		{
			name:     "standard format with slug",
			filename: "T-E04-F02-001-implement-caching.md",
			patternMatch: &patterns.MatchResult{
				Matched: true,
				CaptureGroups: map[string]string{
					"task_key": "T-E04-F02-001",
					"slug":     "implement-caching",
				},
			},
			wantTitle: "Implement Caching",
		},
		{
			name:     "standard format without slug",
			filename: "T-E04-F02-001.md",
			patternMatch: &patterns.MatchResult{
				Matched: true,
				CaptureGroups: map[string]string{
					"task_key": "T-E04-F02-001",
				},
			},
			wantTitle: "",
		},
		{
			name:     "numbered format",
			filename: "01-research-phase.md",
			patternMatch: &patterns.MatchResult{
				Matched: true,
				CaptureGroups: map[string]string{
					"number": "01",
				},
			},
			wantTitle: "Research Phase",
		},
		{
			name:     "numbered format with three digits",
			filename: "123-authentication-system.md",
			patternMatch: &patterns.MatchResult{
				Matched: true,
				CaptureGroups: map[string]string{
					"number": "123",
				},
			},
			wantTitle: "Authentication System",
		},
		{
			name:     "PRP format",
			filename: "implement-auth.prp.md",
			patternMatch: &patterns.MatchResult{
				Matched: true,
				CaptureGroups: map[string]string{
					"slug": "implement-auth",
				},
			},
			wantTitle: "Implement Auth",
		},
		{
			name:     "PRP format with multiple words",
			filename: "user-management-api.prp.md",
			patternMatch: &patterns.MatchResult{
				Matched: true,
				CaptureGroups: map[string]string{
					"slug": "user-management-api",
				},
			},
			wantTitle: "User Management Api",
		},
		{
			name:     "complex slug with many hyphens",
			filename: "T-E04-F02-001-complex-multi-word-feature-name.md",
			patternMatch: &patterns.MatchResult{
				Matched: true,
				CaptureGroups: map[string]string{
					"task_key": "T-E04-F02-001",
					"slug":     "complex-multi-word-feature-name",
				},
			},
			wantTitle: "Complex Multi Word Feature Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTitleFromFilename(tt.filename, tt.patternMatch)
			if got != tt.wantTitle {
				t.Errorf("ExtractTitleFromFilename() = %q, want %q", got, tt.wantTitle)
			}
		})
	}
}

func TestExtractTitleFromMarkdown(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantTitle string
	}{
		{
			name: "H1 with 'Task:' prefix",
			content: `# Task: Implement User Authentication

Content here.`,
			wantTitle: "Implement User Authentication",
		},
		{
			name: "H1 with 'PRP:' prefix",
			content: `# PRP: Design Database Schema

Content here.`,
			wantTitle: "Design Database Schema",
		},
		{
			name: "H1 with 'TODO:' prefix",
			content: `# TODO: Fix Login Bug

Content here.`,
			wantTitle: "Fix Login Bug",
		},
		{
			name: "H1 with 'WIP:' prefix",
			content: `# WIP: Refactor Auth Module

Content here.`,
			wantTitle: "Refactor Auth Module",
		},
		{
			name: "H1 without prefix",
			content: `# Implement Caching Layer

Content here.`,
			wantTitle: "Implement Caching Layer",
		},
		{
			name: "H1 with case variations",
			content: `# task: Update Documentation

Content here.`,
			wantTitle: "Update Documentation",
		},
		{
			name: "no H1 heading",
			content: `Some content without heading.

## H2 Heading`,
			wantTitle: "",
		},
		{
			name: "H1 after frontmatter",
			content: `---
task_key: T-E04-F02-001
---

# Task: Build API Endpoint

Content here.`,
			wantTitle: "Build API Endpoint",
		},
		{
			name: "empty H1",
			content: `#

Content here.`,
			wantTitle: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTitleFromMarkdown(tt.content)
			if got != tt.wantTitle {
				t.Errorf("ExtractTitleFromMarkdown() = %q, want %q", got, tt.wantTitle)
			}
		})
	}
}

func TestExtractDescriptionFromMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantDesc string
	}{
		{
			name: "first paragraph after frontmatter and H1",
			content: `---
task_key: T-E04-F02-001
---

# Task: Implementation

This is the first paragraph describing the task.
It can span multiple lines.

This is the second paragraph.`,
			wantDesc: "This is the first paragraph describing the task.\nIt can span multiple lines.",
		},
		{
			name: "paragraph without frontmatter",
			content: `# PRP: User Authentication

Implement JWT-based authentication for the API. This includes login, logout, and token refresh endpoints.

## Requirements

...`,
			wantDesc: "Implement JWT-based authentication for the API. This includes login, logout, and token refresh endpoints.",
		},
		{
			name: "no paragraph after heading",
			content: `# Task

## Acceptance Criteria

...`,
			wantDesc: "",
		},
		{
			name: "long paragraph (truncated to 500 chars)",
			content: `# Task

` + strings.Repeat("This is a very long paragraph that needs to be truncated. ", 20) + `
More content here that should be truncated.`,
			wantDesc: (strings.Repeat("This is a very long paragraph that needs to be truncated. ", 20))[:500],
		},
		{
			name: "paragraph with blank lines preserved",
			content: `# Task

First line of paragraph.

Second line after blank.

Next paragraph.`,
			wantDesc: "First line of paragraph.",
		},
		{
			name: "no frontmatter or heading",
			content: `This is just content.
Multiple lines.

Another paragraph.`,
			wantDesc: "",
		},
		{
			name: "paragraph immediately after H1",
			content: `# Task
Immediate content without blank line.

More content.`,
			wantDesc: "Immediate content without blank line.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractDescriptionFromMarkdown(tt.content)
			if got != tt.wantDesc {
				t.Errorf("ExtractDescriptionFromMarkdown() = %q, want %q", got, tt.wantDesc)
			}
		})
	}
}

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		filename     string
		patternMatch *patterns.MatchResult
		wantTitle    string
		wantDesc     string
		wantWarnings []string
	}{
		{
			name: "all metadata in frontmatter (priority 1)",
			content: `---
task_key: T-E04-F02-001
title: Implement User Authentication
description: Add JWT-based authentication for the API
---

# Task: Something Else

Different content here.`,
			filename: "T-E04-F02-001-other-title.md",
			patternMatch: &patterns.MatchResult{
				Matched: true,
				CaptureGroups: map[string]string{
					"task_key": "T-E04-F02-001",
					"slug":     "other-title",
				},
			},
			wantTitle: "Implement User Authentication",
			wantDesc:  "Add JWT-based authentication for the API",
		},
		{
			name: "title from filename, description from markdown (priority 2/2)",
			content: `---
task_key: T-E04-F02-002
---

# Task: Ignored Title

This is the description from the markdown body.

More content.`,
			filename: "T-E04-F02-002-implement-caching.md",
			patternMatch: &patterns.MatchResult{
				Matched: true,
				CaptureGroups: map[string]string{
					"task_key": "T-E04-F02-002",
					"slug":     "implement-caching",
				},
			},
			wantTitle: "Implement Caching",
			wantDesc:  "This is the description from the markdown body.",
		},
		{
			name: "title from H1, no description (priority 3)",
			content: `---
task_key: T-E04-F02-003
---

# Task: Build API Endpoint

## Acceptance Criteria

...`,
			filename: "T-E04-F02-003.md",
			patternMatch: &patterns.MatchResult{
				Matched: true,
				CaptureGroups: map[string]string{
					"task_key": "T-E04-F02-003",
				},
			},
			wantTitle: "Build API Endpoint",
			wantDesc:  "",
		},
		{
			name: "no title from any source (use placeholder)",
			content: `---
task_key: T-E04-F02-004
---

Some content without title.`,
			filename: "T-E04-F02-004.md",
			patternMatch: &patterns.MatchResult{
				Matched: true,
				CaptureGroups: map[string]string{
					"task_key": "T-E04-F02-004",
				},
			},
			wantTitle:    "Untitled Task",
			wantDesc:     "",
			wantWarnings: []string{"no title found"},
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

			if len(tt.wantWarnings) > 0 {
				if len(warnings) == 0 {
					t.Errorf("Expected warnings but got none")
				}
				for _, wantWarn := range tt.wantWarnings {
					found := false
					for _, warn := range warnings {
						if containsSubstring(warn, wantWarn) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected warning containing %q, got: %v", wantWarn, warnings)
					}
				}
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		(strings.Contains(strings.ToLower(s), strings.ToLower(substr)))
}
