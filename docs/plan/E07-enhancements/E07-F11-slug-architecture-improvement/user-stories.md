# User Stories: Slug Architecture Improvement

**Feature**: E07-F11 - Slug Architecture Improvement
**Epic**: E07 - Database & State Management Enhancements
**Created**: 2025-12-30

---

## Story Template

Each story follows this format:

```
## Story: [Story Title]

**ID**: S-[Phase]-[Number]
**Phase**: [1-6]
**Priority**: [P0|P1|P2]

### User Story
As a [user type]
I want [capability]
So that [benefit]

### Acceptance Criteria

#### AC1: [Criterion Name]
Given [context]
When [action]
Then [expected outcome]

### Dependencies
- [List prerequisite stories or external dependencies]

### Estimated Size
[XS|S|M|L|XL] - [Brief justification]
```

---

# Phase 1: Database Schema (P0)

## Story 1.1: Add Slug Columns to Database

**ID**: S-1-1
**Phase**: 1
**Priority**: P0

### User Story
As a **system architect**
I want **slug columns added to epics, features, and tasks tables**
So that **slugs can be stored as database fields instead of being computed on-the-fly**

### Acceptance Criteria

#### AC1: Slug Columns Added
Given the existing database schema
When the migration script is executed
Then slug columns are added to epics, features, and tasks tables
And the columns are of type TEXT
And the columns are nullable (to support existing records)

#### AC2: Indexes Created
Given the slug columns exist
When the migration script completes
Then indexes are created on epics(slug), features(slug), and tasks(slug)
And the indexes are named idx_epics_slug, idx_features_slug, idx_tasks_slug

#### AC3: Migration is Idempotent
Given the migration script exists
When the migration is run multiple times
Then no errors occur
And columns are not duplicated
And indexes are not duplicated

#### AC4: Schema Verification
Given the migration has completed
When I query the database schema
Then I can see slug columns in all three tables
And I can see the indexes on slug columns

### Dependencies
- None (foundation for all other work)

### Estimated Size
**S** - Standard ALTER TABLE operations, well-understood SQL migration pattern

---

## Story 1.2: Backfill Slugs from Existing File Paths

**ID**: S-1-2
**Phase**: 1
**Priority**: P0

### User Story
As a **database administrator**
I want **existing slugs extracted from file_path and stored in the slug column**
So that **existing epics/features/tasks have slugs without requiring manual data entry**

### Acceptance Criteria

#### AC1: Epic Slug Extraction
Given an epic with file_path = "docs/plan/E05-task-mgmt-cli-capabilities/epic.md"
When the backfill script runs
Then the slug column is set to "task-mgmt-cli-capabilities"

#### AC2: Feature Slug Extraction
Given a feature with file_path = "docs/plan/E06-intelligent-scanning/E06-F04-incremental-sync-engine/prd.md"
When the backfill script runs
Then the slug column is set to "incremental-sync-engine"

#### AC3: Task Slug Extraction
Given a task with file_path = "docs/plan/E04-epic/E04-F01-feature/tasks/T-E04-F01-001-some-task-description.md"
When the backfill script runs
Then the slug column is set to "some-task-description"

#### AC4: NULL Handling
Given an entity with no file_path (NULL or empty string)
When the backfill script runs
Then the slug column remains NULL
And no error occurs

#### AC5: Verification Report
Given the backfill script has completed
When I request a summary
Then I see count of epics with slugs extracted
And I see count of features with slugs extracted
And I see count of tasks with slugs extracted
And I see count of entities with NULL slugs

### Dependencies
- Story 1.1 (slug columns must exist)

### Estimated Size
**M** - Requires SQL string manipulation logic, testing edge cases, verification

---

## Story 1.3: Create Migration CLI Command

**ID**: S-1-3
**Phase**: 1
**Priority**: P0

### User Story
As a **developer**
I want **a CLI command to run the slug migration**
So that **I can easily migrate existing databases without writing SQL manually**

### Acceptance Criteria

#### AC1: Migration Command Exists
Given the shark CLI
When I run `shark migrate add-slug-columns --help`
Then I see usage instructions
And I see available flags (--dry-run, --verbose)

