# UAT Acceptance Plan: E21 Entity Polymorphism and Duplication Reduction

**Author**: QA Agent
**Date**: 2026-03-19
**Status**: Draft
**Epic**: E21

---

## 1. Acceptance Scenarios

### Scenario 1: Add New Entity Type (Journey 1)

**Persona**: New Entity Author

**Preconditions**:
- E21 F01 through F05 are complete and merged
- All existing tests pass (`make fmt && make lint && make test`)
- EntityRegistry is initialized at application startup with all 5 existing entity types
- A hypothetical 6th entity type ("Milestone") is defined for this validation exercise

**Steps**:

1. Define a `Milestone` model struct in a new file `internal/models/milestone.go` that implements the `Entity` interface (GetID, GetKey, GetTitle, GetSlug, GetEntityType, GetStatus, SetStatus, GetDescription, GetFilePath, GetContextData, SetContextData, GetCreatedAt, GetUpdatedAt, Validate)
2. Add a compile-time interface satisfaction check: `var _ Entity = (*Milestone)(nil)`
3. Create a typed `MilestoneRepository` and an `EntityRepository` adapter (`MilestoneRepositoryAdapter`) in 1 file
4. Register the Milestone in the EntityRegistry with a single `Register()` call
5. Verify: `make fmt && make lint && make test` passes with zero errors
6. Verify: NoteService, ContextService, and ResumeService accept the new entity type without any code changes to those services
7. Verify: Status transitions work via the shared `EntityService.TransitionStatus` without writing any transition logic
8. Count the total files created or modified

**Expected Outcomes**:
- The 6th entity type is fully functional for all cross-cutting operations (notes, context, resume, status transitions, document linking) by implementing the Entity interface and registering with the EntityRegistry
- Total files created or modified: 5 or fewer (model, repository, adapter, registration, database migration)
- Zero modifications to NoteService, ContextService, ResumeService, or EntityService
- Zero new switch statements or setter methods added
- All existing tests continue to pass

**Pass Criteria**: Files touched <= 5 AND zero cross-cutting service modifications AND `make test` passes

---

### Scenario 2: Fix Cross-Cutting Bug (Journey 2)

**Persona**: Codebase Maintainer

**Preconditions**:
- E21 F03 (Status Transition Unification) is complete
- EntityService.TransitionStatus is the single implementation for shared transition logic
- A simulated bug is introduced: backward transition reason requirement is incorrectly skipped when `RequireRejectionReason` is true

**Steps**:

1. Identify the bug location: it must be in `EntityService.TransitionStatus` (a single method in `entity_service.go`)
2. Apply the fix to the single method in `entity_service.go`
3. Update or add a test in `entity_service_test.go` that reproduces the bug and verifies the fix
4. Run `make test` and verify ALL 5 entity types inherit the fix automatically
5. Count files modified for the fix

**Expected Outcomes**:
- The bug exists in exactly 1 file (`entity_service.go`), not spread across 5 service files
- The fix is applied in 1 file, tested in 1 test file
- Running `make test` confirms all 5 entity types pass with the fix (no per-entity test updates needed)
- Total files modified: 2 (service + test)

**Pass Criteria**: Files modified <= 2 AND all entity types inherit the fix AND `make test` passes

---

### Scenario 3: Add Cross-Cutting Feature (Journey 3)

**Persona**: Codebase Maintainer

**Preconditions**:
- E21 F01 and F04 (Document Operations Unification) are complete
- EntityService handles document linking polymorphically via EntityRepository

**Steps**:

1. Add a new cross-cutting method to EntityService (e.g., `GetEntitySummary(ctx, repo EntityRepository, key string) (*EntitySummary, error)`) that retrieves an entity by key and returns a summary struct with Key, Title, Status, EntityType
2. Write a single test for the new method using a mock EntityRepository
3. Verify the method works for all 5 entity types by passing different entity type adapters
4. Count files created or modified

**Expected Outcomes**:
- The new feature is implemented once in `entity_service.go` (~15-20 lines)
- One test file is updated (`entity_service_test.go`) with parameterized tests covering all 5 entity types
- The feature works identically for Epic, Feature, Task, Bug, and ChangeCard without per-entity implementation
- Total files modified: 2 (service + test)
- Total new lines of code: ~30 (vs. ~150 under the old pattern of 30 lines x 5 entities)

**Pass Criteria**: Files modified <= 2 AND feature works for all 5 entity types AND `make test` passes

---

### Scenario 4: Onboarding and Code Navigability (Journey 4)

**Persona**: New Entity Author (first-time contributor variant)

