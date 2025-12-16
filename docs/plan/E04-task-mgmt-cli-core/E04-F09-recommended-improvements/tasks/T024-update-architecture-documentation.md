---
status: created
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F09-recommended-improvements
created: 2025-12-16
assigned_agent: general-purpose
dependencies: [T023-update-cli-use-config.md]
estimated_time: 2 hours
---

# Task: Update Architecture Documentation

## Goal

Update the architecture documentation to reflect all the improvements made: context support, repository interfaces, domain errors, and configuration management.

## Success Criteria

- [ ] Architecture diagrams updated to show domain layer
- [ ] Package structure documentation reflects new organization
- [ ] Interface-based design explained and documented
- [ ] Context usage patterns documented
- [ ] Domain error catalog created
- [ ] Configuration options documented
- [ ] Examples updated to use new patterns
- [ ] Migration notes added for future developers

## Implementation Guidance

### Overview

Update all architecture documentation files to reflect the new architecture patterns. This includes diagrams, package descriptions, design patterns, and examples.

### Key Requirements

- Update architecture diagrams to show `internal/domain/` package
- Document repository interface pattern
- Document context usage throughout system
- Document domain error handling
- Document configuration system
- Provide examples of new patterns

Reference: [PRD - Documentation](../01-feature-prd.md#documentation)

### Files to Create/Modify

**Architecture Documentation**:
- `docs/architecture/SYSTEM_DESIGN.md` - Update system architecture
- `docs/architecture/ARCHITECTURE_REVIEW.md` - Add "After" section showing improvements
- `docs/architecture/GO_BEST_PRACTICES.md` - Add patterns for new features
- `docs/architecture/PACKAGE_STRUCTURE.md` (if exists) - Update package organization

**New Documentation**:
- `docs/architecture/DOMAIN_LAYER.md` (optional) - Detailed domain layer docs
- `docs/architecture/ERROR_HANDLING.md` (optional) - Error handling patterns
- `docs/architecture/CONFIGURATION.md` (optional) - Configuration guide

### Documentation Updates

**1. System Architecture Diagram**:
Update to show domain layer as central abstraction:
```
┌─────────────────────────────────────────────┐
│         Presentation Layer                  │
│  ┌──────────────┐    ┌──────────────┐      │
│  │ HTTP Server  │    │  CLI Tool    │      │
│  │ (cmd/server) │    │  (cmd/pm)    │      │
│  └──────────────┘    └──────────────┘      │
│         │                    │              │
│         ├────────────────────┘              │
│         │   (uses interfaces)               │
└─────────┼─────────────────────────────────┘
          │
┌─────────▼─────────────────────────────────┐
│          Domain Layer (NEW)                │
│  ┌──────────────────────────────────┐     │
│  │  Repository Interfaces           │     │
│  │  - TaskRepository                │     │
│  │  - EpicRepository                │     │
│  │  - FeatureRepository             │     │
│  │  - TaskHistoryRepository         │     │
│  └──────────────────────────────────┘     │
│  ┌──────────────────────────────────┐     │
│  │  Domain Errors                   │     │
│  │  - ErrTaskNotFound               │     │
│  │  - ErrDuplicateKey               │     │
│  │  - ErrInvalidStatus              │     │
│  └──────────────────────────────────┘     │
└────────────────┬──────────────────────────┘
                 │
┌────────────────▼──────────────────────────┐
│         Data Layer                         │
│  ┌──────────────┐    ┌──────────────┐    │
│  │    SQLite    │    │     Mock     │    │
│  │ (repository/ │    │ (repository/ │    │
│  │   sqlite/)   │    │    mock/)    │    │
│  └──────────────┘    └──────────────┘    │
└────────────────────────────────────────────┘
```

**2. Context Usage Patterns**:
Document how context flows through the system:
```go
// HTTP Handler Pattern
func handleGetTask(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()  // Use request context
    task, err := taskRepo.GetByID(ctx, taskID)
    // ...
}

// CLI Command Pattern
func runTaskList(cmd *cobra.Command, args []string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    tasks, err := taskRepo.List(ctx)
    // ...
}
```

**3. Repository Interface Pattern**:
Document the interface-based design:
```go
// Interface definition (domain package)
type TaskRepository interface {
    GetByID(ctx context.Context, id int64) (*models.Task, error)
    // ... other methods
}

// SQLite implementation
type taskRepository struct {
    db Database
}

func NewTaskRepository(db Database) domain.TaskRepository {
    return &taskRepository{db: db}
}

// Usage in main
taskRepo := sqlite.NewTaskRepository(db)  // Returns interface

// Testing with mock
mockRepo := mock.NewTaskRepository()
```

**4. Error Handling Pattern**:
Document domain error usage:
```go
// Repository returns domain error
task, err := repo.GetByID(ctx, id)
if errors.Is(err, domain.ErrTaskNotFound) {
    // Handle not found specifically
    fmt.Fprintf(os.Stderr, "Task not found: %s\n", taskKey)
    fmt.Fprintf(os.Stderr, "Use 'pm task list' to see available tasks.\n")
    return nil
}
if err != nil {
    // Handle other errors
    return fmt.Errorf("failed to get task: %w", err)
}
```

**5. Configuration Documentation**:
Document config system:
- Environment variables
- Config file format
- Precedence order
- Default values

### Integration Points

- **All Phases**: Documentation reflects changes from T001-T023
- **Examples**: Code examples use new patterns
- **Diagrams**: Visual representation of architecture
- **Future Reference**: Helps future developers understand system

## Validation Gates

**Content Review**:
- All architecture diagrams updated
- All code examples use new patterns
- No outdated information remains
- Documentation is clear and accurate

**Completeness Check**:
- Context usage documented
- Repository interfaces documented
- Domain errors documented
- Configuration documented
- Migration notes included

**Accuracy Verification**:
- Code examples compile and work
- Diagrams match actual architecture
- Package structure matches codebase

**Readability**:
- Documentation is well-organized
- Examples are clear
- Diagrams are easy to understand

## Context & Resources

- **PRD**: [Documentation Section](../01-feature-prd.md#documentation)
- **Current Docs**: `docs/architecture/` directory
- **All Previous Tasks**: T001-T023 (document changes from these)
- **Mermaid**: For diagrams (if used)

## Notes for Agent

- Review all previous tasks (T001-T023) to understand what changed
- Update existing docs, don't just add new ones
- Keep documentation concise but complete
- Use code examples to illustrate patterns
- Add "Before/After" sections to show improvements
- Include migration notes for future developers
- Document **why** decisions were made, not just **what** changed
- Reference specific files and line numbers where helpful
- This task documents the improved architecture for future reference
- Good documentation prevents architectural drift
