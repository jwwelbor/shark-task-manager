# Implementation Phases: Template Variables for Related Docs and Tasks

**Feature:** E07-F29
**Version:** 1.0
**Last Updated:** 2026-02-13

## Overview

Implementation divided into 4 phases, each independently testable and deployable.

---

## Phase 1: Database Schema & Context Data Extension

**Duration:** 2-3 days
**Dependencies:** None (uses existing E07-F05 infrastructure)

### Tasks

1. **Create Relationship Tables**
   - Add `feature_relationships` table to `internal/db/db.go`
   - Add `epic_relationships` table to `internal/db/db.go`
   - Create indexes (from_id, to_id, type)
   - Add CHECK constraints for relationship types
   - Add FOREIGN KEY constraints with CASCADE delete

2. **Create Auto-Migration**
   - Add migration function in `internal/db/migrate.go`
   - Check if tables exist before creating
   - Safe to run multiple times (idempotent)
   - Test migration with existing databases

3. **Extend ContextData Model**
   - Add `RelatedFeatures []string` field to `internal/models/context_data.go`
   - Add `RelatedEpics []string` field
   - Add validation for new fields in `Validate()` method
   - Update JSON marshaling/unmarshaling
   - Add tests for extended model

### Acceptance Criteria

- [ ] `feature_relationships` table created with all constraints
- [ ] `epic_relationships` table created with all constraints
- [ ] Indexes created successfully
- [ ] Migration runs without errors on existing databases
- [ ] ContextData can parse JSON with new fields
- [ ] Backward compatibility: Old JSON without new fields still works
- [ ] Unit tests pass for extended ContextData model

### Testing

```bash
# Test migration
go test ./internal/db -run TestMigration_RelationshipTables

# Test context data extension
go test ./internal/models -run TestContextData_ExtendedFields

# Integration test
./bin/shark init --non-interactive
sqlite3 shark-tasks.db "SELECT name FROM sqlite_master WHERE type='table' AND name LIKE '%relationships';"
# Should output: feature_relationships, epic_relationships
```

---

## Phase 2: Repository Layer

**Duration:** 3-4 days
**Dependencies:** Phase 1

### Tasks

1. **FeatureRelationshipRepository**
   - Create `internal/repository/feature_relationship_repository.go`
   - Implement Create, GetByID, Delete methods
   - Implement GetByFeatureID, GetOutgoing, GetIncoming methods
   - Implement ListRelatedFeatures (for placeholder population)
   - Add relationship type validation
   - Add self-reference prevention

2. **EpicRelationshipRepository**
   - Create `internal/repository/epic_relationship_repository.go`
   - Same methods as FeatureRelationshipRepository
   - Implement ListRelatedEpics

3. **Unit Tests (Mocked)**
   - Mock tests for all repository methods
   - Test validation logic
   - Test error handling

4. **Integration Tests (Real DB)**
   - Test CRUD operations with real database
   - Test CASCADE delete behavior
   - Test UNIQUE constraint enforcement
   - Test bidirectional relationship queries

### Acceptance Criteria

- [ ] FeatureRelationshipRepository implements all required methods
- [ ] EpicRelationshipRepository implements all required methods
- [ ] ListRelatedFeatures returns feature keys (not full objects)
- [ ] Relationship type validation works at application level
- [ ] Self-reference prevention works
- [ ] UNIQUE constraint prevents duplicate relationships
- [ ] CASCADE delete removes relationships when entities deleted
- [ ] Unit tests pass (≥ 80% coverage)
- [ ] Integration tests pass with real database

### Testing

```bash
# Unit tests
go test ./internal/repository -run TestFeatureRelationshipRepository

# Integration tests
go test ./internal/repository -run TestFeatureRelationshipRepository_Integration

# Coverage
go test ./internal/repository -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## Phase 3: Placeholder Extension

**Duration:** 3-4 days
**Dependencies:** Phase 1, Phase 2

### Tasks

1. **Helper Functions**
   - Add `formatDocPathsAsCSV()` to `internal/config/template_helpers.go`
   - Add `extractRelatedTasksFromContext()`
   - Add `extractRelatedFeaturesFromContext()`
   - Add `extractRelatedEpicsFromContext()`
   - Implement graceful error handling (empty string fallback)

2. **Extended Placeholder Functions**
   - Implement `TaskPlaceholdersWithRelated()`
   - Implement `FeaturePlaceholdersWithRelated()`
   - Implement `EpicPlaceholdersWithRelated()`
   - Inject document repository via parameter
   - Inject relationship repositories
   - Add performance logging (> 50ms warnings)

3. **Unit Tests**
   - Mock document repository tests
   - Mock relationship repository tests
   - Test with no documents/relationships (empty string)
   - Test with errors (graceful degradation)
   - Test JSON parse errors

4. **Integration Tests**
   - Test with real documents from E07-F05
   - Test with real relationships
   - Test template population end-to-end

### Acceptance Criteria

- [ ] `{related_docs}` placeholder populates with comma-separated paths
- [ ] `{related_tasks}` placeholder populates from context_data
- [ ] `{related_features}` placeholder populates from table + context_data
- [ ] `{related_epics}` placeholder populates from table + context_data
- [ ] Empty strings returned for no relationships (not errors)
- [ ] Errors logged as WARNING (not ERROR)
- [ ] Placeholder population never fails (graceful degradation)
- [ ] Unit tests pass (≥ 80% coverage)
- [ ] Integration tests pass with real database

### Testing

```bash
# Unit tests
go test ./internal/config -run TestTaskPlaceholdersWithRelated

