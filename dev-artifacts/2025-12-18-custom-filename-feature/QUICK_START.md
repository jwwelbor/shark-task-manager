# E07-F01 Quick Start Guide

**Feature**: Custom Filenames for Epics and Features
**Epic**: E07-enhancements
**Feature Key**: E07-F01-custom-filenames-for-epics-and-features
**Status**: Ready for Development
**Created**: 2025-12-18

---

## 🎯 What You're Building

Add `--filename` and `--force` flags to epic and feature creation commands, enabling users to specify custom file paths for documentation instead of being locked into the standard directory structure.

### User Experience (Goal State)

```bash
# Create epic with custom filename
./bin/shark epic create "Q1 Roadmap" --filename="docs/roadmap/q1.md"

# Create feature with custom filename
./bin/shark feature create "API Design" --epic=E07 --filename="docs/api/design.md"

# Force reassignment from another entity
./bin/shark epic create "New Epic" --filename="docs/shared.md" --force
```

---

## 📋 Implementation Tasks (10 Total)

| # | Task | Priority | Agent | Duration | Status |
|---|------|----------|-------|----------|--------|
| 1 | Repository methods (GetByFilePath, UpdateFilePath) | 1 | backend | 2-3h | todo |
| 2 | Extract ValidateCustomFilename to shared module | 2 | backend | 1-2h | todo |
| 3 | Add FilePath field to models | 3 | backend | 30m | todo |
| 4 | Implement epic creator function | 4 | backend | 2-3h | todo |
| 5 | Implement feature creator function | 5 | backend | 2-3h | todo |
| 6 | Add CLI flags to epic create command | 6 | backend | 1-2h | todo |
| 7 | Add CLI flags to feature create command | 7 | backend | 1-2h | todo |
| 8 | Database schema (file_path columns, indexes) | 8 | backend | 1h | todo |
| 9 | Update documentation | 9 | docs | 1-2h | todo |
| 10 | Write tests and verify | 10 | qa | 3-4h | todo |

**Total Estimated Effort**: 17-24 hours (similar to task filename feature)

---

## 🚀 Start Here

### Step 1: Understand the Reference
Read the task filename implementation to understand the pattern:
```bash
cat dev-artifacts/2025-12-18-custom-filename-feature/PROGRESS.md
cat ~/.claude/plans/ancient-purring-tower.md
```

### Step 2: Review the PRD
```bash
cat docs/plan/E07-enhancements/E07-F01-custom-filenames-for-epics-and-features/prd.md
```

### Step 3: Read Task T-E07-F01-001
```bash
cat docs/tasks/todo/T-E07-F01-001.md
```

### Step 4: Start Implementation
```bash
# Mark task as in progress
./bin/shark task start T-E07-F01-001

# Then follow the implementation steps in the task
```

---

## 🔑 Key Files to Modify

### Phase 1-2: Foundation
- `internal/repository/epic_repository.go` - Add GetByFilePath, UpdateFilePath
- `internal/repository/feature_repository.go` - Add GetByFilePath, UpdateFilePath
- `internal/taskcreation/validation.go` - NEW: Shared ValidateCustomFilename
- `internal/db/db.go` - Add indexes on file_path columns

### Phase 3-5: Logic
- `internal/models/epic.go` - Add FilePath field
- `internal/models/feature.go` - Add FilePath field
- `internal/taskcreation/creator.go` - Create CreateEpic, CreateFeature functions

### Phase 6-7: CLI
- `internal/cli/commands/epic.go` - Add --filename, --force flags
- `internal/cli/commands/feature.go` - Add --filename, --force flags

### Phase 8-9: Documentation
- `docs/CLI_REFERENCE.md` - Document new flags
- `CLAUDE.md` - Update command references

### Phase 10: Testing
- `internal/repository/epic_repository_test.go` - NEW
- `internal/repository/feature_repository_test.go` - NEW
- `internal/taskcreation/validation_test.go` - NEW
- `internal/cli/commands/epic_test.go` - NEW/UPDATE
- `internal/cli/commands/feature_test.go` - NEW/UPDATE

---

## 📐 Core Implementation Pattern

### Validation
```go
// All entities use same validation function
absPath, relPath, err := ValidateCustomFilename(filename, projectRoot)
if err != nil {
    // Reject: absolute path, traversal, wrong extension, etc.
}
```

### Collision Detection
```go
// Check if another entity owns the file
existing, err := epicRepository.GetByFilePath(ctx, relPath)
if existing != nil && !force {
    return nil, fmt.Errorf("file '%s' is already claimed by %s", relPath, existing.Key)
}
```

### Force Reassignment
```go
// Clear file from old entity, assign to new one
if existing != nil && force {
    epicRepository.UpdateFilePath(ctx, existing.Key, nil)  // Clear old
}
// Then create new epic with file_path set
```

### File Operations
```go
// Check if file exists
fileExists := fileExists(fullPath)

if !fileExists {
    // Create file with markdown header
    createFile(fullPath, epicMarkdownHeader)
} else {
    // Just associate - don't modify
}
```

---

## ✅ Acceptance Criteria Per Task

### T-E07-F01-001: Repository Methods
- [ ] GetByFilePath returns nil for non-existent paths
- [ ] UpdateFilePath handles nil to clear file_path
- [ ] Indexes created for performance
- [ ] All tests passing

### T-E07-F01-002: Validation Module
- [ ] ValidateCustomFilename in shared location
- [ ] Task creator updated to use it
- [ ] All validation tests passing

