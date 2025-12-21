---
feature_key: E07-F07-epic-index-discovery-integration
epic_key: E07
title: Epic Index Discovery Integration
description: Integrate the discovery package with the sync command to enable epic-index.md parsing and conflict resolution
---

# Epic Index Discovery Integration

**Feature Key**: E07-F07-epic-index-discovery-integration

---

## Epic

- **Epic PRD**: [enhancements](../../epic.md)

---

## Goal

### Problem

The shark-task-manager has a fully implemented `internal/discovery` package that can parse `epic-index.md` files, scan folder structures, detect conflicts between the two sources, and resolve them using configurable strategies. However, this critical functionality is completely disconnected from the CLI and unusable by end users. The `shark sync` command only scans task files and ignores epic-index.md entirely, leaving a significant gap in the tool's ability to maintain consistency between documentation indices and actual folder structures.

Developers and teams who maintain epic-index.md files (a common pattern for organizing project documentation) have no way to leverage this index for automated epic/feature discovery, conflict detection, or database synchronization. This forces manual reconciliation between the index file and folder structure, leading to inconsistencies, broken references, and outdated documentation.

### Solution

Wire the discovery package into the `shark sync` command by adding new CLI flags and integration logic. Users will be able to:
- Specify an `--index` flag pointing to their epic-index.md file
- Choose a conflict resolution strategy (`--discovery-strategy`) to control how conflicts between index and folders are resolved
- Run discovery scans that populate the database with epics and features from both the index file and folder structure
- Receive detailed conflict reports showing mismatches and actionable suggestions
- Optionally auto-create missing epics/features based on the chosen strategy

### Impact

**Expected Outcomes:**
- Developers can synchronize 100% of their documentation structure (epics, features, tasks) in a single `shark sync` command
- Teams maintaining epic-index.md files can detect and resolve conflicts automatically, reducing manual reconciliation time by 90%
- Database accuracy improves as epics and features are discoverable from both index files and folder structures
- Documentation quality increases through automated conflict detection that catches broken links, missing folders, and relationship mismatches

**Measurable Metrics:**
- 100% of discovery package functionality accessible via CLI within 2 weeks
- Reduce documentation drift (index vs folders) detection time from manual hours to < 5 seconds automated
- Enable 3 distinct conflict resolution strategies (index-precedence, folder-precedence, merge) with clear behavior documentation

---

## User Personas

### Persona 1: Software Developer (Individual Contributor)

**Profile**:
- **Role/Title**: Backend/Frontend Developer working on multi-epic projects
- **Experience Level**: 2-5 years, comfortable with CLI tools and markdown documentation
- **Key Characteristics**:
  - Uses shark to track task progress across multiple features
  - Maintains local epic-index.md to organize documentation
  - Works independently but needs to sync with team's documentation structure
  - Values automation and consistency

**Goals Related to This Feature**:
1. Sync local epic/feature structure to database without manual entry
2. Catch documentation drift early (index references missing folders)
3. Ensure database matches project documentation organization

**Pain Points This Feature Addresses**:
- Currently must manually create epics/features in database even when epic-index.md exists
- No way to validate that epic-index.md links point to real folders
- Broken documentation references aren't caught until someone manually reviews

**Success Looks Like**:
Runs `shark sync --index=docs/plan/epic-index.md` once and automatically populates database with all epics/features from both the index and folder structure. Receives clear conflict reports if there are mismatches, enabling quick fixes.

---

### Persona 2: Technical Lead / Architect

**Profile**:
- **Role/Title**: Senior Engineer or Tech Lead coordinating multiple developers
- **Experience Level**: 5+ years, responsible for architecture decisions and documentation standards
- **Key Characteristics**:
  - Maintains canonical epic-index.md as source of truth for project structure
  - Reviews and approves changes to epic/feature organization
  - Needs to enforce consistency across team's documentation
  - Values clear conflict resolution policies

**Goals Related to This Feature**:
1. Use epic-index.md as authoritative source for project structure
2. Automatically detect when developers create folders not listed in index
3. Enforce documentation standards through automated validation

**Pain Points This Feature Addresses**:
- Can't enforce epic-index.md as source of truth programmatically
- Manual review required to catch undocumented epics/features
- No automated way to validate that folder structure matches index

