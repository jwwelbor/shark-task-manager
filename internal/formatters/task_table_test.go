package formatters

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/assert"
)

func TestDefaultTaskTableConfig(t *testing.T) {
	config := DefaultTaskTableConfig()

	assert.True(t, config.ShowKey)
	assert.True(t, config.ShowTitle)
	assert.True(t, config.ShowStatus)
	assert.True(t, config.ShowPriority)
	assert.True(t, config.ShowAgentType)
	assert.True(t, config.ShowExecutionOrder)
	assert.True(t, config.ShowRejections)
	assert.Equal(t, 40, config.TitleMaxLength)
	assert.True(t, config.ColorEnabled)
	assert.True(t, config.UseHeader)
	assert.False(t, config.UsePterm)
}

func TestFeatureGetTaskTableConfig(t *testing.T) {
	config := FeatureGetTaskTableConfig()

	assert.True(t, config.ShowKey)
	assert.True(t, config.ShowTitle)
	assert.True(t, config.ShowStatus)
	assert.True(t, config.ShowPriority)
	assert.True(t, config.ShowAgentType)
	assert.False(t, config.ShowExecutionOrder)
	assert.False(t, config.ShowRejections)
	assert.Equal(t, 60, config.TitleMaxLength)
	assert.True(t, config.ColorEnabled)
	assert.True(t, config.UseHeader)
	assert.True(t, config.UsePterm)
}

func TestFormatTaskTable_EmptyTasks(t *testing.T) {
	tasks := []*models.Task{}
	config := DefaultTaskTableConfig()

	result := FormatTaskTable(tasks, nil, config)

	assert.NotNil(t, result.Headers)
	assert.Len(t, result.Rows, 0)
	assert.Equal(t, []string{"Key", "Title", "Status", "Priority", "Agent Type", "Order"}, result.Headers)
}

func TestFormatTaskTable_SingleTask(t *testing.T) {
	agentType := "backend"
	execOrder := 1

	tasks := []*models.Task{
		{
			Key:            "E07-F01-001",
			Title:          "Test Task",
			Status:         models.TaskStatusTodo,
			Priority:       5,
			AgentType:      &agentType,
			ExecutionOrder: &execOrder,
			RejectionCount: 0,
		},
	}

	config := DefaultTaskTableConfig()
	config.ColorEnabled = false // Disable color for predictable testing

	result := FormatTaskTable(tasks, nil, config)

	assert.Len(t, result.Rows, 1)
	assert.Equal(t, []string{"E07-F01-001", "Test Task", "todo", "5", "backend", "1"}, result.Rows[0])
}

func TestFormatTaskTable_TitleTruncation(t *testing.T) {
	agentType := "backend"
	longTitle := "This is a very long task title that should be truncated to fit within the configured maximum length"

	tests := []struct {
		name           string
		titleMaxLength int
		expectedTitle  string
	}{
		{
			name:           "truncate at 40",
			titleMaxLength: 40,
			expectedTitle:  "This is a very long task title that s...",
		},
		{
			name:           "truncate at 60",
			titleMaxLength: 60,
			expectedTitle:  "This is a very long task title that should be truncated t...",
		},
		{
			name:           "no truncation needed",
			titleMaxLength: 200,
			expectedTitle:  longTitle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := []*models.Task{
				{
					Key:       "E07-F01-001",
					Title:     longTitle,
					Status:    models.TaskStatusTodo,
					Priority:  5,
					AgentType: &agentType,
				},
			}

			config := DefaultTaskTableConfig()
			config.TitleMaxLength = tt.titleMaxLength
			config.ColorEnabled = false

			result := FormatTaskTable(tasks, nil, config)

			assert.Len(t, result.Rows, 1)
			assert.Equal(t, tt.expectedTitle, result.Rows[0][1]) // Title is column 1
		})
	}
}