#### AC2: Dry-Run Mode
Given I want to preview migration changes
When I run `shark migrate add-slug-columns --dry-run`
Then I see which columns would be added
And I see which indexes would be created
And I see estimated number of records to backfill
And no database changes occur

#### AC3: Successful Migration
Given a database without slug columns
When I run `shark migrate add-slug-columns`
Then the migration executes successfully
And I see progress messages
And I see a success confirmation
And slug columns exist in the database

#### AC4: Already Migrated Database
Given a database with slug columns already present
When I run `shark migrate add-slug-columns`
Then I see a message "Migration already applied"
And no errors occur
And the command exits successfully

#### AC5: Migration Failure Handling
Given a migration that encounters an error
When the migration fails
Then the transaction is rolled back
And the database state is unchanged
And I see a clear error message
And the command exits with non-zero status

### Dependencies
- Story 1.1 (migration SQL must be defined)
- Story 1.2 (backfill logic must be defined)

### Estimated Size
**M** - New CLI command, transaction handling, error reporting, testing

---

# Phase 2: Slug Storage (P0)

## Story 2.1: Generate and Store Slug During Epic Creation

**ID**: S-2-1
**Phase**: 2
**Priority**: P0

### User Story
As a **system**
I want **to generate a slug from the epic title and store it in the database at creation time**
So that **the slug is deterministic, immutable, and available for path resolution**

### Acceptance Criteria

#### AC1: Slug Generated from Title
Given I create an epic with title "Task Management CLI - Extended Capabilities"
When the epic creation completes
Then the slug is generated as "task-mgmt-cli-capabilities"
And the slug is stored in the database

#### AC2: Slug Stored Before File Creation
Given I create an epic
When the creation process executes
Then the slug is generated BEFORE the database record is created
And the database record is created BEFORE the file is written
And the file path uses the generated slug

#### AC3: File Path Matches Slug
Given I create an epic with title "My Epic Title"
When the epic is created
Then the slug is "my-epic-title"
And the file_path is "docs/plan/E0X-my-epic-title/epic.md"
And the file actually exists at that path

#### AC4: Unicode Handling
Given I create an epic with title "Café Résumé Naïve"
When the epic is created
Then the slug is "cafe-resume-naive" (unicode normalized)
And no special characters remain except hyphens

#### AC5: Slug Truncation
Given I create an epic with a very long title (>100 characters)
When the epic is created
Then the slug is truncated to 100 characters
And the truncation doesn't break in the middle of a word

### Dependencies
- Story 1.1 (slug column must exist)

### Estimated Size
**S** - Using existing slug.Generate() function, just need to store result

---

## Story 2.2: Generate and Store Slug During Feature Creation

**ID**: S-2-2
**Phase**: 2
**Priority**: P0

### User Story
As a **system**
I want **to generate a slug from the feature title and store it in the database at creation time**
So that **feature slugs are deterministic and available for path resolution**

### Acceptance Criteria

#### AC1: Slug Generated from Title
Given I create a feature with title "Incremental Sync Engine"
When the feature creation completes
Then the slug is generated as "incremental-sync-engine"
And the slug is stored in the database

#### AC2: Feature Path Uses Slug
Given I create a feature with title "My Feature"
When the feature is created in epic E06
Then the slug is "my-feature"
And the file_path is "docs/plan/E06-epic-slug/E06-F0X-my-feature/prd.md"
And the file actually exists at that path

#### AC3: Custom Folder Path Respected
Given I create a feature with custom_folder_path set
When the feature is created
Then the slug is still generated and stored
And the custom folder path is used for file location
And the folder name still includes the slug