**Success Looks Like**:
Sets `--discovery-strategy=index-precedence` in CI/CD pipeline to enforce that only epics/features listed in epic-index.md are imported. Pipeline fails with clear error report if folders exist that aren't documented in the index, ensuring documentation discipline.

---

### Persona 3: Product Manager / Documentation Owner

**Profile**:
- **Role/Title**: Product Manager or Technical Writer maintaining product documentation
- **Experience Level**: Moderate technical proficiency, strong documentation skills
- **Key Characteristics**:
  - Creates and maintains epic-index.md for stakeholder visibility
  - May not directly manage folder structure (developers do)
  - Needs to ensure documentation reflects actual implementation
  - Values bidirectional sync and conflict warnings

**Goals Related to This Feature**:
1. Ensure epic-index.md accurately reflects implemented features
2. Get notified when developers create features not documented in index
3. Merge documentation (index) with implementation reality (folders)

**Pain Points This Feature Addresses**:
- Epic-index.md gets out of sync with actual folder structure
- No automated detection of missing documentation for new features
- Can't easily reconcile what's documented vs what's implemented

**Success Looks Like**:
Runs `shark sync --index=docs/plan/epic-index.md --discovery-strategy=merge` weekly. Receives conflict report showing features in folders but not in index (need documentation) and features in index but missing folders (need implementation or cleanup). Uses report to update epic-index.md accordingly.

---

## User Stories

### Must-Have Stories

**Story 1**: As a software developer, I want to run `shark sync` with an `--index` flag pointing to epic-index.md so that epics and features are automatically discovered from the index file and synchronized to the database.

**Acceptance Criteria**:
- [ ] `shark sync --index=path/to/epic-index.md` flag is recognized and parsed
- [ ] IndexParser reads epic-index.md and extracts all epic and feature links
- [ ] Discovered epics and features are written to the database (respecting dry-run mode)
- [ ] Sync report shows counts of epics/features discovered from index
- [ ] Invalid index file path returns clear error message with suggestion

**Story 2**: As a technical lead, I want to choose a conflict resolution strategy via `--discovery-strategy` flag so that I can control how conflicts between epic-index.md and folder structure are handled.

**Acceptance Criteria**:
- [ ] `--discovery-strategy` flag accepts values: `index-precedence`, `folder-precedence`, `merge`
- [ ] `index-precedence` imports only items from epic-index.md, warns about folder-only items
- [ ] `folder-precedence` imports only items from folder structure, warns about index-only items
- [ ] `merge` imports items from both sources, creating merged records
- [ ] Invalid strategy value returns error listing valid options
- [ ] Default strategy is `merge` if not specified

**Story 3**: As a developer, I want to see detailed conflict reports during sync so that I can identify and fix mismatches between epic-index.md and folder structure.

**Acceptance Criteria**:
- [ ] Sync report includes dedicated "Conflicts" section
- [ ] Each conflict shows: type, key, path, and actionable suggestion
- [ ] Conflict types include: epic_index_only, epic_folder_only, feature_index_only, feature_folder_only, relationship_mismatch
- [ ] Suggestions are specific (e.g., "Create folder for epic E04 or remove from epic-index.md")
- [ ] JSON output mode includes structured conflict data for scripting

**Story 4**: As a developer, I want epics and features discovered from epic-index.md to include their titles and metadata so that the database has rich information beyond just keys.

**Acceptance Criteria**:
- [ ] Epic title extracted from link text in epic-index.md
- [ ] Feature title extracted from link text in epic-index.md
- [ ] Titles are stored in database during sync
- [ ] If epic.md or prd.md exists, metadata from those files takes precedence over link text
- [ ] Missing metadata fields are handled gracefully (nullable in database)

**Story 5**: As a developer, I want the sync command to scan both folder structure and epic-index.md in a single operation so that I get complete project visibility without running multiple commands.

**Acceptance Criteria**:
- [ ] When `--index` is provided, both folder scan and index parse run in single sync
- [ ] FolderScanner and IndexParser operate in parallel or sequentially (implementation choice)
- [ ] Results from both scans are merged according to selected strategy
- [ ] Sync report shows counts from both sources (epics_from_index, epics_from_folders, etc.)
- [ ] Performance is acceptable (< 5 seconds for projects with 50 epics, 200 features)

---

### Should-Have Stories