# Integration tests
go test ./internal/config -run TestPlaceholders_Integration

# End-to-end template test
./bin/shark task create E07 F29 "Test Task"
./bin/shark related-docs add "Spec" docs/spec.md --task=E07-F29-001
./bin/shark task get E07-F29-001 --json | jq '.orchestrator_action.instruction'
# Should contain: docs/spec.md
```

---

## Phase 4: Orchestrator Action Integration

**Duration:** 2-3 days
**Dependencies:** Phase 3

### Tasks

1. **Update TaskRepository**
   - Modify `GetOrchestratorActionForTask()` to use `TaskPlaceholdersWithRelated()`
   - Inject DocumentRepository
   - Handle errors gracefully (fallback to basic placeholders)

2. **Update FeatureService/EpicService**
   - Modify orchestrator action generation methods
   - Use `FeaturePlaceholdersWithRelated()` and `EpicPlaceholdersWithRelated()`
   - Inject DocumentRepository and RelationshipRepositories

3. **Update OrchestratorActionService (if exists)**
   - Inject required repositories
   - Use extended placeholder functions
   - Add constructor for dependency injection

4. **End-to-End Testing**
   - Test full workflow: task transition → orchestrator action generation
   - Verify placeholders populated in actual instructions
   - Test with templates using new placeholders
   - Test backward compatibility (templates without new placeholders)

### Acceptance Criteria

- [ ] Task status transitions generate instructions with `{related_docs}`
- [ ] Task status transitions generate instructions with `{related_tasks}`
- [ ] Feature status transitions generate instructions with `{related_features}`
- [ ] Epic actions generate instructions with `{related_epics}`
- [ ] Existing templates without new placeholders still work
- [ ] Error handling works (graceful fallback)
- [ ] End-to-end tests pass
- [ ] No regressions in existing orchestrator action tests

### Testing

```bash
# End-to-end workflow test
./bin/shark task create E07 F29 "Implement feature X"
./bin/shark related-docs add "API Spec" docs/api-spec.md --task=E07-F29-001
./bin/shark task start E07-F29-001
./bin/shark task get E07-F29-001 --json | jq '.orchestrator_action'
# Verify: instruction contains docs/api-spec.md

# Template backward compatibility
./bin/shark config get status_metadata.in_progress.orchestrator_action.instruction_template
# Update template: "Work on {id}. Read: {related_docs}"
./bin/shark task start E07-F29-002
# Verify: instruction populated correctly