**Preconditions**:
- E21 is complete
- A reviewer (QA or tech lead) acts as the "new contributor" for this scenario

**Steps**:

1. Read `internal/models/entity.go` -- verify the Entity interface is clearly documented with godoc comments explaining each method's purpose and contract
2. Read `internal/services/entity_service.go` -- verify the shared TransitionStatus method has clear documentation of its 10-step process and the TransitionFeatures opt-in mechanism
3. Read one entity-specific service (e.g., `epic_service.go`) -- verify it is clear which methods delegate to EntityService and which contain epic-specific logic
4. Verify that `entity.go` contains compile-time interface satisfaction checks for all 5 entity types (`var _ Entity = (*Epic)(nil)` etc.)
5. Attempt to answer: "What does a new entity need to implement?" -- the answer should be fully derivable from `entity.go` + `entity_service.go` without reading all 5 entity implementations

**Expected Outcomes**:
- The Entity interface in `entity.go` is self-documenting: a contributor reads 1 file to understand the contract for all entities
- EntityService in `entity_service.go` is self-documenting: a contributor reads 1 file to understand all shared operations
- Entity-specific services contain only unique logic (dependency graphs, progress rollups, triage, approvals), not duplicated cross-cutting code
- A new contributor can answer "what do I need to implement?" by reading 2 files, not by comparing 5 independent implementations

**Pass Criteria**: All documentation present AND interface contract clear AND entity-specific services contain only unique logic (subjective assessment by reviewer)

---

## 2. Success Metrics Validation Plan

### Metric 1: Service Layer Line Reduction

**KPI**: Net reduction of 800+ lines (minimum viable: 500+)

**Measurement Procedure**:
```bash
# BEFORE (baseline - capture at epic start, before any F01 work)
wc -l internal/services/*_service.go > /tmp/e21-baseline-lines.txt
cat /tmp/e21-baseline-lines.txt  # Record total

# AFTER (at each feature boundary and at epic completion)
wc -l internal/services/*_service.go internal/services/entity_service.go internal/services/entity_registry.go internal/services/entity_repository.go internal/services/*_adapter.go > /tmp/e21-after-lines.txt
cat /tmp/e21-after-lines.txt  # Record total

# CALCULATION
# Net reduction = (baseline total) - (after total)
# New shared files are counted in the "after" total
```

**Schedule**: Measure at F01, F02, F03, F04, F05 boundaries and at epic completion.

**Pass/Fail**:
- 800+ lines: Primary target met
- 500-799 lines: Minimum viable met
- <500 lines: FAIL -- refactoring did not achieve sufficient reduction

---

### Metric 2: Files Required for New Entity Type

**KPI**: 5 or fewer files (minimum viable: 8 or fewer)

**Measurement Procedure**:
1. After F02 completion, create a documented "New Entity Checklist" listing every file that must be created or modified
2. Optionally: Execute Scenario 1 (spike a placeholder Milestone entity) and count actual files
3. Compare against the 15+ file baseline documented in the epic

**Pass/Fail**:
- 5 or fewer files: Primary target met
- 6-8 files: Minimum viable met
- 9+ files: FAIL -- insufficient reduction in extension effort

---

### Metric 3: Switch Statement Elimination

**KPI**: 0 entity-type switch branches in cross-cutting services

**Measurement Procedure**:
```bash
# Count switch/case branches dispatching by entity type
grep -n 'case models.EntityType\|case "epic"\|case "feature"\|case "task"\|case "bug"\|case "change' \
  internal/services/note_service.go \
  internal/services/context_service.go \
  internal/services/resume_service.go | wc -l

# Also check for broader patterns
grep -rn 'switch.*[Ee]ntity[Tt]ype\|switch.*entityType' \
  internal/services/note_service.go \
  internal/services/context_service.go \
  internal/services/resume_service.go | wc -l
```

**Schedule**: Measure after F02 completion.

**Pass/Fail**:
- 0 branches: Target met
- Any non-zero count: FAIL -- the switch elimination is binary

---

### Metric 4: Test Pass Rate at Phase Boundaries

**KPI**: 100% test pass rate at every feature boundary

**Measurement Procedure**:
```bash
# At each feature boundary (after F01, F02, F03, F04, F05)
make fmt && make lint && make test

# Record: total tests, passing tests, failing tests, lint warnings
# Any failure or warning = FAIL for that boundary
```

**Schedule**: After every feature merge (F01, F02, F03, F04, F05).

**Pass/Fail**:
- 100% pass, 0 lint warnings: Target met
- Any test failure: FAIL -- work cannot proceed to next feature until resolved
- Any lint warning: FAIL -- must be fixed before declaring feature complete

