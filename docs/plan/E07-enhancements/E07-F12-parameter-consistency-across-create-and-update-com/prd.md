# Feature PRD: Parameter Consistency Across Create and Update Commands

**Feature Key**: E07-F12
**Epic**: E07 - Enhancements
**Status**: Draft
**Priority**: Medium
**Complexity**: M (Medium)

---

## Goal

### Problem

The Shark CLI has parameter inconsistencies between create and update commands:

1. **Epic create** lacks `--order` and `--notes` flags that would be useful for organizing epics and documenting creation context
2. **Feature create** lacks `--notes` flag for documenting feature creation rationale
3. **Inconsistent interface**: Update commands have parameters not available during creation, violating DRY principles and creating user confusion

This creates friction for users who want to:
- Set epic ordering at creation time (currently requires separate update call)
- Document why an epic or feature was created (no creation_notes field)
- Have a predictable, consistent CLI interface across all entity types

### Solution

Add missing parameters to create commands to achieve parity with update commands, following the principle that "anything you can update, you should be able to set at creation time."

**Specific changes:**
1. Add `--order` and `--notes` to `epic create`
2. Add `--notes` to `feature create`
3. Add corresponding support in `epic update` and `feature update` for symmetry
4. Update database schema to support order and creation notes
5. Ensure all changes are backward compatible

### Impact

**Expected outcomes:**
- Reduce friction for users setting up new epics and features
- Eliminate need for two-step create-then-update workflow
- Improve documentation and traceability of why entities were created
- Establish consistent parameter pattern for future commands

**Metrics:**
- 100% parameter parity between create and update commands (excluding immutable fields)
- Zero breaking changes to existing usage
- All unit and integration tests passing

---

## User Personas

### Persona 1: Product Manager / Technical Lead

**Profile**:
- **Role/Title**: Product Manager or Technical Lead coordinating development work
- **Experience Level**: Intermediate to advanced CLI users, familiar with project management
- **Key Characteristics**:
  - Manages multiple epics and features simultaneously
  - Needs to document decision-making context
  - Values efficiency and consistency in tooling

**Goals Related to This Feature**:
1. Create well-organized epics with clear ordering from the start
2. Document why epics and features were created for future reference
3. Minimize command-line steps when setting up new work

**Pain Points This Feature Addresses**:
- Currently must create epic, then immediately update it to set order
- No way to capture creation context/rationale in a structured way
- Inconsistent parameter availability creates mental overhead

**Success Looks Like**:
Can create a fully-configured epic in a single command: `shark epic create "Q1 Goals" --order=1 --notes="Strategic initiative from board meeting"`

---

### Persona 2: AI Development Agent

**Profile**:
- **Role/Title**: Automated agent creating and managing tasks
- **Experience Level**: Programmatic CLI user, requires predictable interfaces
- **Key Characteristics**:
  - Operates based on consistent patterns
  - Needs full control over entity properties at creation time
  - Cannot easily handle multi-step workflows

**Goals Related to This Feature**:
1. Create entities with all properties in single atomic operation
2. Rely on consistent parameter patterns across commands
3. Avoid coordination overhead of create-then-update sequences

**Pain Points This Feature Addresses**:
- Unpredictable parameter availability complicates agent logic
- Create-then-update pattern requires additional state tracking
- Lack of creation context makes debugging agent behavior difficult

**Success Looks Like**:
Can programmatically create epics and features with full configuration in one call, with reliable parameter contracts

---

## User Stories

### Must-Have Stories

**Story 1**: As a product manager, I want to set epic order at creation time so that I don't have to create-then-update.

**Acceptance Criteria**:
- [ ] `epic create` command accepts `--order <int>` flag
- [ ] Order value is stored in database epic.order_index column
- [ ] Order value is optional (defaults to NULL if not provided)
- [ ] Epic list can be sorted by order_index
- [ ] Backward compatible (existing scripts continue to work)

---

**Story 2**: As a product manager, I want to document why I created an epic or feature so that future team members understand the context.

**Acceptance Criteria**:
- [ ] `epic create` command accepts `--notes <string>` flag
- [ ] `feature create` command accepts `--notes <string>` flag
- [ ] Notes are stored in database (epic.creation_notes, feature.creation_notes)
- [ ] Notes are optional (defaults to NULL if not provided)
- [ ] Notes are visible when using `epic get` or `feature get`
- [ ] Notes can be updated later using `epic update` or `feature update`

