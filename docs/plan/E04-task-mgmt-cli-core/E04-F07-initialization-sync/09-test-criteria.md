# Test Criteria: Initialization & Synchronization

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F07-initialization-sync
**Date**: 2025-12-16
**Author**: tdd-agent

## Purpose

This document defines comprehensive test criteria for initialization and synchronization features, covering unit tests, integration tests, performance tests, and acceptance tests aligned with PRD requirements.

---

## Test Coverage Requirements

**From Epic NFR**:
- Minimum: >80% unit test coverage
- Critical paths: 100% coverage (CRUD operations, transactions, conflict resolution)

---

## Unit Tests

### 1. Repository Extensions (`internal/repository/`)

#### TaskRepository.BulkCreate

**Test Cases**:
```go
func TestBulkCreate_Success(t *testing.T)
func TestBulkCreate_ValidatesAllTasks(t *testing.T)
func TestBulkCreate_RollsBackOnError(t *testing.T)
func TestBulkCreate_UsesPreparedStatements(t *testing.T)
func TestBulkCreate_PerformanceWith100Tasks(t *testing.T)  // <1 second
```

#### TaskRepository.GetByKeys

**Test Cases**:
```go
func TestGetByKeys_ReturnsAllExistingTasks(t *testing.T)
func TestGetByKeys_ReturnsPartialResults_MissingKeys(t *testing.T)
func TestGetByKeys_ReturnsEmptyMap_NoKeysFound(t *testing.T)
func TestGetByKeys_PerformanceWith100Keys(t *testing.T)  // <100ms
```

#### TaskRepository.UpdateMetadata

**Test Cases**:
```go
func TestUpdateMetadata_UpdatesTitleDescriptionFilePath(t *testing.T)
func TestUpdateMetadata_PreservesStatus(t *testing.T)
func TestUpdateMetadata_PreservesPriority(t *testing.T)
func TestUpdateMetadata_PreservesAgentType(t *testing.T)
func TestUpdateMetadata_UpdatesUpdatedAtTimestamp(t *testing.T)
```

#### EpicRepository.CreateIfNotExists

**Test Cases**:
```go
func TestCreateIfNotExists_CreatesNewEpic(t *testing.T)
func TestCreateIfNotExists_ReturnsExistingEpic(t *testing.T)
func TestCreateIfNotExists_IsIdempotent(t *testing.T)
func TestCreateIfNotExists_HandlesRaceCondition(t *testing.T)
```

#### FeatureRepository.CreateIfNotExists

**Test Cases**:
```go
func TestCreateIfNotExists_CreatesNewFeature(t *testing.T)
func TestCreateIfNotExists_ReturnsExistingFeature(t *testing.T)
func TestCreateIfNotExists_ValidatesForeignKey(t *testing.T)
```

---

### 2. Initializer (`internal/init/`)

#### Initializer.Initialize

**Test Cases**:
```go
func TestInitialize_CreatesDatabase(t *testing.T)
func TestInitialize_CreatesFolders(t *testing.T)
func TestInitialize_CreatesConfig(t *testing.T)
func TestInitialize_CopiesTemplates(t *testing.T)
func TestInitialize_IsIdempotent(t *testing.T)
func TestInitialize_CompletesInUnder5Seconds(t *testing.T)
```

#### Database Creation

**Test Cases**:
```go
func TestCreateDatabase_CreatesSchemaCorrectly(t *testing.T)
func TestCreateDatabase_SetsPermissions600Unix(t *testing.T)
func TestCreateDatabase_SkipsIfExists(t *testing.T)
```

#### Folder Creation

**Test Cases**:
```go
func TestCreateFolders_CreatesAllFolders(t *testing.T)
func TestCreateFolders_SkipsExistingFolders(t *testing.T)
func TestCreateFolders_SetsPermissions755(t *testing.T)
```

#### Config Creation

**Test Cases**:
```go
func TestCreateConfig_WritesValidJSON(t *testing.T)
func TestCreateConfig_PromptsIfExists_Interactive(t *testing.T)
func TestCreateConfig_SkipsIfExists_NonInteractive(t *testing.T)
func TestCreateConfig_OverwritesWithForce(t *testing.T)
func TestCreateConfig_AtomicWrite(t *testing.T)
```

#### Template Copying