**Story 6**: As a technical writer, I want to validate my epic-index.md file without making database changes so that I can check for broken links and conflicts before committing.

**Acceptance Criteria**:
- [ ] `shark sync --index=epic-index.md --dry-run` runs discovery without database writes
- [ ] Dry-run report shows what would be imported/skipped
- [ ] Conflicts are detected and reported in dry-run mode
- [ ] Validation errors (malformed links, invalid paths) are shown

**Story 7**: As a developer, I want related documents discovered from feature folders to be cataloged so that I can see all documentation associated with each feature.

**Acceptance Criteria**:
- [ ] FolderScanner catalogs related docs (e.g., architecture.md, wireframes.md) in feature folders
- [ ] Related docs are stored in database or included in sync report
- [ ] Related docs exclude prd.md (already tracked separately)
- [ ] Sync report shows count of related documents cataloged

**Story 8**: As a team lead, I want the sync command to respect validation levels (strict, balanced, permissive) so that I can control how strictly folder naming conventions are enforced.

**Acceptance Criteria**:
- [ ] `--validation-level` flag accepts: strict, balanced, permissive
- [ ] Strict mode requires exact E##-F## naming conventions
- [ ] Balanced mode (default) uses patterns from .sharkconfig.json
- [ ] Permissive mode accepts any reasonable folder structure
- [ ] Invalid folder names are reported with suggestions based on validation level

---

### Could-Have Stories

**Story 9**: As a developer, I want to auto-create missing epics and features based on discovery results so that manual database setup is eliminated.

**Acceptance Criteria**:
- [ ] `--create-missing` flag works with discovery package
- [ ] Missing epics detected in epic-index.md or folders are created in database
- [ ] Missing features are created with proper epic relationships
- [ ] Auto-created records have default status (e.g., "draft")
- [ ] Sync report shows count of auto-created epics/features

**Story 10**: As a product manager, I want to generate an epic-index.md file from the current database state so that I can bootstrap documentation from existing structure.

**Acceptance Criteria**:
- [ ] `shark generate epic-index` command creates epic-index.md from database
- [ ] Generated file follows standard markdown format with links to epic/feature folders
- [ ] Epics are sorted by key (E01, E02, etc.)
- [ ] Features are nested under their parent epics
- [ ] Output path is configurable via flag

---

### Edge Case & Error Stories

**Error Story 1**: As a developer, when I provide an invalid or missing epic-index.md path, I want to see a clear error message so that I can correct the path.

**Acceptance Criteria**:
- [ ] Missing index file returns: "Error: epic-index.md not found at path: {path}. Please check the file exists."
- [ ] Error includes suggestion: "Use --index=/path/to/epic-index.md or create the file first."
- [ ] Exit code is non-zero (1) for CI/CD integration

**Error Story 2**: As a developer, when epic-index.md contains malformed links or invalid epic keys, I want specific validation errors so that I can fix the formatting.

**Acceptance Criteria**:
- [ ] Malformed markdown links are reported with line/context information
- [ ] Invalid epic keys (e.g., "E4" instead of "E04") are flagged with correction suggestion
- [ ] Broken relative paths are reported with expected path format
- [ ] Sync continues with valid entries, reports skipped invalid entries

**Error Story 3**: As a user, when there are conflicts and I choose `index-precedence` strategy but folders referenced in index don't exist, I want the sync to fail with actionable guidance so that I can resolve the structural issues.

**Acceptance Criteria**:
- [ ] `index-precedence` with missing folders returns error (not just warning)
- [ ] Error lists all missing folders with expected paths
- [ ] Suggestion: "Create missing folders or switch to --discovery-strategy=merge"
- [ ] Option to force import with `--force` flag (creates database records despite missing folders)

---

## Requirements

### Functional Requirements

**Category: CLI Integration**

1. **REQ-F-001**: Index File Discovery
   - **Description**: The sync command must accept an `--index` flag that specifies the path to an epic-index.md file to parse for epic and feature discovery
   - **User Story**: Links to Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `--index` flag is defined in sync command flags
     - [ ] Path validation confirms file exists before parsing
     - [ ] Relative and absolute paths are both supported
     - [ ] Default value is empty (discovery package not invoked unless flag provided)