# Regression tests
go test ./internal/repository -run TestTaskRepository_OrchestratorAction
go test ./internal/services -run TestFeatureService_OrchestratorAction
```

---

## Optional Phase 5: CLI Commands (Future)

**Duration:** 3-5 days
**Dependencies:** Phase 2
**Status:** Out of scope for MVP (Phase 2 enhancement)

### Tasks

1. **Feature Relationship Commands**
   - `shark feature relate <from> <to> --type=<type>`
   - `shark feature unrelate <from> <to> --type=<type>`
   - `shark feature relationships <feature> [--type=<type>]`

2. **Epic Relationship Commands**
   - `shark epic relate <from> <to> --type=<type>`
   - `shark epic unrelate <from> <to> --type=<type>`
   - `shark epic relationships <epic> [--type=<type>]`

3. **Visualization Commands**
   - `shark feature graph <feature>` (Graphviz/Mermaid output)
   - `shark epic graph <epic>`

### Deferred Rationale

- MVP focuses on placeholder population (immediate value)
- Relationships can be added via SQL or context_data JSON
- CLI commands add ~5 days development + testing
- Analytics features depend on relationship data existing first

---

## Testing Strategy

### Unit Tests

**Per Phase:**
- Phase 1: ContextData model tests
- Phase 2: Repository tests (mocked DB)
- Phase 3: Placeholder function tests (mocked repos)
- Phase 4: Orchestrator action tests (mocked repos)

**Coverage Target:** ≥ 80% for all new code

### Integration Tests

**Database Integration:**
- Phase 1: Migration tests (real SQLite)
- Phase 2: Repository CRUD tests (real SQLite)
- Phase 3: Placeholder population with real docs/relationships
- Phase 4: End-to-end status transition tests

### End-to-End Tests

**Scenarios:**
1. Task with related docs → placeholder population → instruction
2. Feature with related features → placeholder population → instruction
3. Task with no relationships → empty placeholders → valid instruction
4. Error in doc query → graceful degradation → empty placeholder

### Performance Tests

**Benchmarks:**
- Placeholder population: < 50ms (p95)
- Document query: < 10ms (p95)
- Relationship query: < 15ms (p95)

**Load Tests:**
- 100 concurrent task transitions
- Measure p95/p99 latencies

---

## Deployment Plan

### Database Migration

**Step 1: Backup**
```bash
cp shark-tasks.db shark-tasks.db.backup
```

**Step 2: Deploy Application**
```bash
make build
./bin/shark --version  # Verify new version
```

**Step 3: Run Migration**
```bash
./bin/shark init --non-interactive  # Auto-migration runs
```

**Step 4: Verify**
```bash
sqlite3 shark-tasks.db "SELECT name FROM sqlite_master WHERE type='table';"
# Should include: feature_relationships, epic_relationships
```

### Rollback Plan

**If migration fails:**
```bash
# 1. Stop application
# 2. Restore backup
cp shark-tasks.db.backup shark-tasks.db

# 3. Deploy previous version
git checkout <previous-tag>
make build
```

### Feature Flags (Optional)

**Gradual Rollout:**
```json
// .sharkconfig.json
{
  "features": {
    "template_variables_v2": true  // Enable new placeholders
  }
}
```

```go
// Check flag before using extended placeholders
if config.IsFeatureEnabled("template_variables_v2") {
    placeholders, _ = TaskPlaceholdersWithRelated(ctx, task, docRepo)
} else {
    placeholders = TaskPlaceholders(task)  // Fallback to basic
}
```

---

## Risk Mitigation

### Risk 1: Migration Failures

**Mitigation:**
- Idempotent migration (safe to run multiple times)
- Automatic backup before migration
- Test migration on copy of production database

### Risk 2: Performance Degradation

**Mitigation:**
- Benchmark tests before merge
- Monitor p95/p99 latencies post-deployment
- Graceful degradation (empty strings) prevents failures

### Risk 3: Backward Compatibility

**Mitigation:**
- Existing placeholder functions unchanged
- New placeholders optional (omitempty in JSON)
- All existing tests must pass

### Risk 4: Database Corruption

**Mitigation:**
- Relationship tables can be dropped and recreated
- Context data is JSON (can be manually edited)
- Backup/restore procedures in place

---

## Success Criteria

**Phase 1 Complete:**
- [x] Tables created successfully
- [x] Migration runs without errors
- [x] ContextData extended with new fields

**Phase 2 Complete:**
- [x] Repositories implement all required methods
- [x] Relationship CRUD operations work
- [x] Integration tests pass

**Phase 3 Complete:**
- [x] Placeholder functions return correct CSV strings
- [x] Graceful error handling works
- [x] Unit tests pass (≥ 80% coverage)

**Phase 4 Complete:**
- [x] Orchestrator actions use new placeholders
- [x] End-to-end workflow tests pass
- [x] No regressions in existing tests

**Feature Complete:**
- [x] All acceptance criteria met
- [x] Documentation updated
- [x] Performance benchmarks meet targets
- [x] Deployed to production without issues

---

## Timeline

**Total Estimated Duration:** 10-14 days

| Phase | Duration | Dependencies | Start | End |
|-------|----------|--------------|-------|-----|
| Phase 1: Schema & Models | 2-3 days | None | Day 1 | Day 3 |
| Phase 2: Repositories | 3-4 days | Phase 1 | Day 4 | Day 7 |
| Phase 3: Placeholders | 3-4 days | Phase 1, 2 | Day 8 | Day 11 |
| Phase 4: Integration | 2-3 days | Phase 3 | Day 12 | Day 14 |

**Buffer:** 2-3 days for unexpected issues, testing, documentation

**Total with Buffer:** 12-17 days (2.5-3.5 weeks)