**Test Cases**:
```go
func TestCopyTemplates_CopiesAllTemplates(t *testing.T)
func TestCopyTemplates_SkipsExisting_NoForce(t *testing.T)
func TestCopyTemplates_OverwritesWithForce(t *testing.T)
```

---

### 3. File Scanner (`internal/sync/scanner.go`)

#### FileScanner.Scan

**Test Cases**:
```go
func TestScan_FindsAllTaskFiles(t *testing.T)
func TestScan_FiltersNonTaskFiles(t *testing.T)
func TestScan_RecursivelyScansDirectories(t *testing.T)
func TestScan_ExtractsFileMetadata(t *testing.T)
func TestScan_InfersEpicFromPath(t *testing.T)
func TestScan_InfersFeatureFromPath(t *testing.T)
func TestScan_HandlesLegacyFolderStructure(t *testing.T)
func TestScan_RejectsSymlinks(t *testing.T)
func TestScan_EnforcesFileSizeLimit(t *testing.T)
func TestScan_EnforcesFileCountLimit(t *testing.T)
func TestScan_PerformanceWith100Files(t *testing.T)  // <1 second
```

#### Epic/Feature Inference

**Test Cases**:
```go
func TestInferEpicFeature_FromDirectoryStructure(t *testing.T)
func TestInferEpicFeature_FromTaskKey(t *testing.T)
func TestInferEpicFeature_ReturnsEmptyOnFailure(t *testing.T)
```

---

### 4. Conflict Detector (`internal/sync/conflict.go`)

#### ConflictDetector.DetectConflicts

**Test Cases**:
```go
func TestDetectConflicts_TitleConflict(t *testing.T)
func TestDetectConflicts_DescriptionConflict(t *testing.T)
func TestDetectConflicts_FilePathConflict(t *testing.T)
func TestDetectConflicts_NoConflict_SameValues(t *testing.T)
func TestDetectConflicts_NoConflict_MissingFileTitle(t *testing.T)
func TestDetectConflicts_NoConflict_DatabaseOnlyFields(t *testing.T)
func TestDetectConflicts_MultipleConflicts(t *testing.T)
```

---

### 5. Conflict Resolver (`internal/sync/resolver.go`)

#### ConflictResolver.Resolve

**Test Cases**:
```go
func TestResolve_FileWinsStrategy_UpdatesTitle(t *testing.T)
func TestResolve_FileWinsStrategy_UpdatesDescription(t *testing.T)
func TestResolve_FileWinsStrategy_UpdatesFilePath(t *testing.T)
func TestResolve_DatabaseWinsStrategy_PreservesValues(t *testing.T)
func TestResolve_NewerWinsStrategy_FileNewer(t *testing.T)
func TestResolve_NewerWinsStrategy_DatabaseNewer(t *testing.T)
func TestResolve_PreservesStatus(t *testing.T)
func TestResolve_PreservesPriority(t *testing.T)
func TestResolve_PreservesAgentType(t *testing.T)
func TestResolve_PreservesDependsOn(t *testing.T)
```

---

### 6. Sync Engine (`internal/sync/engine.go`)

#### SyncEngine.Sync

**Test Cases**:
```go
func TestSync_ImportsNewTask(t *testing.T)
func TestSync_UpdatesExistingTask(t *testing.T)
func TestSync_DetectsAndResolvesConflicts(t *testing.T)
func TestSync_DryRunMode_NoChanges(t *testing.T)
func TestSync_CreatesMissingEpic(t *testing.T)
func TestSync_CreatesMissingFeature(t *testing.T)
func TestSync_SkipsTaskWithMissingFeature_NoCreateMissing(t *testing.T)
func TestSync_CleansUpOrphanedTasks(t *testing.T)
func TestSync_RollsBackOnError(t *testing.T)
func TestSync_GeneratesAccurateReport(t *testing.T)
func TestSync_HandlesInvalidYAML(t *testing.T)
func TestSync_HandlesMissingKey(t *testing.T)
func TestSync_HandlesKeyMismatch(t *testing.T)
func TestSync_CreatesHistoryRecords(t *testing.T)
```

---

## Integration Tests

### 1. Init Command Integration

**Test Cases**:
```go
func TestInitCommand_FullExecution(t *testing.T)
func TestInitCommand_Idempotent(t *testing.T)
func TestInitCommand_NonInteractive(t *testing.T)
func TestInitCommand_Force(t *testing.T)
func TestInitCommand_CustomDBPath(t *testing.T)
func TestInitCommand_CustomConfigPath(t *testing.T)
func TestInitCommand_JSONOutput(t *testing.T)
```