2. **REQ-F-002**: Conflict Resolution Strategy Selection
   - **Description**: Users must be able to select a conflict resolution strategy via `--discovery-strategy` flag with three options: index-precedence, folder-precedence, merge
   - **User Story**: Links to Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Flag accepts enum values: index-precedence, folder-precedence, merge
     - [ ] Default strategy is merge
     - [ ] Strategy is passed to ConflictResolver during sync
     - [ ] Invalid strategy returns error with valid options listed

3. **REQ-F-003**: Discovery Report Integration
   - **Description**: Sync report must include discovery-specific metrics (epics from index, features from folders, conflicts detected) and detailed conflict information
   - **User Story**: Links to Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Sync report struct extended with discovery fields
     - [ ] Conflict array includes all detected conflicts with type, key, path, suggestion
     - [ ] Text output format displays conflicts in dedicated section
     - [ ] JSON output includes structured conflict data

4. **REQ-F-004**: Bidirectional Discovery Workflow
   - **Description**: When `--index` flag is provided, both IndexParser and FolderScanner must execute, and their results must be merged according to the selected strategy
   - **User Story**: Links to Story 5
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] IndexParser.Parse() called with index file path
     - [ ] FolderScanner.Scan() called with docs root path
     - [ ] ConflictDetector.Detect() identifies all conflicts
     - [ ] ConflictResolver.Resolve() applies selected strategy
     - [ ] Resolved epics/features are written to database

**Category: Data Integration**

5. **REQ-F-005**: Epic Discovery and Import
   - **Description**: Epics discovered from epic-index.md must be imported into the database with their key, title, and file path (if epic.md exists)
   - **User Story**: Links to Story 4
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Epic key extracted from index link path (e.g., "E04")
     - [ ] Epic title extracted from link text
     - [ ] epic.md path stored if file exists in folder
     - [ ] Database epic table updated/inserted with discovered data
     - [ ] Frontmatter metadata from epic.md takes precedence over link text

6. **REQ-F-006**: Feature Discovery and Import
   - **Description**: Features discovered from epic-index.md or folder structure must be imported with their key, epic relationship, title, and prd.md path (if exists)
   - **User Story**: Links to Story 4
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Feature key extracted correctly (e.g., "E04-F07")
     - [ ] Parent epic relationship maintained (epic_key field)
     - [ ] Feature title extracted from link text or prd.md frontmatter
     - [ ] prd.md path stored if file exists
     - [ ] Related documents cataloged (if FolderScanner finds them)

7. **REQ-F-007**: Conflict Detection and Reporting
   - **Description**: All five conflict types (epic_index_only, epic_folder_only, feature_index_only, feature_folder_only, relationship_mismatch) must be detected and reported with actionable suggestions
   - **User Story**: Links to Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] ConflictDetector.Detect() called with index and folder results
     - [ ] All five conflict types are checked
     - [ ] Each conflict includes suggested resolution action
     - [ ] Conflicts are included in sync report output
     - [ ] JSON output provides machine-readable conflict data

**Category: Validation**

8. **REQ-F-008**: Index File Validation
   - **Description**: Epic-index.md must be validated for correct markdown format, valid epic/feature keys, and resolvable relative paths before import
   - **User Story**: Links to Error Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Markdown link regex validates format: `[text](path)`
     - [ ] Epic keys match pattern: `E\d{2}` or special types (tech-debt, bugs)
     - [ ] Feature keys match pattern: `E\d{2}-F\d{2}`
     - [ ] Malformed entries logged as warnings/errors
     - [ ] Validation errors prevent import of invalid entries

9. **REQ-F-009**: Validation Level Support
   - **Description**: The sync command must support three validation levels (strict, balanced, permissive) that control folder naming convention enforcement
   - **User Story**: Links to Story 8
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] `--validation-level` flag defined with enum: strict, balanced, permissive
     - [ ] ValidationLevel passed to discovery components
     - [ ] Strict enforces E##-F## conventions exactly
     - [ ] Balanced uses .sharkconfig.json patterns
     - [ ] Permissive accepts any reasonable structure

---

### Non-Functional Requirements

**Performance**

1. **REQ-NF-001**: Discovery Performance
   - **Description**: Discovery operations (parsing index + scanning folders + resolving conflicts) must complete within acceptable time limits for typical projects
   - **Measurement**: Execution time from start of discovery to database write
   - **Target**:
     - Small projects (10 epics, 50 features): < 1 second
     - Medium projects (50 epics, 200 features): < 5 seconds
     - Large projects (100 epics, 500 features): < 15 seconds
   - **Justification**: Developers run sync frequently; slow operations reduce adoption