### T-E07-F01-003: Data Models
- [ ] FilePath field added to Epic struct
- [ ] FilePath field added to Feature struct
- [ ] Database tags correct
- [ ] Code compiles

### T-E07-F01-004: Epic Creator
- [ ] CreateEpic function accepts custom filenames
- [ ] Collision detection works
- [ ] Force reassignment works
- [ ] File operations work
- [ ] Tests passing

### T-E07-F01-005: Feature Creator
- [ ] Mirrors epic creator implementation
- [ ] All features working

### T-E07-F01-006: Epic CLI
- [ ] --filename flag works
- [ ] --force flag works
- [ ] Errors handled gracefully
- [ ] Manual CLI testing passed

### T-E07-F01-007: Feature CLI
- [ ] Same as epic CLI

### T-E07-F01-008: Database Schema
- [ ] file_path column exists
- [ ] UNIQUE constraint enforced
- [ ] Indexes created
- [ ] Fresh database initializes

### T-E07-F01-009: Documentation
- [ ] CLI_REFERENCE.md updated
- [ ] CLAUDE.md updated
- [ ] Examples provided
- [ ] Validation rules documented

### T-E07-F01-010: Testing
- [ ] All unit tests passing
- [ ] All integration tests passing
- [ ] Manual E2E testing complete
- [ ] 90%+ code coverage

---

## 🧪 Testing Checklist

### Manual CLI Tests (Do These)
```bash
# Test 1: Basic custom filename
./bin/shark epic create "Q1 Roadmap" --filename="docs/roadmap/q1.md"
# Verify: File created at docs/roadmap/q1.md
# Verify: Database shows file_path set

# Test 2: Collision detection
./bin/shark feature create "Another Feature" --filename="docs/roadmap/q1.md"
# Expected: Error about collision

# Test 3: Force reassignment
./bin/shark epic create "Different Epic" --filename="docs/roadmap/q1.md" --force
# Verify: Old epic's file_path is NULL
# Verify: New epic has file_path set

# Test 4: Validation failures
./bin/shark epic create "Test" --filename="/absolute/path.md"
# Expected: Error about absolute path

./bin/shark feature create "Test" --filename="../../../etc/passwd.md"
# Expected: Error about path traversal

./bin/shark epic create "Test" --filename="docs/file.txt"
# Expected: Error about .md extension

# Test 5: Default behavior
./bin/shark epic create "Standard Epic"
# Verify: Uses default location docs/plan/{epic-key}/epic.md
```

---

## 📚 Documentation References

| Document | Purpose | Link |
|----------|---------|------|
| Feature PRD | Complete requirements | docs/plan/E07-enhancements/E07-F01-custom-filenames-for-epics-and-features/prd.md |
| Reference Plan | Task implementation pattern | ~/.claude/plans/ancient-purring-tower.md |
| Progress | Task implementation details | dev-artifacts/2025-12-18-custom-filename-feature/PROGRESS.md |
| Task Details | Each task in full | docs/tasks/todo/T-E07-F01-{001..010}.md |

---

## 🔗 Implementation Dependencies

```
Start Here:
└─ T-E07-F01-001 (Repository Methods)
└─ T-E07-F01-002 (Validation Module)

Then:
├─ T-E07-F01-003 (Data Models)
├─ T-E07-F01-008 (Database Schema)
└─ Complete these before proceeding

Then:
├─ T-E07-F01-004 (Epic Creator)
└─ T-E07-F01-005 (Feature Creator)

Then:
├─ T-E07-F01-006 (Epic CLI)
└─ T-E07-F01-007 (Feature CLI)

Then:
└─ T-E07-F01-009 (Documentation)

Finally:
└─ T-E07-F01-010 (Testing & Verification)
```

**Critical Path**: Tasks 1, 2, 3, 8, 4, 5, 6, 7, 9, 10 (in order)

---

## 💡 Pro Tips

1. **Use task creator as reference**: Almost every step has a parallel in the task filename feature. When stuck, look there first.

2. **Test incrementally**: After each task, run `make test` to verify nothing broke.

3. **Error messages matter**: Copy error message formats from task creator so users see consistent style.

4. **File operations**: Use same file creation/association logic as task creator.

5. **Keep it simple**: Focus on what's asked. Don't add features beyond the scope.

6. **Document as you go**: Update CLAUDE.md and CLI_REFERENCE.md in task 9 based on actual implementation.

---

## ❓ Troubleshooting

### Database column doesn't exist
Solution: Check that T-E07-F01-008 has been completed and `make build` was run.

### Validation function not found
Solution: Ensure T-E07-F01-002 is completed and validation.go is created.

### Tests failing on collision detection
Solution: Verify repository methods were added correctly in T-E07-F01-001.

### CLI flags not recognized
Solution: Check flag names match exactly (--filename, --force) in epic.go and feature.go.

---

## 📊 Success Metrics

Upon completion:
- ✅ All 10 tasks marked as completed
- ✅ All tests passing: `make test`
- ✅ Code compiles: `make build`
- ✅ Manual CLI testing passes all scenarios
- ✅ 90%+ code coverage
- ✅ Documentation complete

---

## 🎓 Learning Outcomes

By completing this feature, you will understand:
- How to add new repository methods to existing patterns
- How to implement input validation with security checks
- How to handle file system operations safely
- How to add CLI flags and parse them
- How to prevent data collisions
- How to write comprehensive unit and integration tests
- How to keep code DRY through extraction and reuse

---

**Ready to start?** Go ahead and begin with T-E07-F01-001!

Questions? See the PRD and PROGRESS.md files for additional context.
