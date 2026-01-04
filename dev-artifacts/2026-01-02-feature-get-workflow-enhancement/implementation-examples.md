# Implementation Examples: Feature Get Workflow Enhancement

**Reference:** See `architectural-review-and-design.md` for full design details

---

## Example 1: WorkflowService Implementation

### File: `internal/workflow/service.go`

```go
package workflow

import (
    "path/filepath"
    "sort"

    "github.com/jwwelbor/shark-task-manager/internal/config"
    "github.com/jwwelbor/shark-task-manager/internal/models"
)

// Service provides workflow configuration and status management.
// Centralizes workflow config access across all CLI commands.
type Service struct {
    projectRoot string
    workflow    *config.WorkflowConfig
}

// NewService creates a new workflow service for the given project root.
// Loads workflow config from .sharkconfig.json (or uses default if not found).
func NewService(projectRoot string) *Service {
    configPath := filepath.Join(projectRoot, config.DefaultConfigFilename)
    workflow := config.GetWorkflowOrDefault(configPath)

    return &Service{
        projectRoot: projectRoot,
        workflow:    workflow,
    }
}

// GetWorkflow returns the loaded workflow configuration.
func (s *Service) GetWorkflow() *config.WorkflowConfig {
    return s.workflow
}

// GetInitialStatus returns the first entry status from workflow config.
// Falls back to "draft" if workflow config not found or _start_ not defined.
func (s *Service) GetInitialStatus() models.TaskStatus {
    if s.workflow == nil {
        return models.TaskStatus("draft")
    }

    startStatuses, exists := s.workflow.SpecialStatuses[config.StartStatusKey]
    if !exists || len(startStatuses) == 0 {
        return models.TaskStatus("draft")
    }

    return models.TaskStatus(startStatuses[0])
}

// GetAllStatuses returns all statuses defined in workflow config, ordered by phase.
// Order: planning → development → review → qa → approval → done
func (s *Service) GetAllStatuses() []string {
    if s.workflow == nil || s.workflow.StatusFlow == nil {
        return []string{"draft", "ready_for_development", "in_development", "ready_for_qa", "completed"}
    }

    // Group statuses by phase
    phases := []string{"planning", "development", "review", "qa", "approval", "done", "any"}
    statusesByPhase := make(map[string][]string)

    for status := range s.workflow.StatusFlow {
        meta, found := s.workflow.GetStatusMetadata(status)
        phase := "other"
        if found && meta.Phase != "" {
            phase = meta.Phase
        }

        statusesByPhase[phase] = append(statusesByPhase[phase], status)
    }

    // Sort statuses within each phase alphabetically
    for _, statuses := range statusesByPhase {
        sort.Strings(statuses)
    }

    // Concatenate phases in order
    var result []string
    for _, phase := range phases {
        result = append(result, statusesByPhase[phase]...)
    }

    // Add any remaining statuses not in standard phases
    if otherStatuses, exists := statusesByPhase["other"]; exists {
        result = append(result, otherStatuses...)
    }

    return result
}

// GetStatusMetadata returns metadata for a given status.
// Returns empty metadata if status not found.
func (s *Service) GetStatusMetadata(status string) config.StatusMetadata {
    if s.workflow == nil {
        return config.StatusMetadata{}
    }

    meta, _ := s.workflow.GetStatusMetadata(status)
    return meta
}

// GetStatusesByPhase returns all statuses in the given phase.
func (s *Service) GetStatusesByPhase(phase string) []string {
    if s.workflow == nil {
        return []string{}
    }

    return s.workflow.GetStatusesByPhase(phase)
}

// FormatStatusForDisplay returns a formatted status with color and metadata.
func (s *Service) FormatStatusForDisplay(status string, colorEnabled bool) FormattedStatus {
    meta := s.GetStatusMetadata(status)

    formatted := FormattedStatus{
        Status:      status,
        Description: meta.Description,
        Phase:       meta.Phase,
        ColorName:   meta.Color,
    }

    if colorEnabled && meta.Color != "" {
        formatted.Colored = colorizeStatus(status, meta.Color)
    } else {
        formatted.Colored = status
    }

    return formatted
}

// FormattedStatus represents a status formatted for display
type FormattedStatus struct {
    Status      string // Raw status (e.g., "in_progress")
    Colored     string // With ANSI codes (e.g., "\033[33min_progress\033[0m")
    Description string // Human-readable (e.g., "Code implementation in progress")
    Phase       string // Workflow phase (e.g., "development")
    ColorName   string // Color name (e.g., "yellow")
}

// colorizeStatus applies ANSI color codes to a status string
func colorizeStatus(status, colorName string) string {
    colorCodes := map[string]string{
        "red":     "\033[31m",
        "green":   "\033[32m",
        "yellow":  "\033[33m",
        "blue":    "\033[34m",
        "magenta": "\033[35m",
        "cyan":    "\033[36m",
        "white":   "\033[37m",
        "gray":    "\033[90m",
        "orange":  "\033[38;5;208m",
        "purple":  "\033[38;5;141m",
    }

    reset := "\033[0m"
    colorCode, found := colorCodes[colorName]
    if !found {
        return status
    }

    return colorCode + status + reset
}
```

