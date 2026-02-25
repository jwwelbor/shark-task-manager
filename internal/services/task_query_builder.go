package services

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TaskQueryBuilder provides a fluent interface for building complex task queries.
// It supports method chaining for composable filters, sorting, and pagination.
//
// Example usage:
//
//	tasks, total, err := svc.Query().
//	    WithStatus("todo").
//	    WithAgent("backend").
//	    WithPriorityRange(5, 10).
//	    SortBy("priority", "DESC").
//	    Paginate(0, 20).
//	    Build(ctx)
type TaskQueryBuilder struct {
	service    *TaskService
	filters    TaskFilters
	sortFields []SortField
}

// SortField defines a field and order for sorting.
type SortField struct {
	Field string // Field name: "key", "title", "priority", "execution_order", "created_at", "updated_at"
	Order string // "ASC" or "DESC"
}

// Query creates a new TaskQueryBuilder for composable task queries.
//
// Returns:
//   - *TaskQueryBuilder: builder instance for method chaining
func (s *TaskService) Query() *TaskQueryBuilder {
	return &TaskQueryBuilder{
		service:    s,
		filters:    TaskFilters{},
		sortFields: []SortField{},
	}
}

// WithStatus filters tasks by status.
//
// Parameters:
//   - status: status value to filter by
//
// Returns:
//   - *TaskQueryBuilder: builder for method chaining
func (b *TaskQueryBuilder) WithStatus(status string) *TaskQueryBuilder {
	b.filters.Status = status
	return b
}

// WithAgent filters tasks by agent type.
//
// Parameters:
//   - agentType: agent type to filter by
//
// Returns:
//   - *TaskQueryBuilder: builder for method chaining
func (b *TaskQueryBuilder) WithAgent(agentType string) *TaskQueryBuilder {
	b.filters.AgentType = agentType
	return b
}

// WithPriorityRange filters tasks by priority range (inclusive).
//
// Parameters:
//   - min: minimum priority (1-10)
//   - max: maximum priority (1-10)
//
// Returns:
//   - *TaskQueryBuilder: builder for method chaining
func (b *TaskQueryBuilder) WithPriorityRange(min, max int) *TaskQueryBuilder {
	b.filters.MinPriority = min
	b.filters.MaxPriority = max
	return b
}

// WithTitleSearch filters tasks by title substring (case-insensitive).
//
// Parameters:
//   - search: substring to search for in task titles
//
// Returns:
//   - *TaskQueryBuilder: builder for method chaining
func (b *TaskQueryBuilder) WithTitleSearch(search string) *TaskQueryBuilder {
	b.filters.TitleSearch = search
	return b
}

// ShowAll includes completed tasks in results (default: exclude completed).
//
// Returns:
//   - *TaskQueryBuilder: builder for method chaining
func (b *TaskQueryBuilder) ShowAll() *TaskQueryBuilder {
	b.filters.ShowAll = true
	return b
}

// OnlyBlocked filters to only blocked tasks.
//
// Returns:
//   - *TaskQueryBuilder: builder for method chaining
func (b *TaskQueryBuilder) OnlyBlocked() *TaskQueryBuilder {
	b.filters.Blocked = true
	return b
}

// SortBy adds a sort field to the query.
// Multiple calls create multi-field sorting (first call = primary sort, etc.).
//
// Parameters:
//   - field: field name ("key", "title", "priority", "execution_order", "created_at", "updated_at")
//   - order: "ASC" or "DESC"
//
// Returns:
//   - *TaskQueryBuilder: builder for method chaining
func (b *TaskQueryBuilder) SortBy(field, order string) *TaskQueryBuilder {
	b.sortFields = append(b.sortFields, SortField{
		Field: field,
		Order: strings.ToUpper(order),
	})
	return b
}

