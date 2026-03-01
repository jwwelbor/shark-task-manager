# Development Workflows

## Quality Gate — MANDATORY

**Before considering any development work complete, you MUST run all three checks in order:**

```bash
make fmt    # Format all Go code
make lint   # Run golangci-lint static analysis
make test   # Run full test suite
```

**Rules:**
- Run these after ANY series of Go code changes (new features, bug fixes, refactoring, test updates)
- Fix any failures before declaring work done — do not skip or defer
- If `make fmt` changes files, re-run `make lint` and `make test` after
- If tests fail, fix the issue and re-run the full sequence
- This applies even for "small" changes — no exceptions

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
   # Positional syntax (recommended) - use --order for sequencing
   ./bin/shark task create E07 F01 "Task Title" --order=1
   ./bin/shark task create E07 F01 "Second Task" --order=2 --priority=3
   # OR combined format
   ./bin/shark task create E07-F01 "Task Title" --order=1

   # Flag syntax (legacy, still supported)
   ./bin/shark task create --epic=E07 --feature=F01 --title="Task Title" --order=1
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
# Advance to next workflow status
./bin/shark task next-status E07-F20-001
./bin/shark status advance E07-F20-001

# Set status directly
./bin/shark status set E07-F20-001 in_progress
./bin/shark task set-status E07-F20-001 in_progress

# Approve / reopen
./bin/shark task approve E07-F20-001
./bin/shark task reopen E07-F20-001
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
4. **Command must be a thin wrapper**: parse args/flags, call a service method, format output
5. Handle `cli.GlobalConfig` for JSON/verbose output
6. **Do NOT put business logic in commands** - call service methods instead
7. **Do NOT call repository methods directly** from commands - go through the service layer
8. **CRITICAL**: Write tests using MOCKED services (never use real database in CLI tests)

### Adding a Service Method
1. Add method to the appropriate service in `internal/services/` (e.g., `task_service.go`)
2. Service method should:
   - Contain all business logic, validation, and orchestration
   - Call repository methods for data access
   - Use `workflow.Service` for status/transition validation
   - Return domain objects and errors (no CLI/formatting concerns)
   - Manage transactions when multiple repo calls are needed

### Adding a Repository Method
1. Open the relevant repository file (`task_repository.go`, `epic_repository.go`, etc.)
2. Add method that:
   - Takes `*sql.Tx` for transaction support OR works with `r.db`
   - Returns error as second value (`(T, error)`)
   - Uses prepared statements or parameterized queries
   - Includes proper error wrapping: `fmt.Errorf("operation failed: %w", err)`
   - **Contains only data access logic** - no business rules, progress calculation, or status derivation

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