---

### Metric 5: Setter Method Elimination

**KPI**: 0 SetXxxRepo() setter methods on cross-cutting services

**Measurement Procedure**:
```bash
# Count setter methods for entity repository wiring
grep -rn 'func.*Set.*Repo\|func.*Set.*Service' \
  internal/services/note_service.go \
  internal/services/context_service.go \
  internal/services/resume_service.go | wc -l

# Also count setter CALLS in CLI accessors
grep -rn '\.Set.*Repo\|\.Set.*Service' \
  internal/cli/services_global.go \
  internal/cli/services_global_ext.go 2>/dev/null | wc -l
```

**Schedule**: Measure after F02 completion.

**Pass/Fail**:
- 0 setter methods and 0 setter calls: Target met
- Any non-zero count: FAIL

---

### Metric 6: Interface Satisfaction Compile Checks

**KPI**: 5 compile-time checks (one per entity type)

**Measurement Procedure**:
```bash
# Count compile-time interface satisfaction checks
grep -c 'var _ Entity' internal/models/entity.go

# Verify the specific types
grep 'var _ Entity' internal/models/entity.go
# Expected output:
# var _ Entity = (*Epic)(nil)
# var _ Entity = (*Feature)(nil)
# var _ Entity = (*Task)(nil)
# var _ Entity = (*Bug)(nil)
# var _ Entity = (*ChangeCard)(nil)
```

**Schedule**: Measure after F01 completion.

**Pass/Fail**:
- 5 checks present: Target met
- <5 checks: FAIL -- missing entity type implementations

---

## 3. Cross-Epic Integration Scenarios

### 3.1 E15: Service Layer Architecture Refactoring (Active, ~79% complete)

**Overlap**: E15 migrates CLI commands to use the service layer. E21 restructures service internals. Both touch files in `internal/services/`.

**Integration Test**:
1. Before starting E21 F03 (Status Transition Unification), verify E15 has no pending work on `epic_service.go`, `feature_service.go`, or `task_service.go`
2. After E21 F03, verify all E15-migrated CLI commands still work correctly by running the full test suite
3. Verify that CLI commands using `cli.GetTaskService()`, `cli.GetEpicService()`, etc. continue to function after EntityService composition changes

**Risk Level**: Medium. F01 and F02 have zero overlap with E15. F03 is the conflict point.

**Mitigation**: Sequence E21 F01 and F02 first. Begin F03 only after E15 confirmation.

### 3.2 E19: Sprint Management & Planning System (Planned)

**Synergy**: E19 would add a Sprint entity type. If E21 completes first, E19's Sprint implementation effort drops from ~15 files to ~5 files.

**Integration Test**:
1. After E21 completion, verify Scenario 1 (Add New Entity Type) passes -- this directly validates E19's future implementation path
2. Document the "New Entity Checklist" so E19 developers have a clear guide

**Risk Level**: None. E19 benefits from E21. No conflict.

### 3.3 E12: Repository Layer Database Interface Migration (Draft)

**Overlap**: E12 proposes migrating repositories from `*sql.DB` to `db.Database` interface. E21 creates EntityRepository adapters wrapping existing typed repositories.

**Integration Test**:
1. If E12 activates during E21 execution, verify that EntityRepository adapters are thin enough to update quickly (~15-20 lines each)
2. No proactive action required -- E12 is in draft with no progress

**Risk Level**: Low.

---

## 4. Risk-Based Test Priorities

### P0: Zero Behavioral Regression (MUST PASS -- Blocks everything)

**What**: All existing tests pass at every feature boundary with zero behavioral changes.

**How**:
```bash
make fmt && make lint && make test
```

**When**: After every feature merge (F01, F02, F03, F04, F05).

**Failure Action**: STOP all work. Fix the regression before proceeding. The entire value proposition of E21 depends on zero behavioral change. A single test failure indicates the refactoring is unsafe.

**Rationale**: This is a pure internal refactoring. If external behavior changes, the refactoring has introduced a bug. There is no acceptable threshold for behavioral regression -- it must be zero.

---

### P1: Entity Interface Compiles for All 5 Types (MUST PASS -- Foundation for everything)

**What**: All 5 entity types implement the Entity interface and pass compile-time satisfaction checks.

**How**:
1. Verify `var _ Entity = (*Epic)(nil)` (and 4 others) in `internal/models/entity.go`
2. Run `go build ./...` -- compilation succeeds with no errors
3. Run entity-specific accessor tests verifying each method returns the correct value

**When**: After F01 completion.