---

## Example 2: Enhanced TaskRepository.GetStatusBreakdown

### File: `internal/repository/task_repository.go`

```go
// StatusCount represents a status and its count with workflow metadata
type StatusCount struct {
    Status      models.TaskStatus
    Count       int
    Phase       string
    Description string
}

// GetStatusBreakdown returns a count of tasks by status for a feature.
// Now includes ALL workflow-defined statuses (with zero counts) in workflow order.
func (r *TaskRepository) GetStatusBreakdown(ctx context.Context, featureID int64) ([]StatusCount, error) {
    // Query actual task counts
    query := `
        SELECT status, COUNT(*) as count
        FROM tasks
        WHERE feature_id = ?
        GROUP BY status
    `

    rows, err := r.db.QueryContext(ctx, query, featureID)
    if err != nil {
        return nil, fmt.Errorf("failed to get status breakdown: %w", err)
    }
    defer rows.Close()

    // Build map of actual counts
    actualCounts := make(map[string]int)
    for rows.Next() {
        var status string
        var count int
        if err := rows.Scan(&status, &count); err != nil {
            return nil, fmt.Errorf("failed to scan status breakdown: %w", err)
        }
        actualCounts[status] = count
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating status breakdown: %w", err)
    }

    // Get all statuses from workflow config (in order)
    allStatuses := r.getOrderedStatuses()

    // Build result with all statuses (including zero counts)
    var result []StatusCount
    for _, status := range allStatuses {
        count := actualCounts[status]
        meta := r.getStatusMetadata(status)

        result = append(result, StatusCount{
            Status:      models.TaskStatus(status),
            Count:       count,
            Phase:       meta.Phase,
            Description: meta.Description,
        })
    }

    return result, nil
}

// getOrderedStatuses returns all statuses in workflow order
func (r *TaskRepository) getOrderedStatuses() []string {
    if r.workflow == nil {
        // Fallback to default statuses
        return []string{"draft", "ready_for_refinement", "in_refinement", "ready_for_development",
            "in_development", "ready_for_code_review", "in_code_review", "ready_for_qa",
            "in_qa", "ready_for_approval", "in_approval", "completed", "cancelled", "blocked"}
    }

    // Group by phase and sort
    phases := []string{"planning", "development", "review", "qa", "approval", "done", "any"}
    statusesByPhase := make(map[string][]string)

    for status := range r.workflow.StatusFlow {
        meta, found := r.workflow.GetStatusMetadata(status)
        phase := "other"
        if found && meta.Phase != "" {
            phase = meta.Phase
        }
        statusesByPhase[phase] = append(statusesByPhase[phase], status)
    }

    // Sort within each phase
    for _, statuses := range statusesByPhase {
        sort.Strings(statuses)
    }

    // Concatenate in phase order
    var result []string
    for _, phase := range phases {
        result = append(result, statusesByPhase[phase]...)
    }
    if otherStatuses := statusesByPhase["other"]; len(otherStatuses) > 0 {
        result = append(result, otherStatuses...)
    }

    return result
}

// getStatusMetadata returns metadata for a status
func (r *TaskRepository) getStatusMetadata(status string) config.StatusMetadata {
    if r.workflow == nil {
        return config.StatusMetadata{}
    }
    meta, _ := r.workflow.GetStatusMetadata(status)
    return meta
}
```

---

## Example 3: Enhanced Feature Get Command

### File: `internal/cli/commands/feature.go`

