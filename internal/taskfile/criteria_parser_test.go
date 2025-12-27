package taskfile

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestParseCriteria(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []CriterionItem
	}{
		{
			name:     "empty content",
			content:  "",
			expected: []CriterionItem{},
		},
		{
			name: "no criteria",
			content: `# Task Description

This is a task without checkboxes.

Some regular text.`,
			expected: []CriterionItem{},
		},
		{
			name: "single pending criterion",
			content: `# Acceptance Criteria

- [ ] Database schema updated`,
			expected: []CriterionItem{
				{
					Criterion: "Database schema updated",
					Status:    models.CriteriaStatusPending,
				},
			},
		},
		{
			name: "single complete criterion",
			content: `# Acceptance Criteria

- [x] Tests passing`,
			expected: []CriterionItem{
				{
					Criterion: "Tests passing",
					Status:    models.CriteriaStatusComplete,
				},
			},
		},
		{
			name: "uppercase X for complete",
			content: `# Acceptance Criteria

- [X] Feature deployed`,
			expected: []CriterionItem{
				{
					Criterion: "Feature deployed",
					Status:    models.CriteriaStatusComplete,
				},
			},
		},
		{
			name: "mixed pending and complete",
			content: `# Acceptance Criteria

- [ ] API endpoint created
- [x] Tests written
- [ ] Documentation updated
- [X] Code reviewed`,
			expected: []CriterionItem{
				{
					Criterion: "API endpoint created",
					Status:    models.CriteriaStatusPending,
				},
				{
					Criterion: "Tests written",
					Status:    models.CriteriaStatusComplete,
				},
				{
					Criterion: "Documentation updated",
					Status:    models.CriteriaStatusPending,
				},
				{
					Criterion: "Code reviewed",
					Status:    models.CriteriaStatusComplete,
				},
			},
		},
		{
			name: "criteria with extra whitespace",
			content: `# Acceptance Criteria

  - [ ]   Criterion with spaces
-   [x]  Another criterion  `,
			expected: []CriterionItem{
				{
					Criterion: "Criterion with spaces",
					Status:    models.CriteriaStatusPending,
				},
				{
					Criterion: "Another criterion",
					Status:    models.CriteriaStatusComplete,
				},
			},
		},
		{
			name: "ignore empty checkboxes",
			content: `# Acceptance Criteria

- [ ] Valid criterion
- [ ]
- [x]
- [x] Another valid one`,
			expected: []CriterionItem{
				{
					Criterion: "Valid criterion",
					Status:    models.CriteriaStatusPending,
				},
				{
					Criterion: "Another valid one",
					Status:    models.CriteriaStatusComplete,
				},
			},
		},
		{
			name: "criteria in different sections",
			content: `# Task Description

Some text here.

## Acceptance Criteria

- [ ] Criterion one
- [x] Criterion two

## Testing Requirements

- [ ] Unit tests
- [ ] Integration tests`,
			expected: []CriterionItem{
				{
					Criterion: "Criterion one",
					Status:    models.CriteriaStatusPending,
				},
				{
					Criterion: "Criterion two",
					Status:    models.CriteriaStatusComplete,
				},
				{
					Criterion: "Unit tests",
					Status:    models.CriteriaStatusPending,
				},
				{
					Criterion: "Integration tests",
					Status:    models.CriteriaStatusPending,
				},
			},
		},
		{
			name: "ignore non-checkbox list items",
			content: `# Task Description

Regular list:
- Item one
- Item two

Checkboxes:
- [ ] Real criterion
- [x] Another real one`,
			expected: []CriterionItem{
				{
					Criterion: "Real criterion",
					Status:    models.CriteriaStatusPending,
				},
				{
					Criterion: "Another real one",
					Status:    models.CriteriaStatusComplete,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCriteria(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}