---

**Story 3**: As a product manager, I want parameter consistency across create and update commands so that I have a predictable CLI interface.

**Acceptance Criteria**:
- [ ] All parameters in update commands (except status) are available in create commands
- [ ] All parameters in create commands are either in update commands or documented as immutable
- [ ] Help text clearly indicates which parameters are optional
- [ ] Documentation lists all available parameters for each command

---

### Should-Have Stories

**Story 4**: As a developer, I want clear documentation of parameter differences so that I understand the design rationale.

**Acceptance Criteria**:
- [ ] CLAUDE.md documents all epic/feature/task parameters
- [ ] CLAUDE.md explains why status is not available in create
- [ ] CLAUDE.md documents immutable fields (epic, feature for tasks)
- [ ] Parameter analysis document exists in feature folder

---

### Could-Have Stories

**Story 5**: As a developer, I want refactored parameter handling code so that future parameter additions are simpler.

**Acceptance Criteria**:
- [ ] Common parameter registration utilities exist
- [ ] Parameter definitions are DRY across commands
- [ ] Changes are backward compatible

**Note**: This is explicitly marked as "Could-Have" and should be considered for a separate feature if pursued.

---

## Requirements

### Functional Requirements

**Category: Epic Command Enhancement**

1. **REQ-F-001**: Epic Create Order Parameter
   - **Description**: `epic create` must accept optional `--order <int>` flag
   - **User Story**: Links to Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Flag registered in CLI
     - [ ] Order value passed to repository
     - [ ] Order value stored in database
     - [ ] Default behavior unchanged (NULL if not provided)

2. **REQ-F-002**: Epic Create Notes Parameter
   - **Description**: `epic create` must accept optional `--notes <string>` flag
   - **User Story**: Links to Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Flag registered in CLI
     - [ ] Notes value passed to repository
     - [ ] Notes value stored in database
     - [ ] Default behavior unchanged (NULL if not provided)

3. **REQ-F-003**: Epic Update Order Parameter
   - **Description**: `epic update` must accept optional `--order <int>` flag
   - **User Story**: Links to Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Flag registered in CLI
     - [ ] Order can be updated independently
     - [ ] -1 or unset means "no change" (like other update flags)

4. **REQ-F-004**: Epic Update Notes Parameter
   - **Description**: `epic update` must accept optional `--notes <string>` flag
   - **User Story**: Links to Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Flag registered in CLI
     - [ ] Notes can be updated independently
     - [ ] Empty string means "no change" (not clear notes)

**Category: Feature Command Enhancement**

5. **REQ-F-005**: Feature Create Notes Parameter
   - **Description**: `feature create` must accept optional `--notes <string>` flag
   - **User Story**: Links to Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Flag registered in CLI
     - [ ] Notes value passed to repository
     - [ ] Notes value stored in database
     - [ ] Default behavior unchanged (NULL if not provided)

6. **REQ-F-006**: Feature Update Notes Parameter
   - **Description**: `feature update` must accept optional `--notes <string>` flag
   - **User Story**: Links to Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Flag registered in CLI
     - [ ] Notes can be updated independently

**Category: Database Schema**

7. **REQ-F-007**: Epic Table Schema Changes
   - **Description**: Epic table must support order_index and creation_notes
   - **User Story**: Links to Stories 1, 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `order_index` column added (INTEGER, nullable)
     - [ ] `creation_notes` column added (TEXT, nullable)
     - [ ] Index created on order_index for query performance
     - [ ] Migration is idempotent (safe to run multiple times)

8. **REQ-F-008**: Feature Table Schema Changes
   - **Description**: Feature table must support creation_notes
   - **User Story**: Links to Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `creation_notes` column added (TEXT, nullable)
     - [ ] Migration is idempotent

**Category: Repository Layer**

9. **REQ-F-009**: Epic Repository Create Method
   - **Description**: `EpicRepository.Create` must accept and store order and notes
   - **User Story**: Links to Stories 1, 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Order field added to Epic model
     - [ ] CreationNotes field added to Epic model
     - [ ] Create method stores both fields
     - [ ] NULL handling correct

10. **REQ-F-010**: Epic Repository Update Methods
    - **Description**: `EpicRepository.Update` must handle order and notes updates
    - **User Story**: Links to Story 3
    - **Priority**: Must-Have
    - **Acceptance Criteria**:
      - [ ] Update method updates order when provided
      - [ ] Update method updates notes when provided
      - [ ] NULL/unset handling correct