2. **REQ-NF-002**: Memory Efficiency
   - **Description**: Discovery operations must not consume excessive memory when processing large epic-index.md files or deep folder hierarchies
   - **Measurement**: Peak memory usage during sync operation
   - **Target**: < 100MB for projects with 500 features and 1000 related documents
   - **Justification**: Tool should run on developer workstations without resource constraints

**Reliability**

3. **REQ-NF-010**: Atomic Database Operations
   - **Description**: Discovery imports must be atomic - either all discovered entities are imported or none are (on error)
   - **Measurement**: Database state after partial failure
   - **Target**: 100% transaction rollback on any import error
   - **Justification**: Partial imports create inconsistent database state

4. **REQ-NF-011**: Error Handling
   - **Description**: All error conditions (missing files, malformed data, database failures) must be caught and reported with actionable messages; no panics or crashes
   - **Measurement**: Error handling code coverage and manual testing
   - **Target**: 100% of error paths return error values, 0 panics in production code
   - **Justification**: CLI tool stability is critical for developer trust

**Usability**

5. **REQ-NF-020**: Clear Output Formatting
   - **Description**: Sync reports must clearly distinguish discovery results from task sync results with visual separation and labeled sections
   - **Measurement**: User comprehension testing and feedback
   - **Target**: Users can identify epic/feature counts and conflicts within 5 seconds of reading output
   - **Justification**: Information overload reduces usefulness of reports

6. **REQ-NF-021**: Flag Naming Consistency
   - **Description**: New flags must follow existing shark CLI naming conventions (kebab-case, descriptive names, consistent prefix patterns)
   - **Measurement**: Code review checklist compliance
   - **Target**: 100% consistency with existing flags in sync command
   - **Justification**: Consistent CLI improves learnability and reduces errors

**Compatibility**

7. **REQ-NF-030**: Backward Compatibility
   - **Description**: Existing `shark sync` behavior must remain unchanged when `--index` flag is not provided; discovery is opt-in
   - **Measurement**: Regression testing of sync command without new flags
   - **Target**: 100% of existing sync tests pass without modification
   - **Justification**: Breaking changes alienate existing users

8. **REQ-NF-031**: Configuration File Support
   - **Description**: Discovery must respect patterns defined in .sharkconfig.json for epic/feature/task matching
   - **Measurement**: Integration tests with custom patterns
   - **Target**: All custom patterns from config are correctly applied during discovery
   - **Justification**: Users have customized their patterns; discovery must honor them

**Maintainability**

9. **REQ-NF-040**: Code Reusability
   - **Description**: Discovery package integration must not duplicate existing code; reuse ConflictDetector, IndexParser, FolderScanner as-is or with minimal changes
   - **Measurement**: Code review and duplicate code analysis
   - **Target**: < 5% code duplication between discovery package and sync command
   - **Justification**: Reduces maintenance burden and bug surface area

10. **REQ-NF-041**: Test Coverage
    - **Description**: All new CLI integration code must have comprehensive unit and integration tests
    - **Measurement**: Code coverage reports
    - **Target**: > 80% line coverage for new code, 100% coverage of error paths
    - **Justification**: High test coverage ensures reliable CLI behavior

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Successful Discovery with Merge Strategy**
- **Given** a project with epic-index.md containing 5 epics and folder structure containing 6 epics (1 not in index)
- **When** user runs `shark sync --index=docs/plan/epic-index.md --discovery-strategy=merge`
- **Then** database contains all 6 epics (5 from index + 1 folder-only marked as merged)
- **And** sync report shows: "Epics from Index: 5, Epics from Folders: 6, Conflicts: 1"
- **And** conflict report shows: "Epic E07 found in folders but not in epic-index.md. Suggestion: Add epic E07 to epic-index.md or use folder-precedence strategy"

**Scenario 2: Index Precedence with Missing Folders**
- **Given** epic-index.md references epic E08 but folder docs/plan/E08-epic-name/ does not exist
- **When** user runs `shark sync --index=docs/plan/epic-index.md --discovery-strategy=index-precedence`
- **Then** sync fails with error: "Epic E08 referenced in index but folder not found at docs/plan/E08-epic-name/"
- **And** suggestion displayed: "Create folder for epic E08 or remove from epic-index.md"
- **And** exit code is 1 (failure)