**Test Scenario** (TestInitCommand_FullExecution):
```go
// Setup: Clean temp directory
tmpDir := t.TempDir()
os.Chdir(tmpDir)

// Execute: Run pm init
cmd := exec.Command("pm", "init", "--non-interactive")
output, err := cmd.CombinedOutput()
assert.NoError(t, err)

// Verify: Database created
assert.FileExists(t, "shark-tasks.db")

// Verify: Folders created
assert.DirExists(t, "docs/plan")
assert.DirExists(t, "templates")

// Verify: Config created
assert.FileExists(t, ".pmconfig.json")

// Verify: Config is valid JSON
var config map[string]interface{}
data, _ := ioutil.ReadFile(".pmconfig.json")
assert.NoError(t, json.Unmarshal(data, &config))

// Verify: Templates copied
files, _ := ioutil.ReadDir("templates")
assert.True(t, len(files) > 0)

// Verify: Output contains success message
assert.Contains(t, string(output), "Shark CLI initialized successfully")
```

---

### 2. Sync Command Integration

**Test Cases**:
```go
func TestSyncCommand_ImportNewTasks(t *testing.T)
func TestSyncCommand_UpdateExistingTasks(t *testing.T)
func TestSyncCommand_ConflictResolution_FileWins(t *testing.T)
func TestSyncCommand_ConflictResolution_DatabaseWins(t *testing.T)
func TestSyncCommand_ConflictResolution_NewerWins(t *testing.T)
func TestSyncCommand_DryRun(t *testing.T)
func TestSyncCommand_SpecificFolder(t *testing.T)
func TestSyncCommand_CreateMissing(t *testing.T)
func TestSyncCommand_Cleanup(t *testing.T)
func TestSyncCommand_TransactionRollback(t *testing.T)
func TestSyncCommand_JSONOutput(t *testing.T)
```

**Test Scenario** (TestSyncCommand_ImportNewTasks):
```go
// Setup: Initialize database
exec.Command("pm", "init", "--non-interactive").Run()

// Setup: Create test task files
createTestTaskFile(t, "docs/plan/E04-epic/E04-F07-feature/T-E04-F07-001.md",
    "T-E04-F07-001", "Test Task 1", "Description 1")
createTestTaskFile(t, "docs/plan/E04-epic/E04-F07-feature/T-E04-F07-002.md",
    "T-E04-F07-002", "Test Task 2", "Description 2")

// Execute: Run pm sync
cmd := exec.Command("pm", "sync", "--create-missing")
output, err := cmd.CombinedOutput()
assert.NoError(t, err)

// Verify: Output shows 2 tasks imported
assert.Contains(t, string(output), "New tasks imported: 2")

// Verify: Tasks in database
db := openTestDB(t)
defer db.Close()

var count int
db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
assert.Equal(t, 2, count)

// Verify: Task details correct
var task models.Task
db.QueryRow("SELECT key, title, description FROM tasks WHERE key = ?",
    "T-E04-F07-001").Scan(&task.Key, &task.Title, &task.Description)
assert.Equal(t, "Test Task 1", task.Title)
assert.Equal(t, "Description 1", *task.Description)
```

---

### 3. End-to-End Workflows

**Test Cases**:
```go
func TestWorkflow_FirstTimeSetup(t *testing.T)
func TestWorkflow_GitPullSync(t *testing.T)
func TestWorkflow_DryRunThenApply(t *testing.T)
func TestWorkflow_LegacyMigration(t *testing.T)
```

**Test Scenario** (TestWorkflow_GitPullSync):
```go
// Setup: Initialize and sync
exec.Command("pm", "init", "--non-interactive").Run()
createTestTaskFile(t, "T-E04-F07-001.md", "T-E04-F07-001", "Original Title", "")
exec.Command("pm", "sync", "--create-missing").Run()

// Simulate: Git pull (file changed by collaborator)
updateTestTaskFile(t, "T-E04-F07-001.md", "T-E04-F07-001", "Updated Title", "")

// Execute: Sync after git pull
cmd := exec.Command("pm", "sync")
output, err := cmd.CombinedOutput()
assert.NoError(t, err)

// Verify: Database updated with new title
db := openTestDB(t)
defer db.Close()

var title string
db.QueryRow("SELECT title FROM tasks WHERE key = ?", "T-E04-F07-001").Scan(&title)
assert.Equal(t, "Updated Title", title)

// Verify: Conflict detected and resolved
assert.Contains(t, string(output), "Conflicts resolved: 1")
assert.Contains(t, string(output), "Original Title")
assert.Contains(t, string(output), "Updated Title")
```

