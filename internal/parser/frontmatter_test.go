package parser

import (
	"strings"
	"testing"
)

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		wantTitle     string
		wantDesc      string
		wantTaskKey   string
		wantStatus    string
		wantErr       bool
		wantHasFM     bool
	}{
		{
			name: "valid frontmatter with all fields",
			content: `---
task_key: T-E04-F02-001
title: Implement User Authentication
description: Add JWT-based authentication for the API
status: todo
assigned_agent: backend
priority: 1
---

# Task Body

This is the task body.`,
			wantTitle:   "Implement User Authentication",
			wantDesc:    "Add JWT-based authentication for the API",
			wantTaskKey: "T-E04-F02-001",
			wantStatus:  "todo",
			wantErr:     false,
			wantHasFM:   true,
		},
		{
			name: "valid frontmatter with minimal fields",
			content: `---
task_key: T-E04-F02-002
---

# Task`,
			wantTitle:   "",
			wantDesc:    "",
			wantTaskKey: "T-E04-F02-002",
			wantStatus:  "",
			wantErr:     false,
			wantHasFM:   true,
		},
		{
			name: "no frontmatter",
			content: `# Task Title

This is a task without frontmatter.`,
			wantTitle:   "",
			wantDesc:    "",
			wantTaskKey: "",
			wantStatus:  "",
			wantErr:     false,
			wantHasFM:   false,
		},
		{
			name: "invalid YAML syntax - unclosed quote",
			content: `---
task_key: T-E04-F02-001
title: "Unclosed quote
description: Test
---`,
			wantErr:   true,
			wantHasFM: true,
		},
		{
			name: "frontmatter with multiline description",
			content: `---
task_key: T-E04-F02-003
title: Complex Task
description: |
  This is a multiline description
  that spans multiple lines.
status: in_progress
---`,
			wantTitle:   "Complex Task",
			wantDesc:    "This is a multiline description\nthat spans multiple lines.\n",
			wantTaskKey: "T-E04-F02-003",
			wantStatus:  "in_progress",
			wantErr:     false,
			wantHasFM:   true,
		},
		{
			name: "empty frontmatter",
			content: `---
---

# Task`,
			wantTitle:   "",
			wantDesc:    "",
			wantTaskKey: "",
			wantStatus:  "",
			wantErr:     false,
			wantHasFM:   true,
		},
		{
			name: "frontmatter with extra fields",
			content: `---
task_key: T-E04-F02-004
title: Extra Fields Task
custom_field: some value
another_field: 123
---`,
			wantTitle:   "Extra Fields Task",
			wantDesc:    "",
			wantTaskKey: "T-E04-F02-004",
			wantStatus:  "",
			wantErr:     false,
			wantHasFM:   true,
		},
		{
			name: "only opening frontmatter delimiter",
			content: `---
task_key: T-E04-F02-005
title: Incomplete`,
			wantErr:   true,
			wantHasFM: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, err := ParseFrontmatter(tt.content)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseFrontmatter() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseFrontmatter() unexpected error: %v", err)
				return
			}

			if fm.HasFrontmatter != tt.wantHasFM {
				t.Errorf("HasFrontmatter = %v, want %v", fm.HasFrontmatter, tt.wantHasFM)
			}

			if !tt.wantHasFM {
				return // No frontmatter, skip field checks
			}

			if fm.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", fm.Title, tt.wantTitle)
			}

			if fm.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", fm.Description, tt.wantDesc)
			}

			if fm.TaskKey != tt.wantTaskKey {
				t.Errorf("TaskKey = %q, want %q", fm.TaskKey, tt.wantTaskKey)
			}

			if fm.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", fm.Status, tt.wantStatus)
			}
		})
	}
}

func TestGetContentAfterFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "content after frontmatter",
			content: `---
task_key: T-E04-F02-001
---

# Task Body

This is the content.`,
			want: "\n# Task Body\n\nThis is the content.",
		},
		{
			name: "no frontmatter",
			content: `# Task Body

This is the content.`,
			want: "# Task Body\n\nThis is the content.",
		},
		{
			name: "frontmatter at end of file",
			content: `---
task_key: T-E04-F02-001
---`,
			want: "",
		},
		{
			name: "content immediately after delimiter",
			content: `---
task_key: T-E04-F02-001
---
Immediate content`,
			want: "Immediate content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, err := ParseFrontmatter(tt.content)
			if err != nil {
				t.Fatalf("ParseFrontmatter() error: %v", err)
			}

			got := GetContentAfterFrontmatter(tt.content, fm)
			if got != tt.want {
				t.Errorf("GetContentAfterFrontmatter() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFrontmatterFieldTypes(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantPriority int
		wantFeature  string
	}{
		{
			name: "integer priority",
			content: `---
task_key: T-E04-F02-001
priority: 1
---`,
			wantPriority: 1,
		},
		{
			name: "string feature path",
			content: `---
task_key: T-E04-F02-001
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F02-cli-infrastructure
---`,
			wantFeature: "/docs/plan/E04-task-mgmt-cli-core/E04-F02-cli-infrastructure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, err := ParseFrontmatter(tt.content)
			if err != nil {
				t.Fatalf("ParseFrontmatter() error: %v", err)
			}

			if tt.wantPriority != 0 && fm.Priority != tt.wantPriority {
				t.Errorf("Priority = %d, want %d", fm.Priority, tt.wantPriority)
			}

			if tt.wantFeature != "" && fm.Feature != tt.wantFeature {
				t.Errorf("Feature = %q, want %q", fm.Feature, tt.wantFeature)
			}
		})
	}
}

func TestUpdateFrontmatterField(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		fieldName string
		value     string
		want      string
		wantErr   bool
	}{
		{
			name: "add task_key to existing frontmatter",
			content: `---
title: Test Task
---

# Body`,
			fieldName: "task_key",
			value:     "T-E04-F02-001",
			want: `---
title: Test Task
task_key: T-E04-F02-001
---

# Body`,
		},
		{
			name: "update existing task_key",
			content: `---
task_key: T-E04-F02-001
title: Test Task
---

# Body`,
			fieldName: "task_key",
			value:     "T-E04-F02-002",
			want: `---
task_key: T-E04-F02-002
title: Test Task
---

# Body`,
		},
		{
			name: "create frontmatter when missing",
			content: `# Task Body

Content here.`,
			fieldName: "task_key",
			value:     "T-E04-F02-003",
			want: `---
task_key: T-E04-F02-003
---

# Task Body

Content here.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UpdateFrontmatterField(tt.content, tt.fieldName, tt.value)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateFrontmatterField() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("UpdateFrontmatterField() unexpected error: %v", err)
				return
			}

			// Verify by parsing the result
			fm, err := ParseFrontmatter(got)
			if err != nil {
				t.Errorf("Failed to parse updated frontmatter: %v", err)
				return
			}

			// Check that the field was set correctly
			switch tt.fieldName {
			case "task_key":
				if fm.TaskKey != tt.value {
					t.Errorf("TaskKey = %q, want %q", fm.TaskKey, tt.value)
				}
			}

			// Verify body content is preserved
			bodyGot := GetContentAfterFrontmatter(got, fm)
			bodyWant := GetContentAfterFrontmatter(tt.want, &Frontmatter{HasFrontmatter: true})

			if strings.TrimSpace(bodyGot) != strings.TrimSpace(bodyWant) {
				t.Errorf("Body content changed:\ngot: %q\nwant: %q", bodyGot, bodyWant)
			}
		})
	}
}