```go
// Update renderFeatureDetails signature
func renderFeatureDetails(
    feature *models.Feature,
    tasks []*models.Task,
    statusBreakdown []repository.StatusCount,  // Changed from map to slice
    workflowService *workflow.Service,         // Added
    path, filename string,
    relatedDocs []*models.Document,
) {
    // Print feature metadata (unchanged)
    pterm.DefaultSection.Printf("Feature: %s", feature.Key)
    fmt.Println()

    // ... feature info rendering (unchanged) ...

    // Task status breakdown (ENHANCED)
    if len(statusBreakdown) > 0 {
        pterm.DefaultSection.Println("Task Status Breakdown")
        fmt.Println()

        // Build table with headers
        tableData := pterm.TableData{
            {"Status", "Count", "Phase", "Description"},
        }

        // Add rows (already in workflow order from repository)
        colorEnabled := !cli.GlobalConfig.NoColor
        for _, sc := range statusBreakdown {
            // Format status with color if enabled
            statusDisplay := string(sc.Status)
            if colorEnabled {
                formatted := workflowService.FormatStatusForDisplay(string(sc.Status), true)
                statusDisplay = formatted.Colored
            }

            // Truncate long descriptions
            description := sc.Description
            if len(description) > 50 {
                description = description[:47] + "..."
            }

            tableData = append(tableData, []string{
                statusDisplay,
                fmt.Sprintf("%d", sc.Count),
                sc.Phase,
                description,
            })
        }

        _ = pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
        fmt.Println()
    }

    // Optional: Group tasks by phase
    if len(tasks) > 0 {
        renderTasksByPhase(tasks, workflowService)
    }
}

// renderTasksByPhase groups and displays tasks by workflow phase
func renderTasksByPhase(tasks []*models.Task, workflowService *workflow.Service) {
    // Group tasks by phase
    tasksByPhase := make(map[string][]*models.Task)
    for _, task := range tasks {
        meta := workflowService.GetStatusMetadata(string(task.Status))
        phase := meta.Phase
        if phase == "" {
            phase = "other"
        }
        tasksByPhase[phase] = append(tasksByPhase[phase], task)
    }

    // Render each phase
    phases := []string{"planning", "development", "review", "qa", "approval", "done"}
    for _, phase := range phases {
        phaseTasks := tasksByPhase[phase]
        if len(phaseTasks) == 0 {
            continue
        }

        // Phase header
        phaseName := strings.Title(phase)
        pterm.DefaultSection.Printf("%s (%d tasks)", phaseName, len(phaseTasks))
        fmt.Println()

        // Task table
        tableData := pterm.TableData{
            {"Key", "Title", "Status", "Priority", "Agent"},
        }

        for _, task := range phaseTasks {
            title := task.Title
            if len(title) > 30 {
                title = title[:27] + "..."
            }

            agent := "none"
            if task.AgentType != nil {
                agent = string(*task.AgentType)
            }

            tableData = append(tableData, []string{
                task.Key,
                title,
                string(task.Status),
                fmt.Sprintf("%d", task.Priority),
                agent,
            })
        }

        _ = pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
        fmt.Println()
    }
}
```

---

## Example 4: Update runFeatureGet to Use WorkflowService

### File: `internal/cli/commands/feature.go`

```go
func runFeatureGet(cmd *cobra.Command, args []string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    featureKey := args[0]

    // Get database connection
    dbPath, err := cli.GetDBPath()
    if err != nil {
        cli.Error(fmt.Sprintf("Error: Failed to get database path: %v", err))
        return fmt.Errorf("database path error")
    }

    database, err := db.InitDB(dbPath)
    if err != nil {
        cli.Error("Error: Database error. Run with --verbose for details.")
        if cli.GlobalConfig.Verbose {
            fmt.Fprintf(os.Stderr, "Database error: %v\n", err)
        }
        os.Exit(2)
    }

    // Get project root
    projectRoot, err := os.Getwd()
    if err != nil {
        cli.Error(fmt.Sprintf("Error: Failed to get working directory: %v", err))
        os.Exit(1)
    }

    // Create workflow service (NEW)
    workflowService := workflow.NewService(projectRoot)

    // Get repositories
    repoDb := repository.NewDB(database)

    // Create task repository with workflow config (UPDATED)
    taskRepo := repository.NewTaskRepositoryWithWorkflow(repoDb, workflowService.GetWorkflow())

    featureRepo := repository.NewFeatureRepository(repoDb)
    epicRepo := repository.NewEpicRepository(repoDb)
    documentRepo := repository.NewDocumentRepository(repoDb)

    // ... (rest of feature get logic) ...

    // Get task status breakdown (now returns ordered slice)
    statusBreakdown, err := taskRepo.GetStatusBreakdown(ctx, feature.ID)
    if err != nil {
        cli.Error("Error: Database error. Run with --verbose for details.")
        if cli.GlobalConfig.Verbose {
            fmt.Fprintf(os.Stderr, "Failed to get status breakdown: %v\n", err)
        }
        os.Exit(2)
    }

    // ... (rest of rendering logic) ...

    // Render with workflow service (UPDATED)
    renderFeatureDetails(feature, tasks, statusBreakdown, workflowService, dirPath, filename, relatedDocs)
    return nil
}
```