---

## Performance Tests

### 1. Init Performance

**Test**: Init completes in <5 seconds (from PRD)

```go
func BenchmarkInit(b *testing.B) {
    for i := 0; i < b.N; i++ {
        tmpDir := b.TempDir()
        os.Chdir(tmpDir)

        start := time.Now()
        exec.Command("pm", "init", "--non-interactive").Run()
        duration := time.Since(start)

        if duration > 5*time.Second {
            b.Errorf("Init took %v, expected <5s", duration)
        }
    }
}
```

### 2. Sync Performance

**Test**: Sync processes 100 files in <10 seconds (from PRD)

```go
func BenchmarkSync100Files(b *testing.B) {
    // Setup: Create 100 test task files
    exec.Command("pm", "init", "--non-interactive").Run()
    for i := 0; i < 100; i++ {
        key := fmt.Sprintf("T-E04-F07-%03d", i+1)
        createTestTaskFile(b, key+".md", key, fmt.Sprintf("Task %d", i+1), "")
    }

    for i := 0; i < b.N; i++ {
        start := time.Now()
        exec.Command("pm", "sync", "--create-missing").Run()
        duration := time.Since(start)

        if duration > 10*time.Second {
            b.Errorf("Sync took %v, expected <10s", duration)
        }
    }
}
```

### 3. YAML Parsing Performance

**Test**: Parse YAML frontmatter in <10ms per file (from PRD)

```go
func BenchmarkYAMLParsing(b *testing.B) {
    yamlData := []byte(`---
task_key: T-E04-F07-001
title: Test Task
description: Test description
---`)

    for i := 0; i < b.N; i++ {
        start := time.Now()
        taskfile.ParseTaskFileContent(string(yamlData))
        duration := time.Since(start)

        if duration > 10*time.Millisecond {
            b.Errorf("Parsing took %v, expected <10ms", duration)
        }
    }
}
```

---

## Acceptance Tests (from PRD)

### Initialization Acceptance Tests

**AC1**: Initialize New Project
```
GIVEN: A new project with no Shark CLI infrastructure
WHEN: I run `pm init`
THEN: Database file `shark-tasks.db` is created with schema
AND: Folder structure `docs/plan/`, `templates/` is created
AND: Config file `.pmconfig.json` is created with defaults
AND: Task templates are copied to `templates/` folder
AND: Success message displays next steps
```

**AC2**: Idempotent Init
```
GIVEN: Shark CLI is already initialized
WHEN: I run `pm init` again
THEN: Command completes without errors (idempotent)
AND: Existing database is not modified
AND: Existing config is not overwritten (unless --force)
```

### File Scanning Acceptance Tests

**AC3**: Scan Multiple Folders
```
GIVEN: I have 10 task markdown files across multiple features under docs/plan/
WHEN: I run `pm sync`
THEN: All 10 files are scanned and parsed
AND: Sync report shows "Files scanned: 10"
```

**AC4**: Sync Specific Folder
```
GIVEN: I have task files in multiple folders
WHEN: I run `pm sync --folder=docs/plan/E04-task-mgmt-cli-core/E04-F06-task-creation`
THEN: Only task files in the specified folder are scanned
AND: Files in other folders are ignored
```

### New Task Import Acceptance Tests

**AC5**: Import New Task
```
GIVEN: File `docs/plan/E01-epic/E01-F02-feature/T-E01-F02-003.md` exists with valid frontmatter
AND: Task T-E01-F02-003 does not exist in database
WHEN: I run `pm sync --create-missing`
THEN: Task T-E01-F02-003 is created in database
AND: All metadata from frontmatter (key, title, description) is imported
AND: Status is set to "todo" (default for new tasks)
AND: File_path is set to actual file location
AND: Sync report shows "New tasks imported: 1"
```

**AC6**: Skip Invalid Frontmatter
```
GIVEN: File has invalid frontmatter (bad YAML)
WHEN: I run `pm sync`
THEN: Warning is logged: "Invalid frontmatter in <file>"
AND: File is skipped
AND: Sync continues with other files
```