func TestFormatTaskTable_RejectionIndicator(t *testing.T) {
	agentType := "backend"

	tests := []struct {
		name           string
		rejectionCount int
		showRejections bool
		expectedKeyCol string
	}{
		{
			name:           "no rejections",
			rejectionCount: 0,
			showRejections: true,
			expectedKeyCol: "E07-F01-001",
		},
		{
			name:           "one rejection",
			rejectionCount: 1,
			showRejections: true,
			expectedKeyCol: "E07-F01-001 🔴×1",
		},
		{
			name:           "multiple rejections",
			rejectionCount: 3,
			showRejections: true,
			expectedKeyCol: "E07-F01-001 🔴×3",
		},
		{
			name:           "rejections disabled",
			rejectionCount: 3,
			showRejections: false,
			expectedKeyCol: "E07-F01-001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := []*models.Task{
				{
					Key:            "E07-F01-001",
					Title:          "Test Task",
					Status:         models.TaskStatusTodo,
					Priority:       5,
					AgentType:      &agentType,
					RejectionCount: tt.rejectionCount,
				},
			}

			config := DefaultTaskTableConfig()
			config.ShowRejections = tt.showRejections
			config.ColorEnabled = false

			result := FormatTaskTable(tasks, nil, config)

			assert.Len(t, result.Rows, 1)
			assert.Equal(t, tt.expectedKeyCol, result.Rows[0][0]) // Key is column 0
		})
	}
}

func TestFormatTaskTable_AgentType(t *testing.T) {
	backend := "backend"

	tests := []struct {
		name          string
		agentType     *string
		expectedAgent string
	}{
		{
			name:          "agent type set",
			agentType:     &backend,
			expectedAgent: "backend",
		},
		{
			name:          "agent type nil",
			agentType:     nil,
			expectedAgent: "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := []*models.Task{
				{
					Key:       "E07-F01-001",
					Title:     "Test Task",
					Status:    models.TaskStatusTodo,
					Priority:  5,
					AgentType: tt.agentType,
				},
			}

			config := DefaultTaskTableConfig()
			config.ColorEnabled = false

			result := FormatTaskTable(tasks, nil, config)

			assert.Len(t, result.Rows, 1)
			assert.Equal(t, tt.expectedAgent, result.Rows[0][4]) // AgentType is column 4
		})
	}
}

func TestFormatTaskTable_ExecutionOrder(t *testing.T) {
	agentType := "backend"
	order1 := 1

	tests := []struct {
		name           string
		executionOrder *int
		expectedOrder  string
	}{
		{
			name:           "execution order set",
			executionOrder: &order1,
			expectedOrder:  "1",
		},
		{
			name:           "execution order nil",
			executionOrder: nil,
			expectedOrder:  "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks := []*models.Task{
				{
					Key:            "E07-F01-001",
					Title:          "Test Task",
					Status:         models.TaskStatusTodo,
					Priority:       5,
					AgentType:      &agentType,
					ExecutionOrder: tt.executionOrder,
				},
			}

			config := DefaultTaskTableConfig()
			config.ColorEnabled = false

			result := FormatTaskTable(tasks, nil, config)

			assert.Len(t, result.Rows, 1)
			assert.Equal(t, tt.expectedOrder, result.Rows[0][5]) // Order is column 5
		})
	}
}

func TestFormatTaskTable_ColumnVisibility(t *testing.T) {
	agentType := "backend"
	execOrder := 1

	tasks := []*models.Task{
		{
			Key:            "E07-F01-001",
			Title:          "Test Task",
			Status:         models.TaskStatusTodo,
			Priority:       5,
			AgentType:      &agentType,
			ExecutionOrder: &execOrder,
		},
	}

	tests := []struct {
		name            string
		modifyConfig    func(*TaskTableConfig)
		expectedHeaders []string
		expectedRowLen  int
	}{
		{
			name: "all columns",
			modifyConfig: func(c *TaskTableConfig) {
				// Default has all columns
			},
			expectedHeaders: []string{"Key", "Title", "Status", "Priority", "Agent Type", "Order"},
			expectedRowLen:  6,
		},
		{
			name: "no execution order",
			modifyConfig: func(c *TaskTableConfig) {
				c.ShowExecutionOrder = false
			},
			expectedHeaders: []string{"Key", "Title", "Status", "Priority", "Agent Type"},
			expectedRowLen:  5,
		},
		{
			name: "no agent type",
			modifyConfig: func(c *TaskTableConfig) {
				c.ShowAgentType = false
			},
			expectedHeaders: []string{"Key", "Title", "Status", "Priority", "Order"},
			expectedRowLen:  5,
		},
		{
			name: "minimal columns",
			modifyConfig: func(c *TaskTableConfig) {
				c.ShowAgentType = false
				c.ShowExecutionOrder = false
			},
			expectedHeaders: []string{"Key", "Title", "Status", "Priority"},
			expectedRowLen:  4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultTaskTableConfig()
			config.ColorEnabled = false
			tt.modifyConfig(&config)

			result := FormatTaskTable(tasks, nil, config)

			assert.Equal(t, tt.expectedHeaders, result.Headers)
			assert.Len(t, result.Rows, 1)
			assert.Len(t, result.Rows[0], tt.expectedRowLen)
		})
	}
}