---

## Example 5: Refactor Task Creation to Use WorkflowService

### File: `internal/taskcreation/creator.go`

```go
// Creator orchestrates the complete task creation workflow
type Creator struct {
    db              *repository.DB
    keygen          *KeyGenerator
    validator       *Validator
    renderer        *templates.Renderer
    taskRepo        *repository.TaskRepository
    historyRepo     *repository.TaskHistoryRepository
    epicRepo        *repository.EpicRepository
    featureRepo     *repository.FeatureRepository
    projectRoot     string
    workflowService *workflow.Service  // NEW
}

// NewCreator creates a new task creator
func NewCreator(
    db *repository.DB,
    keygen *KeyGenerator,
    validator *Validator,
    renderer *templates.Renderer,
    taskRepo *repository.TaskRepository,
    historyRepo *repository.TaskHistoryRepository,
    epicRepo *repository.EpicRepository,
    featureRepo *repository.FeatureRepository,
    projectRoot string,
    workflowService *workflow.Service,  // NEW
) *Creator {
    return &Creator{
        db:              db,
        keygen:          keygen,
        validator:       validator,
        renderer:        renderer,
        taskRepo:        taskRepo,
        historyRepo:     historyRepo,
        epicRepo:        epicRepo,
        featureRepo:     featureRepo,
        projectRoot:     projectRoot,
        workflowService: workflowService,  // NEW
    }
}

// CreateTask orchestrates the complete task creation workflow
func (c *Creator) CreateTask(ctx context.Context, input CreateTaskInput) (*CreateTaskResult, error) {
    // ... (validation and key generation logic) ...

    // Determine initial status from workflow config (SIMPLIFIED)
    initialStatus := c.workflowService.GetInitialStatus()

    // Create task record
    task := &models.Task{
        FeatureID:      validated.FeatureID,
        Key:            key,
        Title:          input.Title,
        Description:    description,
        Status:         initialStatus,  // Use workflow-defined status
        AgentType:      &validated.AgentType,
        Priority:       input.Priority,
        DependsOn:      dependsOnJSON,
        FilePath:       &filePath,
        ExecutionOrder: executionOrder,
        CreatedAt:      now,
        UpdatedAt:      now,
    }

    // ... (rest of creation logic) ...
}

// REMOVED: getInitialTaskStatus() method (no longer needed)
```

---

## Example 6: JSON Output with Metadata

### Enhanced JSON Response

```json
{
  "id": 123,
  "key": "E04-F06",
  "title": "Advanced Task Metadata Filtering",
  "status": "active",
  "status_breakdown": [
    {
      "status": "draft",
      "count": 5,
      "phase": "planning",
      "description": "Task created but not yet refined"
    },
    {
      "status": "ready_for_refinement",
      "count": 0,
      "phase": "planning",
      "description": "Awaiting specification and analysis"
    },
    {
      "status": "ready_for_development",
      "count": 2,
      "phase": "development",
      "description": "Spec complete, ready for implementation"
    },
    {
      "status": "in_development",
      "count": 0,
      "phase": "development",
      "description": "Code implementation in progress"
    },
    {
      "status": "ready_for_qa",
      "count": 3,
      "phase": "qa",
      "description": "Ready for quality assurance testing"
    },
    {
      "status": "completed",
      "count": 0,
      "phase": "done",
      "description": "Task finished and approved"
    }
  ],
  "tasks": [
    {
      "key": "T-E04-F06-001",
      "title": "Setup metadata schema",
      "status": "draft",
      "status_metadata": {
        "phase": "planning",
        "description": "Task created but not yet refined",
        "color": "gray"
      }
    }
  ]
}
```

