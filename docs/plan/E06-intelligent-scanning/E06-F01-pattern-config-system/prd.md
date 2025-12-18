# Feature: Pattern Configuration System

**Feature Key**: E06-F01-pattern-config-system

---

## Epic

- [Epic PRD](/home/jwwelbor/projects/shark-task-manager/docs/plan/E06-intelligent-scanning/epic.md)
- [Epic Requirements](/home/jwwelbor/projects/shark-task-manager/docs/plan/E06-intelligent-scanning/requirements.md)

---

## Goal

### Problem

The current shark synchronization system assumes a rigid naming convention (E##-epic-slug, E##-F##-feature-slug, T-E##-F##-###.md) for discovering epics, features, and tasks. This approach fails when projects use:
- Different epic naming patterns (tech-debt, bugs, change-cards instead of E##-slug)
- Varied feature file names (PRD_F##-name.md, feature-slug.md, prd.md)
- Alternative task numbering schemes (###-task-name.md, task-name.prp.md in prps/ folder)
- Legacy documentation structures that evolved organically over months or years

Without a flexible pattern matching system, users must either manually restructure hundreds of markdown files to conform to shark's expectations or abandon using shark on existing projects. The lack of configurability creates a massive adoption barrier for real-world teams with established documentation conventions.

### Solution

Implement a comprehensive pattern configuration system that allows users to define custom regex patterns for epic/feature/task discovery while maintaining a "works out of the box" experience through intelligent defaults. The system will:

1. **Regex Pattern Definition**: Store configurable regex patterns in .sharkconfig.json with named capture groups (epic_id, epic_slug, feature_id, feature_slug, task_id, task_slug, number) that enable relationship inference between entities

2. **Pattern Validation**: Validate patterns on config load to ensure correct regex syntax and presence of required capture groups, providing immediate feedback before scan operations

3. **Generation Format Configuration**: Define separate, standardized formats for creating new items via CLI (shark epic create, shark feature create, shark task create) using template placeholders, allowing flexible input while ensuring consistent output

4. **Pattern Preset Library**: Provide pre-built pattern collections (standard, special-epics, numeric-only, legacy-prp) accessible via CLI commands, eliminating the need for users to write complex regex from scratch

5. **Pattern Ordering Logic**: Support multiple patterns per entity type evaluated in array order with first-match-wins semantics, enabling progressive fallback from strict to flexible patterns

This configuration layer serves as the foundation for the intelligent scanning system (E06-F02, E06-F03), enabling shark to adapt to existing documentation rather than forcing documentation to conform to shark.

### Impact

- **Adoption Barrier Reduction**: Enable shark usage on 90% of existing projects without requiring documentation restructuring, reducing setup time from hours to minutes
- **User Flexibility**: Support custom documentation conventions through configuration-only changes, eliminating need for code modifications
- **Developer Experience**: Provide "works out of the box" defaults that recognize common patterns (E##-slug, tech-debt, bugs) while allowing easy extension for team-specific conventions
- **Data Integrity**: Ensure relationship inference reliability through strict validation of required capture groups, maintaining 100% foreign key integrity
- **Configuration Confidence**: Enable pattern testing and validation before scanning, reducing trial-and-error and preventing failed imports

**Measurable Targets**:
- 95% of projects work with default patterns without customization
- Pattern validation catches 100% of syntax errors and missing required capture groups before scan attempts
- Users can add new patterns via presets in <30 seconds without regex knowledge
- Pattern testing command provides immediate feedback (<500ms) for pattern debugging

---

## User Personas

### Persona 1: Technical Lead (Existing Project Migration)

**Profile**:
- **Role**: Technical Lead or Engineering Manager migrating existing multi-epic project to shark
- **Experience**: 5+ years in role, high technical proficiency with CLI tools and Git
- **Context**: Managing 3-5 concurrent epics with 10-30 features each, existing markdown documentation evolved organically over 6-18 months with varied naming conventions
- **Technical Skills**: Comfortable reading regex patterns, experienced with JSON configuration files

**Goals Related to This Feature**:
1. Define custom patterns matching existing documentation conventions without restructuring files
2. Validate patterns before attempting full project import to prevent data corruption
3. Understand which patterns matched which files for troubleshooting failed imports
4. Maintain control over what gets generated when creating new items via CLI

**Pain Points This Feature Addresses**:
- Current sync assumes E##-epic-slug format; many existing epics use descriptive names (identity, notifications, tech-debt)
- No visibility into why files weren't recognized (pattern mismatch vs. validation failure)
- Fear of wrong patterns causing incorrect categorization or relationship corruption
- Uncertainty about whether custom patterns will work before attempting import

**Success Looks Like**:
The Technical Lead edits .sharkconfig.json to add a pattern for "tech-debt" epic type, runs `shark config validate-patterns` to confirm syntax is correct and required capture groups are present, then tests the pattern with `shark config test-pattern --pattern="..." --test-string="tech-debt"` to verify recognition. After confirming pattern works, they run `shark scan --dry-run` and see tech-debt epic and its 15 tasks correctly identified. They proceed with confidence to execute the full import.

---

### Persona 2: Product Manager (Multi-Style Documentation)

**Profile**:
- **Role**: Product Manager or Technical PM managing projects across multiple teams
- **Experience**: 3-5 years in role, moderate technical proficiency, comfortable with markdown and Git
- **Context**: Works with documentation from different teams/time periods with varied conventions, needs flexibility over strict standardization
- **Technical Skills**: Can edit JSON configuration files with examples, prefers presets over writing regex

**Goals Related to This Feature**:
1. Add support for special epic types (tech-debt, bugs, change-cards) without learning regex
2. Ensure new items generated via CLI follow team conventions
3. Validate configuration changes don't break existing imports
4. Understand which pattern matched each file for documentation purposes

**Pain Points This Feature Addresses**:
- Cannot track tech-debt or bug epics that don't use E## format
- Don't know regex syntax well enough to write custom patterns confidently
- Different teams use different feature file patterns (prd.md vs. PRD_F##-name.md)
- Unsure if configuration changes will break existing scans

**Success Looks Like**:
The Product Manager runs `shark config add-pattern --preset=special-epics` to add support for tech-debt, bugs, and change-cards. They run `shark config show` to verify the patterns were added correctly. They test one pattern with a simple command to confirm it recognizes "tech-debt" as a valid epic. After validation, they run a full scan and see all special epic types imported successfully alongside standard E## epics.

---

### Persona 3: AI Agent (Incremental Development Sync)

**Profile**:
- **Role**: Autonomous code generation and task execution agent (Claude Code)
- **Experience**: Stateless between sessions, relies on explicit project state
- **Context**: Creates new task files during feature development, needs fast sync to maintain database consistency
- **Technical Skills**: Can read JSON configuration, cannot manually debug complex regex

**Goals Related to This Feature**:
1. Understand configured generation formats to create files that will be recognized on next sync
2. Receive clear errors when generated files don't match any configured pattern
3. Query generation format at runtime to ensure file naming consistency
4. Avoid creating files that will be orphaned (not imported) due to pattern mismatch

**Pain Points This Feature Addresses**:
- Generated files may have slight variations from ideal format
- No visibility into why created file wasn't imported on subsequent sync
- Cannot predict if filename will match configured patterns
- Sync failures are opaque (agent doesn't understand pattern mismatch errors)

**Success Looks Like**:
The AI Agent queries the configured task generation format before creating a new file, generates filename according to the template (T-E04-F07-003.md), and creates the file. On subsequent `shark sync`, the file is immediately recognized and imported. If the agent accidentally creates a non-standard filename, the sync error message clearly states "File docs/plan/E04/.../task-003.md did not match any configured task patterns" with suggestions for valid formats.

---

## User Stories

### Must-Have Stories

**Story 1: Configure Custom Epic Pattern**
- **As a** Technical Lead
- **I want to** define a custom regex pattern for epic folder names in .sharkconfig.json
- **So that I can** import epics that use naming conventions like "tech-debt" or "bugs" instead of E##-epic-slug
- **Priority**: Must-Have
- **Related Requirements**: REQ-F-003, REQ-F-013

**Story 2: Validate Patterns on Config Load**
- **As a** Technical Lead
- **I want** shark to validate regex syntax and required capture groups when loading .sharkconfig.json
- **So that I can** catch configuration errors immediately rather than discovering them during scan operations
- **Priority**: Must-Have
- **Related Requirements**: REQ-F-013

**Story 3: Add Preset Pattern Without Writing Regex**
- **As a** Product Manager
- **I want to** add common pattern presets via `shark config add-pattern --preset=special-epics`
- **So that I can** support tech-debt, bugs, and change-cards without learning regex syntax
- **Priority**: Must-Have
- **Related Requirements**: REQ-F-015

**Story 4: Define Generation Format Separate from Match Patterns**
- **As a** Technical Lead
- **I want** generation formats (for creating new items) to be separate from match patterns (for importing existing items)
- **So that I can** accept many filename variations during import while consistently generating standardized filenames for new items
- **Priority**: Must-Have
- **Related Requirements**: REQ-F-014

**Story 5: Test Pattern Against String**
- **As a** Technical Lead
- **I want to** test a regex pattern against a sample string via `shark config test-pattern --pattern="..." --test-string="tech-debt"`
- **So that I can** verify pattern matching behavior before running a full scan
- **Priority**: Should-Have
- **Related Requirements**: REQ-F-013 (extension)

**Story 6: Use Multiple Patterns with First-Match-Wins Logic**
- **As a** Product Manager
- **I want** shark to try multiple patterns in array order and use the first match
- **So that I can** support both strict patterns (E##-F##-slug) and flexible fallbacks (descriptive-name) without false positives
- **Priority**: Must-Have
- **Related Requirements**: REQ-F-003

**Story 7: View Current Pattern Configuration**
- **As a** Product Manager
- **I want to** view all configured patterns via `shark config show --patterns`
- **So that I can** understand current recognition rules and debug import issues
- **Priority**: Should-Have
- **Related Requirements**: REQ-F-015

**Story 8: Query Generation Format for Entity Type**
- **As an** AI Agent
- **I want** to query the configured generation format for tasks via API or CLI
- **So that I can** create filenames that match team conventions and will be recognized on sync
- **Priority**: Should-Have
- **Related Requirements**: REQ-F-014

### Alternative Path Stories

**Story 9: Override Validation for Advanced Users**
- **As a** Technical Lead
- **I want to** bypass pattern validation with `--skip-pattern-validation` flag
- **So that I can** use advanced regex features that validation might incorrectly reject
- **Priority**: Could-Have
- **Related Requirements**: REQ-F-013

**Story 10: View Available Presets**
- **As a** Product Manager
- **I want to** list available pattern presets via `shark config list-presets`
- **So that I can** discover what common patterns are available before attempting custom regex
- **Priority**: Should-Have
- **Related Requirements**: REQ-F-015

**Story 11: Export Patterns for Sharing**
- **As a** Technical Lead
- **I want to** export my custom pattern configuration for sharing with other teams
- **So that I can** help other projects adopt similar documentation conventions
- **Priority**: Could-Have
- **Related Requirements**: REQ-F-015 (extension)

### Edge Case Stories

**Story 12: Handle Pattern Ordering Conflicts**
- **As a** Technical Lead
- **I want** shark to warn me if two patterns might match the same string
- **So that I can** prevent unintended matches due to pattern ordering
- **Priority**: Could-Have
- **Related Requirements**: REQ-F-013 (extension)

**Story 13: Detect Capture Group Name Typos**
- **As a** Product Manager
- **I want** validation to warn about capture group names that are close to required names (e.g., "feature_num" instead of "feature_id")
- **So that I can** catch typos that would break relationship inference
- **Priority**: Could-Have
- **Related Requirements**: REQ-F-013 (extension)

---

## Requirements

### Functional Requirements

#### Pattern Definition (Must-Have)

**FR-001: Regex Pattern Storage in .sharkconfig.json**
- System must store regex patterns in .sharkconfig.json under `patterns` object with structure:
  ```json
  {
    "patterns": {
      "epic": {
        "folder": ["pattern1", "pattern2"],
        "file": ["pattern1", "pattern2"],
        "generation": {
          "format": "E{number:02d}-{slug}"
        }
      },
      "feature": { /* similar structure */ },
      "task": { /* similar structure */ }
    }
  }
  ```
- Each entity type (epic, feature, task) must support `folder` and `file` pattern arrays
- Pattern arrays must be evaluated in order with first-match-wins semantics
- Generation format must be a single string template (not an array)

**FR-002: Named Capture Group Support**
- Epic patterns must support capture groups: `epic_id`, `epic_slug`, `number`
- Feature patterns must support capture groups: `epic_id`, `epic_num`, `feature_id`, `feature_slug`, `number`
- Task patterns must support capture groups: `epic_id`, `epic_num`, `feature_id`, `feature_num`, `task_id`, `task_slug`, `number`
- System must extract captured values and make them available to scanning engine for database key construction and relationship inference

**FR-003: Comprehensive Default Patterns**
- System must ship with default patterns covering common conventions:
  - **Epic folders**: `E\d{2}-[a-z0-9-]+` (standard), `tech-debt|bugs|change-cards` (special types)
  - **Feature folders**: `E\d{2}-F\d{2}-[a-z0-9-]+` (standard)
  - **Feature files**: `prd\.md` (prioritized), `PRD_F\d{2}-.+\.md`, `[a-z0-9-]+\.md` (fallback)
  - **Task files**: `T-E\d{2}-F\d{2}-\d{3}.*\.md` (full key), `\d{3}-.+\.md` (number-based), `.+\.prp\.md` (legacy)
- Default patterns must include named capture groups for all required fields
- Defaults must be applied if user config doesn't specify patterns (fallback behavior)

**FR-004: Generation Format Templates**
- System must support placeholder syntax in generation format strings:
  - `{number}` - unformatted number (1, 2, 3)
  - `{number:02d}` - zero-padded 2-digit number (01, 02, 03)
  - `{slug}` - user-provided slug value
  - `{epic}` or `{epic:02d}` - parent epic number
  - `{feature}` or `{feature:02d}` - parent feature number
- System must apply generation format when creating new items via `shark epic create`, `shark feature create`, `shark task create`
- Default generation formats:
  - Epic: `E{number:02d}-{slug}`
  - Feature: `E{epic:02d}-F{number:02d}-{slug}`
  - Task: `T-E{epic:02d}-F{feature:02d}-{number:03d}.md`

#### Pattern Validation (Must-Have)

**FR-005: Config Load Validation**
- System must validate all patterns when loading .sharkconfig.json
- Validation must occur before any scan operations
- Validation failures must prevent config load and display detailed error messages
- System must support `--skip-pattern-validation` flag to bypass validation (advanced users only)

**FR-006: Regex Syntax Validation**
- System must validate regex syntax for all patterns using Go's regexp.Compile
- Invalid regex must produce error with pattern name, invalid regex string, and Go compiler error message
- Example error: "Invalid epic folder pattern 'E\\d{2-[a-z]+': missing closing brace in repetition operator"

**FR-007: Required Capture Group Validation**
- Epic patterns must include at least one of: `epic_id`, `epic_slug`, `number`
- Feature patterns must include at least one of (`epic_id` or `epic_num`) AND one of (`feature_id`, `feature_slug`, `number`)
- Task patterns must include at least one of (`epic_id` or `epic_num`) AND one of (`feature_id` or `feature_num`) AND one of (`task_id`, `number`, `task_slug`)
- Missing required capture groups must produce error listing pattern name, pattern string, required groups, and found groups
- Example error: "Feature folder pattern 'F\d{2}-(?P<slug>[a-z-]+)' missing required epic identifier capture group (epic_id or epic_num)"

**FR-008: Capture Group Name Validation**
- System must validate capture group names match expected set (epic_id, epic_slug, epic_num, feature_id, feature_slug, feature_num, task_id, task_slug, task_num, number, slug)
- Unrecognized capture group names should produce warning (not error) suggesting possible typos
- Example warning: "Pattern contains capture group 'feature_number' which will be ignored. Did you mean 'feature_num' or 'number'?"

#### Pattern Preset Library (Must-Have)

**FR-009: Preset Pattern Collections**
- System must provide built-in preset collections:
  - **standard**: Default E##-slug conventions for all entity types
  - **special-epics**: Patterns for tech-debt, bugs, change-cards epic types
  - **numeric-only**: E001, F001, T001 style numbering without slugs
  - **legacy-prp**: Support for .prp.md files in prps/ subfolder
- Each preset must be a JSON structure containing patterns for one or more entity types
- Presets must be embedded in binary (not external files requiring distribution)

**FR-010: Add Preset Command**
- System must provide `shark config add-pattern --preset=<name>` command
- Command must append preset patterns to existing config (not replace)
- Command must handle duplicate patterns gracefully (skip if exact pattern already exists)
- Command must validate combined patterns after addition
- Command must update .sharkconfig.json with merged patterns
- Command must provide feedback showing which patterns were added vs. skipped

**FR-011: List Presets Command**
- System must provide `shark config list-presets` command
- Command must display preset names with brief descriptions
- Command must be available even if .sharkconfig.json doesn't exist yet

**FR-012: Show Preset Command**
- System must provide `shark config show-preset <name>` command
- Command must display full pattern structure for specified preset
- Command must show patterns in JSON format ready for manual copying
- Command must indicate which entity types the preset affects

#### Pattern Testing (Should-Have)

**FR-013: Test Pattern Command**
- System must provide `shark config test-pattern` command with options:
  - `--pattern=<regex>` - pattern to test
  - `--test-string=<string>` - string to match against
  - `--type=<epic|feature|task>` - entity type for validation context
- Command must compile pattern and attempt match against test string
- On successful match, command must display captured groups and values
- On failed match, command must indicate no match and suggest similar patterns from config
- Command must validate capture groups for specified type and warn if required groups missing
- Command must complete in <500ms for immediate feedback

**FR-014: Validate Patterns Command**
- System must provide `shark config validate-patterns` command
- Command must validate all patterns in current .sharkconfig.json
- Command must report validation results grouped by entity type (epic, feature, task)
- Command must distinguish between errors (must fix) and warnings (should review)
- Command must exit with non-zero status if any errors found (for CI integration)

#### Configuration Display (Should-Have)

**FR-015: Show Configuration Command**
- System must provide `shark config show` command with optional `--patterns` flag
- Without flag, command displays all config settings
- With `--patterns` flag, command displays only pattern-related configuration
- Output must be formatted for readability (pretty-printed JSON or table format)
- Command must indicate which patterns are defaults vs. user-customized

**FR-016: Query Generation Format**
- System must provide `shark config get-format --type=<epic|feature|task>` command
- Command must return generation format template for specified type
- Command must support `--json` flag for programmatic access (AI agents)
- Output example: `E{number:02d}-{slug}` or JSON: `{"format": "E{number:02d}-{slug}", "example": "E04-example-epic"}`

### Non-Functional Requirements

#### Performance (Must-Have)

**NFR-001: Pattern Validation Speed**
- Pattern validation during config load must complete in <100ms for typical configuration (15-20 patterns)
- Pattern testing command must complete in <500ms for immediate feedback
- Validation time must not increase scan operation time by more than 5%

**NFR-002: Pattern Matching Efficiency**
- Compiled regex patterns must be cached for reuse during scanning operations
- First-match-wins evaluation must short-circuit (stop after first match, don't test remaining patterns)
- Pattern matching overhead must be <1ms per file for typical pattern sets (3-5 patterns per entity type)

#### Security (Must-Have)

**NFR-003: Regex Denial of Service Protection**
- System must detect catastrophic backtracking patterns during validation
- System must reject patterns with excessive nested quantifiers (e.g., `(a+)+b`)
- System must implement pattern matching timeout (100ms per match attempt)
- Pattern validation should warn about performance risks (e.g., unbounded alternation)

**NFR-004: Path Injection Prevention**
- Generation format placeholders must be sanitized to prevent directory traversal (no `../` in slug values)
- Generated file paths must be validated to ensure they remain within project boundaries
- System must reject slug values containing filesystem special characters (`/`, `\`, `:`, etc.)

#### Usability (Must-Have)

**NFR-005: Actionable Error Messages**
- All validation errors must include:
  - Which pattern failed (entity type, folder vs. file, array index)
  - The exact pattern string that failed
  - What was wrong (syntax error, missing capture group, etc.)
  - Suggested fix or reference to documentation
- Example: "Feature folder pattern #2 'E\d{2-F\d{2}' has invalid regex syntax: missing closing brace. Suggestion: Change to 'E\d{2}-F\d{2}' or see docs/patterns.md for examples."

**NFR-006: Progressive Disclosure**
- Default commands should work without flags for common cases (`shark config validate-patterns`)
- Advanced options should be available via flags for power users (`--skip-pattern-validation`)
- Help text must include examples for common operations
- CLI output must be concise by default, verbose with `--verbose` flag

**NFR-007: Documentation**
- All pattern syntax must be documented with examples in docs/patterns.md
- Each preset must be documented with use cases and example matches
- Generation format placeholder syntax must be fully documented
- Capture group requirements for each entity type must be clearly documented

#### Reliability (Must-Have)

**NFR-008: Configuration Validation on Write**
- All commands that modify .sharkconfig.json must validate before writing
- Validation failures must leave config file unchanged (atomic write)
- System must create backup of config file before modification (rollback capability)
- Corrupted config files must not crash shark (graceful fallback to defaults with warning)

**NFR-009: Backward Compatibility**
- Future pattern schema changes must support migration from previous versions
- System must detect old config format and offer upgrade command
- Defaults must remain stable across versions (no breaking changes without major version bump)

#### Maintainability (Must-Have)

**NFR-010: Pattern Extensibility**
- Adding new capture group names must require only updating validation whitelist (no changes to matching engine)
- Adding new presets must require only adding JSON structure to preset library (no code changes)
- Pattern matching logic must be entity-type agnostic (same engine for epic/feature/task)

**NFR-011: Test Coverage**
- Pattern validation logic must have >90% unit test coverage
- Each preset must have test cases verifying correct matches and non-matches
- Generation format logic must have test cases for all placeholder types and formatting options
- Regex DoS protection must have test cases for known problematic patterns

---

## Acceptance Criteria

### Configuration Structure

**AC-001: .sharkconfig.json Pattern Schema**
- **Given** a new shark project
- **When** user runs `shark init`
- **Then** .sharkconfig.json is created with default patterns structure including epic/feature/task sections
- **And** each section contains folder and file pattern arrays
- **And** each section contains generation format object

**AC-002: Default Patterns Included**
- **Given** a new .sharkconfig.json
- **When** user inspects patterns section
- **Then** epic folder patterns include E##-slug and special-epics alternatives
- **And** feature folder patterns include E##-F##-slug
- **And** feature file patterns include prd.md (priority), PRD_F##-name.md, and slug.md fallback
- **And** task file patterns include T-E##-F##-###.md, ###-name.md, and .prp.md variants
- **And** all patterns include appropriate named capture groups

### Pattern Validation

**AC-003: Valid Pattern Accepted**
- **Given** a pattern `E(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)` for epic folders
- **When** config is loaded
- **Then** pattern compiles successfully
- **And** required capture groups (number or epic_id) are present
- **And** no validation errors are produced

**AC-004: Invalid Regex Syntax Rejected**
- **Given** a pattern `E\d{2-[a-z]+` with missing closing brace
- **When** config is loaded
- **Then** config load fails with validation error
- **And** error message includes pattern string and syntax problem
- **And** error message suggests fix or documentation reference

**AC-005: Missing Required Capture Group Rejected**
- **Given** a feature folder pattern `F\d{2}-[a-z-]+` without epic identifier capture group
- **When** config is loaded
- **Then** config load fails with validation error
- **And** error message specifies missing epic_id or epic_num capture group
- **And** error message explains why this capture group is required (relationship inference)

**AC-006: Unrecognized Capture Group Warning**
- **Given** a pattern with capture group `(?P<feature_number>\d{2})` instead of `feature_num`
- **When** config is loaded
- **Then** config loads successfully (warning, not error)
- **And** warning message suggests "Did you mean 'feature_num'?"
- **And** system proceeds with pattern (ignoring unrecognized group)

**AC-007: Validation Bypass for Advanced Users**
- **Given** a pattern that validation would reject
- **When** user runs command with `--skip-pattern-validation` flag
- **Then** validation is skipped
- **And** warning message indicates validation was skipped (use at own risk)
- **And** command proceeds with potentially invalid pattern

### Pattern Presets

**AC-008: Add Standard Preset**
- **Given** a config with only epic patterns
- **When** user runs `shark config add-pattern --preset=standard`
- **Then** standard patterns are added for epic, feature, and task
- **And** existing epic patterns are preserved (not replaced)
- **And** duplicate patterns are skipped with informational message
- **And** .sharkconfig.json is updated with merged patterns

**AC-009: Add Special Epics Preset**
- **Given** a config with default patterns
- **When** user runs `shark config add-pattern --preset=special-epics`
- **Then** patterns for tech-debt, bugs, change-cards are added to epic folder patterns array
- **And** patterns are appended (not prepended) to maintain standard pattern priority
- **And** command reports "Added 1 epic folder pattern for special epic types"

**AC-010: List Available Presets**
- **Given** a shark installation
- **When** user runs `shark config list-presets`
- **Then** output includes preset names: standard, special-epics, numeric-only, legacy-prp
- **And** each preset shows brief description of use case
- **And** command works even without .sharkconfig.json (no project initialization required)

**AC-011: Show Preset Details**
- **Given** a shark installation
- **When** user runs `shark config show-preset special-epics`
- **Then** output displays full JSON structure of preset patterns
- **And** output indicates this preset adds patterns for epic.folder
- **And** patterns are shown in ready-to-copy format

**AC-012: Unknown Preset Rejected**
- **Given** a shark installation
- **When** user runs `shark config add-pattern --preset=nonexistent`
- **Then** command fails with error "Unknown preset: nonexistent"
- **And** error message includes list of available presets
- **And** .sharkconfig.json is not modified

### Pattern Testing

**AC-013: Test Pattern Match Success**
- **Given** a pattern `E(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)`
- **When** user runs `shark config test-pattern --pattern="..." --test-string="E04-task-mgmt" --type=epic`
- **Then** output shows "Match successful"
- **And** output displays captured groups: `number=04`, `slug=task-mgmt`
- **And** output shows validation result: "Pattern valid for epic type (has required capture group 'number')"

**AC-014: Test Pattern Match Failure**
- **Given** a pattern `E(?P<number>\d{2})-(?P<slug>[a-z0-9-]+)`
- **When** user runs `shark config test-pattern --pattern="..." --test-string="tech-debt" --type=epic`
- **Then** output shows "No match"
- **And** output suggests similar patterns from config that would match
- **And** output shows validation result for reference (pattern is valid even though this string didn't match)

**AC-015: Test Pattern Validation Failure**
- **Given** a pattern `F(?P<slug>[a-z-]+)` missing epic identifier
- **When** user runs `shark config test-pattern --pattern="..." --test-string="F01-feature" --type=feature`
- **Then** output shows match result (may succeed)
- **And** output shows validation error: "Pattern invalid for feature type: missing required epic identifier (epic_id or epic_num)"
- **And** output explains why validation matters (relationship inference will fail)

**AC-016: Validate All Patterns Command**
- **Given** a .sharkconfig.json with 15 patterns (12 valid, 2 with syntax errors, 1 missing required group)
- **When** user runs `shark config validate-patterns`
- **Then** output groups results by entity type (epic, feature, task)
- **And** output shows 12 patterns passed validation
- **And** output shows 2 patterns with syntax errors (detailed messages)
- **And** output shows 1 pattern with missing capture group (detailed message)
- **And** command exits with status 1 (failure for CI integration)

### Configuration Display

**AC-017: Show All Configuration**
- **Given** a configured .sharkconfig.json
- **When** user runs `shark config show`
- **Then** output displays all config sections (docs_root, default_epic, patterns, etc.)
- **And** output is pretty-printed JSON or table format (readable)
- **And** patterns section shows arrays clearly with index numbers

**AC-018: Show Only Patterns**
- **Given** a configured .sharkconfig.json
- **When** user runs `shark config show --patterns`
- **Then** output displays only patterns section
- **And** output includes epic, feature, and task subsections
- **And** output shows folder patterns, file patterns, and generation formats separately

**AC-019: Query Generation Format**
- **Given** a configured task generation format `T-E{epic:02d}-F{feature:02d}-{number:03d}.md`
- **When** user runs `shark config get-format --type=task`
- **Then** output displays format template: `T-E{epic:02d}-F{feature:02d}-{number:03d}.md`
- **And** output includes example with sample values: `T-E04-F07-003.md`

**AC-020: Query Format JSON Output**
- **Given** a configured task generation format
- **When** user runs `shark config get-format --type=task --json`
- **Then** output is valid JSON with structure: `{"format": "...", "example": "...", "placeholders": [...]}`
- **And** placeholders array lists available placeholders for this entity type: `["epic", "feature", "number", "slug"]`
- **And** AI agents can parse output programmatically

### Generation Format Application

**AC-021: Apply Epic Generation Format**
- **Given** epic generation format `E{number:02d}-{slug}`
- **When** user runs `shark epic create "Identity Platform" --slug=identity-platform`
- **Then** epic folder is created as `E05-identity-platform` (auto-incremented number)
- **And** epic.md file is created within that folder
- **And** database record uses epic_key `E05-identity-platform`

**AC-022: Apply Feature Generation Format**
- **Given** feature generation format `E{epic:02d}-F{number:02d}-{slug}`
- **When** user runs `shark feature create "OAuth Integration" --epic=E04 --slug=oauth-integration`
- **Then** feature folder is created as `E04-F08-oauth-integration`
- **And** prd.md file is created within that folder
- **And** database record uses feature_key `E04-F08-oauth-integration`

**AC-023: Apply Task Generation Format**
- **Given** task generation format `T-E{epic:02d}-F{feature:02d}-{number:03d}.md`
- **When** user runs `shark task create "Implement token refresh" --feature=E04-F07`
- **Then** task file is created as `T-E04-F07-007.md` in feature tasks/ subfolder
- **And** database record uses task_key `T-E04-F07-007`
- **And** file frontmatter includes task_key field with same value

**AC-024: Sanitize Slug Values**
- **Given** epic generation format `E{number:02d}-{slug}`
- **When** user runs `shark epic create "Test/Path" --slug="../malicious"`
- **Then** command rejects slug with error "Invalid slug: contains forbidden characters"
- **And** error message lists forbidden characters: `/`, `\`, `..`, etc.
- **And** no file or database record is created

### First-Match-Wins Pattern Ordering

**AC-025: First Pattern Matches**
- **Given** epic folder patterns: `["E(?P<number>\d{2})-(?P<slug>.+)", "(?P<epic_id>tech-debt|bugs)"]`
- **When** scanning folder `E04-task-mgmt`
- **Then** first pattern matches and extracts `number=04`, `slug=task-mgmt`
- **And** second pattern is never evaluated (short-circuit)
- **And** scan log shows which pattern matched

**AC-026: Second Pattern Matches After First Fails**
- **Given** epic folder patterns: `["E(?P<number>\d{2})-(?P<slug>.+)", "(?P<epic_id>tech-debt|bugs)"]`
- **When** scanning folder `tech-debt`
- **Then** first pattern fails to match
- **And** second pattern matches and extracts `epic_id=tech-debt`
- **And** scan log shows which pattern matched

**AC-027: No Pattern Matches**
- **Given** epic folder patterns: `["E(?P<number>\d{2})-(?P<slug>.+)", "(?P<epic_id>tech-debt|bugs)"]`
- **When** scanning folder `unknown-folder`
- **Then** all patterns are evaluated and all fail
- **And** folder is skipped with warning in scan report
- **And** warning message lists available patterns for reference

### Error Handling

**AC-028: Corrupted Config File Handling**
- **Given** a .sharkconfig.json with invalid JSON syntax
- **When** user runs any shark command
- **Then** command displays error "Failed to parse .sharkconfig.json: invalid JSON at line X"
- **And** command falls back to default patterns with warning
- **And** command suggests running `shark config validate` to check syntax

**AC-029: Regex DoS Pattern Rejected**
- **Given** a pattern with catastrophic backtracking `(a+)+b`
- **When** config is loaded
- **Then** validation detects excessive nested quantifiers
- **And** config load fails with error "Pattern may cause catastrophic backtracking"
- **And** error message explains performance risk

**AC-030: Pattern Matching Timeout**
- **Given** a complex pattern that takes >100ms to evaluate
- **When** scanning a file with that pattern
- **Then** pattern matching times out after 100ms
- **And** file is skipped with warning "Pattern matching timeout for file X"
- **And** scan continues with remaining files (no crash)

---

## Out of Scope

### Explicitly Excluded from This Feature

**1. Scanning Engine Implementation**
- The actual file traversal, parsing, and database import logic is handled by E06-F02 (Epic/Feature Discovery) and E06-F03 (Task Recognition & Import)
- This feature only defines the configuration layer that the scanning engine uses
- Pattern matching application during scans is out of scope (configuration only)

**2. Incremental Sync Mechanics**
- Modification time tracking, conflict detection, and incremental update logic is E06-F04 (Incremental Sync Engine)
- This feature does not handle sync triggers, timestamps, or change detection

**3. epic-index.md Parsing**
- Parsing epic-index.md for explicit epic/feature links is E06-F02
- This feature defines patterns for folder/file recognition, not index file parsing

**4. Import Reporting and Validation**
- Scan reports, conflict summaries, and post-import validation are E06-F05 (Import Reporting & Validation)
- This feature provides pattern testing tools but not full import validation

**5. Database Schema Changes**
- No new database tables or columns are required for this feature
- Pattern matching results (captured groups) are passed to scanning engine for use with existing schema

**6. Pattern Migration Tooling**
- Automatic migration of patterns from v1 to v2 schema (if future schema changes occur) is deferred
- Users will manually update patterns or use migration tool provided in future release

**7. Pattern Sharing/Marketplace**
- Community pattern library, pattern version control, or pattern sharing marketplace is not included
- Users can manually share .sharkconfig.json files but no built-in sharing mechanism

**8. Advanced Regex Features**
- Lookahead/lookbehind assertions, conditional patterns, and other advanced regex features may work but are not officially supported or documented
- Focus is on common patterns that 95% of users need

**9. Pattern Analytics**
- Tracking which patterns are used most frequently, pattern match success rates, or pattern performance metrics is out of scope
- Logging shows which pattern matched but no aggregate analytics

**10. GUI Configuration Editor**
- All configuration is done via CLI commands or manual .sharkconfig.json editing
- No graphical interface for pattern editing or testing (CLI only)

### Future Enhancements (Not in Initial Release)

**1. Pattern Performance Profiling**
- Tool to measure regex matching performance and identify slow patterns
- Could be added if users report performance issues with complex pattern sets

**2. Interactive Pattern Builder**
- Guided CLI wizard that asks questions and generates patterns without regex knowledge
- Example: "What format are your epic folders? (a) E##-slug (b) Descriptive names (c) Custom"

**3. Pattern Recommendations**
- Analyze existing documentation and suggest optimal patterns
- Example: "Detected 15 files matching tech-debt pattern. Add special-epics preset?"

**4. Pattern Version Control Integration**
- Track pattern changes in git, show pattern diffs, support pattern branching
- Useful for teams experimenting with different conventions

**5. Regex-to-Plain-English Translation**
- Display human-readable description of what each pattern matches
- Example: `E\d{2}-[a-z-]+` → "Epic folders starting with E followed by 2 digits, a hyphen, and lowercase letters/hyphens"

### Assumptions

**1. User Configuration Skill Level**
- Users can edit JSON configuration files with examples
- Product Managers may need presets, Technical Leads comfortable with regex
- AI agents can parse JSON programmatically

**2. Documentation Conventions**
- Most projects use consistent conventions within a single epic/feature (not random variation)
- Special epic types (tech-debt, bugs) are common enough to warrant first-class preset support
- Users want to accept flexible input during import but generate consistent output for new items

**3. Performance Constraints**
- Pattern matching overhead of <1ms per file is acceptable
- Validation time of <100ms during config load is acceptable
- Most projects have <20 custom patterns (defaults + few custom additions)

**4. Regex Engine Capabilities**
- Go's regexp package (RE2 syntax) is sufficient for pattern matching needs
- No PCRE-specific features required (lookahead, backreferences)
- Named capture groups provide adequate extraction capability

**5. Configuration Lifecycle**
- Patterns are defined once during initial setup and rarely changed
- Pattern testing during setup is acceptable even if manual
- Pattern validation on every config load is not a performance concern

### Implementation Considerations (Architecture Phase)

The following technical decisions are deferred to the architecture phase:

**1. Pattern Compilation Caching Strategy**
- Whether to cache compiled regexes in memory for reuse
- Cache invalidation strategy if config changes mid-operation
- Shared cache vs. per-scanner-instance cache

**2. Capture Group Extraction Implementation**
- Use Go's `FindStringSubmatch` vs. `SubexpNames` combination
- Handling of optional capture groups (empty vs. absent)
- Default values for missing optional groups

**3. Generation Format Parsing Implementation**
- Template parsing approach (regex vs. string manipulation)
- Placeholder validation timing (parse time vs. generation time)
- Escaping mechanism for literal braces in format strings

**4. Error Message Localization**
- Whether to support multiple languages for error messages
- Message templating system for consistent formatting
- Error code system for programmatic error handling

**5. Config File Backup Strategy**
- Backup file naming convention (.sharkconfig.json.bak vs. timestamped)
- Number of backups to retain
- Backup restoration command

---

*Last Updated*: 2025-12-17