### Conflict Resolution Acceptance Tests

**AC7**: File-Wins Strategy
```
GIVEN: Database shows task T-E01-F02-003 has title="Implement authentication"
AND: File frontmatter shows title="Add user authentication"
WHEN: I run `pm sync --strategy=file-wins`
THEN: Database title is updated to "Add user authentication"
AND: Conflict is reported in sync output
AND: Task_history record is created
```

**AC8**: Database-Wins Strategy
```
GIVEN: Database shows task T-E01-F02-003 has title="Implement authentication"
AND: File frontmatter shows title="Add user authentication"
WHEN: I run `pm sync --strategy=database-wins`
THEN: Database title remains "Implement authentication"
AND: Conflict is reported but database not modified
```

### File Path Update Acceptance Tests

**AC9**: Update File Path
```
GIVEN: Database shows task T-E01-F02-003 at file_path="docs/tasks/created/T-E01-F02-003.md"
AND: Actual file is at `docs/plan/E01-epic/E01-F02-feature/T-E01-F02-003.md`
WHEN: I run `pm sync`
THEN: Database file_path is updated to match actual location
AND: Task remains in feature folder (not moved)
AND: Sync report shows file path conflict resolved
```

### Dry-Run Mode Acceptance Tests

**AC10**: Dry-Run Preview
```
GIVEN: 5 files would be imported during sync
WHEN: I run `pm sync --dry-run --create-missing`
THEN: Sync report shows "New tasks imported: 5"
AND: Message shows "Dry-run mode: No changes will be made"
AND: Database is not modified
AND: Files are not moved
```

### Missing Epic/Feature Acceptance Tests

**AC11**: Skip Task Without Feature
```
GIVEN: File references feature "E99-F99" that doesn't exist in database
WHEN: I run `pm sync` (without --create-missing)
THEN: Warning is logged: "Task references non-existent feature E99-F99"
AND: Task is skipped (not imported)
```

**AC12**: Auto-Create Missing Feature
```
GIVEN: File references non-existent feature E99-F99
WHEN: I run `pm sync --create-missing`
THEN: Epic E99 is auto-created (if doesn't exist)
AND: Feature E99-F99 is auto-created
AND: Task is imported successfully
```

### Transaction Rollback Acceptance Tests

**AC13**: Rollback on Error
```
GIVEN: Sync is processing 10 files
AND: File #5 causes database constraint violation
WHEN: Sync fails
THEN: All database changes are rolled back
AND: Tasks from files #1-4 are not in database
AND: Error message explains the failure
```

### JSON Output Acceptance Tests

**AC14**: JSON Output
```
GIVEN: I run `pm sync --json --create-missing`
WHEN: Sync completes
THEN: Output is valid JSON
AND: JSON contains: files_scanned, tasks_imported, tasks_updated, conflicts_resolved, warnings, errors
```

### Non-Interactive Init Acceptance Tests

**AC15**: Non-Interactive Init
```
GIVEN: I run `pm init --non-interactive` in CI/CD
WHEN: Config file already exists
THEN: No prompt is shown (skip config creation)
AND: Command completes successfully
AND: Exit code is 0
```

---

## Security Tests

### 1. Path Traversal Prevention

**Test Cases**:
```go
func TestSecurity_PathTraversal_Rejected(t *testing.T)
func TestSecurity_AbsolutePathValidation(t *testing.T)
func TestSecurity_SymlinkRejection(t *testing.T)
```

**Test Scenario** (TestSecurity_PathTraversal_Rejected):
```go
// Setup: Create malicious task file with path traversal
maliciousPath := "../../../etc/passwd"
task := &TaskMetadata{
    Key:      "T-E04-F07-001",
    FilePath: maliciousPath,
}

// Execute: Validate file path
err := validateFilePath(task.FilePath)

// Verify: Path is rejected
assert.Error(t, err)
assert.Contains(t, err.Error(), "outside allowed directories")
```

### 2. SQL Injection Prevention

**Test Cases**:
```go
func TestSecurity_SQLInjection_TaskKey(t *testing.T)
func TestSecurity_SQLInjection_Title(t *testing.T)
func TestSecurity_SQLInjection_Description(t *testing.T)
```