// Paginate sets limit and offset for pagination.
//
// Parameters:
//   - offset: number of results to skip
//   - limit: maximum number of results to return
//
// Returns:
//   - *TaskQueryBuilder: builder for method chaining
func (b *TaskQueryBuilder) Paginate(offset, limit int) *TaskQueryBuilder {
	b.filters.Offset = offset
	b.filters.Limit = limit
	return b
}

// Build executes the query and returns filtered, sorted, and paginated results.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//
// Returns:
//   - []*models.Task: filtered and sorted tasks
//   - int: total count before pagination (for UI pagination controls)
//   - error: repository errors or validation errors
//
// Errors:
//   - ValidationError: invalid filter values (e.g., priority out of range)
//   - RepositoryError: database query failed
func (b *TaskQueryBuilder) Build(ctx context.Context) ([]*models.Task, int, error) {
	// Validate priority range
	if b.filters.MinPriority > 0 && (b.filters.MinPriority < 1 || b.filters.MinPriority > 10) {
		return nil, 0, fmt.Errorf("invalid priority range: min priority must be between 1 and 10")
	}
	if b.filters.MaxPriority > 0 && (b.filters.MaxPriority < 1 || b.filters.MaxPriority > 10) {
		return nil, 0, fmt.Errorf("invalid priority range: max priority must be between 1 and 10")
	}
	if b.filters.MinPriority > 0 && b.filters.MaxPriority > 0 && b.filters.MinPriority > b.filters.MaxPriority {
		return nil, 0, fmt.Errorf("invalid priority range: min (%d) cannot be greater than max (%d)", b.filters.MinPriority, b.filters.MaxPriority)
	}

	// Get base results using service's ListTasks
	tasks, err := b.service.ListTasks(ctx, b.filters)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to build query: %w", err)
	}

	// Apply custom sorting if specified
	if len(b.sortFields) > 0 {
		sortTasksMultiField(tasks, b.sortFields)
	}

	total := len(tasks)

	// Apply pagination
	if b.filters.Offset >= total {
		return []*models.Task{}, total, nil
	}

	start := b.filters.Offset
	end := total
	if b.filters.Limit > 0 {
		end = start + b.filters.Limit
		if end > total {
			end = total
		}
	}

	return tasks[start:end], total, nil
}

// sortTasksMultiField sorts tasks by multiple fields in order.
// First sort field is primary, second is secondary (for ties), etc.
func sortTasksMultiField(tasks []*models.Task, sortFields []SortField) {
	sort.Slice(tasks, func(i, j int) bool {
		for _, sf := range sortFields {
			cmp := compareTasksByField(tasks[i], tasks[j], sf.Field)
			if cmp != 0 {
				if sf.Order == "DESC" {
					return cmp > 0
				}
				return cmp < 0
			}
			// If equal, continue to next sort field
		}
		return false // All fields equal
	})
}

// compareTasksByField compares two tasks by a specific field.
// Returns: -1 if task1 < task2, 0 if equal, 1 if task1 > task2
func compareTasksByField(task1, task2 *models.Task, field string) int {
	switch field {
	case "key":
		return strings.Compare(task1.Key, task2.Key)
	case "title":
		return strings.Compare(task1.Title, task2.Title)
	case "priority":
		return task1.Priority - task2.Priority
	case "execution_order":
		order1 := 0
		if task1.ExecutionOrder != nil {
			order1 = *task1.ExecutionOrder
		}
		order2 := 0
		if task2.ExecutionOrder != nil {
			order2 = *task2.ExecutionOrder
		}
		return order1 - order2
	case "created_at":
		if task1.CreatedAt.Before(task2.CreatedAt) {
			return -1
		} else if task1.CreatedAt.After(task2.CreatedAt) {
			return 1
		}
		return 0
	case "updated_at":
		if task1.UpdatedAt.Before(task2.UpdatedAt) {
			return -1
		} else if task1.UpdatedAt.After(task2.UpdatedAt) {
			return 1
		}
		return 0
	default:
		return 0
	}
}