**Scenario 3: Dry-Run Discovery Validation**
- **Given** epic-index.md with 3 malformed links and 7 valid links
- **When** user runs `shark sync --index=docs/plan/epic-index.md --dry-run`
- **Then** no database writes occur
- **And** sync report shows: "Valid entries: 7, Invalid entries: 3, Conflicts: 0"
- **And** detailed errors displayed for each malformed link with line information
- **And** user can fix errors before running actual sync

**Scenario 4: Related Documents Discovery**
- **Given** feature folder E04-F07-init-sync containing prd.md, architecture.md, and wireframes.md
- **When** user runs `shark sync --index=docs/plan/epic-index.md` (with FolderScanner enabled)
- **Then** feature E04-F07 imported with prd_path = "docs/plan/.../prd.md"
- **And** related_docs array contains ["architecture.md", "wireframes.md"] (excluding prd.md)
- **And** sync report shows: "Related Docs Cataloged: 2"

**Scenario 5: No Index Provided (Backward Compatibility)**
- **Given** existing project using shark sync without discovery package
- **When** user runs `shark sync` (no --index flag)
- **Then** sync operates exactly as before (task files only)
- **And** discovery package is not invoked
- **And** no discovery metrics appear in sync report
- **And** all existing tests pass without modification

---

## Out of Scope

### Explicitly Excluded

1. **Automatic epic-index.md Generation from Database**
   - **Why**: While valuable (Story 10 "Could-Have"), it's a separate feature that requires different UX considerations (merge conflicts when file exists, template selection, etc.). This feature focuses on reading epic-index.md, not writing it.
   - **Future**: May be added as `shark generate epic-index` command in future enhancement
   - **Workaround**: Users can manually maintain epic-index.md or use scripts

2. **GUI or Web Interface for Conflict Resolution**
   - **Why**: This is a CLI tool; all interaction is via terminal. Adding GUI requires significant complexity (web server, frontend, etc.) outside project scope.
   - **Future**: If shark evolves to include web UI, conflict resolution could be enhanced
   - **Workaround**: Text-based conflict reports with clear suggestions are sufficient for CLI users

3. **Bi-Directional Sync (Database → epic-index.md Updates)**
   - **Why**: Complexity of merging changes bidirectionally (what if both database and index changed?) is substantial. One-way sync (index → database) is simpler and covers primary use case.
   - **Future**: Could be added if strong user demand emerges
   - **Workaround**: Users manually update epic-index.md when making database changes via other commands

4. **Support for Multiple Index Files**
   - **Why**: Most projects have one canonical epic-index.md. Supporting multiple indices (e.g., per-team indices) adds complexity in conflict resolution and merging logic.
   - **Future**: Could be reconsidered if multi-team projects require this
   - **Workaround**: Users can run sync multiple times with different --index paths (results will merge)

5. **Custom Conflict Resolution Scripts/Hooks**
   - **Why**: While advanced users might want programmable conflict resolution, building a plugin system is significant engineering effort outside this feature's scope.
   - **Future**: If shark develops plugin architecture, this could be added
   - **Workaround**: Users can choose merge strategy and manually handle conflicts, or wrap shark sync with custom scripts

6. **Watch Mode for Continuous Sync**
   - **Why**: Watching files for changes and auto-syncing introduces complexity (file watcher implementation, handling rapid changes, etc.) and isn't core to discovery integration.
   - **Future**: Could be added as separate `shark sync --watch` enhancement
   - **Workaround**: Users can run sync manually or via cron/scheduled tasks

---

### Alternative Approaches Rejected

**Alternative 1: Separate `shark discover` Command**
- **Description**: Create a new top-level command `shark discover` instead of integrating into `shark sync`
- **Why Rejected**: Discovery is fundamentally part of synchronization - you discover entities in order to sync them to the database. Splitting into two commands creates confusion ("do I run discover then sync? or just sync?") and forces users to run multiple commands. Integrated approach is more intuitive.