---

## Example 7: Unit Test for WorkflowService

### File: `internal/workflow/service_test.go`

```go
package workflow

import (
    "testing"

    "github.com/jwwelbor/shark-task-manager/internal/config"
    "github.com/stretchr/testify/assert"
)

func TestWorkflowService_GetInitialStatus(t *testing.T) {
    tests := []struct {
        name           string
        workflow       *config.WorkflowConfig
        expectedStatus string
    }{
        {
            name: "uses first start status from workflow",
            workflow: &config.WorkflowConfig{
                SpecialStatuses: map[string][]string{
                    "_start_": {"draft", "backlog"},
                },
            },
            expectedStatus: "draft",
        },
        {
            name: "falls back to draft when no start statuses",
            workflow: &config.WorkflowConfig{
                SpecialStatuses: map[string][]string{},
            },
            expectedStatus: "draft",
        },
        {
            name:           "falls back to draft when workflow is nil",
            workflow:       nil,
            expectedStatus: "draft",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            service := &Service{
                workflow: tt.workflow,
            }

            status := service.GetInitialStatus()
            assert.Equal(t, tt.expectedStatus, string(status))
        })
    }
}

func TestWorkflowService_GetAllStatuses_Ordering(t *testing.T) {
    workflow := &config.WorkflowConfig{
        StatusFlow: map[string][]string{
            "draft":                   {"ready_for_refinement"},
            "ready_for_refinement":    {"in_refinement"},
            "in_refinement":           {"ready_for_development"},
            "ready_for_development":   {"in_development"},
            "in_development":          {"ready_for_code_review"},
            "ready_for_code_review":   {"in_code_review"},
            "in_code_review":          {"ready_for_qa"},
            "ready_for_qa":            {"in_qa"},
            "in_qa":                   {"completed"},
            "completed":               {},
        },
        StatusMetadata: map[string]config.StatusMetadata{
            "draft":                   {Phase: "planning"},
            "ready_for_refinement":    {Phase: "planning"},
            "in_refinement":           {Phase: "planning"},
            "ready_for_development":   {Phase: "development"},
            "in_development":          {Phase: "development"},
            "ready_for_code_review":   {Phase: "review"},
            "in_code_review":          {Phase: "review"},
            "ready_for_qa":            {Phase: "qa"},
            "in_qa":                   {Phase: "qa"},
            "completed":               {Phase: "done"},
        },
    }

    service := &Service{workflow: workflow}
    statuses := service.GetAllStatuses()

    // Verify statuses are grouped by phase
    planningStatuses := []string{"draft", "in_refinement", "ready_for_refinement"}
    developmentStatuses := []string{"in_development", "ready_for_development"}
    reviewStatuses := []string{"in_code_review", "ready_for_code_review"}
    qaStatuses := []string{"in_qa", "ready_for_qa"}
    doneStatuses := []string{"completed"}

    // Check that planning statuses come before development
    assert.True(t, indexOfAny(statuses, planningStatuses) < indexOfAny(statuses, developmentStatuses))

    // Check that development comes before review
    assert.True(t, indexOfAny(statuses, developmentStatuses) < indexOfAny(statuses, reviewStatuses))

    // Check that review comes before qa
    assert.True(t, indexOfAny(statuses, reviewStatuses) < indexOfAny(statuses, qaStatuses))

    // Check that qa comes before done
    assert.True(t, indexOfAny(statuses, qaStatuses) < indexOfAny(statuses, doneStatuses))
}

func indexOfAny(slice []string, targets []string) int {
    for i, s := range slice {
        for _, target := range targets {
            if s == target {
                return i
            }
        }
    }
    return -1
}
```

---

## Example 8: Config Constant

### File: `internal/config/config.go`

```go
package config

const (
    // DefaultConfigFilename is the standard name for Shark configuration files
    DefaultConfigFilename = ".sharkconfig.json"
)

// GetConfigPath returns the full path to the config file for a project
func GetConfigPath(projectRoot string) string {
    return filepath.Join(projectRoot, DefaultConfigFilename)
}
```

---

**Document Version:** 1.0
**Companion to:** architectural-review-and-design.md