**Failure Action**: F01 is incomplete. No downstream features (F02-F05) can proceed until all 5 types implement the interface.

**Rationale**: The Entity interface is the foundation of the entire epic. If any entity type fails to implement it, all cross-cutting unification is blocked.

---

### P2: Cross-Cutting Services Work with EntityRepository (SHOULD PASS -- Core value delivery)

**What**: NoteService, ContextService, and ResumeService operate correctly using the EntityRegistry and EntityRepository adapters, with zero switch statements and zero setter methods.

**How**:
1. Run Metric 3 (switch statement elimination) -- must be 0
2. Run Metric 5 (setter method elimination) -- must be 0
3. Execute existing NoteService/ContextService/ResumeService tests -- all must pass
4. Manually test: `shark task note add`, `shark epic context set`, `shark task resume` for each entity type

**When**: After F02 completion.

**Failure Action**: F02 is incomplete. The registry pattern is not correctly replacing the switch/setter pattern.

**Rationale**: Cross-cutting service unification is the highest value-to-effort feature. If it fails, the extension benefit (Journey 1) is not achieved.

---

### P3: Line Reduction Targets Met (NICE TO HAVE -- Quantitative validation)

**What**: Service layer line count reduced by 800+ lines (minimum 500+).

**How**: Run Metric 1 measurement procedure.

**When**: At epic completion.

**Failure Action**: If 500+ lines: acceptable (minimum viable). If <500 lines: investigate whether the shared code is larger than expected or the removed duplication is less than estimated. Document findings. This does not block release but indicates the refactoring's quantitative impact is less than projected.

**Rationale**: Line reduction is the lagging indicator. The architectural benefits (extensibility, consistency, maintainability) are captured by P0-P2 regardless of whether the exact line count target is hit.

---

## 5. Persona Acceptance Matrix

| Persona | Scenario | Priority | Acceptance Criteria | How Validated |
|---------|----------|----------|---------------------|---------------|
| **Codebase Maintainer** | Scenario 2: Fix Cross-Cutting Bug | P0, P2 | Bug fix in 1 file applies to all 5 entity types; all tests pass | Execute Scenario 2; run `make test` |
| **Codebase Maintainer** | Scenario 3: Add Cross-Cutting Feature | P2 | New feature in 1 file works for all 5 entity types | Execute Scenario 3; run `make test` |
| **New Entity Author** | Scenario 1: Add New Entity Type | P1, P2 | 6th entity type functional in 5 or fewer files; zero cross-cutting service changes | Execute Scenario 1; count files; verify no NoteService/ContextService changes |
| **New Entity Author** | Scenario 4: Onboarding | P1 | Entity interface and EntityService are self-documenting; contract derivable from 2 files | Code review of `entity.go` and `entity_service.go` |
| **Service Layer Consumer** | (Implicit in Scenarios 1-3) | P0, P2 | CLI commands continue to work; service accessor pattern remains consistent | Full `make test`; manual smoke test of `shark status advance`, `shark task note add` |
| **Test Author** | (Implicit in Scenario 2) | P0 | Cross-cutting logic testable with single parameterized test suite | Review `entity_service_test.go` for parameterized entity type tests |
| **Code Reviewer** | (Implicit in Scenario 4) | P1 | Clear interface contract; entity-specific services contain only unique logic | Code review at each feature boundary |

---

## 6. Quality Gate at Each Feature Boundary

Every feature (F01 through F05) must pass this gate before the next feature begins:

| Check | Command | Required Result |
|-------|---------|-----------------|
| Code formatting | `make fmt` | Zero changes needed |
| Static analysis | `make lint` | Zero warnings |
| Test suite | `make test` | 100% pass rate |
| Interface checks | `go build ./...` | Zero compile errors |
| Metric snapshot | See Section 2 | Recorded for trending |

If any check fails, the feature is not complete. Fix the issue and re-run the full gate.

---

## 7. UAT Exit Criteria

E21 is accepted when ALL of the following are true:

1. All 4 acceptance scenarios (Sections 1.1-1.4) pass
2. All 6 success metrics (Section 2) meet at least their minimum viable targets
3. P0 tests pass: zero behavioral regression across all feature boundaries
4. P1 tests pass: all 5 entity types compile with Entity interface
5. P2 tests pass: cross-cutting services use EntityRegistry with zero switch statements
6. Cross-epic integration verified: E15 compatibility confirmed, E19 path validated
7. `make fmt && make lint && make test` passes on the final state of the codebase

---

*See also*: [Success Metrics](./success-metrics.md), [Requirements](./requirements.md), [User Journeys](./user-journeys.md), [Personas](./personas.md)