**Alternative 2: Always Parse epic-index.md if it Exists**
- **Description**: Auto-detect epic-index.md in docs root and parse it automatically without requiring --index flag
- **Why Rejected**: Implicit behavior can surprise users. Some projects may have epic-index.md for documentation purposes but not want it to drive database structure. Explicit opt-in via flag gives users control and makes behavior predictable.

**Alternative 3: Store Conflict Resolution Strategy in .sharkconfig.json**
- **Description**: Make --discovery-strategy a config file setting rather than CLI flag
- **Why Rejected**: Conflict strategy often varies by context (CI/CD might use strict index-precedence, while local development uses merge). CLI flags provide flexibility for different use cases without editing config file. Config could be added later as default with flag override.

**Alternative 4: Implement Only index-precedence Strategy Initially**
- **Description**: Ship with single strategy to reduce complexity, add others later
- **Why Rejected**: The three strategies (index-precedence, folder-precedence, merge) serve distinct user needs (different personas/use cases). Shipping incomplete functionality forces users into one workflow that may not fit their needs. ConflictResolver already implements all strategies; wiring all three is minimal additional effort.

---

## Success Metrics

### Primary Metrics

1. **Discovery Adoption Rate**
   - **What**: Percentage of shark users who use --index flag in sync commands (tracked via telemetry if implemented, or GitHub issue feedback)
   - **Target**: 40% of active users within 3 months of release
   - **Timeline**: 3 months post-release
   - **Measurement**: Anonymous telemetry (opt-in) or user surveys

2. **Conflict Detection Accuracy**
   - **What**: Percentage of actual index/folder mismatches that are correctly detected and reported
   - **Target**: 100% detection rate for all five conflict types
   - **Timeline**: Validated during QA before release
   - **Measurement**: Integration tests with known conflict scenarios

3. **Time to Sync Completion**
   - **What**: Average time for `shark sync --index` to complete for medium-sized projects (50 epics, 200 features)
   - **Target**: < 5 seconds (90th percentile)
   - **Timeline**: Validated during performance testing
   - **Measurement**: Benchmarks in CI/CD

---

### Secondary Metrics

- **Documentation Drift Reduction**: Decrease in GitHub issues related to "epic-index.md out of sync" or "broken documentation links" (target: 50% reduction in 6 months)
- **Feature Request Satisfaction**: Closes existing feature requests for epic-index.md support (target: 100% of related requests)
- **Error Rate**: Percentage of sync operations that fail due to discovery errors (target: < 2% after fixing initial bugs)

---

## Dependencies & Integrations

### Dependencies

- **Internal Discovery Package**: `internal/discovery` (already implemented) - IndexParser, FolderScanner, ConflictDetector, ConflictResolver
- **Internal Sync Package**: `internal/sync/engine.go` - SyncEngine must be extended to optionally invoke discovery workflow
- **Internal Patterns Package**: `internal/patterns` - PatternRegistry used for folder name validation
- **CLI Command**: `internal/cli/commands/sync.go` - New flags and orchestration logic added here
- **Database Repositories**: `internal/repository` - EpicRepository, FeatureRepository for writing discovered entities

### Integration Requirements

- **Sync Command Extension**: Add `--index`, `--discovery-strategy`, `--validation-level` flags to existing sync command
- **SyncReport Extension**: Extend `internal/sync/types.go` SyncReport struct with discovery metrics (epics_from_index, conflicts, etc.)
- **Configuration File**: Read .sharkconfig.json patterns for validation (already supported by patterns package)
- **Reporting Package**: `internal/reporting` may need updates to format discovery-specific output (conflicts section, discovery metrics)

---

## Compliance & Security Considerations

**Regulatory**: N/A - This is a local CLI tool for developer productivity; no personal data, no regulatory requirements.

**Data Protection**:
- File paths and project structure information are local to user's machine
- Database is SQLite stored locally; no external data transmission
- No encryption required (local filesystem trust model)

**Audit**:
- Sync operations already log to stdout/stderr
- Discovery operations should log: index file parsed, epics/features discovered, conflicts detected
- Existing shark logging patterns followed (no new audit requirements)

**Security Concerns**:
- Path traversal: Validate that --index path doesn't escape project directory (validate with filepath.Clean)
- SQL injection: Use parameterized queries (already standard in repository layer)
- File permissions: Respect OS file permissions when reading epic-index.md and scanning folders

---

*Last Updated*: 2025-12-18
