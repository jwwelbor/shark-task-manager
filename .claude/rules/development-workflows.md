# Development Workflows

## Task & Feature Creation Standards

### Creating Tasks for Development Work

**All development tasks MUST be created through shark** following this workflow:

1. **Create Feature** (if new feature area):
   ```bash
   # Positional syntax (recommended)
   ./bin/shark feature create E07 "Feature Title" --execution-order=1

   # Flag syntax (legacy, still supported)
   ./bin/shark feature create --epic=E07 --title="Feature Title" --execution-order=1
   ```

2. **Create Tasks** in the feature:
   ```bash
   # Positional syntax (recommended)
   ./bin/shark task create E07 F01 "Task Title" --priority=5
   # OR combined format
   ./bin/shark task create E07-F01 "Task Title" --priority=5

   # Flag syntax (legacy, still supported)
   ./bin/shark task create --epic=E07 --feature=F01 --title="Task Title" --priority=5
   ```

3. **Update task file** at `docs/plan/{epic}/{feature}/tasks/{task-key}.md`:
   - Add implementation details to task frontmatter
   - Include specification, acceptance criteria, test plan
   - Link related documents using `related-docs:` frontmatter field
   - Example:
     ```yaml
     ---
     task_key: T-E07-F06-001
     status: todo
     feature: /path/to/feature
     priority: 5
     dependencies: []
     related-docs:
       - path/to/design-doc.md
       - path/to/specification.md
     ---
     ```

4. **Generate related documentation** separately:
   - Design documents go in `docs/plan/{epic}/{feature}/`
   - Implementation guides go in `docs/plan/{epic}/{feature}/implementation/`
   - Link these in task `related-docs:` field

5. **DO NOT** create standalone documentation files unless they're referenced in shark tasks

## Task Status & Lifecycle

Tasks flow through these states:
```
todo → in_progress → ready_for_review → completed
                  ↘ blocked ↗
```

State descriptions:
- **todo**: Created but not started
- **in_progress**: Work has begun
- **ready_for_review**: Implementation complete, awaiting approval
- **completed**: Approved and merged
- **blocked**: Waiting on external dependency

Update status with:
```bash
# Short format (recommended)
./bin/shark task start E07-F20-001
./bin/shark task complete E07-F20-001
./bin/shark task approve E07-F20-001
./bin/shark task block E07-F20-001 --reason="..."
./bin/shark task unblock E07-F20-001

# Traditional format (still supported)
./bin/shark task start T-E07-F20-001
./bin/shark task complete T-E07-F20-001

# Case insensitive
./bin/shark task start e07-f20-001
```

## Development Workspace Structure

When working on development tasks, use the following workspace pattern:

```
dev-artifacts/{YYYY-MM-DD}-{task-name}/
├── analysis/        # Investigation and documentation
├── scripts/         # Verification and test scripts
├── verification/    # Test results and validation
└── shared/          # Reusable development utilities
```

**Date Formatting**: Extract the current date from system context at conversation start (format: YYYY-MM-DD). Use this date consistently for workspace naming.

**Example**: For a task starting on 2025-12-18 to fix a database bug:
```
dev-artifacts/2025-12-18-fix-database-bug/
├── analysis/DebugInfo-{timestamp}-bug-description.md
├── scripts/verify-fix.sh
├── verification/test-results.txt
└── shared/helper-functions.go
```

## Development Patterns

### Specifications or Planning
- **DO NOT** include development time estimates or estimated hours/weeks
- **DO include** task complexity sizing using t-shirt sizes (XS, S, M, L, XL, XXL) or story points (1, 2, 3, 5, 8, 13)
- Tasks rated L, XL, XXL, 5, 8, or 13 must be broken down into smaller chunks
- Use `/docs/PRP_WORKFLOW.md` as reference for development workflows

### Development Artifacts
- Store artifacts in workspace: `dev-artifacts/{YYYY-MM-DD}-{task-name}/`
- Script types:
  - **verification**: Quick tests to validate assumptions
  - **analysis**: Code inspection and pattern discovery
  - **debugging**: Troubleshooting and investigation tools
  - **prototyping**: Experimental implementations
- **Commit guidelines**: Commit only useful artifacts; delete experimental ones
- **Cleanup**: Remove task folders after completion unless valuable for reference

### Debugging & Troubleshooting
- Document debugging sessions with filename: `DebugInfo-{timestamp}-{5-word-bug-description}.md`
- File must include:
  - Identified problem description
  - Relevant file paths
  - Proposed solution

### Migration or Refactoring
- Update code to follow project guidelines
- **DO NOT** create migration scripts/artifacts unless explicitly requested
- **DO NOT** leave deprecated methods around unless requested
- Adjust all tests to work with refactored code

## Common Development Tasks

### Adding a New CLI Command
1. Create file in `internal/cli/commands/` (e.g., `my_command.go`)
2. Implement command handler with Cobra's `&cobra.Command`
3. Register in an `init()` function:
   ```go
   func init() {
       cli.RootCmd.AddCommand(myCmd)
   }
   ```
4. Handle `cli.GlobalConfig` for JSON/verbose output
5. Call appropriate repository methods for data operations
6. **CRITICAL**: Write tests using MOCKED repositories (never use real database in CLI tests)

### Adding a Repository Method
1. Open the relevant repository file (`task_repository.go`, `epic_repository.go`, etc.)
2. Add method that:
   - Takes `*sql.Tx` for transaction support OR works with `r.db`
   - Returns error as second value (`(T, error)`)
   - Uses prepared statements or parameterized queries
   - Includes proper error wrapping: `fmt.Errorf("operation failed: %w", err)`

### Running a Single Test
```bash
go test -v ./internal/repository -run TestTaskStatusUpdate
```

### Database Debugging
```bash
sqlite3 shark-tasks.db          # Open SQLite CLI
.tables                          # List tables
.schema tasks                    # View task table schema
SELECT * FROM tasks LIMIT 5;    # Query data
```

### Hot-Reload Development
```bash
make dev  # Starts air which watches for file changes and rebuilds
```