**Test Scenario** (TestSecurity_SQLInjection_TaskKey):
```go
// Setup: Malicious task key with SQL injection attempt
maliciousKey := "T-E04-F07-001'; DROP TABLE tasks; --"

// Execute: Query database with malicious key
task, err := repo.GetByKey(ctx, maliciousKey)

// Verify: No SQL injection (parameterized query protects us)
assert.NoError(t, err) // Error because key not found, not SQL error
assert.Nil(t, task)

// Verify: Database still intact
var count int
db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&count)
// Count should be unchanged (no tables dropped)
```

### 3. File Size Limits

**Test Cases**:
```go
func TestSecurity_FileSizeLimit_Enforced(t *testing.T)
func TestSecurity_YAMLSizeLimit_Enforced(t *testing.T)
```

### 4. File Count Limits

**Test Cases**:
```go
func TestSecurity_FileCountLimit_Enforced(t *testing.T)
```

---

## Edge Case Tests

### 1. Empty Database + Empty Filesystem

**Test**: Sync with no files and no database records

```go
func TestEdgeCase_EmptyDatabaseEmptyFilesystem(t *testing.T)
```

### 2. Large Frontmatter

**Test**: Frontmatter near maximum size (1MB)

```go
func TestEdgeCase_LargeFrontmatter(t *testing.T)
```

### 3. Concurrent Sync Operations

**Test**: Two sync processes running simultaneously

```go
func TestEdgeCase_ConcurrentSync(t *testing.T)
```

### 4. Context Cancellation

**Test**: User presses Ctrl+C during sync

```go
func TestEdgeCase_ContextCancellation(t *testing.T)
```

### 5. Database Locked

**Test**: Sync while another process holds database lock

```go
func TestEdgeCase_DatabaseLocked(t *testing.T)
```

---

## Test Data Fixtures

### Sample Task File

```markdown
---
key: T-E04-F07-001
title: Implement sync engine
description: Core synchronization logic between filesystem and database
---

# Task: Implement sync engine

## Description

Implement the main synchronization engine that orchestrates file scanning,
frontmatter parsing, conflict detection, and database updates.

## Acceptance Criteria

- [ ] Scan feature folders recursively
- [ ] Parse YAML frontmatter
- [ ] Detect conflicts between file and database
- [ ] Apply resolution strategy
- [ ] Update database in transaction
```

### Sample Database State

```sql
INSERT INTO epics (key, title, status, priority) VALUES
    ('E04', 'Task Management CLI Core', 'active', 'high');

INSERT INTO features (epic_id, key, title, status, progress_pct) VALUES
    (1, 'E04-F07', 'Initialization & Synchronization', 'active', 0.0);

INSERT INTO tasks (feature_id, key, title, description, status, priority, file_path) VALUES
    (1, 'T-E04-F07-001', 'Original Title', 'Original Description', 'todo', 5,
     'docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/T-E04-F07-001.md');
```

---

## Test Execution Plan

### Phase 1: Unit Tests

**Execute**: During development, run after each component is implemented
**Command**: `go test ./internal/...`
**Target**: >80% coverage

### Phase 2: Integration Tests

**Execute**: After unit tests pass, run before merging
**Command**: `go test -tags=integration ./...`
**Target**: All integration tests pass

### Phase 3: Performance Tests

**Execute**: Before release, after all functionality complete
**Command**: `go test -bench=. ./...`
**Target**: Meet PRD performance requirements

### Phase 4: Acceptance Tests

**Execute**: Manual verification or automated E2E tests
**Command**: Manual test scenarios or `go test -tags=e2e ./...`
**Target**: All PRD acceptance criteria met

---

## Continuous Integration

### CI Pipeline

```yaml
name: Test
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21

      - name: Run unit tests
        run: go test -v -coverprofile=coverage.out ./...

      - name: Check coverage
        run: |
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$coverage < 80" | bc -l) )); then
            echo "Coverage $coverage% is below 80%"
            exit 1
          fi

      - name: Run integration tests
        run: go test -tags=integration -v ./...

      - name: Run benchmarks
        run: go test -bench=. -benchtime=5s ./...
```

---

## Definition of Done (Testing)

Feature testing is complete when:
- [ ] All unit tests pass (>80% coverage)
- [ ] All integration tests pass
- [ ] All performance benchmarks meet targets
- [ ] All PRD acceptance criteria verified
- [ ] All security tests pass
- [ ] All edge cases handled
- [ ] CI pipeline passes
- [ ] Code reviewed and approved

---

**Document Complete**: 2025-12-16