11. **REQ-F-011**: Feature Repository Methods
    - **Description**: `FeatureRepository` must support creation_notes
    - **User Story**: Links to Story 2
    - **Priority**: Must-Have
    - **Acceptance Criteria**:
      - [ ] CreationNotes field added to Feature model
      - [ ] Create method stores notes
      - [ ] Update method updates notes

---

### Non-Functional Requirements

**Performance**

1. **REQ-NF-001**: Database Migration Performance
   - **Description**: Schema migration must complete within acceptable time
   - **Measurement**: Time to add columns on test database
   - **Target**: < 5 seconds on database with 10,000 epics/features
   - **Justification**: Migrations block application startup

**Compatibility**

2. **REQ-NF-002**: Backward Compatibility
   - **Description**: Existing scripts and usage must continue to work
   - **Implementation**: All new flags are optional with sensible defaults
   - **Testing**: Existing integration tests must pass without modification
   - **Risk Mitigation**: Prevents breaking changes for users

**Code Quality**

3. **REQ-NF-003**: Test Coverage
   - **Description**: New functionality must have comprehensive test coverage
   - **Target**: 90%+ coverage for new code paths
   - **Testing**: Unit tests + integration tests for CLI commands
   - **Justification**: Ensures reliability and prevents regressions

**Documentation**

4. **REQ-NF-004**: Documentation Completeness
   - **Description**: All new parameters must be documented
   - **Standard**: CLAUDE.md + help text + migration guide
   - **Testing**: Documentation review as part of PR
   - **Justification**: Ensures discoverability and adoption

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Create Epic with Order and Notes**
- **Given** I am a product manager setting up Q1 planning
- **When** I run `shark epic create "Q1 Goals" --order=1 --notes="Board directive from Dec meeting"`
- **Then** Epic is created with order_index=1
- **And** creation_notes contains "Board directive from Dec meeting"
- **And** `shark epic get` displays both order and notes

**Scenario 2: Create Feature with Notes**
- **Given** I am documenting a feature creation decision
- **When** I run `shark feature create --epic=E07 "OAuth Integration" --notes="User request from survey"`
- **Then** Feature is created with creation_notes
- **And** `shark feature get` displays notes

**Scenario 3: Update Epic Order**
- **Given** I have an existing epic E07
- **When** I run `shark epic update E07 --order=5`
- **Then** Epic order_index is updated to 5
- **And** Other epic properties remain unchanged

**Scenario 4: Backward Compatibility**
- **Given** I have an existing script using `shark epic create "Title"`
- **When** I run the script without new flags
- **Then** Epic is created successfully with NULL order and notes
- **And** No errors or warnings are shown

**Scenario 5: Epic List Ordering**
- **Given** I have created multiple epics with different order values
- **When** I run `shark epic list --sort-by=order`
- **Then** Epics are displayed sorted by order_index (NULLs last)

---

## Out of Scope

### Explicitly Excluded

1. **Parameter Refactoring to Shared Modules**
   - **Why**: Increases scope significantly; can be done as separate enhancement
   - **Future**: Tracked as "Could-Have" story; may be separate feature
   - **Workaround**: Current approach (duplicated flag definitions) works fine

2. **Adding Order to Features**
   - **Why**: Features already have execution_order field
   - **Future**: Not planned (execution_order sufficient)
   - **Workaround**: Use existing execution_order

3. **Adding Notes to Task Create**
   - **Why**: Tasks already have description field; notes less critical
   - **Future**: Can revisit if user feedback indicates need
   - **Workaround**: Use description field

4. **Append-Only Notes**
   - **Why**: Adds complexity; simple field replacement sufficient for creation context
   - **Future**: If needed, use task_history pattern (separate table)
   - **Workaround**: Single creation_notes field is replaceable via update

5. **Epic Order Per-Status**
   - **Why**: Adds complexity; global order sufficient
   - **Future**: Can add if use case emerges
   - **Workaround**: Use global order_index

---

### Alternative Approaches Rejected

**Alternative 1: Auto-generate Epic Order**
- **Description**: Automatically assign order based on creation time
- **Why Rejected**: Users need explicit control over ordering; auto-numbering doesn't match product management workflows

**Alternative 2: Use Execution Order for Epics**
- **Description**: Reuse execution_order field name across all entities
- **Why Rejected**: execution_order has specific meaning for features/tasks (implementation sequencing); epic ordering is different (strategic priority)