#### AC4: Slug Independence from Epic
Given I create two features with same title in different epics
When both features are created
Then both have the same slug (slugs don't need to be globally unique)
And the feature keys remain unique (E05-F01 vs E06-F01)

### Dependencies
- Story 1.1 (slug column must exist)
- Story 2.1 (same pattern as epic creation)

### Estimated Size
**S** - Same implementation pattern as epic, minimal differences

---

## Story 2.3: Generate and Store Slug During Task Creation

**ID**: S-2-3
**Phase**: 2
**Priority**: P0

### User Story
As a **system**
I want **to generate a slug from the task title and store it in the database at creation time**
So that **task slugs are deterministic and available for path resolution**

### Acceptance Criteria

#### AC1: Slug Generated from Title
Given I create a task with title "Implement conflict detection"
When the task creation completes
Then the slug is generated as "implement-conflict-detection"
And the slug is stored in the database

#### AC2: Task Filename Uses Slug
Given I create a task with title "Fix bug in parser"
When the task is created with key T-E04-F01-001
Then the slug is "fix-bug-in-parser"
And the filename is "T-E04-F01-001-fix-bug-in-parser.md"
And the file exists at the expected path

#### AC3: Slug Stored Even If No Title
Given I create a task with empty title
When the task is created
Then the slug is NULL or empty string
And the filename is just the task key + ".md"
And the file is created successfully

#### AC4: Existing Slug Logic Unchanged
Given the recent commit (960a807) added slug to filenames
When I create a task
Then the existing slug generation works
And the slug is now also stored in the database
And the filename matches the database slug

### Dependencies
- Story 1.1 (slug column must exist)
- Story 2.1, 2.2 (same pattern)

### Estimated Size
**S** - Task creation already generates slugs for filenames, just need to store

---

## Story 2.4: Test Slug Storage Across All Entity Types

**ID**: S-2-4
**Phase**: 2
**Priority**: P0

### User Story
As a **QA engineer**
I want **to verify that slugs are consistently stored across all entity types**
So that **I can confirm the slug storage implementation is correct**

### Acceptance Criteria

#### AC1: Integration Test Coverage
Given the slug storage implementation
When I run integration tests
Then I see tests for epic creation with slug
And I see tests for feature creation with slug
And I see tests for task creation with slug

#### AC2: Database Verification
Given I create an epic, feature, and task
When I query the database directly
Then I see slug populated for the epic
And I see slug populated for the feature
And I see slug populated for the task
And all slugs match the expected pattern

#### AC3: File Path Consistency
Given I create entities with slugs
When I examine the file system
Then epic folder names match database slugs
And feature folder names match database slugs
And task filenames match database slugs

#### AC4: Custom Path Scenarios
Given I create entities with custom_folder_path
When I examine the results
Then slugs are still stored correctly
And file paths respect custom paths
And slugs are included in folder/file names

### Dependencies
- Story 2.1, 2.2, 2.3 (all creation stories)

### Estimated Size
**M** - Comprehensive testing across entity types and scenarios

---

# Phase 3: PathResolver (P0)

## Story 3.1: Implement PathResolver Interface

**ID**: S-3-1
**Phase**: 3
**Priority**: P0

### User Story
As a **developer**
I want **a PathResolver that reads paths from the database instead of computing them**
So that **path resolution is fast and database-driven**

### Acceptance Criteria

#### AC1: PathResolver Interface Defined
Given the new architecture
When I review the code
Then I see a PathResolver interface/struct
And it has methods: ResolveEpicPath, ResolveFeaturePath, ResolveTaskPath
And it has a database dependency (not filesystem dependency)

#### AC2: Database-First Resolution
Given an epic exists in database with slug
When I call PathResolver.ResolveEpicPath(ctx, "E05")
Then the resolver queries the database for epic E05
And reads the slug from the database
And computes the path using: key + slug + custom_folder_path
And does NOT read any files

#### AC3: File Path Precedence
Given an epic with file_path set in database
When I resolve the epic path
Then the resolver returns the stored file_path directly
And does NOT recompute the path
And does NOT read files

#### AC4: Custom Folder Path Handling
Given an epic with custom_folder_path = "docs/roadmap/2025-q1"
When I resolve the epic path
Then the path is "docs/roadmap/2025-q1/E0X-slug/epic.md"
And the custom path is respected

#### AC5: Error Handling
Given an epic without slug or file_path
When I resolve the epic path
Then I get an error "epic E05 has no slug or file_path"
And the error is clear and actionable

### Dependencies
- Story 2.1, 2.2, 2.3 (slugs must be stored)

### Estimated Size
**M** - New component, database integration, multiple entity types

---

## Story 3.2: Replace PathBuilder with PathResolver in Commands

**ID**: S-3-2
**Phase**: 3
**Priority**: P0

### User Story
As a **developer**
I want **all CLI commands to use PathResolver instead of PathBuilder**
So that **path resolution is database-driven everywhere**

### Acceptance Criteria

#### AC1: Epic Commands Updated
Given the epic create/get commands
When I review the code
Then I see PathResolver used (not PathBuilder)
And PathBuilder is not imported
And paths are resolved from database

#### AC2: Feature Commands Updated
Given the feature create/get commands
When I review the code
Then I see PathResolver used (not PathBuilder)
And PathBuilder is not imported
And paths are resolved from database

#### AC3: Task Commands Updated
Given the task create/get commands
When I review the code
Then I see PathResolver used (not PathBuilder)
And PathBuilder is not imported
And paths are resolved from database

#### AC4: Sync/Discovery Updated
Given the sync and discovery commands
When I review the code
Then PathResolver is used where paths are needed
And PathBuilder is not imported
And file reads are only for content, not metadata

#### AC5: No PathBuilder Usage Remains
Given the entire codebase
When I search for PathBuilder usage
Then I find no imports of utils.PathBuilder
And I find no calls to PathBuilder methods
And all path resolution goes through PathResolver

### Dependencies
- Story 3.1 (PathResolver must be implemented)

### Estimated Size
**L** - Many commands to update, thorough testing needed

---

## Story 3.3: Performance Testing for PathResolver

**ID**: S-3-3
**Phase**: 3
**Priority**: P0

### User Story
As a **performance engineer**
I want **to verify that PathResolver is faster than PathBuilder**
So that **I can confirm the expected 10x performance improvement**

### Acceptance Criteria

#### AC1: Benchmark PathBuilder
Given the old PathBuilder implementation
When I run path resolution 1000 times
Then I measure average time per resolution
And I record this as the baseline

#### AC2: Benchmark PathResolver
Given the new PathResolver implementation
When I run path resolution 1000 times
Then I measure average time per resolution
And I compare to PathBuilder baseline

#### AC3: Performance Improvement Achieved
Given both benchmarks completed
When I compare results
Then PathResolver is at least 5x faster than PathBuilder
And ideally 10x faster (0.1ms vs 1ms target)

#### AC4: Memory Usage
Given both implementations
When I measure memory allocations
Then PathResolver uses less memory (no file reads)
And memory usage is predictable (database query + path building)

#### AC5: No Performance Regression
Given the migration to PathResolver
When I run end-to-end CLI commands
Then command execution time is not slower
And ideally is noticeably faster for commands with many path resolutions

### Dependencies
- Story 3.1 (PathResolver implemented)
- Story 3.2 (PathBuilder replaced)

### Estimated Size
**M** - Benchmark creation, measurement, analysis

---

# Phase 4: Key Lookup Enhancement (P1)

## Story 4.1: Support Numeric and Slugged Keys in Epic Repository

**ID**: S-4-1
**Phase**: 4
**Priority**: P1

### User Story
As a **CLI user**
I want **to use either `E05` or `E05-task-mgmt-cli-capabilities` in commands**
So that **I don't need to remember or type the full slugged key**

### Acceptance Criteria

#### AC1: Numeric Key Lookup
Given an epic with key "E05" and slug "task-mgmt-cli-capabilities"
When I call GetByKey(ctx, "E05")
Then the epic is returned
And the lookup is fast (indexed on key)

#### AC2: Slugged Key Lookup
Given an epic with key "E05" and slug "task-mgmt-cli-capabilities"
When I call GetByKey(ctx, "E05-task-mgmt-cli-capabilities")
Then the same epic is returned
And the numeric key is extracted ("E05")
And the lookup uses the numeric key

#### AC3: Partial Slug Doesn't Match
Given an epic with key "E05" and slug "task-mgmt-cli-capabilities"
When I call GetByKey(ctx, "E05-task-mgmt")
Then the numeric key "E05" is extracted
And the epic is returned (based on numeric key only)

#### AC4: Invalid Key Format
Given an invalid key format
When I call GetByKey(ctx, "invalid-key")
Then I get a "not found" error
And the error message is clear

#### AC5: Epic Not Found
Given no epic with key "E99"
When I call GetByKey(ctx, "E99")
Then I get sql.ErrNoRows
And the error message says "epic E99 not found"

### Dependencies
- Story 2.1 (epics must have slugs)

### Estimated Size
**S** - String manipulation, fallback logic, straightforward implementation

---

## Story 4.2: Support Numeric and Slugged Keys in Feature Repository

**ID**: S-4-2
**Phase**: 4
**Priority**: P1

### User Story
As a **CLI user**
I want **to use either `E06-F04` or `E06-F04-incremental-sync-engine` in commands**
So that **I can reference features with just the numeric key if I prefer**

### Acceptance Criteria

#### AC1: Numeric Key Lookup
Given a feature with key "E06-F04" and slug "incremental-sync-engine"
When I call GetByKey(ctx, "E06-F04")
Then the feature is returned

#### AC2: Slugged Key Lookup
Given a feature with key "E06-F04" and slug "incremental-sync-engine"
When I call GetByKey(ctx, "E06-F04-incremental-sync-engine")
Then the same feature is returned
And the numeric key "E06-F04" is extracted

#### AC3: Feature Key Extraction Logic
Given a slugged key "E06-F04-some-feature-slug"
When the extraction logic runs
Then it correctly identifies "E06-F04" as the numeric key
And it works even if slug contains more hyphens

### Dependencies
- Story 2.2 (features must have slugs)
- Story 4.1 (same pattern as epics)

### Estimated Size
**S** - Same pattern as epic, slightly more complex key parsing

---

## Story 4.3: Support Numeric and Slugged Keys in Task Repository

**ID**: S-4-3
**Phase**: 4
**Priority**: P1

### User Story
As a **CLI user**
I want **to use either `T-E04-F01-001` or `T-E04-F01-001-task-description` in commands**
So that **I can use the shorter numeric key if I prefer**

### Acceptance Criteria

#### AC1: Numeric Key Lookup
Given a task with key "T-E04-F01-001" and slug "task-description"
When I call GetByKey(ctx, "T-E04-F01-001")
Then the task is returned

#### AC2: Slugged Key Lookup
Given a task with key "T-E04-F01-001" and slug "task-description"
When I call GetByKey(ctx, "T-E04-F01-001-task-description")
Then the same task is returned
And the numeric key "T-E04-F01-001" is extracted

#### AC3: Task Key Extraction Logic
Given a slugged key "T-E04-F01-001-some-task-slug-with-hyphens"
When the extraction logic runs
Then it correctly identifies "T-E04-F01-001" as the numeric key
And handles the T- prefix correctly

### Dependencies
- Story 2.3 (tasks must have slugs)
- Story 4.1, 4.2 (same pattern)

### Estimated Size
**S** - Same pattern, task key has T- prefix to handle

---

## Story 4.4: Update CLI Commands to Accept Both Key Formats

**ID**: S-4-4
**Phase**: 4
**Priority**: P1

### User Story
As a **CLI user**
I want **all commands to accept both numeric and slugged keys**
So that **I can use whichever format is more convenient**

### Acceptance Criteria

#### AC1: Epic Commands Accept Both
Given I want to get epic E05
When I run `shark epic get E05`
Then the epic is displayed
When I run `shark epic get E05-task-mgmt-cli-capabilities`
Then the same epic is displayed

#### AC2: Feature Commands Accept Both
Given I want to get feature E06-F04
When I run `shark feature get E06-F04`
Then the feature is displayed
When I run `shark feature get E06-F04-incremental-sync-engine`
Then the same feature is displayed

#### AC3: Task Commands Accept Both
Given I want to get task T-E04-F01-001
When I run `shark task get T-E04-F01-001`
Then the task is displayed
When I run `shark task get T-E04-F01-001-task-description`
Then the same task is displayed

#### AC4: Help Text Updated
Given I run any command with --help
When I read the help text
Then I see examples showing both numeric and slugged keys
And the help explains that both formats work

#### AC5: Error Messages Clear
Given I use an invalid key format
When the command fails
Then the error message is clear
And suggests trying the numeric key format

### Dependencies
- Story 4.1, 4.2, 4.3 (repositories support both formats)

### Estimated Size
**M** - All commands need testing, help text updates

---

# Phase 5: File Format Conversion (P2 - OPTIONAL)

## Story 5.1: Design YAML Frontmatter Template for Epics

**ID**: S-5-1
**Phase**: 5
**Priority**: P2

### User Story
As a **system designer**
I want **a standardized YAML frontmatter template for epic files**
So that **epic metadata is machine-readable and consistent with task files**

### Acceptance Criteria

#### AC1: Template Defines All Fields
Given the epic YAML template
When I review the template
Then I see fields for: epic_key, slug, title, description, status, priority, business_value, created_at, updated_at
And the template matches the database schema

#### AC2: Template Matches Task Pattern
Given the task YAML frontmatter format
When I compare to epic template
Then the structure is consistent
And the field naming conventions match
And the format is easy to parse

#### AC3: Template Supports Optional Fields
Given fields like description and business_value are optional
When I use the template
Then optional fields can be omitted
And the parser handles missing fields gracefully

#### AC4: Example Template Rendered
Given the template definition
When I create an example epic
Then I see a complete YAML frontmatter example
And the example is valid YAML
And the example includes comments explaining each field

### Dependencies
- None (design task)

### Estimated Size
**XS** - Template design, documentation

---

## Story 5.2: Design YAML Frontmatter Template for Features

**ID**: S-5-2
**Phase**: 5
**Priority**: P2

### User Story
As a **system designer**
I want **a standardized YAML frontmatter template for feature files**
So that **feature metadata is machine-readable and consistent**

### Acceptance Criteria

#### AC1: Template Defines All Fields
Given the feature YAML template
When I review the template
Then I see fields for: feature_key, epic_key, slug, title, description, status, execution_order, created_at, updated_at
And the template matches the database schema

#### AC2: Epic Reference Included
Given features belong to epics
When I use the template
Then the epic_key field is required
And there's a recommended format for linking to epic file

#### AC3: Consistent with Task and Epic Templates
Given all three templates exist
When I compare them
Then the structure is consistent
And common fields use the same names and formats

### Dependencies
- Story 5.1 (epic template for consistency)

### Estimated Size
**XS** - Template design, same pattern as epic

---

## Story 5.3: Create Migration Script to Convert Epic Files

**ID**: S-5-3
**Phase**: 5
**Priority**: P2

### User Story
As a **database administrator**
I want **a script to convert existing epic files from markdown format to YAML frontmatter**
So that **I can migrate all epic files consistently**

### Acceptance Criteria

#### AC1: Dry-Run Mode
Given I want to preview changes
When I run `shark migrate convert-epic-files --dry-run`
Then I see which files would be converted
And I see what the new format would look like
And no files are actually changed

#### AC2: Successful Conversion
Given an epic file in markdown format
When I run the conversion
Then the file is converted to YAML frontmatter
And all metadata is preserved
And the content body is preserved
And the file is valid YAML

#### AC3: Backup Created
Given the conversion will modify files
When I run the conversion
Then backup copies are created
And the backup location is reported
And I can restore from backup if needed

#### AC4: Validation After Conversion
Given epic files have been converted
When I run discovery/sync
Then the files are parsed correctly
And slugs match database
And no errors occur

#### AC5: Rollback Support
Given a conversion has completed
When I need to rollback
Then I can restore from backups
And the system returns to original state

### Dependencies
- Story 5.1 (template must be defined)
- Existing epic files with metadata

### Estimated Size
**L** - File I/O, parsing markdown, generating YAML, backup/restore, validation

---

## Story 5.4: Create Migration Script to Convert Feature Files

**ID**: S-5-4
**Phase**: 5
**Priority**: P2

### User Story
As a **database administrator**
I want **a script to convert existing feature files from markdown format to YAML frontmatter**
So that **I can migrate all feature files consistently**

### Acceptance Criteria

#### AC1: Dry-Run Mode
Given I want to preview changes
When I run `shark migrate convert-feature-files --dry-run`
Then I see which files would be converted
And I see the new format
And no files are changed

#### AC2: Successful Conversion
Given a feature file in current format
When I run the conversion
Then the file is converted to YAML frontmatter
And all metadata is preserved
And epic references are maintained

#### AC3: Handles Missing Metadata
Given some feature files may have minimal metadata
When I run the conversion
Then the script handles missing fields gracefully
And uses database values where file values are missing
And reports any issues

### Dependencies
- Story 5.2 (template must be defined)
- Story 5.3 (same pattern as epic conversion)

### Estimated Size
**L** - Same complexity as epic conversion

---

## Story 5.5: Update Discovery to Parse YAML Frontmatter

**ID**: S-5-5
**Phase**: 5
**Priority**: P2

### User Story
As a **system**
I want **discovery to parse YAML frontmatter from epic and feature files**
So that **metadata is extracted correctly from the new format**

### Acceptance Criteria

#### AC1: Parse Epic YAML Frontmatter
Given an epic file with YAML frontmatter
When discovery scans the file
Then the frontmatter is parsed as YAML
And the epic_key is extracted
And the slug is extracted
And all other metadata is extracted

#### AC2: Parse Feature YAML Frontmatter
Given a feature file with YAML frontmatter
When discovery scans the file
Then the frontmatter is parsed as YAML
And the feature_key is extracted
And the slug is extracted
And the epic_key reference is extracted

#### AC3: Validate Slug Matches Database
Given a file with slug in frontmatter
When discovery processes the file
Then the file slug is compared to database slug
And mismatches are reported
And the database slug takes precedence

#### AC4: Handle Parse Errors
Given a file with invalid YAML
When discovery processes the file
Then a clear error is reported
And the file path is included in the error
And discovery continues with other files

### Dependencies
- Story 5.3, 5.4 (files must be in new format)

### Estimated Size
**M** - YAML parsing, validation logic, error handling

---

# Phase 6: Cleanup (P1)

## Story 6.1: Deprecate and Remove PathBuilder

**ID**: S-6-1
**Phase**: 6
**Priority**: P1

### User Story
As a **developer**
I want **PathBuilder removed from the codebase**
So that **there's only one way to resolve paths (PathResolver)**

### Acceptance Criteria

#### AC1: PathBuilder Code Removed
Given the PathResolver is fully implemented
When I review the codebase
Then internal/utils/path_builder.go is deleted
And no imports of PathBuilder exist
And no references to PathBuilder remain

#### AC2: Tests Updated
Given PathBuilder tests existed
When I review test files
Then PathBuilder tests are removed
And PathResolver tests exist
And test coverage is maintained or improved

#### AC3: Documentation Updated
Given PathBuilder was documented
When I review documentation
Then references to PathBuilder are removed
And PathResolver is documented instead
And migration notes explain the change

#### AC4: No Breaking Changes
Given external users may depend on patterns
When PathBuilder is removed
Then no public APIs are broken
And internal refactoring only
And CLI commands work unchanged

### Dependencies
- Story 3.2 (PathBuilder must be fully replaced)

### Estimated Size
**S** - Code deletion, test updates, documentation

---

## Story 6.2: Update Project Documentation

**ID**: S-6-2
**Phase**: 6
**Priority**: P1

### User Story
As a **new developer**
I want **documentation that accurately reflects the slug architecture**
So that **I understand how path resolution works**

### Acceptance Criteria

#### AC1: CLAUDE.md Updated
Given the project CLAUDE.md file
When I review it
Then I see documentation about slug storage
And I see documentation about PathResolver
And I see examples of both numeric and slugged keys
And references to PathBuilder are removed

#### AC2: README.md Updated
Given the project README.md
When I review it
Then I see examples using both key formats
And I see migration instructions if needed
And the architecture description is accurate

#### AC3: Architecture Documentation Added
Given developers need to understand architecture
When I review the docs
Then I see a document explaining slug architecture
And I see diagrams showing database-first approach
And I see decision rationale

#### AC4: Migration Guide Available
Given users need to migrate existing databases
When I review documentation
Then I see a migration guide
And the guide has step-by-step instructions
And the guide includes troubleshooting

### Dependencies
- All previous phases complete

### Estimated Size
**M** - Multiple documentation files, examples, diagrams

---

## Story 6.3: Performance Benchmarking and Reporting

**ID**: S-6-3
**Phase**: 6
**Priority**: P1

### User Story
As a **project stakeholder**
I want **performance benchmarks comparing old and new architecture**
So that **I can verify the expected improvements were achieved**

### Acceptance Criteria

#### AC1: Benchmark Suite Exists
Given the final implementation
When I run the benchmark suite
Then I see benchmarks for:
- Path resolution (old vs new)
- Discovery (old vs new)
- Slug lookup (old vs new)
And results are reproducible

#### AC2: Performance Report
Given benchmarks have been run
When I review the performance report
Then I see before/after comparisons
And I see percentage improvements
And I see whether 10x target was achieved

#### AC3: Memory Usage Report
Given benchmarks include memory profiling
When I review memory usage
Then I see memory allocations (old vs new)
And I see peak memory usage
And I verify no memory leaks

#### AC4: Regression Testing
Given performance tests exist
When I run them in CI/CD
Then they execute automatically
And they fail if performance regresses
And they report results

### Dependencies
- All implementation phases complete

### Estimated Size
**M** - Comprehensive benchmarking, reporting, CI integration

---

## Story 6.4: End-to-End Validation

**ID**: S-6-4
**Phase**: 6
**Priority**: P1

### User Story
As a **QA engineer**
I want **comprehensive end-to-end tests covering the new architecture**
So that **I can verify the entire system works correctly**

### Acceptance Criteria

#### AC1: Create Epic End-to-End
Given I create a new epic
When I run through the full workflow
Then the epic is created with slug
And the slug is stored in database
And the file path includes slug
And I can retrieve the epic by numeric or slugged key

#### AC2: Create Feature End-to-End
Given I create a new feature in an epic
When I run through the full workflow
Then the feature is created with slug
And the slug is stored in database
And the file path includes slug
And custom paths work correctly

#### AC3: Create Task End-to-End
Given I create a new task in a feature
When I run through the full workflow
Then the task is created with slug
And the slug is stored in database
And the filename includes slug

#### AC4: Discovery End-to-End
Given I have created entities
When I run discovery
Then all entities are discovered
And slugs are validated against database
And no errors occur

#### AC5: Migration End-to-End
Given a pre-migration database
When I run all migration steps
Then the database is migrated successfully
And all data is preserved
And the system works correctly

### Dependencies
- All previous stories complete

### Estimated Size
**L** - Comprehensive testing across all workflows

---

# Summary

## Story Count by Phase

- **Phase 1** (Database Schema): 3 stories
- **Phase 2** (Slug Storage): 4 stories
- **Phase 3** (PathResolver): 3 stories
- **Phase 4** (Key Lookup): 4 stories
- **Phase 5** (File Format - Optional): 5 stories
- **Phase 6** (Cleanup): 4 stories

**Total**: 23 user stories

## Story Count by Priority

- **P0 (Must Have)**: 13 stories (Phases 1-3)
- **P1 (Should Have)**: 8 stories (Phases 4, 6)
- **P2 (Nice to Have)**: 5 stories (Phase 5)

## Story Count by Size

- **XS**: 2 stories
- **S**: 8 stories
- **M**: 9 stories
- **L**: 4 stories
- **XL**: 0 stories

## Implementation Order

Stories should be implemented in phase order (1 → 6) and within each phase in story number order (e.g., 1.1 → 1.2 → 1.3). Dependencies are clearly documented in each story.

## Next Steps

1. Review stories with technical team
2. Refine acceptance criteria as needed
3. Create implementation tasks using `/task` command
4. Begin Phase 1 implementation