func TestFormatTaskTable_WithColorFormatting(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	config.ClearWorkflowCache()
	defer config.ClearWorkflowCache()

	// Create workflow service with default config
	workflowService := workflow.NewService(tempDir)

	agentType := "backend"
	tasks := []*models.Task{
		{
			Key:       "E07-F01-001",
			Title:     "Test Task",
			Status:    models.TaskStatusTodo,
			Priority:  5,
			AgentType: &agentType,
		},
	}

	config := DefaultTaskTableConfig()
	config.ColorEnabled = true

	result := FormatTaskTable(tasks, workflowService, config)

	assert.Len(t, result.Rows, 1)
	// When color is enabled, status should contain ANSI codes or remain as plain text
	// We can't assert exact ANSI codes without knowing the workflow config,
	// but we can verify it's not empty
	assert.NotEmpty(t, result.Rows[0][2]) // Status is column 2
}

func TestFormatTaskTable_MultipleTasks(t *testing.T) {
	backend := "backend"
	frontend := "frontend"
	order1 := 1
	order2 := 2

	tasks := []*models.Task{
		{
			Key:            "E07-F01-001",
			Title:          "Backend Task",
			Status:         models.TaskStatusTodo,
			Priority:       5,
			AgentType:      &backend,
			ExecutionOrder: &order1,
		},
		{
			Key:            "E07-F01-002",
			Title:          "Frontend Task",
			Status:         models.TaskStatusInProgress,
			Priority:       3,
			AgentType:      &frontend,
			ExecutionOrder: &order2,
		},
	}

	config := DefaultTaskTableConfig()
	config.ColorEnabled = false

	result := FormatTaskTable(tasks, nil, config)

	assert.Len(t, result.Rows, 2)
	assert.Equal(t, []string{"E07-F01-001", "Backend Task", "todo", "5", "backend", "1"}, result.Rows[0])
	assert.Equal(t, []string{"E07-F01-002", "Frontend Task", "in_progress", "3", "frontend", "2"}, result.Rows[1])
}

func TestBuildHeaders(t *testing.T) {
	tests := []struct {
		name            string
		config          TaskTableConfig
		expectedHeaders []string
	}{
		{
			name:            "all columns",
			config:          DefaultTaskTableConfig(),
			expectedHeaders: []string{"Key", "Title", "Status", "Priority", "Agent Type", "Order"},
		},
		{
			name: "feature get columns",
			config: TaskTableConfig{
				ShowKey:            true,
				ShowTitle:          true,
				ShowStatus:         true,
				ShowPriority:       true,
				ShowAgentType:      true,
				ShowExecutionOrder: false,
			},
			expectedHeaders: []string{"Key", "Title", "Status", "Priority", "Agent Type"},
		},
		{
			name: "minimal columns",
			config: TaskTableConfig{
				ShowKey:    true,
				ShowTitle:  true,
				ShowStatus: true,
			},
			expectedHeaders: []string{"Key", "Title", "Status"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := buildHeaders(tt.config)
			assert.Equal(t, tt.expectedHeaders, headers)
		})
	}
}

func TestFormatRejectionIndicator(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		expected string
	}{
		{
			name:     "single rejection",
			count:    1,
			expected: "🔴×1",
		},
		{
			name:     "multiple rejections",
			count:    5,
			expected: "🔴×5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRejectionIndicator(tt.count)
			assert.Equal(t, tt.expected, result)
		})
	}
}