**Alternative 3: Separate Notes Table**
- **Description**: Create epic_notes and feature_notes tables for append-only notes
- **Why Rejected**: Over-engineering for creation context; single field sufficient; can add later if needed

---

## Success Metrics

### Primary Metrics

1. **Parameter Parity**
   - **What**: Percentage of update parameters available in create
   - **Target**: 100% (excluding immutable fields and status)
   - **Timeline**: By feature completion
   - **Measurement**: Manual audit of command flags

2. **Backward Compatibility**
   - **What**: Existing integration tests pass without modification
   - **Target**: 100% pass rate
   - **Timeline**: Throughout development
   - **Measurement**: CI test results

3. **Test Coverage**
   - **What**: Code coverage for new functionality
   - **Target**: 90%+ coverage
   - **Timeline**: Before feature completion
   - **Measurement**: `make test-coverage` report

---

### Secondary Metrics

- **Documentation Completeness**: All new flags documented in CLAUDE.md and help text
- **User Adoption**: Track usage of new flags in logs (if telemetry available)
- **Migration Success**: Zero migration failures reported

---

## Dependencies & Integrations

### Dependencies

- **SQLite Auto-Migration System**: Required for adding new columns (internal/db/db.go)
- **Repository Pattern**: Changes depend on repository layer (internal/repository/)
- **Cobra CLI Framework**: Flag registration depends on Cobra API

### Integration Requirements

- **No External Systems**: All changes are internal to Shark

---

## Compliance & Security Considerations

- **No Regulatory Impact**: Changes are CLI parameter additions only
- **No Security Impact**: No authentication, authorization, or data protection changes
- **No Audit Impact**: No changes to logging or audit trails (though notes could be used for documentation)

---

## Implementation Plan

### Phase 1: Database Migration (T-E07-F12-001)
- Add order_index and creation_notes columns to epics table
- Add creation_notes column to features table
- Create indexes
- Test migration on backup database

### Phase 2: Model Updates (T-E07-F12-002)
- Add Order field to Epic model
- Add CreationNotes field to Epic model
- Add CreationNotes field to Feature model
- Update model validation

### Phase 3: Repository Layer (T-E07-F12-003)
- Update EpicRepository.Create to handle order and notes
- Update EpicRepository.Update to handle order and notes
- Update FeatureRepository.Create to handle notes
- Update FeatureRepository.Update to handle notes

### Phase 4: CLI Commands (T-E07-F12-004)
- Add flags to epic create command
- Add flags to epic update command
- Add flags to feature create command
- Add flags to feature update command
- Wire flags to repository calls

### Phase 5: Testing (T-E07-F12-005)
- Write unit tests for repository methods
- Write integration tests for CLI commands
- Verify backward compatibility
- Run full test suite

### Phase 6: Documentation (T-E07-F12-006)
- Update CLAUDE.md with new parameters
- Update CLI help text
- Create migration guide if needed
- Update parameter analysis document

---

## Risks & Mitigation

### Risk 1: Database Migration Failures
**Probability**: Low
**Impact**: High
**Mitigation**: Use existing auto-migration system; test on backup database first; migrations are idempotent

### Risk 2: Breaking Existing Scripts
**Probability**: Low
**Impact**: High
**Mitigation**: All new flags are optional; extensive backward compatibility testing

### Risk 3: Inconsistent State Between Old and New Epics
**Probability**: Medium
**Impact**: Low
**Mitigation**: NULL values handled gracefully; old epics work fine without order/notes

---

## Open Questions

✅ **Resolved**: Should epic order be global or per-status?
- **Decision**: Global order (simpler, matches existing patterns)

✅ **Resolved**: Should notes be append-only or replaceable?
- **Decision**: Replaceable (simpler, sufficient for creation context)

✅ **Resolved**: Should we add notes to task create?
- **Decision**: No (out of scope; tasks have description field)

---

## References

- **Parameter Analysis**: `parameter-analysis.md` (this feature folder)
- **Epic PRD**: `E07-enhancements/epic.md`
- **CLI Source**: `internal/cli/commands/epic.go`, `feature.go`
- **Repository Source**: `internal/repository/epic_repository.go`, `feature_repository.go`
- **Database Schema**: `internal/db/db.go`
- **CLAUDE.md**: Root level project documentation

---

*Last Updated*: 2026-01-01
